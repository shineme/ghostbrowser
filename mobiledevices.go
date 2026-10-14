package main

import (
	"fmt"
	"strings"
)

// Device types beyond desktop/laptop. Phones, tablets, foldables, Chromebooks
// and TVs run on the open-source kernel and are emulated through CDP
// (the same device mode Chrome DevTools uses): UA + UA-CH, viewport, DPR,
// touch, screen orientation; the page script adds GPU, noise and fonts.

// DeviceTypes lists the categories the editor offers, in display order.
var DeviceTypes = []DeviceTypeInfo{
	{"desktop", "台式机", []string{"windows", "macos", "linux"}, "high"},
	{"laptop", "笔记本", []string{"windows", "macos", "linux"}, "high"},
	{"phone", "安卓手机", []string{"android"}, "mid"},
	{"tablet", "安卓平板", []string{"android"}, "mid"},
	{"foldable", "折叠屏", []string{"android"}, "mid"},
	{"chromebook", "Chromebook", []string{"chromeos"}, "mid"},
	{"tv", "电视", []string{"android", "tizen", "webos"}, "low"},
}

type DeviceTypeInfo struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	OSes  []string `json:"oses"`
	Trust string   `json:"trust"` // high | mid | low: how well a Chromium kernel can pass as this device
}

func normDevType(t string) string {
	if t == "" {
		return "desktop"
	}
	return t
}

func isMobileType(t string) bool { return t == "phone" || t == "foldable" }

// emulatedViewport: types whose viewport/screen/DPR come from CDP device metrics.
func emulatedViewport(t string) bool {
	return t == "phone" || t == "tablet" || t == "foldable" || t == "tv"
}

type mdev struct {
	typ, os, osVer, name, model string
	w, h                        int
	dpr                         float64
	vendor, renderer            string
	cores, mem, touch           int
	ua                          string // fixed UA (TVs ship old Chromium builds)
	platform                    string
}

const (
	qcom = "Qualcomm"
	arm  = "ARM"
	sams = "Samsung Electronics Co., Ltd."
)

