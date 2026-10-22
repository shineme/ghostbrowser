package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Cloud repair / chunk_missing tests. GHOST_PHP_TEST=php go test -run 'TestRepair|TestChunkGC|TestCommitRejects|TestPushReuploads|TestRefsSurvive'
// GHOST_PHP_SERVER_DIR points the PHP tests at another folder holding a
// ghostsync.php (e.g. the v0.8.7 one, to see them fail).

type repEnv struct {
	t                 *testing.T
	root, base, token string
	server            string // data dir
	devs              map[string]*Store
}

func startRepServer(t *testing.T, extraEnv ...string) *repEnv {
	php := os.Getenv("GHOST_PHP_TEST")
	if php == "" {
		t.Skip("set GHOST_PHP_TEST to php")
	}
	project, _ := os.Getwd()
	docroot := filepath.Join(project, "cloud-server")
	if d := os.Getenv("GHOST_PHP_SERVER_DIR"); d != "" {
		docroot = d
	}
	root := t.TempDir()
	port := freePort()
	env := &repEnv{t: t, root: root, base: "http://127.0.0.1:" + itoa(port) + "/ghostsync.php", token: strings.Repeat("repair-private-token-", 3), server: filepath.Join(root, "server"), devs: map[string]*Store{}}
	cmd := exec.Command(php, "-S", "127.0.0.1:"+itoa(port), "-t", docroot)
	cmd.Env = append(append(os.Environ(), "GHOST_SYNC_TOKEN="+env.token, "GHOST_SYNC_DATA_DIR="+env.server), extraEnv...)
	if e := cmd.Start(); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	a := env.dev("a", "dev-A")
	for i := 0; a.TestSync() != nil; i++ {
		if i > 100 {
			t.Fatal("php not ready")
		}
		time.Sleep(100 * time.Millisecond)
	}
	return env
}

func (env *repEnv) dev(name, device string) *Store {
	if s := env.devs[name]; s != nil {
		return s
	}
	d := filepath.Join(env.root, name)
	_ = os.MkdirAll(d, 0700)
	s := &Store{dir: d, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}, Settings: Settings{ProfilesDir: filepath.Join(d, "profiles"), SyncServer: env.base, SyncToken: env.token, SyncPassword: "pw-repair", DeviceID: device}}
	env.devs[name] = s
	return s
}

