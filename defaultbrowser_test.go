package main

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestParseOpenURLArgs(t *testing.T) {
	cases := []struct {
		args []string
		want []string
	}{
		{nil, []string{}},
		{[]string{"--open-url", "https://example.com/a?b=1"}, []string{"https://example.com/a?b=1"}},
		{[]string{"--open-url=http://example.com"}, []string{"http://example.com"}},
		{[]string{"--open-url", `"https://x.test/q"`}, []string{"https://x.test/q"}},
		{[]string{"https://a.test", "http://b.test/"}, []string{"https://a.test", "http://b.test/"}},
		{[]string{`C:\Users\me\My Docs\page.html`}, []string{"file:///C:/Users/me/My%20Docs/page.html"}},
		{[]string{"file:///C:/x/y.svg"}, []string{"file:///C:/x/y.svg"}},
		{[]string{"--after-update", "1234"}, []string{}},
		{[]string{"--after-update", "1234", "--open-url", "https://u.test"}, []string{"https://u.test"}},
		{[]string{"--open-url", "javascript:alert(1)"}, []string{}},
		{[]string{"--open-url"}, []string{}},
		{[]string{"--mcp-stdio"}, []string{}},
		{[]string{"notaurl", "--single-argument", "https://ok.test"}, []string{"https://ok.test"}},
	}
	for _, c := range cases {
		got := parseOpenURLArgs(c.args)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseOpenURLArgs(%q) = %q, want %q", c.args, got, c.want)
		}
	}
}

func TestSanitizeExternalURL(t *testing.T) {
	ok := map[string]string{
		"https://example.com":             "https://example.com",
		"  HTTPS://Example.com/x  ":       "https://example.com/x",
		"http://127.0.0.1:8080/p?q=1#h":   "http://127.0.0.1:8080/p?q=1#h",
		"'https://quoted.test/'":          "https://quoted.test/",
		`\\server\share\doc.htm`:          "file://server/share/doc.htm",
		"/home/u/page.html":               "file:///home/u/page.html",
		"file:///C:/a/b.html":             "file:///C:/a/b.html",
		"https://例子.测试/路径":                "https://xn--fsqu00a.xn--0zwm56d/%E8%B7%AF%E5%BE%84",
		"https://user@host.test/":         "https://user@host.test/",
		"https://example.com/a%20b?x=%3C": "https://example.com/a%20b?x=%3C",
	}
	for in, want := range ok {
		got, good := sanitizeExternalURL(in)
		if !good || got != want {
			t.Errorf("sanitize(%q) = %q,%v want %q", in, got, good, want)
		}
	}
	bad := []string{"", "about:blank", "javascript:alert(1)", "data:text/html,hi", "ms-settings:defaultapps",
		"ftp://x.test", "chrome://settings", "https://", "http:///nohost", "file://", "https://a.test/\x00x",
		"https://a.test/\nx", "vbscript:x", "mailto:a@b.c", strings.Repeat("a", 9000), "https://exa mple.com/"}
	for _, in := range bad {
		if got, good := sanitizeExternalURL(in); good {
			t.Errorf("sanitize(%q) accepted as %q", in, got)
		}
	}
}

func TestRouteLink(t *testing.T) {
	cases := []struct {
		id                         string
		exists, running, launching bool
		want                       linkAction
	}{
		{"", false, false, false, linkPick},
		{"gone", false, false, false, linkPick},
		{"gone", false, true, false, linkPick},
		{"p", true, false, false, linkLaunch},
		{"p", true, true, false, linkTab},
		{"p", true, false, true, linkQueue},
		{"p", true, true, true, linkQueue},
	}
	for _, c := range cases {
		if got := routeLink(c.id, c.exists, c.running, c.launching); got != c.want {
			t.Errorf("routeLink(%+v) = %s want %s", c, got, c.want)
		}
	}
}

// fakeLinks is a router wired to an in-memory browser.
type fakeLinks struct {
	mu       sync.Mutex
	def      string
	profiles map[string]string
	running  map[string]bool
	tabs     []string // "id url"
	launches []string
	picks    []string
	reports  []string
	launchFn func(id, u string) error
}

