package undetectable

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// cdpTarget is one entry of the DevTools HTTP /json/list endpoint.
type cdpTarget struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	Title                string `json:"title"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

type cdpVersion struct {
	Browser              string `json:"Browser"`
	UserAgent            string `json:"User-Agent"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// CDP is a minimal DevTools Protocol client over a single WebSocket.
//
// A reader goroutine owns the connection: it matches responses to pending
// calls by id and hands protocol events to the registered handlers. Events are
// delivered on their own goroutine so a handler may issue further calls.
type CDP struct {
	conn *websocket.Conn

	mu      sync.Mutex
	next    int
	waiters map[int]chan cdpMessage

	// writeMu serializes frames: gorilla/websocket allows only one concurrent
	// writer, and event handlers issue commands from their own goroutines.
	writeMu sync.Mutex

	handlers map[string][]func(cdpMessage)

	closeOnce sync.Once
	closed    chan struct{}
	err       error
}

func DialCDP(wsURL string, timeout time.Duration) (*CDP, error) {
	d := websocket.Dialer{HandshakeTimeout: timeout}
	// Chrome 153 rejects DevTools WebSocket handshakes that carry an Origin
	// header unless --remote-allow-origins is passed, so send none.
	c, _, err := d.Dial(wsURL, nil)
	if err != nil {
		return nil, err
	}
	cdp := &CDP{
		conn:     c,
		waiters:  map[int]chan cdpMessage{},
		handlers: map[string][]func(cdpMessage){},
		closed:   make(chan struct{}),
	}
	go cdp.readLoop()
	return cdp, nil
}

type cdpMessage struct {
	ID        int             `json:"id"`
	Method    string          `json:"method"`
	SessionID string          `json:"sessionId"`
	Params    json.RawMessage `json:"params"`
	Result    json.RawMessage `json:"result"`
	Error     *cdpError       `json:"error"`
}

func (c *CDP) readLoop() {
	defer func() {
		c.closeOnce.Do(func() { close(c.closed) })
		c.mu.Lock()
		for _, ch := range c.waiters {
			close(ch)
		}
		c.waiters = map[int]chan cdpMessage{}
		c.mu.Unlock()
	}()
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			c.err = err
			c.mu.Unlock()
			return
		}
		var msg cdpMessage
		if json.Unmarshal(data, &msg) != nil {
			continue
		}
		if msg.ID != 0 {
			c.mu.Lock()
			ch := c.waiters[msg.ID]
			delete(c.waiters, msg.ID)
			c.mu.Unlock()
			if ch != nil {
				ch <- msg
				close(ch)
			}
			continue
		}
		if msg.Method == "" {
			continue
		}
		c.mu.Lock()
		hs := append([]func(cdpMessage){}, c.handlers[msg.Method]...)
		c.mu.Unlock()
		for _, h := range hs {
			go h(msg)
		}
	}
}

func (c *CDP) Close() error {
	err := c.conn.Close()
	<-c.closed
	return err
}

// On registers a handler for a protocol event.
func (c *CDP) On(method string, h func(cdpMessage)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[method] = append(c.handlers[method], h)
}

type cdpError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (e *cdpError) Error() string {
	if len(e.Data) > 0 && string(e.Data) != "null" {
		return fmt.Sprintf("%s (%s)", e.Message, string(e.Data))
	}
	return e.Message
}

func (c *CDP) Call(method string, params any, result any) error {
	return c.CallSession("", method, params, result)
}

