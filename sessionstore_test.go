package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSessionStoreSchemaV2RoundTrip(t *testing.T) {
	dir := t.TempDir()
	exp := float64(time.Now().Add(24 * time.Hour).Unix())
	ck, _ := json.Marshal(map[string]any{"name": "sid", "value": "1", "domain": ".a.com", "path": "/", "expires": exp, "secure": true, "httpOnly": true, "sameSite": "None",
		"partitionKey": map[string]any{"topLevelSite": "https://top.com", "hasCrossSiteAncestor": false}})
	in := &sessionSnapV2{Schema: sessionSchema, Device: "dev-a", SavedAt: 100, Cookies: []json.RawMessage{ck},
		Origins: []originStore{{Origin: "https://a.com", SavedAt: 100, Local: map[string]string{"token": "xyz"},
			IDB: []idbDatabase{{Name: "app", Version: 3, Stores: []idbStore{{Name: "kv", KeyPath: json.RawMessage(`null`), Records: []idbRecord{{K: json.RawMessage(`"k"`), V: json.RawMessage(`{"$t":"d","v":5}`)}}}}}}}}}
	if err := writeSessionFile(dir, in); err != nil {
		t.Fatal(err)
	}
	out, err := readSessionFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if out.Schema != 2 || out.Device != "dev-a" || len(out.Cookies) != 1 || len(out.Origins) != 1 || out.Origins[0].Local["token"] != "xyz" ||
		out.Origins[0].IDB[0].Stores[0].Records[0].K == nil || string(out.Origins[0].IDB[0].Stores[0].Records[0].V) != `{"$t":"d","v":5}` {
		t.Fatalf("round trip lost data: %+v", out)
	}
	// readSnapshot (cookie dialog) reads schema-2 cookies too.
	if list, at := readSnapshot(dir); len(list) != 1 || at != 100 {
		t.Fatalf("readSnapshot on v2: %d %d", len(list), at)
	}
}

func TestSessionStoreReadsSchemaV1(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, sessionFile), []byte(`{"device":"d1","savedAt":7,"cookies":[{"name":"a","value":"b","domain":"x.com"}]}`), 0600)
	s, err := readSessionFile(dir)
	if err != nil || s.Schema != 1 || len(s.Cookies) != 1 || s.Device != "d1" {
		t.Fatalf("v1: %+v %v", s, err)
	}
	// A v1 file never goes down the v2 restore path.
	b, _ := os.ReadFile(filepath.Join(dir, sessionFile))
	if restoreSessionAny(b, dir, "other", "ws://unused") {
		t.Fatal("v1 snapshot must use the legacy restore")
	}
}