func newFakeLinks() (*fakeLinks, *linkRouter) {
	f := &fakeLinks{profiles: map[string]string{"a": "Alpha", "b": "Beta"}, running: map[string]bool{}}
	r := newLinkRouter()
	r.defaultID = func() string { f.mu.Lock(); defer f.mu.Unlock(); return f.def }
	r.exists = func(id string) (string, bool) {
		f.mu.Lock()
		defer f.mu.Unlock()
		n, ok := f.profiles[id]
		return n, ok
	}
	r.isRunning = func(id string) bool { f.mu.Lock(); defer f.mu.Unlock(); return f.running[id] }
	r.openTab = func(id, u string) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		if !f.running[id] {
			return errors.New("not running")
		}
		f.tabs = append(f.tabs, id+" "+u)
		return nil
	}
	r.launch = func(id, u string) error {
		if f.launchFn != nil {
			if err := f.launchFn(id, u); err != nil {
				return err
			}
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.launches = append(f.launches, id+" "+u)
		f.running[id] = true
		return nil
	}
	r.showPicker = func(u string) { f.mu.Lock(); f.picks = append(f.picks, u); f.mu.Unlock() }
	r.report = func(id, name, u string, launched bool, err error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		s := id + " " + u
		if launched {
			s += " launched"
		}
		if err != nil {
			s += " err"
		}
		f.reports = append(f.reports, s)
	}
	return f, r
}

func TestLinkRouterDecisions(t *testing.T) {
	f, r := newFakeLinks()
	r.SetReady()
	// no default -> picker, link kept for PendingLinks
	r.Open("https://one.test")
	if !reflect.DeepEqual(f.picks, []string{"https://one.test"}) {
		t.Fatalf("picks %v", f.picks)
	}
	if got := r.TakePicks(); !reflect.DeepEqual(got, []string{"https://one.test"}) || len(r.TakePicks()) != 0 {
		t.Fatalf("TakePicks %v", got)
	}
	// default deleted -> picker
	f.def = "deleted"
	r.Open("https://two.test")
	if len(f.picks) != 2 {
		t.Fatalf("deleted default must ask: %v", f.picks)
	}
	r.dropPick("https://two.test")
	if len(r.TakePicks()) != 0 {
		t.Fatal("dropPick")
	}
	// stopped -> launch straight to the link (no extra tab)
	f.def = "a"
	r.Open("https://three.test")
	if !reflect.DeepEqual(f.launches, []string{"a https://three.test"}) || len(f.tabs) != 0 {
		t.Fatalf("launch %v tabs %v", f.launches, f.tabs)
	}
	// running -> new tab
	r.Open("https://four.test")
	if !reflect.DeepEqual(f.tabs, []string{"a https://four.test"}) || len(f.launches) != 1 {
		t.Fatalf("tab %v launches %v", f.tabs, f.launches)
	}
	// picker choice for another profile
	r.OpenIn("b", "https://five.test")
	if f.launches[1] != "b https://five.test" {
		t.Fatalf("OpenIn %v", f.launches)
	}
}

func TestLinkRouterQueuesDuringLaunch(t *testing.T) {
	f, r := newFakeLinks()
	f.def = "a"
	release := make(chan struct{})
	started := make(chan struct{})
	f.launchFn = func(id, u string) error { close(started); <-release; return nil }
	r.SetReady()
	done := make(chan struct{})
	go func() { r.Open("https://first.test"); close(done) }()
	<-started
	var wg sync.WaitGroup
	for _, u := range []string{"https://q1.test", "https://q2.test", "https://q3.test"} {
		wg.Add(1)
		go func(u string) { defer wg.Done(); r.Open(u) }(u)
	}
	wg.Wait() // queued links return at once
	if len(f.launches) != 0 {
		t.Fatal("launch should still be in progress")
	}
	close(release)
	<-done
	if !reflect.DeepEqual(f.launches, []string{"a https://first.test"}) {
		t.Fatalf("only one launch expected: %v", f.launches)
	}
	tabs := append([]string{}, f.tabs...)
	sort.Strings(tabs)
	if !reflect.DeepEqual(tabs, []string{"a https://q1.test", "a https://q2.test", "a https://q3.test"}) {
		t.Fatalf("queued links must all open: %v", f.tabs)
	}
	if len(f.reports) != 4 {
		t.Fatalf("reports %v", f.reports)
	}
}

