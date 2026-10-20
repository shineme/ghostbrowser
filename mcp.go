package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MCP (Model Context Protocol) server: lets AI clients such as Claude, Cursor
// or Claude Code list / create / start / stop profiles, manage proxies, groups
// and tags.
//
// Transports:
//   - Streamable HTTP on 127.0.0.1:<port>/mcp (POST JSON-RPC, JSON replies),
//     protected by the same guard as the local API (loopback Host, no browser
//     Origin, X-API-Key / Bearer = Settings.apiKey).
//   - stdio: `GhostBrowser.exe --mcp-stdio` reads newline-delimited JSON-RPC
//     on stdin and relays each message to the running app's HTTP endpoint, so
//     clients that only spawn commands (Claude Desktop) drive the same
//     process that owns the browsers.

const (
	mcpDefaultPort  = 25326
	mcpServerName   = "ghostbrowser"
	mcpLatestProto  = "2025-11-25"
	mcpMaxBody      = 4 << 20
	mcpStdioTimeout = 120 * time.Second
)

var mcpProtocols = []string{"2025-11-25", "2025-06-18", "2025-03-26", "2024-11-05"}

// ---------------------------------------------------------------- JSON-RPC

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

const (
	rpcParseError     = -32700
	rpcInvalidRequest = -32600
	rpcMethodNotFound = -32601
	rpcInvalidParams  = -32602
	rpcInternal       = -32603
)

// mcpCore answers MCP messages against a Store; transports wrap it.
type mcpCore struct {
	store    *Store
	calls    atomic.Int64
	lastCall atomic.Int64 // unix ms
	client   atomic.Value // string: last clientInfo.name
}

func newMcpCore(s *Store) *mcpCore { return &mcpCore{store: s} }

// handle takes one raw message (an object or a batch array) and returns the
// raw reply, or nil when nothing is owed (notifications only).
func (c *mcpCore) handle(raw []byte, source string) []byte {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil
	}
	if raw[0] == '[' {
		var batch []json.RawMessage
		if json.Unmarshal(raw, &batch) != nil || len(batch) == 0 {
			b, _ := json.Marshal(rpcResponse{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{Code: rpcInvalidRequest, Message: "invalid batch"}})
			return b
		}
		var out []json.RawMessage
		for _, m := range batch {
			if r := c.handleOne(m, source); r != nil {
				out = append(out, r)
			}
		}
		if len(out) == 0 {
			return nil
		}
		b, _ := json.Marshal(out)
		return b
	}
	return c.handleOne(raw, source)
}

func (c *mcpCore) handleOne(raw []byte, source string) []byte {
	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		b, _ := json.Marshal(rpcResponse{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{Code: rpcParseError, Message: "parse error: " + err.Error()}})
		return b
	}
	isNotification := len(req.ID) == 0
	if req.Method == "" {
		if isNotification { // a response from the client to a server request; we send none
			return nil
		}
		b, _ := json.Marshal(rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: rpcInvalidRequest, Message: "method is required"}})
		return b
	}
	result, rerr := c.dispatch(req, source)
	if isNotification {
		return nil
	}
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	if rerr != nil {
		resp.Error = rerr
	} else {
		if result == nil {
			result = map[string]any{}
		}
		resp.Result = result
	}
	b, err := json.Marshal(resp)
	if err != nil {
		b, _ = json.Marshal(rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: rpcInternal, Message: err.Error()}})
	}
	return b
}

