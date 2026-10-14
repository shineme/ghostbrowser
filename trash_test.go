package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func trashStore(t *testing.T) *Store {
	d := t.TempDir()
	return &Store{dir: d, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}, Settings: Settings{ProfilesDir: filepath.Join(d, "profiles")}}
}

func mkProfileWithData(t *testing.T, s *Store, name string) Profile {
	p := s.PutProfile(Profile{Name: name, InlineProxy: &Proxy{Type: "http", Host: "1.2.3.4", Port: 8080, User: "u", Pass: "secret"}})
	_ = os.MkdirAll(filepath.Join(s.ProfileDir(p.ID), "Default"), 0o700)
	if e := os.WriteFile(filepath.Join(s.ProfileDir(p.ID), "Default", "marker.txt"), []byte(name), 0o600); e != nil {
		t.Fatal(e)
	}
	return p
}

func inList(s *Store, id string) bool {
	for _, p := range s.ListProfiles() {
		if p.ID == id {
			return true
		}
	}
	return false
}

func TestTrashDeleteRestorePurge(t *testing.T) {
	s := trashStore(t)
	a := &autoSyncer{s: s, busy: map[string]bool{}}
	p := mkProfileWithData(t, s, "店铺A-主号")
	keep := mkProfileWithData(t, s, "keep")

	res := a.DeleteProfiles("ui", []string{p.ID}, "local", false)
	if len(res) != 1 || !res[0].Local {
		t.Fatalf("delete: %+v", res)
	}
	if inList(s, p.ID) || !inList(s, keep.ID) {
		t.Fatal("trashed profile still listed (or the other one vanished)")
	}
	if _, ok := s.GetProfile(p.ID); ok {
		t.Fatal("GetProfile (launch path) still returns a trashed profile")
	}
	if !dirGone(s, p.ID) {
		t.Fatal("data dir not moved out of the profiles folder")
	}
	moved := filepath.Join(s.Settings.ProfilesDir, trashDirName, p.ID, "Default", "marker.txt")
	if b, e := os.ReadFile(moved); e != nil || string(b) != "店铺A-主号" {
		t.Fatalf("data not retained in .trash: %v", e)
	}
	tl := s.ListTrash()
	if len(tl) != 1 || tl[0].ID != p.ID || !tl[0].Trashed || tl[0].TrashedAt == 0 || !tl[0].HasData || tl[0].Size == 0 || tl[0].CloudNote != "仅本机" {
		t.Fatalf("trash list: %+v", tl)
	}
	// persisted across a restart
	s2 := &Store{dir: s.dir, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}}
	s2.load()
	if len(s2.Trash) != 1 || len(s2.Profiles) != 1 {
		t.Fatalf("reload: trash=%d profiles=%d", len(s2.Trash), len(s2.Profiles))
	}

	// restore
	rr := s.RestoreFromTrash("ui", []string{p.ID})
	if len(rr) != 1 || rr[0].Error != "" || rr[0].ID != p.ID {
		t.Fatalf("restore: %+v", rr)
	}
	if !inList(s, p.ID) || len(s.ListTrash()) != 0 {
		t.Fatal("restore did not bring it back")
	}
	if b, e := os.ReadFile(filepath.Join(s.ProfileDir(p.ID), "Default", "marker.txt")); e != nil || string(b) != "店铺A-主号" {
		t.Fatalf("data dir not restored: %v", e)
	}
	rp, _ := s.GetProfile(p.ID)
	if rp.Trashed || rp.TrashedAt != 0 || rp.TrashDir != "" || rp.InlineProxy == nil {
		t.Fatalf("restored record: %+v", rp)
	}

	// purge
	a.DeleteProfiles("ui", []string{p.ID}, "local", false)
	trDir := filepath.Join(s.Settings.ProfilesDir, trashDirName, p.ID)
	if n := s.PurgeTrash("ui", []string{p.ID}, "彻底删除"); n != 1 {
		t.Fatalf("purge n=%d", n)
	}
	if _, e := os.Stat(trDir); !os.IsNotExist(e) {
		t.Fatal("purge left the data folder")
	}
	if len(s.ListTrash()) != 0 || inList(s, p.ID) {
		t.Fatal("purged profile still somewhere")
	}
	s3 := &Store{dir: s.dir, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}}
	s3.load()
	if _, ok := s3.Trash[p.ID]; ok {
		t.Fatal("purged record (with proxy credentials) still in data.json")
	}
	// audit
	seen := map[string]bool{}
	for _, e := range s.AuditQuery(AuditFilter{}).Items {
		seen[e.Action] = true
	}
	for _, act := range []string{"profile.delete", "profile.restore", "profile.purge"} {
		if !seen[act] {
			t.Errorf("no audit entry %s (%v)", act, seen)
		}
	}

	// empty
	a.DeleteProfiles("ui", []string{keep.ID}, "local", false)
	if n := s.EmptyTrash("ui"); n != 1 || len(s.ListTrash()) != 0 {
		t.Fatalf("empty: %d", n)
	}
}

