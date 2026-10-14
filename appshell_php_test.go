package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ghostbrowser/internal/updsig"
)

// GHOST_PHP_TEST=php go test -run TestAppShellPHP .
func TestAppShellPHPEndpoints(t *testing.T) {
	php := os.Getenv("GHOST_PHP_TEST")
	if php == "" {
		t.Skip("set GHOST_PHP_TEST to php")
	}
	project, _ := os.Getwd()
	root := t.TempDir()
	src, err := os.ReadFile(filepath.Join(project, "cloud-server", "ghostsync.php"))
	if err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(root, "ghostsync.php"), src, 0o644)
	port := freePort()
	token := strings.Repeat("appshell-token-", 3)
	base := "http://127.0.0.1:" + itoa(port) + "/ghostsync.php"
	cmd := exec.Command(php, "-S", "127.0.0.1:"+itoa(port), "-t", root)
	cmd.Env = append(os.Environ(), "GHOST_SYNC_TOKEN="+token, "GHOST_SYNC_DATA_DIR="+filepath.Join(root, "data"))
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	get := func(r, tok string) (int, []byte) {
		for i := 0; i < 50; i++ {
			req, _ := http.NewRequest("GET", base+"?r="+r, nil)
			if tok != "" {
				req.Header.Set("Authorization", "Bearer "+tok)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return resp.StatusCode, b
		}
		t.Fatal("php server did not start")
		return 0, nil
	}
	// nothing published yet
	if c, b := get("/api/update/latest", token); c != 404 || !strings.Contains(string(b), "update/latest.json") {
		t.Fatalf("latest without file: %d %s", c, b)
	}
	if c, b := get("/api/catalog", token); c != 200 || !strings.Contains(string(b), `"groups":[]`) {
		t.Fatalf("empty catalog: %d %s", c, b)
	}
	if c, _ := get("/api/catalog", "wrong-token-wrong-token-wrong-token-xx"); c != 401 {
		t.Fatalf("catalog must need the token: %d", c)
	}
	// publish a signed release + the sample catalog
	pubOld := updatePubKey
	t.Cleanup(func() { updatePubKey = pubOld })
	keyDir := t.TempDir()
	out, err := exec.Command("go", "run", "./cmd/sign-release", "keygen", "-out", keyDir).CombinedOutput()
	if err != nil {
		t.Fatalf("keygen: %v %s", err, out)
	}
	pub, _ := os.ReadFile(filepath.Join(keyDir, "ghostbrowser-update.pub"))
	updatePubKey = strings.TrimSpace(string(pub))
	_ = os.MkdirAll(filepath.Join(root, "update"), 0o755)
	exe := filepath.Join(root, "update", "GhostBrowser-99.0.0.exe")
	payload := []byte(strings.Repeat("MZ-new-build-", 20000))
	_ = os.WriteFile(exe, payload, 0o644)
	out, err = exec.Command("go", "run", "./cmd/sign-release", "sign", "-key", filepath.Join(keyDir, "ghostbrowser-update.key"),
		"-file", exe, "-version", "99.0.0", "-notes", "## 新功能\n- 托盘常驻", "-out", filepath.Join(root, "update", "latest.json")).CombinedOutput()
	if err != nil {
		t.Fatalf("sign: %v %s", err, out)
	}
	cat, _ := os.ReadFile(filepath.Join(project, "cloud-server", "catalog.json"))
	_ = os.WriteFile(filepath.Join(root, "catalog.json"), cat, 0o644)

	c, b := get("/api/update/latest", token)
	var d latestDoc
	if c != 200 || json.Unmarshal(b, &d) != nil || d.Version != "99.0.0" || d.URL != "GhostBrowser-99.0.0.exe" {
		t.Fatalf("latest: %d %s", c, b)
	}
	sum := sha256.Sum256(payload)
	if d.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatal("sha256 in latest.json")
	}
	pk, _ := updsig.ParsePublic(updatePubKey)
	if err := updsig.Verify(pk, d.Version, d.SHA256, d.Sig); err != nil {
		t.Fatal(err)
	}
	if c, b := get("/api/update/file/GhostBrowser-99.0.0.exe", token); c != 200 || len(b) != len(payload) {
		t.Fatalf("file: %d %d", c, len(b))
	}
	for _, bad := range []string{"/api/update/file/../ghostsync.php", "/api/update/file/latest.json", "/api/update/file/.htaccess"} {
		if c, _ := get(bad, token); c != 400 && c != 404 {
			t.Fatalf("%s served: %d", bad, c)
		}
	}
	if c, _ := get("/api/update/file/GhostBrowser-99.0.0.exe", ""); c != 401 {
		t.Fatalf("file without token: %d", c)
	}
	// client end to end against PHP
	st := Settings{SyncServer: base, SyncToken: token}
	u := newTestUpdater()
	info := u.Check(st)
	if !info.Available || info.Latest != "99.0.0" || !strings.Contains(info.Notes, "托盘常驻") {
		t.Fatalf("check: %+v", info)
	}
	dst := filepath.Join(t.TempDir(), "dl.exe")
	if err := u.download(st, u.doc, u.src, u.docBase, dst); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dst); len(got) != len(payload) {
		t.Fatal("downloaded size")
	}
	cs := (&catalogSvc{}).Refresh(t.TempDir(), st)
	if cs.Offline || len(cs.Catalog.Groups) != 3 || cs.Catalog.Groups[1].Name != "电商" || len(cs.Catalog.Announcements) == 0 {
		t.Fatalf("catalog via php: %+v", cs)
	}
	// existing sync routes still work
	if c, b := get("/api/ping", token); c != 200 || !strings.Contains(string(b), "ghostbrowser-sync") {
		t.Fatalf("ping: %d %s", c, b)
	}
}
