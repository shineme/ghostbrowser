package main

import (
	"encoding/json"
	"ghostbrowser/internal/undetectable"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// This opt-in check exercises the actual Store/App import/export/launch path
// against the bundled browser, with its own JSON store and browser data.
func TestNativeJSONIntegration(t *testing.T) {
	if os.Getenv("GHOST_NATIVE_TEST") != "1" {
		t.Skip("set GHOST_NATIVE_TEST=1 to launch the bundled Chromium")
	}
	kit := findUndetectableKit()
	if kit == "" {
		t.Fatal("kit missing")
	}
	root := t.TempDir()
	s := &Store{dir: root, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}, Settings: Settings{UndetectableKit: kit, KernelPath: filepath.Join(kit, "runtime", "chrome", "chrome.exe"), KernelMode: "undetectable", ProfilesDir: filepath.Join(root, "profiles")}}
	a := &App{store: s}
	spec, err := undetectable.LoadSpec(filepath.Join(kit, "examples", "native-profile.json"))
	if err != nil {
		t.Fatal(err)
	}
	spec.Name = "JSON integration"
	spec.Headless = true
	spec.Data["CPUCores"] = json.RawMessage(`6`)
	spec.Data["memory"] = json.RawMessage(`8`)
	spec.Data["resConfig"] = json.RawMessage(`"1440x900"`)
	spec.Data["tzId"] = json.RawMessage(`"Asia/Shanghai"`)
	spec.Data["langs"] = json.RawMessage(`"fr-FR#fr"`)
	spec.Data["langHeader"] = json.RawMessage(`"fr-FR,fr;q=0.9"`)
	spec.ConfigMeta["opaquePreservationMarker"] = json.RawMessage(`{"kept":true}`)
	spec.Runtime.PixelRatio = 1.5
	for _, key := range []string{"windows-intel-a", "windows-intel-a2", "windows-intel-b"} {
		t.Run(key, func(t *testing.T) {
			spec.TemplatePath = filepath.Join(kit, "fingerprints", "templates", templates[key])
			raw, _ := json.Marshal(spec)
			p, err := a.ImportNativeJSON(string(raw))
			if err != nil {
				t.Fatal(err)
			}
			exported, err := a.ExportNativeJSON(p.ID)
			if err != nil {
				t.Fatal(err)
			}
			var roundtrip undetectable.Spec
			if err = json.Unmarshal([]byte(exported), &roundtrip); err != nil {
				t.Fatal(err)
			}
			var marker struct {
				Kept bool `json:"kept"`
			}
			_ = json.Unmarshal(roundtrip.ConfigMeta["opaquePreservationMarker"], &marker)
			if !marker.Kept {
				t.Fatal("unknown metadata lost")
			}
			if p.Fingerprint.HardwareConc != 6 || p.Fingerprint.ScreenWidth != 1440 {
				t.Fatal("JSON fingerprint not imported")
			}
			rs, err := a.StartProfile(p.ID)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				runReg.stop(p.ID)
				deadline := time.Now().Add(10 * time.Second)
				for time.Now().Before(deadline) {
					if _, ok := runReg.get(p.ID); !ok {
						return
					}
					time.Sleep(100 * time.Millisecond)
				}
				t.Error("run registry retained closed profile")
			}()
			rs2, err := a.StartProfile(p.ID)
			if err != nil || rs2.PID != rs.PID {
				t.Fatal("repeated launch did not reuse existing session")
			}
			if second, e := roundtrip.Start(0); e == nil {
				second.Close()
				t.Fatal("cross-process profile lock did not reject a second launcher")
			}
			snapshot, err := a.NativeSnapshot(p.ID)
			if err != nil {
				t.Fatal(err)
			}
			var observed struct {
				Live       undetectable.Fingerprint `json:"live"`
				Mismatches []string                 `json:"mismatches"`
			}
			if err = json.Unmarshal([]byte(snapshot), &observed); err != nil {
				t.Fatal(err)
			}
			if len(observed.Mismatches) > 0 {
				t.Fatalf("fingerprint mismatch: %v", observed.Mismatches)
			}
			if observed.Live.Cores != 6 || observed.Live.Memory != 8 || observed.Live.Timezone != "Asia/Shanghai" {
				t.Fatalf("custom JSON values not applied: %+v", observed.Live)
			}
			if observed.Live.DPR != 1.5 || observed.Live.Screen[0] != 1440 {
				t.Fatal("screen overrides not applied")
			}
			t.Logf("native import/export, new-tab UA-CH, profile reuse and custom values passed; pid=%d", rs.PID)
		})
	}
}
