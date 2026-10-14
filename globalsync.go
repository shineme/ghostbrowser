package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"time"
)

// Global data (settings, proxy list, fingerprint library) is synced as one
// extra encrypted record on the server with the reserved id "_global".

const globalID = "_global"
const globalName = "软件全局数据（设置 / 代理 / 指纹库）"

// GlobalSyncState is local bookkeeping only, kept outside Settings so a save
// from the settings page can never roll it back.
type GlobalSyncState struct {
	Version int64  `json:"version"` // cloud version the local copy is based on
	Hash    string `json:"hash"`    // content hash at the last successful sync
	Keep    int64  `json:"keep"`    // user kept local over this cloud version
}

type globalPayload struct {
	Format   int           `json:"format"`
	Device   string        `json:"device"`
	SavedAt  int64         `json:"savedAt"`
	Settings Settings      `json:"settings"`
	Proxies  []Proxy       `json:"proxies"`
	Library  fpLibraryData `json:"library"`
}

func (l *fpLibrary) snapshot() fpLibraryData {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, _ := json.Marshal(l.d)
	var out fpLibraryData
	_ = json.Unmarshal(b, &out)
	if out.Values == nil {
		out.Values = map[string][]string{}
	}
	return out
}

func (l *fpLibrary) replace(d fpLibraryData) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if d.Values == nil {
		d.Values = map[string][]string{}
	}
	l.d = d
	return l.saveLocked()
}

func (s *Store) globalState() GlobalSyncState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Global
}

func (s *Store) setGlobalState(g GlobalSyncState) {
	s.mu.Lock()
	s.Global = g
	s.saveLocked()
	s.mu.Unlock()
}

func (s *Store) globalPayload() globalPayload {
	st := s.GetSettings()
	px := s.ListProxies()
	sort.Slice(px, func(i, j int) bool { return px[i].ID < px[j].ID })
	return globalPayload{Format: 1, Device: st.DeviceName, SavedAt: time.Now().Unix(), Settings: st, Proxies: px, Library: userLib.snapshot()}
}

// globalHash covers what the user edits. Per-device fields and proxy check
// results (latency, exit IP...) are left out so routine checks on two
// computers do not keep re-uploading or conflicting.
func globalHash(g globalPayload) string {
	st := g.Settings
	st.DeviceID, st.DeviceName, st.ProfilesDir = "", "", ""
	type pxKey struct {
		ID, Name, Type, Host string
		Port                 int
		User, Pass, Change   string
		Tag                  string
	}
	keys := make([]pxKey, 0, len(g.Proxies))
	for _, p := range g.Proxies {
		keys = append(keys, pxKey{p.ID, p.Name, p.Type, p.Host, p.Port, p.User, p.Pass, p.ChangeIP, p.Tag})
	}
	b, _ := json.Marshal(struct {
		S Settings
		P []pxKey
		L fpLibraryData
	}{st, keys, g.Library})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *Store) globalDirty() bool {
	return globalHash(s.globalPayload()) != s.globalState().Hash
}

