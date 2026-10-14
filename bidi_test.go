package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// fakeBiDi is a tiny in-memory WebDriver BiDi endpoint.
type fakeBiDi struct {
	mu      sync.Mutex
	cookies []map[string]any
	calls   []string
	closed  chan struct{}
}

func (f *fakeBiDi) handler(t *testing.T) http.HandlerFunc {
	up := websocket.Upgrader{}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/session" {
			http.NotFound(w, r)
			return
		}
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			var m struct {
				ID     int64           `json:"id"`
				Method string          `json:"method"`
				Params json.RawMessage `json:"params"`
			}
			if c.ReadJSON(&m) != nil {
				return
			}
			f.mu.Lock()
			f.calls = append(f.calls, m.Method)
			f.mu.Unlock()
			ok := func(res any) { c.WriteJSON(map[string]any{"type": "success", "id": m.ID, "result": res}) }
			switch m.Method {
			case "session.new":
				c.WriteJSON(map[string]any{"type": "event", "method": "log.entryAdded", "params": map[string]any{}}) // events interleave
				ok(map[string]any{"sessionId": "s1", "capabilities": map[string]any{}})
			case "browsingContext.getTree":
				ok(map[string]any{"contexts": []any{map[string]any{"context": "ctx-1", "url": "about:blank", "children": []any{}}}})
			case "browsingContext.activate", "browsingContext.navigate":
				var p map[string]any
				json.Unmarshal(m.Params, &p)
				if p["context"] != "ctx-1" {
					c.WriteJSON(map[string]any{"type": "error", "id": m.ID, "error": "no such frame", "message": "bad ctx"})
					continue
				}
				ok(map[string]any{})
			case "script.evaluate":
				var p struct {
					Expression string `json:"expression"`
				}
				json.Unmarshal(m.Params, &p)
				switch p.Expression {
				case "navigator.language":
					ok(map[string]any{"type": "success", "realm": "r", "result": map[string]any{"type": "string", "value": "de-DE"}})
				case "obj":
					ok(map[string]any{"type": "success", "result": map[string]any{"type": "object", "value": []any{
						[]any{"n", map[string]any{"type": "number", "value": 4}},
						[]any{"a", map[string]any{"type": "array", "value": []any{map[string]any{"type": "boolean", "value": true}, map[string]any{"type": "null"}}}},
					}}})
				default:
					ok(map[string]any{"type": "exception", "exceptionDetails": map[string]any{"text": "ReferenceError: x"}})
				}
			case "storage.getCookies":
				f.mu.Lock()
				ok(map[string]any{"cookies": f.cookies, "partitionKey": map[string]any{}})
				f.mu.Unlock()
			case "storage.setCookie":
				var p struct {
					Cookie map[string]any `json:"cookie"`
				}
				json.Unmarshal(m.Params, &p)
				if p.Cookie["name"] == "bad" {
					c.WriteJSON(map[string]any{"type": "error", "id": m.ID, "error": "unable to set cookie", "message": "nope"})
					continue
				}
				p.Cookie["size"] = 1
				f.mu.Lock()
				f.cookies = append(f.cookies, p.Cookie)
				f.mu.Unlock()
				ok(map[string]any{"partitionKey": map[string]any{}})
			case "browser.close":
				ok(map[string]any{})
				close(f.closed)
				return
			default:
				c.WriteJSON(map[string]any{"type": "error", "id": m.ID, "error": "unknown command", "message": m.Method})
			}
		}
	}
}

func startFakeBiDi(t *testing.T) (*fakeBiDi, string) {
	f := &fakeBiDi{closed: make(chan struct{})}
	srv := httptest.NewServer(f.handler(t))
	t.Cleanup(srv.Close)
	return f, "ws" + strings.TrimPrefix(srv.URL, "http") + "/session"
}

