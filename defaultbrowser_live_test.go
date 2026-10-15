package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// GHOST_CHROME=/usr/bin/chromium go test -run TestDefaultBrowserLive
// The real launch path (Store.Launch + emulator) and the real tab opener:
// a stopped profile launches straight to the link, a running one gets a new
// tab, and links that arrive mid-launch all open.
func TestDefaultBrowserLive(t *testing.T) {
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	var mu sync.Mutex
	uas := map[string]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		uas[r.URL.Path] = r.Header.Get("User-Agent")
		mu.Unlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<title>%s</title><body>%s</body>", r.URL.Path, r.URL.Path)
	}))
	defer srv.Close()

	root, err := os.MkdirTemp("", "defbrowser-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { time.Sleep(500 * time.Millisecond); _ = os.RemoveAll(root) })
	s := &Store{dir: root, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}, Settings: Settings{ChromiumPath: bin, ProfilesDir: filepath.Join(root, "profiles"), DefaultOS: "windows", DeviceID: "defbrowser"}}
	a := &App{store: s, auto: &autoSyncer{s: s, busy: map[string]bool{}}}
	mk := func(name string) string {
		fp := GenerateTyped("desktop", "windows", "")
		fp.StartupParameters = "--headless=new --no-sandbox"
		return s.PutProfile(Profile{Name: name, Kernel: "chromium", Fingerprint: fp, StartURLs: []string{srv.URL + "/own-start-page"}}).ID
	}
	idA, idB := mk("默认A"), mk("默认B")
	defer runReg.stop(idA)
	defer runReg.stop(idB)

	var rmu sync.Mutex
	var reports []string
	r := newLinkRouter()
	r.defaultID = func() string { return s.GetSettings().DefaultProfileID }
	r.exists = func(id string) (string, bool) {
		p, ok := s.GetProfile(id)
		return func() string {
			if ok {
				return p.Name
			}
			return ""
		}(), ok
	}
	r.isRunning = func(id string) bool { _, ok := runReg.get(id); return ok }
	r.openTab = openTabIn
	r.launch = a.launchToURL
	r.showPicker = func(u string) { t.Errorf("picker shown for %s", u) }
	r.report = func(id, name, u string, launched bool, err error) {
		rmu.Lock()
		defer rmu.Unlock()
		reports = append(reports, fmt.Sprintf("%s %s %v %v", name, u, launched, err))
		if err != nil {
			t.Errorf("open %s in %s: %v", u, name, err)
		}
	}
	r.SetReady()

	pages := func(id string) []string {
		rs, ok := runReg.get(id)
		if !ok {
			return nil
		}
		raw, err := cdpCall(rs.WSURL, "Target.getTargets", nil)
		if err != nil {
			return nil
		}
		var tl struct {
			TargetInfos []struct{ Type, URL string } `json:"targetInfos"`
		}
		_ = json.Unmarshal(raw, &tl)
		out := []string{}
		for _, ti := range tl.TargetInfos {
			if ti.Type == "page" {
				out = append(out, strings.TrimPrefix(ti.URL, srv.URL))
			}
		}
		sort.Strings(out)
		return out
	}
	waitPages := func(id string, want []string) []string {
		var got []string
		for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(250 * time.Millisecond) {
			got = pages(id)
			if strings.Join(got, ",") == strings.Join(want, ",") {
				return got
			}
		}
		return got
	}

	// 1. stopped default profile: launches and its first tab lands on the link
	st := s.GetSettings()
	st.DefaultProfileID = idA
	s.PutSettings(st)
	r.Open(srv.URL + "/cold-link")
	if got := waitPages(idA, []string{"/cold-link"}); strings.Join(got, ",") != "/cold-link" {
		t.Fatalf("cold launch should land on the link only (no about:blank, no own start page): %v", got)
	}
	if p, _ := s.GetProfile(idA); len(p.StartURLs) != 1 || !strings.HasSuffix(p.StartURLs[0], "/own-start-page") {
		t.Fatalf("profile start pages must be kept: %v", p.StartURLs)
	}
	// the emulator patched the page that loaded the link
	mu.Lock()
	ua := uas["/cold-link"]
	mu.Unlock()
	if p, _ := s.GetProfile(idA); ua != p.Fingerprint.UserAgent && !strings.Contains(ua, "Windows NT") {
		t.Errorf("link page loaded without the fingerprint UA: %q", ua)
	}

	// 2. running default profile: a new tab with the link
	r.Open(srv.URL + "/warm-link")
	if got := waitPages(idA, []string{"/cold-link", "/warm-link"}); len(got) != 2 {
		t.Fatalf("running profile should get a new tab: %v", got)
	}

	// 3. links while the profile is launching queue and all open
	st.DefaultProfileID = idB
	s.PutSettings(st)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); r.Open(srv.URL + "/b-first") }()
	time.Sleep(150 * time.Millisecond) // inside the launch
	for _, p := range []string{"/b-q1", "/b-q2"} {
		wg.Add(1)
		go func(p string) { defer wg.Done(); r.Open(srv.URL + p) }(p)
	}
	wg.Wait()
	if got := waitPages(idB, []string{"/b-first", "/b-q1", "/b-q2"}); len(got) != 3 {
		t.Fatalf("queued links during launch must all open: %v", got)
	}
	mu.Lock()
	for _, p := range []string{"/b-q1", "/b-q2"} {
		if !strings.Contains(uas[p], "Windows NT") {
			t.Errorf("%s loaded without the fingerprint UA: %q", p, uas[p])
		}
	}
	mu.Unlock()
	t.Logf("reports: %v", reports)
}
