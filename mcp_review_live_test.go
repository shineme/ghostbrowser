package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Second-pass review: real MCP handshake over the real HTTP listener and over
// the real binary started with --mcp-stdio (GHOST_MCP_BIN=<go build output>).

func mcpReviewServer(t *testing.T) (*Store, string) {
	t.Helper()
	s := seededStore(t)
	port := freePort()
	writeMcpConfig(s.dir, mcpConfig{Enabled: true, Port: port})
	m := &mcpServer{}
	if err := m.start(s, false); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.stop)
	return s, fmt.Sprintf("http://127.0.0.1:%d/mcp", port)
}

func rawPost(t *testing.T, url, body string, hdr map[string]string) (int, string, http.Header) {
	t.Helper()
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range hdr {
		if k == "Host" {
			req.Host = v
		} else {
			req.Header.Set(k, v)
		}
	}
	resp, err := localHTTP.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b), resp.Header
}

func TestMcpReviewHTTPLive(t *testing.T) {
	s, url := mcpReviewServer(t)
	key := map[string]string{"X-API-Key": "gb_testkey", "MCP-Protocol-Version": "2025-06-18"}
	code, out, h := rawPost(t, url, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"review","version":"1"}}}`, key)
	if code != 200 || !strings.Contains(h.Get("Content-Type"), "application/json") || !strings.Contains(out, `"protocolVersion":"2025-06-18"`) || !strings.Contains(out, `"tools"`) {
		t.Fatalf("initialize: %d %s %v", code, out, h)
	}
	if code, out, _ := rawPost(t, url, `{"jsonrpc":"2.0","method":"notifications/initialized"}`, key); code != 202 || out != "" {
		t.Fatalf("notification: %d %q", code, out)
	}
	code, out, _ = rawPost(t, url, `{"jsonrpc":"2.0","id":"x2","method":"tools/list"}`, key)
	var tl struct {
		ID     string `json:"id"`
		Result struct {
			Tools []struct {
				Name        string         `json:"name"`
				InputSchema map[string]any `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(out), &tl) != nil || tl.ID != "x2" || len(tl.Result.Tools) < 5 {
		t.Fatalf("tools/list: %s", out)
	}
	for _, tool := range tl.Result.Tools {
		if tool.InputSchema["type"] != "object" {
			t.Errorf("tool %s inputSchema type %v", tool.Name, tool.InputSchema["type"])
		}
	}
	code, out, _ = rawPost(t, url, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_environments","arguments":{"group":"电商"}}}`, key)
	if code != 200 || !strings.Contains(out, "Shop-1") || strings.Contains(out, "Social-2") || !strings.Contains(out, `"isError":false`) {
		t.Fatalf("tools/call: %s", out)
	}
	// Tool errors are results with isError, protocol errors are JSON-RPC errors.
	_, out, _ = rawPost(t, url, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"start_environment","arguments":{"id":"nope"}}}`, key)
	if !strings.Contains(out, `"isError":true`) {
		t.Errorf("bad id should be an isError result: %s", out)
	}
	_, out, _ = rawPost(t, url, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"no_such_tool"}}`, key)
	if !strings.Contains(out, `"code":-32602`) {
		t.Errorf("unknown tool: %s", out)
	}
	_, out, _ = rawPost(t, url, `{"jsonrpc":"2.0","id":6,"method":"bogus"}`, key)
	if !strings.Contains(out, `"code":-32601`) {
		t.Errorf("unknown method: %s", out)
	}
	_, out, _ = rawPost(t, url, `{"jsonrpc":"2.0","id":7,`, key)
	if !strings.Contains(out, `"code":-32700`) {
		t.Errorf("parse error: %s", out)
	}
	// Guard: missing key, DNS-rebinding Host, browser Origin, cross-site fetch.
	body := `{"jsonrpc":"2.0","id":9,"method":"tools/list"}`
	for name, hdr := range map[string]map[string]string{
		"no key":     {},
		"bearer bad": {"Authorization": "Bearer nope"},
		"host":       {"X-API-Key": "gb_testkey", "Host": "evil.example:25326"},
		"origin":     {"X-API-Key": "gb_testkey", "Origin": "http://127.0.0.1:1234"},
		"sec-fetch":  {"X-API-Key": "gb_testkey", "Sec-Fetch-Site": "cross-site"},
	} {
		if code, _, _ := rawPost(t, url, body, hdr); code != 401 && code != 403 {
			t.Errorf("%s: got %d", name, code)
		}
	}
	if code, _, _ := rawPost(t, url, body, map[string]string{"Authorization": "Bearer gb_testkey"}); code != 200 {
		t.Errorf("bearer key refused: %d", code)
	}
	// GET (no SSE stream offered) → 405.
	greq, _ := http.NewRequest("GET", url, nil)
	greq.Header.Set("X-API-Key", "gb_testkey")
	resp, err := localHTTP.Do(greq)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode != 405 {
			t.Errorf("GET: %d", resp.StatusCode)
		}
	}
	// Audit log persisted with the calls.
	b, err := os.ReadFile(filepath.Join(s.dir, "auditlog.json"))
	if err != nil || !strings.Contains(string(b), "mcp.call") || !strings.Contains(string(b), "list_environments") || !strings.Contains(string(b), "review") {
		t.Fatalf("audit log: %v %s", err, b)
	}
	if r := s.AuditQuery(AuditFilter{Action: "mcp.call"}); r.Total != 2 || r.Failed != 1 {
		t.Errorf("audit counts %+v", r)
	}
}

