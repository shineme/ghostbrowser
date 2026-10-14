package main

// Partitioned session store (ghost-session.json schema 2).
//
// Schema 1 kept the browser's cookies only. Many web apps keep their login in
// LocalStorage or IndexedDB, and Chrome's partitioned (CHIPS) cookies were
// dropped on restore, so a device switch still logged those sites out. Schema 2
// adds, per origin, the LocalStorage items and an IndexedDB dump, and keeps the
// full CDP cookie objects including partitionKey.
//
// Capture runs through CDP in an isolated world of every open http(s) tab
// (pristine globals, invisible to the page). Origins captured earlier but not
// open now are carried over from the previous snapshot, so coverage grows as
// the user visits sites. Restore opens a hidden tab per origin, answers its
// navigation with an empty document through Fetch interception (no network, no
// site scripts), then writes LocalStorage and IndexedDB from there.
//
// Compatibility: schema 2 stores cookies under "cookies2", not "cookies", so a
// v0.7 client (which only knows "cookies") restores nothing rather than half a
// session. This client refuses to read or overwrite a schema newer than its own.
// The value codec below is our own (tagged JSON for Date, RegExp, binary,
// Blob/File, Map, Set, BigInt and non-finite numbers).

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	sessionSchema        = 2
	sessionStoreCapMBDef = 32
	sessionOriginMaxAge  = 90 * 24 * 3600 // drop origins not seen for 90 days
	sessionIDBEvery      = 4              // ticks (×15 s) between IndexedDB captures while running
)

var errSessionSchema = errors.New("会话快照来自更新版本的 GhostBrowser，请升级后再同步")

type sessionSnapV2 struct {
	Schema  int               `json:"schema"`
	Device  string            `json:"device"`
	SavedAt int64             `json:"savedAt"`
	Cookies []json.RawMessage `json:"cookies2"` // full CDP cookie objects (partitionKey kept)
	Origins []originStore     `json:"origins,omitempty"`
	Dropped []string          `json:"dropped,omitempty"` // origins left out by the size cap
}

type originStore struct {
	Origin     string            `json:"origin"`
	SavedAt    int64             `json:"savedAt"`
	Local      map[string]string `json:"local,omitempty"`
	IDB        []idbDatabase     `json:"idb,omitempty"`
	IDBAt      int64             `json:"idbAt,omitempty"`
	IDBPartial bool              `json:"idbPartial,omitempty"` // some records skipped (unsupported type / budget)
	IDBDropped bool              `json:"idbDropped,omitempty"` // the size cap left IndexedDB out
}

type idbDatabase struct {
	Name    string     `json:"name"`
	Version int64      `json:"version"`
	Stores  []idbStore `json:"stores"`
}

type idbStore struct {
	Name          string          `json:"name"`
	KeyPath       json.RawMessage `json:"keyPath"`
	AutoIncrement bool            `json:"autoIncrement"`
	Indexes       []idbIndex      `json:"indexes,omitempty"`
	Records       []idbRecord     `json:"records"`
	Truncated     bool            `json:"truncated,omitempty"`
	Skipped       int             `json:"skipped,omitempty"`
}

type idbIndex struct {
	Name       string          `json:"name"`
	KeyPath    json.RawMessage `json:"keyPath"`
	Unique     bool            `json:"unique"`
	MultiEntry bool            `json:"multiEntry"`
}

type idbRecord struct {
	K json.RawMessage `json:"k"`
	V json.RawMessage `json:"v"`
}

// ---- settings ----

var sessionCfgStore atomic.Pointer[Store]

func sessionStoreOn(st Settings) bool { return st.SyncSessionStorage == nil || *st.SyncSessionStorage }

func sessionCapBytes(st Settings) int64 {
	mb := st.SessionStoreCapMB
	if mb <= 0 {
		mb = sessionStoreCapMBDef
	}
	return int64(min(mb, 256)) << 20
}

func sessionCfg() (on bool, capBytes int64) {
	if s := sessionCfgStore.Load(); s != nil {
		st := s.GetSettings()
		return sessionStoreOn(st), sessionCapBytes(st)
	}
	return true, sessionStoreCapMBDef << 20
}

// ---- file ----

var sessionLocks sync.Map // dir -> *sync.Mutex