var mobileDevices = []mdev{
	// ---- Android phones ----
	{typ: "phone", os: "android", osVer: "14", name: "Samsung Galaxy S24 Ultra", model: "SM-S928B", w: 412, h: 915, dpr: 2.625, vendor: qcom, renderer: "Adreno (TM) 750", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "Samsung Galaxy S24", model: "SM-S921B", w: 360, h: 780, dpr: 3, vendor: sams, renderer: "Samsung Xclipse 940", cores: 10, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "Samsung Galaxy S23 Ultra", model: "SM-S918B", w: 412, h: 915, dpr: 2.625, vendor: qcom, renderer: "Adreno (TM) 740", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "Samsung Galaxy S23", model: "SM-S911B", w: 360, h: 780, dpr: 3, vendor: qcom, renderer: "Adreno (TM) 740", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "13", name: "Samsung Galaxy S22", model: "SM-S901B", w: 360, h: 780, dpr: 3, vendor: sams, renderer: "Samsung Xclipse 920", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "Samsung Galaxy A54 5G", model: "SM-A546B", w: 384, h: 854, dpr: 2.8125, vendor: arm, renderer: "Mali-G68", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "Samsung Galaxy A34 5G", model: "SM-A346B", w: 412, h: 915, dpr: 2.625, vendor: arm, renderer: "Mali-G68", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "Samsung Galaxy A15", model: "SM-A155F", w: 412, h: 892, dpr: 2.625, vendor: arm, renderer: "Mali-G57", cores: 8, mem: 4, touch: 10},
	{typ: "phone", os: "android", osVer: "15", name: "Google Pixel 9 Pro", model: "Pixel 9 Pro", w: 410, h: 914, dpr: 3.125, vendor: arm, renderer: "Mali-G715", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "15", name: "Google Pixel 9", model: "Pixel 9", w: 412, h: 924, dpr: 2.625, vendor: arm, renderer: "Mali-G715", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "Google Pixel 8 Pro", model: "Pixel 8 Pro", w: 448, h: 998, dpr: 3, vendor: arm, renderer: "Mali-G715", cores: 9, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "Google Pixel 8", model: "Pixel 8", w: 412, h: 915, dpr: 2.625, vendor: arm, renderer: "Mali-G715", cores: 9, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "Google Pixel 7", model: "Pixel 7", w: 412, h: 915, dpr: 2.625, vendor: arm, renderer: "Mali-G710", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "Google Pixel 7a", model: "Pixel 7a", w: 412, h: 915, dpr: 2.625, vendor: arm, renderer: "Mali-G710", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "13", name: "Google Pixel 6a", model: "Pixel 6a", w: 412, h: 915, dpr: 2.625, vendor: arm, renderer: "Mali-G78", cores: 8, mem: 4, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "Xiaomi 14", model: "23127PN0CG", w: 393, h: 873, dpr: 2.75, vendor: qcom, renderer: "Adreno (TM) 750", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "Redmi Note 13 Pro 5G", model: "23090RA98G", w: 393, h: 873, dpr: 2.75, vendor: qcom, renderer: "Adreno (TM) 710", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "13", name: "Redmi Note 12", model: "23021RAA2Y", w: 393, h: 873, dpr: 2.75, vendor: qcom, renderer: "Adreno (TM) 610", cores: 8, mem: 4, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "POCO X6 Pro", model: "23122PCD1G", w: 393, h: 873, dpr: 2.75, vendor: arm, renderer: "Mali-G615", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "OnePlus 12", model: "CPH2581", w: 412, h: 915, dpr: 3.5, vendor: qcom, renderer: "Adreno (TM) 750", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "13", name: "OnePlus Nord CE 3 Lite", model: "CPH2465", w: 412, h: 915, dpr: 2.625, vendor: qcom, renderer: "Adreno (TM) 619", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "OPPO Reno11", model: "CPH2599", w: 412, h: 915, dpr: 2.625, vendor: arm, renderer: "Mali-G68", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "vivo V30", model: "V2318", w: 393, h: 873, dpr: 2.75, vendor: qcom, renderer: "Adreno (TM) 720", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "13", name: "Motorola moto g84 5G", model: "moto g84 5G", w: 412, h: 915, dpr: 2.625, vendor: qcom, renderer: "Adreno (TM) 619", cores: 8, mem: 8, touch: 10},
	{typ: "phone", os: "android", osVer: "14", name: "Sony Xperia 1 V", model: "XQ-DQ72", w: 411, h: 960, dpr: 4, vendor: qcom, renderer: "Adreno (TM) 740", cores: 8, mem: 8, touch: 10},

	// ---- Android tablets (Chrome drops "Mobile" from the UA) ----
	{typ: "tablet", os: "android", osVer: "14", name: "Samsung Galaxy Tab S9", model: "SM-X710", w: 800, h: 1280, dpr: 2, vendor: qcom, renderer: "Adreno (TM) 740", cores: 8, mem: 8, touch: 10},
	{typ: "tablet", os: "android", osVer: "13", name: "Samsung Galaxy Tab S8 Ultra", model: "SM-X900", w: 924, h: 1480, dpr: 2, vendor: qcom, renderer: "Adreno (TM) 730", cores: 8, mem: 8, touch: 10},
	{typ: "tablet", os: "android", osVer: "14", name: "Samsung Galaxy Tab S9 FE", model: "SM-X510", w: 720, h: 1152, dpr: 2, vendor: arm, renderer: "Mali-G68", cores: 8, mem: 8, touch: 10},
	{typ: "tablet", os: "android", osVer: "14", name: "Samsung Galaxy Tab A9+", model: "SM-X210", w: 600, h: 960, dpr: 2, vendor: qcom, renderer: "Adreno (TM) 619", cores: 8, mem: 4, touch: 10},
	{typ: "tablet", os: "android", osVer: "14", name: "Google Pixel Tablet", model: "Pixel Tablet", w: 800, h: 1280, dpr: 2, vendor: arm, renderer: "Mali-G710", cores: 8, mem: 8, touch: 10},
	{typ: "tablet", os: "android", osVer: "13", name: "Xiaomi Pad 6", model: "23043RP34G", w: 800, h: 1280, dpr: 2.25, vendor: qcom, renderer: "Adreno (TM) 650", cores: 8, mem: 8, touch: 10},
	{typ: "tablet", os: "android", osVer: "13", name: "Lenovo Tab P11 (2nd Gen)", model: "TB350FU", w: 800, h: 1333, dpr: 1.5, vendor: arm, renderer: "Mali-G57", cores: 8, mem: 4, touch: 10},
	{typ: "tablet", os: "android", osVer: "14", name: "OnePlus Pad", model: "OPD2203", w: 800, h: 1120, dpr: 2.5, vendor: arm, renderer: "Mali-G710", cores: 8, mem: 8, touch: 10},

	// ---- Foldables (phone UA; folded and unfolded are separate presets) ----
	{typ: "foldable", os: "android", osVer: "14", name: "Samsung Galaxy Z Fold6 · 折叠", model: "SM-F956B", w: 344, h: 882, dpr: 2.625, vendor: qcom, renderer: "Adreno (TM) 750", cores: 8, mem: 8, touch: 10},
	{typ: "foldable", os: "android", osVer: "14", name: "Samsung Galaxy Z Fold6 · 展开", model: "SM-F956B", w: 707, h: 823, dpr: 2.625, vendor: qcom, renderer: "Adreno (TM) 750", cores: 8, mem: 8, touch: 10},
	{typ: "foldable", os: "android", osVer: "14", name: "Samsung Galaxy Z Flip6", model: "SM-F741B", w: 412, h: 1006, dpr: 2.625, vendor: qcom, renderer: "Adreno (TM) 750", cores: 8, mem: 8, touch: 10},
	{typ: "foldable", os: "android", osVer: "15", name: "Google Pixel 9 Pro Fold · 折叠", model: "Pixel 9 Pro Fold", w: 412, h: 924, dpr: 2.625, vendor: arm, renderer: "Mali-G715", cores: 8, mem: 8, touch: 10},
	{typ: "foldable", os: "android", osVer: "15", name: "Google Pixel 9 Pro Fold · 展开", model: "Pixel 9 Pro Fold", w: 791, h: 820, dpr: 2.625, vendor: arm, renderer: "Mali-G715", cores: 8, mem: 8, touch: 10},

	// ---- Chromebooks ----
	{typ: "chromebook", os: "chromeos", osVer: "16093.68.0", name: "Acer Chromebook Plus 515", w: 1536, h: 864, dpr: 1.25, vendor: "Google Inc. (Intel)", renderer: "ANGLE (Intel, Mesa Intel(R) UHD Graphics (ADL GT2), OpenGL ES 3.2)", cores: 8, mem: 8},
	{typ: "chromebook", os: "chromeos", osVer: "16093.68.0", name: "HP Chromebook x360 14c", w: 1536, h: 864, dpr: 1.25, vendor: "Google Inc. (Intel)", renderer: "ANGLE (Intel, Mesa Intel(R) Xe Graphics (TGL GT2), OpenGL ES 3.2)", cores: 8, mem: 8, touch: 10},
	{typ: "chromebook", os: "chromeos", osVer: "16063.45.0", name: "ASUS Chromebook CX1", w: 1366, h: 768, dpr: 1, vendor: "Google Inc. (Intel)", renderer: "ANGLE (Intel, Mesa Intel(R) UHD Graphics (JSL), OpenGL ES 3.2)", cores: 2, mem: 4},
	{typ: "chromebook", os: "chromeos", osVer: "16063.45.0", name: "Samsung Galaxy Chromebook Go", w: 1366, h: 768, dpr: 1, vendor: "Google Inc. (Intel)", renderer: "ANGLE (Intel, Mesa Intel(R) UHD Graphics (JSL), OpenGL ES 3.2)", cores: 2, mem: 4},
	{typ: "chromebook", os: "chromeos", osVer: "16063.45.0", name: "Lenovo Chromebook Duet", w: 960, h: 600, dpr: 2, vendor: "Google Inc. (ARM)", renderer: "ANGLE (ARM, Mali-G72, OpenGL ES 3.2)", cores: 8, mem: 4, touch: 10, platform: "Linux aarch64"},

	// ---- TVs (their browsers ship old Chromium; low trust with a current kernel) ----
	{typ: "tv", os: "android", osVer: "12", name: "Chromecast with Google TV", model: "Chromecast", w: 960, h: 540, dpr: 2, vendor: arm, renderer: "Mali-G31", cores: 4, mem: 2,
		ua: "Mozilla/5.0 (Linux; Android 12; Chromecast) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 Safari/537.36"},
	{typ: "tv", os: "android", osVer: "12", name: "Sony BRAVIA (Google TV)", model: "BRAVIA VH2", w: 960, h: 540, dpr: 2, vendor: arm, renderer: "Mali-G52", cores: 4, mem: 2,
		ua: "Mozilla/5.0 (Linux; Android 12; BRAVIA VH2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 Safari/537.36"},
	{typ: "tv", os: "tizen", osVer: "8.0", name: "Samsung Smart TV (Tizen 8)", w: 1920, h: 1080, dpr: 1, vendor: arm, renderer: "Mali-G52", cores: 4, mem: 2, platform: "Linux armv7l",
		ua: "Mozilla/5.0 (SMART-TV; LINUX; Tizen 8.0) AppleWebKit/537.36 (KHTML, like Gecko) 108.0.5359.1/8.0 TV Safari/537.36"},
	{typ: "tv", os: "webos", osVer: "24", name: "LG Smart TV (webOS 24)", w: 1920, h: 1080, dpr: 1, vendor: arm, renderer: "Mali-G52", cores: 4, mem: 2, platform: "Linux armv7l",
		ua: "Mozilla/5.0 (Web0S; Linux/SmartTV) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.5359.211 Safari/537.36 WebAppManager"},
}