func TestMcpReviewStdioBinaryLive(t *testing.T) {
	bin := os.Getenv("GHOST_MCP_BIN")
	if bin == "" {
		t.Skip("set GHOST_MCP_BIN to a `go build` of this package")
	}
	s, url := mcpReviewServer(t)
	// The stdio process reads %APPDATA%/GhostBrowser (XDG_CONFIG_HOME on Linux).
	cfgHome := t.TempDir()
	dir := filepath.Join(cfgHome, "GhostBrowser")
	_ = os.MkdirAll(dir, 0o755)
	data, _ := json.Marshal(map[string]any{"settings": map[string]any{"apiKey": "gb_testkey"}})
	_ = os.WriteFile(filepath.Join(dir, "data.json"), data, 0o600)
	var port int
	fmt.Sscanf(url, "http://127.0.0.1:%d/mcp", &port)
	writeMcpConfig(dir, mcpConfig{Enabled: true, Port: port})
	_ = s

	cmd := exec.Command(bin, "--mcp-stdio")
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+cfgHome, "HOME="+cfgHome)
	in, _ := cmd.StdinPipe()
	outp, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { in.Close(); cmd.Process.Kill(); cmd.Wait() }()
	rd := bufio.NewReader(outp)
	send := func(msg string) string {
		t.Helper()
		if _, err := io.WriteString(in, msg+"\n"); err != nil {
			t.Fatal(err)
		}
		ch := make(chan string, 1)
		go func() { l, _ := rd.ReadString('\n'); ch <- l }()
		select {
		case l := <-ch:
			if strings.Count(strings.TrimRight(l, "\n"), "\n") != 0 {
				t.Fatalf("multi-line stdio message %q", l)
			}
			return l
		case <-time.After(10 * time.Second):
			t.Fatalf("no reply to %s", msg)
		}
		return ""
	}
	if l := send(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"claude-desktop","version":"1"}}}`); !strings.Contains(l, `"protocolVersion":"2024-11-05"`) {
		t.Fatalf("stdio initialize: %s", l)
	}
	_, _ = io.WriteString(in, `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n")
	if l := send(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_environments","arguments":{}}}`); !strings.Contains(l, "Shop-1") || !strings.Contains(l, `"id":2`) {
		t.Fatalf("stdio tools/call: %s", l)
	}
	if l := send(`{"jsonrpc":"2.0","id":3,"method":"ping"}`); !strings.Contains(l, `"id":3`) || !strings.Contains(l, `"result":{}`) {
		t.Fatalf("stdio ping: %s", l)
	}
}