func TestTrashAutoPurge(t *testing.T) {
	s := trashStore(t)
	old := mkProfileWithData(t, s, "old")
	fresh := mkProfileWithData(t, s, "fresh")
	s.TrashProfile(old.ID, false)
	s.TrashProfile(fresh.ID, false)
	s.mu.Lock()
	s.Trash[old.ID].TrashedAt = time.Now().Add(-31 * 24 * time.Hour).Unix()
	s.Trash[fresh.ID].TrashedAt = time.Now().Add(-29 * 24 * time.Hour).Unix()
	s.saveLocked()
	s.mu.Unlock()
	if n := s.PurgeExpiredTrash(); n != 1 {
		t.Fatalf("default 30 days: purged %d", n)
	}
	if _, ok := s.trashed(old.ID); ok {
		t.Fatal("expired entry kept")
	}
	if _, e := os.Stat(filepath.Join(s.Settings.ProfilesDir, trashDirName, old.ID)); !os.IsNotExist(e) {
		t.Fatal("expired data kept")
	}
	// custom trashDays
	s.Settings.TrashDays = 7
	if n := s.PurgeExpiredTrash(); n != 1 || len(s.ListTrash()) != 0 {
		t.Fatalf("trashDays=7: purged %d", n)
	}
}

// GHOST_PHP_TEST=php go test -run TestTrashSync .
func TestTrashSync(t *testing.T) {
	env := startCdelServer(t)
	userLib.open(t.TempDir()) // another test may have left the library pointing at a removed temp dir
	A := env.device("PC-A", "dev-A")
	B := env.device("PC-B", "dev-B")
	for i := 0; A.s.TestSync() != nil; i++ {
		if i > 100 {
			t.Fatal("php not ready")
		}
		time.Sleep(100 * time.Millisecond)
	}
	synced := newSyncedProfile(t, A, "synced")
	viaB := newSyncedProfile(t, A, "deleted-by-B")
	if r := B.a.Reconcile(); r.Failed != 0 {
		t.Fatalf("B reconcile: %+v", r)
	}
	_ = A.a.Reconcile() // A takes B's global data first
	localOnly := A.s.PutProfile(Profile{Name: "never-uploaded", Fingerprint: GenerateFingerprint("windows", "141")})
	_ = os.MkdirAll(A.s.ProfileDir(localOnly.ID), 0o700)

	// A trashes a synced and a never-uploaded profile, both "only local".
	A.a.DeleteProfiles("ui", []string{synced.ID, localOnly.ID}, "local", false)
	if r := A.a.Reconcile(); r.Pulled != 0 || r.Failed != 0 {
		t.Fatalf("A reconcile after trash: %+v", r)
	}
	if _, ok := A.s.GetProfile(synced.ID); ok {
		t.Fatal("trashed profile re-pulled from cloud")
	}
	if _, ok := findMeta(env.list(true), localOnly.ID); ok {
		t.Fatal("trashed never-uploaded profile was pushed")
	}
	A = env.restart(A)
	if r := A.a.Reconcile(); r.Pulled != 0 {
		t.Fatalf("after restart: %+v", r)
	}
	if len(A.s.ListTrash()) != 2 {
		t.Fatalf("trash after restart: %+v", A.s.ListTrash())
	}
	// Restore the synced one: auto sync follows it again (marker cleared).
	if rr := A.s.RestoreFromTrash("ui", []string{synced.ID}); rr[0].Error != "" || rr[0].ID != synced.ID {
		t.Fatalf("restore: %+v", rr)
	}
	if _, ok := A.s.localDeleted()[synced.ID]; ok {
		t.Fatal("local-delete marker kept after restore")
	}
	if r := A.a.Reconcile(); r.Failed != 0 {
		t.Fatalf("reconcile after restore: %+v", r)
	}
	if _, ok := A.s.GetProfile(synced.ID); !ok {
		t.Fatal("restored profile lost")
	}

	// B deletes viaB from the cloud: A's copy goes to the trash (not destroyed).
	if res := B.a.DeleteProfiles("ui", []string{viaB.ID}, "cloud", false); len(res) != 1 || res[0].Cloud != "deleted" {
		t.Fatalf("B cloud delete: %+v", res)
	}
	if bt := B.s.ListTrash(); len(bt) != 1 || !bt[0].RestoreAsNew {
		t.Fatalf("B trash: %+v", bt)
	}
	_ = A.a.Reconcile()
	if _, ok := A.s.GetProfile(viaB.ID); ok {
		t.Fatal("tombstone did not remove A's copy")
	}
	it, ok := A.s.trashed(viaB.ID)
	if !ok || !it.TrashCloudDeleted {
		t.Fatalf("A copy not in trash as cloud-deleted: %+v", it)
	}
	// Restoring it creates a new local profile that survives the tombstone and is uploaded.
	rr := A.s.RestoreFromTrash("ui", []string{viaB.ID})
	if rr[0].Error != "" || rr[0].OldID != viaB.ID || rr[0].ID == viaB.ID {
		t.Fatalf("restore as new: %+v", rr)
	}
	if b, e := os.ReadFile(filepath.Join(A.s.ProfileDir(rr[0].ID), "marker.txt")); e != nil || string(b) != "deleted-by-B" {
		t.Fatalf("data not carried to new id: %v", e)
	}
	if r := A.a.Reconcile(); r.Failed != 0 {
		t.Fatalf("reconcile after restore-as-new: %+v", r)
	}
	if _, ok := A.s.GetProfile(rr[0].ID); !ok {
		t.Fatal("restored-as-new profile removed by tombstone")
	}
	if _, ok := findMeta(env.list(false), rr[0].ID); !ok {
		t.Fatal("restored-as-new profile not uploaded")
	}
}

// A received share (someone else's data) is deleted outright, never kept in
// the recycle bin where it could be restored after the share ended.
func TestTrashSkipsSharedCopy(t *testing.T) {
	s := trashStore(t)
	a := &autoSyncer{s: s, busy: map[string]bool{}}
	p := mkProfileWithData(t, s, "共享-店铺")
	p.ShareID = "a1b2c3d4e5f6a7b8c9d0e1f2"
	s.PutProfile(p)
	res := a.DeleteProfiles("ui", []string{p.ID}, "local", false)
	if len(res) != 1 || !res[0].Local || inList(s, p.ID) {
		t.Fatalf("delete %+v", res)
	}
	if len(s.ListTrash()) != 0 {
		t.Fatal("shared copy must not go to the recycle bin")
	}
	if _, e := os.Stat(s.ProfileDir(p.ID)); !os.IsNotExist(e) {
		t.Fatal("shared copy data must be removed")
	}
}
