package main

// A minimal WebDriver BiDi client for the Firefox kernel. Firefox started with
// --remote-debugging-port=N serves BiDi at ws://127.0.0.1:N/session and allows
// one session at a time, so GhostBrowser keeps a single connection per running
// profile (firefoxRun) and every feature goes through it.
//
// Only what the manager needs is implemented: session.new / session.end,
// browsingContext.getTree / navigate / activate / create, script.evaluate,
// storage.getCookies / storage.setCookie and browser.close.

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type bidiMsg struct {
	Type    string          `json:"type"` // success | error | event
	ID      int64           `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   string          `json:"error,omitempty"`
	Message string          `json:"message,omitempty"`
}

type bidiClient struct {
	ws      *websocket.Conn
	wmu     sync.Mutex
	next    atomic.Int64
	mu      sync.Mutex
	pending map[int64]chan bidiMsg
	closed  chan struct{}
	once    sync.Once
	onEvent func(bidiMsg)
	Timeout time.Duration
}

// BiDiError is an error answer from the browser.
type BiDiError struct{ Code, Message string }

func (e *BiDiError) Error() string { return "bidi " + e.Code + ": " + e.Message }

var errBiDiClosed = errors.New("bidi: connection closed")

// dialBiDi connects to ws://127.0.0.1:port/session.
func dialBiDi(url string, onEvent func(bidiMsg)) (*bidiClient, error) {
	d := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	ws, _, err := d.Dial(url, http.Header{})
	if err != nil {
		return nil, err
	}
	ws.SetReadLimit(64 << 20)
	c := &bidiClient{ws: ws, pending: map[int64]chan bidiMsg{}, closed: make(chan struct{}), onEvent: onEvent, Timeout: 20 * time.Second}
	go c.read()
	return c, nil
}

func (c *bidiClient) read() {
	defer c.shutdown()
	for {
		var m bidiMsg
		if err := c.ws.ReadJSON(&m); err != nil {
			return
		}
		if m.Type == "event" || (m.ID == 0 && m.Method != "") {
			if c.onEvent != nil {
				c.onEvent(m)
			}
			continue
		}
		c.mu.Lock()
		ch := c.pending[m.ID]
		delete(c.pending, m.ID)
		c.mu.Unlock()
		if ch != nil {
			ch <- m
		}
	}
}

func (c *bidiClient) shutdown() {
	c.once.Do(func() {
		close(c.closed)
		_ = c.ws.Close()
	})
}

// Close drops the connection without closing the browser.
func (c *bidiClient) Close() {
	if c != nil {
		c.shutdown()
	}
}

// Done is closed when the websocket goes away (the browser exited or crashed).
func (c *bidiClient) Done() <-chan struct{} { return c.closed }

// Call sends one command and waits for its answer.
func (c *bidiClient) Call(method string, params any) (json.RawMessage, error) {
	if params == nil {
		params = map[string]any{}
	}
	id := c.next.Add(1)
	ch := make(chan bidiMsg, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()
	c.wmu.Lock()
	err := c.ws.WriteJSON(map[string]any{"id": id, "method": method, "params": params})
	c.wmu.Unlock()
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, err
	}
	t := c.Timeout
	if t <= 0 {
		t = 20 * time.Second
	}
	select {
	case m := <-ch:
		if m.Type == "error" || m.Error != "" {
			return nil, &BiDiError{Code: m.Error, Message: m.Message}
		}
		return m.Result, nil
	case <-c.closed:
		return nil, errBiDiClosed
	case <-time.After(t):
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("bidi %s: timeout", method)
	}
}

// NewSession starts the (single) BiDi session.
func (c *bidiClient) NewSession() (string, error) {
	raw, err := c.Call("session.new", map[string]any{"capabilities": map[string]any{}})
	if err != nil {
		return "", err
	}
	var r struct {
		SessionID string `json:"sessionId"`
	}
	_ = json.Unmarshal(raw, &r)
	return r.SessionID, nil
}

// BiDiContext is one browsing context (a tab) from browsingContext.getTree.
type BiDiContext struct {
	Context  string        `json:"context"`
	URL      string        `json:"url"`
	Children []BiDiContext `json:"children"`
}

// Contexts lists the top-level browsing contexts (tabs).
func (c *bidiClient) Contexts() ([]BiDiContext, error) {
	raw, err := c.Call("browsingContext.getTree", map[string]any{"maxDepth": 0})
	if err != nil {
		return nil, err
	}
	var r struct {
		Contexts []BiDiContext `json:"contexts"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	return r.Contexts, nil
}

// firstContext returns the first tab, or "" when there is none.
func (c *bidiClient) firstContext() (string, error) {
	l, err := c.Contexts()
	if err != nil {
		return "", err
	}
	if len(l) == 0 {
		return "", errors.New("没有打开的标签页")
	}
	return l[0].Context, nil
}

// Navigate loads url in ctx and waits until the document is complete.
func (c *bidiClient) Navigate(ctx, url string) error {
	_, err := c.Call("browsingContext.navigate", map[string]any{"context": ctx, "url": url, "wait": "complete"})
	return err
}