func TestLinkRouterLaunchFailureRetriesQueued(t *testing.T) {
	f, r := newFakeLinks()
	f.def = "a"
	calls := 0
	release := make(chan struct{})
	started := make(chan struct{})
	f.launchFn = func(id, u string) error {
		calls++
		if calls == 1 {
			close(started)
			<-release
			return errors.New("kernel missing")
		}
		return nil
	}
	r.SetReady()
	done := make(chan struct{})
	go func() { r.Open("https://first.test"); close(done) }()
	<-started
	r.Open("https://second.test")
	close(release)
	<-done
	if !reflect.DeepEqual(f.launches, []string{"a https://second.test"}) {
		t.Fatalf("queued link should relaunch after failure: %v", f.launches)
	}
	if f.reports[0] != "a https://first.test launched err" {
		t.Fatalf("failure not reported: %v", f.reports)
	}
}

func TestLinkRouterEarlyLinksWaitForReady(t *testing.T) {
	f, r := newFakeLinks()
	f.def = "a"
	r.Open("https://cold1.test")
	r.Open("https://cold2.test")
	if len(f.launches)+len(f.tabs)+len(f.picks) != 0 {
		t.Fatal("nothing may open before the app is ready")
	}
	r.SetReady()
	deadline := time.Now().Add(2 * time.Second)
	for len(f.tabs) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !reflect.DeepEqual(f.launches, []string{"a https://cold1.test"}) || !reflect.DeepEqual(f.tabs, []string{"a https://cold2.test"}) {
		t.Fatalf("launches %v tabs %v", f.launches, f.tabs)
	}
}