func sessionLock(dir string) func() {
	m, _ := sessionLocks.LoadOrStore(dir, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// decodeSession reads either schema. Schema 1 is lifted into the v2 struct.
func decodeSession(b []byte) (*sessionSnapV2, error) {
	var head struct {
		Schema  int               `json:"schema"`
		Device  string            `json:"device"`
		SavedAt int64             `json:"savedAt"`
		Cookies []json.RawMessage `json:"cookies"`
	}
	if err := json.Unmarshal(b, &head); err != nil {
		return nil, err
	}
	if head.Schema > sessionSchema {
		return nil, errSessionSchema
	}
	if head.Schema < 2 {
		return &sessionSnapV2{Schema: 1, Device: head.Device, SavedAt: head.SavedAt, Cookies: head.Cookies}, nil
	}
	var s sessionSnapV2
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func readSessionFile(dir string) (*sessionSnapV2, error) {
	b, err := os.ReadFile(filepath.Join(dir, sessionFile))
	if err != nil {
		return nil, err
	}
	return decodeSession(b)
}

func writeSessionFile(dir string, s *sessionSnapV2) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, sessionFile+".tmp")
	if err = os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, sessionFile))
}

// ---- size cap ----

func jsonLen(v any) int64 { b, _ := json.Marshal(v); return int64(len(b)) }

// applySessionCap keeps all cookies, then LocalStorage of the most recently
// seen origins, then their IndexedDB, until capBytes is reached. What does not
// fit is marked (IDBDropped) or listed in Dropped.
func applySessionCap(s *sessionSnapV2, capBytes int64) {
	sort.SliceStable(s.Origins, func(i, j int) bool { return s.Origins[i].SavedAt > s.Origins[j].SavedAt })
	used := jsonLen(s.Cookies) + 256
	keep := s.Origins[:0]
	var dropped []string
	idbs := make([][]idbDatabase, len(s.Origins))
	for i := range s.Origins {
		idbs[i] = s.Origins[i].IDB
		s.Origins[i].IDB = nil
	}
	var kept []int
	for i, o := range s.Origins {
		n := jsonLen(o)
		if used+n > capBytes {
			dropped = append(dropped, o.Origin)
			continue
		}
		used += n
		kept = append(kept, i)
	}
	for _, i := range kept {
		o := s.Origins[i]
		if len(idbs[i]) > 0 {
			n := jsonLen(idbs[i])
			if used+n <= capBytes {
				o.IDB, o.IDBDropped = idbs[i], false
				used += n
			} else {
				o.IDBDropped = true
			}
		}
		keep = append(keep, o)
	}
	s.Origins, s.Dropped = keep, dropped
}

// ---- cookies ----

// cookieParamV2 is cookieParam plus partitioned (CHIPS) cookies.
func cookieParamV2(raw json.RawMessage) map[string]any {
	var c map[string]any
	if json.Unmarshal(raw, &c) != nil || c["name"] == nil || c["domain"] == nil {
		return nil
	}
	out := map[string]any{}
	for _, k := range []string{"name", "value", "domain", "path", "secure", "httpOnly", "sameSite", "priority", "sourceScheme", "sourcePort", "partitionKey"} {
		if v, ok := c[k]; ok && v != nil {
			out[k] = v
		}
	}
	if _, ok := out["partitionKey"]; ok {
		out["secure"] = true // Chrome requires Secure for partitioned cookies
	}
	if s, _ := c["session"].(bool); !s {
		if e, ok := c["expires"].(float64); ok && e > float64(time.Now().Unix()) {
			out["expires"] = e
		} else if ok {
			return nil
		}
	}
	return out
}

// altPartitionKey converts between the two CDP shapes: a top-level-site string
// (Chromium ≤ 118) and {topLevelSite, hasCrossSiteAncestor} (newer).
func altPartitionKey(v any) any {
	switch k := v.(type) {
	case string:
		return map[string]any{"topLevelSite": k, "hasCrossSiteAncestor": false}
	case map[string]any:
		if s, ok := k["topLevelSite"].(string); ok {
			return s
		}
	}
	return nil
}

func setCookiesV2(c *cdpConn, params []map[string]any) (ok int) {
	set := func(list []map[string]any) bool {
		_, err := c.call("", "Storage.setCookies", map[string]any{"cookies": list})
		return err == nil
	}
	for i := 0; i < len(params); i += 200 {
		j := min(i+200, len(params))
		if set(params[i:j]) {
			ok += j - i
			continue
		}
		for _, one := range params[i:j] {
			if set([]map[string]any{one}) {
				ok++
				continue
			}
			if pk, has := one["partitionKey"]; has {
				if alt := altPartitionKey(pk); alt != nil {
					cp := map[string]any{}
					for k, v := range one {
						cp[k] = v
					}
					cp["partitionKey"] = alt
					if set([]map[string]any{cp}) {
						ok++
					}
				}
			}
		}
	}
	return ok
}

