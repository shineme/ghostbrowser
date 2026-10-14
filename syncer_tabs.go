package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Window / tab actions of the 多窗口同步器 dialog. They act on the selected
// master + followers over CDP and work whether or not sync is running (only
// the profiles must be running). While sync runs, tab mirroring is paused
// during an action and the tab mapping is rebuilt afterwards, so an action
// that opens or closes tabs in every window isn't mirrored a second time.

// SyncerPrefs are the dialog's last choices, kept in Settings.
type SyncerPrefs struct {
	SyncOptions
	Master    string   `json:"master"`
	Followers []string `json:"followers"`
	NewTab    bool     `json:"newTab"` // 批量访问网址: open in a new tab
}

// TabActionResult reports what a window/tab action did.
type TabActionResult struct {
	Action string   `json:"action"`
	Done   int      `json:"done"`
	Total  int      `json:"total"`
	Failed []string `json:"failed"` // "环境名：原因"
	URL    string   `json:"url,omitempty"`
}

var schemeRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://`)

// normalizeURL validates a typed address and adds https:// when it has no scheme.
func normalizeURL(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("请输入网址")
	}
	low := strings.ToLower(s)
	if low == "about:blank" || strings.HasPrefix(low, "chrome://") || strings.HasPrefix(low, "edge://") {
		return s, nil
	}
	if strings.ContainsAny(s, " \t\r\n") {
		return "", fmt.Errorf("网址里不能有空格")
	}
	if strings.HasPrefix(s, "//") {
		s = "https:" + s
	} else if !schemeRe.MatchString(s) {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("网址格式不正确")
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	case "file":
		return u.String(), nil
	default:
		return "", fmt.Errorf("只支持 http / https 网址")
	}
	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("网址缺少域名")
	}
	if host != "localhost" && !strings.Contains(host, ".") && !strings.Contains(host, ":") {
		return "", fmt.Errorf("网址格式不正确")
	}
	if strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") || strings.Contains(host, "..") {
		return "", fmt.Errorf("网址格式不正确")
	}
	return u.String(), nil
}

// isBlankURL: an empty tab (about:blank or the new tab page).
func isBlankURL(u string) bool {
	u = strings.ToLower(strings.TrimSpace(u))
	switch u {
	case "", "about:blank", "chrome://newtab/", "chrome://newtab", "chrome://new-tab-page/", "edge://newtab/":
		return true
	}
	return strings.HasPrefix(u, "chrome-search://local-ntp") || strings.HasPrefix(u, "chrome://new-tab-page")
}

// browserTabs is one connected browser for a tab action.
type browserTabs struct {
	c    *cdpConn
	tabs []pageTab
}

func openBrowserTabs(ws string) (*browserTabs, error) {
	c, err := dialCDP(ws, nil)
	if err != nil {
		return nil, err
	}
	tabs, err := pageTabs(c)
	if err != nil {
		c.c.Close()
		return nil, err
	}
	if len(tabs) == 0 {
		c.c.Close()
		return nil, fmt.Errorf("浏览器没有打开的标签页")
	}
	return &browserTabs{c: c, tabs: tabs}, nil
}

func (b *browserTabs) close() { b.c.c.Close() }

func (b *browserTabs) active() string { return activeOf(b.c, b.tabs) }

func (b *browserTabs) closeTab(tid string) error {
	_, err := b.c.call("", "Target.closeTarget", map[string]any{"targetId": tid})
	return err
}

func (b *browserTabs) navigate(tid, u string) error {
	sid, err := attachTarget(b.c, tid)
	if err != nil {
		return err
	}
	defer b.c.call("", "Target.detachFromTarget", map[string]any{"sessionId": sid})
	raw, err := b.c.call(sid, "Page.navigate", map[string]any{"url": u})
	if err != nil {
		return err
	}
	var r struct {
		ErrorText string `json:"errorText"`
	}
	_ = json.Unmarshal(raw, &r)
	if r.ErrorText != "" && r.ErrorText != "net::ERR_ABORTED" {
		return fmt.Errorf("打开失败：%s", r.ErrorText)
	}
	return nil
}