func TestBiDiClientCommands(t *testing.T) {
	f, url := startFakeBiDi(t)
	var events int
	var emu sync.Mutex
	c, err := dialBiDi(url, func(bidiMsg) { emu.Lock(); events++; emu.Unlock() })
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if sid, err := c.NewSession(); err != nil || sid != "s1" {
		t.Fatalf("session %q %v", sid, err)
	}
	ctx, err := c.firstContext()
	if err != nil || ctx != "ctx-1" {
		t.Fatalf("ctx %q %v", ctx, err)
	}
	if err := c.Activate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.Navigate("zzz", "http://x"); err == nil || !strings.Contains(err.Error(), "no such frame") {
		t.Fatalf("expected bidi error, got %v", err)
	}
	if v, err := c.Eval(ctx, "navigator.language"); err != nil || v != "de-DE" {
		t.Fatalf("eval %v %v", v, err)
	}
	v, err := c.Eval(ctx, "obj")
	m, _ := v.(map[string]any)
	if err != nil || m["n"] != 4.0 || len(m["a"].([]any)) != 2 || m["a"].([]any)[0] != true {
		t.Fatalf("obj %#v %v", v, err)
	}
	if _, err := c.Eval(ctx, "x"); err == nil || !strings.Contains(err.Error(), "ReferenceError") {
		t.Fatalf("exception not surfaced: %v", err)
	}
	if _, err := c.Call("nope.nope", nil); err == nil {
		t.Fatal("unknown command must fail")
	}
	emu.Lock()
	if events != 1 {
		t.Fatalf("events %d", events)
	}
	emu.Unlock()

	// Cookies round trip, one bad cookie does not stop the rest.
	exp := float64(time.Now().Add(24 * time.Hour).Unix())
	n, err := c.SetCookies([]Cookie{
		{Name: "sid", Value: "abc", Domain: ".example.com", Path: "/", Expires: exp, HTTPOnly: true, Secure: true, SameSite: "Lax"},
		{Name: "bad", Value: "x", Domain: "example.com"},
		{Name: "s2", Value: "v", Domain: "a.test", SameSite: "None"},
	})
	if err != nil || n != 2 {
		t.Fatalf("set %d %v", n, err)
	}
	list, err := c.GetCookies()
	if err != nil || len(list) != 2 {
		t.Fatalf("get %v %v", list, err)
	}
	if list[0].Name != "sid" || list[0].Expires != exp || !list[0].HTTPOnly || list[0].SameSite != "Lax" || list[0].Value != "abc" {
		t.Fatalf("cookie 0 %+v", list[0])
	}
	if list[1].Expires != 0 || !list[1].Secure || list[1].SameSite != "None" {
		t.Fatalf("SameSite=None must be secure, session cookie no expiry: %+v", list[1])
	}

	// Session snapshot through the adapter, readable by cookies.go, restored on another device.
	dir := t.TempDir()
	if err := firefoxSnapshotSession(dir, "devA", c); err != nil {
		t.Fatal(err)
	}
	snap, at := readSnapshot(dir)
	if len(snap) != 2 || at == 0 {
		t.Fatalf("snapshot %v", snap)
	}
	b, _ := os.ReadFile(filepath.Join(dir, sessionFile))
	if !strings.Contains(string(b), `"device":"devA"`) || !strings.Contains(string(b), `"session":true`) {
		t.Fatalf("snapshot json %s", b)
	}
	f.mu.Lock()
	f.cookies = nil
	f.mu.Unlock()
	if n := firefoxRestoreSession(dir, "devA", c); n != 0 {
		t.Fatal("same device must not restore")
	}
	if n := firefoxRestoreSession(dir, "devB", c); n != 2 {
		t.Fatalf("restored %d", n)
	}
	b, _ = os.ReadFile(filepath.Join(dir, sessionFile))
	if !strings.Contains(string(b), `"device":"devB"`) {
		t.Fatal("restore must re-snapshot as this device")
	}

	if err := c.CloseBrowser(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-f.closed:
	case <-time.After(2 * time.Second):
		t.Fatal("browser.close not received")
	}
	select {
	case <-c.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("client did not notice the socket closing")
	}
	if _, err := c.Contexts(); err == nil {
		t.Fatal("call after close must fail")
	}
}

func TestBiDiCallTimeout(t *testing.T) {
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := up.Upgrade(w, r, nil)
		defer c.Close()
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	c, err := dialBiDi("ws"+strings.TrimPrefix(srv.URL, "http")+"/session", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Timeout = 200 * time.Millisecond
	if _, err := c.Call("session.status", nil); err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("want timeout, got %v", err)
	}
}