// ---- origins ----

// storageOrigin returns the http(s) origin of a tab URL, "" for anything else
// (extensions, chrome://, devtools, about:, file:).
func storageOrigin(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + strings.ToLower(u.Host)
}

func evalValue(c *cdpConn, sid string, ctx int, expr string, out any) error {
	p := map[string]any{"expression": expr, "returnByValue": true, "awaitPromise": true}
	if ctx > 0 {
		p["contextId"] = ctx
	}
	raw, err := c.call(sid, "Runtime.evaluate", p)
	if err != nil {
		return err
	}
	var r struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		Exception *struct {
			Text      string `json:"text"`
			Exception struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	if err = json.Unmarshal(raw, &r); err != nil {
		return err
	}
	if r.Exception != nil {
		return fmt.Errorf("%s %s", r.Exception.Text, r.Exception.Exception.Description)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(r.Result.Value, out)
}

func isolatedWorld(c *cdpConn, sid string) (int, error) {
	raw, err := c.call(sid, "Page.getFrameTree", nil)
	if err != nil {
		return 0, err
	}
	var ft struct {
		FrameTree struct {
			Frame struct {
				ID string `json:"id"`
			} `json:"frame"`
		} `json:"frameTree"`
	}
	_ = json.Unmarshal(raw, &ft)
	raw, err = c.call(sid, "Page.createIsolatedWorld", map[string]any{"frameId": ft.FrameTree.Frame.ID, "worldName": "__gb_ss"})
	if err != nil {
		return 0, err
	}
	var w struct {
		ID int `json:"executionContextId"`
	}
	_ = json.Unmarshal(raw, &w)
	return w.ID, nil
}

type originDump struct {
	Local   map[string]string `json:"local"`
	LSErr   string            `json:"lsErr"`
	IDB     []idbDatabase     `json:"idb"`
	Partial bool              `json:"partial"`
	Size    int64             `json:"size"`
}

// captureOrigins dumps LocalStorage (and IndexedDB when withIDB) of every open
// http(s) origin. Errors on one origin are skipped.
func captureOrigins(c *cdpConn, withIDB bool, budget int64) map[string]originDump {
	out := map[string]originDump{}
	tabs, err := pageTabs(c)
	if err != nil {
		return out
	}
	deadline := time.Now().Add(40 * time.Second)
	for _, t := range tabs {
		o := storageOrigin(t.URL)
		if o == "" || time.Now().After(deadline) {
			continue
		}
		if _, done := out[o]; done {
			continue
		}
		sid, err := attachTarget(c, t.TargetID)
		if err != nil {
			continue
		}
		var d originDump
		ctx, err := isolatedWorld(c, sid)
		if err == nil {
			err = evalValue(c, sid, ctx, fmt.Sprintf("(%s)(%d,%v)", jsDumpOrigin, budget, withIDB), &d)
		}
		_, _ = c.call("", "Target.detachFromTarget", map[string]any{"sessionId": sid})
		if err != nil || d.Local == nil && d.LSErr != "" {
			continue
		}
		budget -= d.Size
		out[o] = d
	}
	return out
}

// mergeOrigins folds a fresh capture into the previous snapshot's origins.
func mergeOrigins(prev []originStore, fresh map[string]originDump, withIDB bool, now int64) []originStore {
	by := map[string]originStore{}
	for _, o := range prev {
		if now-o.SavedAt < sessionOriginMaxAge {
			by[o.Origin] = o
		}
	}
	for origin, d := range fresh {
		o := by[origin]
		o.Origin, o.SavedAt, o.Local = origin, now, d.Local
		if withIDB {
			o.IDB, o.IDBAt, o.IDBPartial, o.IDBDropped = d.IDB, now, d.Partial, false
		}
		if len(o.Local) == 0 && len(o.IDB) == 0 {
			delete(by, origin)
			continue
		}
		by[origin] = o
	}
	list := make([]originStore, 0, len(by))
	for _, o := range by {
		list = append(list, o)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Origin < list[j].Origin })
	return list
}

// snapshotSessionV2 writes a schema-2 snapshot. withIDB=false refreshes
// cookies and LocalStorage and keeps the previous IndexedDB dump.
func snapshotSessionV2(dir, device, ws string, withIDB bool, capBytes int64) error {
	raw, err := exportCookies(ws)
	if err != nil {
		return err
	}
	var cookies []json.RawMessage
	if json.Unmarshal([]byte(raw), &cookies) != nil {
		return nil
	}
	c, err := dialCDP(ws, nil)
	if err != nil {
		return err
	}
	fresh := captureOrigins(c, withIDB, capBytes)
	c.c.Close()

	unlock := sessionLock(dir)
	defer unlock()
	var prev []originStore
	if old, err := readSessionFile(dir); err == errSessionSchema {
		return err // never overwrite a newer client's snapshot
	} else if err == nil {
		prev = old.Origins
	}
	now := time.Now().Unix()
	s := &sessionSnapV2{Schema: sessionSchema, Device: device, SavedAt: now, Cookies: cookies, Origins: mergeOrigins(prev, fresh, withIDB, now)}
	applySessionCap(s, capBytes)
	return writeSessionFile(dir, s)
}

// ---- restore ----

const blankDoc = "<!doctype html><meta charset=utf-8><title></title>"

type restoreStats struct {
	Cookies int `json:"cookies"`
	Origins int `json:"origins"`
	LSKeys  int `json:"lsKeys"`
	IDBPut  int `json:"idbPut"`
	IDBErr  int `json:"idbErr"`
}

// restoreSessionV2 writes cookies (partitioned included), then LocalStorage
// and IndexedDB of every origin in the snapshot.
func restoreSessionV2(s *sessionSnapV2, ws string) (restoreStats, error) {
	var st restoreStats
	var c *cdpConn
	c, err := dialCDP(ws, func(m cdpMsg) {
		if m.Method != "Fetch.requestPaused" {
			return
		}
		var p struct {
			RequestID string `json:"requestId"`
		}
		_ = json.Unmarshal(m.Params, &p)
		_, _ = c.call(m.Session, "Fetch.fulfillRequest", map[string]any{"requestId": p.RequestID, "responseCode": 200,
			"responseHeaders": []map[string]string{{"name": "Content-Type", "value": "text/html; charset=utf-8"}, {"name": "Cache-Control", "value": "no-store"}},
			"body": base64.StdEncoding.EncodeToString([]byte(blankDoc))})
	})
	if err != nil {
		return st, err
	}
	defer c.c.Close()
	params := []map[string]any{}
	for _, raw := range s.Cookies {
		if p := cookieParamV2(raw); p != nil {
			params = append(params, p)
		}
	}
	st.Cookies = setCookiesV2(c, params)
	if len(s.Origins) == 0 {
		return st, nil
	}
	tid, sid, err := hiddenTab(c)
	if err != nil {
		return st, err
	}
	defer c.call("", "Target.closeTarget", map[string]any{"targetId": tid})
	_, _ = c.call(sid, "Network.enable", nil)
	_, _ = c.call(sid, "Network.setBypassServiceWorker", map[string]any{"bypass": true})
	if _, err = c.call(sid, "Fetch.enable", map[string]any{"patterns": []map[string]any{{"urlPattern": "*", "requestStage": "Request"}}}); err != nil {
		return st, err
	}
	restored := map[string]bool{}
	for _, o := range s.Origins {
		if storageOrigin(o.Origin) != o.Origin || len(o.Local) == 0 && len(o.IDB) == 0 {
			continue
		}
		if !openBlank(c, sid, o.Origin) {
			continue
		}
		ok := false
		if len(o.Local) > 0 {
			var n int
			b, _ := json.Marshal(o.Local)
			if evalValue(c, sid, 0, fmt.Sprintf("(%s)(%s)", jsRestoreLocal, b), &n) == nil {
				st.LSKeys += n
				ok = ok || n > 0
			}
		}
		for _, db := range o.IDB {
			var r struct{ Put, Err int }
			b, _ := json.Marshal(db)
			if evalValue(c, sid, 0, fmt.Sprintf("(%s)(%s)", jsRestoreIDB, b), &r) == nil {
				st.IDBPut += r.Put
				st.IDBErr += r.Err
				ok = ok || r.Put > 0
			} else {
				st.IDBErr++
			}
		}
		if ok {
			st.Origins++
			restored[o.Origin] = true
		}
	}
	// Tabs that loaded one of these sites before the restore see it after a reload.
	if tabs, err := pageTabs(c); err == nil {
		for _, t := range tabs {
			if t.TargetID != tid && restored[storageOrigin(t.URL)] {
				if s2, err := attachTarget(c, t.TargetID); err == nil {
					_, _ = c.call(s2, "Page.reload", nil)
					_, _ = c.call("", "Target.detachFromTarget", map[string]any{"sessionId": s2})
				}
			}
		}
	}
	return st, nil
}

func hiddenTab(c *cdpConn) (tid, sid string, err error) {
	raw, err := c.call("", "Target.createTarget", map[string]any{"url": "about:blank", "background": true, "hidden": true})
	if err != nil {
		raw, err = c.call("", "Target.createTarget", map[string]any{"url": "about:blank", "background": true})
	}
	if err != nil {
		return "", "", err
	}
	var t struct {
		TargetID string `json:"targetId"`
	}
	_ = json.Unmarshal(raw, &t)
	sid, err = attachTarget(c, t.TargetID)
	return t.TargetID, sid, err
}

// openBlank navigates the hidden tab to origin; Fetch interception answers
// with an empty document, so the page runs nothing of the site.
func openBlank(c *cdpConn, sid, origin string) bool {
	if _, err := c.call(sid, "Page.navigate", map[string]any{"url": origin + "/"}); err != nil {
		return false
	}
	for end := time.Now().Add(8 * time.Second); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
		var got string
		if evalValue(c, sid, 0, `document.readyState==="complete"?location.origin:""`, &got) == nil && got == origin {
			return true
		}
	}
	return false
}

