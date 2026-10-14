package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Open-source kernel version compatibility (see docs/chromium-compat.md).
//
// Tested on Chromium 110–155 and ungoogled-chromium 155. Below 110 the CDP
// surface GhostBrowser relies on (UA-CH metadata overrides, browser-level
// flattened auto-attach, locale override) is untested, so such kernels are
// refused. 110–119 work but need the legacy start-up path (new tabs reload
// once so the document-start script applies), so the UI shows a warning.
const (
	chromiumMinMajor         = 110 // refuse to launch below this
	chromiumRecommendedMajor = 120 // warn below this
	chromiumLegacyBelow      = 120 // new tabs need the reload fallback below this
)

var fullVerRe = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)\.(\d+)`)

// compareVersions compares dotted numeric versions ("99.0.1.2" < "120.0.0.0").
func compareVersions(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

// chromeFullFrom finds the newest x.y.z.w version folder or <version>.manifest
// next to a Windows chrome.exe (Chrome, Chromium, Edge, CfT, ungoogled).
func chromeFullFrom(dir string) string {
	best := ""
	ents, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range ents {
		name := e.Name()
		m := fullVerRe.FindString(name)
		if m == "" || !strings.HasPrefix(name, m) {
			continue
		}
		if (e.IsDir() && name == m) || strings.HasSuffix(name, ".manifest") {
			if best == "" || compareVersions(m, best) > 0 {
				best = m
			}
		}
	}
	return best
}

type kernelVerEntry struct {
	mod  time.Time
	full string
}

var kernelVerCache sync.Map // path -> kernelVerEntry

// kernelVersion returns the kernel's full version ("" when unknown). It reads
// the version folder next to the executable, and on Linux/macOS (where
// `--version` prints and exits) asks the binary itself. Windows chrome.exe
// opens a window for --version, so it is never run there.
func kernelVersion(path string) string {
	if path == "" {
		return ""
	}
	fi, err := os.Stat(path)
	if err != nil {
		return ""
	}
	if v, ok := kernelVerCache.Load(path); ok && v.(kernelVerEntry).mod.Equal(fi.ModTime()) {
		return v.(kernelVerEntry).full
	}
	full := chromeFullFrom(filepath.Dir(path))
	if full == "" && strings.HasSuffix(strings.ToLower(path), ".exe") {
		full = peProductVersion(path) // portable builds without a version folder
	}
	if full == "" && runtime.GOOS != "windows" {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		out, _ := exec.CommandContext(ctx, path, "--version").Output()
		cancel()
		full = fullVerRe.FindString(string(out))
	}
	kernelVerCache.Store(path, kernelVerEntry{fi.ModTime(), full})
	return full
}

// peProductVersion reads the "ProductVersion" string from a Windows
// executable's VS_VERSIONINFO resource (UTF-16LE key followed by its value),
// without running it.
func peProductVersion(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	if fi, err := f.Stat(); err != nil || fi.Size() > 512<<20 {
		return ""
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return ""
	}
	key := utf16le("ProductVersion")
	for off := 0; ; {
		i := bytes.Index(b[off:], key)
		if i < 0 {
			return ""
		}
		j := off + i + len(key)
		// skip NUL terminator and 32-bit padding, then read the UTF-16 value
		for j+1 < len(b) && b[j] == 0 && b[j+1] == 0 {
			j += 2
		}
		var sb strings.Builder
		for ; j+1 < len(b) && b[j+1] == 0 && b[j] != 0 && sb.Len() < 32; j += 2 {
			sb.WriteByte(b[j])
		}
		if v := fullVerRe.FindString(sb.String()); v != "" {
			return v
		}
		off += i + len(key)
	}
}

func utf16le(s string) []byte {
	out := make([]byte, 0, len(s)*2)
	for _, r := range s {
		out = append(out, byte(r), 0)
	}
	return out
}

func majorOf(full string) int {
	m, _ := strconv.Atoi(strings.SplitN(full, ".", 2)[0])
	return m
}

// KernelCompat describes the open-source kernel for the settings page.
type KernelCompat struct {
	Path    string `json:"path"`
	Version string `json:"version"` // full version, "" when unknown
	Major   int    `json:"major"`
	Level   string `json:"level"`   // ok | warn | unsupported | unknown | missing
	Message string `json:"message"` // Chinese, empty when ok
}

func kernelCompatFor(path, full string) KernelCompat {
	k := KernelCompat{Path: path, Version: full, Major: majorOf(full), Level: "ok"}
	lower := strings.ToLower(filepath.Base(path))
	switch {
	case path == "":
		k.Level, k.Message = "missing", "没有找到开源内核，请选择 chrome.exe"
	case full == "":
		k.Level, k.Message = "unknown", "无法识别内核版本：启动时会按内核实际版本自动对齐 UA"
	case k.Major < chromiumMinMajor:
		k.Level, k.Message = "unsupported", fmt.Sprintf("Chrome %d 版本过旧，不受支持：需要 %d 及以上，推荐 %d 以上", k.Major, chromiumMinMajor, chromiumRecommendedMajor)
	case k.Major < chromiumRecommendedMajor:
		k.Level, k.Message = "warn", fmt.Sprintf("Chrome %d 可以使用，但新标签页会多加载一次以注入指纹，窗口同步可能无效，建议升级到 %d 以上", k.Major, chromiumRecommendedMajor)
	}
	if k.Level == "ok" && isBrandedChrome(path) && k.Major >= 137 {
		k.Level, k.Message = "warn", fmt.Sprintf("品牌版 Google Chrome %d 不再支持命令行加载扩展：扩展中心分配的扩展不会生效（指纹不受影响）。需要扩展请改用 Chromium 或 Chrome for Testing", k.Major)
	}
	if k.Level == "ok" && strings.Contains(lower, "msedge") {
		k.Level, k.Message = "warn", "Edge 内核的 UA-CH 品牌是 Microsoft Edge，与 Chrome 指纹模板不一致，推荐 Chromium 或 Chrome for Testing"
	}
	return k
}

// isBrandedChrome: Google Chrome stable/beta/dev/canary, not Chrome for
// Testing (which keeps --load-extension) and not Chromium builds.
func isBrandedChrome(path string) bool {
	p := strings.ToLower(strings.ReplaceAll(path, `\`, "/"))
	if strings.Contains(p, "for testing") || strings.Contains(p, "chrome-win") || strings.Contains(p, "chrome-linux") || strings.Contains(p, "chrome-mac") {
		return false
	}
	return strings.Contains(p, "/google/chrome") || strings.Contains(p, "google chrome.app") || strings.HasSuffix(p, "/google-chrome") || strings.Contains(p, "/opt/google/chrome")
}

// kernelUnsupportedErr is the launch-time guard (nil when the version is
// unknown: the CDP session then aligns the UA with the real version).
func kernelUnsupportedErr(path string) error {
	full := kernelVersion(path)
	if full == "" || majorOf(full) >= chromiumMinMajor {
		return nil
	}
	return fmt.Errorf("开源内核版本过旧（Chrome %s），不受支持：请在「设置 → 开源内核」换成 Chrome %d 及以上（推荐 %d 以上）的 Chromium / Chrome for Testing", full, chromiumMinMajor, chromiumRecommendedMajor)
}

// alignUAMajor rewrites the UA's Chrome token to the kernel's major so UA,
// UA-CH brands and the engine's real features agree. TVs keep their own
// fixed browser build.
func alignUAMajor(fp *Fingerprint, major int) {
	if major <= 0 || fp.DeviceType == "tv" {
		return
	}
	m := strconv.Itoa(major)
	fp.UserAgent = chromeMajorTok.ReplaceAllString(fp.UserAgent, "Chrome/"+m+".0.0.0")
	fp.BrowserVersion = m
}

// ensureScript is the legacy (< Chrome 120) fallback: a tab created with a URL
// while auto-attach holds it paused commits its first document without the
// document-start script, so check and reload once.
func (e *emulator) ensureScript(sid string) {
	for i := 0; i < 40; i++ {
		time.Sleep(150 * time.Millisecond)
		r, err := e.conn.call(sid, "Runtime.evaluate", map[string]any{"expression": `location.href + "|" + (window.__gb === 1)`, "returnByValue": true})
		if err != nil {
			return // tab closed / detached
		}
		v := string(r)
		if strings.Contains(v, `|true"`) {
			return
		}
		if strings.Contains(v, `"about:blank|`) || !strings.Contains(v, `|false"`) {
			continue // first navigation not committed yet
		}
		_, _ = e.conn.call(sid, "Page.reload", map[string]any{"ignoreCache": false})
		return
	}
}