var mobilePresets = func() []DevicePreset {
	out := []DevicePreset{}
	for i, d := range mobileDevices {
		out = append(out, DevicePreset{
			ID: fmt.Sprintf("%s-%03d", d.typ, i+1), Name: fmt.Sprintf("%s · %dx%d", d.name, d.w, d.h),
			OS: d.os, OSVersion: d.osVer, Vendor: d.vendor, Renderer: d.renderer,
			Width: d.w, Height: d.h, DPR: d.dpr, Cores: d.cores, Memory: capMem(d.mem), Touch: d.touch,
			DeviceType: d.typ, Model: d.model, DeviceName: d.name, UA: d.ua, Platform: d.platform,
		})
	}
	return out
}()

var androidFonts = []string{"Roboto", "Noto Sans", "Noto Serif", "Noto Color Emoji", "Droid Sans Mono", "Cutive Mono", "Coming Soon", "Dancing Script", "Carrois Gothic SC", "Noto Sans CJK SC", "Noto Sans Arabic", "Noto Sans Devanagari", "Noto Sans Thai", "Source Sans Pro"}
var chromeosFonts = []string{"Roboto", "Noto Sans", "Noto Serif", "Noto Color Emoji", "Arimo", "Tinos", "Cousine", "Noto Sans CJK SC", "Noto Sans Mono", "Google Sans"}
var tvFonts = []string{"Roboto", "Noto Sans", "Noto Serif", "Noto Sans CJK SC", "Noto Color Emoji"}

