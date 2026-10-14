package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestFpExtraSeed(t *testing.T) {
	a1, a2 := effectiveSeed("p1", 42, false), effectiveSeed("p1", 42, false)
	b := effectiveSeed("p2", 42, false)
	if a1 != a2 || a1 == b {
		t.Fatalf("seed determinism: %d %d %d", a1, a2, b)
	}
	if effectiveSeed("p1", 43, false) == a1 {
		t.Fatal("noise seed ignored")
	}
	r1, r2 := effectiveSeed("p1", 42, true), effectiveSeed("p1", 42, true)
	if r1 == r2 {
		t.Fatal("random seed repeated")
	}
	for _, s := range []int64{a1, b, r1, r2} {
		if s < 1 || s >= 2147483647 {
			t.Fatalf("seed out of range %d", s)
		}
	}
}

func TestFpExtraModes(t *testing.T) {
	if fpMode("", true) != "noise" || fpMode("", false) != "real" || fpMode("off", true) != "off" || fpMode("real", true) != "real" {
		t.Fatal("fpMode")
	}
	if mediaMode(Fingerprint{HideMedia: true}) != "off" || mediaMode(Fingerprint{HideMedia: true, MediaDevicesMode: "custom"}) != "custom" {
		t.Fatal("mediaMode")
	}
}

func TestFpExtraFlags(t *testing.T) {
	off := false
	fp := Fingerprint{ScreenWidth: 1920, ScreenHeight: 1080, WindowMode: "custom", WindowWidth: 1200, WindowHeight: 800, WindowX: 10, WindowY: 20,
		HardwareAcceleration: &off, BlockImages: true, BlockSound: true, WebGPU: "off"}
	got := launchFlags(fp)
	want := []string{"--window-size=1200,800", "--window-position=10,20", "--disable-gpu", "--blink-settings=imagesEnabled=false", "--mute-audio", "--disable-blink-features=WebGPU"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("flags %q", got)
	}
	if f := launchFlags(Fingerprint{ScreenWidth: 1920, ScreenHeight: 1080, WindowMode: "maximized"}); !reflect.DeepEqual(f, []string{"--start-maximized"}) {
		t.Fatalf("maximized %q", f)
	}
	if f := launchFlags(Fingerprint{ScreenWidth: 1920, ScreenHeight: 1080}); !reflect.DeepEqual(f, []string{"--window-size=1536,864"}) {
		t.Fatalf("auto %q", f)
	}
	on := true
	if f := strings.Join(launchFlags(Fingerprint{ScreenWidth: 1920, ScreenHeight: 1080, HardwareAcceleration: &on}), " "); strings.Contains(f, "disable-gpu") {
		t.Fatal("gpu disabled while on")
	}
}

func TestFpExtraStartupParams(t *testing.T) {
	keep, rej := parseStartupParams(`--disable-features=Foo,Translate --user-data-dir=/x "--window-size=800,600" --remote-debugging-port=1 --proxy-server=x --user-agent="a b" stray --lang='zh CN' --blink-settings=imagesEnabled=true,dark=1`)
	wantKeep := []string{"--disable-features=Foo,Translate", "--window-size=800,600", "--lang=zh CN", "--blink-settings=imagesEnabled=true,dark=1"}
	if !reflect.DeepEqual(keep, wantKeep) {
		t.Fatalf("keep %q", keep)
	}
	if len(rej) != 5 {
		t.Fatalf("rejected %q", rej)
	}
	base := []string{"--user-data-dir=/d", "--disable-features=Translate,MediaRouter", "--window-size=1536,864", "--blink-settings=imagesEnabled=false"}
	got := mergeFlags(append(base, keep...))
	want := []string{"--user-data-dir=/d", "--disable-features=Translate,MediaRouter,Foo", "--window-size=800,600", "--blink-settings=imagesEnabled=true,dark=1", "--lang=zh CN"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("merge %q", got)
	}
	got = mergeFlags([]string{"--enable-features=A", "--enable-features=B,A", "about:blank"})
	if !reflect.DeepEqual(got, []string{"--enable-features=A,B", "about:blank"}) {
		t.Fatalf("enable merge %q", got)
	}
}

func TestFpExtraPortScanRules(t *testing.T) {
	n := newFpNet(Fingerprint{PortScanProtection: true, ScanWhitelist: "9222, localhost:3000"})
	n.topURL["s"] = "https://example.com/"
	n.topFrame["s"] = "F"
	cases := []struct {
		url, typ, frame string
		ok              bool
	}{
		{"http://127.0.0.1:8080/", "XHR", "F", false},
		{"http://localhost:3000/x", "Fetch", "F", true},
		{"http://127.0.0.1:9222/json", "Fetch", "F", true},
		{"http://[::1]:5000/", "Image", "F", false},
		{"http://127.0.0.1:8080/", "Document", "F", true},
		{"http://127.0.0.1:8080/", "Document", "child", false},
		{"https://example.org/", "XHR", "F", true},
	}
	for _, c := range cases {
		if got := n.allowLocal("s", c.url, c.typ, c.frame); got != c.ok {
			t.Errorf("%s %s: got %v", c.url, c.typ, got)
		}
	}
	n.topURL["s"] = "http://localhost:8000/"
	if !n.allowLocal("s", "http://127.0.0.1:8080/", "XHR", "F") {
		t.Error("local page should reach local services")
	}
	if newFpNet(Fingerprint{}) != nil {
		t.Error("guard should be off by default")
	}
}

func TestFpExtraVoices(t *testing.T) {
	w := speechVoicesFor("windows", []string{"zh-CN"})
	if len(w) == 0 || !strings.HasPrefix(w[0]["name"].(string), "Microsoft") {
		t.Fatalf("windows voices %v", w)
	}
	if m := speechVoicesFor("macos", nil); m[0]["name"] != "Samantha" {
		t.Fatal("mac voices")
	}
	if len(speechVoicesFor("android", nil)) != 0 {
		t.Fatal("android voices")
	}
}
