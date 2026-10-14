package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func mcpCall(t *testing.T, c *mcpCore, msg string) map[string]any {
	t.Helper()
	out := c.handle([]byte(msg), "mcp")
	var r map[string]any
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("bad reply %q: %v", out, err)
	}
	if r["error"] != nil {
		t.Fatalf("rpc error: %v", r["error"])
	}
	return r["result"].(map[string]any)
}

func seededStore(t *testing.T) *Store {
	s := testStore(t)
	s.Profiles["a1"] = &Profile{ID: "a1", Name: "Shop-1", Group: "电商", Tags: []string{"美区"}}
	s.Profiles["b2"] = &Profile{ID: "b2", Name: "Social-2", Group: "社媒"}
	return s
}

func TestMcpInitializeListCall(t *testing.T) {
	s := seededStore(t)
	c := newMcpCore(s)
	r := mcpCall(t, c, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","clientInfo":{"name":"tester"}}}`)
	if r["protocolVersion"] != "2025-06-18" {
		t.Fatalf("proto %v", r["protocolVersion"])
	}
	if c.handle([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`), "mcp") != nil {
		t.Fatal("notification got a reply")
	}
	r = mcpCall(t, c, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	tools := r["tools"].([]any)
	found := false
	for _, x := range tools {
		if x.(map[string]any)["name"] == "list_environments" {
			found = true
		}
	}
	if !found || len(tools) < 3 {
		t.Fatalf("tools: %d found=%v", len(tools), found)
	}
	r = mcpCall(t, c, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_environments","arguments":{}}}`)
	if r["isError"] != false {
		t.Fatalf("isError: %v", r)
	}
	text := r["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "Shop-1") || !strings.Contains(text, "Social-2") {
		t.Fatalf("missing envs: %s", text)
	}
	r = mcpCall(t, c, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"list_environments","arguments":{"group":"电商"}}}`)
	text = r["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "Shop-1") || strings.Contains(text, "Social-2") {
		t.Fatalf("group filter: %s", text)
	}
	if n := s.AuditQuery(AuditFilter{Action: "mcp.call"}).Total; n != 2 {
		t.Fatalf("audited mcp calls = %d", n)
	}
	// unknown method
	var e map[string]any
	_ = json.Unmarshal(c.handle([]byte(`{"jsonrpc":"2.0","id":5,"method":"nope"}`), "mcp"), &e)
	if e["error"] == nil {
		t.Fatal("expected method-not-found")
	}
}

func TestMcpHTTPGuard(t *testing.T) {
	s := seededStore(t)
	m := &mcpServer{store: s, core: newMcpCore(s)}
	srv := httptest.NewServer(m.handler())
	defer srv.Close()
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_environments","arguments":{}}}`
	post := func(key string) (*http.Response, string) {
		req, _ := http.NewRequest("POST", srv.URL+"/mcp", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if key != "" {
			req.Header.Set("X-API-Key", key)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var sb strings.Builder
		buf := make([]byte, 4096)
		for {
			n, e := resp.Body.Read(buf)
			sb.Write(buf[:n])
			if e != nil {
				break
			}
		}
		return resp, sb.String()
	}
	if resp, _ := post(""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no key: %d", resp.StatusCode)
	}
	if resp, _ := post("wrong"); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong key: %d", resp.StatusCode)
	}
	resp, out := post("gb_testkey")
	if resp.StatusCode != 200 || !strings.Contains(out, "Shop-1") {
		t.Fatalf("with key: %d %s", resp.StatusCode, out)
	}
	s.Settings.APIKeyOff = true
	if resp, out := post(""); resp.StatusCode != 200 || !strings.Contains(out, "Shop-1") {
		t.Fatalf("key off: %d %s", resp.StatusCode, out)
	}
	// browser Origin is always refused
	req, _ := http.NewRequest("POST", srv.URL+"/mcp", strings.NewReader(body))
	req.Header.Set("Origin", "https://evil.example")
	r2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r2.Body.Close()
	if r2.StatusCode != http.StatusForbidden {
		t.Fatalf("origin: %d", r2.StatusCode)
	}
}

func TestLocalAPIGroupsTagsKernels(t *testing.T) {
	s := seededStore(t)
	a := &localAPI{store: s}
	mux := http.NewServeMux()
	mux.HandleFunc("/groups", a.handleGroups)
	mux.HandleFunc("/group", a.handleCreateGroup)
	mux.HandleFunc("/tags", a.handleTags)
	mux.HandleFunc("/kernels", a.handleKernels)
	srv := httptest.NewServer(a.guard(mux))
	defer srv.Close()
	get := func(method, path, body string) map[string]any {
		req, _ := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
		req.Header.Set("X-API-Key", "gb_testkey")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var m map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&m)
		return m
	}
	if m := get("POST", "/group", `{"name":"新分组"}`); m["status"] != "ok" {
		t.Fatalf("create group: %v", m)
	}
	m := get("GET", "/groups", "")
	if b, _ := json.Marshal(m); !strings.Contains(string(b), "新分组") || !strings.Contains(string(b), "电商") {
		t.Fatalf("groups: %s", b)
	}
	m = get("GET", "/tags", "")
	if b, _ := json.Marshal(m); !strings.Contains(string(b), "美区") {
		t.Fatalf("tags: %s", b)
	}
	if m = get("GET", "/kernels", ""); m["status"] != "ok" {
		t.Fatalf("kernels: %v", m)
	}
}