// PushGlobal uploads settings, proxies and the fingerprint library.
func (s *Store) PushGlobal() (err error) {
	t0 := time.Now()
	defer func() { s.logSync(globalID, "push-global", 0, t0, err) }()
	st := s.GetSettings()
	if e := syncReady(st, true); e != nil {
		return e
	}
	g := s.globalState()
	if g.Keep > g.Version {
		g.Version = g.Keep
	}
	payload := s.globalPayload()
	raw, _ := json.Marshal(payload)
	secret, e := s.dataSecret()
	if e != nil {
		return e
	}
	enc, e := encrypt(raw, secret)
	if e != nil {
		return e
	}
	sum := sha256.Sum256(enc)
	c := s.sync()
	path := "/api/profiles/" + globalID
	var lease struct {
		Token   string `json:"lockToken"`
		Version int64  `json:"version"`
	}
	if e = c.jsonRequest("POST", path+"/lock", map[string]string{"device": st.DeviceID}, &lease); e != nil {
		return fmt.Errorf("lock: %w", e)
	}
	if lease.Token == "" {
		return fmt.Errorf("sync server did not issue a lock token")
	}
	c.lockToken = lease.Token
	committed := false
	defer func() {
		if !committed {
			_ = c.jsonRequest("POST", path+"/unlock", map[string]any{}, nil)
		}
	}()
	if lease.Version != g.Version {
		return fmt.Errorf("conflict: cloud global data is v%d, local is v%d; pull first", lease.Version, g.Version)
	}
	resp, e := c.req("PUT", path+"/blob?version="+strconv.FormatInt(g.Version, 10), bytes.NewReader(enc), "application/octet-stream")
	if e != nil {
		return e
	}
	e = syncStatus(resp)
	resp.Body.Close()
	if e != nil {
		return e
	}
	cm := cloudMeta{ID: globalID, Name: globalName, Version: g.Version + 1, Sha256: hex.EncodeToString(sum[:]),
		Meta: json.RawMessage(`{"encrypted":true,"format":"GBE1","profileMetadata":"in-blob"}`)}
	var result struct {
		OK      bool  `json:"ok"`
		Version int64 `json:"version"`
	}
	if e = c.jsonRequest("POST", path+"/commit", cm, &result); e != nil {
		return fmt.Errorf("commit: %w", e)
	}
	if !result.OK || result.Version != cm.Version {
		return fmt.Errorf("server returned an invalid commit acknowledgement")
	}
	committed = true
	s.setGlobalState(GlobalSyncState{Version: cm.Version, Hash: globalHash(payload)})
	return nil
}

// PullGlobal replaces local settings, proxies and library with the cloud copy.
// Kept from this computer: its device ID and name, the sync connection that is
// working right now (address, token, password), its profiles folder, and its
// kernel path whenever the cloud one does not exist on this computer.
func (s *Store) PullGlobal() (err error) {
	t0 := time.Now()
	defer func() { s.logSync(globalID, "pull-global", 0, t0, err) }()
	st := s.GetSettings()
	if e := syncReady(st, true); e != nil {
		return e
	}
	resp, e := s.sync().req("GET", "/api/profiles/"+globalID+"/blob", nil, "")
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if e = syncStatus(resp); e != nil {
		return e
	}
	version, e := strconv.ParseInt(resp.Header.Get("X-Profile-Version"), 10, 64)
	if e != nil || version < 1 {
		return fmt.Errorf("sync server returned no valid version")
	}
	enc, e := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if e != nil {
		return e
	}
	secret, e := s.dataSecret()
	if e != nil {
		return e
	}
	raw, e := decrypt(enc, secret)
	if e != nil {
		return fmt.Errorf("decrypt failed; check the sync password: %w", e)
	}
	var g globalPayload
	if e = json.Unmarshal(raw, &g); e != nil || g.Format != 1 {
		return fmt.Errorf("unrecognised global data format")
	}
	ns := g.Settings
	ns.DeviceID, ns.DeviceName = st.DeviceID, st.DeviceName
	ns.SyncServer, ns.SyncToken, ns.SyncPassword = st.SyncServer, st.SyncToken, st.SyncPassword
	exists := func(p string) bool { _, e := os.Stat(p); return p != "" && e == nil }
	if !exists(ns.KernelPath) {
		ns.KernelPath = st.KernelPath
	}
	if !exists(ns.ChromiumPath) {
		ns.ChromiumPath = st.ChromiumPath
	}
	if !exists(ns.UndetectableKit) {
		ns.UndetectableKit = st.UndetectableKit
	}
	ns.ProfilesDir = st.ProfilesDir // where this computer keeps its browser data
	ns.KernelMode = "undetectable"
	s.mu.Lock()
	s.Settings = ns
	s.Proxies = map[string]*Proxy{}
	for i := range g.Proxies {
		p := g.Proxies[i]
		s.Proxies[p.ID] = &p
	}
	s.mu.Unlock()
	if e = userLib.replace(g.Library); e != nil {
		return e
	}
	s.setGlobalState(GlobalSyncState{Version: version, Hash: globalHash(s.globalPayload())})
	return nil
}
