package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnsureAgainstRealShop runs the production path end-to-end: the actual
// shop URLs, the real manifest-or-builtin hash, and a live 213 MB download.
// Skipped unless GB_LIVE_SHOP=1 so it never fires in ordinary CI.
func TestEnsureAgainstRealShop(t *testing.T) {
	if os.Getenv("GB_LIVE_SHOP") != "1" {
		t.Skip("set GB_LIVE_SHOP=1 to install from the live shop URLs")
	}
	root := t.TempDir()
	kit := filepath.Join(root, "undetectable-kit")
	e := &envInstaller{}
	var last EnvStatus
	e.emit = func(s EnvStatus) { last = s }

	st := e.ensure(kit, false)
	t.Logf("phases seen; final=%s manifestFromServer=%v", st.Phase, st.Manifest)
	if !st.Ready {
		t.Fatalf("live install failed: %s (%s)", st.Error, last.Message)
	}
	exe := filepath.Join(kit, "runtime", "chrome", "chrome.exe")
	if _, err := os.Stat(exe); err != nil {
		t.Fatalf("chrome.exe absent: %v", err)
	}
	rep := inspectAt(kit)
	t.Logf("kit=%s templates=%v chrome=%s", rep.Kit, rep.Templates, rep.ChromeExe)

	// The installed tree must satisfy the same checks the launcher relies on.
	if _, err := kitPath(Settings{UndetectableKit: kit}); err != nil {
		t.Fatalf("kitPath rejects the installed kit: %v", err)
	}
	if defaultKernelPath(Settings{KernelMode: "undetectable", UndetectableKit: kit}) == "" {
		t.Errorf("defaultKernelPath did not resolve the installed chrome.exe")
	}
	// KernelMajor reads the version folder or flat manifest next to chrome.exe.
	major := chromeMajorFrom(filepath.Dir(exe))
	if major == "" {
		t.Errorf("no Chrome version detected next to chrome.exe")
	} else {
		t.Logf("detected kernel major = %s", major)
	}

	// The installed tree must produce a valid launch spec, and the real kernel
	// must boot and apply the fingerprint.
	s := &Store{dir: t.TempDir(), Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{},
		Settings: Settings{KernelMode: "undetectable", UndetectableKit: kit, ProfilesDir: t.TempDir(), KernelPath: exe}}
	n, fp, err := s.nativePreset("")
	if err != nil {
		t.Fatalf("nativePreset: %v", err)
	}
	p := s.PutProfile(Profile{Name: "live probe", Fingerprint: fp, Native: n, StartURLs: []string{"about:blank"}})
	rs, err := s.Launch(p.ID)
	if err != nil {
		t.Fatalf("Launch from installed kit failed: %v", err)
	}
	defer runReg.stop(p.ID)
	t.Logf("booted pid=%d cdp=%d", rs.PID, rs.DebugPort)
	snap, err := (&App{store: s}).NativeSnapshot(p.ID)
	if err != nil {
		t.Fatalf("NativeSnapshot: %v", err)
	}
	var live struct {
		Live struct {
			UserAgent string `json:"userAgent"`
		} `json:"live"`
	}
	if json.Unmarshal([]byte(snap), &live) != nil || !strings.Contains(live.Live.UserAgent, "Chrome/") {
		t.Errorf("kernel did not report a Chrome UA, snapshot=%s", snap)
	} else {
		t.Logf("live UA = %s", live.Live.UserAgent)
	}
}
