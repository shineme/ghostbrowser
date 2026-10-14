package main

// Cloud-aware profile deletion.
//
//   - "local": the profile and its data folder are removed from this device
//     only. When the cloud holds a copy, a marker (Store.CloudDel.LocalDeleted)
//     stops auto sync from downloading it again; the cloud tab lists it as
//     "仅云端（已在本机删除）" with 恢复到本机 / 删除云端.
//   - "cloud": the server replaces the profile with a tombstone (data, history
//     and chunk references are released). Other devices see the tombstone in
//     the profile list and delete their own copy; a stale device cannot upload
//     it back. When the server cannot be reached the local copy is removed and
//     the cloud delete is queued (persisted) and retried until it succeeds.

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type localDeletedMark struct {
	Name         string `json:"name"`
	CloudVersion int64  `json:"cloudVersion"` // cloud version when the local copy was deleted
	DeletedAt    int64  `json:"deletedAt"`
}

type pendingCloudDelete struct {
	Name      string `json:"name"`
	QueuedAt  int64  `json:"queuedAt"`
	Attempts  int    `json:"attempts"`
	LastError string `json:"lastError,omitempty"`
	Force     bool   `json:"force,omitempty"`
}

type cloudDeleteState struct {
	LocalDeleted map[string]localDeletedMark   `json:"localDeleted,omitempty"`
	Pending      map[string]pendingCloudDelete `json:"pending,omitempty"`
}

// restorePush marks the next push of an id as an explicit re-creation over a
// cloud tombstone.
var restorePush sync.Map

// ---------------------------------------------------------------- errors

// errCloudDeleted: HTTP 410, the profile was deleted from the cloud.
type errCloudDeleted struct{ msg string }

func (e errCloudDeleted) Error() string { return e.msg }

// errCloudLocked: HTTP 423, another device holds the profile's sync lease.
type errCloudLocked struct{ msg string }

func (e errCloudLocked) Error() string { return e.msg }

func cloudStatusErr(code int, body []byte) error {
	var v struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(body, &v)
	msg := strings.TrimSpace(v.Error)
	if code == 410 {
		if msg == "" {
			msg = "该环境已从云端删除"
		}
		return errCloudDeleted{msg}
	}
	if !hasCJK(msg) {
		msg = "该环境正在被另一台设备同步（HTTP 423），暂时不能操作；稍后重试" + map[bool]string{true: "", false: "：" + msg}[msg == ""]
	}
	return errCloudLocked{msg}
}

func hasCJK(s string) bool {
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}

func isCloudDeleted(e error) bool { var x errCloudDeleted; return errors.As(e, &x) }
func isCloudLocked(e error) bool  { var x errCloudLocked; return errors.As(e, &x) }

// ---------------------------------------------------------------- store state

func (s *Store) markLocalDeleted(id string, m localDeletedMark) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.CloudDel.LocalDeleted == nil {
		s.CloudDel.LocalDeleted = map[string]localDeletedMark{}
	}
	s.CloudDel.LocalDeleted[id] = m
	s.saveLocked()
}

func (s *Store) clearLocalDeleted(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.CloudDel.LocalDeleted[id]; ok {
		delete(s.CloudDel.LocalDeleted, id)
		s.saveLocked()
	}
}

func (s *Store) localDeleted() map[string]localDeletedMark {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]localDeletedMark{}
	for k, v := range s.CloudDel.LocalDeleted {
		out[k] = v
	}
	return out
}

func (s *Store) queueCloudDelete(id string, q pendingCloudDelete) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.CloudDel.Pending == nil {
		s.CloudDel.Pending = map[string]pendingCloudDelete{}
	}
	s.CloudDel.Pending[id] = q
	s.saveLocked()
}

func (s *Store) clearCloudDelete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.CloudDel.Pending[id]; ok {
		delete(s.CloudDel.Pending, id)
		s.saveLocked()
	}
}

func (s *Store) pendingDeletes() map[string]pendingCloudDelete {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]pendingCloudDelete{}
	for k, v := range s.CloudDel.Pending {
		out[k] = v
	}
	return out
}

// ---------------------------------------------------------------- protocol

