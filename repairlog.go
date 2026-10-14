package main

// One-click repair for the self-check, with a per-profile repair log stored in
// %APPDATA%/GhostBrowser/repairlog/<profileId>.json. Every repair records the
// prior value of each changed fingerprint field (the backup), so an entry can
// be rolled back field by field without clobbering later unrelated edits.

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RepairChange is one field changed by a repair.
type RepairChange struct {
	Item  string          `json:"item"`  // check key (language, timezone, ...)
	Label string          `json:"label"` // 中文
	Field string          `json:"field"` // Fingerprint json field
	Old   string          `json:"old"`   // display text
	New   string          `json:"new"`
	Prior json.RawMessage `json:"prior"` // exact old value (backup)
	After json.RawMessage `json:"after"` // exact new value
}

// RepairEntry is one row of the repair history.
type RepairEntry struct {
	ID         string         `json:"id"`
	ProfileID  string         `json:"profileId"`
	At         int64          `json:"at"`
	Kind       string         `json:"kind"` // repair | restore
	Items      []string       `json:"items"`
	Changes    []RepairChange `json:"changes"`
	Skipped    []string       `json:"skipped"` // items asked for but not changed, with reason
	Restored   bool           `json:"restored"`
	RestoredAt int64          `json:"restoredAt,omitempty"`
	RestoreOf  string         `json:"restoreOf,omitempty"`
}

// RepairResult is what RepairEnvironment returns.
type RepairResult struct {
	Entry   *RepairEntry `json:"entry"` // nil when nothing changed
	Changed int          `json:"changed"`
	Skipped []string     `json:"skipped"`
	Profile Profile      `json:"profile"`
}

var repairMu sync.Mutex

func (s *Store) repairFile(id string) string {
	return filepath.Join(s.dir, "repairlog", id+".json")
}

func (s *Store) RepairHistory(id string) []RepairEntry {
	repairMu.Lock()
	defer repairMu.Unlock()
	return s.loadRepairs(id)
}

func (s *Store) loadRepairs(id string) []RepairEntry {
	out := []RepairEntry{}
	b, err := os.ReadFile(s.repairFile(id))
	if err == nil {
		_ = json.Unmarshal(b, &out)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At > out[j].At })
	return out
}

func (s *Store) saveRepairs(id string, list []RepairEntry) error {
	if len(list) > 200 {
		list = list[:200]
	}
	f := s.repairFile(id)
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(list, "", " ")
	tmp := f + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, f)
}

// fpFields / setFPFields read and write Fingerprint fields by json name.
func fpFields(fp Fingerprint) map[string]json.RawMessage {
	b, _ := json.Marshal(fp)
	m := map[string]json.RawMessage{}
	_ = json.Unmarshal(b, &m)
	return m
}

func setFPFields(fp Fingerprint, vals map[string]json.RawMessage) (Fingerprint, error) {
	m := fpFields(fp)
	for k, v := range vals {
		if len(v) == 0 || string(v) == "null" {
			delete(m, k)
		} else {
			m[k] = v
		}
	}
	b, _ := json.Marshal(m)
	var out Fingerprint
	err := json.Unmarshal(b, &out)
	return out, err
}

func rawOr(m map[string]json.RawMessage, k string) json.RawMessage {
	if v, ok := m[k]; ok {
		return v
	}
	return json.RawMessage("null")
}

func displayRaw(field string, v json.RawMessage) string {
	if len(v) == 0 || string(v) == "null" {
		return "（空）"
	}
	switch field {
	case "fonts":
		var xs []string
		_ = json.Unmarshal(v, &xs)
		return fmt.Sprintf("%d 个字体", len(xs))
	case "languages":
		var xs []string
		_ = json.Unmarshal(v, &xs)
		if len(xs) == 0 {
			return "（空）"
		}
		return strings.Join(xs, ", ")
	case "webrtc":
		var s string
		_ = json.Unmarshal(v, &s)
		return webrtcCN(s)
	case "timezoneFromIP", "languageFromIP", "geoFromIP":
		if string(v) == "true" {
			return "跟随 IP"
		}
		return "固定值"
	}
	var s string
	if json.Unmarshal(v, &s) == nil {
		if s == "" {
			return "（空）"
		}
		return s
	}
	return string(v)
}