func (env *repEnv) raw(method, path string, hdr map[string]string, body []byte) (int, []byte) {
	req, _ := http.NewRequest(method, syncURL(env.base, path), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+env.token)
	req.Header.Set("X-Device-ID", "dev-raw")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		env.t.Fatal(e)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func (env *repEnv) chunkFiles() []string {
	l, _ := filepath.Glob(filepath.Join(env.server, "chunks", "??", "*"))
	return l
}

func (env *repEnv) ageChunks(d time.Duration) {
	old := time.Now().Add(-d)
	for _, f := range env.chunkFiles() {
		_ = os.Chtimes(f, old, old)
	}
	// let the periodic collector run on the next commit
	_ = os.Chtimes(filepath.Join(env.server, "chunks", ".gc"), old, old)
}

func (env *repEnv) index() map[string]map[string]any {
	b, _ := os.ReadFile(filepath.Join(env.server, "index.json"))
	var v struct {
		Profiles map[string]map[string]any `json:"profiles"`
	}
	_ = json.Unmarshal(b, &v)
	return v.Profiles
}

func repProfile(t *testing.T, s *Store, name string, size int) (Profile, []byte) {
	p := s.PutProfile(Profile{Name: name, Fingerprint: GenerateFingerprint("windows", "141")})
	dir := s.ProfileDir(p.ID)
	big := make([]byte, size)
	_, _ = rand.Read(big)
	_ = os.MkdirAll(filepath.Join(dir, "Default"), 0700)
	_ = os.WriteFile(filepath.Join(dir, "Default", "big.bin"), big, 0600)
	_ = os.WriteFile(filepath.Join(dir, "small.txt"), []byte("v1"), 0600)
	return p, big
}

func pulledBig(s *Store, id string) []byte {
	b, _ := os.ReadFile(filepath.Join(s.ProfileDir(id), "Default", "big.bin"))
	return b
}

// The chunk list sent before the blob (PUT refs, then PUT blob) must survive
// the blob upload, or the collector sees the whole profile as unreferenced.
func TestRefsSurviveBlobUpload(t *testing.T) {
	env := startRepServer(t)
	a := env.dev("a", "dev-A")
	p, _ := repProfile(t, a, "refs", 9<<20)
	if _, e := a.pushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	sha, _ := env.index()[p.ID]["sha256"].(string)
	b, e := os.ReadFile(filepath.Join(env.server, "refs", sha+".txt"))
	if e != nil {
		t.Fatalf("no chunk list stored for the committed version: %v", e)
	}
	if n := len(strings.Fields(string(b))); n < 3 || n != len(env.chunkFiles()) {
		t.Fatalf("chunk list has %d ids, server has %d chunks", n, len(env.chunkFiles()))
	}
}

// Dedupe vs collector race: a chunk that exists but is old and unreferenced is
// reported present by /api/chunks/missing, the client skips it, and the
// collector (forced by a cloud delete) runs before the commit.
func TestChunkGCRaceDedupe(t *testing.T) {
	env := startRepServer(t, "GHOST_SYNC_KEEP_VERSIONS=1")
	a, b := env.dev("a", "dev-A"), env.dev("b", "dev-B")
	p, big := repProfile(t, a, "race", 9<<20)
	other, _ := repProfile(t, a, "other", 1<<10)
	if _, e := a.pushProfile(other.ID); e != nil {
		t.Fatal(e)
	}
	if _, e := a.pushProfile(p.ID); e != nil { // v1: content A
		t.Fatal(e)
	}
	bigPath := filepath.Join(a.ProfileDir(p.ID), "Default", "big.bin")
	alt := make([]byte, 9<<20)
	_, _ = rand.Read(alt)
	_ = os.WriteFile(bigPath, alt, 0600)
	if _, e := a.pushProfile(p.ID); e != nil { // v2: content B; v1 (and its chunk list) is dropped
		t.Fatal(e)
	}
	env.ageChunks(72 * time.Hour) // A's chunks: old and unreferenced
	_ = os.WriteFile(bigPath, big, 0600)
	fired := false
	testHookAfterMissing = func() {
		if fired {
			return
		}
		fired = true
		// another device deletes a profile: forces the chunk collector now
		code, body := env.raw("POST", "/api/profiles/"+other.ID+"/delete", nil, []byte(`{"force":true}`))
		if code != 200 {
			t.Errorf("delete: %d %s", code, body)
		}
	}
	retried := false
	testHookCommitRetry = func() { retried = true }
	defer func() { testHookAfterMissing, testHookCommitRetry = nil, nil }()
	if _, e := a.pushProfile(p.ID); e != nil { // v3: content A again (dedupe)
		t.Fatal(e)
	}
	if !fired {
		t.Fatal("hook did not run")
	}
	if retried { // the missing check refreshed the chunks, so the collector kept them
		t.Fatal("collector deleted chunks the missing check reported present")
	}
	if _, e := b.pullProfile(p.ID, 0); e != nil {
		t.Fatalf("pull after the race: %v", e)
	}
	if !bytes.Equal(pulledBig(b, p.ID), big) {
		t.Fatal("pulled data differs")
	}
}

// Versions uploaded through ghostsync.php <= 0.8.7 have no chunk list on the
// server: the collector must not treat their (old) chunks as garbage.
func TestChunkGCKeepsLegacyUnlisted(t *testing.T) {
	env := startRepServer(t, "GHOST_SYNC_KEEP_VERSIONS=1") // no history: the repaired version replaces the unlisted one
	a, b := env.dev("a", "dev-A"), env.dev("b", "dev-B")
	p, big := repProfile(t, a, "legacy", 9<<20)
	other, _ := repProfile(t, a, "other", 1<<10)
	for _, id := range []string{other.ID, p.ID} {
		if _, e := a.pushProfile(id); e != nil {
			t.Fatal(e)
		}
	}
	sha, _ := env.index()[p.ID]["sha256"].(string)
	if e := os.Remove(filepath.Join(env.server, "refs", sha+".txt")); e != nil { // as the old server left it
		t.Fatal(e)
	}
	env.ageChunks(72 * time.Hour)
	if code, body := env.raw("POST", "/api/profiles/"+other.ID+"/delete", nil, []byte(`{"force":true}`)); code != 200 {
		t.Fatalf("delete: %d %s", code, body)
	}
	if _, e := b.pullProfile(p.ID, 0); e != nil {
		t.Fatalf("collector deleted the chunks of a profile without a chunk list: %v", e)
	}
	if !bytes.Equal(pulledBig(b, p.ID), big) {
		t.Fatal("pulled data differs")
	}
	// once every kept version has a chunk list, garbage is collected again
	if _, e := a.RepairCloud(p.ID); e != nil {
		t.Fatal(e)
	}
	stray := filepath.Join(env.server, "chunks", "ff", strings.Repeat("ff", 32))
	_ = os.MkdirAll(filepath.Dir(stray), 0700)
	_ = os.WriteFile(stray, []byte("GBC1garbage"), 0600)
	env.ageChunks(72 * time.Hour)
	_ = os.WriteFile(filepath.Join(a.ProfileDir(p.ID), "small.txt"), []byte("v3"), 0600)
	if _, e := a.pushProfile(p.ID); e != nil { // periodic collection runs after this commit
		t.Fatal(e)
	}
	if _, e := os.Stat(stray); e == nil {
		t.Fatal("stray chunk was not collected")
	}
}

// A commit whose chunk list names a chunk the server lacks is refused (409
// chunk_missing), and a missing chunk on /api/chunks/get is a 409, not a 404.
func TestCommitRejectsMissingChunk(t *testing.T) {
	env := startRepServer(t)
	id := "rawprofile01"
	ghost := strings.Repeat("ab", 32)
	code, body := env.raw("POST", "/api/profiles/"+id+"/lock", nil, []byte(`{"device":"dev-raw"}`))
	if code != 200 {
		t.Fatalf("lock %d %s", code, body)
	}
	var lease struct {
		Token string `json:"lockToken"`
	}
	_ = json.Unmarshal(body, &lease)
	h := map[string]string{"X-Lock-Token": lease.Token}
	if code, body = env.raw("PUT", "/api/profiles/"+id+"/refs", h, []byte(ghost)); code != 200 {
		t.Fatalf("refs %d %s", code, body)
	}
	blob := append([]byte("GBE1"), bytes.Repeat([]byte{7}, 60)...)
	if code, body = env.raw("PUT", "/api/profiles/"+id+"/blob?version=0", h, blob); code != 200 {
		t.Fatalf("blob %d %s", code, body)
	}
	sum := sha256.Sum256(blob)
	cm, _ := json.Marshal(map[string]any{"id": id, "name": "raw", "version": 1, "sha256": hex.EncodeToString(sum[:]), "meta": map[string]any{"encrypted": true}})
	code, body = env.raw("POST", "/api/profiles/"+id+"/commit", h, cm)
	var v struct {
		Error   string   `json:"error"`
		Code    string   `json:"code"`
		Missing []string `json:"missing"`
	}
	_ = json.Unmarshal(body, &v)
	if code != 409 || v.Code != "chunk_missing" || len(v.Missing) != 1 || v.Missing[0] != ghost {
		t.Fatalf("commit with a missing chunk: %d %s", code, body)
	}
	if r := env.index()[id]; r == nil || r["version"].(float64) != 0 {
		t.Fatalf("broken manifest was committed: %+v", r)
	}
	code, body = env.raw("POST", "/api/chunks/get", nil, []byte(`{"ids":["`+ghost+`"]}`))
	v.Code, v.Missing = "", nil
	_ = json.Unmarshal(body, &v)
	if code != 409 || v.Code != "chunk_missing" || len(v.Missing) != 1 {
		t.Fatalf("chunks/get of a missing chunk: %d %s", code, body)
	}
}

// Chunks vanish between the missing check and the commit: the push sends them
// again after the commit's chunk_missing and succeeds.
func TestPushReuploadsOnCommitChunkMissing(t *testing.T) {
	env := startRepServer(t)
	a, b := env.dev("a", "dev-A"), env.dev("b", "dev-B")
	p, big := repProfile(t, a, "retry", 9<<20)
	if _, e := a.pushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	_ = os.WriteFile(filepath.Join(a.ProfileDir(p.ID), "small.txt"), []byte("v2"), 0600)
	testHookAfterMissing = func() {
		for _, f := range env.chunkFiles() {
			_ = os.Remove(f)
		}
	}
	retried := false
	testHookCommitRetry = func() { retried = true }
	defer func() { testHookAfterMissing, testHookCommitRetry = nil, nil }()
	if _, e := a.pushProfile(p.ID); e != nil || !retried {
		t.Fatalf("push should recover from chunk_missing (retried=%v): %v", retried, e)
	}
	testHookAfterMissing = nil
	if _, e := b.pullProfile(p.ID, 0); e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(pulledBig(b, p.ID), big) {
		t.Fatal("pulled data differs")
	}
}

// Chunks deleted on the server: pull reports chunk_missing in Chinese, the
// check endpoint and CheckCloud see it, 修复云端 on the uploading device
// restores the profile.
func TestRepairCloudRestoresProfile(t *testing.T) {
	admin := strings.Repeat("admin-token-", 4)
	env := startRepServer(t, "GHOST_SYNC_ADMIN_TOKEN="+admin)
	adminPage := func() string {
		_, b := env.raw("GET", "/admin", map[string]string{"X-Admin-Token": admin}, nil)
		return string(b)
	}
	a, b := env.dev("a", "dev-A"), env.dev("b", "dev-B")
	p, big := repProfile(t, a, "broken", 13<<20)
	if _, e := a.pushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	files := env.chunkFiles()
	for _, f := range files[:2] {
		_ = os.Remove(f)
	}
	_, e := b.pullProfile(p.ID, 0)
	if !isChunkMissing(e) || !strings.Contains(e.Error(), "修复云端") {
		t.Fatalf("pull of a broken profile: %v", e)
	}
	code, body := env.raw("GET", "/api/profiles/"+p.ID+"/check", nil, nil)
	var ck struct {
		Refs    bool `json:"refs"`
		Chunks  int  `json:"chunks"`
		Missing int  `json:"missing"`
	}
	_ = json.Unmarshal(body, &ck)
	if code != 200 || !ck.Refs || ck.Missing != 2 || ck.Chunks != len(files) {
		t.Fatalf("check: %d %s", code, body)
	}
	if c, e := a.CheckCloud(p.ID); e != nil || c.Missing != 2 || !c.Chunked {
		t.Fatalf("CheckCloud: %+v %v", c, e)
	}
	if pg := adminPage(); !strings.Contains(pg, "1 个缺分块") || !strings.Contains(pg, p.ID) {
		t.Fatalf("admin page does not flag the broken profile")
	}
	// a device without the data cannot repair
	if _, e := b.RepairCloud(p.ID); e == nil {
		t.Fatal("repair without local data should fail")
	}
	r, e := a.RepairCloud(p.ID)
	if e != nil || r.Uploaded != 2 || r.Version != 2 || !strings.Contains(r.Message, "补传 2") {
		t.Fatalf("repair: %+v %v", r, e)
	}
	if _, e = b.pullProfile(p.ID, 0); e != nil {
		t.Fatalf("pull after repair: %v", e)
	}
	if !bytes.Equal(pulledBig(b, p.ID), big) {
		t.Fatal("pulled data differs")
	}
	if c, e := a.CheckCloud(p.ID); e != nil || c.Missing != 0 {
		t.Fatalf("CheckCloud after repair: %+v %v", c, e)
	}
	if pg := adminPage(); strings.Contains(pg, "缺分块</span>") || !strings.Contains(pg, "完整") {
		t.Fatalf("admin page after repair still flags the profile")
	}
	// nothing missing: repair still commits a new version
	if r, e = a.RepairCloud(p.ID); e != nil || r.Uploaded != 0 || r.Version != 3 {
		t.Fatalf("second repair: %+v %v", r, e)
	}
	if log := a.readSyncLog(); len(log) == 0 || log[0].Action != "repair" || !log[0].OK {
		t.Fatalf("sync log: %+v", log)
	}
	// global record
	if r, e := a.RepairGlobal(); e != nil || r.Version < 1 {
		t.Fatalf("RepairGlobal: %+v %v", r, e)
	}
}

const nginx404 = "<html>\r\n<head><title>404 Not Found</title></head>\r\n<body>\r\n<center><h1>404 Not Found</h1></center>\r\n<hr><center>nginx</center>\r\n</body>\r\n</html>\r\n"

// syncStatus: an HTML body (nginx error_page) names the request and explains
// fastcgi_intercept_errors; 409 chunk_missing and an old server's HTML 404 on
// /api/chunks/get both become errChunkMissing.
func TestSyncStatusHTMLAndChunkMissing(t *testing.T) {
	mode := "html"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mode == "json409" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(409)
			_, _ = w.Write([]byte(`{"error":"Chunk x is missing.","code":"chunk_missing","missing":["` + strings.Repeat("cd", 32) + `"],"count":3}`))
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(404)
		_, _ = w.Write([]byte(nginx404))
	}))
	defer srv.Close()
	c := &syncClient{base: srv.URL + "/ghostsync.php", token: "t", hc: srv.Client()}
	e := c.jsonRequest("POST", "/api/profiles/abc/commit", map[string]any{}, nil)
	if e == nil || !strings.Contains(e.Error(), "POST /api/profiles/abc/commit") || !strings.Contains(e.Error(), "fastcgi_intercept_errors") ||
		!strings.Contains(e.Error(), "404 Not Found") || strings.Contains(e.Error(), "<html>") || isChunkMissing(e) {
		t.Fatalf("html error: %v", e)
	}
	e = c.getChunks(make([]byte, 32), []string{strings.Repeat("ef", 32)}, func(string, []byte) error { return nil }, nil)
	if !isChunkMissing(e) || !strings.Contains(e.Error(), "云端缺少该环境的部分数据分块") {
		t.Fatalf("old-server 404 on chunks/get: %v", e)
	}
	mode = "json409"
	e = c.getChunks(make([]byte, 32), []string{strings.Repeat("ef", 32)}, func(string, []byte) error { return nil }, nil)
	if !isChunkMissing(e) || len(chunkMissingIDs(e)) != 1 || !strings.Contains(e.Error(), "3 个数据分块") || !strings.Contains(e.Error(), "修复云端") {
		t.Fatalf("409 chunk_missing: %v", e)
	}
	// plain path (not a .php URL) is shown as the URL path
	c.base = srv.URL
	mode = "html"
	if e = c.jsonRequest("GET", "/api/ping", nil, nil); e == nil || !strings.Contains(e.Error(), "GET /api/ping") {
		t.Fatalf("path label: %v", e)
	}
}
