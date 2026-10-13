package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Opt-in check against the real PHP router + SQLite, not a mock backend.
func TestPHPSyncRoundTrip(t *testing.T) {
	php := os.Getenv("GHOST_PHP_TEST")
	if php == "" {
		t.Skip("set GHOST_PHP_TEST to php.exe")
	}
	project, _ := os.Getwd()
	root := t.TempDir()
	port := freePort()
	base := "http://127.0.0.1:" + itoa(port)
	token := strings.Repeat("test-private-token-", 4)
	cmd := exec.Command(php, "-d", "extension_dir="+filepath.Join(filepath.Dir(php), "ext"), "-d", "extension=pdo_sqlite", "-S", "127.0.0.1:"+itoa(port), "-t", filepath.Join(project, "cloud-server", "public"), filepath.Join(project, "cloud-server", "router.php"))
	cmd.Env = append(os.Environ(), "GHOST_SYNC_TOKEN="+token, "GHOST_SYNC_DATA_DIR="+filepath.Join(root, "server"))
	var log bytes.Buffer
	cmd.Stdout = &log
	cmd.Stderr = &log
	if e := cmd.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		if t.Failed() {
			t.Log(log.String())
		}
	}()
	store := func(name, device string) *Store {
		d := filepath.Join(root, name)
		_ = os.MkdirAll(d, 0700)
		return &Store{dir: d, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}, Settings: Settings{ProfilesDir: filepath.Join(d, "profiles"), SyncServer: base, SyncToken: token, SyncPassword: "local-encryption-password", DeviceID: device}}
	}
	source := store("source", "device-A")
	deadline := time.Now().Add(10 * time.Second)
	for {
		if e := source.TestSync(); e == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("PHP did not become ready")
		}
		time.Sleep(100 * time.Millisecond)
	}
	bad := store("bad", "device-bad")
	bad.Settings.SyncToken = "wrong-token"
	if e := bad.TestSync(); e == nil {
		t.Fatal("invalid bearer accepted")
	}
	px := source.PutProxy(Proxy{Name: "private proxy", Type: "http", Host: "127.0.0.1", Port: 8888, User: "proxy-user", Pass: "PRIVATE_PROXY_PASSWORD"})
	p := source.PutProfile(Profile{Name: "同步往返检查", ProxyID: px.ID, Cookies: `[{"name":"test","value":"PRIVATE_COOKIE_VALUE"}]`, Fingerprint: GenerateFingerprint("windows", "153")})
	dir := source.ProfileDir(p.ID)
	_ = os.MkdirAll(filepath.Join(dir, "Default", "Network"), 0700)
	if e := os.WriteFile(filepath.Join(dir, "Default", "Network", "Cookies"), []byte("PRIVATE_BROWSER_DATABASE"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := source.PushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	local, _ := source.GetProfile(p.ID)
	if local.CloudVer != 1 {
		t.Fatal("successful commit version not recorded")
	}
	list, e := source.ListCloud()
	if e != nil || len(list) != 1 {
		t.Fatalf("cloud listing: %v %+v", e, list)
	}
	if bytes.Contains(list[0].Meta, []byte("PRIVATE_")) {
		t.Fatal("private profile metadata leaked to server listing")
	}
	_ = filepath.Walk(filepath.Join(root, "server"), func(path string, fi os.FileInfo, e error) error {
		if e == nil && !fi.IsDir() {
			b, _ := os.ReadFile(path)
			if bytes.Contains(b, []byte("PRIVATE_PROXY_PASSWORD")) || bytes.Contains(b, []byte("PRIVATE_COOKIE_VALUE")) {
				t.Errorf("plaintext secret in server file: %s", fi.Name())
			}
		}
		return e
	})
	dest := store("destination", "device-B")
	if e = dest.PullProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	restored, ok := dest.GetProfile(p.ID)
	if !ok || restored.Name != p.Name || restored.CloudVer != 1 || restored.InlineProxy == nil || restored.InlineProxy.Pass != "PRIVATE_PROXY_PASSWORD" {
		t.Fatalf("private metadata not restored: %+v", restored)
	}
	b, e := os.ReadFile(filepath.Join(dest.ProfileDir(p.ID), "Default", "Network", "Cookies"))
	if e != nil || string(b) != "PRIVATE_BROWSER_DATABASE" {
		t.Fatal("browser data round-trip failed")
	}
	dest.Settings.SyncPassword = "wrong-password"
	if e = dest.PullProfile(p.ID); e == nil {
		t.Fatal("wrong encryption password accepted")
	}
	b, _ = os.ReadFile(filepath.Join(dest.ProfileDir(p.ID), "Default", "Network", "Cookies"))
	if string(b) != "PRIVATE_BROWSER_DATABASE" {
		t.Fatal("failed decrypt destroyed local data")
	}
	dest.Settings.SyncPassword = source.Settings.SyncPassword

	// A rejected commit must neither advance the local version nor publish an
	// uncommitted blob. The client must release its failed upload lease.
	u, _ := url.Parse(base)
	proxy := httputil.NewSingleHostReverseProxy(u)
	failure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/commit") {
			http.Error(w, "injected commit failure", 500)
			return
		}
		proxy.ServeHTTP(w, r)
	}))
	source.Settings.SyncServer = failure.URL
	if e = source.PushProfile(p.ID); e == nil {
		t.Fatal("commit HTTP 500 was ignored")
	}
	failure.Close()
	source.Settings.SyncServer = base
	local, _ = source.GetProfile(p.ID)
	if local.CloudVer != 1 {
		t.Fatal("failed commit advanced local version")
	}
	list, _ = source.ListCloud()
	if list[0].Version != 1 || list[0].Locked != "" {
		t.Fatal("failed commit published a version or retained its lease")
	}
	if e = source.PushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	if e = dest.PushProfile(p.ID); e == nil {
		t.Fatal("stale device overwrote cloud version")
	}
	if e = dest.PullProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	restored, _ = dest.GetProfile(p.ID)
	if restored.CloudVer != 2 {
		t.Fatal("new cloud version not restored")
	}

	// PHP's own lease, optimistic version and hash checks are independently
	// exercised against the real endpoints.
	c := source.sync()
	path := "/api/profiles/" + p.ID
	var lease struct {
		Token string `json:"lockToken"`
	}
	if e = c.jsonRequest("POST", path+"/lock", map[string]string{"device": "device-A"}, &lease); e != nil {
		t.Fatal(e)
	}
	c.lockToken = lease.Token
	if e = dest.sync().jsonRequest("POST", path+"/lock", map[string]string{"device": "device-B"}, nil); e == nil {
		t.Fatal("second lease accepted")
	}
	opaque := append([]byte("GBE1"), make([]byte, 44)...)
	r, e := c.req("PUT", path+"/blob?version=0", bytes.NewReader(opaque), "application/octet-stream")
	if e != nil {
		t.Fatal(e)
	}
	if r.StatusCode != 412 {
		t.Fatalf("stale PUT status %d", r.StatusCode)
	}
	r.Body.Close()
	r, e = c.req("PUT", path+"/blob?version=2", bytes.NewReader(opaque), "application/octet-stream")
	if e != nil {
		t.Fatal(e)
	}
	e = syncStatus(r)
	r.Body.Close()
	if e != nil {
		t.Fatal(e)
	}
	wrong := cloudMeta{ID: p.ID, Name: p.Name, Version: 3, Sha256: strings.Repeat("0", 64), Meta: json.RawMessage(`{"encrypted":true}`)}
	if e = c.jsonRequest("POST", path+"/commit", wrong, nil); e == nil {
		t.Fatal("wrong hash committed")
	}
	list, _ = source.ListCloud()
	if list[0].Version != 2 {
		t.Fatal("bad commit replaced cloud data")
	}
	if e = c.jsonRequest("POST", path+"/unlock", map[string]any{}, nil); e != nil {
		t.Fatal(e)
	}

	// Corruption on download is rejected before replacement.
	corrupt := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, _ := http.NewRequest(r.Method, base+r.URL.RequestURI(), r.Body)
		req.Header = r.Header.Clone()
		resp, e := http.DefaultClient.Do(req)
		if e != nil {
			http.Error(w, e.Error(), 500)
			return
		}
		defer resp.Body.Close()
		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		payload, _ := ioRead(resp)
		if len(payload) > 48 {
			payload[len(payload)-1] ^= 1
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(payload)
	}))
	dest.Settings.SyncServer = corrupt.URL
	if e = dest.PullProfile(p.ID); e == nil {
		t.Fatal("corrupt cloud data restored")
	}
	corrupt.Close()
	b, _ = os.ReadFile(filepath.Join(dest.ProfileDir(p.ID), "Default", "Network", "Cookies"))
	if string(b) != "PRIVATE_BROWSER_DATABASE" {
		t.Fatal("corrupt restore destroyed local data")
	}
	sum := sha256.Sum256(b)
	t.Logf("PHP upload/download, encrypted metadata, versions, lease/hash conflicts, failed commits and local-data preservation passed; restored database sha=%s", hex.EncodeToString(sum[:]))
}

func ioRead(r *http.Response) ([]byte, error) { return io.ReadAll(r.Body) }
