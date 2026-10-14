package main

// Remote site shortcuts catalog + announcements, served by the sync server
// (ghostsync.php ?r=/api/catalog from catalog.json) or Settings.CatalogURL.
// The last good copy is cached in the data folder so it works offline.

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CatalogSite struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Icon string `json:"icon"`
}

type CatalogGroup struct {
	ID    string        `json:"id"`
	Name  string        `json:"name"`
	Sites []CatalogSite `json:"sites"`
}

// flexTime accepts a unix number, an RFC 3339 string, "2006-01-02" or "".
type flexTime int64

func (f *flexTime) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" || s == `""` {
		*f = 0
		return nil
	}
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		*f = flexTime(int64(n))
		return nil
	}
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, strings.TrimSpace(str), time.Local); err == nil {
			*f = flexTime(t.Unix())
			return nil
		}
	}
	*f = 0
	return nil
}

type Announcement struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Body  string   `json:"body"`  // markdown; the UI renders it without raw HTML
	Level string   `json:"level"` // info | warn | critical
	Start flexTime `json:"start"`
	End   flexTime `json:"end"`
}

type Catalog struct {
	Revision      int64          `json:"revision"`
	Groups        []CatalogGroup `json:"groups"`
	Announcements []Announcement `json:"announcements"`
}

// CatalogState is what the UI gets.
type CatalogState struct {
	Catalog   Catalog  `json:"catalog"`
	FetchedAt int64    `json:"fetchedAt"` // when the cached copy was fetched (0 = never)
	Offline   bool     `json:"offline"`   // last refresh failed, showing the cache
	Error     string   `json:"error,omitempty"`
	Read      []string `json:"read"` // announcement IDs already read
	Source    string   `json:"source"`
}

type catalogCache struct {
	Catalog   Catalog  `json:"catalog"`
	FetchedAt int64    `json:"fetchedAt"`
	Source    string   `json:"source"`
	Read      []string `json:"read"`
}

type catalogSvc struct {
	mu      sync.Mutex
	path    string
	cache   catalogCache
	loaded  bool
	offline bool
	err     string
}

var catalogs = &catalogSvc{}

func safeSiteURL(u string) bool {
	p, err := url.Parse(strings.TrimSpace(u))
	return err == nil && (p.Scheme == "https" || p.Scheme == "http") && p.Host != ""
}

// parseCatalog validates a catalog document: unsafe site URLs (javascript:,
// file:) are dropped, empty groups removed, levels normalised, and
// announcements sorted newest first.
func parseCatalog(raw []byte) (Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(raw, &c); err != nil {
		return Catalog{}, errors.New("catalog.json 格式不对")
	}
	groups := []CatalogGroup{}
	for gi, g := range c.Groups {
		sites := []CatalogSite{}
		for si, s := range g.Sites {
			s.URL = strings.TrimSpace(s.URL)
			if !safeSiteURL(s.URL) {
				continue
			}
			if s.ID == "" {
				s.ID = strconv.Itoa(gi) + "-" + strconv.Itoa(si)
			}
			if strings.TrimSpace(s.Name) == "" {
				if p, err := url.Parse(s.URL); err == nil {
					s.Name = p.Host
				}
			}
			if s.Icon != "" && !safeSiteURL(s.Icon) && !strings.HasPrefix(s.Icon, "data:image/") {
				s.Icon = ""
			}
			sites = append(sites, s)
		}
		if len(sites) == 0 {
			continue
		}
		g.Sites = sites
		if g.ID == "" {
			g.ID = "g" + strconv.Itoa(gi)
		}
		groups = append(groups, g)
	}
	c.Groups = groups
	ann := []Announcement{}
	for i, a := range c.Announcements {
		if strings.TrimSpace(a.Title) == "" && strings.TrimSpace(a.Body) == "" {
			continue
		}
		if a.ID == "" {
			a.ID = "a" + strconv.Itoa(i)
		}
		switch a.Level {
		case "warn", "critical":
		default:
			a.Level = "info"
		}
		ann = append(ann, a)
	}
	sort.SliceStable(ann, func(i, j int) bool { return ann[i].Start > ann[j].Start })
	c.Announcements = ann
	return c, nil
}

