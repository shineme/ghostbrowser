package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

// Profile mirrors one row of the `profiles` table in the Undetectable storage
// database. `data` and `configMeta` are JSON documents stored as TEXT.
type Profile struct {
	ID         string
	Name       string
	Data       string
	ConfigMeta string
	Modify     string
	IsDeleted  int
	IsTemp     int
	HasInfo    bool
	InfoSize   int64
}

type Data struct {
	Raw map[string]json.RawMessage
}

func (d *Data) Str(key string) string {
	v, ok := d.Raw[key]
	if !ok {
		return ""
	}
	var s string
	if json.Unmarshal(v, &s) == nil {
		return s
	}
	return strings.Trim(string(v), `"`)
}

func (d *Data) Bool(key string) bool {
	v, ok := d.Raw[key]
	if !ok {
		return false
	}
	var b bool
	if json.Unmarshal(v, &b) == nil {
		return b
	}
	return strings.Trim(string(v), `"`) == "true"
}

func (d *Data) Int(key string) int64 {
	v, ok := d.Raw[key]
	if !ok {
		return 0
	}
	var n json.Number
	if json.Unmarshal(v, &n) == nil {
		i, _ := n.Int64()
		return i
	}
	var f float64
	if json.Unmarshal(v, &f) == nil {
		return int64(f)
	}
	s := strings.Trim(string(v), `"`)
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

func (d *Data) Obj(key string) map[string]json.RawMessage {
	v, ok := d.Raw[key]
	if !ok {
		return nil
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(v, &m) != nil {
		return nil
	}
	return m
}

func parseData(s string) *Data {
	d := &Data{Raw: map[string]json.RawMessage{}}
	_ = json.Unmarshal([]byte(s), &d.Raw)
	return d
}

type AdditionalInfo struct {
	Raw map[string]json.RawMessage
}

func (a *AdditionalInfo) Obj(key string) map[string]json.RawMessage {
	v, ok := a.Raw[key]
	if !ok {
		return nil
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(v, &m) != nil {
		return nil
	}
	return m
}

type UserAgentData struct {
	Architecture    string        `json:"architecture"`
	Bitness         string        `json:"bitness"`
	Brands          []BrandVersion `json:"brands"`
	FormFactors     []string      `json:"formFactors"`
	FullVersionList []BrandVersion `json:"fullVersionList"`
	Mobile          bool          `json:"mobile"`
	Model           string        `json:"model"`
	Platform        string        `json:"platform"`
	PlatformVersion string        `json:"platformVersion"`
	UAFullVersion   string        `json:"uaFullVersion"`
	Wow64           bool          `json:"wow64"`
}

type BrandVersion struct {
	Brand   string `json:"brand"`
	Version string `json:"version"`
}

type GPUAdapterInfo struct {
	Architecture string `json:"architecture"`
	Description  string `json:"description"`
	Device       string `json:"device"`
	Driver       string `json:"driver"`
	Vendor       string `json:"vendor"`
}

type Store struct {
	DB       *sql.DB
	Profiles string // profiles root directory
}

func openStore() (*Store, error) {
	root := profilesRoot()
	dbPath := filepath.Join(root, "storage")
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("profile database not found at %s", dbPath)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath)+"?mode=ro&_pragma=query_only(1)")
	if err != nil {
		return nil, err
	}
	return &Store{DB: db, Profiles: root}, nil
}

// profilesRoot reads settings.json (profilesPath) the same way the browser's
// IniConnector does, falling back to the default location.
func profilesRoot() string {
	if p := os.Getenv("UNDETECTABLE_PROFILES"); p != "" {
		return p
	}
	cfg := filepath.Join(os.Getenv("LOCALAPPDATA"), "Undetectable", "settings.json")
	if b, err := os.ReadFile(cfg); err == nil {
		var s struct {
			ProfilesPath string `json:"profilesPath"`
		}
		if json.Unmarshal(b, &s) == nil && strings.TrimSpace(s.ProfilesPath) != "" {
			return filepath.Clean(filepath.FromSlash(s.ProfilesPath))
		}
	}
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "Undetectable", "Profiles")
}

