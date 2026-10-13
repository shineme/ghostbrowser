package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func shaOf(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// TestEnsureInstallsRealPackages drives the whole pipeline against the actual
// delivery archives: download -> sha256 verify -> extract -> remap chrome/ to
// runtime/chrome/, using a manifest served by the fake shop.
func TestEnsureInstallsRealPackages(t *testing.T) {
	root := t.TempDir()
	realChrome := filepath.Join(os.TempDir(), "gbprobe", "chrome.zip")
	realKit := filepath.Join(os.TempDir(), "gbprobe", "undetectable-kit.zip")
	cz, err := os.ReadFile(realChrome)
	if err != nil {
		t.Skipf("delivery archives not staged at %s: %v", realChrome, err)
	}
	kz, err := os.ReadFile(realKit)
	if err != nil {
		t.Skipf("missing %s: %v", realKit, err)
	}

	// The fake shop serves both zips plus manifest.json, like the real one will.
	mux := http.NewServeMux()
	mux.HandleFunc("/chrome.zip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(cz)
	})
	mux.HandleFunc("/undetectable-kit.zip", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(kz) })
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(envManifest{
			Chrome: envManifestItem{SHA256: shaOf(cz), Size: int64(len(cz))},
			FpKit:  envManifestItem{SHA256: shaOf(kz), Size: int64(len(kz))},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	base := srv.URL
	oldChrome, oldKit, oldMan := chromeZipURL, fpKitZipURL, manifestFetch
	chromeZipURL, fpKitZipURL = base+"/chrome.zip", base+"/undetectable-kit.zip"
	manifestFetch = func(context.Context) (envManifest, bool) {
		resp, err := http.Get(base + "/manifest.json")
		if err != nil {
			return envManifest{}, false
		}
		defer resp.Body.Close()
		var m envManifest
		if json.NewDecoder(resp.Body).Decode(&m) != nil {
			return m, false
		}
		return m, true
	}
	defer func() { chromeZipURL, fpKitZipURL, manifestFetch = oldChrome, oldKit, oldMan }()

	kit := filepath.Join(root, "undetectable-kit")
	e := &envInstaller{}
	var events []EnvStatus
	e.emit = func(s EnvStatus) { events = append(events, s) }

	st := e.ensure(kit, false)
	if !st.Ready {
		t.Fatalf("install failed: phase=%s error=%s", st.Phase, st.Error)
	}
	exe := filepath.Join(kit, "runtime", "chrome", "chrome.exe")
	if info, err := os.Stat(exe); err != nil || info.IsDir() {
		t.Fatalf("chrome.exe missing after install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(kit, "runtime", "chrome", "chrome.dll")); err != nil {
		t.Fatalf("chrome.dll missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(kit, "fingerprints", "library.json")); err != nil {
		t.Fatalf("library.json missing: %v", err)
	}
	tpls, _ := filepath.Glob(filepath.Join(kit, "fingerprints", "templates", "*.bin"))
	if len(tpls) != 3 {
		t.Fatalf("expected 3 templates, got %v", tpls)
	}
	// No leftover staging or partial files.
	if left, _ := filepath.Glob(filepath.Join(root, "_envsetup-work", "*")); len(left) > 0 {
		t.Errorf("work dir not cleaned: %v", left)
	}
	// Progress reached the UI in order, ending ready.
	if len(events) < 4 {
		t.Fatalf("expected progress events, got %d", len(events))
	}
	sawDownload, sawExtract := false, false
	for _, ev := range events {
		if ev.Phase == "downloading" && ev.Total > 0 && ev.Done >= 0 {
			sawDownload = true
		}
		if ev.Phase == "extracting" {
			sawExtract = true
		}
	}
	if !sawDownload || !sawExtract {
		t.Errorf("progress phases missing: download=%v extract=%v", sawDownload, sawExtract)
	}
	if last := events[len(events)-1]; last.Phase != "ready" || !last.Manifest {
		t.Errorf("final event = %+v", last)
	}

	// Second run is a no-op; force re-downloads.
	if st2 := e.ensure(kit, false); !st2.Ready || st2.Phase != "ready" {
		t.Errorf("re-run should report ready without work: %+v", st2)
	}
	before := readVersion(t, exe)
	if st3 := e.ensure(kit, true); !st3.Ready {
		t.Fatalf("force reinstall failed: %+v", st3)
	}
	if readVersion(t, exe) != before {
		t.Errorf("forced reinstall changed chrome.exe unexpectedly")
	}
}

func readVersion(t *testing.T, exe string) int64 {
	t.Helper()
	info, err := os.Stat(exe)
	if err != nil {
		t.Fatal(err)
	}
	return info.Size()
}

// TestEnsureRejectsTamperedPackage proves a bad sha256 stops the install and
// leaves no half-written kernel behind.
func TestEnsureRejectsTamperedPackage(t *testing.T) {
	root := t.TempDir()
	payload := []byte("not-a-real-chromium")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chrome.zip":
			_, _ = w.Write(payload)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldChrome, oldMan := chromeZipURL, manifestFetch
	chromeZipURL = srv.URL + "/chrome.zip"
	manifestFetch = func(context.Context) (envManifest, bool) {
		return envManifest{Chrome: envManifestItem{SHA256: shaOf([]byte("different"))}}, true
	}
	defer func() { chromeZipURL, manifestFetch = oldChrome, oldMan }()

	kit := filepath.Join(root, "undetectable-kit")
	e := &envInstaller{}
	st := e.ensure(kit, false)
	if st.Ready || st.Phase != "failed" {
		t.Fatalf("tampered package must fail, got %+v", st)
	}
	if _, err := os.Stat(filepath.Join(kit, "runtime", "chrome", "chrome.exe")); err == nil {
		t.Errorf("tampered payload was installed anyway")
	}
	if !contains(st.Error, "校验不通过") {
		t.Errorf("error should mention verification, got %q", st.Error)
	}
}

// TestSafeRel covers zip-slip entry names.
func TestSafeRel(t *testing.T) {
	for name, wantOK := range map[string]bool{
		"chrome/chrome.exe":     true,
		"./chrome/a.dll":        true,
		"../evil.txt":           false,
		"chrome/../../evil.txt": false,
		"/absolute/evil":        false,
		"C:\\Windows\\evil":     false,
		"undetectable-kit/fingerprints/library.json": true,
	} {
		got, ok := safeRel(name)
		if ok != wantOK {
			t.Errorf("safeRel(%q) ok=%v want %v", name, ok, wantOK)
		}
		if ok && filepath.IsAbs(got) {
			t.Errorf("safeRel(%q) returned absolute %q", name, got)
		}
	}
}

// TestPlacePackageRemapsChrome checks the layout fix that a blind extract misses.
func TestPlacePackageRemapsChrome(t *testing.T) {
	root := t.TempDir()
	stage := filepath.Join(root, "stage")
	writeFile(t, filepath.Join(stage, "chrome", "chrome.exe"), []byte("exe"))
	writeFile(t, filepath.Join(stage, "chrome", "153.0.8010.37.manifest"), []byte("m"))
	kit := filepath.Join(root, "kit")
	if err := placePackage("chrome", stage, kit); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(kit, "runtime", "chrome", "chrome.exe")); err != nil {
		t.Fatalf("chrome.exe not remapped into runtime/chrome: %v", err)
	}

	// fpkit: undetectable-kit/ wrapper folded into the kit root.
	fstage := filepath.Join(root, "fstage", "undetectable-kit")
	writeFile(t, filepath.Join(fstage, "fingerprints", "library.json"), []byte("{}"))
	writeFile(t, filepath.Join(fstage, "fingerprints", "templates", "windows-intel-a.bin"), []byte("t"))
	if err := placePackage("fpkit", filepath.Join(root, "fstage"), kit); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(kit, "fingerprints", "library.json")); err != nil {
		t.Fatalf("fpkit not flattened to kit root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(kit, "undetectable-kit")); err == nil {
		t.Errorf("fpkit left a nested undetectable-kit/ directory")
	}
}

// TestChromeMajorFrom covers both kernel layouts: the flat delivered package
// that only carries <version>.manifest, and a classic install with sibling
// version directories.
func TestChromeMajorFrom(t *testing.T) {
	root := t.TempDir()
	flat := filepath.Join(root, "chrome")
	if err := os.MkdirAll(flat, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(flat, "chrome.exe"), []byte("x"))
	writeFile(t, filepath.Join(flat, "153.0.8010.37.manifest"), []byte("m"))
	if got := chromeMajorFrom(flat); got != "153" {
		t.Errorf("flat layout major = %q, want 153", got)
	}

	classic := filepath.Join(root, "Application")
	for _, v := range []string{"141.0.7390.122", "152.0.1111.22"} {
		if err := os.MkdirAll(filepath.Join(classic, v), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if got := chromeMajorFrom(classic); got != "152" {
		t.Errorf("version-dir layout major = %q, want 152", got)
	}

	if got := chromeMajorFrom(filepath.Join(root, "nope")); got != "" {
		t.Errorf("missing dir major = %q, want empty", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
