package main

import (
	"strings"
	"testing"
)

func TestUserLibrary(t *testing.T) {
	userLib.open(t.TempDir())
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"
	if e := userLib.Add("macos", "userAgents", ua); e != nil {
		t.Fatal(e)
	}
	_ = userLib.Add("macos", "gpus", "Google Inc. (Apple)\x1fANGLE (Apple, ANGLE Metal Renderer: Apple M5, Unspecified Version)")
	if e := userLib.Add("macos", "bogus", "x"); e == nil {
		t.Fatal("unknown field accepted")
	}
	c := FingerprintCatalog("macos", "141")
	if c.UserAgents[0] != ua || len(c.Custom["userAgents"]) != 1 {
		t.Fatalf("custom UA not first: %v", c.UserAgents[:2])
	}
	if !strings.Contains(c.GPUs[0]["renderer"], "M5") || c.GPUs[0]["custom"] != "1" {
		t.Fatalf("custom gpu: %v", c.GPUs[0])
	}
	if w := FingerprintCatalog("windows", "141"); len(w.Custom["userAgents"]) != 0 {
		t.Fatal("mac UA leaked into windows")
	}
	p, _ := userLib.SavePreset(DevicePreset{OS: "macos", Renderer: "ANGLE (Apple, ANGLE Metal Renderer: Apple M5, Unspecified Version)", Width: 1512, Height: 982, DPR: 2})
	if l := DevicePresetsFor("macos"); l[0].ID != p.ID || !l[0].Custom {
		t.Fatal("preset not listed first")
	}
	userLib.open(userLib.path[:len(userLib.path)-len("/library.json")]) // reload from disk
	if len(userLib.presets()) != 1 || len(userLib.values("macos", "userAgents")) != 1 {
		t.Fatal("library not persisted")
	}
	_ = userLib.Remove("macos", "userAgents", ua)
	_ = userLib.DeletePreset(p.ID)
	if len(userLib.presets()) != 0 || len(userLib.values("macos", "userAgents")) != 0 {
		t.Fatal("remove failed")
	}
}