func (s *Store) List() ([]Profile, error) {
	rows, err := s.DB.Query(
		`select id, coalesce(name,''), coalesce(data,''), coalesce(configMeta,''),
		        coalesce(modify,''), coalesce(isDeleted,0), coalesce(isTemp,0)
		 from profiles order by modify desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Profile
	for rows.Next() {
		var p Profile
		if err := rows.Scan(&p.ID, &p.Name, &p.Data, &p.ConfigMeta, &p.Modify,
			&p.IsDeleted, &p.IsTemp); err != nil {
			return nil, err
		}
		fi, err := os.Stat(filepath.Join(s.Profiles, p.ID, "startInfo"))
		if err == nil {
			p.HasInfo = true
			p.InfoSize = fi.Size()
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) Get(id string) (*Profile, error) {
	row := s.DB.QueryRow(
		`select id, coalesce(name,''), coalesce(data,''), coalesce(configMeta,''),
		        coalesce(modify,''), coalesce(isDeleted,0), coalesce(isTemp,0)
		 from profiles where id=?`, id)
	var p Profile
	if err := row.Scan(&p.ID, &p.Name, &p.Data, &p.ConfigMeta, &p.Modify,
		&p.IsDeleted, &p.IsTemp); err != nil {
		return nil, fmt.Errorf("profile %s not found: %w", id, err)
	}
	if fi, err := os.Stat(filepath.Join(s.Profiles, p.ID, "startInfo")); err == nil {
		p.HasInfo = true
		p.InfoSize = fi.Size()
	}
	return &p, nil
}

func (s *Store) Dir(id string) string { return filepath.Join(s.Profiles, id) }

func (s *Store) StartInfoPath(id string) string {
	return filepath.Join(s.Profiles, id, "startInfo")
}

// Config bundles everything the launcher needs for one profile.
type Config struct {
	Profile *Profile
	Data    *Data
	Meta    map[string]json.RawMessage
	Info    *AdditionalInfo
	UAD     *UserAgentData
	GPU     *GPUAdapterInfo

	WebGLConfig map[string]json.RawMessage

	// FromDB is false when the directory exists but the database has no row
	// for it. That happens for profiles created by other tooling; startInfo is
	// still authoritative for the browser, so the launcher keeps working.
	FromDB bool

	// InfoPath is the profile's startInfo file.
	InfoPath string
}

func (s *Store) Config(id string) (*Config, error) {
	p, err := s.Get(id)
	if err != nil {
		if fi, statErr := os.Stat(s.Dir(id)); statErr != nil || !fi.IsDir() {
			return nil, err
		}
		p = &Profile{ID: id, Name: id}
	}
	c := &Config{Profile: p, Data: parseData(p.Data), FromDB: p.Data != "",
		InfoPath: s.StartInfoPath(id)}
	if err := json.Unmarshal([]byte(p.ConfigMeta), &c.Meta); err != nil {
		if c.FromDB {
			return nil, fmt.Errorf("configMeta is not valid JSON: %w", err)
		}
		c.Meta = map[string]json.RawMessage{}
	}
	if c.Meta == nil {
		c.Meta = map[string]json.RawMessage{}
	}
	var ai string
	if v, ok := c.Meta["additional_info"]; ok {
		if json.Unmarshal(v, &ai) != nil {
			ai = strings.Trim(string(v), `"`)
		}
	}
	c.Info = &AdditionalInfo{Raw: map[string]json.RawMessage{}}
	if ai != "" {
		_ = json.Unmarshal([]byte(ai), &c.Info.Raw)
	}
	if v, ok := c.Info.Raw["userAgentDataObj"]; ok {
		var u UserAgentData
		if json.Unmarshal(v, &u) == nil {
			c.UAD = &u
		}
	}
	if v, ok := c.Info.Raw["gpuDataObj"]; ok {
		var g struct {
			AdapterInfo GPUAdapterInfo `json:"adapterInfo"`
		}
		if json.Unmarshal(v, &g) == nil {
			c.GPU = &g.AdapterInfo
		}
	}
	if v, ok := c.Meta["webglConfig"]; ok {
		c.WebGLConfig = map[string]json.RawMessage{}
		if json.Unmarshal(v, &c.WebGLConfig) != nil {
			var s2 string
			if json.Unmarshal(v, &s2) == nil {
				_ = json.Unmarshal([]byte(s2), &c.WebGLConfig)
			}
		}
	}
	return c, nil
}