// listCloudFull returns live cloud entries and tombstones (servers without
// tombstone support simply return none).
func (s *Store) listCloudFull() (live, tombs []cloudMeta, err error) {
	var out []cloudMeta
	err = s.sync().jsonRequest("GET", "/api/profiles?tombstones=1", nil, &out)
	live, tombs = []cloudMeta{}, []cloudMeta{}
	for _, c := range out {
		if c.Deleted {
			tombs = append(tombs, c)
		} else {
			live = append(live, c)
		}
	}
	return live, tombs, err
}

// deleteCloud asks the server to tombstone a profile. state: deleted |
// already | absent.
func (s *Store) deleteCloud(id string, force bool) (string, error) {
	if !syncIDPattern.MatchString(id) || id == globalID {
		return "", fmt.Errorf("invalid profile ID")
	}
	if e := syncReady(s.GetSettings(), false); e != nil {
		return "", e
	}
	var out struct {
		OK    bool   `json:"ok"`
		State string `json:"state"`
	}
	if e := s.sync().jsonRequest("POST", "/api/profiles/"+id+"/delete", map[string]any{"force": force}, &out); e != nil {
		return "", e
	}
	if !out.OK {
		return "", fmt.Errorf("sync server refused the delete")
	}
	if out.State == "" {
		out.State = "deleted" // server without tombstones: the entry is simply gone
	}
	return out.State, nil
}

// PushProfileRestore uploads a local profile over a cloud tombstone,
// re-creating it in the cloud on purpose.
func (s *Store) PushProfileRestore(id string) error {
	restorePush.Store(id, true)
	defer restorePush.Delete(id)
	return s.PushProfile(id)
}

// ---------------------------------------------------------------- deleting

// DeleteResult reports what happened to one profile.
type DeleteResult struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Local   bool   `json:"local"`           // the local copy was removed
	Cloud   string `json:"cloud,omitempty"` // deleted | queued | locked | failed | skipped
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

func (a *autoSyncer) emitEv(ev SyncEvent) {
	if a.emit != nil {
		a.emit(ev)
	}
}

// DeleteProfiles deletes profiles locally ("local") or locally and in the
// cloud ("cloud"). force overrides another device's sync lease.
func (a *autoSyncer) DeleteProfiles(source string, ids []string, mode string, force bool) []DeleteResult {
	s := a.s
	st := s.GetSettings()
	syncOn := syncReady(st, false) == nil
	tombOK := true
	if mode == "cloud" && syncOn {
		if si, e := s.ServerInfo(false); e == nil && !si.has("tombstones") {
			tombOK = false
		}
	}
	marks, pend := s.localDeleted(), s.pendingDeletes()
	out := []DeleteResult{}
	for _, id := range ids {
		if id == "" || id == globalID {
			continue
		}
		p, ok := s.GetProfile(id)
		r := DeleteResult{ID: id}
		switch {
		case ok:
			r.Name = p.Name
		case marks[id].Name != "":
			r.Name = marks[id].Name
		case pend[id].Name != "":
			r.Name = pend[id].Name
		}
		claimed := a.claim(id) // keeps the post-exit upload away
		runReg.stop(id)
		if mode == "cloud" && syncOn {
			inCloud := !ok || p.CloudVer > 0 || p.Storage == "cloud"
			if !inCloud {
				r.Cloud = "skipped"
			} else {
				_, e := s.deleteCloud(id, force)
				switch {
				case e == nil:
					r.Cloud = "deleted"
					s.clearCloudDelete(id)
					s.auditNamed(source, "sync.delete", id, r.Name, nil, "已删除云端副本，其他设备会同步删除")
					if !tombOK {
						r.Message = "服务器的 ghostsync.php 版本较旧，其他设备不会自动删除各自的副本；请更新服务器端"
					}
				case isCloudLocked(e):
					r.Cloud, r.Error = "locked", e.Error()
					s.auditNamed(source, "sync.delete", id, r.Name, e, "")
				default:
					r.Cloud, r.Error = "queued", e.Error()
					s.queueCloudDelete(id, pendingCloudDelete{Name: r.Name, QueuedAt: time.Now().Unix(), Attempts: 1, LastError: e.Error(), Force: force})
					s.auditNamed(source, "sync.delete", id, r.Name, e, "云端删除已排队，联网后自动重试")
				}
			}
			if r.Cloud == "locked" {
				if claimed {
					a.release(id)
				}
				out = append(out, r)
				continue // nothing deleted: the user decides whether to force
			}
			if ok {
				s.DeleteProfile(id)
				r.Local = true
			}
			s.clearLocalDeleted(id)
			if ok {
				s.auditNamed(source, "profile.delete", id, r.Name, nil, "同时删除云端")
			}
		} else if ok {
			if syncOn && (p.CloudVer > 0 || p.Storage == "cloud") {
				s.markLocalDeleted(id, localDeletedMark{Name: p.Name, CloudVersion: p.CloudVer, DeletedAt: time.Now().Unix()})
			}
			s.DeleteProfile(id)
			r.Local = true
			s.auditNamed(source, "profile.delete", id, r.Name, nil, map[bool]string{true: "仅删除本机，云端副本保留", false: ""}[syncOn && (p.CloudVer > 0 || p.Storage == "cloud")])
		}
		if claimed {
			a.release(id)
		}
		out = append(out, r)
	}
	return out
}