// buildUAFor builds the User-Agent and navigator.platform for an OS + device type.
// Android and ChromeOS use Chrome's reduced UA (fixed "Android 10; K" / CrOS token);
// the real model and version travel in UA-CH, set at launch.
func buildUAFor(osName, devType, osVer, major string) (ua, platform string) {
	switch osName {
	case "android":
		mobile := "Mobile "
		if devType == "tablet" || devType == "tv" {
			mobile = ""
		}
		return fmt.Sprintf("Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 %sSafari/537.36", major, mobile), "Linux armv81"
	case "chromeos":
		return fmt.Sprintf("Mozilla/5.0 (X11; CrOS x86_64 14541.0.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 Safari/537.36", major), "Linux x86_64"
	case "tizen":
		return "Mozilla/5.0 (SMART-TV; LINUX; Tizen 8.0) AppleWebKit/537.36 (KHTML, like Gecko) 108.0.5359.1/8.0 TV Safari/537.36", "Linux armv7l"
	case "webos":
		return "Mozilla/5.0 (Web0S; Linux/SmartTV) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.5359.211 Safari/537.36 WebAppManager", "Linux armv7l"
	}
	return buildUA(osName, osVer, major)
}

// presetUA applies a preset's own UA / platform when it has one (TVs, ARM Chromebooks).
func presetUA(d DevicePreset, major string) (string, string) {
	ua, pf := buildUAFor(d.OS, d.DeviceType, d.OSVersion, major)
	if d.UA != "" {
		ua = d.UA
		if strings.Contains(ua, "%s") {
			ua = fmt.Sprintf(ua, major)
		}
	}
	if d.Platform != "" {
		pf = d.Platform
		if d.OS == "chromeos" && strings.Contains(pf, "aarch64") {
			ua = strings.Replace(ua, "CrOS x86_64", "CrOS aarch64", 1)
		}
	}
	return ua, pf
}

func fontsFor(osName string) []string {
	switch osName {
	case "macos":
		return macFonts
	case "linux":
		return linuxFonts
	case "android":
		return androidFonts
	case "chromeos":
		return chromeosFonts
	case "tizen", "webos":
		return tvFonts
	}
	return winFonts
}

// PresetsByType lists the presets of one device type, optionally narrowed to one OS.
func PresetsByType(devType, osName string) []DevicePreset {
	devType = normDevType(devType)
	out := []DevicePreset{}
	for _, p := range DevicePresetsFor(osName) {
		if normDevType(p.DeviceType) == devType {
			out = append(out, p)
		}
	}
	return out
}
