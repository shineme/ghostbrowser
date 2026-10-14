package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// GHOST_PHP_TEST=php go test -run TestCloudDelete .
type cdelEnv struct {
	t      *testing.T
	root   string
	base   string
	token  string
	server string
}

func startCdelServer(t *testing.T) *cdelEnv {
	php := os.Getenv("GHOST_PHP_TEST")
	if php == "" {
		t.Skip("set GHOST_PHP_TEST to php")
	}
	project, _ := os.Getwd()
	root := t.TempDir()
	port := freePort()
	env := &cdelEnv{t: t, root: root, base: "http://127.0.0.1:" + itoa(port) + "/ghostsync.php", token: strings.Repeat("test-private-token-", 4), server: filepath.Join(root, "server")}
	cmd := exec.Command(php, "-S", "127.0.0.1:"+itoa(port), "-t", filepath.Join(project, "cloud-server"))
	cmd.Env = append(os.Environ(), "GHOST_SYNC_TOKEN="+env.token, "GHOST_SYNC_DATA_DIR="+env.server)
	if e := cmd.Start(); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	return env
}

type cdelDev struct {
	s    *Store
	a    *autoSyncer
	mu   sync.Mutex
	evs  []SyncEvent
	name string
}

func (d *cdelDev) events(phase string) []SyncEvent {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := []SyncEvent{}
	for _, e := range d.evs {
		if e.Phase == phase {
			out = append(out, e)
		}
	}
	return out
}

func (env *cdelEnv) device(name, id string) *cdelDev {
	d := filepath.Join(env.root, name)
	_ = os.MkdirAll(d, 0700)
	s := &Store{dir: d, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}, Settings: Settings{ProfilesDir: filepath.Join(d, "profiles"), SyncServer: env.base, SyncToken: env.token, SyncPassword: "pw-123456", DeviceID: id, DeviceName: name}}
	return env.wrap(name, s)
}

func (env *cdelEnv) wrap(name string, s *Store) *cdelDev {
	dev := &cdelDev{s: s, name: name}
	dev.a = &autoSyncer{s: s, busy: map[string]bool{}, emit: func(e SyncEvent) {
		if e.Phase != "busy" && e.Phase != "idle" {
			env.t.Logf("%s: %+v", name, e)
		}
		dev.mu.Lock()
		dev.evs = append(dev.evs, e)
		dev.mu.Unlock()
	}}
	return dev
}

// restart reloads a device from its data.json, like a fresh app start.
func (env *cdelEnv) restart(d *cdelDev) *cdelDev {
	s := &Store{dir: d.s.dir, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}}
	s.load()
	return env.wrap(d.name, s)
}

