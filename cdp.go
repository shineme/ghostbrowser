package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// localHTTP never goes through a system/env proxy when talking to 127.0.0.1.
var localHTTP = &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{Proxy: nil}}

func waitDebugger(port int, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := localHTTP.Get(fmt.Sprintf("http://127.0.0.1:%d/json/version", port))
		if err == nil {
			var v struct {
				WS string `json:"webSocketDebuggerUrl"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&v)
			resp.Body.Close()
			if v.WS != "" {
				return v.WS, nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return "", fmt.Errorf("debugger not ready")
}

func cdpCall(ws, method string, params any) (json.RawMessage, error) {
	c, _, err := websocket.DefaultDialer.Dial(ws, nil)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	_ = c.SetReadDeadline(time.Now().Add(15 * time.Second))
	if err := c.WriteJSON(map[string]any{"id": 1, "method": method, "params": params}); err != nil {
		return nil, err
	}
	for {
		var m struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := c.ReadJSON(&m); err != nil {
			return nil, err
		}
		if m.ID == 1 {
			if m.Error != nil {
				return nil, fmt.Errorf("%s", m.Error.Message)
			}
			return m.Result, nil
		}
	}
}

// importCookies accepts the common JSON export format (EditThisCookie / Cookie-Editor / Undetectable).
func importCookies(ws, raw string) error {
	var in []map[string]any
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		return fmt.Errorf("cookies must be a JSON array: %w", err)
	}
	out := []map[string]any{}
	for _, c := range in {
		ck := map[string]any{"name": c["name"], "value": c["value"], "domain": c["domain"], "path": "/"}
		if v, ok := c["path"]; ok {
			ck["path"] = v
		}
		if v, ok := c["secure"]; ok {
			ck["secure"] = v
		}
		if v, ok := c["httpOnly"]; ok {
			ck["httpOnly"] = v
		}
		for _, k := range []string{"expirationDate", "expires"} {
			if v, ok := c[k].(float64); ok && v > 0 {
				ck["expires"] = v
			}
		}
		if v, ok := c["sameSite"].(string); ok {
			switch strings.ToLower(v) {
			case "lax":
				ck["sameSite"] = "Lax"
			case "strict":
				ck["sameSite"] = "Strict"
			case "none", "no_restriction":
				ck["sameSite"] = "None"
			}
		}
		out = append(out, ck)
	}
	_, err := cdpCall(ws, "Storage.setCookies", map[string]any{"cookies": out})
	return err
}

func exportCookies(ws string) (string, error) {
	r, err := cdpCall(ws, "Storage.getCookies", map[string]any{})
	if err != nil {
		return "", err
	}
	var v struct {
		Cookies json.RawMessage `json:"cookies"`
	}
	_ = json.Unmarshal(r, &v)
	return string(v.Cookies), nil
}

func closeBrowser(ws string) error {
	_, err := cdpCall(ws, "Browser.close", map[string]any{})
	return err
}
