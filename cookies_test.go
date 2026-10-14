package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCookiesJSONNetscapeRoundTrip(t *testing.T) {
	in := `[
	 {"name":"sid","value":"abc","domain":".example.com","path":"/","expirationDate":1999999999.5,"httpOnly":true,"secure":true,"sameSite":"no_restriction","hostOnly":false},
	 {"name":"pref","value":"a\tb","domain":"www.example.com","path":"/x","session":true,"sameSite":"lax","hostOnly":true},
	 {"name":"","value":"skip","domain":"x.com"}
	]`
	list, skipped, err := parseCookies("auto", in)
	if err != nil || skipped != 1 || len(list) != 2 {
		t.Fatalf("parse: %v skipped=%d %+v", err, skipped, list)
	}
	if list[0].SameSite != "None" || list[0].Expires != 1999999999.5 || list[1].Expires != 0 || list[1].SameSite != "Lax" || list[1].Domain != "www.example.com" {
		t.Fatalf("normalized: %+v", list)
	}
	ns, _ := formatCookies("netscape", list)
	if !strings.Contains(ns, "#HttpOnly_.example.com\tTRUE\t/\tTRUE\t1999999999\tsid\tabc\n") {
		t.Fatalf("netscape:\n%s", ns)
	}
	if !strings.Contains(ns, "www.example.com\tFALSE\t/x\tFALSE\t0\tpref\ta\tb\n") {
		t.Fatalf("netscape:\n%s", ns)
	}
	back, skipped, err := parseCookies("auto", ns)
	if err != nil || skipped != 0 || len(back) != 2 {
		t.Fatalf("reparse: %v %d %+v", err, skipped, back)
	}
	if !back[0].HTTPOnly || !back[0].Secure || back[0].Domain != ".example.com" || back[0].Expires != 1999999999 || back[1].Value != "a\tb" || back[1].HTTPOnly {
		t.Fatalf("round trip: %+v", back)
	}
	js, _ := formatCookies("json", back)
	again, _, err := parseCookies("json", js)
	if err != nil || len(again) != 2 || again[0] != back[0] || again[1] != back[1] {
		t.Fatalf("json round trip: %v %+v", err, again)
	}
}

func TestCookiesFormatsAndMerge(t *testing.T) {
	if _, _, err := parseCookies("json", "not json"); err == nil {
		t.Fatal("expected JSON error")
	}
	if _, _, err := parseCookies("auto", "   "); err == nil {
		t.Fatal("expected empty error")
	}
	l, _, err := parseCookies("auto", `{"cookies":[{"name":"a","value":"1","domain":"a.com","expires":-1}]}`)
	if err != nil || len(l) != 1 || l[0].Path != "/" || l[0].Expires != 0 {
		t.Fatalf("wrapped: %v %+v", err, l)
	}
	if _, _, err := parseNetscape("garbage line\n"); err == nil {
		t.Fatal("expected netscape error")
	}
	m := mergeCookies([]Cookie{{Name: "a", Domain: "x", Path: "/", Value: "old"}, {Name: "b", Domain: "x", Path: "/"}}, []Cookie{{Name: "a", Domain: "x", Path: "/", Value: "new"}})
	if len(m) != 2 || m[0].Value != "new" {
		t.Fatalf("merge %+v", m)
	}
	p := Cookie{Name: "a", Domain: "x", Path: "/", SameSite: "None"}.cdpParam()
	if p["secure"] != true || p["expires"] != nil {
		t.Fatalf("cdp %+v", p)
	}
}

func TestCookiesStagedForStoppedProfile(t *testing.T) {
	s := testStoreForCookies(t)
	p := s.PutProfile(Profile{Name: "c"})
	r, err := s.ImportCookies(p.ID, "netscape", ".a.com\tTRUE\t/\tFALSE\t0\tk\tv\n")
	if err != nil || r.Mode != "staged" || r.Count != 1 {
		t.Fatalf("import %+v %v", r, err)
	}
	r, err = s.ImportCookies(p.ID, "json", `[{"name":"k","value":"v2","domain":".a.com"},{"name":"z","value":"1","domain":"b.com"}]`)
	if err != nil || r.Count != 2 {
		t.Fatalf("import2 %+v %v", r, err)
	}
	if info := s.CookieInfo(p.ID); info.Staged != 2 || info.Running {
		t.Fatalf("info %+v", info)
	}
	out, err := s.ExportCookies(p.ID, "netscape")
	if err != nil || !strings.Contains(out, "\tk\tv2\n") || strings.Contains(out, "\tk\tv\n") {
		t.Fatalf("export %v\n%s", err, out)
	}
}

func testStoreForCookies(t *testing.T) *Store {
	d := t.TempDir()
	return &Store{dir: d, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}, Settings: Settings{ProfilesDir: filepath.Join(d, "profiles")}}
}

// GHOST_CHROME=/usr/bin/chromium go test -run TestCookiesLive
func TestCookiesLive(t *testing.T) {
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	port := freePort()
	cmd := exec.Command(bin, "--headless=new", "--no-sandbox", "--user-data-dir="+t.TempDir(), fmt.Sprintf("--remote-debugging-port=%d", port), "about:blank")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	ws, err := waitDebugger(port, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	// Chrome caps lifetimes at 400 days, so use an expiry inside that window.
	exp := time.Now().Add(30 * 24 * time.Hour).Unix()
	list, _, _ := parseCookies("netscape", fmt.Sprintf("#HttpOnly_.ghost.test\tTRUE\t/\tTRUE\t%d\tsid\tabc\nwww.ghost.test\tFALSE\t/p\tFALSE\t0\tpref\tzh\n", exp))
	n, err := liveSetCookies(ws, list)
	if err != nil || n != 2 {
		t.Fatalf("set: %d %v", n, err)
	}
	got, err := liveGetCookies(ws)
	if err != nil {
		t.Fatal(err)
	}
	ns := formatNetscape(got)
	if !strings.Contains(ns, fmt.Sprintf("#HttpOnly_.ghost.test\tTRUE\t/\tTRUE\t%d\tsid\tabc", exp)) || !strings.Contains(ns, "www.ghost.test\tFALSE\t/p\tFALSE\t0\tpref\tzh") {
		t.Fatalf("live export:\n%s", ns)
	}
}