// activeAnnouncements keeps those inside their start/end window.
func activeAnnouncements(list []Announcement, now int64) []Announcement {
	out := []Announcement{}
	for _, a := range list {
		if a.Start > 0 && int64(a.Start) > now {
			continue
		}
		if a.End > 0 && int64(a.End) < now {
			continue
		}
		out = append(out, a)
	}
	return out
}

func (c *catalogSvc) load(dir string) {
	if c.loaded && c.path != "" {
		return
	}
	c.path = filepath.Join(dir, "catalog-cache.json")
	c.loaded = true
	if b, err := os.ReadFile(c.path); err == nil {
		_ = json.Unmarshal(b, &c.cache)
	}
}

func (c *catalogSvc) save() {
	if c.path == "" {
		return
	}
	b, _ := json.MarshalIndent(c.cache, "", "  ")
	tmp := c.path + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, c.path)
	}
}

func (c *catalogSvc) state() CatalogState {
	cat := c.cache.Catalog
	cat.Announcements = activeAnnouncements(cat.Announcements, time.Now().Unix())
	if cat.Groups == nil {
		cat.Groups = []CatalogGroup{}
	}
	read := append([]string{}, c.cache.Read...)
	return CatalogState{Catalog: cat, FetchedAt: c.cache.FetchedAt, Offline: c.offline, Error: c.err, Read: read, Source: c.cache.Source}
}

// Get returns the cached catalog (no network).
func (c *catalogSvc) Get(dir string) CatalogState {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load(dir)
	return c.state()
}

func catalogSource(st Settings) (string, string, string) {
	if s := strings.TrimSpace(st.CatalogURL); s != "" {
		tok := ""
		if st.SyncServer != "" && sameHost(s, st.SyncServer) {
			tok = st.SyncToken
		}
		return s, tok, "custom"
	}
	if st.SyncServer != "" && st.SyncToken != "" {
		return syncURL(strings.TrimRight(st.SyncServer, "/"), "/api/catalog"), st.SyncToken, "server"
	}
	return "", "", ""
}

// Refresh fetches the catalog; on failure the cache stays and Offline is set.
func (c *catalogSvc) Refresh(dir string, st Settings) CatalogState {
	link, tok, src := catalogSource(st)
	var raw json.RawMessage
	var err error
	if link == "" {
		err = errors.New("没有配置同步服务器，快捷入口与公告使用本地缓存")
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err = fetchJSON(ctx, upd.hc, link, tok, &raw)
		cancel()
	}
	var cat Catalog
	if err == nil {
		cat, err = parseCatalog(raw)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load(dir)
	if err != nil {
		c.offline, c.err = true, err.Error()
		return c.state()
	}
	c.offline, c.err = false, ""
	c.cache.Catalog, c.cache.FetchedAt, c.cache.Source = cat, time.Now().Unix(), src
	// forget read marks of announcements that no longer exist
	live := map[string]bool{}
	for _, a := range cat.Announcements {
		live[a.ID] = true
	}
	keep := []string{}
	for _, id := range c.cache.Read {
		if live[id] {
			keep = append(keep, id)
		}
	}
	c.cache.Read = keep
	c.save()
	return c.state()
}

// MarkRead records announcement IDs as read.
func (c *catalogSvc) MarkRead(dir string, ids []string) CatalogState {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load(dir)
	have := map[string]bool{}
	for _, id := range c.cache.Read {
		have[id] = true
	}
	for _, id := range ids {
		if id != "" && !have[id] {
			have[id] = true
			c.cache.Read = append(c.cache.Read, id)
		}
	}
	c.save()
	return c.state()
}
