package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

// GhostBrowser's own device library. Every entry is a real, publicly documented
// device string (GPU renderer, screen, CPU/RAM class) that a browser of that OS
// actually reports, so each generated fingerprint stays internally consistent.
// This is first-party data, independent of any third-party kernel.

// ---- Windows GPUs (NVIDIA / AMD / Intel, desktop + laptop) ----
var winGPUsExt = []gpu{
	// NVIDIA GeForce RTX 40-series
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 4090 (0x00002684) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 4080 (0x00002704) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 4070 Ti (0x00002782) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 4070 (0x00002786) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 4060 (0x00002882) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 4060 Ti (0x00002803) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 4050 Laptop GPU (0x000028A1) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 4070 Laptop GPU (0x00002820) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	// NVIDIA GeForce RTX 30-series
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 3090 (0x00002204) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 3080 (0x00002206) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 3070 (0x00002484) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 3070 Laptop GPU (0x0000249D) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 3060 (0x00002504) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 3060 Ti (0x00002489) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 3060 Laptop GPU (0x00002520) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 3050 (0x00002507) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 3050 Laptop GPU (0x000025A2) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	// NVIDIA GeForce GTX
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce GTX 1660 SUPER (0x000021C4) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce GTX 1660 Ti (0x00002182) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce GTX 1650 (0x00001F82) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce GTX 1050 Ti (0x00001C82) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce GTX 1080 Ti (0x00001B06) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	// AMD Radeon
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon RX 7900 XTX (0x0000744C) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon RX 7800 XT (0x0000747E) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon RX 7600 (0x00007480) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon RX 6800 XT (0x000073BF) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon RX 6700 XT (0x000073DF) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon RX 6600 (0x000073FF) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon RX 6600 XT (0x000073EF) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon RX 580 (0x000067DF) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon(TM) Graphics (0x00001638) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon(TM) 780M Graphics (0x000015BF) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon(TM) 760M Graphics (0x000015BF) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	// Intel integrated
	{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) UHD Graphics 630 (0x00003E92) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) UHD Graphics 620 (0x00005917) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) UHD Graphics 770 (0x00004680) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) UHD Graphics 750 (0x00004C8A) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) Iris(R) Xe Graphics (0x00009A49) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) Iris(R) Xe Graphics (0x000046A6) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) Arc(TM) A770 Graphics (0x000056A0) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) HD Graphics 620 (0x00005916) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) HD Graphics 530 (0x00001912) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
}

// ---- macOS GPUs (Apple Silicon + Intel Macs) ----
var macGPUsExt = []gpu{
	{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M1, Unspecified Version)"},
	{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M1 Pro, Unspecified Version)"},
	{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M1 Max, Unspecified Version)"},
	{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M2, Unspecified Version)"},
	{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M2 Pro, Unspecified Version)"},
	{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M2 Max, Unspecified Version)"},
	{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M3, Unspecified Version)"},
	{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M3 Pro, Unspecified Version)"},
	{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M3 Max, Unspecified Version)"},
	{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M4, Unspecified Version)"},
	{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M4 Pro, Unspecified Version)"},
	{"Google Inc. (Intel)", "ANGLE (Intel Inc., Intel(R) Iris(TM) Plus Graphics 655, OpenGL 4.1)"},
	{"Google Inc. (Intel)", "ANGLE (Intel Inc., Intel(R) UHD Graphics 630, OpenGL 4.1)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon Pro 5500M OpenGL Engine, OpenGL 4.1)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon Pro 560X OpenGL Engine, OpenGL 4.1)"},
}

