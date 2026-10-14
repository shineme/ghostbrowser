package main

import (
	"bytes"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Operation audit log: an append-only record of what happened to which
// profile and who asked for it (the UI, the local API, an MCP client, auto
// sync). Newest first, capped at auditMax entries, kept in
// %APPDATA%/GhostBrowser/auditlog.json next to sync-log.json.

type AuditEntry struct {
	ID     string `json:"id"`
	Time   int64  `json:"time"`   // unix milliseconds
	Source string `json:"source"` // ui | api | mcp | sync
	Action string `json:"action"` // category.verb: profile.start, sync.push, api.call, mcp.call, group.create ...
	Target string `json:"target"` // profile ID when the action is about one profile
	Name   string `json:"name"`   // profile / group name at the time of the action
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// AuditFilter narrows AuditLog. Empty fields match everything.
type AuditFilter struct {
	Category string `json:"category"` // profile | sync | api | mcp | group | proxy
	Action   string `json:"action"`   // exact action, e.g. profile.start
	Source   string `json:"source"`
	Target   string `json:"target"` // profile ID
	Query    string `json:"query"`  // substring of name / detail / error / target
	OnlyFail bool   `json:"onlyFail"`
	Since    int64  `json:"since"` // unix ms
	Limit    int    `json:"limit"` // 0 = all
}

type AuditTargetCount struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type AuditResult struct {
	Items      []AuditEntry       `json:"items"`
	Total      int                `json:"total"`   // entries matching the filter (before Limit)
	All        int                `json:"all"`     // entries in the log
	Failed     int                `json:"failed"`  // failed entries in the whole log
	Categories map[string]int     `json:"categories"`
	Sources    map[string]int     `json:"sources"`
	Targets    []AuditTargetCount `json:"targets"` // profiles that appear in the log
}

const auditMax = 1000

var audit struct {
	mu   sync.Mutex
	dir  string // which store the cache belongs to
	list []AuditEntry
}

func (s *Store) auditFile() string { return filepath.Join(s.dir, "auditlog.json") }

// auditLoadLocked fills the cache for this store; caller holds audit.mu.
func (s *Store) auditLoadLocked() {
	if audit.dir == s.dir && audit.list != nil {
		return
	}
	audit.dir, audit.list = s.dir, []AuditEntry{}
	if b, e := os.ReadFile(s.auditFile()); e == nil {
		_ = json.Unmarshal(b, &audit.list)
	}
	if audit.list == nil {
		audit.list = []AuditEntry{}
	}
}

func auditID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// audit appends one entry. It never fails the operation it records.
func (s *Store) audit(e AuditEntry) {
	if s == nil || s.dir == "" {
		return
	}
	if e.Time == 0 {
		e.Time = time.Now().UnixMilli()
	}
	if e.ID == "" {
		e.ID = auditID()
	}
	if len(e.Detail) > 400 {
		e.Detail = e.Detail[:400] + "…"
	}
	if len(e.Error) > 400 {
		e.Error = e.Error[:400] + "…"
	}
	audit.mu.Lock()
	defer audit.mu.Unlock()
	s.auditLoadLocked()
	list := make([]AuditEntry, 0, len(audit.list)+1)
	list = append(list, e)
	list = append(list, audit.list...)
	if len(list) > auditMax {
		list = list[:auditMax]
	}
	audit.list = list
	b, _ := json.Marshal(list)
	tmp := s.auditFile() + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, s.auditFile())
	}
}

// auditProfile records an action on one profile; the name is looked up now.
func (s *Store) auditProfile(source, action, id string, err error, detail string) {
	name := ""
	if s != nil {
		if p, ok := s.GetProfile(id); ok {
			name = p.Name
		}
	}
	s.auditNamed(source, action, id, name, err, detail)
}

func (s *Store) auditNamed(source, action, id, name string, err error, detail string) {
	e := AuditEntry{Source: source, Action: action, Target: id, Name: name, OK: err == nil, Detail: detail}
	if err != nil {
		e.Error = err.Error()
	}
	s.audit(e)
}

