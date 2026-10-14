package main

// Recycle bin (最近删除) for profiles.
//
// A profile deleted on this device (只删本地, 同时删云端, or removed because
// another device deleted it from the cloud) is not destroyed at once: its
// record moves from Store.Profiles to Store.Trash and its data folder moves to
// <ProfilesDir>/.trash/<id>. Everything that iterates profiles (list, launch,
// auto sync push, syncer, bulk ops, local API) reads Store.Profiles only, so a
// trashed profile is invisible to it without further checks. Re-downloading
// from the cloud is blocked by the existing local-delete marker
// (Store.CloudDel.LocalDeleted), which the delete path already sets.
//
// Restore moves record and folder back. When the cloud copy was deleted
// (tombstone), the restored profile gets a new ID and starts as a local-only
// profile, so the tombstone cannot remove it again and auto sync uploads it as
// a new profile. Purge deletes the folder and the record (including any inline
// proxy credentials stored on it). Entries older than Settings.TrashDays
// (default 30) are purged on start-up and whenever the bin is listed.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const trashDirName = ".trash"
const defaultTrashDays = 30

// TrashItem is one entry of the recycle bin as shown in the UI.
type TrashItem struct {
	Profile
	Size         int64  `json:"size"`         // bytes of the retained data folder
	ExpiresAt    int64  `json:"expiresAt"`    // unix seconds when it is purged automatically
	HasData      bool   `json:"hasData"`      // the data folder was retained
	CloudNote    string `json:"cloudNote"`    // short status of the cloud copy
	RestoreAsNew bool   `json:"restoreAsNew"` // restore creates a new local profile (cloud copy deleted)
}

func trashDays(st Settings) int {
	if st.TrashDays <= 0 {
		return defaultTrashDays
	}
	return st.TrashDays
}

func (s *Store) trashRoot() string { return filepath.Join(s.GetSettings().ProfilesDir, trashDirName) }

// moveDir renames src to dst, retrying briefly (a just-closed browser can keep
// files locked for a moment on Windows).
func moveDir(src, dst string) error {
	if _, e := os.Stat(src); e != nil {
		return e
	}
	_ = os.MkdirAll(filepath.Dir(dst), 0o755)
	_ = os.RemoveAll(dst)
	var err error
	for i := 0; i < 10; i++ {
		if err = os.Rename(src, dst); err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return err
}

func dirSize(dir string) int64 {
	var n int64
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if fi, e := d.Info(); e == nil {
				n += fi.Size()
			}
		}
		return nil
	})
	return n
}

