package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reScheme = regexp.MustCompile(`(?i)^(https?|socks5h?|socks4a?|socks)://`)
	reIPv4   = regexp.MustCompile(`^\d{1,3}(\.\d{1,3}){3}$`)
	reDomain = regexp.MustCompile(`(?i)^([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z][a-z0-9-]*$`)
	reSplit  = regexp.MustCompile(`[\s,;|]+`)
)

func normType(t string) string {
	switch t = strings.ToLower(t); t {
	case "socks", "socks5h":
		return "socks5"
	case "socks4a":
		return "socks4"
	}
	return t
}

func isProtoWord(t string) bool {
	switch normType(t) {
	case "http", "https", "socks5", "socks4":
		return true
	}
	return false
}

func hostLike(h string) bool {
	h = strings.Trim(h, "[]")
	if reIPv4.MatchString(h) || strings.EqualFold(h, "localhost") || reDomain.MatchString(h) {
		return true
	}
	return net.ParseIP(h) != nil && strings.Contains(h, ":")
}

func portOf(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	return n, err == nil && n > 0 && n < 65536
}

// splitHostPort understands "[v6]:port" and "host:port".
func splitAddr(s string) (string, int, bool) {
	if strings.HasPrefix(s, "[") {
		if i := strings.Index(s, "]:"); i > 0 {
			if p, ok := portOf(s[i+2:]); ok {
				return s[1:i], p, true
			}
		}
		return "", 0, false
	}
	i := strings.LastIndex(s, ":")
	if i <= 0 {
		return "", 0, false
	}
	p, ok := portOf(s[i+1:])
	if !ok || !hostLike(s[:i]) {
		return "", 0, false
	}
	return s[:i], p, true
}

// guessType picks a protocol from the port when the line does not name one.
func guessType(port int) string {
	switch port {
	case 1080, 1081, 1085, 1086, 9050, 9150, 10808, 7891:
		return "socks5"
	}
	return "http"
}

// parseCore parses one proxy written in any of the common shapes:
//
//	host:port                    host:port:user:pass
//	user:pass:host:port          user:pass@host:port
//	host:port@user:pass          scheme://user:pass@host:port
//	[2001:db8::1]:8080           host port user pass (whitespace)
func parseCore(s string) (Proxy, error) {
	p := Proxy{}
	if m := reScheme.FindString(s); m != "" {
		p.Type = normType(m[:len(m)-3])
		s = s[len(m):]
	}
	s = strings.TrimSuffix(s, "/")
	if at := strings.LastIndex(s, "@"); at > 0 {
		l, r := s[:at], s[at+1:]
		if h, port, ok := splitAddr(r); ok {
			p.Host, p.Port = h, port
			p.User, p.Pass, _ = strings.Cut(l, ":")
		} else if h, port, ok := splitAddr(l); ok {
			p.Host, p.Port = h, port
			p.User, p.Pass, _ = strings.Cut(r, ":")
		} else {
			return p, fmt.Errorf("找不到 主机:端口")
		}
		if u, e := url.PathUnescape(p.User); e == nil {
			p.User = u
		}
		if pw, e := url.PathUnescape(p.Pass); e == nil {
			p.Pass = pw
		}
		return p, nil
	}
	if h, port, ok := splitAddr(s); ok {
		p.Host, p.Port = h, port
		return p, nil
	}
	if strings.HasPrefix(s, "[") { // [v6]:port:user:pass
		if i := strings.Index(s, "]:"); i > 0 {
			rest := strings.SplitN(s[i+2:], ":", 3)
			if port, ok := portOf(rest[0]); ok && len(rest) == 3 {
				p.Host, p.Port, p.User, p.Pass = s[1:i], port, rest[1], rest[2]
				return p, nil
			}
		}
	}
	parts := strings.Split(s, ":")
	if len(parts) >= 4 {
		if port, ok := portOf(parts[1]); ok && hostLike(parts[0]) {
			p.Host, p.Port, p.User, p.Pass = parts[0], port, parts[2], strings.Join(parts[3:], ":")
			return p, nil
		}
		n := len(parts)
		if port, ok := portOf(parts[n-1]); ok && hostLike(parts[n-2]) {
			p.Host, p.Port, p.User, p.Pass = parts[n-2], port, parts[0], strings.Join(parts[1:n-2], ":")
			return p, nil
		}
	}
	if len(parts) == 3 { // host:port:user (no password)
		if port, ok := portOf(parts[1]); ok && hostLike(parts[0]) {
			p.Host, p.Port, p.User = parts[0], port, parts[2]
			return p, nil
		}
	}
	return p, fmt.Errorf("无法识别格式")
}