// ---- Linux GPUs (Mesa Intel / AMD, NVIDIA proprietary) ----
var linuxGPUsExt = []gpu{
	{"Google Inc. (Intel)", "ANGLE (Intel, Mesa Intel(R) UHD Graphics 620 (KBL GT2), OpenGL 4.6)"},
	{"Google Inc. (Intel)", "ANGLE (Intel, Mesa Intel(R) UHD Graphics 630 (CFL GT2), OpenGL 4.6)"},
	{"Google Inc. (Intel)", "ANGLE (Intel, Mesa Intel(R) Iris(R) Xe Graphics (TGL GT2), OpenGL 4.6)"},
	{"Google Inc. (Intel)", "ANGLE (Intel, Mesa Intel(R) HD Graphics 520 (SKL GT2), OpenGL 4.6)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon RX 6700 XT (navi22, LLVM 15.0.7, DRM 3.49), OpenGL 4.6)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon Graphics (renoir, LLVM 15.0.7, DRM 3.49), OpenGL 4.6)"},
	{"Google Inc. (NVIDIA Corporation)", "ANGLE (NVIDIA Corporation, NVIDIA GeForce RTX 3060/PCIe/SSE2, OpenGL 4.6.0)"},
	{"Google Inc. (NVIDIA Corporation)", "ANGLE (NVIDIA Corporation, NVIDIA GeForce GTX 1650/PCIe/SSE2, OpenGL 4.5.0)"},
	{"Google Inc. (NVIDIA Corporation)", "ANGLE (NVIDIA Corporation, NVIDIA GeForce RTX 4070/PCIe/SSE2, OpenGL 4.6.0)"},
}

// ---- Screen pools ----
var winScreensExt = []screen{
	{1920, 1080, 1}, {1920, 1080, 1.25}, {1366, 768, 1}, {1536, 864, 1.25}, {1600, 900, 1},
	{1440, 900, 1}, {1680, 1050, 1}, {1920, 1200, 1}, {2560, 1440, 1}, {2560, 1440, 1.25},
	{2560, 1600, 1.25}, {3440, 1440, 1}, {3840, 2160, 1.5}, {3840, 2160, 2}, {1280, 720, 1}, {1600, 1000, 1.25},
}
var macScreensExt = []screen{
	{1440, 900, 2}, {1512, 982, 2}, {1536, 960, 2}, {1680, 1050, 2}, {1728, 1117, 2},
	{1800, 1169, 2}, {1920, 1080, 2}, {2056, 1329, 2}, {2560, 1440, 2},
}
var linuxScreensExt = []screen{
	{1920, 1080, 1}, {1366, 768, 1}, {2560, 1440, 1}, {1600, 900, 1}, {3840, 2160, 2}, {1920, 1200, 1},
}

// pickGPU / pickScreen / pickFonts return the expanded pools for an OS.
func pickGPUs(osName string) []gpu {
	if mobileOS(osName) {
		seen, out := map[string]bool{}, []gpu{}
		for _, d := range mobileDevices {
			if d.os == osName && !seen[d.renderer] {
				seen[d.renderer] = true
				out = append(out, gpu{d.vendor, d.renderer})
			}
		}
		return out
	}
	switch osName {
	case "macos":
		return macGPUsExt
	case "linux":
		return linuxGPUsExt
	default:
		return winGPUsExt
	}
}

func mobileOS(o string) bool { return o == "android" || o == "chromeos" || o == "tizen" || o == "webos" }

func pickScreens(osName string) []screen {
	if mobileOS(osName) {
		seen, out := map[screen]bool{}, []screen{}
		for _, d := range mobileDevices {
			sc := screen{d.w, d.h, d.dpr}
			if d.os == osName && !seen[sc] {
				seen[sc] = true
				out = append(out, sc)
			}
		}
		return out
	}
	switch osName {
	case "macos":
		return macScreensExt
	case "linux":
		return linuxScreensExt
	default:
		return winScreensExt
	}
}

// Browser major versions GhostBrowser will mint UAs for (recent Chrome stable line).
var browserMajors = []string{"138", "139", "140", "141", "142"}

