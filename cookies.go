package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Cookie import / export for one profile, in JSON (Cookie-Editor /
// EditThisCookie style) and Netscape cookies.txt.
//
// Running profile: cookies go straight into the browser over CDP
// (Storage.setCookies / Storage.getCookies, the browser-wide counterparts of
// Network.setCookie / Network.getAllCookies, which only exist on page sessions).
//
// Stopped profile: Chrome's Cookies SQLite database is encrypted with a key tied
// to the Windows user (DPAPI / App-Bound Encryption) and locked while in use, so
// it is never edited directly. Imports are staged in Profile.Cookies and written
// over CDP right after the next launch (launchChromium / launchUndetectable
// already do this). Exports of a stopped profile come from the plain session
// snapshot GhostBrowser keeps while the browser runs (ghost-session.json, see
// sessionsync.go) merged with anything still staged.

type Cookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"` // unix seconds; 0 / negative = session cookie
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite,omitempty"` // Lax | Strict | None | ""
}

type CookieImportResult struct {
	Count   int    `json:"count"`
	Skipped int    `json:"skipped"`
	Mode    string `json:"mode"`   // live | staged
	Staged  string `json:"staged"` // the profile's staged JSON after the import (stopped profiles)
}

type CookieInfo struct {
	Running    bool  `json:"running"`
	SnapshotAt int64 `json:"snapshotAt"`
	Snapshot   int   `json:"snapshot"` // cookies in the last session snapshot
	Staged     int   `json:"staged"`   // cookies waiting for the next launch
}

func normSameSite(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "lax":
		return "Lax"
	case "strict":
		return "Strict"
	case "none", "no_restriction":
		return "None"
	}
	return ""
}

func (c *Cookie) normalize() bool {
	c.Name = strings.TrimSpace(c.Name)
	c.Domain = strings.TrimSpace(c.Domain)
	if c.Name == "" || c.Domain == "" {
		return false
	}
	if c.Path == "" {
		c.Path = "/"
	}
	if c.Expires < 0 || math.IsNaN(c.Expires) || math.IsInf(c.Expires, 0) {
		c.Expires = 0
	}
	c.SameSite = normSameSite(c.SameSite)
	return true
}

// parseCookiesJSON accepts arrays from Cookie-Editor, EditThisCookie, CDP
// (expires) and Puppeteer, or an object with a "cookies" array.
func parseCookiesJSON(data string) ([]Cookie, int, error) {
	data = strings.TrimSpace(strings.TrimPrefix(data, "\ufeff"))
	var arr []map[string]any
	if err := json.Unmarshal([]byte(data), &arr); err != nil {
		var wrap struct {
			Cookies []map[string]any `json:"cookies"`
		}
		if json.Unmarshal([]byte(data), &wrap) != nil || wrap.Cookies == nil {
			return nil, 0, fmt.Errorf("JSON 格式不对：需要 Cookie 数组，例如 [{\"name\":…,\"value\":…,\"domain\":…}]")
		}
		arr = wrap.Cookies
	}
	out, skipped := []Cookie{}, 0
	for _, m := range arr {
		c := Cookie{Name: str(m["name"]), Value: str(m["value"]), Domain: str(m["domain"]), Path: str(m["path"])}
		c.HTTPOnly, _ = m["httpOnly"].(bool)
		c.Secure, _ = m["secure"].(bool)
		c.SameSite, _ = m["sameSite"].(string)
		session, _ := m["session"].(bool)
		for _, k := range []string{"expires", "expirationDate", "expiry"} {
			if f, ok := num(m[k]); ok && f > 0 {
				c.Expires = f
				break
			}
		}
		if session {
			c.Expires = 0
		}
		if host, _ := m["hostOnly"].(bool); !host && c.Domain != "" && !strings.HasPrefix(c.Domain, ".") && m["hostOnly"] != nil {
			c.Domain = "." + c.Domain
		}
		if !c.normalize() {
			skipped++
			continue
		}
		out = append(out, c)
	}
	return out, skipped, nil
}

func str(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	}
	return fmt.Sprint(v)
}

func num(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case string:
		f, err := strconv.ParseFloat(t, 64)
		return f, err == nil
	}
	return 0, false
}

