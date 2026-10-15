package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gopkg.in/yaml.v3"
)

// Proxy subscriptions (机场订阅): a URL that returns a Clash YAML profile or a
// base64 / plain list of share links. Every node becomes an ordinary Proxy in
// the list, grouped by its subscription (Proxy.SubID / SubName). Plain
// http / socks5 nodes are used directly; every other protocol (vless, vmess,
// ss, trojan, hysteria2, tuic, anytls…) runs through the managed mihomo core
// (core.go), which exposes one local mixed listener per node in use.

const (
	subDefaultUA    = "clash-verge/v1.7"
	subMaxBytes     = 20 << 20
	subFetchTimeout = 30 * time.Second
)

type Subscription struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	URL             string         `json:"url"`
	UA              string         `json:"ua"`              // "" = clash-verge/v1.7
	AutoUpdateHours int            `json:"autoUpdateHours"` // 0 = off
	Enabled         bool           `json:"enabled"`
	CreatedAt       int64          `json:"createdAt"`
	LastUpdated     int64          `json:"lastUpdated"` // last successful refresh
	LastAttempt     int64          `json:"lastAttempt"`
	LastError       string         `json:"lastError"`
	NodeCount       int            `json:"nodeCount"`
	Skipped         int            `json:"skipped"` // lines / nodes that could not be used (ssr links, unknown types)
	Format          string         `json:"format"`  // clash | uri
	Protocols       map[string]int `json:"protocols"`
	// subscription-userinfo
	Upload   int64 `json:"upload"`
	Download int64 `json:"download"`
	Total    int64 `json:"total"`
	Expire   int64 `json:"expire"`
}

// SubPreview is what the add dialog shows before anything is saved.
type SubPreview struct {
	Name      string         `json:"name"`
	Format    string         `json:"format"`
	Count     int            `json:"count"`
	CoreCount int            `json:"coreCount"` // nodes that need the mihomo core
	Skipped   int            `json:"skipped"`
	Protocols map[string]int `json:"protocols"`
	Interval  int            `json:"interval"` // profile-update-interval (hours), 0 = not sent
	Upload    int64          `json:"upload"`
	Download  int64          `json:"download"`
	Total     int64          `json:"total"`
	Expire    int64          `json:"expire"`
}

// SubRefreshResult counts what one refresh changed.
type SubRefreshResult struct {
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Gone    int `json:"gone"`    // missing from the source but still used by a profile
	Removed int `json:"removed"` // missing from the source and unused: deleted
}

// ---------------------------------------------------------------- protocols

// directProtocols can be handed to the browser as they are.
var directProtocols = map[string]bool{"http": true, "socks5": true}

// clashTypes are the outbound types accepted from a Clash profile (mihomo can run all of them).
var clashTypes = map[string]bool{"http": true, "socks5": true, "ss": true, "ssr": true, "vmess": true, "vless": true, "trojan": true,
	"hysteria": true, "hysteria2": true, "tuic": true, "anytls": true, "wireguard": true, "snell": true, "ssh": true, "mieru": true}

// needsCore: a subscription node the browser cannot speak itself.
func (p Proxy) needsCore() bool {
	if p.RawNode == "" || p.Protocol == "" {
		return false
	}
	return !directProtocols[p.Protocol]
}

// ---------------------------------------------------------------- fetch

type subFetch struct {
	Body   []byte
	Header http.Header
}

var subHTTPClient = func(px *url.URL) *http.Client {
	tr := &http.Transport{Proxy: nil, TLSHandshakeTimeout: 15 * time.Second, ResponseHeaderTimeout: subFetchTimeout}
	if px != nil {
		tr.Proxy = http.ProxyURL(px)
	}
	return &http.Client{Timeout: subFetchTimeout, Transport: tr}
}

func fetchSubOnce(raw, ua string, px *url.URL) (subFetch, error) {
	req, err := http.NewRequest("GET", raw, nil)
	if err != nil {
		return subFetch{}, err
	}
	if ua == "" {
		ua = subDefaultUA
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "*/*")
	resp, err := subHTTPClient(px).Do(req)
	if err != nil {
		return subFetch{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return subFetch{}, fmt.Errorf("订阅服务器返回 HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, subMaxBytes+1))
	if err != nil {
		return subFetch{}, fmt.Errorf("读取订阅内容失败：%v", err)
	}
	if len(b) > subMaxBytes {
		return subFetch{}, fmt.Errorf("订阅内容超过 20 MB，已拒绝")
	}
	return subFetch{Body: b, Header: resp.Header}, nil
}

// fetchSub downloads directly first, then through the Mihomo listener the user
// connected (代理 → Clash / Mihomo) when the direct request fails.
func fetchSub(raw, ua string) (subFetch, error) {
	f, err := fetchSubOnce(raw, ua, nil)
	if err == nil {
		return f, nil
	}
	if strings.HasPrefix(err.Error(), "订阅") { // the server answered: retrying elsewhere will not help
		return f, err
	}
	if px := mihomoListenerURL(); px != nil {
		if f2, err2 := fetchSubOnce(raw, ua, px); err2 == nil {
			return f2, nil
		}
	}
	return f, fmt.Errorf("下载订阅失败：%s", shortErr(err))
}

func validSubURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("订阅链接必须以 http:// 或 https:// 开头")
	}
	return raw, nil
}

// ---------------------------------------------------------------- headers

// parseUserInfo reads "upload=1; download=2; total=3; expire=4".
func parseUserInfo(h string) (up, down, total, expire int64) {
	for _, part := range strings.Split(h, ";") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		n, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "upload":
			up = int64(n)
		case "download":
			down = int64(n)
		case "total":
			total = int64(n)
		case "expire":
			expire = int64(n)
		}
	}
	return
}