// Expected holds the fingerprint values the database asks for.
type Expected struct {
	UA           string
	FullVersion  string
	OSVersion    string
	Platform     string // navigator.platform, e.g. Win32
	UADPlatform  string // userAgentData.platform, e.g. Windows
	Arch         string
	Bitness      string
	Cores        int64
	MemoryGB     int64
	Timezone     string
	Languages    []string
	LangHeader   string
	Width        int64
	Height       int64
	DPR          float64
	ColorDepth   int64
	PixelDepth   int64
	WebGLVendor  string
	WebGLRender  string
	PublicRTC    string
	StorageQuota uint64
	HeapLimit    uint64
}

var (
	reRes     = regexp.MustCompile(`^(\d+)x(\d+)$`)
	reFullVer = regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
)

func (c *Config) Expected() *Expected {
	e := &Expected{
		UA:          c.Data.Str("UA"),
		OSVersion:   c.Data.Str("osVers"),
		Platform:    "Win32",
		UADPlatform: "Windows",
		Cores:       c.Data.Int("CPUCores"),
		MemoryGB:    c.Data.Int("memory"),
		Timezone:    c.Data.Str("tzId"),
		LangHeader:  c.Data.Str("langHeader"),
		WebGLRender: c.Data.Str("webGL"),
		PublicRTC:   c.Data.Str("publicRTC"),
		ColorDepth:  24,
		PixelDepth:  24,
		DPR:         1,
	}
	// A profile directory without a database row still has a startInfo file,
	// which is what the browser actually reads. Fill the expectations from it
	// so verify compares against the values that are really in play.
	if !c.FromDB {
		c.expectedFromInfo(e)
		return e
	}
	if c.UAD != nil {
		e.Arch = c.UAD.Architecture
		e.Bitness = c.UAD.Bitness
		e.FullVersion = c.UAD.UAFullVersion
		if c.UAD.Platform != "" {
			e.UADPlatform = c.UAD.Platform
		}
	}
	if e.OSVersion == "10" {
		e.UADPlatform = "Windows"
	}
	// navigator.platform stays a legacy token regardless of the reported OS.
	if e.UADPlatform == "macOS" {
		e.Platform = "MacIntel"
	}
	if e.UADPlatform == "Linux" {
		e.Platform = "Linux x86_64"
	}
	if e.UADPlatform == "Android" {
		e.Platform = "Linux armv8l"
	}
	if e.UADPlatform == "Chrome OS" {
		e.Platform = "Linux x86_64"
	}
	if e.FullVersion == "" {
		if v := c.Data.Str("broVers"); v != "" {
			e.FullVersion = v
		}
	}
	if e.FullVersion == "" {
		e.FullVersion = reFullVer.FindString(c.Data.Str("UA"))
	}
	if e.FullVersion == "" {
		e.FullVersion = "153.0.8010.53"
	}
	// Rebuild the user agent with a concrete Chrome version when the stored UA
	// carries an empty version token.
	if strings.Contains(e.UA, "Chrome/ ") || strings.Contains(e.UA, "Chrome/ Safari") || e.UA == "" {
		e.UA = fmt.Sprintf(
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36",
			e.FullVersion)
	}
	if c.GPU != nil {
		e.WebGLVendor = gpuVendorString(c.GPU.Vendor)
	}
	if e.WebGLVendor == "" {
		e.WebGLVendor = "Google Inc. (Intel)"
	}
	if l := c.Data.Str("langs"); l != "" {
		for _, part := range strings.Split(l, "#") {
			if part = strings.TrimSpace(part); part != "" {
				e.Languages = append(e.Languages, part)
			}
		}
	}
	if len(e.Languages) == 0 {
		e.Languages = []string{"en-US", "en"}
	}
	if m := reRes.FindStringSubmatch(c.Data.Str("resConfig")); m != nil {
		e.Width, _ = strconv.ParseInt(m[1], 10, 64)
		e.Height, _ = strconv.ParseInt(m[2], 10, 64)
	}
	if e.Width == 0 {
		e.Width, e.Height = 1536, 864
	}
	if c.Data.Bool("emulateDPR") {
		e.DPR = 1.25
	}
	if v, ok := c.Info.Raw["storage"]; ok {
		var n uint64
		if json.Unmarshal(v, &n) == nil {
			e.StorageQuota = n
		}
	}
	if v, ok := c.Info.Raw["performance"]; ok {
		var perf struct {
			JSHeapSizeLimit uint64 `json:"jsHeapSizeLimit"`
		}
		if json.Unmarshal(v, &perf) == nil {
			e.HeapLimit = perf.JSHeapSizeLimit
		}
	}
	if e.StorageQuota == 0 {
		e.StorageQuota = 10737418240
	}
	if e.HeapLimit == 0 {
		e.HeapLimit = 4395630592
	}
	return e
}