func (env *cdelEnv) raw(method, path, device string, body string) (int, string) {
	req, _ := http.NewRequest(method, syncURL(env.base, path), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+env.token)
	req.Header.Set("Content-Type", "application/json")
	if device != "" {
		req.Header.Set("X-Device-ID", device)
	}
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		env.t.Fatal(e)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func (env *cdelEnv) list(tombs bool) []cloudMeta {
	p := "/api/profiles"
	if tombs {
		p += "?tombstones=1"
	}
	code, body := env.raw("GET", p, "", "")
	if code != 200 {
		env.t.Fatalf("list %d %s", code, body)
	}
	var out []cloudMeta
	_ = json.Unmarshal([]byte(body), &out)
	return out
}

func findMeta(l []cloudMeta, id string) (cloudMeta, bool) {
	for _, c := range l {
		if c.ID == id {
			return c, true
		}
	}
	return cloudMeta{}, false
}

func newSyncedProfile(t *testing.T, d *cdelDev, name string) Profile {
	p := d.s.PutProfile(Profile{Name: name, Fingerprint: GenerateFingerprint("windows", "141")})
	_ = os.MkdirAll(d.s.ProfileDir(p.ID), 0700)
	_ = os.WriteFile(filepath.Join(d.s.ProfileDir(p.ID), "marker.txt"), []byte(name), 0600)
	if e := d.s.PushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	return p
}

func dirGone(s *Store, id string) bool {
	_, e := os.Stat(s.ProfileDir(id))
	return os.IsNotExist(e)
}

func TestCloudDeleteTwoDevices(t *testing.T) {
	env := startCdelServer(t)
	A := env.device("PC-A", "dev-A")
	B := env.device("PC-B", "dev-B")
	for i := 0; A.s.TestSync() != nil; i++ {
		if i > 100 {
			t.Fatal("php not ready")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if si, e := A.s.ServerInfo(true); e != nil || !si.has("tombstones") {
		t.Fatalf("ping features: %+v %v", si, e)
	}
	pLocal := newSyncedProfile(t, A, "only-local")
	pCloud := newSyncedProfile(t, A, "cloud-del")
	pRun := newSyncedProfile(t, A, "running-on-B")
	pLaunch := newSyncedProfile(t, A, "launch-on-B")
	if r := B.a.Reconcile(); r.Failed != 0 || r.Pulled < 4 {
		t.Fatalf("B initial reconcile: %+v", r)
	}
	_ = A.a.Reconcile()

	// (a) A deletes only locally: never pulled back, even after a restart.
	res := A.a.DeleteProfiles("ui", []string{pLocal.ID}, "local", false)
	if len(res) != 1 || !res[0].Local || res[0].Cloud != "" {
		t.Fatalf("local delete: %+v", res)
	}
	if _, ok := A.s.GetProfile(pLocal.ID); ok || !dirGone(A.s, pLocal.ID) {
		t.Fatal("local copy still on A")
	}
	A = env.restart(A)
	if r := A.a.Reconcile(); r.Pulled != 0 || r.Failed != 0 {
		t.Fatalf("A reconcile after local delete: %+v", r)
	}
	if _, ok := A.s.GetProfile(pLocal.ID); ok {
		t.Fatal("A re-pulled a profile deleted only locally")
	}
	if _, ok := findMeta(env.list(false), pLocal.ID); !ok {
		t.Fatal("cloud copy vanished after a local-only delete")
	}
	co := A.s.CloudOnly()
	var item CloudOnlyItem
	for _, it := range co.Items {
		if it.ID == pLocal.ID {
			item = it
		}
	}
	if !item.LocalDeleted || item.CloudNewer || item.Name != "only-local" {
		t.Fatalf("cloud-only item: %+v (%s)", item, co.Error)
	}
	// B uploads a newer version: A still does not pull, but shows "云端有更新".
	bp, _ := B.s.GetProfile(pLocal.ID)
	bp.LastRun = time.Now().Unix() + 2
	B.s.PutProfile(*bp)
	if e := B.s.PushProfile(pLocal.ID); e != nil {
		t.Fatal(e)
	}
	if r := A.a.Reconcile(); r.Pulled != 0 {
		t.Fatalf("A pulled a locally deleted profile after a cloud update: %+v", r)
	}
	for _, it := range A.s.CloudOnly().Items {
		if it.ID == pLocal.ID && !it.CloudNewer {
			t.Fatalf("cloudNewer not set: %+v", it)
		}
	}
	// Restore brings it back and auto sync follows it again.
	if e := A.s.RestoreLocalDeleted(pLocal.ID); e != nil {
		t.Fatal(e)
	}
	if got, _ := os.ReadFile(filepath.Join(A.s.ProfileDir(pLocal.ID), "marker.txt")); string(got) != "only-local" {
		t.Fatalf("restored marker %q", got)
	}
	if len(A.s.localDeleted()) != 0 {
		t.Fatal("marker kept after restore")
	}
	t.Log("PASS (a) local-only delete: not re-pulled, listed, restorable")

	// (b) A deletes in the cloud: tombstone, B deletes its copy, no resurrection.
	bc, _ := B.s.GetProfile(pCloud.ID)
	bc.LastRun = time.Now().Unix() + 3 // B has unsynced changes (stale device)
	B.s.PutProfile(*bc)
	before, _ := findMeta(env.list(false), pCloud.ID)
	oldSha := before.Sha256
	if _, e := os.Stat(filepath.Join(env.server, "blobs", oldSha+".gbe")); e != nil {
		t.Fatalf("blob before delete: %v", e)
	}
	res = A.a.DeleteProfiles("ui", []string{pCloud.ID}, "cloud", false)
	if len(res) != 1 || !res[0].Local || res[0].Cloud != "deleted" || res[0].Message != "" {
		t.Fatalf("cloud delete: %+v", res)
	}
	if tm, ok := findMeta(env.list(true), pCloud.ID); !ok || !tm.Deleted || tm.DeletedByName != "PC-A" || tm.Version != 2 {
		t.Fatalf("tombstone: %+v %v", tm, ok)
	}
	// (e) old clients never see tombstones.
	if _, ok := findMeta(env.list(false), pCloud.ID); ok {
		t.Fatal("list without tombstones=1 returned a deleted profile")
	}
	if _, e := os.Stat(filepath.Join(env.server, "blobs", oldSha+".gbe")); !os.IsNotExist(e) {
		t.Fatal("deleted profile's blob was not released")
	}
	if _, e := os.Stat(filepath.Join(env.server, "refs", oldSha+".txt")); !os.IsNotExist(e) {
		t.Fatal("deleted profile's chunk refs were not released")
	}
	t.Log("PASS (e) old-client list excludes tombstones")
	// Stale push from B is refused.
	if e := B.s.PushProfile(pCloud.ID); !isCloudDeleted(e) {
		t.Fatalf("stale push not refused: %v", e)
	}
	if e := B.s.PullProfile(pCloud.ID); !isCloudDeleted(e) {
		t.Fatalf("pull of a deleted profile: %v", e)
	}
	if _, ok := findMeta(env.list(false), pCloud.ID); ok {
		t.Fatal("stale push resurrected the profile")
	}
	// (c) B is running pRun when the tombstone arrives.
	runReg.add(pRun.ID, &RunState{ProfileID: pRun.ID}, nil)
	oldExit := runReg.onExit
	runReg.onExit = func(id string) { B.a.AfterExit(id) }
	defer func() { runReg.onExit = oldExit }()
	if res = A.a.DeleteProfiles("ui", []string{pRun.ID}, "cloud", false); res[0].Cloud != "deleted" {
		t.Fatalf("delete pRun: %+v", res)
	}
	if res = A.a.DeleteProfiles("ui", []string{pLaunch.ID}, "cloud", false); res[0].Cloud != "deleted" {
		t.Fatalf("delete pLaunch: %+v", res)
	}
	// BeforeLaunch on B refuses and removes.
	if e := B.a.BeforeLaunch(pLaunch.ID); e == nil || !strings.Contains(e.Error(), "PC-A") {
		t.Fatalf("BeforeLaunch: %v", e)
	}
	if _, ok := B.s.GetProfile(pLaunch.ID); ok || !dirGone(B.s, pLaunch.ID) {
		t.Fatal("BeforeLaunch kept a deleted profile")
	}
	r := B.a.Reconcile()
	if r.Pulled != 0 {
		t.Fatalf("B pulled after deletions: %+v", r)
	}
	if _, ok := B.s.GetProfile(pCloud.ID); ok || !dirGone(B.s, pCloud.ID) {
		t.Fatal("B kept a profile deleted in the cloud")
	}
	if ev := B.events("deleted"); len(ev) < 1 || !strings.Contains(ev[len(ev)-1].Message, "PC-A") {
		t.Fatalf("deleted events: %+v", ev)
	}
	if _, ok := B.s.GetProfile(pRun.ID); !ok {
		t.Fatal("running profile deleted while its browser was open")
	}
	if len(B.events("deleted-pending")) != 1 {
		t.Fatalf("deleted-pending events: %+v", B.events("deleted-pending"))
	}
	runReg.remove(pRun.ID) // the browser closes
	B.a.Drain(15 * time.Second)
	if _, ok := B.s.GetProfile(pRun.ID); ok || !dirGone(B.s, pRun.ID) {
		t.Fatal("running profile not deleted after exit")
	}
	if _, ok := findMeta(env.list(false), pRun.ID); ok {
		t.Fatal("exit push resurrected pRun")
	}
	t.Log("PASS (c) running profile deleted after exit; BeforeLaunch refuses")
	// Neither device pulls anything back.
	if r := A.a.Reconcile(); r.Pulled != 0 || r.Failed != 0 {
		t.Fatalf("A final reconcile: %+v", r)
	}
	if r := B.a.Reconcile(); r.Pulled != 0 || r.Failed != 0 {
		t.Fatalf("B final reconcile: %+v", r)
	}
	B2 := env.restart(B)
	if r := B2.a.Reconcile(); r.Pulled != 0 || r.Failed != 0 {
		t.Fatalf("B restart reconcile: %+v", r)
	}
	found := false
	for _, e := range auditOf(B.s) {
		if e.Action == "sync.delete-local" && e.Target == pCloud.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("no sync.delete-local audit entry on B")
	}
	t.Log("PASS (b) cloud delete: tombstone, B deletes, stale push refused, no re-pull")

	// Explicit re-creation over a tombstone works.
	re := A.s.PutProfile(Profile{ID: pCloud.ID, Name: "recreated", Fingerprint: GenerateFingerprint("windows", "141")})
	_ = os.MkdirAll(A.s.ProfileDir(re.ID), 0700)
	if e := A.s.PushProfile(re.ID); !isCloudDeleted(e) {
		t.Fatalf("push without restore over a tombstone: %v", e)
	}
	if e := A.s.PushProfileRestore(re.ID); e != nil {
		t.Fatal(e)
	}
	if m, ok := findMeta(env.list(false), re.ID); !ok || m.Deleted || m.Version != 3 {
		t.Fatalf("recreated: %+v %v", m, ok)
	}

	// Tombstone retention: an old tombstone is purged on the next list.
	idxFile := filepath.Join(env.server, "index.json")
	raw, _ := os.ReadFile(idxFile)
	var idx map[string]map[string]map[string]any
	_ = json.Unmarshal(raw, &idx)
	idx["profiles"][pRun.ID]["deleted_at"] = time.Now().Unix() - 91*86400
	raw, _ = json.Marshal(idx)
	_ = os.WriteFile(idxFile, raw, 0600)
	if _, ok := findMeta(env.list(true), pRun.ID); ok {
		t.Fatal("expired tombstone not purged")
	}
	if _, ok := findMeta(env.list(true), pLaunch.ID); !ok {
		t.Fatal("fresh tombstone purged")
	}
}

func auditOf(s *Store) []AuditEntry {
	audit.mu.Lock()
	defer audit.mu.Unlock()
	audit.dir, audit.list = "", nil
	s.auditLoadLocked()
	return append([]AuditEntry(nil), audit.list...)
}

func TestCloudDeleteOfflineAndLock(t *testing.T) {
	env := startCdelServer(t)
	A := env.device("PC-A", "dev-A")
	B := env.device("PC-B", "dev-B")
	for i := 0; A.s.TestSync() != nil; i++ {
		if i > 100 {
			t.Fatal("php not ready")
		}
		time.Sleep(100 * time.Millisecond)
	}
	pOff := newSyncedProfile(t, A, "offline")
	pLock := newSyncedProfile(t, A, "locked")
	if r := B.a.Reconcile(); r.Pulled < 2 || r.Failed != 0 {
		t.Fatalf("B reconcile: %+v", r)
	}
	_ = A.a.Reconcile() // settle global data

	// (d) offline: local copy removed, cloud delete queued and retried.
	good := A.s.Settings.SyncServer
	A.s.mu.Lock()
	A.s.Settings.SyncServer = "http://127.0.0.1:" + itoa(freePort()) + "/ghostsync.php"
	A.s.mu.Unlock()
	res := A.a.DeleteProfiles("ui", []string{pOff.ID}, "cloud", false)
	if len(res) != 1 || !res[0].Local || res[0].Cloud != "queued" || res[0].Error == "" {
		t.Fatalf("offline delete: %+v", res)
	}
	if A.a.RetryPendingDeletes() != 1 {
		t.Fatal("pending delete lost while offline")
	}
	A = env.restart(A) // the queue survives a restart
	if q := A.s.pendingDeletes(); len(q) != 1 || q[pOff.ID].Name != "offline" || q[pOff.ID].Attempts < 2 {
		t.Fatalf("persisted queue: %+v", q)
	}
	A.s.mu.Lock()
	A.s.Settings.SyncServer = good
	A.s.mu.Unlock()
	if r := A.a.Reconcile(); r.Pulled != 0 || r.Failed != 0 {
		t.Fatalf("A reconcile after reconnect: %+v", r)
	}
	if len(A.s.pendingDeletes()) != 0 {
		t.Fatal("queued delete not sent")
	}
	if _, ok := A.s.GetProfile(pOff.ID); ok {
		t.Fatal("queued delete pulled back")
	}
	if tm, ok := findMeta(env.list(true), pOff.ID); !ok || !tm.Deleted {
		t.Fatalf("no tombstone after retry: %+v", tm)
	}
	if len(A.events("deleted-cloud")) != 1 {
		t.Fatalf("deleted-cloud events: %+v", A.events("deleted-cloud"))
	}
	_ = B.a.Reconcile()
	if _, ok := B.s.GetProfile(pOff.ID); ok {
		t.Fatal("B kept the offline-deleted profile")
	}
	t.Log("PASS (d) offline delete queued, persisted, retried")

	// (f) another device holds the sync lease: 423 unless forced.
	code, body := env.raw("POST", "/api/profiles/"+pLock.ID+"/lock", "dev-B", `{"device":"dev-B"}`)
	if code != 200 {
		t.Fatalf("B lock: %d %s", code, body)
	}
	code, body = env.raw("POST", "/api/profiles/"+pLock.ID+"/delete", "dev-A", `{}`)
	if code != 423 || !strings.Contains(body, "另一台设备") {
		t.Fatalf("raw delete under lease: %d %s", code, body)
	}
	res = A.a.DeleteProfiles("ui", []string{pLock.ID}, "cloud", false)
	if len(res) != 1 || res[0].Cloud != "locked" || res[0].Local || !strings.Contains(res[0].Error, "另一台设备") {
		t.Fatalf("locked delete: %+v", res)
	}
	if _, ok := A.s.GetProfile(pLock.ID); !ok {
		t.Fatal("local copy deleted although the cloud delete was refused")
	}
	res = A.a.DeleteProfiles("ui", []string{pLock.ID}, "cloud", true)
	if res[0].Cloud != "deleted" || !res[0].Local {
		t.Fatalf("forced delete: %+v", res)
	}
	if tm, ok := findMeta(env.list(true), pLock.ID); !ok || !tm.Deleted {
		t.Fatalf("forced tombstone: %+v", tm)
	}
	// B's commit under its old lease is refused too.
	t.Log("PASS (f) lease held by another device -> 423, force overrides")
}