// nameFromDisposition takes the filename (RFC 5987 filename* first) without its extension.
func nameFromDisposition(h string) string {
	if h == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(h)
	name := ""
	if err == nil {
		name = params["filename"]
	}
	if name == "" { // lenient fallback for servers that send odd encodings
		for _, part := range strings.Split(h, ";") {
			k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
			if !ok {
				continue
			}
			k = strings.ToLower(k)
			if k == "filename*" {
				if i := strings.Index(v, "''"); i >= 0 {
					v = v[i+2:]
				}
				if d, e := url.PathUnescape(v); e == nil {
					v = d
				}
				name = v
				break
			}
			if k == "filename" {
				name = strings.Trim(v, `"`)
			}
		}
	}
	// Some servers encode spaces as "+" inside filename*.
	if !strings.Contains(name, " ") {
		name = strings.ReplaceAll(name, "+", " ")
	}
	for _, ext := range []string{".yaml", ".yml", ".txt", ".json", ".conf"} {
		if strings.HasSuffix(strings.ToLower(name), ext) {
			name = name[:len(name)-len(ext)]
			break
		}
	}
	return strings.TrimSpace(name)
}

func subNameFor(raw string, h http.Header) string {
	if n := nameFromDisposition(h.Get("Content-Disposition")); n != "" {
		return n
	}
	if u, err := url.Parse(raw); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return "订阅"
}

// ---------------------------------------------------------------- parsing

// parseSubscription turns a subscription body into Clash-style proxy maps.
func parseSubscription(body []byte) (nodes []map[string]any, format string, skipped int, err error) {
	t := strings.TrimSpace(strings.TrimPrefix(string(body), "\ufeff"))
	if t == "" {
		return nil, "", 0, fmt.Errorf("订阅内容为空")
	}
	if strings.HasPrefix(t, "{") {
		var sb struct {
			Outbounds []any `json:"outbounds"`
		}
		if json.Unmarshal([]byte(t), &sb) == nil && sb.Outbounds != nil {
			return nil, "sing-box", 0, fmt.Errorf("暂不支持 sing-box 格式：请使用 Clash 格式的订阅链接（大多数机场在订阅链接后加 &flag=clash 即可）")
		}
	}
	if looksLikeClash(t) {
		var y struct {
			Proxies []map[string]any `yaml:"proxies"`
		}
		if e := yaml.Unmarshal([]byte(t), &y); e != nil {
			return nil, "clash", 0, fmt.Errorf("Clash 配置解析失败：%v", e)
		}
		for _, m := range y.Proxies {
			m = normalizeYAMLMap(m)
			typ := strings.ToLower(str(m["type"]))
			if !clashTypes[typ] || str(m["server"]) == "" || toInt(m["port"]) <= 0 {
				skipped++
				continue
			}
			m["type"] = typ
			nodes = append(nodes, m)
		}
		if len(nodes) == 0 {
			return nil, "clash", skipped, fmt.Errorf("Clash 配置里没有可用的节点")
		}
		return nodes, "clash", skipped, nil
	}
	text := t
	if !strings.Contains(t, "://") {
		if d, ok := decodeB64(t); ok && strings.Contains(d, "://") {
			text = d
		}
	}
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r", ""), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		m, e := parseShareLink(line)
		if e != nil {
			skipped++
			continue
		}
		nodes = append(nodes, m)
	}
	if len(nodes) == 0 {
		return nil, "uri", skipped, fmt.Errorf("没有识别出任何节点：订阅需为 Clash YAML、Base64 或分享链接列表")
	}
	return nodes, "uri", skipped, nil
}

func looksLikeClash(t string) bool {
	if strings.HasPrefix(t, "proxies:") || strings.Contains(t, "\nproxies:") {
		return true
	}
	return false
}