// CPU core / RAM classes that match consumer machines of each OS.
var winCores = []int{4, 6, 8, 8, 12, 16, 16, 24}
var winMem = []int{8, 8, 16, 16, 16, 32, 32, 64}
var macCores = []int{8, 8, 10, 10, 12, 14, 16}
var macMem = []int{8, 16, 16, 18, 24, 32}
var linuxCores = []int{4, 6, 8, 12, 16}
var linuxMem = []int{8, 16, 16, 32}

// DevicePreset is one complete, coherent machine profile.
type DevicePreset struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	OS        string  `json:"os"`
	OSVersion string  `json:"osVersion"`
	Vendor    string  `json:"webglVendor"`
	Renderer  string  `json:"webglRenderer"`
	Width     int     `json:"screenWidth"`
	Height    int     `json:"screenHeight"`
	DPR       float64 `json:"pixelRatio"`
	Cores     int     `json:"hardwareConcurrency"`
	Memory    int     `json:"deviceMemory"`
	Touch     int     `json:"maxTouchPoints"`
	Custom    bool    `json:"custom,omitempty"`
	// desktop | laptop | phone | tablet | foldable | chromebook | tv
	DeviceType string `json:"deviceType"`
	Model      string `json:"model,omitempty"`      // UA-CH model (phones / tablets)
	DeviceName string `json:"deviceName,omitempty"` // marketing name
	UA         string `json:"userAgent,omitempty"`  // fixed UA for devices with their own browser build (TVs); %s = major
	Platform   string `json:"platform,omitempty"`
}

var laptopScreens = []screen{{1920, 1080, 1.25}, {1536, 864, 1.25}, {1366, 768, 1}, {1920, 1200, 1.25}, {2560, 1600, 1.5}, {1600, 900, 1}, {1920, 1080, 1.5}}
var desktopScreens = []screen{{1920, 1080, 1}, {2560, 1440, 1}, {1920, 1200, 1}, {3440, 1440, 1}, {3840, 2160, 1.5}, {1680, 1050, 1}, {1600, 900, 1}, {2560, 1440, 1.25}}

func isLaptopGPU(r string) bool {
	for _, k := range []string{"Laptop", "Iris", "UHD Graphics 620", "HD Graphics 620", "780M", "760M", "Radeon(TM) Graphics"} {
		if strings.Contains(r, k) {
			return true
		}
	}
	return false
}

func gpuShort(r string) string {
	s := r
	if i := strings.Index(s, ", "); i >= 0 {
		s = s[i+2:]
	}
	for _, cut := range []string{" (0x", " OpenGL", ", Unspecified", "/PCIe", " (", ","} {
		if i := strings.Index(s, cut); i > 0 {
			s = s[:i]
		}
	}
	s = strings.NewReplacer("ANGLE Metal Renderer: ", "", "NVIDIA GeForce ", "", "AMD Radeon ", "Radeon ", "Intel(R) ", "", "(TM)", "", "(R)", "", "Mesa ", "").Replace(s)
	return strings.TrimSpace(s)
}

var appleCores = map[string][2]int{"M1": {8, 8}, "M1 Pro": {10, 16}, "M1 Max": {10, 32}, "M2": {8, 8}, "M2 Pro": {12, 16}, "M2 Max": {12, 32}, "M3": {8, 8}, "M3 Pro": {12, 18}, "M3 Max": {16, 36}, "M4": {10, 16}, "M4 Pro": {14, 24}}

func capMem(m int) int {
	if m > 8 { // navigator.deviceMemory is capped at 8 by Chrome
		return 8
	}
	return m
}