func gpuVendorString(vendor string) string {
	switch strings.ToLower(vendor) {
	case "intel":
		return "Google Inc. (Intel)"
	case "nvidia":
		return "Google Inc. (NVIDIA)"
	case "amd", "ati":
		return "Google Inc. (AMD)"
	case "apple":
		return "Google Inc. (Apple)"
	default:
		if vendor == "" {
			return ""
		}
		return "Google Inc. (" + vendor + ")"
	}
}

// expectedFromInfo derives the expectations from the on-disk startInfo file for
// profiles that have no row in the database.
func (c *Config) expectedFromInfo(e *Expected) {
	e.FullVersion = "153.0.8010.53"
	if v, ok := c.Info.Raw["userAgentDataObj"]; ok {
		var u UserAgentData
		if json.Unmarshal(v, &u) == nil {
			if u.UAFullVersion != "" {
				e.FullVersion = u.UAFullVersion
			}
			e.UADPlatform = u.Platform
			e.Arch = u.Architecture
			e.Bitness = u.Bitness
		}
	}
	e.UA = fmt.Sprintf(
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36",
		e.FullVersion)
	e.Platform = "Win32"
	e.UADPlatform = "Windows"
	e.OSVersion = "11"
	e.Arch = "x86"
	e.Bitness = "64"
	e.Cores = 12
	e.MemoryGB = 16
	e.Width, e.Height = 1536, 864
	e.DPR = 1.25
	e.StorageQuota = 10737418240
	e.HeapLimit = 4395630592

	// Prefer the values that are actually stored in startInfo.
	if l, _, err := LoadStartInfo(c.InfoPath); err == nil {
		if l.UA1 >= 0 {
			if t := l.Slots[l.UA1].Text(); t != "" {
				e.UA = t
			}
		}
		// startInfo for the older layout stores an empty Chrome version token;
		// the launcher supplies the concrete version over DevTools.
		if strings.Contains(e.UA, "Chrome/ ") {
			e.UA = fmt.Sprintf(
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36",
				e.FullVersion)
		}
		if l.Platform >= 0 {
			e.Platform = l.Slots[l.Platform].Text()
		}
		if l.UAPlatform >= 0 {
			e.UADPlatform = l.Slots[l.UAPlatform].Text()
		}
		if l.OSVers >= 0 {
			e.OSVersion = l.Slots[l.OSVers].Text()
		}
		if l.Arch >= 0 {
			e.Arch = l.Slots[l.Arch].Text()
		}
		if l.Bitness >= 0 {
			e.Bitness = l.Slots[l.Bitness].Text()
		}
		if l.TZ >= 0 {
			e.Timezone = l.Slots[l.TZ].Text()
		}
		if l.Lang >= 0 {
			e.Languages = []string{l.Slots[l.Lang].Text()}
		}
		if l.WebGLRender >= 0 {
			e.WebGLRender = l.Slots[l.WebGLRender].Text()
		}
		if l.WebGLVendor >= 0 {
			e.WebGLVendor = l.Slots[l.WebGLVendor].Text()
		}
		if l.PublicRTC >= 0 {
			e.PublicRTC = l.Slots[l.PublicRTC].Text()
		}
		buf := EncodeSlots(l.Slots)
		if anchor, err := locateScalars(l.Slots, buf, l); err == nil {
			if sc, err := ReadScalars(buf, anchor); err == nil {
				e.Cores = int64(sc.Cores)
				e.MemoryGB = int64(sc.MemoryGB)
				e.Width = int64(sc.Width)
				e.Height = int64(sc.Height)
				e.DPR = sc.DPR
				e.ColorDepth = int64(sc.ColorDepth)
				e.PixelDepth = int64(sc.PixelDepth)
				e.StorageQuota = sc.Storage
				e.HeapLimit = sc.HeapLimit
			}
		}
	}
}