func (c *mcpCore) dispatch(req rpcRequest, source string) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
			ClientInfo      struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"clientInfo"`
		}
		_ = json.Unmarshal(req.Params, &p)
		ver := mcpLatestProto
		for _, v := range mcpProtocols {
			if v == p.ProtocolVersion {
				ver = v
			}
		}
		if p.ClientInfo.Name != "" {
			c.client.Store(p.ClientInfo.Name)
		}
		return map[string]any{
			"protocolVersion": ver,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": mcpServerName, "title": "GhostBrowser", "version": mcpAppVersion()},
			"instructions": "GhostBrowser manages antidetect browser profiles (called environments). " +
				"Use list_environments to find IDs, start_environment returns a CDP websocket for Puppeteer/Playwright. " +
				"Nothing here deletes data.",
		}, nil
	case "notifications/initialized", "notifications/cancelled", "notifications/progress", "notifications/roots/list_changed":
		return nil, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		tools := make([]map[string]any, 0, len(mcpTools))
		for _, t := range mcpTools {
			tools = append(tools, t.describe())
		}
		return map[string]any{"tools": tools}, nil
	case "tools/call":
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil, &rpcError{Code: rpcInvalidParams, Message: "invalid params: " + err.Error()}
		}
		t := findMcpTool(p.Name)
		if t == nil {
			return nil, &rpcError{Code: rpcInvalidParams, Message: "unknown tool: " + p.Name}
		}
		if p.Arguments == nil {
			p.Arguments = map[string]any{}
		}
		return c.callTool(t, p.Arguments, source), nil
	case "resources/list":
		return map[string]any{"resources": []any{}}, nil
	case "prompts/list":
		return map[string]any{"prompts": []any{}}, nil
	}
	return nil, &rpcError{Code: rpcMethodNotFound, Message: "method not found: " + req.Method}
}

// callTool runs one tool. Tool failures come back as isError results (per the
// MCP spec) so the model can read and correct them.
func (c *mcpCore) callTool(t *mcpTool, args map[string]any, source string) map[string]any {
	c.calls.Add(1)
	c.lastCall.Store(time.Now().UnixMilli())
	out, err := func() (out any, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("internal error: %v", r)
			}
		}()
		return t.Run(c, args)
	}()
	argJSON, _ := json.Marshal(args)
	detail := t.Name
	if len(argJSON) > 2 {
		detail += " " + string(argJSON)
	}
	if client, _ := c.client.Load().(string); client != "" {
		detail += " · " + client
	}
	c.store.auditNamed(source, "mcp.call", "", t.Name, err, detail)
	if err != nil {
		return map[string]any{"content": []any{map[string]any{"type": "text", "text": "Error: " + err.Error()}}, "isError": true}
	}
	text, _ := json.MarshalIndent(out, "", "  ")
	res := map[string]any{"content": []any{map[string]any{"type": "text", "text": string(text)}}, "isError": false}
	// structuredContent must be an object.
	if b, e := json.Marshal(out); e == nil && len(b) > 0 && b[0] == '{' {
		res["structuredContent"] = json.RawMessage(b)
	}
	return res
}

// ---------------------------------------------------------------- HTTP transport

type mcpConfig struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
}

type mcpServer struct {
	mu      sync.Mutex
	store   *Store
	core    *mcpCore
	srv     *http.Server
	port    int
	err     string
	started int64
}

var mcpSrv = &mcpServer{}

func mcpConfigFile(dir string) string { return filepath.Join(dir, "mcp.json") }

func readMcpConfig(dir string) mcpConfig {
	c := mcpConfig{Port: mcpDefaultPort}
	if b, e := os.ReadFile(mcpConfigFile(dir)); e == nil {
		_ = json.Unmarshal(b, &c)
	}
	if c.Port <= 0 || c.Port > 65535 {
		c.Port = mcpDefaultPort
	}
	return c
}

func writeMcpConfig(dir string, c mcpConfig) {
	b, _ := json.MarshalIndent(c, "", "  ")
	_ = os.WriteFile(mcpConfigFile(dir), b, 0o600)
}

func (m *mcpServer) handler() http.Handler {
	mux := http.NewServeMux()
	serve := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
		case http.MethodGet, http.MethodDelete:
			// No server-initiated stream and no sessions: allowed by the spec.
			w.Header().Set("Allow", "POST")
			http.Error(w, "use POST", http.StatusMethodNotAllowed)
			return
		default:
			w.Header().Set("Allow", "POST")
			http.Error(w, "use POST", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, mcpMaxBody))
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		out := m.core.handle(body, "mcp")
		if out == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(out)
	}
	mux.HandleFunc("/mcp", serve)
	mux.HandleFunc("/mcp/", serve)
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "name": mcpServerName, "version": mcpAppVersion()})
	})
	return (&localAPI{store: m.store}).guard(mux)
}

// start listens when mcp.json says enabled (or force is set).
func (m *mcpServer) start(s *Store, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store = s
	if m.core == nil || m.core.store != s {
		m.core = newMcpCore(s)
	}
	cfg := readMcpConfig(s.dir)
	if !cfg.Enabled && !force {
		return nil
	}
	if m.srv != nil {
		return nil
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", cfg.Port))
	if err != nil {
		m.err = fmt.Sprintf("端口 %d 无法监听：%v", cfg.Port, err)
		return errors.New(m.err)
	}
	m.err, m.port, m.started = "", cfg.Port, time.Now().Unix()
	m.srv = &http.Server{Handler: m.handler(), ReadHeaderTimeout: 10 * time.Second}
	srv := m.srv
	go func() { _ = srv.Serve(ln) }()
	return nil
}

func (m *mcpServer) stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.srv != nil {
		_ = m.srv.Close()
		m.srv = nil
	}
	m.started = 0
}

// McpStatus is what the UI shows and what the user copies into a client.
type McpStatus struct {
	Running      bool     `json:"running"`
	Enabled      bool     `json:"enabled"`
	Port         int      `json:"port"`
	URL          string   `json:"url"`
	Error        string   `json:"error"`
	KeyRequired  bool     `json:"keyRequired"`
	Tools        []string `json:"tools"`
	Calls        int64    `json:"calls"`
	LastCall     int64    `json:"lastCall"` // unix ms
	LastClient   string   `json:"lastClient"`
	StartedAt    int64    `json:"startedAt"`
	ExePath      string   `json:"exePath"`
	Protocol     string   `json:"protocol"`
	ConfigHTTP   string   `json:"configHttp"`   // Cursor / Windsurf style mcpServers JSON (url + header)
	ConfigStdio  string   `json:"configStdio"`  // Claude Desktop style mcpServers JSON (command + args)
	ClaudeCodeCL string   `json:"claudeCodeCl"` // claude mcp add ... command line
}

func (m *mcpServer) status(s *Store) McpStatus {
	m.mu.Lock()
	running, port, errS, started, core := m.srv != nil, m.port, m.err, m.started, m.core
	m.mu.Unlock()
	cfg := readMcpConfig(s.dir)
	if !running {
		port = cfg.Port
	}
	st := s.GetSettings()
	out := McpStatus{Running: running, Enabled: cfg.Enabled, Port: port, Error: errS, KeyRequired: !st.APIKeyOff,
		StartedAt: started, Protocol: mcpLatestProto, URL: fmt.Sprintf("http://127.0.0.1:%d/mcp", port)}
	for _, t := range mcpTools {
		out.Tools = append(out.Tools, t.Name)
	}
	if core != nil {
		out.Calls, out.LastCall = core.calls.Load(), core.lastCall.Load()
		out.LastClient, _ = core.client.Load().(string)
	}
	exe, _ := os.Executable()
	if exe == "" {
		exe = "GhostBrowser.exe"
	}
	out.ExePath = exe
	httpEntry := map[string]any{"url": out.URL}
	if out.KeyRequired {
		httpEntry["headers"] = map[string]string{"X-API-Key": st.APIKey}
	}
	b, _ := json.MarshalIndent(map[string]any{"mcpServers": map[string]any{mcpServerName: httpEntry}}, "", "  ")
	out.ConfigHTTP = string(b)
	b, _ = json.MarshalIndent(map[string]any{"mcpServers": map[string]any{mcpServerName: map[string]any{"command": exe, "args": []string{"--mcp-stdio"}}}}, "", "  ")
	out.ConfigStdio = string(b)
	out.ClaudeCodeCL = "claude mcp add --transport http " + mcpServerName + " " + out.URL
	if out.KeyRequired {
		out.ClaudeCodeCL += ` --header "X-API-Key: ` + st.APIKey + `"`
	}
	return out
}

// ---------------------------------------------------------------- stdio transport

// serveMcpStdio reads newline-delimited JSON-RPC from r, passes each message
// to handle and writes every reply as one line to w.
func serveMcpStdio(r io.Reader, w io.Writer, handle func([]byte) []byte) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), mcpMaxBody)
	var wmu sync.Mutex
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		out := handle(append([]byte(nil), line...))
		if out == nil {
			continue
		}
		var buf bytes.Buffer
		if json.Compact(&buf, out) != nil { // stdio messages must not contain newlines
			buf.Reset()
			buf.Write(bytes.ReplaceAll(out, []byte("\n"), nil))
		}
		buf.WriteByte('\n')
		wmu.Lock()
		_, err := w.Write(buf.Bytes())
		wmu.Unlock()
		if err != nil {
			return err
		}
	}
	return sc.Err()
}

// mcpBridge relays stdio messages to the running app's HTTP endpoint.
func mcpBridge(url, key string) func([]byte) []byte {
	cl := &http.Client{Timeout: mcpStdioTimeout}
	return func(msg []byte) []byte {
		req, _ := http.NewRequest("POST", url, bytes.NewReader(msg))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if key != "" {
			req.Header.Set("X-API-Key", key)
		}
		resp, err := cl.Do(req)
		if err != nil {
			return bridgeError(msg, "GhostBrowser 没有运行，或 MCP 服务未开启（在 GhostBrowser「接口与审计」页开启）: "+err.Error())
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, mcpMaxBody))
		if resp.StatusCode == http.StatusAccepted || len(bytes.TrimSpace(body)) == 0 {
			return nil
		}
		if resp.StatusCode != http.StatusOK {
			return bridgeError(msg, fmt.Sprintf("GhostBrowser MCP HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body))))
		}
		return body
	}
}

// bridgeError answers a request (not a notification) with a JSON-RPC error.
func bridgeError(msg []byte, text string) []byte {
	var req rpcRequest
	if json.Unmarshal(msg, &req) != nil || len(req.ID) == 0 {
		return nil
	}
	b, _ := json.Marshal(rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: rpcInternal, Message: text}})
	return b
}

// runMcpStdio is the `--mcp-stdio` entry point. It only reads data.json and
// mcp.json (never writes them), since the GUI process owns those files.
func runMcpStdio() int {
	dir := appDataDir()
	var d struct {
		Settings Settings `json:"settings"`
	}
	if b, e := os.ReadFile(filepath.Join(dir, "data.json")); e == nil {
		_ = json.Unmarshal(b, &d)
	}
	cfg := readMcpConfig(dir)
	key := ""
	if !d.Settings.APIKeyOff {
		key = d.Settings.APIKey
	}
	err := serveMcpStdio(os.Stdin, os.Stdout, mcpBridge(fmt.Sprintf("http://127.0.0.1:%d/mcp", cfg.Port), key))
	if err != nil {
		fmt.Fprintln(os.Stderr, "ghostbrowser mcp:", err)
		return 1
	}
	return 0
}

func mcpAppVersion() string { return "0.8.7" }