func TestDefaultBrowserKeys(t *testing.T) {
	exe := `D:\Tools\GhostBrowser\GhostBrowser.exe`
	keys := defaultBrowserKeys(exe)
	m := map[string]map[string]string{}
	for _, k := range keys {
		if strings.HasPrefix(k.Path, `\`) || strings.Contains(k.Path, "HKEY") {
			t.Fatalf("paths are relative to HKCU: %s", k.Path)
		}
		m[k.Path] = k.Values
	}
	want := map[string]map[string]string{
		`Software\Clients\StartMenuInternet\GhostBrowser`:                              {"": "GhostBrowser"},
		`Software\Clients\StartMenuInternet\GhostBrowser\DefaultIcon`:                  {"": exe + ",0"},
		`Software\Clients\StartMenuInternet\GhostBrowser\shell\open\command`:           {"": `"` + exe + `"`},
		`Software\Clients\StartMenuInternet\GhostBrowser\Capabilities\StartMenu`:       {"StartMenuInternet": "GhostBrowser"},
		`Software\Clients\StartMenuInternet\GhostBrowser\Capabilities\URLAssociations`: {"http": "GhostBrowserURL", "https": "GhostBrowserURL"},
		`Software\RegisteredApplications`:                                              {"GhostBrowser": `Software\Clients\StartMenuInternet\GhostBrowser\Capabilities`},
		`Software\Classes\GhostBrowserURL\shell\open\command`:                          {"": `"` + exe + `" --open-url "%1"`},
		`Software\Classes\GhostBrowserHTML\shell\open\command`:                         {"": `"` + exe + `" --open-url "%1"`},
		`Software\Classes\.html\OpenWithProgids`:                                       {"GhostBrowserHTML": ""},
	}
	for p, vals := range want {
		if !reflect.DeepEqual(m[p], vals) {
			t.Errorf("%s = %v want %v", p, m[p], vals)
		}
	}
	caps := m[`Software\Clients\StartMenuInternet\GhostBrowser\Capabilities`]
	if caps["ApplicationName"] != "GhostBrowser" || caps["ApplicationIcon"] != exe+",0" || caps["ApplicationDescription"] == "" {
		t.Errorf("capabilities %v", caps)
	}
	fa := m[`Software\Clients\StartMenuInternet\GhostBrowser\Capabilities\FileAssociations`]
	for _, e := range []string{".htm", ".html", ".shtml", ".xht", ".xhtml", ".svg"} {
		if fa[e] != "GhostBrowserHTML" {
			t.Errorf("file assoc %s = %q", e, fa[e])
		}
	}
	u := m[`Software\Classes\GhostBrowserURL`]
	if _, ok := u["URL Protocol"]; !ok || u["FriendlyTypeName"] == "" {
		t.Errorf("url progid %v", u)
	}
	if _, ok := m[`Software\Classes\GhostBrowserHTML`]["URL Protocol"]; ok {
		t.Error("HTML ProgId must not be a URL protocol")
	}
	// unregister covers everything written: each key is under a deleted tree or a shared value
	trees, shared := defaultBrowserTrees(), defaultBrowserSharedValues()
	for _, k := range keys {
		covered := false
		for _, tr := range trees {
			if k.Path == tr || strings.HasPrefix(k.Path, tr+`\`) {
				covered = true
			}
		}
		if name, ok := shared[k.Path]; ok {
			if _, has := k.Values[name]; has && len(k.Values) == 1 {
				covered = true
			}
		}
		if !covered {
			t.Errorf("unregister leaves %s behind", k.Path)
		}
	}
}

func TestDefaultBrowserRegHelpers(t *testing.T) {
	exe := `C:\new\GhostBrowser.exe`
	if needsReRegister(false, "", exe) {
		t.Error("not registered: never re-register")
	}
	if needsReRegister(true, dbOpenCommand(exe), exe) {
		t.Error("same path: nothing to do")
	}
	if !needsReRegister(true, dbOpenCommand(`C:\old\GhostBrowser.exe`), exe) {
		t.Error("moved EXE must re-register")
	}
	if !isSystemDefaultChoice("GhostBrowserURL", "GhostBrowserURL") || isSystemDefaultChoice("ChromeHTML", "GhostBrowserURL") || isSystemDefaultChoice("GhostBrowserURL", "MSEdgeHTM") {
		t.Error("isSystemDefaultChoice")
	}
}

func TestDefaultProfileIsDeviceLocal(t *testing.T) {
	s := testStore(t)
	st := s.GetSettings()
	st.DefaultProfileID = "p1"
	s.PutSettings(st)
	if s.globalPayload().Settings.DefaultProfileID != "" {
		t.Fatal("global payload must not carry the default browser profile")
	}
	h1 := globalHash(s.globalPayload())
	st.DefaultProfileID = "p2"
	s.PutSettings(st)
	if h2 := globalHash(s.globalPayload()); h1 != h2 {
		t.Fatal("changing the default profile must not mark global settings dirty")
	}
	// settings page saves (stale copy) and global pulls keep this device's value
	in := Settings{DefaultProfileID: "stale"}
	keepDeviceLocal(&in, s.GetSettings())
	if in.DefaultProfileID != "p2" {
		t.Fatalf("keepDeviceLocal: %q", in.DefaultProfileID)
	}
}

func TestDefaultProfileSurvivesGlobalPull(t *testing.T) {
	env := startSecServer(t)
	a := env.dev("owner", "A", "pw-default-browser")
	b := env.dev("owner", "B", "pw-default-browser")
	sa := a.GetSettings()
	sa.DefaultProfileID, sa.DefaultOS = "on-A", "macos"
	a.PutSettings(sa)
	if e := a.PushGlobal(); e != nil {
		t.Fatal(e)
	}
	sb := b.GetSettings()
	sb.DefaultProfileID = "on-B"
	b.PutSettings(sb)
	if e := b.PullGlobal(); e != nil {
		t.Fatal(e)
	}
	got := b.GetSettings()
	if got.DefaultOS != "macos" {
		t.Fatalf("pull did not apply: %+v", got.DefaultOS)
	}
	if got.DefaultProfileID != "on-B" {
		t.Fatalf("default browser profile must stay device-local, got %q", got.DefaultProfileID)
	}
}