// CallSession issues a command, optionally scoped to an attached target
// session (flattened mode).
func (c *CDP) CallSession(session, method string, params any, result any) error {
	c.mu.Lock()
	c.next++
	id := c.next
	ch := make(chan cdpMessage, 1)
	c.waiters[id] = ch
	c.mu.Unlock()

	req := map[string]any{"id": id, "method": method}
	if params != nil {
		req["params"] = params
	}
	if session != "" {
		req["sessionId"] = session
	}
	c.writeMu.Lock()
	err := c.conn.WriteJSON(req)
	c.writeMu.Unlock()
	if err != nil {
		c.mu.Lock()
		delete(c.waiters, id)
		c.mu.Unlock()
		return err
	}
	select {
	case msg, ok := <-ch:
		if !ok {
			c.mu.Lock()
			err := c.err
			c.mu.Unlock()
			if err == nil {
				err = fmt.Errorf("connection closed")
			}
			return err
		}
		if msg.Error != nil {
			return msg.Error
		}
		if result != nil && len(msg.Result) > 0 {
			return json.Unmarshal(msg.Result, result)
		}
		return nil
	case <-time.After(30 * time.Second):
		c.mu.Lock()
		delete(c.waiters, id)
		c.mu.Unlock()
		return fmt.Errorf("%s timed out", method)
	case <-c.closed:
		return fmt.Errorf("connection closed")
	}
}

// EvaluateIn runs an expression in a target session ("" for the default one).
func (c *CDP) EvaluateIn(session, expr string, out any) error {
	var res struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails json.RawMessage `json:"exceptionDetails"`
	}
	if err := c.CallSession(session, "Runtime.evaluate", map[string]any{
		"expression": expr, "returnByValue": true, "awaitPromise": true,
	}, &res); err != nil {
		return err
	}
	if len(res.ExceptionDetails) > 0 && string(res.ExceptionDetails) != "null" {
		return fmt.Errorf("evaluate failed: %s", string(res.ExceptionDetails))
	}
	if out == nil || len(res.Result.Value) == 0 {
		return nil
	}
	return json.Unmarshal(res.Result.Value, out)
}

func (c *CDP) Evaluate(expr string, out any) error {
	var res struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails json.RawMessage `json:"exceptionDetails"`
	}
	if err := c.Call("Runtime.evaluate", map[string]any{
		"expression": expr, "returnByValue": true, "awaitPromise": true,
	}, &res); err != nil {
		return err
	}
	if len(res.ExceptionDetails) > 0 && string(res.ExceptionDetails) != "null" {
		return fmt.Errorf("evaluate failed: %s", string(res.ExceptionDetails))
	}
	if out == nil || len(res.Result.Value) == 0 {
		return nil
	}
	return json.Unmarshal(res.Result.Value, out)
}

// HTTPJSON performs a GET against the DevTools HTTP endpoint.
func HTTPJSON(url string, out any) error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned HTTP %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// FreePort asks the OS for an unused localhost TCP port.
func FreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// WaitForCDP polls /json/version until the browser answers or the deadline
// expires. exitCheck is called between attempts to detect an early crash.
func WaitForCDP(port int, timeout time.Duration, exitCheck func() (int, bool)) (*cdpVersion, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/json/version", port)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var v cdpVersion
		if err := HTTPJSON(url, &v); err == nil && v.WebSocketDebuggerURL != "" {
			return &v, nil
		}
		if exitCheck != nil {
			if code, exited := exitCheck(); exited {
				return nil, fmt.Errorf("browser exited during startup with code %d", code)
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return nil, fmt.Errorf("DevTools endpoint on port %d did not come up within %s", port, timeout)
}

func ListTargets(port int) ([]cdpTarget, error) {
	var out []cdpTarget
	err := HTTPJSON(fmt.Sprintf("http://127.0.0.1:%d/json/list", port), &out)
	return out, err
}

func HTTPVersion(port int) (*cdpVersion, error) {
	var v cdpVersion
	err := HTTPJSON(fmt.Sprintf("http://127.0.0.1:%d/json/version", port), &v)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func PageTarget(port int) (*cdpTarget, error) {
	targets, err := ListTargets(port)
	if err != nil {
		return nil, err
	}
	for i := range targets {
		if targets[i].Type == "page" && targets[i].WebSocketDebuggerURL != "" {
			return &targets[i], nil
		}
	}
	return nil, fmt.Errorf("no page target on port %d", port)
}