// parseNetscape reads cookies.txt: domain, includeSubdomains, path, secure,
// expiry, name, value (tab separated). "#HttpOnly_" prefixes mark httpOnly.
func parseNetscape(data string) ([]Cookie, int, error) {
	out, skipped := []Cookie{}, 0
	sc := bufio.NewScanner(strings.NewReader(strings.TrimPrefix(data, "\ufeff")))
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		httpOnly := false
		if strings.HasPrefix(line, "#HttpOnly_") {
			httpOnly, line = true, strings.TrimPrefix(line, "#HttpOnly_")
		} else if strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 7 {
			// Some tools write spaces instead of tabs.
			f = strings.Fields(line)
			if len(f) < 6 {
				skipped++
				continue
			}
			if len(f) == 6 {
				f = append(f, "")
			}
		}
		exp, _ := strconv.ParseFloat(strings.TrimSpace(f[4]), 64)
		c := Cookie{Domain: f[0], Path: f[2], Secure: strings.EqualFold(f[3], "TRUE"), Expires: exp, Name: f[5], Value: strings.Join(f[6:], "\t"), HTTPOnly: httpOnly}
		if strings.EqualFold(f[1], "TRUE") && !strings.HasPrefix(c.Domain, ".") {
			c.Domain = "." + c.Domain
		}
		if !c.normalize() {
			skipped++
			continue
		}
		out = append(out, c)
	}
	if err := sc.Err(); err != nil {
		return nil, 0, err
	}
	if len(out) == 0 && skipped > 0 {
		return nil, skipped, fmt.Errorf("没有识别到 Netscape 格式的 Cookie（每行 7 列，以 Tab 分隔）")
	}
	return out, skipped, nil
}

func parseCookies(format, data string) ([]Cookie, int, error) {
	if strings.TrimSpace(data) == "" {
		return nil, 0, fmt.Errorf("内容为空")
	}
	switch format {
	case "json":
		return parseCookiesJSON(data)
	case "netscape":
		return parseNetscape(data)
	case "", "auto":
		if t := strings.TrimSpace(data); strings.HasPrefix(t, "[") || strings.HasPrefix(t, "{") {
			return parseCookiesJSON(data)
		}
		return parseNetscape(data)
	}
	return nil, 0, fmt.Errorf("未知格式 %q", format)
}

func formatCookiesJSON(list []Cookie) string {
	if list == nil {
		list = []Cookie{}
	}
	b, _ := json.MarshalIndent(list, "", "  ")
	return string(b)
}