var fieldLabels = map[string]string{
	"languages": "语言", "languageFromIP": "语言跟随 IP", "timezone": "时区", "timezoneFromIP": "时区跟随 IP",
	"latitude": "纬度", "longitude": "经度", "geoFromIP": "定位跟随 IP", "webrtc": "WebRTC",
	"noiseSeed": "噪声种子", "fonts": "字体列表", "startupParameters": "启动参数",
}

// planRepair computes new field values for the requested items. exit may be
// a failed GeoMatch; IP-dependent items are then skipped.
func planRepair(p Profile, items []string, exit GeoMatch, hasProxy bool, rnd *rand.Rand) (map[string]any, map[string]string, []string) {
	fp := p.Fingerprint
	set := map[string]any{}
	itemOf := map[string]string{}
	var skipped []string
	put := func(item, field string, v any) { set[field] = v; itemOf[field] = item }
	for _, it := range items {
		if p.useFirefox() { // Chromium-only fixes never touch a Firefox profile
			if it == "seed" || it == "fonts" || it == "webdriver" {
				skipped = append(skipped, it+"：Firefox 内核不适用")
				continue
			}
			if fp.FFStrict && (it == "language" || it == "timezone") {
				skipped = append(skipped, it+"：Firefox 严格模式固定为 en-US / UTC")
				continue
			}
		}
		switch it {
		case "language":
			if !exit.OK || exit.Country == "" {
				skipped = append(skipped, "语言：出口未知，无法跟随 IP")
				continue
			}
			put(it, "languageFromIP", true)
			put(it, "languages", exit.Languages)
		case "timezone":
			if !exit.OK || exit.Timezone == "" {
				skipped = append(skipped, "时区：出口未知，无法跟随 IP")
				continue
			}
			put(it, "timezoneFromIP", true)
			put(it, "timezone", exit.Timezone)
		case "geo":
			if !exit.OK || (exit.Lat == 0 && exit.Lon == 0) {
				skipped = append(skipped, "地理位置：出口未知，无法跟随 IP")
				continue
			}
			put(it, "geoFromIP", true)
			put(it, "latitude", exit.Lat)
			put(it, "longitude", exit.Lon)
		case "webrtc":
			put(it, "webrtc", "proxy")
		case "seed":
			put(it, "noiseSeed", randomSeed())
		case "fonts":
			_, _, pool, ok := fontRange(fp.OS)
			if !ok {
				skipped = append(skipped, "字体：该设备类型不需要字体列表")
				continue
			}
			put(it, "fonts", subsetFonts(rnd, pool))
		case "webdriver":
			cleaned := strings.Join(strings.Fields(automationFlag.ReplaceAllString(fp.StartupParameters, "")), " ")
			put(it, "startupParameters", cleaned)
		case "clock":
			skipped = append(skipped, "系统时钟：系统级设置，无法自动修复，请开启系统自动对时")
		case "exit", "ua":
			skipped = append(skipped, it+"：无法自动修复")
		default:
			skipped = append(skipped, it+"：未知项目")
		}
	}
	_ = hasProxy
	return set, itemOf, skipped
}

// RepairEnvironment applies fixes for items (empty = every fixable item that
// is not passing) and logs the change. Clock / exit / UA are report-only.
func (s *Store) RepairEnvironment(id string, items []string) (RepairResult, error) {
	p, ok := s.GetProfile(id)
	if !ok {
		return RepairResult{}, fmt.Errorf("环境不存在")
	}
	px := profileProxy(s, p)
	var exit GeoMatch
	needExit := len(items) == 0
	for _, it := range items {
		if it == "language" || it == "timezone" || it == "geo" {
			needExit = true
		}
	}
	if needExit {
		exit = s.liveExit(px, 60*time.Second)
	}
	if len(items) == 0 {
		rep := buildEnvReport(envInput{Profile: *p, Proxy: px, Exit: exit, Chromium: p.useChromium(), Firefox: p.useFirefox(), Now: time.Now(), Clock: ClockReport{Status: "unknown"}})
		items = rep.Fixable
	}
	return s.applyRepair(p, items, exit, px != nil)
}