func (b *browserTabs) newTab(u string) (string, error) {
	raw, err := b.c.call("", "Target.createTarget", map[string]any{"url": u})
	if err != nil {
		return "", err
	}
	var r struct {
		TargetID string `json:"targetId"`
	}
	_ = json.Unmarshal(raw, &r)
	return r.TargetID, nil
}

func (b *browserTabs) activate(tid string) error {
	_, err := b.c.call("", "Target.activateTarget", map[string]any{"targetId": tid})
	return err
}

// window returns the browser window of a tab and its bounds.
func (b *browserTabs) window(tid string) (int, Rect, string, error) {
	raw, err := b.c.call("", "Browser.getWindowForTarget", map[string]any{"targetId": tid})
	if err != nil {
		return 0, Rect{}, "", err
	}
	var w struct {
		WindowID int `json:"windowId"`
		Bounds   struct {
			Left, Top, Width, Height int
			WindowState              string `json:"windowState"`
		} `json:"bounds"`
	}
	_ = json.Unmarshal(raw, &w)
	return w.WindowID, Rect{w.Bounds.Left, w.Bounds.Top, w.Bounds.Width, w.Bounds.Height}, w.Bounds.WindowState, nil
}

// ---------- single-browser actions ----------

// closeCurrentTab closes the active tab; the last tab is blanked instead so
// the window stays open.
func (b *browserTabs) closeCurrentTab() error {
	act := b.active()
	if len(b.tabs) <= 1 {
		return b.navigate(act, "about:blank")
	}
	return b.closeTab(act)
}

func (b *browserTabs) closeOtherTabs() error {
	act := b.active()
	var first error
	for _, t := range b.tabs {
		if t.TargetID != act {
			if err := b.closeTab(t.TargetID); err != nil && first == nil {
				first = err
			}
		}
	}
	return first
}

// closeBlankTabs closes empty tabs, keeping one tab when every tab is empty.
func (b *browserTabs) closeBlankTabs() error {
	var blank []pageTab
	for _, t := range b.tabs {
		if isBlankURL(t.URL) {
			blank = append(blank, t)
		}
	}
	if len(blank) == len(b.tabs) && len(blank) > 0 {
		keep := b.active()
		out := blank[:0]
		for _, t := range blank {
			if t.TargetID != keep {
				out = append(out, t)
			}
		}
		blank = out
	}
	var first error
	for _, t := range blank {
		if err := b.closeTab(t.TargetID); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (b *browserTabs) openURL(u string, newTab bool) error {
	if newTab {
		_, err := b.newTab(u)
		return err
	}
	return b.navigate(b.active(), u)
}

// unifyTo makes this browser's tabs the given URLs in order and activates
// the tab at index act. CDP has no tab-strip order, so the tabs are opened
// fresh in order (Target.createTarget appends to the strip) and the old ones
// closed afterwards; that is the only way the order is guaranteed.
func (b *browserTabs) unifyTo(urls []string, act int) error {
	if len(urls) == 0 {
		return fmt.Errorf("主控窗口没有标签页")
	}
	ids := []string{}
	var first error
	for _, u := range urls {
		if isBlankURL(u) && !strings.HasPrefix(u, "about:") {
			u = "about:blank" // the new tab page can't always be opened over CDP
		}
		id, err := b.newTab(u)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return first
	}
	for _, t := range b.tabs {
		if err := b.closeTab(t.TargetID); err != nil && first == nil {
			first = err
		}
	}
	if act < 0 || act >= len(ids) {
		act = 0
	}
	if err := b.activate(ids[act]); err != nil && first == nil {
		first = err
	}
	return first
}

// ---------- multi-browser runner ----------

type tabTarget struct {
	id, ws string
}

func resolveTargets(ids []string, wsOf func(string) (string, bool), name func(string) string, res *TabActionResult) []tabTarget {
	seen := map[string]bool{}
	var out []tabTarget
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		res.Total++
		ws, ok := wsOf(id)
		if !ok {
			res.Failed = append(res.Failed, name(id)+"：没有在运行")
			continue
		}
		out = append(out, tabTarget{id, ws})
	}
	return out
}

// eachBrowser runs fn on every target in parallel.
func eachBrowser(ts []tabTarget, name func(string) string, res *TabActionResult, fn func(id string, b *browserTabs) error) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, t := range ts {
		wg.Add(1)
		go func(t tabTarget) {
			defer wg.Done()
			b, err := openBrowserTabs(t.ws)
			if err == nil {
				err = fn(t.id, b)
				b.close()
			}
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				res.Failed = append(res.Failed, name(t.id)+"："+err.Error())
				return
			}
			res.Done++
		}(t)
	}
	wg.Wait()
}