// ParseProxyLine parses one line of pasted text. Besides the proxy itself a
// line may carry a protocol word, a change-IP link and a trailing "# name".
func ParseProxyLine(line, defType string) (Proxy, error) {
	line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
	if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
		return Proxy{}, fmt.Errorf("empty")
	}
	name := ""
	if i := strings.Index(line, " #"); i > 0 {
		name, line = strings.TrimSpace(line[i+2:]), strings.TrimSpace(line[:i])
	}
	toks := reSplit.Split(line, -1)
	var core, changeURL, proto string
	var rest []string
	for _, t := range toks {
		switch {
		case t == "":
		case isProtoWord(t):
			proto = normType(t)
		case core == "" && (reScheme.MatchString(t) || strings.Contains(t, ":")):
			// "http://host:port" is the proxy; a later "https://..." with a path is a change-IP link.
			if strings.Count(t, "/") > 2 && !strings.Contains(t, "@") {
				if _, err := parseCore(t); err != nil {
					changeURL = t
					continue
				}
			}
			core = t
		case strings.HasPrefix(strings.ToLower(t), "http://") || strings.HasPrefix(strings.ToLower(t), "https://"):
			changeURL = t
		default:
			rest = append(rest, t)
		}
	}
	if core == "" && len(rest) >= 2 { // "host port [user pass]"
		core, rest = strings.Join(rest, ":"), nil
	}
	if core == "" {
		return Proxy{}, fmt.Errorf("找不到 主机:端口")
	}
	p, err := parseCore(core)
	if err != nil {
		return p, err
	}
	if p.User == "" && len(rest) >= 1 { // "host:port user pass"
		p.User = rest[0]
		if len(rest) >= 2 {
			p.Pass = rest[1]
		}
	}
	if p.Type == "" {
		p.Type = proto
	}
	if p.Type == "" {
		if defType != "" && defType != "auto" {
			p.Type = normType(defType)
		} else {
			p.Type, p.Guessed = guessType(p.Port), true
		}
	}
	if p.Type == "socks4" {
		return p, fmt.Errorf("暂不支持 SOCKS4")
	}
	p.ChangeIP = changeURL
	p.Name = name
	if p.Name == "" {
		p.Name = fmt.Sprintf("%s:%d", p.Host, p.Port)
	}
	return p, nil
}

// ParseProxy keeps the old single-line entry point.
func ParseProxy(line string) (Proxy, error) { return ParseProxyLine(line, "http") }

type ProxyPreview struct {
	Line  int    `json:"line"`
	Raw   string `json:"raw"`
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	Dup   bool   `json:"dup"` // already saved, or repeated earlier in the paste
	Proxy Proxy  `json:"proxy"`
}

func proxyKey(p Proxy) string {
	return strings.ToLower(fmt.Sprintf("%s:%d:%s", p.Host, p.Port, p.User))
}

