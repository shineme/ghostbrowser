package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// v0.8.9: files that change during an upload, excluded caches, refusing a
// running profile. PHP-backed ones: GHOST_PHP_TEST=php go test -run 'TestVolatile|TestExcluded|TestRunningProfile'

const swIndex = "Default/Service Worker/ScriptCache/index"

func writeRel(t *testing.T, dir, rel string, data []byte) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, data, 0600); e != nil {
		t.Fatal(e)
	}
}

func readRel(s *Store, id, rel string) []byte {
	b, _ := os.ReadFile(filepath.Join(s.ProfileDir(id), filepath.FromSlash(rel)))
	return b
}

// serverRefs returns the chunk list stored for the profile's current version.
func (env *repEnv) serverRefs(id string) []string {
	sha, _ := env.index()[id]["sha256"].(string)
	b, e := os.ReadFile(filepath.Join(env.server, "refs", sha+".txt"))
	if e != nil {
		env.t.Fatalf("no refs for the committed version: %v", e)
	}
	l := strings.Fields(string(b))
	sort.Strings(l)
	return l
}

// manifestIDs decodes the cloud manifest of the current version (unique,
// sorted chunk IDs).
func manifestIDs(t *testing.T, s *Store, id string) ([]string, *syncManifest) {
	t.Helper()
	resp, e := s.sync().req("GET", "/api/profiles/"+id+"/blob", nil, "")
	if e != nil {
		t.Fatal(e)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	secret, _ := s.dataSecret()
	raw, e := decrypt(buf.Bytes(), secret)
	if e != nil {
		t.Fatal(e)
	}
	m, ok, e := decodeManifest(raw)
	if e != nil || !ok {
		t.Fatalf("manifest: %v %v", ok, e)
	}
	seen := map[string]bool{}
	var l []string
	for _, f := range m.Files {
		for _, c := range f.Chunks {
			if !seen[c] {
				seen[c] = true
				l = append(l, c)
			}
		}
	}
	sort.Strings(l)
	return l, m
}

func sameList(a, b []string) bool { return strings.Join(a, ",") == strings.Join(b, ",") }

// A file rewritten between the scan and the upload is read again and
// uploaded with its new content; refs and manifest agree and a pull on another
// device restores the new content.
func TestVolatileFileRechunkedDuringUpload(t *testing.T) {
	env := startRepServer(t)
	a, b := env.dev("a", "dev-A"), env.dev("b", "dev-B")
	p, big := repProfile(t, a, "volatile", 9<<20)
	dir := a.ProfileDir(p.ID)
	writeRel(t, dir, swIndex, bytes.Repeat([]byte("old-index "), 300))
	newContent := bytes.Repeat([]byte("NEW-INDEX-CONTENT "), 500) // other size too
	testHookAfterMissing = func() { writeRel(t, dir, swIndex, newContent) }
	defer func() { testHookAfterMissing = nil }()
	if _, e := a.pushProfile(p.ID); e != nil {
		t.Fatalf("push with a file changing mid-upload: %v", e)
	}
	testHookAfterMissing = nil
	ids, m := manifestIDs(t, a, p.ID)
	if refs := env.serverRefs(p.ID); !sameList(refs, ids) {
		t.Fatalf("refs (%d) differ from the manifest (%d)", len(refs), len(ids))
	}
	for _, f := range m.Files {
		if f.Path == swIndex && f.Size != int64(len(newContent)) {
			t.Fatalf("manifest keeps the old size %d", f.Size)
		}
	}
	if c, e := a.CheckCloud(p.ID); e != nil || c.Missing != 0 {
		t.Fatalf("CheckCloud: %+v %v", c, e)
	}
	if _, e := b.pullProfile(p.ID, 0); e != nil {
		t.Fatalf("pull: %v", e)
	}
	if !bytes.Equal(readRel(b, p.ID, swIndex), newContent) || !bytes.Equal(pulledBig(b, p.ID), big) {
		t.Fatal("pulled data differs")
	}
}

// The v0.8.8 report: a memory-mapped cache index changed without moving its
// size or modification time, so the chunk cache still held the old ID; the
// server had lost that chunk (GC) and the upload failed with "index changed
// during sync". Both a normal push and 修复云端 must now succeed.
func TestVolatileStaleCacheIndex(t *testing.T) {
	env := startRepServer(t)
	a, b := env.dev("a", "dev-A"), env.dev("b", "dev-B")
	p, _ := repProfile(t, a, "stale", 1<<20)
	dir := a.ProfileDir(p.ID)
	old := bytes.Repeat([]byte{1}, 4096)
	writeRel(t, dir, swIndex, old)
	if _, e := a.pushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	key, _ := profileChunkKey(a.Profiles[p.ID])
	oldID := chunkID(key, old)
	sameTimeRewrite := func(data []byte) {
		path := filepath.Join(dir, filepath.FromSlash(swIndex))
		st, _ := os.Stat(path)
		writeRel(t, dir, swIndex, data)
		_ = os.Chtimes(path, st.ModTime(), st.ModTime())
	}
	// server lost the chunk; content changed in place
	_ = os.Remove(filepath.Join(env.server, "chunks", oldID[:2], oldID))
	v2 := bytes.Repeat([]byte{2}, 4096)
	sameTimeRewrite(v2)
	if _, e := a.pushProfile(p.ID); e != nil {
		t.Fatalf("push with a stale cache entry: %v", e)
	}
	ids, _ := manifestIDs(t, a, p.ID)
	if refs := env.serverRefs(p.ID); !sameList(refs, ids) {
		t.Fatal("refs differ from the manifest")
	}
	for _, id := range ids {
		if id == oldID {
			t.Fatal("manifest still references the vanished content")
		}
	}
	if _, e := b.pullProfile(p.ID, 0); e != nil || !bytes.Equal(readRel(b, p.ID, swIndex), v2) {
		t.Fatalf("pull after push: %v", e)
	}
	// same again through 修复云端 (hashes everything afresh)
	v2ID := chunkID(key, v2)
	_ = os.Remove(filepath.Join(env.server, "chunks", v2ID[:2], v2ID))
	v3 := bytes.Repeat([]byte{3}, 4096)
	sameTimeRewrite(v3)
	r, e := a.RepairCloud(p.ID)
	if e != nil || r.Uploaded < 1 {
		t.Fatalf("repair: %+v %v", r, e)
	}
	if _, e := b.pullProfile(p.ID, 0); e != nil || !bytes.Equal(readRel(b, p.ID, swIndex), v3) {
		t.Fatalf("pull after repair: %v", e)
	}
}

// A file that keeps changing fails with the Chinese message naming it, and
// nothing is committed.
func TestVolatileFileKeepsChanging(t *testing.T) {
	env := startRepServer(t)
	a := env.dev("a", "dev-A")
	p, _ := repProfile(t, a, "busy", 1<<20)
	dir := a.ProfileDir(p.ID)
	writeRel(t, dir, swIndex, []byte("first"))
	defer func(d time.Duration) { rechunkPause = d }(rechunkPause)
	rechunkPause = 10 * time.Millisecond
	n := 0
	testHookAfterMissing = func() { writeRel(t, dir, swIndex, []byte("second")) }
	testHookRechunkRead = func(rel string, attempt int) {
		n++
		writeRel(t, dir, rel, []byte(strings.Repeat("x", n)))
	}
	defer func() { testHookAfterMissing, testHookRechunkRead = nil, nil }()
	_, e := a.pushProfile(p.ID)
	want := "文件 " + swIndex + " 在同步过程中一直在变化，说明浏览器或残留进程仍在写入，请关闭后重试"
	if e == nil || e.Error() != want {
		t.Fatalf("got %v", e)
	}
	if n != rechunkRetries+1 {
		t.Fatalf("snapshot attempts %d, want %d", n, rechunkRetries+1)
	}
	if v, _ := env.index()[p.ID]["version"].(float64); v != 0 {
		t.Fatalf("a version was committed: %v", v)
	}
}

func excludedFixture(t *testing.T, dir string) (skipped, kept []string) {
	skipped = []string{
		"GrShaderCache/index", "GrShaderCache/data_0", "DawnGraphiteCache/index", "GraphiteDawnCache/index",
		"DawnWebGPUCache/index", "Default/DawnCache/data_1", "ShaderCache/index", "Default/GPUCache/index",
		"BrowserMetrics/BrowserMetrics-1.pma", "BrowserMetrics-spare.pma", "optimization_guide_model_store/1/model.tflite",
		"Default/blob_storage/0d6f/1", "extensions_crx_cache/x.crx", "Default/Media Cache/index",
		"lockfile", "Default/Cache/Cache_Data/index", "Default/Code Cache/js/index",
	}
	kept = []string{
		swIndex, "Default/Service Worker/Database/CURRENT", "Default/Service Worker/CacheStorage/abc/def/index",
		"Default/IndexedDB/https_x.com_0.indexeddb.leveldb/000003.log", "Default/Network/Cookies", "Default/Local Storage/leveldb/CURRENT",
		"Default/segmentation_platform/ukm_db", "Default/lockfile", "Local State",
	}
	for _, r := range append(append([]string{}, skipped...), kept...) {
		writeRel(t, dir, r, []byte(r))
	}
	return
}

// Pure caches stay local, site data does not; same list for the chunked and
// the legacy zip format.
func TestExcludedCacheDirs(t *testing.T) {
	dir := t.TempDir()
	skipped, kept := excludedFixture(t, dir)
	got := map[string]bool{}
	if e := walkSyncFiles(dir, func(rel string, _ os.FileInfo) error { got[rel] = true; return nil }); e != nil {
		t.Fatal(e)
	}
	zb, e := zipDir(dir, &Profile{ID: "x"})
	if e != nil {
		t.Fatal(e)
	}
	zr, _ := zip.NewReader(bytes.NewReader(zb), int64(len(zb)))
	inZip := map[string]bool{}
	for _, f := range zr.File {
		inZip[f.Name] = true
	}
	for _, r := range skipped {
		if got[r] || inZip[r] {
			t.Errorf("%s should not be synced", r)
		}
	}
	for _, r := range kept {
		if !got[r] || !inZip[r] {
			t.Errorf("%s must be synced", r)
		}
	}
}

// Backward compatibility: a manifest made before a directory was excluded
// still restores its files.
func TestExcludedCacheOldManifestRestores(t *testing.T) {
	env := startRepServer(t)
	a, b := env.dev("a", "dev-A"), env.dev("b", "dev-B")
	p, _ := repProfile(t, a, "compat", 1<<20)
	writeRel(t, a.ProfileDir(p.ID), "GrShaderCache/index", []byte("legacy shader index"))
	delete(syncExcludedDirs, "grshadercache") // as v0.8.8 did
	_, e := a.pushProfile(p.ID)
	syncExcludedDirs["grshadercache"] = true
	if e != nil {
		t.Fatal(e)
	}
	if _, e = b.pullProfile(p.ID, 0); e != nil {
		t.Fatal(e)
	}
	if string(readRel(b, p.ID, "GrShaderCache/index")) != "legacy shader index" {
		t.Fatal("old manifest lost the file on restore")
	}
	// the next push from b no longer carries it
	if _, e = b.pushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	_, m := manifestIDs(t, b, p.ID)
	for _, f := range m.Files {
		if strings.HasPrefix(f.Path, "GrShaderCache/") {
			t.Fatal("excluded cache uploaded again")
		}
	}
}

// 上传 / 修复云端 refuse a running profile, and one whose directory a leftover
// browser process still holds.
func TestRunningProfileRefused(t *testing.T) {
	env := startRepServer(t)
	a := env.dev("a", "dev-A")
	p, _ := repProfile(t, a, "running", 1<<20)
	if _, e := a.pushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	runReg.add(p.ID, &RunState{ProfileID: p.ID}, nil)
	for name, f := range map[string]func() error{
		"push":   func() error { return a.PushProfile(p.ID) },
		"repair": func() error { _, e := a.RepairCloud(p.ID); return e },
		"inner":  func() error { var n int64; return a.pushProfileInner(p.ID, &n, a.newTransfer(p.ID, "up"), nil) },
	} {
		if e := f(); e == nil || e.Error() != "环境正在运行，请先关闭后再修复/上传" {
			t.Errorf("%s while running: %v", name, e)
		}
	}
	runReg.mu.Lock()
	delete(runReg.m, p.ID)
	runReg.mu.Unlock()

	testHookProfileHeld = func(string) bool { return true }
	defer func() { testHookProfileHeld = nil }()
	for name, f := range map[string]func() error{
		"push":   func() error { return a.PushProfile(p.ID) },
		"repair": func() error { _, e := a.RepairCloud(p.ID); return e },
	} {
		e := f()
		var le *leftoverError
		if !errors.As(e, &le) || !strings.HasPrefix(e.Error(), "检测到该环境仍有浏览器进程在后台运行（可能是残留进程），请在任务管理器结束后重试") {
			t.Errorf("%s with a leftover process: %v", name, e)
		}
	}
	testHookProfileHeld = nil
	if _, e := a.RepairCloud(p.ID); e != nil {
		t.Fatalf("repair once free: %v", e)
	}
}

func TestCmdlineHasDir(t *testing.T) {
	d := `C:\Users\me\GhostBrowser\profiles\abc`
	for cmd, want := range map[string]bool{
		`"C:\k\chrome.exe" --user-data-dir="C:\Users\me\GhostBrowser\profiles\abc" --type=gpu-process`: true,
		`chrome.exe --user-data-dir=c:/users/me/ghostbrowser/profiles/abc`:                               true,
		`chrome_crashpad_handler.exe --database=C:\Users\me\GhostBrowser\profiles\abc\Crashpad`:           true,
		`chrome.exe --user-data-dir="C:\Users\me\GhostBrowser\profiles\abcd"`:                             false,
		`notepad.exe C:\other`: false,
	} {
		if got := cmdlineHasDir(cmd, d); got != want {
			t.Errorf("%s: %v", cmd, got)
		}
	}
}

// Live: a real Chromium on the profile directory is detected (SingletonLock),
// listed as a leftover and ended by KillLeftoverProcesses.
func TestRunningProfileLiveChromium(t *testing.T) {
	chrome := os.Getenv("GHOST_CHROME")
	if chrome == "" {
		t.Skip("set GHOST_CHROME")
	}
	d := t.TempDir()
	s := &Store{dir: d, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}, Settings: Settings{ProfilesDir: filepath.Join(d, "profiles")}}
	p := s.PutProfile(Profile{Name: "live", Fingerprint: GenerateFingerprint("windows", "141")})
	dir := s.ProfileDir(p.ID)
	_ = os.MkdirAll(dir, 0700)
	cmd := exec.Command(chrome, "--headless=new", "--no-sandbox", "--disable-gpu", "--user-data-dir="+dir, "about:blank")
	if e := cmd.Start(); e != nil {
		t.Fatal(e)
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	defer func() { _ = cmd.Process.Kill() }()
	for i := 0; !profileDirHeld(dir); i++ {
		if i > 100 {
			t.Skip("this Chromium build does not take the profile lock in headless mode")
		}
		time.Sleep(100 * time.Millisecond)
	}
	e := s.profileBusy(p.ID)
	var le *leftoverError
	if !errors.As(e, &le) || len(le.pids) == 0 || !strings.Contains(e.Error(), "结束残留进程") {
		t.Fatalf("busy: %v", e)
	}
	n, e := s.KillLeftoverProcesses(p.ID)
	if e != nil || n == 0 {
		t.Fatalf("kill: %d %v", n, e)
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("chromium still running")
	}
	if profileDirHeld(dir) {
		t.Fatal("stale SingletonLock still reported as held")
	}
	if e := s.profileBusy(p.ID); e != nil {
		t.Fatalf("after kill: %v", e)
	}
}
