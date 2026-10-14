package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Store struct {
	launchMu sync.Mutex
	mu       sync.Mutex
	dir      string
	Profiles map[string]*Profile
	Proxies  map[string]*Proxy
	Settings Settings
	Global   GlobalSyncState
	CloudDel cloudDeleteState // cloud-aware deletion: local-only markers, queued cloud deletes

	onTransfer func(TransferEvent)
	bodyCap    atomic.Int64 // largest upload body the sync server accepted after a 413
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func appDataDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = "."
	}
	d := filepath.Join(base, "GhostBrowser")
	_ = os.MkdirAll(d, 0o755)
	return d
}

func NewStore() *Store {
	s := &Store{dir: appDataDir(), Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}}
	s.load()
	if s.Settings.ProfilesDir == "" {
		s.Settings.ProfilesDir = filepath.Join(s.dir, "profiles")
	}
	if s.Settings.APIPort == 0 {
		s.Settings.APIPort = 25325
		s.Settings.APIEnabled = true
	}
	// The custom Chromium is the only kernel; older builds could pick stock or
	// custom, so any stored mode other than undetectable is migrated here.
	if s.Settings.KernelMode != "undetectable" {
		s.Settings.KernelMode = "undetectable"
	}
	if kit := findUndetectableKit(); kit != "" && s.Settings.UndetectableKit == "" {
		s.Settings.UndetectableKit = kit
	}
	if s.Settings.APIKey == "" {
		s.Settings.APIKey = newAPIKey()
	}
	if s.Settings.DefaultOS == "" {
		s.Settings.DefaultOS = "windows"
	}
	if s.Settings.DeviceID == "" {
		s.Settings.DeviceID = newID()
		h, _ := os.Hostname()
		s.Settings.DeviceName = h
	}
	if s.Settings.SyncPromptAction == "" { // first run with the start-up prompt
		s.Settings.SyncPromptAction = "pull"
		s.Settings.SyncPromptSecs = 3
	}
	if s.Settings.SyncPromptSecs > 300 {
		s.Settings.SyncPromptSecs = 300
	}
	if s.Settings.Theme == "" {
		s.Settings.Theme = "dark"
	}
	if s.Settings.KernelPath == "" {
		s.Settings.KernelPath = defaultKernelPath(s.Settings)
	}
	_ = os.MkdirAll(s.Settings.ProfilesDir, 0o755)
	s.saveLocked()
	return s
}

type diskData struct {
	Profiles []*Profile      `json:"profiles"`
	Proxies  []*Proxy        `json:"proxies"`
	Settings Settings        `json:"settings"`
	Global   GlobalSyncState `json:"globalSync"`
	CloudDel cloudDeleteState `json:"cloudDelete"`
}

func (s *Store) file() string { return filepath.Join(s.dir, "data.json") }

func (s *Store) load() {
	b, err := os.ReadFile(s.file())
	if err != nil {
		return
	}
	var d diskData
	if json.Unmarshal(b, &d) != nil {
		return
	}
	for _, p := range d.Profiles {
		s.Profiles[p.ID] = p
	}
	for _, p := range d.Proxies {
		s.Proxies[p.ID] = p
	}
	s.Settings = d.Settings
	s.Global = d.Global
	s.CloudDel = d.CloudDel
}

func (s *Store) saveLocked() {
	d := diskData{Settings: s.Settings, Global: s.Global, CloudDel: s.CloudDel}
	for _, p := range s.Profiles {
		d.Profiles = append(d.Profiles, p)
	}
	for _, p := range s.Proxies {
		d.Proxies = append(d.Proxies, p)
	}
	b, _ := json.MarshalIndent(d, "", "  ")
	tmp := s.file() + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, s.file())
	}
}

func (s *Store) Save() { s.mu.Lock(); defer s.mu.Unlock(); s.saveLocked() }

func (s *Store) ListProfiles() []Profile {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Profile, 0, len(s.Profiles))
	for _, p := range s.Profiles {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out
}

func (s *Store) GetProfile(id string) (*Profile, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.Profiles[id]
	if !ok {
		return nil, false
	}
	cp := *p
	return &cp, true
}

func (s *Store) PutProfile(p Profile) Profile {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ID == "" {
		p.ID = newID()
		p.CreatedAt = time.Now().Unix()
	}
	if p.Storage == "" {
		p.Storage = "local"
	}
	s.Profiles[p.ID] = &p
	s.saveLocked()
	return p
}

func (s *Store) DeleteProfile(id string) {
	s.mu.Lock()
	delete(s.Profiles, id)
	s.saveLocked()
	dir := filepath.Join(s.Settings.ProfilesDir, id)
	s.mu.Unlock()
	_ = os.RemoveAll(dir)
}

func (s *Store) ListProxies() []Proxy {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Proxy, 0, len(s.Proxies))
	for _, p := range s.Proxies {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Store) PutProxy(p Proxy) Proxy {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ID == "" {
		p.ID = newID()
	}
	if p.Name == "" {
		p.Name = p.Host
	}
	if p.Status == "" {
		p.Status = "unknown"
	}
	s.Proxies[p.ID] = &p
	s.saveLocked()
	return p
}

func (s *Store) DeleteProxy(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Proxies, id)
	for _, p := range s.Profiles {
		if p.ProxyID == id {
			p.ProxyID = ""
		}
	}
	s.saveLocked()
}

func (s *Store) GetSettings() Settings { s.mu.Lock(); defer s.mu.Unlock(); return s.Settings }

func (s *Store) PutSettings(st Settings) Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	st.DeviceID = s.Settings.DeviceID
	if st.APIKey == "" {
		st.APIKey = s.Settings.APIKey
	}
	s.Settings = st
	_ = os.MkdirAll(st.ProfilesDir, 0o755)
	s.saveLocked()
	return st
}

func (s *Store) ProfileDir(id string) string {
	return filepath.Join(s.GetSettings().ProfilesDir, id)
}