// devicePresets is built once, deterministically, so preset IDs stay stable across runs.
var devicePresets = func() []DevicePreset {
	r := rand.New(rand.NewSource(20261013))
	out := []DevicePreset{}
	add := func(osName, osVer string, g gpu, sc screen, cores, mem, touch int) {
		osLabel := map[string]string{"windows": "Win" + osVer, "macos": "macOS " + osVer, "linux": "Linux"}[osName]
		typ := "desktop"
		switch osName {
		case "windows", "linux":
			if isLaptopGPU(g.renderer) {
				typ = "laptop"
			}
		case "macos":
			if sc.w <= 1728 { // MacBook Air / Pro default scaled resolutions
				typ = "laptop"
			}
		}
		out = append(out, DevicePreset{
			ID:   fmt.Sprintf("%s-%03d", osName[:3], len(out)+1),
			Name: fmt.Sprintf("%s · %s · %dx%d", osLabel, gpuShort(g.renderer), sc.w, sc.h),
			OS:   osName, OSVersion: osVer, Vendor: g.vendor, Renderer: g.renderer,
			Width: sc.w, Height: sc.h, DPR: sc.dpr, Cores: cores, Memory: capMem(mem), Touch: touch, DeviceType: typ,
		})
	}
	// Windows: every GPU twice (Win10 + Win11) with a screen/CPU class that fits the machine type.
	for _, ver := range []string{"11", "10"} {
		for _, g := range winGPUsExt {
			laptop := isLaptopGPU(g.renderer)
			sc := pick(r, desktopScreens)
			touch := 0
			if laptop {
				sc = pick(r, laptopScreens)
				if r.Float64() < 0.2 {
					touch = 10
				}
			}
			cores := pick(r, gpuTierCores(g.renderer))
			if laptop && cores > 16 {
				cores = 16
			}
			add("windows", ver, g, sc, cores, pick(r, winMem), touch)
		}
	}
	// macOS: Apple Silicon with the chip's real core count, Intel/AMD Macs on Intel-class cores.
	for _, ver := range []string{"14.7", "15.3"} {
		for _, g := range macGPUsExt {
			cores, mem := pick(r, []int{4, 6, 8}), 8
			for chip, cm := range appleCores {
				if strings.HasSuffix(g.renderer, "Apple "+chip+", Unspecified Version)") {
					cores, mem = cm[0], cm[1]
				}
			}
			add("macos", ver, g, pick(r, macScreensExt), cores, mem, 0)
		}
	}
	for _, g := range linuxGPUsExt {
		add("linux", "x86_64", g, pick(r, linuxScreensExt), pick(r, linuxCores), pick(r, linuxMem), 0)
	}
	return out
}()

// DevicePresetsFor lists presets for one OS ("" = all).
func DevicePresetsFor(osName string) []DevicePreset {
	out := []DevicePreset{}
	all := append(userLib.presets(), devicePresets...)
	for _, p := range append(all, mobilePresets...) {
		if osName == "" || p.OS == osName {
			out = append(out, p)
		}
	}
	return out
}

// gpuTierCores keeps CPU class in line with the GPU: nobody pairs an RTX 4090 with a 4-thread CPU.
func gpuTierCores(r string) []int {
	for _, k := range []string{"4090", "4080", "3090", "3080", "7900", "7800", "6800", "4070 Ti", "1080 Ti"} {
		if strings.Contains(r, k) {
			return []int{16, 20, 24, 32}
		}
	}
	for _, k := range []string{"4070", "4060", "3070", "3060", "7600", "6700", "6600", "A770", "780M"} {
		if strings.Contains(r, k) {
			return []int{8, 12, 16, 16, 20}
		}
	}
	return []int{4, 4, 6, 8, 8, 12}
}

// FPCatalog is everything the editor's dropdowns need for one OS, so every
// choice the user makes stays inside what that OS really reports.
type FPCatalog struct {
	OSVersions    []string            `json:"osVersions"`
	BrowserMajors []string            `json:"browserMajors"`
	UserAgents    []string            `json:"userAgents"`
	Platforms     []string            `json:"platforms"`
	Vendors       []string            `json:"vendors"`
	GPUs          []map[string]string `json:"gpus"`
	Fonts         []string            `json:"fonts"`
	Screens       []string            `json:"screens"`
	Cores         []string            `json:"cores"`
	Memory        []string            `json:"memory"`
	Custom        map[string][]string `json:"custom"` // the user's own entries per field, deletable
}

