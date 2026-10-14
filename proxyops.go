package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ---------------------------------------------------------------- change IP

// rotateProxy calls the proxy's change-IP URL, waits for the new exit to come
// up, then checks the proxy again.
func (s *Store) rotateProxy(id string) (Proxy, error) {
	px, ok := s.GetProxy(id)
	if !ok {
		return Proxy{}, fmt.Errorf("proxy not found")
	}
	u := strings.TrimSpace(px.ChangeIP)
	if u == "" {
		return *px, fmt.Errorf("这个代理没有填写换 IP 链接")
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return *px, fmt.Errorf("换 IP 链接必须以 http:// 或 https:// 开头")
	}
	old := px.IP
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Get(u)
	if err != nil {
		return *px, fmt.Errorf("换 IP 请求失败: %w", err)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return *px, fmt.Errorf("换 IP 链接返回 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var r Proxy
	for i := 0; i < 4; i++ { // providers take a few seconds to switch
		time.Sleep(3 * time.Second)
		r = CheckProxy(*px)
		if r.Status == "ok" && (old == "" || r.IP != old) {
			break
		}
	}
	if cur, ok := s.GetProxy(id); ok {
		r.Name, r.Tag, r.ChangeIP, r.Speed, r.SpeedAt, r.RotateOnLaunch = cur.Name, cur.Tag, cur.ChangeIP, cur.Speed, cur.SpeedAt, cur.RotateOnLaunch
	}
	r = s.PutProxy(r)
	if r.Status != "ok" {
		return r, fmt.Errorf("换 IP 后代理不可用: %s", r.Error)
	}
	if old != "" && r.IP == old {
		return r, fmt.Errorf("换 IP 后出口仍是 %s（服务商可能还没切换，稍后再检测）", old)
	}
	return r, nil
}

func (a *App) RotateProxy(id string) (Proxy, error) {
	r, err := a.store.rotateProxy(id)
	if r.ID != "" {
		a.emitProxy(r)
	}
	return r, err
}

// rotateBeforeLaunch changes the exit IP of a profile's proxy when the proxy
// asks for it. A failure does not block the launch; the geo check still runs.
func (s *Store) rotateBeforeLaunch(p *Profile) {
	if p.ProxyID == "" {
		return
	}
	if px, ok := s.GetProxy(p.ProxyID); ok && px.RotateOnLaunch && px.ChangeIP != "" {
		_, _ = s.rotateProxy(px.ID)
	}
}

// ---------------------------------------------------------------- health

// proxyHealthLoop re-checks every proxy on the interval set in Settings.
func (a *App) proxyHealthLoop() {
	last := time.Now()
	for {
		time.Sleep(time.Minute)
		mins := a.store.GetSettings().ProxyCheckMins
		if mins <= 0 || time.Since(last) < time.Duration(mins)*time.Minute {
			continue
		}
		last = time.Now()
		if len(a.store.ListProxies()) == 0 {
			continue
		}
		list := a.CheckProxies(nil)
		bad := 0
		for _, p := range list {
			if p.Status == "fail" {
				bad++
			}
		}
		if a.ctx != nil {
			wruntime.EventsEmit(a.ctx, "proxy:health", map[string]any{"checked": len(list), "failed": bad, "at": time.Now().Unix()})
		}
	}
}

// ---------------------------------------------------------------- bulk

// AssignProxies gives each profile one of the proxies: "order" cycles through
// them, "random" picks at random, "one" uses the first for all. An empty
// proxy list switches the profiles to a direct connection.
func (a *App) AssignProxies(profileIDs, proxyIDs []string, mode string) (int, error) {
	var px []string
	for _, id := range proxyIDs {
		if _, ok := a.store.GetProxy(id); ok {
			px = append(px, id)
		}
	}
	if len(proxyIDs) > 0 && len(px) == 0 {
		return 0, fmt.Errorf("选中的代理不存在")
	}
	n := 0
	perm := rand.Perm(len(profileIDs))
	for i, id := range profileIDs {
		p, ok := a.store.GetProfile(id)
		if !ok {
			continue
		}
		p.InlineProxy = nil
		p.ProxyID = ""
		if len(px) > 0 {
			switch mode {
			case "random":
				p.ProxyID = px[rand.Intn(len(px))]
				if len(profileIDs) <= len(px) { // no repeats when there are enough
					p.ProxyID = px[perm[i]%len(px)]
				}
			case "one":
				p.ProxyID = px[0]
			default:
				p.ProxyID = px[i%len(px)]
			}
		}
		a.store.PutProfile(*p)
		a.store.auditNamed("ui", "proxy.assign", p.ID, p.Name, nil, p.ProxyID)
		n++
	}
	return n, nil
}

// SetProfilesGroup moves profiles into a group and optionally adds a tag.
func (a *App) SetProfilesGroup(ids []string, group, addTag string) int {
	n := 0
	addTag = strings.TrimSpace(addTag)
	for _, id := range ids {
		p, ok := a.store.GetProfile(id)
		if !ok {
			continue
		}
		if group != "\x00" {
			p.Group = strings.TrimSpace(group)
		}
		if addTag != "" {
			has := false
			for _, t := range p.Tags {
				if t == addTag {
					has = true
				}
			}
			if !has {
				p.Tags = append(p.Tags, addTag)
			}
		}
		a.store.PutProfile(*p)
		a.store.auditNamed("ui", "group.set", p.ID, p.Name, nil, strings.TrimSpace(p.Group+" "+addTag))
		n++
	}
	return n
}

// StopProfiles stops several running profiles.
func (a *App) StopProfiles(ids []string) {
	for _, id := range ids {
		a.StopProfile(id)
	}
}

type BulkResult struct {
	OK     int      `json:"ok"`
	Failed int      `json:"failed"`
	Errors []string `json:"errors"`
}

// PushProfiles uploads several stopped profiles now.
func (a *App) PushProfiles(ids []string) BulkResult {
	var r BulkResult
	for _, id := range ids {
		if _, running := runReg.get(id); running {
			r.Failed++
			r.Errors = append(r.Errors, a.profileName(id)+": 正在运行，关闭后会自动上传")
			continue
		}
		if e := a.store.PushProfile(id); e != nil {
			r.Failed++
			r.Errors = append(r.Errors, a.profileName(id)+": "+e.Error())
			continue
		}
		r.OK++
	}
	return r
}

func (a *App) profileName(id string) string {
	if p, ok := a.store.GetProfile(id); ok {
		return p.Name
	}
	return id
}

// ---------------------------------------------------------------- running

// FocusProfile brings a running profile's browser window to the front.
func (a *App) FocusProfile(id string) error {
	rs, ok := runReg.get(id)
	if !ok {
		return fmt.Errorf("环境没有在运行")
	}
	if rs.Kernel == "firefox" {
		return focusFirefox(id)
	}
	return focusBrowser(rs.DebugPort, rs.WSURL)
}
