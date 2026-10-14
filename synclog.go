package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Sync log: the last 300 sync operations, kept in %APPDATA%/GhostBrowser/sync-log.json.

type SyncLogEntry struct {
	Time   int64  `json:"time"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Action string `json:"action"` // push | pull | restore | push-global | pull-global
	OK     bool   `json:"ok"`
	Error  string `json:"error"`
	Bytes  int64  `json:"bytes"` // transferred
	Ms     int64  `json:"ms"`
	Ver    int64  `json:"version"`
}

const syncLogMax = 300

var syncLogMu sync.Mutex

func (s *Store) syncLogFile() string { return filepath.Join(s.dir, "sync-log.json") }

func (s *Store) readSyncLog() []SyncLogEntry {
	var out []SyncLogEntry
	if b, e := os.ReadFile(s.syncLogFile()); e == nil {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

func (s *Store) logSync(id, action string, bytes int64, t0 time.Time, err error) {
	if s == nil || s.dir == "" {
		return
	}
	e := SyncLogEntry{Time: time.Now().Unix(), ID: id, Action: action, OK: err == nil, Bytes: bytes, Ms: time.Since(t0).Milliseconds()}
	if err != nil {
		e.Error = err.Error()
	}
	if id == globalID {
		e.Name = globalName
		e.Ver = s.globalState().Version
	} else if p, ok := s.GetProfile(id); ok {
		e.Name = p.Name
		e.Ver = p.CloudVer
	}
	s.auditNamed("sync", "sync."+action, id, e.Name, err, fmt.Sprintf("%d B, %d ms", bytes, e.Ms))
	syncLogMu.Lock()
	defer syncLogMu.Unlock()
	list := append([]SyncLogEntry{e}, s.readSyncLog()...)
	if len(list) > syncLogMax {
		list = list[:syncLogMax]
	}
	b, _ := json.Marshal(list)
	tmp := s.syncLogFile() + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, s.syncLogFile())
	}
}

// ---------------------------------------------------------------- versions

type CloudVersion struct {
	Version    int64  `json:"version"`
	Size       int64  `json:"size"`
	UpdatedAt  int64  `json:"updatedAt"`
	DeviceName string `json:"deviceName"`
	Current    bool   `json:"current"`
}

func (s *Store) CloudHistory(id string) ([]CloudVersion, error) {
	if !syncIDPattern.MatchString(id) {
		return nil, fmt.Errorf("invalid profile ID")
	}
	if e := syncReady(s.GetSettings(), false); e != nil {
		return nil, e
	}
	if si, err := s.ServerInfo(false); err == nil && !si.has("history") {
		return nil, fmt.Errorf("服务器不支持版本历史，请更新 ghostsync.php")
	}
	var out []CloudVersion
	e := s.sync().jsonRequest("GET", "/api/profiles/"+id+"/history", nil, &out)
	if out == nil {
		out = []CloudVersion{}
	}
	return out, e
}

// RestoreVersion brings back an older cloud version locally and uploads it as
// the newest version, so every device ends up on the restored copy.
func (s *Store) RestoreVersion(id string, version int64) error {
	t0 := time.Now()
	hist, e := s.CloudHistory(id)
	if e != nil {
		return e
	}
	var cur int64
	found := false
	for _, h := range hist {
		if h.Current {
			cur = h.Version
		}
		if h.Version == version {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("版本 v%d 已不在服务器上", version)
	}
	n, e := s.pullProfile(id, version)
	if e == nil && version != cur {
		p, ok := s.GetProfile(id)
		if ok {
			p.CloudVer = cur
			p.KeepLocal = 0
			s.PutProfile(*p)
		}
		var m int64
		m, e = s.pushProfile(id)
		n += m
	}
	s.logSync(id, "restore", n, t0, e)
	return e
}
