package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
)

// Groups are the Profile.Group labels. A group can also exist with no
// profiles in it yet (created from the API / MCP before anything is moved
// in); those names are kept in groups.json.

type GroupInfo struct {
	Name    string `json:"name"`
	Count   int    `json:"count"`   // profiles in the group
	Running int    `json:"running"` // of which running
	Saved   bool   `json:"saved"`   // created explicitly (kept even when empty)
}

type TagInfo struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type KernelInfo struct {
	ID      string `json:"id"`   // custom | chromium (the value of Profile.Kernel is "" | "chromium")
	Name    string `json:"name"` // shown to people
	Path    string `json:"path"`
	Found   bool   `json:"found"`
	Version string `json:"version"` // Chrome major, "" when unknown
	Auto    bool   `json:"auto"`    // path was auto-detected (no explicit setting)
	Note    string `json:"note"`
}

var groupsMu sync.Mutex

func (s *Store) groupsFile() string { return filepath.Join(s.dir, "groups.json") }

func (s *Store) savedGroups() []string {
	var out []string
	if s.dir == "" {
		return out
	}
	if b, e := os.ReadFile(s.groupsFile()); e == nil {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

func (s *Store) ListGroups() []GroupInfo {
	groupsMu.Lock()
	saved := s.savedGroups()
	groupsMu.Unlock()
	m := map[string]*GroupInfo{}
	for _, g := range saved {
		m[g] = &GroupInfo{Name: g, Saved: true}
	}
	for _, p := range s.ListProfiles() {
		g := strings.TrimSpace(p.Group)
		if g == "" {
			continue
		}
		gi := m[g]
		if gi == nil {
			gi = &GroupInfo{Name: g}
			m[g] = gi
		}
		gi.Count++
		if _, ok := runReg.get(p.ID); ok {
			gi.Running++
		}
	}
	out := make([]GroupInfo, 0, len(m))
	for _, g := range m {
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func validGroupName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("分组名不能为空")
	}
	if utf8.RuneCountInString(name) > 64 {
		return "", fmt.Errorf("分组名最多 64 个字符")
	}
	if strings.ContainsAny(name, "\x00\r\n\t") {
		return "", fmt.Errorf("分组名不能包含控制字符")
	}
	return name, nil
}

// CreateGroup registers an (empty) group. Creating a name that already exists
// is an error so callers notice typos.
func (s *Store) CreateGroup(name string) (GroupInfo, error) {
	name, err := validGroupName(name)
	if err != nil {
		return GroupInfo{}, err
	}
	for _, g := range s.ListGroups() {
		if strings.EqualFold(g.Name, name) {
			return g, fmt.Errorf("分组「%s」已存在", g.Name)
		}
	}
	groupsMu.Lock()
	defer groupsMu.Unlock()
	list := append(s.savedGroups(), name)
	b, _ := json.Marshal(list)
	tmp := s.groupsFile() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return GroupInfo{}, err
	}
	if err := os.Rename(tmp, s.groupsFile()); err != nil {
		return GroupInfo{}, err
	}
	return GroupInfo{Name: name, Saved: true}, nil
}

func (s *Store) ListTags() []TagInfo {
	m := map[string]int{}
	for _, p := range s.ListProfiles() {
		seen := map[string]bool{}
		for _, t := range p.Tags {
			t = strings.TrimSpace(t)
			if t == "" || seen[t] {
				continue
			}
			seen[t] = true
			m[t]++
		}
	}
	out := make([]TagInfo, 0, len(m))
	for t, n := range m {
		out = append(out, TagInfo{t, n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// ListKernels reports both browser kernels: the custom (Undetectable) one for
// Windows desktop profiles and the open-source Chromium for everything else.
func (s *Store) ListKernels() []KernelInfo {
	st := s.GetSettings()
	custom := KernelInfo{ID: "custom", Name: "定制 Chromium（原生指纹）", Path: st.KernelPath, Note: "Windows 桌面 / 笔记本环境"}
	if custom.Path == "" {
		custom.Path, custom.Auto = defaultKernelPath(st), true
	}
	custom.Found = custom.Path != "" && fileExists(custom.Path)
	if custom.Found {
		custom.Version = chromeMajorFrom(filepath.Dir(custom.Path))
	}
	open := KernelInfo{ID: "chromium", Name: "开源 Chromium（CDP 注入）", Note: "手机、平板、Mac、Linux 等环境"}
	open.Auto = strings.TrimSpace(st.ChromiumPath) == ""
	open.Path = chromiumKernel(st)
	open.Found = open.Path != ""
	if open.Found {
		open.Version = chromeMajorFrom(filepath.Dir(open.Path))
	}
	return []KernelInfo{custom, open}
}