// RetryPendingDeletes sends queued cloud deletes. Returns how many are left.
func (a *autoSyncer) RetryPendingDeletes() int {
	pend := a.s.pendingDeletes()
	if len(pend) == 0 || syncReady(a.s.GetSettings(), false) != nil {
		return len(pend)
	}
	left := 0
	for id, q := range pend {
		_, e := a.s.deleteCloud(id, q.Force)
		if e == nil {
			a.s.clearCloudDelete(id)
			a.s.auditNamed("sync", "sync.delete", id, q.Name, nil, "排队的云端删除已完成")
			a.emitEv(SyncEvent{Phase: "deleted-cloud", ID: id, Name: q.Name, Message: fmt.Sprintf("环境「%s」的云端副本已删除（之前离线排队）", q.Name)})
			continue
		}
		left++
		q.Attempts++
		q.LastError = e.Error()
		a.s.queueCloudDelete(id, q)
		if isCloudDeleted(e) { // should not happen (delete is idempotent), but never retry forever
			a.s.clearCloudDelete(id)
			left--
		}
	}
	return left
}

// WatchDeletes retries queued cloud deletes once a minute.
func (a *autoSyncer) WatchDeletes() {
	for {
		time.Sleep(60 * time.Second)
		if len(a.s.pendingDeletes()) > 0 && a.enabled() {
			a.RetryPendingDeletes()
		}
	}
}

// ---------------------------------------------------------------- remote deletions

var exitDeletes = struct {
	sync.Mutex
	m map[string]cloudMeta
}{m: map[string]cloudMeta{}}

func tombDevice(t cloudMeta) string {
	if t.DeletedByName != "" {
		return t.DeletedByName
	}
	if t.DeletedBy != "" {
		return t.DeletedBy
	}
	return "另一台设备"
}

// applyTombstones removes local copies of profiles deleted from the cloud on
// another device. A profile whose browser is open is removed after it closes.
func (a *autoSyncer) applyTombstones(tombs []cloudMeta) (removed int) {
	for _, t := range tombs {
		a.s.clearLocalDeleted(t.ID)
		a.s.clearCloudDelete(t.ID)
		if a.removeForTomb(t) {
			removed++
		}
	}
	return removed
}

func (a *autoSyncer) removeForTomb(t cloudMeta) bool {
	p, ok := a.s.GetProfile(t.ID)
	if !ok {
		return false
	}
	if t.Version > 0 && p.CloudVer >= t.Version {
		return false // re-created locally after the tombstone (should not happen)
	}
	dev := tombDevice(t)
	if _, running := runReg.get(t.ID); running {
		exitDeletes.Lock()
		_, seen := exitDeletes.m[t.ID]
		exitDeletes.m[t.ID] = t
		exitDeletes.Unlock()
		if !seen {
			a.emitEv(SyncEvent{Phase: "deleted-pending", ID: t.ID, Name: p.Name, Message: fmt.Sprintf("环境「%s」已在 %s 删除，关闭浏览器后会同步删除本机副本", p.Name, dev)})
		}
		return false
	}
	if !a.claim(t.ID) {
		return false // a sync is moving it right now; the next pass handles it
	}
	defer a.release(t.ID)
	a.s.DeleteProfile(t.ID)
	a.s.auditNamed("sync", "sync.delete-local", t.ID, p.Name, nil, "云端已在 "+dev+" 删除，已删除本机副本")
	a.emitEv(SyncEvent{Phase: "deleted", ID: t.ID, Name: p.Name, Message: fmt.Sprintf("环境「%s」已在 %s 删除，已同步删除本机副本", p.Name, dev)})
	return true
}