// syncTabAction runs one action of the dialog.
//   - open (url, newTab), closeCurrent, closeOthers, closeBlank, show: on master + followers
//   - unify, size: master is the source, followers are changed
func syncTabAction(master string, followers []string, action, rawURL string, newTab bool, wsOf func(string) (string, bool), name func(string) string) (TabActionResult, error) {
	res := TabActionResult{Action: action, Failed: []string{}}
	if master == "" && len(followers) == 0 {
		return res, fmt.Errorf("请先选择窗口")
	}
	all := append([]string{master}, followers...)
	if master == "" {
		all = followers
	}
	// Mirroring is paused while tabs change under it; the mapping is rebuilt after.
	if s := groupSync.running(); s != nil {
		resume := s.pauseTabs()
		defer resume()
	}
	switch action {
	case "open":
		u, err := normalizeURL(rawURL)
		if err != nil {
			return res, err
		}
		res.URL = u
		eachBrowser(resolveTargets(all, wsOf, name, &res), name, &res, func(_ string, b *browserTabs) error { return b.openURL(u, newTab) })
	case "closeCurrent":
		eachBrowser(resolveTargets(all, wsOf, name, &res), name, &res, func(_ string, b *browserTabs) error { return b.closeCurrentTab() })
	case "closeOthers":
		eachBrowser(resolveTargets(all, wsOf, name, &res), name, &res, func(_ string, b *browserTabs) error { return b.closeOtherTabs() })
	case "closeBlank":
		eachBrowser(resolveTargets(all, wsOf, name, &res), name, &res, func(_ string, b *browserTabs) error { return b.closeBlankTabs() })
	case "show":
		ts := resolveTargets(all, wsOf, name, &res)
		// followers first, master last so it ends on top
		for i, j := 0, len(ts)-1; i < j; i, j = i+1, j-1 {
			ts[i], ts[j] = ts[j], ts[i]
		}
		for _, t := range ts {
			b, err := openBrowserTabs(t.ws)
			if err == nil {
				err = b.showWindows()
				b.close()
			}
			if err != nil {
				res.Failed = append(res.Failed, name(t.id)+"："+err.Error())
				continue
			}
			res.Done++
			time.Sleep(30 * time.Millisecond)
		}
	case "unify", "size":
		if master == "" {
			return res, fmt.Errorf("请先选择主控窗口")
		}
		if len(followers) == 0 {
			return res, fmt.Errorf("请至少选择一个受控窗口")
		}
		mws, ok := wsOf(master)
		if !ok {
			return res, fmt.Errorf("主控环境没有在运行")
		}
		mb, err := openBrowserTabs(mws)
		if err != nil {
			return res, fmt.Errorf("连接主控环境失败：%v", err)
		}
		act := mb.active()
		urls, ai := []string{}, 0
		for i, t := range mb.tabs {
			if t.TargetID == act {
				ai = i
			}
			urls = append(urls, t.URL)
		}
		_, mr, _, werr := mb.window(act)
		mb.close()
		fs := []string{}
		for _, f := range followers {
			if f != master {
				fs = append(fs, f)
			}
		}
		ts := resolveTargets(fs, wsOf, name, &res)
		if action == "unify" {
			eachBrowser(ts, name, &res, func(_ string, b *browserTabs) error { return b.unifyTo(urls, ai) })
		} else {
			if werr != nil {
				return res, fmt.Errorf("读取主控窗口大小失败：%v", werr)
			}
			eachBrowser(ts, name, &res, func(_ string, b *browserTabs) error { return b.resizeTo(mr.W, mr.H) })
		}
	default:
		return res, fmt.Errorf("未知操作：%s", action)
	}
	return res, nil
}