func auditCategory(action string) string {
	c, _, _ := strings.Cut(action, ".")
	return c
}

func (f AuditFilter) match(e AuditEntry) bool {
	if f.Category != "" && auditCategory(e.Action) != f.Category {
		return false
	}
	if f.Action != "" && e.Action != f.Action {
		return false
	}
	if f.Source != "" && e.Source != f.Source {
		return false
	}
	if f.Target != "" && e.Target != f.Target {
		return false
	}
	if f.OnlyFail && e.OK {
		return false
	}
	if f.Since > 0 && e.Time < f.Since {
		return false
	}
	if q := strings.ToLower(strings.TrimSpace(f.Query)); q != "" {
		hay := strings.ToLower(e.Name + "\x00" + e.Detail + "\x00" + e.Error + "\x00" + e.Target + "\x00" + e.Action)
		if !strings.Contains(hay, q) {
			return false
		}
	}
	return true
}

// AuditQuery returns matching entries (newest first) plus counts over the whole log.
func (s *Store) AuditQuery(f AuditFilter) AuditResult {
	r := AuditResult{Items: []AuditEntry{}, Categories: map[string]int{}, Sources: map[string]int{}, Targets: []AuditTargetCount{}}
	if s == nil || s.dir == "" {
		return r
	}
	audit.mu.Lock()
	s.auditLoadLocked()
	all := append([]AuditEntry(nil), audit.list...)
	audit.mu.Unlock()

	tgt := map[string]*AuditTargetCount{}
	r.All = len(all)
	for _, e := range all {
		r.Categories[auditCategory(e.Action)]++
		r.Sources[e.Source]++
		if !e.OK {
			r.Failed++
		}
		if e.Target != "" {
			t := tgt[e.Target]
			if t == nil {
				t = &AuditTargetCount{ID: e.Target, Name: e.Name}
				tgt[e.Target] = t
			}
			if t.Name == "" {
				t.Name = e.Name
			}
			t.Count++
		}
		if f.match(e) {
			r.Total++
			if f.Limit <= 0 || len(r.Items) < f.Limit {
				r.Items = append(r.Items, e)
			}
		}
	}
	for _, t := range tgt {
		if p, ok := s.GetProfile(t.ID); ok { // current name wins
			t.Name = p.Name
		}
		r.Targets = append(r.Targets, *t)
	}
	sort.Slice(r.Targets, func(i, j int) bool {
		if r.Targets[i].Count != r.Targets[j].Count {
			return r.Targets[i].Count > r.Targets[j].Count
		}
		return r.Targets[i].Name < r.Targets[j].Name
	})
	return r
}

func (s *Store) ClearAudit() {
	if s == nil || s.dir == "" {
		return
	}
	audit.mu.Lock()
	defer audit.mu.Unlock()
	audit.dir, audit.list = s.dir, []AuditEntry{}
	_ = os.Remove(s.auditFile())
}

var auditSourceZH = map[string]string{"ui": "界面", "api": "本地 API", "mcp": "MCP", "sync": "同步"}

// auditExport renders entries as CSV (with a UTF-8 BOM so Excel opens Chinese
// correctly) or as indented JSON.
func auditExport(list []AuditEntry, format string) []byte {
	if format == "json" {
		b, _ := json.MarshalIndent(list, "", "  ")
		return b
	}
	var buf bytes.Buffer
	buf.WriteString("\ufeff")
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"时间", "来源", "操作", "环境ID", "名称", "结果", "错误", "详情"})
	for _, e := range list {
		res := "成功"
		if !e.OK {
			res = "失败"
		}
		src := auditSourceZH[e.Source]
		if src == "" {
			src = e.Source
		}
		_ = w.Write([]string{time.UnixMilli(e.Time).Format("2006-01-02 15:04:05"), src, e.Action, e.Target, e.Name, res, e.Error, e.Detail})
	}
	w.Flush()
	return buf.Bytes()
}