func formatNetscape(list []Cookie) string {
	var sb strings.Builder
	sb.WriteString("# Netscape HTTP Cookie File\n# Exported by GhostBrowser\n\n")
	for _, c := range list {
		dom := c.Domain
		if c.HTTPOnly {
			dom = "#HttpOnly_" + dom
		}
		sub := "FALSE"
		if strings.HasPrefix(c.Domain, ".") {
			sub = "TRUE"
		}
		sec := "FALSE"
		if c.Secure {
			sec = "TRUE"
		}
		exp := int64(0)
		if c.Expires > 0 {
			exp = int64(c.Expires)
		}
		fmt.Fprintf(&sb, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n", dom, sub, c.Path, sec, exp, c.Name, c.Value)
	}
	return sb.String()
}

func formatCookies(format string, list []Cookie) (string, error) {
	switch format {
	case "", "json":
		return formatCookiesJSON(list), nil
	case "netscape":
		return formatNetscape(list), nil
	}
	return "", fmt.Errorf("未知格式 %q", format)
}

func cookieKey(c Cookie) string { return c.Name + "\x00" + c.Domain + "\x00" + c.Path }

// mergeCookies: later lists win for the same name + domain + path.
func mergeCookies(lists ...[]Cookie) []Cookie {
	idx := map[string]int{}
	out := []Cookie{}
	for _, l := range lists {
		for _, c := range l {
			if i, ok := idx[cookieKey(c)]; ok {
				out[i] = c
				continue
			}
			idx[cookieKey(c)] = len(out)
			out = append(out, c)
		}
	}
	return out
}

// cdpParam is what Storage.setCookies accepts.
func (c Cookie) cdpParam() map[string]any {
	m := map[string]any{"name": c.Name, "value": c.Value, "domain": c.Domain, "path": c.Path, "secure": c.Secure, "httpOnly": c.HTTPOnly}
	if c.Expires > 0 {
		m["expires"] = c.Expires
	}
	if c.SameSite != "" {
		m["sameSite"] = c.SameSite
		if c.SameSite == "None" {
			m["secure"] = true // Chrome rejects SameSite=None without Secure
		}
	}
	return m
}

func liveSetCookies(ws string, list []Cookie) (int, error) {
	params := make([]map[string]any, len(list))
	for i, c := range list {
		params[i] = c.cdpParam()
	}
	ok := 0
	var lastErr error
	for i := 0; i < len(params); i += 200 {
		j := min(i+200, len(params))
		if _, err := cdpCall(ws, "Storage.setCookies", map[string]any{"cookies": params[i:j]}); err == nil {
			ok += j - i
			continue
		}
		// One bad cookie fails the whole batch: retry one by one.
		for _, one := range params[i:j] {
			if _, err := cdpCall(ws, "Storage.setCookies", map[string]any{"cookies": []map[string]any{one}}); err == nil {
				ok++
			} else {
				lastErr = err
			}
		}
	}
	if ok == 0 && lastErr != nil {
		return 0, lastErr
	}
	return ok, nil
}

func liveGetCookies(ws string) ([]Cookie, error) {
	raw, err := exportCookies(ws)
	if err != nil {
		return nil, err
	}
	return cdpCookies([]byte(raw))
}

func cdpCookies(raw []byte) ([]Cookie, error) {
	var in []struct {
		Name     string  `json:"name"`
		Value    string  `json:"value"`
		Domain   string  `json:"domain"`
		Path     string  `json:"path"`
		Expires  float64 `json:"expires"`
		HTTPOnly bool    `json:"httpOnly"`
		Secure   bool    `json:"secure"`
		Session  bool    `json:"session"`
		SameSite string  `json:"sameSite"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	out := make([]Cookie, 0, len(in))
	for _, x := range in {
		c := Cookie{Name: x.Name, Value: x.Value, Domain: x.Domain, Path: x.Path, Expires: x.Expires, HTTPOnly: x.HTTPOnly, Secure: x.Secure, SameSite: x.SameSite}
		if x.Session {
			c.Expires = 0
		}
		if c.normalize() {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Domain < out[j].Domain })
	return out, nil
}

func readSnapshot(dir string) ([]Cookie, int64) {
	b, err := os.ReadFile(filepath.Join(dir, sessionFile))
	if err != nil {
		return nil, 0
	}
	var snap struct {
		SavedAt int64           `json:"savedAt"`
		Cookies json.RawMessage `json:"cookies"`
	}
	if json.Unmarshal(b, &snap) != nil {
		return nil, 0
	}
	list, _ := cdpCookies(snap.Cookies)
	now := float64(time.Now().Unix())
	keep := list[:0]
	for _, c := range list {
		if c.Expires == 0 || c.Expires > now {
			keep = append(keep, c)
		}
	}
	return keep, snap.SavedAt
}

func stagedCookies(p *Profile) []Cookie {
	if strings.TrimSpace(p.Cookies) == "" {
		return nil
	}
	list, _, _ := parseCookiesJSON(p.Cookies)
	return list
}

func (s *Store) ImportCookies(id, format, data string) (CookieImportResult, error) {
	p, ok := s.GetProfile(id)
	if !ok {
		return CookieImportResult{}, fmt.Errorf("环境不存在")
	}
	list, skipped, err := parseCookies(format, data)
	if err != nil {
		return CookieImportResult{}, err
	}
	if len(list) == 0 {
		return CookieImportResult{}, fmt.Errorf("没有可导入的 Cookie（每条需要 name 和 domain）")
	}
	if n, live, err := firefoxSetLiveCookies(id, list); live { // Firefox kernel (WebDriver BiDi)
		if err != nil {
			return CookieImportResult{}, fmt.Errorf("写入浏览器失败：%w", err)
		}
		if r := getFirefoxRun(id); r != nil {
			_ = firefoxSnapshotSession(r.dir, r.device, r.bidi)
		}
		return CookieImportResult{Count: n, Skipped: skipped + len(list) - n, Mode: "live"}, nil
	}
	if rs, ok := runReg.get(id); ok && strings.HasPrefix(rs.WSURL, "ws") {
		n, err := liveSetCookies(rs.WSURL, list)
		if err != nil {
			return CookieImportResult{}, fmt.Errorf("写入浏览器失败：%w", err)
		}
		_ = snapshotSession(s.ProfileDir(id), s.GetSettings().DeviceID, rs.WSURL)
		return CookieImportResult{Count: n, Skipped: skipped + len(list) - n, Mode: "live"}, nil
	}
	merged := mergeCookies(stagedCookies(p), list)
	b, _ := json.Marshal(merged)
	p.Cookies = string(b)
	s.PutProfile(*p)
	return CookieImportResult{Count: len(list), Skipped: skipped, Mode: "staged", Staged: p.Cookies}, nil
}

func (s *Store) ExportCookies(id, format string) (string, error) {
	p, ok := s.GetProfile(id)
	if !ok {
		return "", fmt.Errorf("环境不存在")
	}
	var list []Cookie
	if l, live, err := firefoxLiveCookies(id); live { // Firefox kernel (WebDriver BiDi)
		if err != nil {
			return "", fmt.Errorf("读取浏览器 Cookie 失败：%w", err)
		}
		return formatCookies(format, l)
	}
	if rs, ok := runReg.get(id); ok && strings.HasPrefix(rs.WSURL, "ws") {
		l, err := liveGetCookies(rs.WSURL)
		if err != nil {
			return "", fmt.Errorf("读取浏览器 Cookie 失败：%w", err)
		}
		list = l
	} else {
		snap, _ := readSnapshot(s.ProfileDir(id))
		list = mergeCookies(snap, stagedCookies(p))
	}
	return formatCookies(format, list)
}

func (s *Store) CookieInfo(id string) CookieInfo {
	info := CookieInfo{}
	p, ok := s.GetProfile(id)
	if !ok {
		return info
	}
	_, info.Running = runReg.get(id)
	snap, at := readSnapshot(s.ProfileDir(id))
	info.Snapshot, info.SnapshotAt = len(snap), at
	info.Staged = len(stagedCookies(p))
	return info
}
