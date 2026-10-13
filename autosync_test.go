package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// GHOST_PHP_TEST=php go test -run TestAutoSync
func TestAutoSyncServerFirst(t *testing.T) {
	php := os.Getenv("GHOST_PHP_TEST")
	if php == "" {
		t.Skip("set GHOST_PHP_TEST to php")
	}
	project, _ := os.Getwd()
	root := t.TempDir()
	port := freePort()
	base := "http://127.0.0.1:" + itoa(port)
	token := strings.Repeat("test-private-token-", 4)
	cmd := exec.Command(php, "-S", "127.0.0.1:"+itoa(port), "-t", filepath.Join(project, "cloud-server", "public"), filepath.Join(project, "cloud-server", "router.php"))
	cmd.Env = append(os.Environ(), "GHOST_SYNC_TOKEN="+token, "GHOST_SYNC_DATA_DIR="+filepath.Join(root, "server"))
	if e := cmd.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	mk := func(name, device string) (*Store, *autoSyncer) {
		d := filepath.Join(root, name)
		_ = os.MkdirAll(d, 0700)
		s := &Store{dir: d, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}, Settings: Settings{ProfilesDir: filepath.Join(d, "profiles"), SyncServer: base, SyncToken: token, SyncPassword: "pw-123456", DeviceID: device}}
		return s, &autoSyncer{s: s, busy: map[string]bool{}, emit: func(e SyncEvent) { t.Logf("%s: %+v", name, e) }}
	}
	a, autoA := mk("a", "dev-A")
	b, autoB := mk("b", "dev-B")
	for i := 0; a.TestSync() != nil; i++ {
		if i > 100 {
			t.Fatal("php not ready")
		}
		time.Sleep(100 * time.Millisecond)
	}
	p := a.PutProfile(Profile{Name: "auto", Fingerprint: GenerateFingerprint("windows", "141")})
	_ = os.MkdirAll(a.ProfileDir(p.ID), 0700)
	_ = os.WriteFile(filepath.Join(a.ProfileDir(p.ID), "marker.txt"), []byte("v1"), 0600)

	// A: local-only profile is uploaded on start-up.
	if r := autoA.Reconcile(); r.Pushed != 1 || r.Failed != 0 {
		t.Fatalf("A reconcile: %+v", r)
	}
	// B: empty device downloads it.
	if r := autoB.Reconcile(); r.Pulled != 1 || r.Failed != 0 {
		t.Fatalf("B reconcile: %+v", r)
	}
	if got, _ := os.ReadFile(filepath.Join(b.ProfileDir(p.ID), "marker.txt")); string(got) != "v1" {
		t.Fatalf("B marker = %q", got)
	}
	// B runs the browser and closes it: pushed as v2.
	bp, _ := b.GetProfile(p.ID)
	bp.LastRun = time.Now().Unix() + 1
	b.PutProfile(*bp)
	_ = os.WriteFile(filepath.Join(b.ProfileDir(p.ID), "marker.txt"), []byte("v2"), 0600)
	if e := retry(1, b.PushProfile, p.ID); e != nil {
		t.Fatal(e)
	}
	// A also has an unsynced local run, but the server is newer: server wins.
	ap, _ := a.GetProfile(p.ID)
	ap.LastRun = time.Now().Unix() + 5
	a.PutProfile(*ap)
	if r := autoA.Reconcile(); r.Pulled != 1 || r.Pushed != 0 {
		t.Fatalf("A second reconcile: %+v", r)
	}
	if got, _ := os.ReadFile(filepath.Join(a.ProfileDir(p.ID), "marker.txt")); string(got) != "v2" {
		t.Fatalf("A marker = %q", got)
	}
	// Nothing left to do.
	if r := autoA.Reconcile(); r.Pulled+r.Pushed+r.Failed != 0 {
		t.Fatalf("A idle reconcile: %+v", r)
	}
}
