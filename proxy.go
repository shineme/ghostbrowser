package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ParseProxy accepts the common formats:
//
//	host:port
//	host:port:user:pass
//	user:pass@host:port
//	socks5://user:pass@host:port
func ParseProxy(line string) (Proxy, error) {
	line = strings.TrimSpace(line)
	p := Proxy{Type: "http"}
	if line == "" {
		return p, fmt.Errorf("empty")
	}
	if i := strings.Index(line, "://"); i > 0 {
		p.Type = strings.ToLower(line[:i])
		if p.Type == "socks" || p.Type == "socks5h" {
			p.Type = "socks5"
		}
		line = line[i+3:]
	}
	if at := strings.LastIndex(line, "@"); at > 0 {
		cred := line[:at]
		line = line[at+1:]
		if c := strings.SplitN(cred, ":", 2); len(c) == 2 {
			p.User, p.Pass = c[0], c[1]
		}
	}
	parts := strings.Split(line, ":")
	if len(parts) < 2 {
		return p, fmt.Errorf("bad proxy: %s", line)
	}
	p.Host = parts[0]
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return p, fmt.Errorf("bad port")
	}
	p.Port = port
	if len(parts) >= 4 && p.User == "" {
		p.User, p.Pass = parts[2], strings.Join(parts[3:], ":")
	}
	p.Name = fmt.Sprintf("%s:%d", p.Host, p.Port)
	return p, nil
}

func (p Proxy) URL() *url.URL {
	scheme := p.Type
	if scheme == "" || scheme == "https" {
		scheme = "http"
	}
	u := &url.URL{Scheme: scheme, Host: fmt.Sprintf("%s:%d", p.Host, p.Port)}
	if p.User != "" {
		u.User = url.UserPassword(p.User, p.Pass)
	}
	return u
}

type ipInfo struct {
	Status      string  `json:"status"`
	Query       string  `json:"query"`
	CountryCode string  `json:"countryCode"`
	City        string  `json:"city"`
	Timezone    string  `json:"timezone"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
}

// CheckProxy connects through the proxy and reads exit IP + geo + timezone.
// A Proxy with Type "none" checks the direct connection.
func CheckProxy(p Proxy) Proxy {
	tr := &http.Transport{}
	if p.Type != "none" && p.Host != "" {
		tr.Proxy = http.ProxyURL(p.URL())
	}
	cl := &http.Client{Transport: tr, Timeout: 15 * time.Second}
	start := time.Now()
	resp, err := cl.Get("http://ip-api.com/json/?fields=status,query,countryCode,city,timezone,lat,lon")
	p.Checked = time.Now().Unix()
	if err != nil {
		p.Status = "fail"
		return p
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var info ipInfo
	if json.Unmarshal(b, &info) != nil || info.Status != "success" {
		p.Status = "fail"
		return p
	}
	p.Latency = time.Since(start).Milliseconds()
	p.Status = "ok"
	p.IP, p.Country, p.City, p.Timezone, p.Lat, p.Lon = info.Query, info.CountryCode, info.City, info.Timezone, info.Lat, info.Lon
	return p
}