func TestSessionStoreSchemaGuard(t *testing.T) {
	// Older client (v0.7, schema 1 only): it decodes "cookies" and finds none in
	// a schema-2 file, so it restores nothing instead of half a session.
	v2, _ := json.Marshal(&sessionSnapV2{Schema: 2, Device: "a", Cookies: []json.RawMessage{json.RawMessage(`{"name":"n","value":"v","domain":"d.com"}`)},
		Origins: []originStore{{Origin: "https://d.com", Local: map[string]string{"k": "v"}}}})
	var legacy sessionSnap
	if err := json.Unmarshal(v2, &legacy); err != nil || len(legacy.Cookies) != 0 {
		t.Fatalf("legacy client would see %d cookies of a v2 snapshot", len(legacy.Cookies))
	}
	// This client rejects a newer schema: no read, no restore, no overwrite.
	dir := t.TempDir()
	future := []byte(`{"schema":3,"device":"x","savedAt":1,"cookies2":[],"origins":[{"origin":"https://z.com"}]}`)
	_ = os.WriteFile(filepath.Join(dir, sessionFile), future, 0600)
	if _, err := readSessionFile(dir); err != errSessionSchema {
		t.Fatalf("schema 3 must be rejected, got %v", err)
	}
	if !restoreSessionAny(future, dir, "me", "ws://unused") {
		t.Fatal("schema 3 must not fall through to the legacy restore")
	}
	if err := snapshotSessionLegacy(dir, "me", "ws://127.0.0.1:1/x"); err != errSessionSchema {
		t.Fatalf("legacy snapshot must refuse to overwrite schema 3: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, sessionFile)); string(b) != string(future) {
		t.Fatal("newer snapshot was overwritten")
	}
	if list, _ := readSnapshot(dir); len(list) != 0 {
		t.Fatal("cookie dialog must ignore a newer schema")
	}
}

func TestSessionStorePartitionKey(t *testing.T) {
	exp := float64(time.Now().Add(time.Hour).Unix())
	obj, _ := json.Marshal(map[string]any{"name": "c", "value": "v", "domain": "cdn.x.net", "path": "/", "expires": exp, "secure": false, "session": false,
		"partitionKey": map[string]any{"topLevelSite": "https://top.com", "hasCrossSiteAncestor": false}})
	p := cookieParamV2(obj)
	pk, _ := p["partitionKey"].(map[string]any)
	if pk == nil || pk["topLevelSite"] != "https://top.com" || p["secure"] != true || p["expires"] != exp {
		t.Fatalf("partitionKey not kept: %+v", p)
	}
	if cookieParam(obj) != nil {
		t.Fatal("legacy param still drops partitioned cookies (unchanged behaviour)")
	}
	str, _ := json.Marshal(map[string]any{"name": "c", "value": "v", "domain": "cdn.x.net", "session": true, "partitionKey": "https://top.com"})
	if p := cookieParamV2(str); p["partitionKey"] != "https://top.com" {
		t.Fatalf("string partitionKey: %+v", p)
	}
	if altPartitionKey("https://top.com").(map[string]any)["topLevelSite"] != "https://top.com" || altPartitionKey(map[string]any{"topLevelSite": "https://t"}) != "https://t" {
		t.Fatal("altPartitionKey")
	}
	expired, _ := json.Marshal(map[string]any{"name": "c", "value": "v", "domain": "d", "expires": 10.0})
	if cookieParamV2(expired) != nil {
		t.Fatal("expired cookie must be skipped")
	}
}

func TestSessionStoreSizeCap(t *testing.T) {
	big := strings.Repeat("x", 4000)
	rec := []idbRecord{{K: json.RawMessage(`1`), V: json.RawMessage(`"` + big + `"`)}}
	db := []idbDatabase{{Name: "d", Version: 1, Stores: []idbStore{{Name: "s", KeyPath: json.RawMessage(`null`), Records: rec}}}}
	s := &sessionSnapV2{Schema: 2, Cookies: []json.RawMessage{json.RawMessage(`{"name":"a","value":"b","domain":"c"}`)}, Origins: []originStore{
		{Origin: "https://old.com", SavedAt: 1, Local: map[string]string{"k": big + big}},
		{Origin: "https://new.com", SavedAt: 3, Local: map[string]string{"k": "v"}, IDB: db},
		{Origin: "https://mid.com", SavedAt: 2, Local: map[string]string{"k": "v"}, IDB: db},
	}}
	applySessionCap(s, 6000)
	if len(s.Cookies) != 1 {
		t.Fatal("cookies must always be kept")
	}
	got := map[string]originStore{}
	for _, o := range s.Origins {
		got[o.Origin] = o
	}
	if n, ok := got["https://new.com"]; !ok || len(n.IDB) != 1 || n.IDBDropped {
		t.Fatalf("newest origin should keep IndexedDB: %+v", n)
	}
	if m, ok := got["https://mid.com"]; !ok || len(m.IDB) != 0 || !m.IDBDropped || m.Local["k"] != "v" {
		t.Fatalf("second origin keeps LocalStorage, drops IndexedDB: %+v", m)
	}
	if _, ok := got["https://old.com"]; ok || len(s.Dropped) != 1 || s.Dropped[0] != "https://old.com" {
		t.Fatalf("oldest oversized origin should be dropped: %v", s.Dropped)
	}
	if b, _ := json.Marshal(s); len(b) > 6000 {
		t.Fatalf("snapshot %d bytes exceeds cap", len(b))
	}
}

func TestSessionStoreOriginsAndMerge(t *testing.T) {
	for in, want := range map[string]string{"https://A.com/x?y": "https://a.com", "http://h:8080/": "http://h:8080", "chrome-extension://abc/p.html": "", "about:blank": "", "chrome://newtab/": "", "file:///c": ""} {
		if got := storageOrigin(in); got != want {
			t.Errorf("storageOrigin(%q)=%q want %q", in, got, want)
		}
	}
	now := int64(10_000_000)
	prev := []originStore{{Origin: "https://keep.com", SavedAt: now - 100, Local: map[string]string{"a": "1"}},
		{Origin: "https://stale.com", SavedAt: now - sessionOriginMaxAge - 1, Local: map[string]string{"a": "1"}},
		{Origin: "https://a.com", SavedAt: now - 100, Local: map[string]string{"old": "1"}, IDB: []idbDatabase{{Name: "x"}}, IDBAt: 5}}
	out := mergeOrigins(prev, map[string]originDump{"https://a.com": {Local: map[string]string{"new": "2"}}}, false, now)
	if len(out) != 2 || out[0].Origin != "https://a.com" || out[0].Local["new"] != "2" || len(out[0].IDB) != 1 || out[1].Origin != "https://keep.com" {
		t.Fatalf("merge: %+v", out)
	}
	if cap := sessionCapBytes(Settings{}); cap != 32<<20 {
		t.Fatalf("default cap %d", cap)
	}
	off := false
	if sessionStoreOn(Settings{SyncSessionStorage: &off}) || !sessionStoreOn(Settings{}) {
		t.Fatal("toggle default must be on")
	}
}
