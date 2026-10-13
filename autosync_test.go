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
	base := "http://127.0.0.1:" + itoa(port) + "/ghostsync.php"
	token := strings.Repeat("test-private-token-", 4)
	cmd := exec.Command(php, "-S", "127.0.0.1:"+itoa(port), "-t", filepath.Join(project, "cloud-server"))
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

	// A pushes v3; B has a stale v2 and gets the start-up prompt.
	ap, _ = a.GetProfile(p.ID)
	ap.LastRun = time.Now().Unix() + 10
	a.PutProfile(*ap)
	_ = os.WriteFile(filepath.Join(a.ProfileDir(p.ID), "marker.txt"), []byte("v3"), 0600)
	if e := retry(1, a.PushProfile, p.ID); e != nil {
		t.Fatal(e)
	}
	b.Settings.SyncPromptSecs, b.Settings.SyncPromptAction = 30, "pull"
	go func() {
		for autoB.Pending() == nil {
			time.Sleep(50 * time.Millisecond)
		}
		pr := autoB.Pending()
		if len(pr.Items) != 1 || pr.Items[0].CloudVersion != 3 || pr.Items[0].LocalVersion != 2 {
			t.Errorf("prompt: %+v", pr)
		}
		autoB.Hold()
		autoB.Decide(nil) // keep local on B
	}()
	if r := autoB.Reconcile(); r.Pulled != 0 || r.Pushed != 1 || r.Failed != 0 {
		t.Fatalf("B keep-local reconcile: %+v", r)
	}
	if got, _ := os.ReadFile(filepath.Join(b.ProfileDir(p.ID), "marker.txt")); string(got) != "v2" {
		t.Fatalf("B kept marker = %q", got)
	}
	// B's copy replaced the cloud (v4); A pulls it via the countdown default.
	a.Settings.SyncPromptSecs, a.Settings.SyncPromptAction = 1, "pull"
	start := time.Now()
	if r := autoA.Reconcile(); r.Pulled != 1 || time.Since(start) < 900*time.Millisecond {
		t.Fatalf("A countdown reconcile: %+v after %v", r, time.Since(start))
	}
	if got, _ := os.ReadFile(filepath.Join(a.ProfileDir(p.ID), "marker.txt")); string(got) != "v2" {
		t.Fatalf("A marker after countdown = %q", got)
	}
	if ap, _ = a.GetProfile(p.ID); ap.CloudVer != 4 {
		t.Fatalf("A cloud version = %d", ap.CloudVer)
	}
}

func TestSyncURL(t *testing.T) {
	if u := syncURL("https://x.com/sync/ghostsync.php", "/api/profiles/ab/blob?version=3"); u != "https://x.com/sync/ghostsync.php?r=%2Fapi%2Fprofiles%2Fab%2Fblob&version=3" {
		t.Fatal(u)
	}
	if u := syncURL("https://x.com", "/api/ping"); u != "https://x.com/api/ping" {
		t.Fatal(u)
	}
}
