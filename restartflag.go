package main

// 待重启: a running browser keeps the proxy, fingerprint, kernel and launch
// arguments it was started with. When the user edits any of those while it
// runs, the profile is flagged "待重启" until it is restarted. Name, group,
// tags and notes are not launch-relevant and never set the flag.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"
)

type launchTracker struct {
	mu   sync.Mutex
	hash map[string]string // profile ID -> launch config hash at launch time
	url  map[string]string // one-shot start URL for the next launch (网站快捷入口)
}

var launchCfg = &launchTracker{hash: map[string]string{}, url: map[string]string{}}

// launchProxyKey keeps only what reaches the browser: type, endpoint and
// credentials. Check results (exit IP, latency, country) change on every
// proxy check and must not flag running profiles.
func launchProxyKey(p *Proxy) string {
	if p == nil {
		return ""
	}
	return strings.Join([]string{p.Type, p.Host, itoa(p.Port), p.User, p.Pass}, "\x1f")
}

// launchConfigHash hashes the launch-relevant part of a profile. The proxy
// assigned by ID is resolved so editing the proxy entry itself also counts.
func launchConfigHash(s *Store, p Profile) string {
	type native struct {
		Template, Source string
		Args             []string
		Headless         bool
	}
	doc := struct {
		Kernel      string
		ProxyID     string
		Proxy       string
		Inline      string
		Fingerprint Fingerprint
		Native      *native
		Extensions  []string
	}{Kernel: p.Kernel, ProxyID: p.ProxyID, Inline: launchProxyKey(p.InlineProxy), Fingerprint: p.Fingerprint}
	if p.ProxyID != "" && s != nil {
		if px, ok := s.GetProxy(p.ProxyID); ok {
			doc.Proxy = launchProxyKey(px)
		}
	}
	if p.Native != nil {
		doc.Native = &native{Template: p.Native.TemplateKey, Source: p.Native.SourceConfigID, Args: p.Native.ExtraArgs, Headless: p.Native.Headless}
	}
	ext := append([]string(nil), p.ExtensionIDs...)
	sort.Strings(ext)
	doc.Extensions = ext
	b, _ := json.Marshal(doc)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:12])
}

// launched records the config a profile was started with.
func (l *launchTracker) launched(s *Store, id string) {
	p, ok := s.GetProfile(id)
	if !ok {
		return
	}
	h := launchConfigHash(s, *p)
	l.mu.Lock()
	l.hash[id] = h
	l.mu.Unlock()
}

func (l *launchTracker) forget(id string) {
	l.mu.Lock()
	delete(l.hash, id)
	l.mu.Unlock()
}

// setURL arms a one-shot start URL used by the next launch of id.
func (l *launchTracker) setURL(id, u string) {
	l.mu.Lock()
	l.url[id] = u
	l.mu.Unlock()
}

// takeURL returns and clears the one-shot start URL.
func (l *launchTracker) takeURL(id string) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	u := l.url[id]
	delete(l.url, id)
	return u
}

// pendingRestart lists running profiles whose saved launch config differs
// from the one they were started with.
func (s *Store) pendingRestart() []string {
	out := []string{}
	for _, rs := range runReg.list() {
		launchCfg.mu.Lock()
		h, ok := launchCfg.hash[rs.ProfileID]
		launchCfg.mu.Unlock()
		if !ok {
			continue
		}
		p, found := s.GetProfile(rs.ProfileID)
		if !found {
			continue
		}
		if launchConfigHash(s, *p) != h {
			out = append(out, rs.ProfileID)
		}
	}
	sort.Strings(out)
	return out
}

// waitStopped waits until a profile is no longer in the run registry.
func waitStopped(id string, max time.Duration) bool {
	end := time.Now().Add(max)
	for time.Now().Before(end) {
		if _, ok := runReg.get(id); !ok {
			return true
		}
		time.Sleep(150 * time.Millisecond)
	}
	_, ok := runReg.get(id)
	return !ok
}
