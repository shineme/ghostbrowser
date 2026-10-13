package undetectable

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
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
	Architecture    string         `json:"architecture"`
	Bitness         string         `json:"bitness"`
	Brands          []BrandVersion `json:"brands"`
	FormFactors     []string       `json:"formFactors"`
	FullVersionList []BrandVersion `json:"fullVersionList"`
	Mobile          bool           `json:"mobile"`
	Model           string         `json:"model"`
	Platform        string         `json:"platform"`
	PlatformVersion string         `json:"platformVersion"`
	UAFullVersion   string         `json:"uaFullVersion"`
	Wow64           bool           `json:"wow64"`
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

// Config bundles everything the launcher needs for one profile.
type Config struct {
	Override *Expected
	Profile  *Profile
	Data     *Data
	Meta     map[string]json.RawMessage
	Info     *AdditionalInfo
	UAD      *UserAgentData
	GPU      *GPUAdapterInfo

	WebGLConfig map[string]json.RawMessage

	// FromDB is false when the directory exists but the database has no row
	// for it. That happens for profiles created by other tooling; startInfo is
	// still authoritative for the browser, so the launcher keeps working.
	FromDB bool

	// InfoPath is the profile's startInfo file.
	InfoPath string
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
	if c.Override != nil {
		return c.Override
	}
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