// resizeTo gives the browser's window the size w×h, keeping its position.
func (b *browserTabs) resizeTo(w, h int) error {
	wid, _, state, err := b.window(b.active())
	if err != nil {
		return err
	}
	if state != "normal" {
		if _, err := b.c.call("", "Browser.setWindowBounds", map[string]any{"windowId": wid, "bounds": map[string]any{"windowState": "normal"}}); err != nil {
			return err
		}
	}
	_, err = b.c.call("", "Browser.setWindowBounds", map[string]any{"windowId": wid, "bounds": map[string]any{"width": w, "height": h}})
	return err
}

// showWindows restores every minimised window of the browser and raises it,
// through each window's own visible tab so no window changes its active tab.
func (b *browserTabs) showWindows() error {
	type win struct {
		id   int
		tabs []string
	}
	var wins []*win
	byID := map[int]*win{}
	var first error
	for _, t := range b.tabs {
		wid, _, state, err := b.window(t.TargetID)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		w := byID[wid]
		if w == nil {
			w = &win{id: wid}
			byID[wid] = w
			wins = append(wins, w)
			if state == "minimized" {
				if _, err := b.c.call("", "Browser.setWindowBounds", map[string]any{"windowId": wid, "bounds": map[string]any{"windowState": "normal"}}); err != nil && first == nil {
					first = err
				}
			}
		}
		w.tabs = append(w.tabs, t.TargetID)
	}
	if len(wins) == 0 {
		return first
	}
	time.Sleep(60 * time.Millisecond)
	act := b.active()
	raise := func(w *win) {
		tid := w.tabs[0]
		for _, id := range w.tabs {
			if id == act {
				tid = id
				break
			}
			if sid, err := attachTarget(b.c, id); err == nil {
				v := tabVisible(b.c, sid)
				_, _ = b.c.call("", "Target.detachFromTarget", map[string]any{"sessionId": sid})
				if v {
					tid = id
					break
				}
			}
		}
		if err := b.activate(tid); err != nil && first == nil {
			first = err
		}
	}
	var last *win
	for _, w := range wins {
		hasAct := false
		for _, id := range w.tabs {
			hasAct = hasAct || id == act
		}
		if hasAct {
			last = w
			continue
		}
		raise(w)
	}
	if last != nil {
		raise(last)
	}
	return first
}

// loadSyncerPrefs returns the saved dialog prefs (defaults when none).
func loadSyncerPrefs(st *Store) SyncerPrefs {
	p := SyncerPrefs{SyncOptions: defaultSyncOptions(), Followers: []string{}}
	if sp := st.GetSettings().Syncer; sp != nil {
		p = *sp
		p.SyncOptions = p.SyncOptions.norm()
		if p.Followers == nil {
			p.Followers = []string{}
		}
	}
	return p
}

// saveSyncerPrefs normalises and stores the dialog prefs.
func saveSyncerPrefs(st *Store, p SyncerPrefs) SyncerPrefs {
	p.SyncOptions = p.SyncOptions.norm()
	seen := map[string]bool{p.Master: true}
	fs := []string{}
	for _, f := range p.Followers {
		if f != "" && !seen[f] {
			seen[f] = true
			fs = append(fs, f)
		}
	}
	p.Followers = fs
	s := st.GetSettings()
	cp := p
	s.Syncer = &cp
	st.PutSettings(s)
	return p
}
