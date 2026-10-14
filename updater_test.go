package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"ghostbrowser/internal/updsig"
)

func TestCompareAppVersions(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"0.7.3", "0.7.2", 1}, {"v0.7.2", "0.7.2", 0}, {"0.7.10", "0.7.9", 1}, {"0.8", "0.7.99", 1},
		{"1.0.0-beta", "1.0.0", -1}, {"1.0.0", "1.0.0-rc1", 1}, {"0.7.2", "0.7.2.1", -1}, {"1.0.0-beta", "1.0.0-alpha", 1},
	} {
		if got := compareAppVersions(c.a, c.b); got != c.want {
			t.Errorf("compare(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

type updFixture struct {
	srv   *httptest.Server
	priv  ed25519.PrivateKey
	exe   []byte
	doc   latestDoc
	mu    sync.Mutex
	token string
}

func newUpdFixture(t *testing.T) *updFixture {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	old := updatePubKey
	updatePubKey = base64.StdEncoding.EncodeToString(pub)
	t.Cleanup(func() { updatePubKey = old })
	f := &updFixture{priv: priv, exe: []byte(strings.Repeat("NEW-GHOSTBROWSER-EXE", 50000))}
	sum := sha256.Sum256(f.exe)
	hexsum := hex.EncodeToString(sum[:])
	f.doc = latestDoc{Version: "99.0.0", Notes: "- 修复若干问题", URL: "GhostBrowser-99.exe", SHA256: hexsum, Sig: updsig.Sign(priv, "99.0.0", hexsum)}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer "+f.token {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"error":"Invalid token."}`))
			return
		}
		switch r.URL.Query().Get("r") {
		case "/api/update/latest":
			_ = json.NewEncoder(w).Encode(f.doc)
		case "/api/update/file/GhostBrowser-99.exe":
			w.Header().Set("Content-Length", itoa(len(f.exe)))
			_, _ = w.Write(f.exe)
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(f.srv.Close)
	f.token = strings.Repeat("tok", 12)
	return f
}

func (f *updFixture) settings() Settings {
	return Settings{SyncServer: f.srv.URL + "/ghostsync.php", SyncToken: f.token}
}

func newTestUpdater() *updater {
	return &updater{hc: &http.Client{Timeout: 10 * time.Second}, exePath: os.Executable, prog: UpdateProgress{Phase: "idle"}}
}

func TestUpdaterCheckAndInstall(t *testing.T) {
	f := newUpdFixture(t)
	u := newTestUpdater()
	info := u.Check(f.settings())
	if info.Error != "" || !info.Available || info.Latest != "99.0.0" || info.Source != "server" {
		t.Fatalf("check: %+v", info)
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "GhostBrowser.exe")
	_ = os.WriteFile(exe, []byte("OLD"), 0o755)
	u.exePath = func() (string, error) { return exe, nil }
	var relaunched string
	quit := make(chan struct{})
	var phases []string
	var pmu sync.Mutex
	u.emit = func(name string, v any) {
		if p, ok := v.(UpdateProgress); ok {
			pmu.Lock()
			phases = append(phases, p.Phase)
			pmu.Unlock()
		}
	}
	u.relaunch = func(e string) error { relaunched = e; return nil }
	u.quit = func() { close(quit) }
	if err := u.Install(f.settings()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-quit:
	case <-time.After(10 * time.Second):
		t.Fatalf("install did not finish: %+v", u.Progress())
	}
	got, _ := os.ReadFile(exe)
	if string(got) != string(f.exe) {
		t.Fatal("exe not replaced")
	}
	if old, _ := os.ReadFile(exe + ".old"); string(old) != "OLD" {
		t.Fatal("old exe not kept as .old")
	}
	if relaunched != exe {
		t.Fatalf("relaunch %q", relaunched)
	}
	pmu.Lock()
	joined := strings.Join(phases, ",")
	pmu.Unlock()
	for _, want := range []string{"downloading", "verifying", "installing", "restarting"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("phase %s missing: %s", want, joined)
		}
	}
}

func installExpectError(t *testing.T, f *updFixture, want string) {
	t.Helper()
	u := newTestUpdater()
	if info := u.Check(f.settings()); !info.Available {
		t.Fatalf("check: %+v", info)
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "GhostBrowser.exe")
	_ = os.WriteFile(exe, []byte("OLD"), 0o755)
	u.exePath = func() (string, error) { return exe, nil }
	u.relaunch = func(string) error { t.Error("must not relaunch"); return nil }
	u.quit = func() { t.Error("must not quit") }
	done := make(chan UpdateProgress, 4)
	u.emit = func(name string, v any) {
		if p, ok := v.(UpdateProgress); ok && p.Phase == "error" {
			done <- p
		}
	}
	if err := u.Install(f.settings()); err != nil {
		t.Fatal(err)
	}
	select {
	case p := <-done:
		if !strings.Contains(p.Error, want) {
			t.Fatalf("error %q, want %q", p.Error, want)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no error reported")
	}
	if b, _ := os.ReadFile(exe); string(b) != "OLD" {
		t.Fatal("exe replaced despite failed verification")
	}
	if _, err := os.Stat(exe + ".download"); !os.IsNotExist(err) {
		t.Fatal("partial download left behind")
	}
}

func TestUpdaterRejectsBadSignature(t *testing.T) {
	f := newUpdFixture(t)
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	f.doc.Sig = updsig.Sign(other, f.doc.Version, f.doc.SHA256) // signed by someone else
	installExpectError(t, f, "签名校验失败")
}

func TestUpdaterRejectsReplayedSignature(t *testing.T) {
	f := newUpdFixture(t)
	f.doc.Version = "100.0.0" // valid sig for 99.0.0 served as a newer version
	installExpectError(t, f, "签名校验失败")
}

func TestUpdaterRejectsSHAMismatch(t *testing.T) {
	f := newUpdFixture(t)
	f.exe = append([]byte("TAMPERED"), f.exe...) // file swapped, latest.json intact
	installExpectError(t, f, "SHA-256")
}

func TestUpdaterNoUpdateAndErrors(t *testing.T) {
	f := newUpdFixture(t)
	f.doc.Version = appVersion()
	u := newTestUpdater()
	if info := u.Check(f.settings()); info.Available || info.Error != "" {
		t.Fatalf("same version must not be available: %+v", info)
	}
	if err := u.Install(f.settings()); err == nil {
		t.Fatal("install without update must fail")
	}
	bad := f.settings()
	bad.SyncToken = "wrong-token-wrong-token-wrong-token"
	if info := newTestUpdater().Check(bad); info.Error == "" || !strings.Contains(info.Error, "401") {
		t.Fatalf("expected 401 error: %+v", info)
	}
	if info := newTestUpdater().Check(Settings{}); !strings.Contains(info.Error, "没有配置更新源") {
		t.Fatalf("no source: %+v", info)
	}
}

func TestDownloadURLResolution(t *testing.T) {
	st := Settings{SyncServer: "https://s.test/ghostsync.php", SyncToken: "tok"}
	if u, tok := downloadURL(st, "server", "", "GB.exe"); !strings.Contains(u, "r=%2Fapi%2Fupdate%2Ffile%2FGB.exe") || tok != "tok" {
		t.Fatalf("bare name: %s %s", u, tok)
	}
	if u, tok := downloadURL(st, "custom", "https://cdn.test/x/latest.json", "GB.exe"); u != "https://cdn.test/x/GB.exe" || tok != "" {
		t.Fatalf("relative: %s %q", u, tok)
	}
	if u, _ := downloadURL(st, "custom", "", "file:///etc/passwd"); u != "" {
		t.Fatal("file: URL accepted")
	}
}

func TestReplaceExeRollsBack(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "a.exe")
	_ = os.WriteFile(exe, []byte("OLD"), 0o755)
	if err := replaceExe(exe, filepath.Join(dir, "missing")); err == nil {
		t.Fatal("expected error")
	}
	if b, _ := os.ReadFile(exe); string(b) != "OLD" {
		t.Fatal("rollback failed")
	}
}
