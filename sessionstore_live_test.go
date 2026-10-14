package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// GHOST_CHROME=/usr/bin/chromium go test -run TestSessionStoreLive
// Sets LocalStorage, IndexedDB (rich types) and a partitioned cookie, takes a
// schema-2 snapshot, wipes the browser, restores as another device, verifies.
func TestSessionStoreLive(t *testing.T) {
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body>site</body></html>")
	}))
	defer srv.Close()
	origin := storageOrigin(srv.URL)

	port := freePort()
	cmd := exec.Command(bin, "--headless=new", "--no-sandbox", "--user-data-dir="+t.TempDir(), fmt.Sprintf("--remote-debugging-port=%d", port), "about:blank")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	ws, err := waitDebugger(port, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	c, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.c.Close()
	if v, err := c.call("", "Browser.getVersion", nil); err == nil {
		t.Logf("browser: %s", v)
	}

	openTab := func() (string, string) {
		raw, err := c.call("", "Target.createTarget", map[string]any{"url": srv.URL + "/app"})
		if err != nil {
			t.Fatal(err)
		}
		var tt struct{ TargetID string }
		_ = json.Unmarshal(raw, &tt)
		sid, err := attachTarget(c, tt.TargetID)
		if err != nil {
			t.Fatal(err)
		}
		for end := time.Now().Add(10 * time.Second); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
			var o string
			if evalValue(c, sid, 0, `document.readyState==="complete"?location.origin:""`, &o) == nil && o == origin {
				return tt.TargetID, sid
			}
		}
		t.Fatal("tab did not load")
		return "", ""
	}
	tid, sid := openTab()
	seed := `(async () => {
	  localStorage.setItem('auth', 'tok-123'); localStorage.setItem('$t', 'tricky');
	  await new Promise((ok, no) => { const r = indexedDB.open('appdb', 3);
	    r.onupgradeneeded = () => { const db = r.result;
	      const s = db.createObjectStore('users', { keyPath: 'id' }); s.createIndex('byName', 'name', { unique: false });
	      db.createObjectStore('kv'); };
	    r.onsuccess = () => { const db = r.result; const tx = db.transaction(['users', 'kv'], 'readwrite');
	      tx.objectStore('users').put({ id: 1, name: 'ann', at: new Date(1700000000000), bytes: new Uint8Array([1, 2, 255]), tags: new Set(['a']), m: new Map([[1, 'x']]), $t: 'own', big: 12345678901234567890n, nan: NaN });
	      tx.objectStore('kv').put(new Blob(['hello'], { type: 'text/plain' }), 'blob');
	      tx.objectStore('kv').put('v2', [1, 'compound']);
	      tx.oncomplete = () => { db.close(); ok(); }; tx.onerror = () => no(tx.error); };
	    r.onerror = () => no(r.error); });
	  return true; })()`
	if err := evalValue(c, sid, 0, seed, nil); err != nil {
		t.Fatal(err)
	}
	exp := float64(time.Now().Add(48 * time.Hour).Unix())
	if _, err := c.call("", "Storage.setCookies", map[string]any{"cookies": []map[string]any{
		{"name": "plain", "value": "p1", "domain": "127.0.0.1", "path": "/", "expires": exp},
		{"name": "chips", "value": "c1", "domain": "cdn.example.net", "path": "/", "secure": true, "sameSite": "None", "expires": exp,
			"partitionKey": map[string]any{"topLevelSite": "https://example.com", "hasCrossSiteAncestor": false}},
	}}); err != nil {
		t.Fatalf("set partitioned cookie: %v", err)
	}

	dir := t.TempDir()
	if err := snapshotSessionV2(dir, "dev-A", ws, true, 32<<20); err != nil {
		t.Fatal(err)
	}
	snap, err := readSessionFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(dir + "/" + sessionFile)
	t.Logf("snapshot %d bytes, %d cookies, %d origins", len(b), len(snap.Cookies), len(snap.Origins))
	if !strings.Contains(string(b), `"partitionKey"`) {
		t.Fatal("snapshot lacks the partitioned cookie's partitionKey")
	}
	if len(snap.Origins) != 1 || snap.Origins[0].Local["auth"] != "tok-123" || len(snap.Origins[0].IDB) != 1 {
		t.Fatalf("origin capture: %s", b)
	}
	if snap.Origins[0].IDBPartial {
		t.Fatalf("IndexedDB capture was partial: %s", b)
	}

	// Wipe everything as if this were a fresh device.
	wipe := `(async () => { localStorage.clear(); await new Promise(ok => { const r = indexedDB.deleteDatabase('appdb'); r.onsuccess = r.onerror = r.onblocked = () => ok(); });
	  return (await indexedDB.databases()).length + localStorage.length; })()`
	var left int
	if err := evalValue(c, sid, 0, wipe, &left); err != nil || left != 0 {
		t.Fatalf("wipe: %v left=%d", err, left)
	}
	_, _ = c.call("", "Target.closeTarget", map[string]any{"targetId": tid})
	_, _ = c.call("", "Storage.clearCookies", nil)
	if raw, _ := exportCookies(ws); strings.Contains(raw, "chips") {
		t.Fatal("cookies not cleared")
	}
	before := hits
	snap.Device = "dev-B"
	st, err := restoreSessionV2(snap, ws)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("restore: %+v", st)
	if hits != before {
		t.Fatalf("restore reached the site's server (%d requests); it must be intercepted", hits-before)
	}
	raw, _ := exportCookies(ws)
	if !strings.Contains(raw, `"chips"`) || !strings.Contains(raw, `"plain"`) || !strings.Contains(raw, "example.com") {
		t.Fatalf("cookies not restored (partitioned included): %s", raw)
	}

	_, sid = openTab()
	var got struct {
		Auth, Tricky, Name, Blob, Compound, Big string
		At                                   int64
		Bytes                                []int
		Tag, MapV, Own                       string
		NaN, Indexed                         bool
	}
	check := `(async () => {
	  const db = await new Promise((ok, no) => { const r = indexedDB.open('appdb'); r.onsuccess = () => ok(r.result); r.onerror = () => no(r.error); });
	  const g = (s, k) => new Promise(ok => { const q = db.transaction(s).objectStore(s).get(k); q.onsuccess = () => ok(q.result); });
	  const u = await g('users', 1); const bl = await g('kv', 'blob'); const cp = await g('kv', [1, 'compound']);
	  const ix = await new Promise(ok => { const q = db.transaction('users').objectStore('users').index('byName').get('ann'); q.onsuccess = () => ok(!!q.result); });
	  const r = { Auth: localStorage.getItem('auth'), Tricky: localStorage.getItem('$t'), Name: u && u.name, At: u && u.at.getTime(), Bytes: u && Array.from(u.bytes),
	    Tag: u && [...u.tags][0], MapV: u && u.m.get(1), Own: u && u.$t, Big: u && String(u.big), NaN: u && Number.isNaN(u.nan), Indexed: ix && db.version === 3,
	    Blob: bl ? await bl.text() : '', Compound: cp };
	  db.close(); return r; })()`
	if err := evalValue(c, sid, 0, check, &got); err != nil {
		t.Fatal(err)
	}
	t.Logf("after restore: %+v", got)
	if got.Auth != "tok-123" || got.Tricky != "tricky" || got.Name != "ann" || got.At != 1700000000000 || fmt.Sprint(got.Bytes) != "[1 2 255]" ||
		got.Tag != "a" || got.MapV != "x" || got.Own != "own" || got.Big != "12345678901234567890" || !got.NaN || !got.Indexed || got.Blob != "hello" || got.Compound != "v2" {
		t.Fatalf("restored data differs: %+v", got)
	}
}
