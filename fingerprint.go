package main

import (
	"fmt"
	"math/rand"
	"time"
)

// Internally consistent building blocks: every value picked for an OS comes
// from a pool that real devices of that OS actually report.

type gpu struct{ vendor, renderer string }

var winGPUs = []gpu{
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 3060 (0x00002504) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 4060 (0x00002882) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce GTX 1660 SUPER (0x000021C4) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 3070 Laptop GPU (0x0000249D) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) UHD Graphics 630 (0x00003E92) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) Iris(R) Xe Graphics (0x00009A49) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon RX 6600 (0x000073FF) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon(TM) Graphics (0x00001638) Direct3D11 vs_5_0 ps_5_0, D3D11)"},
}

var macGPUs = []gpu{
	{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M1, Unspecified Version)"},
	{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M2, Unspecified Version)"},
	{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M3 Pro, Unspecified Version)"},
	{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M1 Pro, Unspecified Version)"},
}

var linuxGPUs = []gpu{
	{"Google Inc. (Intel)", "ANGLE (Intel, Mesa Intel(R) UHD Graphics 620 (KBL GT2), OpenGL 4.6)"},
	{"Google Inc. (NVIDIA Corporation)", "ANGLE (NVIDIA Corporation, NVIDIA GeForce GTX 1650/PCIe/SSE2, OpenGL 4.5.0)"},
}

type screen struct {
	w, h int
	dpr  float64
}

var winScreens = []screen{{1920, 1080, 1}, {1366, 768, 1}, {1536, 864, 1.25}, {2560, 1440, 1}, {1600, 900, 1}, {1440, 900, 1}, {1920, 1200, 1}}
var macScreens = []screen{{1440, 900, 2}, {1512, 982, 2}, {1728, 1117, 2}, {1680, 1050, 2}, {1920, 1080, 2}}
var linuxScreens = []screen{{1920, 1080, 1}, {1366, 768, 1}, {2560, 1440, 1}}

var winFonts = []string{"Arial", "Arial Black", "Bahnschrift", "Calibri", "Cambria", "Cambria Math", "Candara", "Comic Sans MS", "Consolas", "Constantia", "Corbel", "Courier New", "Ebrima", "Franklin Gothic Medium", "Gabriola", "Gadugi", "Georgia", "Impact", "Ink Free", "Javanese Text", "Leelawadee UI", "Lucida Console", "Lucida Sans Unicode", "Malgun Gothic", "Microsoft Himalaya", "Microsoft JhengHei", "Microsoft New Tai Lue", "Microsoft PhagsPa", "Microsoft Sans Serif", "Microsoft Tai Le", "Microsoft YaHei", "Microsoft Yi Baiti", "MingLiU-ExtB", "Mongolian Baiti", "MS Gothic", "MV Boli", "Myanmar Text", "Nirmala UI", "Palatino Linotype", "Segoe MDL2 Assets", "Segoe Print", "Segoe Script", "Segoe UI", "Segoe UI Emoji", "Segoe UI Historic", "Segoe UI Symbol", "SimSun", "Sitka Small", "Sylfaen", "Symbol", "Tahoma", "Times New Roman", "Trebuchet MS", "Verdana", "Webdings", "Wingdings", "Yu Gothic"}
var macFonts = []string{"American Typewriter", "Andale Mono", "Arial", "Arial Black", "Arial Hebrew", "Arial Rounded MT Bold", "Arial Unicode MS", "Avenir", "Avenir Next", "Baskerville", "Big Caslon", "Bradley Hand", "Brush Script MT", "Chalkboard", "Chalkduster", "Charter", "Cochin", "Comic Sans MS", "Copperplate", "Courier", "Courier New", "Didot", "Futura", "Geneva", "Georgia", "Gill Sans", "Helvetica", "Helvetica Neue", "Herculanum", "Hoefler Text", "Impact", "Lucida Grande", "Luminari", "Marker Felt", "Menlo", "Monaco", "Noteworthy", "Optima", "Palatino", "Papyrus", "PingFang SC", "Rockwell", "SF Pro", "Skia", "Snell Roundhand", "Tahoma", "Times", "Times New Roman", "Trattatello", "Trebuchet MS", "Verdana", "Zapfino"}
var linuxFonts = []string{"DejaVu Sans", "DejaVu Sans Mono", "DejaVu Serif", "Liberation Mono", "Liberation Sans", "Liberation Serif", "Noto Color Emoji", "Noto Sans", "Noto Sans CJK SC", "Noto Serif", "Ubuntu", "Ubuntu Mono"}

func pick[T any](r *rand.Rand, xs []T) T { return xs[r.Intn(len(xs))] }

// subsetFonts keeps the core fonts and randomly drops a few optional ones so
// that every profile's font list differs a little, like real machines do.
func subsetFonts(r *rand.Rand, all []string) []string {
	out := make([]string, 0, len(all))
	for i, f := range all {
		if i < 12 || r.Float64() > 0.18 {
			out = append(out, f)
		}
	}
	return out
}

