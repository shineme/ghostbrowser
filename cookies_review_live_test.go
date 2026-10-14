package main

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

func cookieBrowser(t *testing.T) string {
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	port := freePort()
	cmd := exec.Command(bin, "--headless=new", "--no-sandbox", "--user-data-dir="+liveUserDataDir(t), fmt.Sprintf("--remote-debugging-port=%d", port), "about:blank")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	ws, err := waitDebugger(port, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

// Review: JSON export from one real browser imports into another unchanged, and
// the reported count matches what the browser actually kept.
func TestCookiesReviewRoundTripLive(t *testing.T) {
	a, b := cookieBrowser(t), cookieBrowser(t)
	exp := float64(time.Now().Add(48 * time.Hour).Unix())
	src := fmt.Sprintf(`[
 {"name":"sid","value":"a b=c","domain":".shop.test","path":"/","secure":true,"httpOnly":true,"sameSite":"no_restriction","expirationDate":%v,"hostOnly":false},
 {"name":"pref","value":"中文","domain":"www.shop.test","path":"/p","hostOnly":true,"session":true,"sameSite":"lax"},
 {"name":"__Host-x","value":"1","domain":".shop.test","path":"/","secure":true},
 {"name":"bad","value":"1","domain":"","path":"/"}
]`, exp)
	list, skipped, err := parseCookies("json", src)
	if err != nil {
		t.Fatal(err)
	}
	n, err := liveSetCookies(a, list)
	got, _ := liveGetCookies(a)
	t.Logf("parsed %d skipped %d set %d err %v; browser has %d", len(list), skipped, n, err, len(got))
	if n != len(got) {
		t.Errorf("reported %d imported but the browser kept %d", n, len(got))
	}
	out, _ := formatCookies("json", got)
	l2, _, err := parseCookies("json", out)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := liveSetCookies(b, l2); err != nil {
		t.Fatal(err)
	}
	got2, _ := liveGetCookies(b)
	if fmt.Sprint(formatNetscape(got)) != fmt.Sprint(formatNetscape(got2)) {
		t.Errorf("round trip differs:\n%s\n---\n%s", formatNetscape(got), formatNetscape(got2))
	}
	ns := formatNetscape(got2)
	l3, _, _ := parseCookies("netscape", ns)
	if formatNetscape(l3) != ns {
		t.Errorf("netscape re-parse differs")
	}
	t.Logf("%s", ns)
}
