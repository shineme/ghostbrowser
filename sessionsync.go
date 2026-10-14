package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Chrome on Windows encrypts its cookie database with a key bound to the
// Windows user (DPAPI), so a synced "Cookies" file is unreadable on another
// computer and the user would be logged out. While a browser runs we keep a
// plain snapshot of its cookies inside the profile folder (it is synced inside
// the encrypted blob). When the profile next starts on a different device, the
// snapshot is written back through CDP, restoring every login.

const sessionFile = "ghost-session.json"

type sessionSnap struct {
	Device  string            `json:"device"`
	SavedAt int64             `json:"savedAt"`
	Cookies []json.RawMessage `json:"cookies"`
}

func snapshotSession(dir, device, ws string) error {
	if ws == "" || !strings.HasPrefix(ws, "ws") {
		return nil
	}
	// session store: schema 2 with LocalStorage / IndexedDB / partitioned cookies (sessionstore.go).
	if on, capBytes := sessionCfg(); on {
		return snapshotSessionV2(dir, device, ws, true, capBytes)
	}
	return snapshotSessionLegacy(dir, device, ws)
}

// snapshotSessionLegacy writes the schema-1 (cookies only) snapshot.
func snapshotSessionLegacy(dir, device, ws string) error {
	if ws == "" || !strings.HasPrefix(ws, "ws") {
		return nil
	}
	if _, err := readSessionFile(dir); err == errSessionSchema {
		return err // never overwrite a newer client's snapshot
	}
	raw, err := exportCookies(ws)
	if err != nil {
		return err
	}
	var list []json.RawMessage
	if json.Unmarshal([]byte(raw), &list) != nil {
		return nil
	}
	b, _ := json.Marshal(sessionSnap{Device: device, SavedAt: time.Now().Unix(), Cookies: list})
	tmp := filepath.Join(dir, sessionFile+".tmp")
	if err = os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, sessionFile))
}

// cookieParam keeps the fields Storage.setCookies accepts.
func cookieParam(raw json.RawMessage) map[string]any {
	var c map[string]any
	if json.Unmarshal(raw, &c) != nil || c["name"] == nil || c["domain"] == nil {
		return nil
	}
	if _, partitioned := c["partitionKey"]; partitioned {
		return nil // partition key formats differ between Chromium versions
	}
	out := map[string]any{}
	for _, k := range []string{"name", "value", "domain", "path", "secure", "httpOnly", "sameSite", "priority", "sourceScheme", "sourcePort"} {
		if v, ok := c[k]; ok {
			out[k] = v
		}
	}
	if s, _ := c["session"].(bool); !s {
		if e, ok := c["expires"].(float64); ok && e > float64(time.Now().Unix()) {
			out["expires"] = e
		} else if ok {
			return nil // already expired
		}
	}
	return out
}

// restoreSession writes the snapshot back when it was taken on another device.
func restoreSession(dir, device, ws string) {
	b, err := os.ReadFile(filepath.Join(dir, sessionFile))
	if err != nil {
		return
	}
	if restoreSessionAny(b, dir, device, ws) { // schema 2 (sessionstore.go)
		return
	}
	var snap sessionSnap
	if json.Unmarshal(b, &snap) != nil || snap.Device == device || len(snap.Cookies) == 0 {
		return
	}
	params := []map[string]any{}
	for _, raw := range snap.Cookies {
		if p := cookieParam(raw); p != nil {
			params = append(params, p)
		}
	}
	// In chunks: one bad cookie must not drop the whole set.
	for i := 0; i < len(params); i += 200 {
		j := min(i+200, len(params))
		if _, err := cdpCall(ws, "Storage.setCookies", map[string]any{"cookies": params[i:j]}); err != nil {
			for _, one := range params[i:j] {
				_, _ = cdpCall(ws, "Storage.setCookies", map[string]any{"cookies": []map[string]any{one}})
			}
		}
	}
	_ = snapshotSession(dir, device, ws)
}

// afterStart restores a session from another device and keeps the snapshot
// fresh while the browser runs (every 15 s, plus before a stop from the app).
func (s *Store) afterStart(id, ws string) {
	sessionCfgStore.Store(s)
	dir := s.ProfileDir(id)
	device := s.GetSettings().DeviceID
	restoreSession(dir, device, ws)
	go func() {
		time.Sleep(5 * time.Second)
		for n := 1; ; n++ {
			rs, ok := runReg.get(id)
			if !ok || rs.WSURL != ws {
				return
			}
			_ = snapshotSessionTick(dir, device, ws, n)
			time.Sleep(15 * time.Second)
		}
	}()
}