func buildUA(osName, osVer, major string) (ua, platform string) {
	switch osName {
	case "macos":
		return fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 Safari/537.36", major), "MacIntel"
	case "linux":
		return fmt.Sprintf("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 Safari/537.36", major), "Linux x86_64"
	default:
		// Chrome freezes the Windows token to "Windows NT 10.0" for 10 and 11.
		return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 Safari/537.36", major), "Win32"
	}
}

// GenerateFingerprint creates a new, internally consistent fingerprint.
func GenerateFingerprint(osName, browserMajor string) Fingerprint {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if osName == "" {
		osName = "windows"
	}
	if browserMajor == "" {
		browserMajor = "141"
	}
	var g gpu
	var sc screen
	var fonts []string
	var osVer string
	cores := pick(r, []int{4, 8, 8, 12, 16})
	mem := pick(r, []int{4, 8, 8, 8})
	switch osName {
	case "macos":
		g, sc, fonts = pick(r, macGPUs), pick(r, macScreens), subsetFonts(r, macFonts)
		osVer = pick(r, []string{"14.6", "15.1", "15.5"})
		cores = pick(r, []int{8, 10, 12})
		mem = 8
	case "linux":
		g, sc, fonts = pick(r, linuxGPUs), pick(r, linuxScreens), subsetFonts(r, linuxFonts)
		osVer = "x86_64"
	default:
		g, sc, fonts = pick(r, winGPUs), pick(r, winScreens), subsetFonts(r, winFonts)
		osVer = pick(r, []string{"10", "11", "11"})
	}
	ua, platform := buildUA(osName, osVer, browserMajor)
	return Fingerprint{
		OS: osName, OSVersion: osVer, BrowserVersion: browserMajor,
		UserAgent: ua, Platform: platform,
		ScreenWidth: sc.w, ScreenHeight: sc.h, PixelRatio: sc.dpr, ColorDepth: 24,
		HardwareConc: cores, DeviceMemory: mem, MaxTouchPoints: 0,
		WebGLVendor: g.vendor, WebGLRenderer: g.renderer,
		Languages: []string{"en-US", "en"}, LanguageFromIP: true,
		Timezone: "America/New_York", TimezoneFromIP: true, GeoFromIP: true,
		CanvasNoise: true, AudioNoise: true, WebGLNoise: true, ClientRectNoise: true,
		NoiseSeed: r.Int63n(1 << 40),
		WebRTC:    "proxy",
		Fonts:     fonts,
	}
}

// GPU / screen pools exposed to the UI's dropdowns.
func GPUOptions(osName string) []map[string]string {
	src := winGPUs
	switch osName {
	case "macos":
		src = macGPUs
	case "linux":
		src = linuxGPUs
	}
	out := []map[string]string{}
	for _, g := range src {
		out = append(out, map[string]string{"vendor": g.vendor, "renderer": g.renderer})
	}
	return out
}

func ScreenOptions(osName string) []string {
	src := winScreens
	switch osName {
	case "macos":
		src = macScreens
	case "linux":
		src = linuxScreens
	}
	out := []string{}
	for _, s := range src {
		out = append(out, fmt.Sprintf("%dx%d@%g", s.w, s.h, s.dpr))
	}
	return out
}

// languageForCountry maps a proxy exit country to Accept-Language.
func languageForCountry(cc string) []string {
	m := map[string][]string{
		"US": {"en-US", "en"}, "GB": {"en-GB", "en"}, "CA": {"en-CA", "en", "fr-CA"}, "AU": {"en-AU", "en"},
		"DE": {"de-DE", "de", "en-US", "en"}, "FR": {"fr-FR", "fr", "en-US", "en"}, "ES": {"es-ES", "es", "en"},
		"IT": {"it-IT", "it", "en"}, "NL": {"nl-NL", "nl", "en"}, "BR": {"pt-BR", "pt", "en"}, "PT": {"pt-PT", "pt", "en"},
		"RU": {"ru-RU", "ru", "en"}, "UA": {"uk-UA", "uk", "ru", "en"}, "PL": {"pl-PL", "pl", "en"}, "TR": {"tr-TR", "tr", "en"},
		"JP": {"ja-JP", "ja", "en"}, "KR": {"ko-KR", "ko", "en"}, "CN": {"zh-CN", "zh", "en"}, "TW": {"zh-TW", "zh", "en"},
		"HK": {"zh-HK", "zh", "en"}, "SG": {"en-SG", "en", "zh-CN"}, "VN": {"vi-VN", "vi", "en"}, "TH": {"th-TH", "th", "en"},
		"ID": {"id-ID", "id", "en"}, "IN": {"en-IN", "en", "hi"}, "MX": {"es-MX", "es", "en"}, "AR": {"es-AR", "es", "en"},
	}
	if v, ok := m[cc]; ok {
		return v
	}
	return []string{"en-US", "en"}
}