// restoreSessionAny handles schema 2 (and refuses newer). It reports whether
// the legacy schema-1 path should be skipped.
func restoreSessionAny(b []byte, dir, device, ws string) bool {
	s, err := decodeSession(b)
	if err == errSessionSchema {
		return true
	}
	if err != nil || s.Schema < 2 {
		return false
	}
	if s.Device == device {
		return true
	}
	_, _ = restoreSessionV2(s, ws)
	on, capBytes := sessionCfg()
	if on {
		_ = snapshotSessionV2(dir, device, ws, true, capBytes)
	} else {
		_ = snapshotSession(dir, device, ws)
	}
	return true
}

// snapshotSessionTick is the 15-second refresh: IndexedDB every 4th tick.
func snapshotSessionTick(dir, device, ws string, n int) error {
	if on, capBytes := sessionCfg(); on {
		return snapshotSessionV2(dir, device, ws, n%sessionIDBEvery == 0, capBytes)
	}
	return snapshotSessionLegacy(dir, device, ws)
}

// ---- info (UI) ----

type SessionStoreInfo struct {
	Enabled     bool   `json:"enabled"`
	Schema      int    `json:"schema"`
	SavedAt     int64  `json:"savedAt"`
	Cookies     int    `json:"cookies"`
	Partitioned int    `json:"partitioned"`
	Origins     int    `json:"origins"`
	LSKeys      int    `json:"lsKeys"`
	IDBDBs      int    `json:"idbDbs"`
	IDBRecords  int    `json:"idbRecords"`
	IDBPartial  bool   `json:"idbPartial"`
	Dropped     int    `json:"dropped"`
	Bytes       int64  `json:"bytes"`
	CapBytes    int64  `json:"capBytes"`
	Error       string `json:"error,omitempty"`
}