// TrashProfile moves a profile to the recycle bin. cloudDeleted records that
// the cloud copy was (or will be) deleted. Returns false when there is no such
// profile.
func (s *Store) TrashProfile(id string, cloudDeleted bool) bool {
	s.mu.Lock()
	p, ok := s.Profiles[id]
	if !ok {
		s.mu.Unlock()
		return false
	}
	if p.ShareID != "" {
		// A copy received through sharing is someone else's data and cannot be
		// restored once the share ends: delete it outright, never keep it here.
		s.mu.Unlock()
		s.DeleteProfile(id)
		return true
	}
	root := filepath.Join(s.Settings.ProfilesDir, trashDirName)
	src := filepath.Join(s.Settings.ProfilesDir, id)
	s.mu.Unlock()

	dst := filepath.Join(root, id)
	dataDir := ""
	if _, e := os.Stat(src); e == nil {
		if moveDir(src, dst) == nil {
			dataDir = dst
		} else {
			dataDir = src // could not move it: keep it where it is (restore still works)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *p
	cp.Trashed = true
	cp.TrashedAt = time.Now().Unix()
	cp.TrashDir = dataDir
	cp.TrashCloudDeleted = cloudDeleted
	if s.Trash == nil {
		s.Trash = map[string]*Profile{}
	}
	delete(s.Profiles, id)
	s.Trash[id] = &cp
	s.saveLocked()
	return true
}

func (s *Store) trashed(id string) (*Profile, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.Trash[id]
	if !ok {
		return nil, false
	}
	cp := *p
	return &cp, true
}

// markTrashCloudDeleted notes that a trashed profile's cloud copy is gone.
func (s *Store) markTrashCloudDeleted(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.Trash[id]; ok && !p.TrashCloudDeleted {
		p.TrashCloudDeleted = true
		s.saveLocked()
	}
}

// ListTrash returns the recycle bin, newest first. Expired entries are purged first.
func (s *Store) ListTrash() []TrashItem {
	s.PurgeExpiredTrash()
	st := s.GetSettings()
	days := int64(trashDays(st)) * 86400
	marks := s.localDeleted()
	pend := s.pendingDeletes()
	s.mu.Lock()
	ps := make([]Profile, 0, len(s.Trash))
	for _, p := range s.Trash {
		ps = append(ps, *p)
	}
	s.mu.Unlock()
	out := make([]TrashItem, 0, len(ps))
	for _, p := range ps {
		it := TrashItem{Profile: p, ExpiresAt: p.TrashedAt + days}
		if p.TrashDir != "" {
			if _, e := os.Stat(p.TrashDir); e == nil {
				it.HasData = true
				it.Size = dirSize(p.TrashDir)
			}
		}
		_, isPend := pend[p.ID]
		_, live := s.GetProfile(p.ID)
		switch {
		case live:
			it.CloudNote = "本机已从云端重新下载此环境"
		case isPend:
			it.CloudNote = "云端删除排队中"
			it.RestoreAsNew = true
		case p.TrashCloudDeleted:
			it.CloudNote = "云端已删除"
			it.RestoreAsNew = true
		case marks[p.ID].DeletedAt != 0:
			it.CloudNote = "云端副本保留"
		case p.CloudVer > 0:
			it.CloudNote = "云端副本保留"
		default:
			it.CloudNote = "仅本机"
		}
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TrashedAt > out[j].TrashedAt })
	return out
}

// RestoreResult reports one restore.
type RestoreResult struct {
	ID    string `json:"id"`              // ID of the restored profile (new when restored as new)
	OldID string `json:"oldId,omitempty"` // set when the profile got a new ID
	Name  string `json:"name"`
	Error string `json:"error,omitempty"`
}

// RestoreFromTrash moves profiles back to the normal list.
func (s *Store) RestoreFromTrash(source string, ids []string) []RestoreResult {
	out := []RestoreResult{}
	pend := s.pendingDeletes()
	for _, id := range ids {
		p, ok := s.trashed(id)
		if !ok {
			continue
		}
		r := RestoreResult{ID: id, Name: p.Name}
		_, isPend := pend[id]
		asNew := p.TrashCloudDeleted || isPend
		newID := id
		if asNew {
			newID = newID2(s)
		} else if _, live := s.GetProfile(id); live {
			r.Error = "本机已有同 ID 的环境（可能已从云端重新下载），请先删除它或彻底删除此项"
			out = append(out, r)
			continue
		}
		dst := s.ProfileDir(newID)
		if p.TrashDir != "" {
			if _, e := os.Stat(p.TrashDir); e == nil && p.TrashDir != dst {
				if e := moveDir(p.TrashDir, dst); e != nil {
					r.Error = "数据文件夹移回失败（可能被占用）：" + e.Error()
					s.auditNamed(source, "profile.restore", id, p.Name, e, "")
					out = append(out, r)
					continue
				}
			}
		}
		np := *p
		np.Trashed, np.TrashedAt, np.TrashDir, np.TrashCloudDeleted = false, 0, "", false
		detail := "已从最近删除恢复"
		if asNew {
			np.ID = newID
			np.CloudVer, np.SyncedAt, np.KeepLocal, np.ChunkKey = 0, 0, 0, ""
			np.Storage = "local"
			r.ID, r.OldID = newID, id
			detail = "云端副本已删除，已作为新的本地环境恢复"
		}
		s.mu.Lock()
		delete(s.Trash, id)
		s.Profiles[np.ID] = &np
		s.saveLocked()
		s.mu.Unlock()
		if !asNew {
			s.clearLocalDeleted(id) // auto sync follows it again
		}
		s.auditNamed(source, "profile.restore", np.ID, np.Name, nil, detail)
		out = append(out, r)
	}
	return out
}

// newID2 returns an ID used by neither a profile nor a trash entry.
func newID2(s *Store) string {
	for {
		id := newID()
		s.mu.Lock()
		_, a := s.Profiles[id]
		_, b := s.Trash[id]
		s.mu.Unlock()
		if !a && !b {
			return id
		}
	}
}

// PurgeTrash permanently deletes trashed profiles: data folder and record
// (inline proxy credentials live on the record and go with it). Shared proxies
// in the proxy list are not touched. Returns how many were purged.
func (s *Store) PurgeTrash(source string, ids []string, detail string) int {
	n := 0
	for _, id := range ids {
		p, ok := s.trashed(id)
		if !ok {
			continue
		}
		var err error
		if p.TrashDir != "" && p.TrashDir != s.ProfileDir(id) {
			err = os.RemoveAll(p.TrashDir)
		} else if p.TrashDir != "" {
			if _, live := s.GetProfile(id); !live { // never remove a live profile's folder
				err = os.RemoveAll(p.TrashDir)
			}
		}
		s.mu.Lock()
		delete(s.Trash, id)
		s.saveLocked()
		s.mu.Unlock()
		s.auditNamed(source, "profile.purge", id, p.Name, err, detail)
		n++
	}
	return n
}

// EmptyTrash purges everything in the recycle bin.
func (s *Store) EmptyTrash(source string) int {
	s.mu.Lock()
	ids := make([]string, 0, len(s.Trash))
	for id := range s.Trash {
		ids = append(ids, id)
	}
	s.mu.Unlock()
	return s.PurgeTrash(source, ids, "清空回收站")
}

// PurgeExpiredTrash removes entries older than Settings.TrashDays.
func (s *Store) PurgeExpiredTrash() int {
	cut := time.Now().Unix() - int64(trashDays(s.GetSettings()))*86400
	s.mu.Lock()
	ids := []string{}
	for id, p := range s.Trash {
		if p.TrashedAt < cut {
			ids = append(ids, id)
		}
	}
	s.mu.Unlock()
	if len(ids) == 0 {
		return 0
	}
	return s.PurgeTrash("ui", ids, fmt.Sprintf("超过 %d 天自动彻底删除", trashDays(s.GetSettings())))
}
