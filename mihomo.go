package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Mihomo / Clash external controller client (RESTful API on e.g.
// 127.0.0.1:9090, "Authorization: Bearer <secret>").
//
// The controller address and secret live in appDataDir()/mihomo.json, not in
// Settings, so they never travel with cloud sync. The local mixed/http port the
// core listens on (read from GET /configs) becomes mihomoListenerURL(), used by
// the Extension Center as a download fallback and by MihomoAddProxy, which
// creates an ordinary http Proxy pointing at it so profiles can use the node
// selected in Mihomo.

type MihomoConfig struct {
	Controller string `json:"controller"` // http://127.0.0.1:9090
	Secret     string `json:"secret"`
	// Last seen listener, cached so mihomoListenerURL() works without a round trip.
	ListenHost string `json:"listenHost,omitempty"`
	ListenPort int    `json:"listenPort,omitempty"`
	ListenKind string `json:"listenKind,omitempty"` // mixed | http
	SavedAt    int64  `json:"savedAt,omitempty"`
}

type MihomoStatus struct {
	Configured bool   `json:"configured"`
	Connected  bool   `json:"connected"`
	Controller string `json:"controller"`
	HasSecret  bool   `json:"hasSecret"`
	Version    string `json:"version"`
	Meta       bool   `json:"meta"`
	Mode       string `json:"mode"`       // rule | global | direct
	ListenHost string `json:"listenHost"` // host the mixed/http port is reachable on
	ListenPort int    `json:"listenPort"` // 0 = no mixed-port / port configured
	ListenKind string `json:"listenKind"` // mixed | http | ""
	Error      string `json:"error"`
}

type MihomoNode struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`          // Shadowsocks, Vmess, Selector, URLTest, Direct...
	Now     string   `json:"now,omitempty"` // groups: selected member
	All     []string `json:"all,omitempty"` // groups: members
	Delay   int      `json:"delay"`         // last history delay ms, 0 = unknown / timeout
	Alive   bool     `json:"alive"`
	UDP     bool     `json:"udp"`
	IsGroup bool     `json:"isGroup"`
}

type MihomoProxies struct {
	Groups []MihomoNode `json:"groups"` // Selector / URLTest / Fallback / LoadBalance (GLOBAL last)
	Nodes  []MihomoNode `json:"nodes"`  // everything that is not a group
}

type mihomoStore struct {
	mu   sync.Mutex
	path string // "" = appDataDir()/mihomo.json
}

var mihomoCfg = &mihomoStore{}

func (m *mihomoStore) file() string {
	if m.path != "" {
		return m.path
	}
	return filepath.Join(appDataDir(), "mihomo.json")
}

func (m *mihomoStore) load() MihomoConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	var c MihomoConfig
	if b, err := os.ReadFile(m.file()); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	return c
}

func (m *mihomoStore) save(c MihomoConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c.SavedAt = time.Now().Unix()
	b, _ := json.MarshalIndent(c, "", "  ")
	tmp := m.file() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, m.file())
}

// normalizeController accepts "9090", "127.0.0.1:9090" or a full URL.
func normalizeController(s string) (string, error) {
	s = strings.TrimRight(strings.TrimSpace(s), "/")
	if s == "" {
		return "", fmt.Errorf("请填写 Mihomo 控制器地址，例如 127.0.0.1:9090")
	}
	if _, err := strconv.Atoi(s); err == nil {
		s = "127.0.0.1:" + s
	}
	if !strings.Contains(s, "://") {
		s = "http://" + s
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("控制器地址格式不对：%s", s)
	}
	return u.Scheme + "://" + u.Host + strings.TrimRight(u.Path, "/"), nil
}

type mihomoClient struct {
	base   string
	secret string
	hc     *http.Client
}

func newMihomoClient(controller, secret string) *mihomoClient {
	// Never route controller calls through a system proxy (it is usually Mihomo itself).
	return &mihomoClient{base: controller, secret: secret, hc: &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{Proxy: nil}}}
}

func (c *mihomoClient) do(method, path string, body any, out any) error {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, rd)
	if err != nil {
		return err
	}
	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("连不上 Mihomo 控制器（%s）：请确认 Clash / Mihomo 已运行并开启 external-controller", c.base)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("Mihomo 拒绝访问（HTTP %d）：密钥（secret）不正确", resp.StatusCode)
	}
	if resp.StatusCode >= 300 {
		var e struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(b, &e)
		if e.Message == "" {
			e.Message = strings.TrimSpace(string(b))
		}
		return fmt.Errorf("Mihomo 返回 HTTP %d：%s", resp.StatusCode, e.Message)
	}
	if out != nil && len(b) > 0 {
		if err := json.Unmarshal(b, out); err != nil {
			return fmt.Errorf("Mihomo 返回内容无法解析：%w", err)
		}
	}
	return nil
}

func (c *mihomoClient) version() (string, bool, error) {
	var v struct {
		Version string `json:"version"`
		Meta    bool   `json:"meta"`
	}
	err := c.do("GET", "/version", nil, &v)
	return v.Version, v.Meta, err
}

type mihomoConfigs struct {
	Port        int    `json:"port"`
	MixedPort   int    `json:"mixed-port"`
	Mode        string `json:"mode"`
	AllowLan    bool   `json:"allow-lan"`
	BindAddress string `json:"bind-address"`
}

func (c *mihomoClient) configs() (mihomoConfigs, error) {
	var v mihomoConfigs
	err := c.do("GET", "/configs", nil, &v)
	return v, err
}

// listener picks the port profiles should use: mixed-port first, then the http port.
// The host is the controller's host when it is loopback, else that host (LAN core).
func listenerFrom(controller string, cf mihomoConfigs) (string, int, string) {
	host := "127.0.0.1"
	if u, err := url.Parse(controller); err == nil {
		if h := u.Hostname(); h != "" && h != "localhost" {
			if ip := net.ParseIP(h); ip == nil || !ip.IsLoopback() {
				host = h
			}
		}
	}
	if cf.MixedPort > 0 {
		return host, cf.MixedPort, "mixed"
	}
	if cf.Port > 0 {
		return host, cf.Port, "http"
	}
	return host, 0, ""
}

// mihomoListenerURL returns the cached Mihomo mixed/http listener as an HTTP
// proxy URL, or nil when Mihomo was never connected.
func mihomoListenerURL() *url.URL {
	c := mihomoCfg.load()
	if c.Controller == "" || c.ListenPort <= 0 {
		return nil
	}
	h := c.ListenHost
	if h == "" {
		h = "127.0.0.1"
	}
	return &url.URL{Scheme: "http", Host: net.JoinHostPort(h, strconv.Itoa(c.ListenPort))}
}

func mihomoFromConfig() (*mihomoClient, MihomoConfig, error) {
	c := mihomoCfg.load()
	if c.Controller == "" {
		return nil, c, fmt.Errorf("还没有连接 Mihomo：请先填写控制器地址")
	}
	return newMihomoClient(c.Controller, c.Secret), c, nil
}

// MihomoConnect verifies the controller, stores it with the secret and caches the listener.
func MihomoConnect(controller, secret string) (MihomoStatus, error) {
	ctl, err := normalizeController(controller)
	if err != nil {
		return MihomoStatus{}, err
	}
	secret = strings.TrimSpace(secret)
	cl := newMihomoClient(ctl, secret)
	st, err := mihomoProbe(cl, MihomoConfig{Controller: ctl, Secret: secret})
	if err != nil {
		return st, err
	}
	cfg := MihomoConfig{Controller: ctl, Secret: secret, ListenHost: st.ListenHost, ListenPort: st.ListenPort, ListenKind: st.ListenKind}
	if err := mihomoCfg.save(cfg); err != nil {
		return st, err
	}
	return st, nil
}

func mihomoProbe(cl *mihomoClient, cfg MihomoConfig) (MihomoStatus, error) {
	st := MihomoStatus{Configured: cfg.Controller != "", Controller: cfg.Controller, HasSecret: cfg.Secret != ""}
	v, meta, err := cl.version()
	if err != nil {
		st.Error = err.Error()
		return st, err
	}
	st.Version, st.Meta, st.Connected = v, meta, true
	cf, err := cl.configs()
	if err != nil {
		st.Error = err.Error()
		return st, err
	}
	st.Mode = cf.Mode
	st.ListenHost, st.ListenPort, st.ListenKind = listenerFrom(cfg.Controller, cf)
	return st, nil
}

// GetMihomoStatus re-probes the saved controller and refreshes the cached listener.
// It never returns an error: problems are reported in Status.Error.
func GetMihomoStatus() MihomoStatus {
	cl, cfg, err := mihomoFromConfig()
	if err != nil {
		return MihomoStatus{}
	}
	st, err := mihomoProbe(cl, cfg)
	if err == nil && (st.ListenPort != cfg.ListenPort || st.ListenHost != cfg.ListenHost || st.ListenKind != cfg.ListenKind) {
		cfg.ListenHost, cfg.ListenPort, cfg.ListenKind = st.ListenHost, st.ListenPort, st.ListenKind
		_ = mihomoCfg.save(cfg)
	}
	return st
}

var mihomoGroupTypes = map[string]bool{"Selector": true, "URLTest": true, "Fallback": true, "LoadBalance": true, "Relay": true}

func GetMihomoProxies() (MihomoProxies, error) {
	cl, _, err := mihomoFromConfig()
	if err != nil {
		return MihomoProxies{}, err
	}
	var raw struct {
		Proxies map[string]struct {
			Name    string   `json:"name"`
			Type    string   `json:"type"`
			Now     string   `json:"now"`
			All     []string `json:"all"`
			UDP     bool     `json:"udp"`
			Alive   *bool    `json:"alive"`
			History []struct {
				Delay int `json:"delay"`
			} `json:"history"`
		} `json:"proxies"`
	}
	if err := cl.do("GET", "/proxies", nil, &raw); err != nil {
		return MihomoProxies{}, err
	}
	out := MihomoProxies{Groups: []MihomoNode{}, Nodes: []MihomoNode{}}
	for key, p := range raw.Proxies {
		n := MihomoNode{Name: p.Name, Type: p.Type, Now: p.Now, All: p.All, UDP: p.UDP, IsGroup: mihomoGroupTypes[p.Type]}
		if n.Name == "" {
			n.Name = key
		}
		if len(p.History) > 0 {
			n.Delay = p.History[len(p.History)-1].Delay
		}
		if p.Alive != nil {
			n.Alive = *p.Alive
		} else {
			n.Alive = n.Delay > 0
		}
		if n.IsGroup {
			out.Groups = append(out.Groups, n)
		} else {
			out.Nodes = append(out.Nodes, n)
		}
	}
	// GLOBAL holds every name in config order; use it to order the rest.
	order := map[string]int{}
	if g, ok := raw.Proxies["GLOBAL"]; ok {
		for i, name := range g.All {
			order[name] = i + 1
		}
	}
	rank := func(name string) int {
		if name == "GLOBAL" {
			return 1 << 30
		}
		if i, ok := order[name]; ok {
			return i
		}
		return 1 << 29
	}
	for _, l := range [][]MihomoNode{out.Groups, out.Nodes} {
		sort.SliceStable(l, func(i, j int) bool {
			ri, rj := rank(l[i].Name), rank(l[j].Name)
			if ri != rj {
				return ri < rj
			}
			return l[i].Name < l[j].Name
		})
	}
	return out, nil
}

func MihomoSelect(group, node string) error {
	if strings.TrimSpace(group) == "" || strings.TrimSpace(node) == "" {
		return fmt.Errorf("请选择策略组和节点")
	}
	cl, _, err := mihomoFromConfig()
	if err != nil {
		return err
	}
	return cl.do("PUT", "/proxies/"+url.PathEscape(group), map[string]string{"name": node}, nil)
}

const mihomoDelayURL = "https://www.gstatic.com/generate_204"

// MihomoTestDelay asks the core to measure one node (or group). Returns ms.
func MihomoTestDelay(node string, testURL string, timeoutMs int) (int, error) {
	if strings.TrimSpace(node) == "" {
		return 0, fmt.Errorf("请选择节点")
	}
	if testURL == "" {
		testURL = mihomoDelayURL
	}
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	cl, _, err := mihomoFromConfig()
	if err != nil {
		return 0, err
	}
	q := url.Values{"url": {testURL}, "timeout": {strconv.Itoa(timeoutMs)}}
	var v struct {
		Delay int `json:"delay"`
	}
	if err := cl.do("GET", "/proxies/"+url.PathEscape(node)+"/delay?"+q.Encode(), nil, &v); err != nil {
		if strings.Contains(err.Error(), "HTTP 504") || strings.Contains(err.Error(), "HTTP 503") || strings.Contains(err.Error(), "HTTP 408") {
			return 0, fmt.Errorf("节点超时（%d ms 内无响应）", timeoutMs)
		}
		return 0, err
	}
	return v.Delay, nil
}

// MihomoAddProxy creates (or updates) an http Proxy that points at Mihomo's
// mixed/http listener, so a profile can route through the node selected in Mihomo.
func (s *Store) MihomoAddProxy() (Proxy, error) {
	st := GetMihomoStatus()
	if !st.Connected {
		if st.Error != "" {
			return Proxy{}, fmt.Errorf("%s", st.Error)
		}
		return Proxy{}, fmt.Errorf("还没有连接 Mihomo")
	}
	if st.ListenPort <= 0 {
		return Proxy{}, fmt.Errorf("Mihomo 没有开启 mixed-port 或 port，无法作为 HTTP 代理使用")
	}
	for _, p := range s.ListProxies() {
		if p.Tag == "mihomo" && p.Host == st.ListenHost && p.Port == st.ListenPort && p.Type == "http" {
			return p, nil
		}
	}
	return s.PutProxy(Proxy{Name: fmt.Sprintf("Mihomo %s:%d", st.ListenHost, st.ListenPort), Type: "http",
		Host: st.ListenHost, Port: st.ListenPort, Tag: "mihomo"}), nil
}