func (s *Store) SessionStoreInfo(id string) SessionStoreInfo {
	st := s.GetSettings()
	info := SessionStoreInfo{Enabled: sessionStoreOn(st), CapBytes: sessionCapBytes(st)}
	dir := s.ProfileDir(id)
	if fi, err := os.Stat(filepath.Join(dir, sessionFile)); err == nil {
		info.Bytes = fi.Size()
	}
	snap, err := readSessionFile(dir)
	if err != nil {
		if err == errSessionSchema {
			info.Error = err.Error()
		}
		return info
	}
	info.Schema, info.SavedAt, info.Cookies, info.Origins, info.Dropped = snap.Schema, snap.SavedAt, len(snap.Cookies), len(snap.Origins), len(snap.Dropped)
	for _, raw := range snap.Cookies {
		var c struct {
			PK any `json:"partitionKey"`
		}
		if json.Unmarshal(raw, &c) == nil && c.PK != nil {
			info.Partitioned++
		}
	}
	for _, o := range snap.Origins {
		info.LSKeys += len(o.Local)
		info.IDBDBs += len(o.IDB)
		info.IDBPartial = info.IDBPartial || o.IDBPartial || o.IDBDropped
		for _, db := range o.IDB {
			for _, st := range db.Stores {
				info.IDBRecords += len(st.Records)
			}
		}
	}
	return info
}
