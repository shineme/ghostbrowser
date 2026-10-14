package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func fakeMihomo(t *testing.T, secret string) (*httptest.Server, *string) {
	selected := "HK-01"
	mux := http.NewServeMux()
	auth := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+secret {
				w.WriteHeader(401)
				io.WriteString(w, `{"message":"Unauthorized"}`)
				return
			}
			h(w, r)
		}
	}
	mux.HandleFunc("/version", auth(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"meta":true,"version":"v1.19.0"}`)
	}))
	mux.HandleFunc("/configs", auth(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"port":7890,"mixed-port":7897,"mode":"rule","allow-lan":false}`)
	}))
	mux.HandleFunc("/proxies", auth(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"proxies": map[string]any{
			"GLOBAL": map[string]any{"name": "GLOBAL", "type": "Selector", "now": "Proxy", "all": []string{"Proxy", "HK-01", "JP-02", "DIRECT"}},
			"Proxy":  map[string]any{"name": "Proxy", "type": "Selector", "now": selected, "all": []string{"HK-01", "JP-02"}},
			"HK-01":  map[string]any{"name": "HK-01", "type": "Shadowsocks", "udp": true, "alive": true, "history": []map[string]any{{"delay": 120}, {"delay": 88}}},
			"JP-02":  map[string]any{"name": "JP-02", "type": "Vmess", "alive": false, "history": []any{}},
			"DIRECT": map[string]any{"name": "DIRECT", "type": "Direct", "udp": true},
		}})
	}))
	mux.HandleFunc("/proxies/", auth(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/proxies/")
		if strings.HasSuffix(name, "/delay") {
			name = strings.TrimSuffix(name, "/delay")
			if r.URL.Query().Get("url") == "" || r.URL.Query().Get("timeout") == "" {
				w.WriteHeader(400)
				return
			}
			if name == "JP-02" {
				w.WriteHeader(504)
				io.WriteString(w, `{"message":"Timeout"}`)
				return
			}
			io.WriteString(w, `{"delay":77}`)
			return
		}
		if r.Method != "PUT" || name != "Proxy" {
			w.WriteHeader(404)
			return
		}
		var body struct{ Name string }
		json.NewDecoder(r.Body).Decode(&body)
		if body.Name != "HK-01" && body.Name != "JP-02" {
			w.WriteHeader(400)
			io.WriteString(w, `{"message":"Selector update error: proxy not exist"}`)
			return
		}
		selected = body.Name
		w.WriteHeader(204)
	}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &selected
}

func TestMihomoController(t *testing.T) {
	old := mihomoCfg.path
	mihomoCfg.path = filepath.Join(t.TempDir(), "mihomo.json")
	defer func() { mihomoCfg.path = old }()
	srv, selected := fakeMihomo(t, "s3cret")

	if mihomoListenerURL() != nil {
		t.Fatal("listener before connect")
	}
	if _, err := GetMihomoProxies(); err == nil {
		t.Fatal("expected not-configured error")
	}
	if _, err := MihomoConnect(srv.URL, "wrong"); err == nil || !strings.Contains(err.Error(), "密钥") {
		t.Fatalf("wrong secret: %v", err)
	}
	st, err := MihomoConnect(strings.TrimPrefix(srv.URL, "http://"), "s3cret")
	if err != nil || !st.Connected || st.Version != "v1.19.0" || !st.Meta || st.ListenPort != 7897 || st.ListenKind != "mixed" || st.ListenHost != "127.0.0.1" || st.Mode != "rule" {
		t.Fatalf("connect %+v %v", st, err)
	}
	if u := mihomoListenerURL(); u == nil || u.String() != "http://127.0.0.1:7897" {
		t.Fatalf("listener %v", u)
	}
	if st := GetMihomoStatus(); !st.Connected || !st.HasSecret || st.Controller != srv.URL {
		t.Fatalf("status %+v", st)
	}
	px, err := GetMihomoProxies()
	if err != nil || len(px.Groups) != 2 || px.Groups[0].Name != "Proxy" || px.Groups[1].Name != "GLOBAL" || px.Groups[0].Now != "HK-01" {
		t.Fatalf("groups %+v %v", px.Groups, err)
	}
	if len(px.Nodes) != 3 || px.Nodes[0].Name != "HK-01" || px.Nodes[0].Delay != 88 || !px.Nodes[0].Alive || px.Nodes[1].Alive || px.Nodes[2].Name != "DIRECT" {
		t.Fatalf("nodes %+v", px.Nodes)
	}
	if err := MihomoSelect("Proxy", "JP-02"); err != nil || *selected != "JP-02" {
		t.Fatalf("select %v %s", err, *selected)
	}
	if err := MihomoSelect("Proxy", "nope"); err == nil || !strings.Contains(err.Error(), "proxy not exist") {
		t.Fatalf("select bad %v", err)
	}
	if d, err := MihomoTestDelay("HK-01", "", 0); err != nil || d != 77 {
		t.Fatalf("delay %d %v", d, err)
	}
	if _, err := MihomoTestDelay("JP-02", "", 1000); err == nil || !strings.Contains(err.Error(), "超时") {
		t.Fatalf("timeout %v", err)
	}
	s := testStoreForCookies(t)
	p1, err := s.MihomoAddProxy()
	if err != nil || p1.Type != "http" || p1.Host != "127.0.0.1" || p1.Port != 7897 || p1.Tag != "mihomo" {
		t.Fatalf("add proxy %+v %v", p1, err)
	}
	if p2, _ := s.MihomoAddProxy(); p2.ID != p1.ID || len(s.ListProxies()) != 1 {
		t.Fatal("MihomoAddProxy should reuse the existing proxy")
	}
}

func TestMihomoNormalizeController(t *testing.T) {
	for in, want := range map[string]string{"9090": "http://127.0.0.1:9090", "127.0.0.1:9090/": "http://127.0.0.1:9090", "https://box.lan:9090/ui": "https://box.lan:9090/ui"} {
		if got, err := normalizeController(in); err != nil || got != want {
			t.Fatalf("%s -> %s %v", in, got, err)
		}
	}
	if _, err := normalizeController(""); err == nil {
		t.Fatal("expected error")
	}
	if h, p, k := listenerFrom("http://192.168.1.5:9090", mihomoConfigs{Port: 7890}); h != "192.168.1.5" || p != 7890 || k != "http" {
		t.Fatalf("lan listener %s %d %s", h, p, k)
	}
}