func (s *Store) applyRepair(p *Profile, items []string, exit GeoMatch, hasProxy bool) (RepairResult, error) {
	repairMu.Lock()
	defer repairMu.Unlock()
	set, itemOf, skipped := planRepair(*p, items, exit, hasProxy, rand.New(rand.NewSource(time.Now().UnixNano())))
	before := fpFields(p.Fingerprint)
	vals := map[string]json.RawMessage{}
	for k, v := range set {
		b, _ := json.Marshal(v)
		vals[k] = b
	}
	nfp, err := setFPFields(p.Fingerprint, vals)
	if err != nil {
		return RepairResult{}, err
	}
	after := fpFields(nfp)
	var changes []RepairChange
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		o, n := rawOr(before, k), rawOr(after, k)
		if string(o) == string(n) {
			continue
		}
		changes = append(changes, RepairChange{Item: itemOf[k], Label: fieldLabels[k], Field: k, Old: displayRaw(k, o), New: displayRaw(k, n), Prior: o, After: n})
	}
	res := RepairResult{Skipped: skipped, Profile: *p}
	if res.Skipped == nil {
		res.Skipped = []string{}
	}
	if len(changes) == 0 {
		return res, nil
	}
	p.Fingerprint = nfp
	res.Profile = s.PutProfile(*p)
	e := RepairEntry{ID: newID(), ProfileID: p.ID, At: time.Now().Unix(), Kind: "repair", Items: uniqItems(changes), Changes: changes, Skipped: res.Skipped}
	list := append([]RepairEntry{e}, s.loadRepairs(p.ID)...)
	if err := s.saveRepairs(p.ID, list); err != nil {
		return res, fmt.Errorf("已修复，但写入修复记录失败：%v", err)
	}
	res.Entry, res.Changed = &e, len(changes)
	return res, nil
}

func uniqItems(cs []RepairChange) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, c := range cs {
		if !seen[c.Item] {
			seen[c.Item] = true
			out = append(out, c.Item)
		}
	}
	return out
}

// RestoreRepair rolls back one repair entry: every field it changed goes back
// to its backed-up value. The rollback itself is logged as a "restore" entry.
func (s *Store) RestoreRepair(id, entryID string) (RepairResult, error) {
	repairMu.Lock()
	defer repairMu.Unlock()
	p, ok := s.GetProfile(id)
	if !ok {
		return RepairResult{}, fmt.Errorf("环境不存在")
	}
	list := s.loadRepairs(id)
	idx := -1
	for i := range list {
		if list[i].ID == entryID {
			idx = i
		}
	}
	if idx < 0 {
		return RepairResult{}, fmt.Errorf("修复记录不存在")
	}
	e := list[idx]
	if e.Kind != "repair" {
		return RepairResult{}, fmt.Errorf("恢复记录不能再次恢复")
	}
	if e.Restored {
		return RepairResult{}, fmt.Errorf("该记录已恢复过")
	}
	cur := fpFields(p.Fingerprint)
	vals := map[string]json.RawMessage{}
	var back []RepairChange
	for _, c := range e.Changes {
		vals[c.Field] = c.Prior
		now := rawOr(cur, c.Field)
		back = append(back, RepairChange{Item: c.Item, Label: c.Label, Field: c.Field, Old: displayRaw(c.Field, now), New: c.Old, Prior: now, After: c.Prior})
	}
	nfp, err := setFPFields(p.Fingerprint, vals)
	if err != nil {
		return RepairResult{}, err
	}
	p.Fingerprint = nfp
	out := s.PutProfile(*p)
	now := time.Now().Unix()
	list[idx].Restored, list[idx].RestoredAt = true, now
	r := RepairEntry{ID: newID(), ProfileID: id, At: now, Kind: "restore", Items: e.Items, Changes: back, Skipped: []string{}, RestoreOf: e.ID}
	list = append([]RepairEntry{r}, list...)
	if err := s.saveRepairs(id, list); err != nil {
		return RepairResult{Profile: out}, fmt.Errorf("已恢复，但写入修复记录失败：%v", err)
	}
	return RepairResult{Entry: &r, Changed: len(back), Skipped: []string{}, Profile: out}, nil
}