// PreviewProxyText parses pasted text (or a .txt/.csv file) line by line.
// It also accepts a JSON array of proxy objects or strings.
func PreviewProxyText(text, defType string, existing []Proxy) []ProxyPreview {
	seen := map[string]bool{}
	for _, p := range existing {
		seen[proxyKey(p)] = true
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r", ""), "\n")
	if t := strings.TrimSpace(text); strings.HasPrefix(t, "[") {
		var arr []json.RawMessage
		if json.Unmarshal([]byte(t), &arr) == nil {
			lines = lines[:0]
			for _, r := range arr {
				var s string
				if json.Unmarshal(r, &s) == nil {
					lines = append(lines, s)
					continue
				}
				var o map[string]any
				if json.Unmarshal(r, &o) == nil {
					g := func(ks ...string) string {
						for _, k := range ks {
							if v, ok := o[k]; ok && v != nil {
								return strings.TrimSpace(fmt.Sprint(v))
							}
						}
						return ""
					}
					l := fmt.Sprintf("%s:%s", g("host", "ip", "server", "address"), g("port"))
					if u := g("user", "username", "login"); u != "" {
						l = u + ":" + g("pass", "password") + "@" + l
					}
					if t := g("type", "protocol", "scheme"); t != "" {
						l = normType(t) + "://" + l
					}
					lines = append(lines, l)
				}
			}
		}
	}
	out := []ProxyPreview{}
	for i, l := range lines {
		raw := strings.TrimSpace(l)
		if raw == "" || strings.HasPrefix(raw, "#") || strings.HasPrefix(raw, "//") {
			continue
		}
		// skip a CSV header row
		if i == 0 && regexp.MustCompile(`(?i)^(type|host|ip|proxy|protocol)\b`).MatchString(raw) && !strings.Contains(raw, ".") {
			continue
		}
		pv := ProxyPreview{Line: i + 1, Raw: raw}
		p, err := ParseProxyLine(raw, defType)
		if err != nil {
			pv.Error = err.Error()
		} else {
			pv.OK, pv.Proxy = true, p
			k := proxyKey(p)
			pv.Dup = seen[k]
			seen[k] = true
		}
		out = append(out, pv)
	}
	return out
}

func (p Proxy) URL() *url.URL {
	scheme := p.Type
	if scheme == "" || scheme == "https" {
		scheme = "http"
	}
	host := p.Host
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	u := &url.URL{Scheme: scheme, Host: fmt.Sprintf("%s:%d", host, p.Port)}
	if p.User != "" {
		u.User = url.UserPassword(p.User, p.Pass)
	}
	return u
}

func (p Proxy) client(timeout time.Duration) *http.Client {
	tr := &http.Transport{DisableCompression: true, MaxIdleConnsPerHost: 2}
	if p.Type != "none" && p.Host != "" {
		tr.Proxy = http.ProxyURL(p.URL())
	}
	return &http.Client{Transport: tr, Timeout: timeout}
}

type geo struct {
	IP, CC, Region, City, TZ, ISP, Kind string
	Lat, Lon                            float64
}

func geoIPAPI(cl *http.Client) (geo, error) {
	resp, err := cl.Get("http://ip-api.com/json/?fields=status,message,query,countryCode,regionName,city,timezone,lat,lon,isp,org,mobile,proxy,hosting")
	if err != nil {
		return geo{}, err
	}
	defer resp.Body.Close()
	var v struct {
		Status, Message, Query, CountryCode, RegionName, City, Timezone, ISP, Org string
		Lat, Lon                                                                  float64
		Mobile, Hosting                                                           bool
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if json.Unmarshal(b, &v) != nil || v.Status != "success" {
		return geo{}, fmt.Errorf("ip-api: %s %s", resp.Status, v.Message)
	}
	kind := "residential"
	if v.Mobile {
		kind = "mobile"
	} else if v.Hosting {
		kind = "hosting"
	}
	isp := v.ISP
	if isp == "" {
		isp = v.Org
	}
	return geo{v.Query, v.CountryCode, v.RegionName, v.City, v.Timezone, isp, kind, v.Lat, v.Lon}, nil
}

func geoIPWho(cl *http.Client) (geo, error) {
	resp, err := cl.Get("https://ipwho.is/")
	if err != nil {
		return geo{}, err
	}
	defer resp.Body.Close()
	var v struct {
		Success     bool    `json:"success"`
		Message     string  `json:"message"`
		IP          string  `json:"ip"`
		CountryCode string  `json:"country_code"`
		Region      string  `json:"region"`
		City        string  `json:"city"`
		Lat         float64 `json:"latitude"`
		Lon         float64 `json:"longitude"`
		Timezone    struct {
			ID string `json:"id"`
		} `json:"timezone"`
		Connection struct {
			ISP string `json:"isp"`
		} `json:"connection"`
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if json.Unmarshal(b, &v) != nil || !v.Success {
		return geo{}, fmt.Errorf("ipwho.is: %s", v.Message)
	}
	return geo{v.IP, v.CountryCode, v.Region, v.City, v.Timezone.ID, v.Connection.ISP, "", v.Lat, v.Lon}, nil
}

func probe(p Proxy) (Proxy, error) {
	cl := p.client(15 * time.Second)
	start := time.Now()
	g, err := geoIPAPI(cl)
	if err != nil {
		var e2 error
		if g, e2 = geoIPWho(cl); e2 != nil {
			return p, err
		}
	}
	p.Connect = time.Since(start).Milliseconds()
	// Warm round trips over the same tunnel: the latency a page would feel.
	best := int64(0)
	for i := 0; i < 2; i++ {
		t := time.Now()
		resp, err := cl.Get("http://www.gstatic.com/generate_204")
		if err != nil {
			break
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if ms := time.Since(t).Milliseconds(); best == 0 || ms < best {
			best = ms
		}
	}
	if best == 0 {
		best = p.Connect
	}
	p.Latency = best
	p.IP, p.Country, p.Region, p.City, p.Timezone, p.ISP, p.Lat, p.Lon = g.IP, g.CC, g.Region, g.City, g.TZ, g.ISP, g.Lat, g.Lon
	if g.Kind != "" {
		p.IPType = g.Kind
	}
	return p, nil
}

// CheckProxy connects through the proxy and reads exit IP, country, region,
// city, ISP, IP type, timezone and latency. When the protocol was only guessed
// it also tries the other one and keeps whichever works.
// A Proxy with Type "none" checks the direct connection.
func CheckProxy(p Proxy) Proxy {
	p.Checked = time.Now().Unix()
	if p.Guessed && p.Type != "none" {
		if t := sniffProto(p.Host, p.Port); t != "" {
			p.Type = t
		}
	}
	r, err := probe(p)
	if err != nil && p.Guessed && p.Type != "none" {
		alt := p
		if p.Type == "socks5" {
			alt.Type = "http"
		} else {
			alt.Type = "socks5"
		}
		if r2, e2 := probe(alt); e2 == nil {
			r, err = r2, nil
		}
	}
	if err != nil {
		p.Status, p.Error = "fail", shortErr(err)
		return p
	}
	r.Guessed, r.Status, r.Error = false, "ok", ""
	return r
}

// sniffProto sends a SOCKS5 greeting: a SOCKS5 server answers with 0x05,
// an HTTP proxy answers with "HTTP/" or closes. "" when the port is dead.
func sniffProto(host string, port int) string {
	c, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 5*time.Second)
	if err != nil {
		return ""
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(4 * time.Second))
	if _, err := c.Write([]byte{5, 2, 0, 2}); err != nil {
		return ""
	}
	b := make([]byte, 2)
	if n, _ := io.ReadAtLeast(c, b, 1); n >= 1 && b[0] == 5 {
		return "socks5"
	}
	return "http"
}

// SpeedTestProxy downloads a few MB through the proxy and records the speed.
func SpeedTestProxy(p Proxy) Proxy {
	cl := p.client(25 * time.Second)
	p.SpeedAt = time.Now().Unix()
	start := time.Now()
	resp, err := cl.Get("https://speed.cloudflare.com/__down?bytes=5000000")
	if err != nil {
		p.Speed, p.Error = 0, shortErr(err)
		return p
	}
	defer resp.Body.Close()
	n, _ := io.Copy(io.Discard, resp.Body)
	sec := time.Since(start).Seconds()
	if n < 100000 || sec <= 0 {
		p.Speed, p.Error = 0, "测速失败："+resp.Status
		return p
	}
	p.Speed = int64(float64(n) * 8 / 1000 / sec)
	return p
}

func shortErr(err error) string {
	s := err.Error()
	for _, k := range []struct{ in, out string }{
		{"407", "认证失败（用户名或密码错误）"}, {"Proxy Authentication Required", "认证失败（用户名或密码错误）"},
		{"connection refused", "连接被拒绝（端口或协议不对）"}, {"i/o timeout", "连接超时"}, {"deadline exceeded", "连接超时"},
		{"no such host", "域名解析失败"}, {"username/password authentication failed", "认证失败（用户名或密码错误）"},
		{"malformed HTTP response", "协议不匹配（试试 SOCKS5）"}, {"EOF", "连接被代理关闭"},
	} {
		if strings.Contains(s, k.in) {
			return k.out
		}
	}
	if len(s) > 120 {
		s = s[:120]
	}
	return s
}

// runPool runs fn over items with bounded concurrency.
func runPool[T any](items []T, n int, fn func(T)) {
	sem := make(chan struct{}, n)
	done := make(chan struct{})
	for _, it := range items {
		sem <- struct{}{}
		go func(it T) { defer func() { <-sem; done <- struct{}{} }(); fn(it) }(it)
	}
	for range items {
		<-done
	}
}
