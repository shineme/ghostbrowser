package main

import (
	"strings"
	"testing"
)

func TestGenerateTypedEveryType(t *testing.T) {
	for _, dt := range DeviceTypes {
		for _, osName := range dt.OSes {
			fp := GenerateTyped(dt.ID, osName, "141")
			if fp.OS != osName || normDevType(fp.DeviceType) != dt.ID {
				t.Fatalf("%s/%s: got %s/%s", dt.ID, osName, fp.OS, fp.DeviceType)
			}
			if fp.UserAgent == "" || fp.WebGLRenderer == "" || fp.ScreenWidth == 0 || len(fp.Fonts) == 0 {
				t.Fatalf("%s/%s incomplete: %+v", dt.ID, osName, fp)
			}
			mobile := strings.Contains(fp.UserAgent, "Mobile Safari")
			if mobile != isMobileType(fp.DeviceType) {
				t.Fatalf("%s/%s mobile token mismatch: %s", dt.ID, osName, fp.UserAgent)
			}
			if osName == "android" && fp.DeviceModel == "" {
				t.Fatalf("%s android without model", dt.ID)
			}
			if needsOpenKernel(fp) != !(osName == "windows" && (dt.ID == "desktop" || dt.ID == "laptop")) {
				t.Fatalf("%s/%s kernel rule wrong", dt.ID, osName)
			}
		}
	}
}

func TestUAOverrideMeta(t *testing.T) {
	fp := GenerateTyped("phone", "android", "141")
	o := uaOverride(fp, []map[string]string{{"brand": "Chromium", "version": "141"}}, []map[string]string{{"brand": "Chromium", "version": "141.0.1.2"}})
	m := o["userAgentMetadata"].(map[string]any)
	if m["platform"] != "Android" || m["mobile"] != true || m["model"] == "" || m["fullVersion"] != "141.0.1.2" {
		t.Fatalf("bad meta %+v", m)
	}
	if w := windowFor(GenerateTyped("tv", "tizen", "141")); w.scale >= 1 || w.w > 1280 {
		t.Fatalf("tv window not scaled: %+v", w)
	}
}