// NewTab opens a new tab and returns its context id.
func (c *bidiClient) NewTab() (string, error) {
	raw, err := c.Call("browsingContext.create", map[string]any{"type": "tab"})
	if err != nil {
		return "", err
	}
	var r struct {
		Context string `json:"context"`
	}
	_ = json.Unmarshal(raw, &r)
	return r.Context, nil
}

// Activate brings the tab (and its window) to the front.
func (c *bidiClient) Activate(ctx string) error {
	_, err := c.Call("browsingContext.activate", map[string]any{"context": ctx})
	return err
}

// Eval runs a JS expression in ctx (promises awaited) and decodes the
// result's serialized value into a Go value (strings, numbers, bools, null,
// arrays and plain objects).
func (c *bidiClient) Eval(ctx, expr string) (any, error) {
	raw, err := c.Call("script.evaluate", map[string]any{
		"expression": expr, "awaitPromise": true,
		"target": map[string]any{"context": ctx},
	})
	if err != nil {
		return nil, err
	}
	var r struct {
		Type             string          `json:"type"`
		Result           json.RawMessage `json:"result"`
		ExceptionDetails struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	if r.Type == "exception" {
		return nil, fmt.Errorf("script: %s", r.ExceptionDetails.Text)
	}
	return bidiValue(r.Result), nil
}

// bidiValue turns a BiDi RemoteValue into plain Go values.
func bidiValue(raw json.RawMessage) any {
	var v struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	}
	if json.Unmarshal(raw, &v) != nil {
		return nil
	}
	switch v.Type {
	case "string", "boolean":
		var x any
		_ = json.Unmarshal(v.Value, &x)
		return x
	case "number":
		var f float64
		if json.Unmarshal(v.Value, &f) == nil {
			return f
		}
		var s string // NaN, -0, Infinity
		_ = json.Unmarshal(v.Value, &s)
		return s
	case "array":
		var items []json.RawMessage
		_ = json.Unmarshal(v.Value, &items)
		out := make([]any, len(items))
		for i, it := range items {
			out[i] = bidiValue(it)
		}
		return out
	case "object":
		var pairs [][2]json.RawMessage
		_ = json.Unmarshal(v.Value, &pairs)
		out := map[string]any{}
		for _, p := range pairs {
			var k string
			if json.Unmarshal(p[0], &k) != nil {
				k, _ = bidiValue(p[0]).(string)
			}
			out[k] = bidiValue(p[1])
		}
		return out
	}
	return nil // undefined, null and non-serializable values
}

// bidiCookie is the BiDi network.Cookie shape.
type bidiCookie struct {
	Name  string `json:"name"`
	Value struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	HTTPOnly bool   `json:"httpOnly"`
	Secure   bool   `json:"secure"`
	SameSite string `json:"sameSite"` // strict | lax | none | default
	Expiry   *int64 `json:"expiry,omitempty"`
}

// GetCookies returns every cookie in the default user context as GhostBrowser cookies.
func (c *bidiClient) GetCookies() ([]Cookie, error) {
	raw, err := c.Call("storage.getCookies", map[string]any{})
	if err != nil {
		return nil, err
	}
	var r struct {
		Cookies []bidiCookie `json:"cookies"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	out := make([]Cookie, 0, len(r.Cookies))
	for _, b := range r.Cookies {
		ck := Cookie{Name: b.Name, Value: b.Value.Value, Domain: b.Domain, Path: b.Path, HTTPOnly: b.HTTPOnly, Secure: b.Secure, SameSite: normSameSite(b.SameSite)}
		if b.Expiry != nil {
			ck.Expires = float64(*b.Expiry)
		}
		if ck.normalize() {
			out = append(out, ck)
		}
	}
	return out, nil
}

// bidiCookieParam is what storage.setCookie accepts for one cookie.
func bidiCookieParam(c Cookie) map[string]any {
	m := map[string]any{
		"name": c.Name, "value": map[string]any{"type": "string", "value": c.Value},
		"domain": c.Domain, "path": c.Path, "httpOnly": c.HTTPOnly, "secure": c.Secure,
	}
	if m["path"] == "" {
		m["path"] = "/"
	}
	if c.Expires > 0 {
		m["expiry"] = int64(c.Expires)
	}
	if s := normSameSite(c.SameSite); s != "" {
		m["sameSite"] = map[string]string{"Lax": "lax", "Strict": "strict", "None": "none"}[s]
		if s == "None" {
			m["secure"] = true
		}
	}
	return m
}

// SetCookies writes cookies one by one (BiDi has no batch call) and returns how many stuck.
func (c *bidiClient) SetCookies(list []Cookie) (int, error) {
	ok := 0
	var lastErr error
	for _, ck := range list {
		if _, err := c.Call("storage.setCookie", map[string]any{"cookie": bidiCookieParam(ck)}); err != nil {
			lastErr = err
			if errors.Is(err, errBiDiClosed) {
				break
			}
			continue
		}
		ok++
	}
	if ok == 0 && lastErr != nil {
		return 0, lastErr
	}
	return ok, nil
}

// CloseBrowser asks Firefox to close every window and exit.
func (c *bidiClient) CloseBrowser() error {
	_, err := c.Call("browser.close", map[string]any{})
	if errors.Is(err, errBiDiClosed) {
		return nil // the socket closing is the browser going away
	}
	return err
}