// normalizeYAMLMap converts nested map[any]any (yaml.v2 style) to map[string]any so it marshals as JSON.
func normalizeYAMLMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = normYAML(v)
	}
	return out
}

func normYAML(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return normalizeYAMLMap(x)
	case map[any]any:
		o := map[string]any{}
		for k, vv := range x {
			o[fmt.Sprint(k)] = normYAML(vv)
		}
		return o
	case []any:
		for i := range x {
			x[i] = normYAML(x[i])
		}
		return x
	}
	return v
}

func decodeB64(s string) (string, bool) {
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, s)
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(s); err == nil {
			return string(b), true
		}
	}
	// tolerate missing padding with std alphabet
	if m := len(s) % 4; m != 0 {
		if b, err := base64.StdEncoding.DecodeString(s + strings.Repeat("=", 4-m)); err == nil {
			return string(b), true
		}
	}
	return "", false
}

func toInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(x))
		return n
	}
	return 0
}

func truthy(s string) bool {
	switch strings.ToLower(s) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// splitUserHost parses "cred@host:port" (the part after "scheme://", before "?" / "#").
func splitUserHost(s string) (user, host string, port int, err error) {
	if at := strings.LastIndex(s, "@"); at >= 0 {
		user, s = s[:at], s[at+1:]
	}
	s = strings.TrimSuffix(s, "/")
	if i := strings.IndexAny(s, "/"); i >= 0 {
		s = s[:i]
	}
	h, p, e := net.SplitHostPort(s)
	if e != nil {
		return "", "", 0, fmt.Errorf("找不到 主机:端口")
	}
	n, e := strconv.Atoi(p)
	if e != nil || n <= 0 || n > 65535 {
		return "", "", 0, fmt.Errorf("端口无效")
	}
	return user, h, n, nil
}

// splitLink cuts "scheme://body?query#frag".
func splitLink(line string) (scheme, body string, q url.Values, frag string) {
	scheme, rest, _ := strings.Cut(line, "://")
	scheme = strings.ToLower(scheme)
	if i := strings.Index(rest, "#"); i >= 0 {
		frag, rest = rest[i+1:], rest[:i]
		if d, err := url.PathUnescape(frag); err == nil {
			frag = d
		}
	}
	if i := strings.Index(rest, "?"); i >= 0 {
		q, _ = url.ParseQuery(rest[i+1:])
		rest = rest[:i]
	}
	if q == nil {
		q = url.Values{}
	}
	return scheme, rest, q, strings.TrimSpace(frag)
}

func unesc(s string) string {
	if d, err := url.PathUnescape(s); err == nil {
		return d
	}
	return s
}

func setIf(m map[string]any, k, v string) {
	if v != "" {
		m[k] = v
	}
}

func alpnList(s string) []any {
	if s == "" {
		return nil
	}
	var out []any
	for _, a := range strings.Split(s, ",") {
		if a = strings.TrimSpace(a); a != "" {
			out = append(out, a)
		}
	}
	return out
}

// applyTransport maps v2ray-style query params (type, host, path, serviceName…) onto a Clash node.
func applyTransport(m map[string]any, network, host, path, service string) {
	switch network {
	case "", "tcp", "raw":
		if network != "" {
			m["network"] = "tcp"
		}
	case "ws", "httpupgrade":
		m["network"] = "ws"
		o := map[string]any{}
		if path != "" {
			o["path"] = path
		}
		if host != "" {
			o["headers"] = map[string]any{"Host": host}
		}
		if network == "httpupgrade" {
			o["v2ray-http-upgrade"] = true
		}
		m["ws-opts"] = o
	case "grpc":
		m["network"] = "grpc"
		m["grpc-opts"] = map[string]any{"grpc-service-name": service}
	case "h2":
		m["network"] = "h2"
		o := map[string]any{}
		if path != "" {
			o["path"] = path
		}
		if host != "" {
			o["host"] = []any{host}
		}
		m["h2-opts"] = o
	case "http":
		m["network"] = "http"
		o := map[string]any{}
		if path != "" {
			o["path"] = []any{path}
		}
		if host != "" {
			o["headers"] = map[string]any{"Host": []any{host}}
		}
		m["http-opts"] = o
	case "xhttp", "splithttp":
		m["network"] = "xhttp"
		o := map[string]any{}
		if path != "" {
			o["path"] = path
		}
		if host != "" {
			o["host"] = host
		}
		m["xhttp-opts"] = o
	default:
		m["network"] = network
	}
}

// parseShareLink converts one share link into a Clash proxy map.
func parseShareLink(line string) (map[string]any, error) {
	if !strings.Contains(line, "://") {
		return nil, fmt.Errorf("不是分享链接")
	}
	scheme, body, q, name := splitLink(line)
	switch scheme {
	case "vmess":
		return parseVmess(body, name)
	case "ss":
		return parseSS(body, q, name)
	case "ssr":
		return nil, fmt.Errorf("暂不支持 SSR 链接")
	}
	user, host, port, err := splitUserHost(body)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = net.JoinHostPort(host, strconv.Itoa(port))
	}
	m := map[string]any{"name": name, "server": host, "port": port}
	insecure := truthy(q.Get("insecure")) || truthy(q.Get("allowInsecure")) || truthy(q.Get("allow_insecure"))
	switch scheme {
	case "http", "https", "socks5", "socks", "socks5h":
		m["type"] = "http"
		if strings.HasPrefix(scheme, "socks") {
			m["type"] = "socks5"
		}
		if user != "" {
			u, p, _ := strings.Cut(user, ":")
			m["username"], m["password"] = unesc(u), unesc(p)
		}
		if scheme == "https" {
			m["tls"] = true
			setIf(m, "sni", q.Get("sni"))
			if insecure {
				m["skip-cert-verify"] = true
			}
		}
	case "vless":
		m["type"] = "vless"
		m["uuid"] = unesc(user)
		m["udp"] = true
		setIf(m, "flow", q.Get("flow"))
		sec := q.Get("security")
		if sec == "tls" || sec == "reality" || sec == "xtls" {
			m["tls"] = true
			setIf(m, "servername", firstNonEmpty(q.Get("sni"), q.Get("peer")))
			setIf(m, "client-fingerprint", q.Get("fp"))
			if a := alpnList(q.Get("alpn")); a != nil {
				m["alpn"] = a
			}
			if insecure {
				m["skip-cert-verify"] = true
			}
		}
		if sec == "reality" {
			ro := map[string]any{"public-key": q.Get("pbk")}
			setIf(ro, "short-id", q.Get("sid"))
			m["reality-opts"] = ro
		}
		applyTransport(m, q.Get("type"), q.Get("host"), q.Get("path"), q.Get("serviceName"))
	case "trojan":
		m["type"] = "trojan"
		m["password"] = unesc(user)
		m["udp"] = true
		setIf(m, "sni", firstNonEmpty(q.Get("sni"), q.Get("peer")))
		setIf(m, "client-fingerprint", q.Get("fp"))
		if a := alpnList(q.Get("alpn")); a != nil {
			m["alpn"] = a
		}
		if insecure {
			m["skip-cert-verify"] = true
		}
		if q.Get("security") == "reality" {
			ro := map[string]any{"public-key": q.Get("pbk")}
			setIf(ro, "short-id", q.Get("sid"))
			m["reality-opts"] = ro
		}
		applyTransport(m, q.Get("type"), q.Get("host"), q.Get("path"), q.Get("serviceName"))
	case "hysteria2", "hy2":
		m["type"] = "hysteria2"
		pw := unesc(user)
		if pw == "" {
			pw = q.Get("auth")
		}
		m["password"] = pw
		m["udp"] = true
		setIf(m, "sni", q.Get("sni"))
		setIf(m, "obfs", q.Get("obfs"))
		setIf(m, "obfs-password", q.Get("obfs-password"))
		setIf(m, "fingerprint", q.Get("pinSHA256"))
		setIf(m, "ports", q.Get("mport"))
		if a := alpnList(q.Get("alpn")); a != nil {
			m["alpn"] = a
		}
		if insecure {
			m["skip-cert-verify"] = true
		}
	case "tuic":
		m["type"] = "tuic"
		u, p, _ := strings.Cut(user, ":")
		m["uuid"], m["password"] = unesc(u), unesc(p)
		m["udp"] = true
		setIf(m, "sni", q.Get("sni"))
		setIf(m, "congestion-controller", q.Get("congestion_control"))
		setIf(m, "udp-relay-mode", q.Get("udp_relay_mode"))
		if a := alpnList(q.Get("alpn")); a != nil {
			m["alpn"] = a
		}
		if insecure {
			m["skip-cert-verify"] = true
		}
	case "anytls":
		m["type"] = "anytls"
		m["password"] = unesc(user)
		m["udp"] = true
		setIf(m, "sni", q.Get("sni"))
		setIf(m, "client-fingerprint", q.Get("fp"))
		if insecure {
			m["skip-cert-verify"] = true
		}
	default:
		return nil, fmt.Errorf("不支持的协议 %s", scheme)
	}
	return m, nil
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func parseVmess(body, frag string) (map[string]any, error) {
	d, ok := decodeB64(body)
	if !ok {
		return nil, fmt.Errorf("vmess 链接不是 Base64")
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(d), &v); err != nil {
		return nil, fmt.Errorf("vmess 链接内容无法解析")
	}
	host, port := str(v["add"]), toInt(v["port"])
	if host == "" || port <= 0 {
		return nil, fmt.Errorf("vmess 链接缺少地址")
	}
	name := firstNonEmpty(str(v["ps"]), frag, net.JoinHostPort(host, strconv.Itoa(port)))
	m := map[string]any{"name": name, "type": "vmess", "server": host, "port": port, "uuid": str(v["id"]),
		"alterId": toInt(v["aid"]), "cipher": firstNonEmpty(str(v["scy"]), "auto"), "udp": true}
	if t := str(v["tls"]); t == "tls" || t == "true" {
		m["tls"] = true
		setIf(m, "servername", str(v["sni"]))
		setIf(m, "client-fingerprint", str(v["fp"]))
		if a := alpnList(str(v["alpn"])); a != nil {
			m["alpn"] = a
		}
	}
	net := str(v["net"])
	if net == "tcp" && str(v["type"]) == "http" {
		net = "http"
	}
	service := str(v["path"])
	applyTransport(m, net, str(v["host"]), str(v["path"]), service)
	return m, nil
}

func parseSS(body string, q url.Values, name string) (map[string]any, error) {
	var method, pass, rest string
	if at := strings.LastIndex(body, "@"); at >= 0 {
		cred := body[:at]
		rest = body[at+1:]
		if d, ok := decodeB64(unesc(cred)); ok && strings.Contains(d, ":") {
			cred = d
		} else {
			cred = unesc(cred)
		}
		method, pass, _ = strings.Cut(cred, ":")
	} else { // legacy: ss://base64(method:pass@host:port)
		d, ok := decodeB64(body)
		if !ok {
			return nil, fmt.Errorf("ss 链接无法解析")
		}
		at := strings.LastIndex(d, "@")
		if at < 0 {
			return nil, fmt.Errorf("ss 链接无法解析")
		}
		method, pass, _ = strings.Cut(d[:at], ":")
		rest = d[at+1:]
	}
	_, host, port, err := splitUserHost(rest)
	if err != nil {
		return nil, err
	}
	if method == "" {
		return nil, fmt.Errorf("ss 链接缺少加密方式")
	}
	if name == "" {
		name = net.JoinHostPort(host, strconv.Itoa(port))
	}
	m := map[string]any{"name": name, "type": "ss", "server": host, "port": port, "cipher": method, "password": pass, "udp": true}
	if pl := q.Get("plugin"); pl != "" {
		parts := strings.Split(pl, ";")
		opts := map[string]string{}
		for _, p := range parts[1:] {
			k, v, _ := strings.Cut(p, "=")
			opts[k] = v
		}
		switch {
		case strings.Contains(parts[0], "obfs"):
			m["plugin"] = "obfs"
			po := map[string]any{"mode": opts["obfs"]}
			setIf(po, "host", opts["obfs-host"])
			m["plugin-opts"] = po
		case strings.Contains(parts[0], "v2ray"):
			m["plugin"] = "v2ray-plugin"
			po := map[string]any{"mode": firstNonEmpty(opts["mode"], "websocket")}
			setIf(po, "host", opts["host"])
			setIf(po, "path", opts["path"])
			if _, ok := opts["tls"]; ok {
				po["tls"] = true
			}
			m["plugin-opts"] = po
		default:
			return nil, fmt.Errorf("不支持的 ss 插件 %s", parts[0])
		}
	}
	return m, nil
}

// ---------------------------------------------------------------- node → Proxy

// nodeKey identifies a node across refreshes: type + server + port + credential.
func nodeKey(m map[string]any) string {
	cred := ""
	for _, k := range []string{"uuid", "password", "auth-str", "auth", "private-key"} {
		if v := str(m[k]); v != "" {
			cred = v
			break
		}
	}
	if u := str(m["username"]); u != "" {
		cred = u + ":" + cred
	}
	sum := sha256.Sum256([]byte(strings.ToLower(str(m["type"])) + "|" + strings.ToLower(str(m["server"])) + "|" + strconv.Itoa(toInt(m["port"])) + "|" + cred))
	return hex.EncodeToString(sum[:8])
}

// nodeProtocol is the label shown in the list ("https" = an HTTP proxy over TLS).
func nodeProtocol(m map[string]any) string {
	t := strings.ToLower(str(m["type"]))
	tls := m["tls"] == true || truthy(str(m["tls"]))
	if t == "http" && tls {
		return "https"
	}
	if t == "socks5" && tls {
		return "socks5-tls"
	}
	return t
}

// nodeToProxy builds the Proxy record for one subscription node.
func nodeToProxy(m map[string]any, sub *Subscription) (Proxy, error) {
	host, port := str(m["server"]), toInt(m["port"])
	if host == "" || port <= 0 || port > 65535 {
		return Proxy{}, fmt.Errorf("节点缺少地址")
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return Proxy{}, err
	}
	proto := nodeProtocol(m)
	p := Proxy{Name: strings.TrimSpace(str(m["name"])), Host: host, Port: port, Protocol: proto, RawNode: string(raw), NodeKey: nodeKey(m)}
	if sub != nil {
		p.SubID, p.SubName = sub.ID, sub.Name
	}
	if p.Name == "" {
		p.Name = net.JoinHostPort(host, strconv.Itoa(port))
	}
	switch proto {
	case "http", "socks5":
		p.Type = proto
		p.User, p.Pass = str(m["username"]), str(m["password"])
	default:
		p.Type = proto // routed through the mihomo core; launch resolves it to a local listener
	}
	return p, nil
}

// subNodeID is deterministic so two computers refreshing the same source
// create the same proxy IDs (global sync stays conflict-free).
func subNodeID(subID, key string) string {
	sum := sha256.Sum256([]byte(subID + "|" + key))
	return "s" + hex.EncodeToString(sum[:8])
}

// buildSubNodes converts parsed maps to proxies with unique keys (a repeated
// key gets "#2", "#3"… in source order, so it stays stable).
func buildSubNodes(maps []map[string]any, sub *Subscription) ([]Proxy, map[string]int, int) {
	seen := map[string]int{}
	protos := map[string]int{}
	var out []Proxy
	skipped := 0
	for _, m := range maps {
		p, err := nodeToProxy(m, sub)
		if err != nil {
			skipped++
			continue
		}
		seen[p.NodeKey]++
		if n := seen[p.NodeKey]; n > 1 {
			p.NodeKey = fmt.Sprintf("%s#%d", p.NodeKey, n)
		}
		protos[p.Protocol]++
		out = append(out, p)
	}
	return out, protos, skipped
}

// ---------------------------------------------------------------- store

var subLocks sync.Map // id -> *sync.Mutex, one refresh per subscription at a time

func subLock(id string) *sync.Mutex {
	v, _ := subLocks.LoadOrStore(id, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func (s *Store) ListSubscriptions() []Subscription {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Subscription, 0, len(s.Subs))
	for _, x := range s.Subs {
		out = append(out, *x)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *Store) getSub(id string) (Subscription, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	x, ok := s.Subs[id]
	if !ok {
		return Subscription{}, false
	}
	return *x, true
}

func previewFrom(f subFetch, raw string) (SubPreview, []map[string]any, error) {
	maps, format, skipped, err := parseSubscription(f.Body)
	pv := SubPreview{Name: subNameFor(raw, f.Header), Format: format, Skipped: skipped, Protocols: map[string]int{}}
	pv.Upload, pv.Download, pv.Total, pv.Expire = parseUserInfo(f.Header.Get("Subscription-Userinfo"))
	if n, e := strconv.Atoi(strings.TrimSpace(f.Header.Get("Profile-Update-Interval"))); e == nil && n > 0 {
		pv.Interval = n
	}
	if err != nil {
		return pv, nil, err
	}
	nodes, protos, bad := buildSubNodes(maps, nil)
	pv.Count, pv.Protocols, pv.Skipped = len(nodes), protos, skipped+bad
	for _, n := range nodes {
		if n.needsCore() {
			pv.CoreCount++
		}
	}
	return pv, maps, nil
}

// PreviewSubscription downloads and parses a source without saving anything.
func (s *Store) PreviewSubscription(raw, ua string) (SubPreview, error) {
	raw, err := validSubURL(raw)
	if err != nil {
		return SubPreview{}, err
	}
	f, err := fetchSub(raw, ua)
	if err != nil {
		return SubPreview{}, err
	}
	pv, _, err := previewFrom(f, raw)
	return pv, err
}

// AddSubscription saves a new source and imports its nodes.
func (s *Store) AddSubscription(raw, name string) (Subscription, SubRefreshResult, error) {
	raw, err := validSubURL(raw)
	if err != nil {
		return Subscription{}, SubRefreshResult{}, err
	}
	for _, x := range s.ListSubscriptions() {
		if x.URL == raw {
			return x, SubRefreshResult{}, fmt.Errorf("这个订阅已经添加过了：%s", x.Name)
		}
	}
	f, err := fetchSub(raw, subDefaultUA)
	if err != nil {
		return Subscription{}, SubRefreshResult{}, err
	}
	pv, maps, err := previewFrom(f, raw)
	if err != nil {
		return Subscription{}, SubRefreshResult{}, err
	}
	sub := Subscription{ID: newID(), Name: strings.TrimSpace(name), URL: raw, UA: subDefaultUA, Enabled: true, CreatedAt: time.Now().Unix(), AutoUpdateHours: 24}
	if sub.Name == "" {
		sub.Name = pv.Name
	}
	if pv.Interval > 0 {
		sub.AutoUpdateHours = pv.Interval
	}
	res := s.commitSubRefresh(&sub, pv, maps, true)
	return sub, res, nil
}

// commitSubRefresh stores the subscription and upserts its nodes in one save.
func (s *Store) commitSubRefresh(sub *Subscription, pv SubPreview, maps []map[string]any, isNew bool) SubRefreshResult {
	nodes, protos, _ := buildSubNodes(maps, sub)
	now := time.Now().Unix()
	sub.LastUpdated, sub.LastAttempt, sub.LastError = now, now, ""
	sub.NodeCount, sub.Protocols, sub.Skipped, sub.Format = len(nodes), protos, pv.Skipped, pv.Format
	sub.Upload, sub.Download, sub.Total, sub.Expire = pv.Upload, pv.Download, pv.Total, pv.Expire
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Subs == nil {
		s.Subs = map[string]*Subscription{}
	}
	if !isNew {
		if cur, ok := s.Subs[sub.ID]; ok { // keep edits made while downloading
			sub.Name, sub.URL, sub.UA, sub.AutoUpdateHours, sub.Enabled = cur.Name, cur.URL, cur.UA, cur.AutoUpdateHours, cur.Enabled
			for i := range nodes {
				nodes[i].SubName = cur.Name
			}
		} else {
			return SubRefreshResult{} // deleted meanwhile
		}
	}
	cp := *sub
	s.Subs[sub.ID] = &cp
	res := s.upsertSubNodesLocked(sub.ID, nodes)
	s.saveLocked()
	return res
}

// upsertSubNodesLocked keeps proxy IDs stable by NodeKey. Nodes that left the
// source are deleted, or kept and marked gone when a profile still uses them.
func (s *Store) upsertSubNodesLocked(subID string, nodes []Proxy) SubRefreshResult {
	var res SubRefreshResult
	existing := map[string]*Proxy{}
	for _, p := range s.Proxies {
		if p.SubID == subID {
			existing[p.NodeKey] = p
		}
	}
	used := map[string]bool{}
	for _, pr := range s.Profiles {
		if pr.ProxyID != "" {
			used[pr.ProxyID] = true
		}
	}
	keep := map[string]bool{}
	for _, n := range nodes {
		keep[n.NodeKey] = true
		if cur, ok := existing[n.NodeKey]; ok {
			changed := cur.Host != n.Host || cur.Port != n.Port || cur.RawNode != n.RawNode
			cur.Name, cur.Host, cur.Port, cur.Type, cur.Protocol = n.Name, n.Host, n.Port, n.Type, n.Protocol
			cur.User, cur.Pass, cur.RawNode, cur.SubName, cur.Gone = n.User, n.Pass, n.RawNode, n.SubName, false
			if changed {
				cur.Status = "unknown"
			}
			res.Updated++
			continue
		}
		n.ID = subNodeID(subID, n.NodeKey)
		if old, ok := s.Proxies[n.ID]; ok && old.SubID != subID { // never clobber an unrelated record
			n.ID = newID()
		}
		n.Status = "unknown"
		cp := n
		s.Proxies[n.ID] = &cp
		res.Added++
	}
	for key, p := range existing {
		if keep[key] {
			continue
		}
		if used[p.ID] {
			if !p.Gone {
				p.Gone = true
			}
			res.Gone++
			continue
		}
		delete(s.Proxies, p.ID)
		res.Removed++
	}
	return res
}

func (s *Store) markSubError(id string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if x, ok := s.Subs[id]; ok {
		x.LastAttempt, x.LastError = time.Now().Unix(), err.Error()
		s.saveLocked()
	}
}

// UpdateSubscription re-downloads one source and upserts its nodes.
func (s *Store) UpdateSubscription(id string) (Subscription, SubRefreshResult, error) {
	l := subLock(id)
	l.Lock()
	defer l.Unlock()
	sub, ok := s.getSub(id)
	if !ok {
		return Subscription{}, SubRefreshResult{}, fmt.Errorf("订阅不存在")
	}
	f, err := fetchSub(sub.URL, sub.UA)
	if err != nil {
		s.markSubError(id, err)
		cur, _ := s.getSub(id)
		return cur, SubRefreshResult{}, err
	}
	pv, maps, err := previewFrom(f, sub.URL)
	if err != nil {
		s.markSubError(id, err)
		cur, _ := s.getSub(id)
		return cur, SubRefreshResult{}, err
	}
	res := s.commitSubRefresh(&sub, pv, maps, false)
	cur, _ := s.getSub(id)
	return cur, res, nil
}

// EditSubscription changes name, URL, UA, interval and the enabled switch.
func (s *Store) EditSubscription(in Subscription) (Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	x, ok := s.Subs[in.ID]
	if !ok {
		return Subscription{}, fmt.Errorf("订阅不存在")
	}
	u, err := validSubURL(in.URL)
	if err != nil {
		return *x, err
	}
	if in.AutoUpdateHours < 0 || in.AutoUpdateHours > 24*30 {
		return *x, fmt.Errorf("自动更新间隔应在 0–720 小时之间")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = x.Name
	}
	ua := strings.TrimSpace(in.UA)
	if ua == "" {
		ua = subDefaultUA
	}
	if name != x.Name {
		for _, p := range s.Proxies {
			if p.SubID == x.ID {
				p.SubName = name
			}
		}
	}
	x.Name, x.URL, x.UA, x.AutoUpdateHours, x.Enabled = name, u, ua, in.AutoUpdateHours, in.Enabled
	s.saveLocked()
	return *x, nil
}

// DeleteSubscription removes a source. deleteNodes also deletes its nodes
// (profiles using them switch to a direct connection, like deleting a proxy);
// otherwise the nodes stay as manual proxies.
func (s *Store) DeleteSubscription(id string, deleteNodes bool) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Subs[id]; !ok {
		return 0, fmt.Errorf("订阅不存在")
	}
	delete(s.Subs, id)
	n := 0
	for pid, p := range s.Proxies {
		if p.SubID != id {
			continue
		}
		n++
		if deleteNodes {
			delete(s.Proxies, pid)
			for _, pr := range s.Profiles {
				if pr.ProxyID == pid {
					pr.ProxyID = ""
				}
			}
		} else {
			p.SubID, p.SubName, p.Gone = "", "", false
			if p.Tag == "" {
				p.Tag = "订阅"
			}
		}
	}
	s.saveLocked()
	return n, nil
}

// dueSubscriptions: enabled sources whose interval has passed (failed
// attempts retry at most every 30 minutes).
func (s *Store) dueSubscriptions(now time.Time) []string {
	var ids []string
	for _, x := range s.ListSubscriptions() {
		if !x.Enabled || x.AutoUpdateHours <= 0 {
			continue
		}
		if now.Unix()-x.LastUpdated < int64(x.AutoUpdateHours)*3600 {
			continue
		}
		if now.Unix()-x.LastAttempt < 1800 {
			continue
		}
		ids = append(ids, x.ID)
	}
	return ids
}

// ---------------------------------------------------------------- app bindings

func (a *App) emitSubs() {
	if a.ctx != nil && a.ctx.Value("events") != nil {
		wruntime.EventsEmit(a.ctx, "subs:updated", a.store.ListSubscriptions())
	}
}

func (a *App) subscriptionLoop() {
	time.Sleep(20 * time.Second)
	for {
		for _, id := range a.store.dueSubscriptions(time.Now()) {
			_, _, _ = a.store.UpdateSubscription(id)
			a.emitSubs()
		}
		time.Sleep(10 * time.Minute)
	}
}

func (a *App) ListSubscriptions() []Subscription { return a.store.ListSubscriptions() }

// PreviewSubscription shows the node count by protocol before saving (ua "" = clash-verge/v1.7).
func (a *App) PreviewSubscription(url, ua string) (SubPreview, error) {
	return a.store.PreviewSubscription(url, ua)
}

func (a *App) AddSubscription(url, name string) (Subscription, error) {
	sub, res, err := a.store.AddSubscription(url, name)
	if err == nil {
		a.store.auditNamed("ui", "sub.add", sub.ID, sub.Name, nil, fmt.Sprintf("%d nodes", res.Added))
		a.emitSubs()
	}
	return sub, err
}

func (a *App) UpdateSubscription(id string) (SubRefreshResult, error) {
	sub, res, err := a.store.UpdateSubscription(id)
	a.emitSubs()
	if err != nil {
		return res, fmt.Errorf("%s：%v", sub.Name, err)
	}
	return res, nil
}

// UpdateAllSubscriptions refreshes every enabled source, 3 at a time.
func (a *App) UpdateAllSubscriptions() []Subscription {
	var ids []string
	for _, x := range a.store.ListSubscriptions() {
		if x.Enabled {
			ids = append(ids, x.ID)
		}
	}
	runPool(ids, 3, func(id string) { _, _, _ = a.store.UpdateSubscription(id) })
	a.emitSubs()
	return a.store.ListSubscriptions()
}

func (a *App) EditSubscription(sub Subscription) (Subscription, error) {
	out, err := a.store.EditSubscription(sub)
	if err == nil {
		a.emitSubs()
	}
	return out, err
}

func (a *App) DeleteSubscription(id string, deleteNodes bool) error {
	sub, _ := a.store.getSub(id)
	n, err := a.store.DeleteSubscription(id, deleteNodes)
	if err == nil {
		detail := fmt.Sprintf("kept %d nodes", n)
		if deleteNodes {
			detail = fmt.Sprintf("deleted %d nodes", n)
		}
		a.store.auditNamed("ui", "sub.delete", id, sub.Name, nil, detail)
		a.emitSubs()
	}
	return err
}