// Acc reads the canvas noise accumulator (configMeta.acc).
func (c *Config) Acc() float64 {
	if v, ok := c.Meta["acc"]; ok {
		var f float64
		if json.Unmarshal(v, &f) == nil {
			return f
		}
		var s string
		if json.Unmarshal(v, &s) == nil {
			f, _ = strconv.ParseFloat(s, 64)
			return f
		}
	}
	return 608
}

func (c *Config) CanvasParams() (p1, p2, r0, r1 uint32) {
	num := func(key string) uint32 {
		if v, ok := c.Meta[key]; ok {
			var f float64
			if json.Unmarshal(v, &f) == nil {
				return uint32(f)
			}
			var s string
			if json.Unmarshal(v, &s) == nil {
				n, _ := strconv.ParseUint(s, 10, 32)
				return uint32(n)
			}
		}
		return 0
	}
	return num("pixel1"), num("pixel2"), num("CanvasWebglRandomParameter"), num("CanvasWebglRandomParameter2")
}

// Proxy describes the profile proxy row.
type Proxy struct {
	Type   string `json:"type"`
	Host   string `json:"host"`
	Port   string `json:"port"`
	User   string `json:"user"`
	Pass   string `json:"pass"`
	IP     string `json:"IP"`
	Status string `json:"status"`
}

func (s *Store) Proxy(id string) (*Proxy, error) {
	p, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	d := parseData(p.Data)
	raw := d.Str("proxy")
	if raw == "" || raw == "null" {
		return nil, nil
	}
	var pr Proxy
	if err := json.Unmarshal([]byte(raw), &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// Server returns the Chrome-style proxy server flag value.
func (p *Proxy) Server() string {
	if p == nil || p.Host == "" {
		return ""
	}
	scheme := "http"
	switch strings.ToLower(p.Type) {
	case "socks5":
		scheme = "socks5"
	case "socks5u":
		scheme = "socks5u"
	case "https":
		scheme = "https"
	case "http":
		scheme = "http"
	}
	if p.Port == "" {
		return fmt.Sprintf("%s://%s", scheme, p.Host)
	}
	return fmt.Sprintf("%s://%s:%s", scheme, p.Host, p.Port)
}

func (c *Config) SortedMetaKeys() []string {
	keys := make([]string, 0, len(c.Meta))
	for k := range c.Meta {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