var osVersionsByOS = map[string][]string{
	"windows": {"11", "10"},
	"macos":   {"15.5", "15.3", "14.7", "13.7"},
	"linux":    {"x86_64"},
	"android":  {"15", "14", "13", "12"},
	"chromeos": {"16093.68.0", "16063.45.0"},
	"tizen":    {"8.0", "7.0"},
	"webos":    {"24", "23"},
}

func FingerprintCatalog(osName, kernelMajor string) FPCatalog {
	return FingerprintCatalogFor(osName, "", kernelMajor)
}

// FingerprintCatalogFor is the catalog for one OS and device type (phone vs tablet UAs differ).
func FingerprintCatalogFor(osName, devType, kernelMajor string) FPCatalog {
	if _, ok := osVersionsByOS[osName]; !ok {
		osName = "windows"
	}
	majors := append([]string{}, browserMajors...)
	if kernelMajor != "" {
		found := false
		for _, m := range majors {
			found = found || m == kernelMajor
		}
		if !found {
			majors = append(majors, kernelMajor)
		}
	}
	sort.Slice(majors, func(i, j int) bool {
		return len(majors[i]) > len(majors[j]) || (len(majors[i]) == len(majors[j]) && majors[i] > majors[j])
	})
	c := FPCatalog{OSVersions: osVersionsByOS[osName], BrowserMajors: majors}
	for _, m := range majors {
		ua, pf := buildUAFor(osName, devType, "", m)
		c.UserAgents = append(c.UserAgents, ua)
		c.Platforms = []string{pf}
	}
	seen := map[string]bool{}
	for _, g := range pickGPUs(osName) {
		if !seen[g.vendor] {
			seen[g.vendor] = true
			c.Vendors = append(c.Vendors, g.vendor)
		}
		c.GPUs = append(c.GPUs, map[string]string{"vendor": g.vendor, "renderer": g.renderer})
	}
	c.Fonts = fontsFor(osName)
	if devType == "tv" && (osName == "tizen" || osName == "webos") {
		c.UserAgents = c.UserAgents[:1] // TV browsers have one fixed build
	}
	if osName == "android" || osName == "chromeos" || osName == "linux" {
		c.Platforms = append(c.Platforms, map[string][]string{"android": {"Linux aarch64", "Linux armv8l"}, "chromeos": {"Linux aarch64"}, "linux": {"Linux aarch64"}}[osName]...)
	}
	c.Screens = ScreenOptions(osName)
	c.Cores = []string{"2", "4", "6", "8", "10", "12", "14", "16", "20", "24", "32"}
	c.Memory = []string{"2", "4", "8"}
	c.Custom = map[string][]string{}
	for f := range libraryFields {
		c.Custom[f] = userLib.values(osName, f)
	}
	c.OSVersions = mergeFront(c.Custom["osVersions"], c.OSVersions)
	c.BrowserMajors = mergeFront(c.Custom["browserMajors"], c.BrowserMajors)
	c.UserAgents = mergeFront(c.Custom["userAgents"], c.UserAgents)
	c.Platforms = mergeFront(c.Custom["platforms"], c.Platforms)
	c.Vendors = mergeFront(c.Custom["vendors"], c.Vendors)
	c.Fonts = mergeFront(c.Custom["fonts"], c.Fonts)
	c.Screens = mergeFront(c.Custom["screens"], c.Screens)
	c.Cores = mergeFront(c.Custom["cores"], c.Cores)
	c.Memory = mergeFront(c.Custom["memory"], c.Memory)
	var gpus []map[string]string
	for _, g := range c.Custom["gpus"] {
		if v, r, ok := strings.Cut(g, "\x1f"); ok {
			gpus = append(gpus, map[string]string{"vendor": v, "renderer": r, "custom": "1"})
			c.Vendors = mergeFront(c.Vendors, []string{v})
		}
	}
	c.GPUs = append(gpus, c.GPUs...)
	return c
}