// takeExitDelete reports (and forgets) a deletion deferred until exit.
func takeExitDelete(id string) (cloudMeta, bool) {
	exitDeletes.Lock()
	defer exitDeletes.Unlock()
	t, ok := exitDeletes.m[id]
	delete(exitDeletes.m, id)
	return t, ok
}

// ---------------------------------------------------------------- cloud tab

// CloudOnlyItem is a cloud profile with no local copy on this device.
type CloudOnlyItem struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Version        int64  `json:"version"`
	UpdatedAt      int64  `json:"updatedAt"`
	DeviceName     string `json:"deviceName"`
	Size           int64  `json:"size"`
	LockedBy       string `json:"lockedBy"`
	LocalDeleted   bool   `json:"localDeleted"`   // deleted here with "only local"
	DeletedVersion int64  `json:"deletedVersion"` // cloud version at that time
	DeletedAt      int64  `json:"deletedAt"`
	CloudNewer     bool   `json:"cloudNewer"` // another device uploaded since
	PendingDelete  bool   `json:"pendingDelete"`
	PendingError   string `json:"pendingError,omitempty"`
}

type CloudOnlyList struct {
	Items []CloudOnlyItem `json:"items"`
	Error string          `json:"error,omitempty"` // server unreachable: items come from local state
}

func (s *Store) CloudOnly() CloudOnlyList {
	out := CloudOnlyList{Items: []CloudOnlyItem{}}
	if e := syncReady(s.GetSettings(), false); e != nil {
		return out
	}
	marks, pend := s.localDeleted(), s.pendingDeletes()
	local := map[string]bool{}
	for _, p := range s.ListProfiles() {
		local[p.ID] = true
	}
	live, _, err := s.listCloudFull()
	seen := map[string]bool{}
	if err != nil {
		out.Error = err.Error()
	}
	for _, c := range live {
		if c.ID == globalID || local[c.ID] {
			continue
		}
		seen[c.ID] = true
		m, isMark := marks[c.ID]
		q, isPend := pend[c.ID]
		out.Items = append(out.Items, CloudOnlyItem{ID: c.ID, Name: c.Name, Version: c.Version, UpdatedAt: c.Updated, DeviceName: c.Device, Size: c.Size, LockedBy: c.Locked,
			LocalDeleted: isMark, DeletedVersion: m.CloudVersion, DeletedAt: m.DeletedAt, CloudNewer: isMark && c.Version > m.CloudVersion,
			PendingDelete: isPend, PendingError: q.LastError})
	}
	if err != nil { // offline: still show what this device knows about
		for id, m := range marks {
			if !seen[id] && !local[id] {
				out.Items = append(out.Items, CloudOnlyItem{ID: id, Name: m.Name, Version: m.CloudVersion, LocalDeleted: true, DeletedVersion: m.CloudVersion, DeletedAt: m.DeletedAt})
				seen[id] = true
			}
		}
	}
	for id, q := range pend {
		if !seen[id] && !local[id] {
			out.Items = append(out.Items, CloudOnlyItem{ID: id, Name: q.Name, PendingDelete: true, PendingError: q.LastError})
		}
	}
	return out
}

// RestoreLocalDeleted downloads a profile deleted only on this device and lets
// auto sync follow it again.
func (s *Store) RestoreLocalDeleted(id string) error {
	if _, ok := s.GetProfile(id); ok {
		s.clearLocalDeleted(id)
		return nil
	}
	e := s.PullProfile(id)
	name := s.localDeleted()[id].Name
	if p, ok := s.GetProfile(id); ok {
		name = p.Name
	}
	s.auditNamed("ui", "sync.pull", id, name, e, "恢复到本机")
	if e != nil {
		return e
	}
	s.clearLocalDeleted(id)
	return nil
}
