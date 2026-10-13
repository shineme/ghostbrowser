package main

import (
	"bytes"
	"crypto/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// GHOST_PHP_TEST=php go test -run TestChunked
func TestChunkedIncrementalAndRestore(t *testing.T) {
	php := os.Getenv("GHOST_PHP_TEST")
	if php == "" {
		t.Skip("set GHOST_PHP_TEST to php")
	}
	project, _ := os.Getwd()
	root := t.TempDir()
	port := freePort()
	base := "http://127.0.0.1:" + itoa(port) + "/ghostsync.php"
	token := strings.Repeat("chunk-private-token-", 3)
	cmd := exec.Command(php, "-S", "127.0.0.1:"+itoa(port), "-t", filepath.Join(project, "cloud-server"))
	cmd.Env = append(os.Environ(), "GHOST_SYNC_TOKEN="+token, "GHOST_SYNC_DATA_DIR="+filepath.Join(root, "server"))
	if e := cmd.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	mk := func(name, device string) *Store {
		d := filepath.Join(root, name)
		_ = os.MkdirAll(d, 0700)
		return &Store{dir: d, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}, Settings: Settings{ProfilesDir: filepath.Join(d, "profiles"), SyncServer: base, SyncToken: token, SyncPassword: "pw-chunks", DeviceID: device}}
	}
	a, b := mk("a", "dev-A"), mk("b", "dev-B")
	for i := 0; a.TestSync() != nil; i++ {
		if i > 100 {
			t.Fatal("php not ready")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if si, e := a.ServerInfo(true); e != nil || !si.has("chunks") || !si.has("history") {
		t.Fatalf("server info: %+v %v", si, e)
	}
	p := a.PutProfile(Profile{Name: "chunky", Fingerprint: GenerateFingerprint("windows", "141")})
	dir := a.ProfileDir(p.ID)
	big := make([]byte, 9<<20) // 3 chunks
	_, _ = rand.Read(big)
	_ = os.MkdirAll(filepath.Join(dir, "Default", "Cache"), 0700)
	_ = os.WriteFile(filepath.Join(dir, "Default", "big.bin"), big, 0600)
	_ = os.WriteFile(filepath.Join(dir, "Default", "Cache", "skip.bin"), []byte("cache"), 0600)
	_ = os.WriteFile(filepath.Join(dir, "small.txt"), []byte("v1"), 0600)
	_ = os.WriteFile(filepath.Join(dir, "empty.txt"), nil, 0600)

	n1, e := a.pushProfile(p.ID)
	if e != nil {
		t.Fatal(e)
	}
	_ = os.WriteFile(filepath.Join(dir, "small.txt"), []byte("v2"), 0600)
	n2, e := a.pushProfile(p.ID)
	if e != nil {
		t.Fatal(e)
	}
	t.Logf("first push %d bytes, second %d bytes", n1, n2)
	if n1 < 9<<20 || n2 > 64<<10 {
		t.Fatalf("not incremental: %d then %d", n1, n2)
	}
	// B pulls the full copy.
	if _, e = b.pullProfile(p.ID, 0); e != nil {
		t.Fatal(e)
	}
	bdir := b.ProfileDir(p.ID)
	if got, _ := os.ReadFile(filepath.Join(bdir, "Default", "big.bin")); !bytes.Equal(got, big) {
		t.Fatal("big file differs")
	}
	if got, _ := os.ReadFile(filepath.Join(bdir, "small.txt")); string(got) != "v2" {
		t.Fatalf("small = %q", got)
	}
	if _, e = os.Stat(filepath.Join(bdir, "empty.txt")); e != nil {
		t.Fatal("empty file missing")
	}
	if _, e = os.Stat(filepath.Join(bdir, "Default", "Cache")); e == nil {
		t.Fatal("cache was synced")
	}
	bp, _ := b.GetProfile(p.ID)
	if bp.CloudVer != 2 || bp.Name != "chunky" || bp.ChunkKey == "" {
		t.Fatalf("B profile: %+v", bp)
	}
	// A second pull reuses local pieces: only the manifest travels.
	got, e := b.pullProfile(p.ID, 0)
	if e != nil || got > 64<<10 {
		t.Fatalf("repeat pull moved %d bytes (%v)", got, e)
	}
	// History and restore of v1.
	h, e := b.CloudHistory(p.ID)
	if e != nil || len(h) != 2 || !h[0].Current || h[1].Version != 1 {
		t.Fatalf("history: %+v %v", h, e)
	}
	if e = b.RestoreVersion(p.ID, 1); e != nil {
		t.Fatal(e)
	}
	if got, _ := os.ReadFile(filepath.Join(bdir, "small.txt")); string(got) != "v1" {
		t.Fatalf("restored small = %q", got)
	}
	bp, _ = b.GetProfile(p.ID)
	if bp.CloudVer != 3 {
		t.Fatalf("restore should upload v3, got v%d", bp.CloudVer)
	}
	log := b.readSyncLog()
	if len(log) == 0 || log[0].Action != "restore" || !log[0].OK {
		t.Fatalf("sync log: %+v", log)
	}
	// A pulls the restored copy.
	if _, e = a.pullProfile(p.ID, 0); e != nil {
		t.Fatal(e)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "small.txt")); string(got) != "v1" {
		t.Fatalf("A small = %q", got)
	}
}
