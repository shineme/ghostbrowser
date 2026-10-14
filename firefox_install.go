package main

// One-click install of the official Mozilla Firefox (ESR or release, Windows
// x64) into %APPDATA%/GhostBrowser/firefox/<channel>. Nothing is bundled: the
// installer comes from Mozilla's official download service over HTTPS, every
// redirect must stay on a Mozilla host, and the file is checked against the
// SHA512SUMS Mozilla publishes for that release before it runs. The full
// installer is then run silently into the folder (no shortcuts, no
// maintenance service, no default-browser agent). Progress is pushed to the UI
// as "firefox:install" events with the same shape as the Chromium install.

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var (
	firefoxDownloadBase = "https://download.mozilla.org/"
	firefoxSumsBase     = "https://archive.mozilla.org/pub/firefox/releases/"
	// firefoxAllowedHosts are Mozilla's download hosts; tests add theirs.
	firefoxAllowedHosts = []string{"download.mozilla.org", "download-installer.cdn.mozilla.net", "archive.mozilla.org", "ftp.mozilla.org"}
	// firefoxRunInstaller runs the silent installer (swapped by tests).
	firefoxRunInstaller = runFirefoxInstaller
)

func firefoxInstallRoot() string { return filepath.Join(appDataDir(), "firefox") }

var ffInst = &kernelInstaller{st: KernelProgress{Phase: "idle", ETA: -1}, label: "Firefox 官方安装包"}

// firefoxProduct maps a channel to Mozilla's bouncer product name.
func firefoxProduct(channel string) (string, string) {
	if channel == "release" {
		return "firefox-latest-ssl", "release"
	}
	return "firefox-esr-latest-ssl", "esr"
}

// firefoxBouncerURL is the official download link for one channel.
func firefoxBouncerURL(channel string) string {
	prod, _ := firefoxProduct(channel)
	return firefoxDownloadBase + "?product=" + prod + "&os=win64&lang=zh-CN"
}

func firefoxHostAllowed(u *url.URL) bool {
	if u == nil || u.Scheme != "https" {
		return false
	}
	h := strings.ToLower(u.Hostname())
	for _, a := range firefoxAllowedHosts {
		if h == a {
			return true
		}
	}
	return false
}

// firefoxHTTPClient refuses redirects that leave HTTPS or Mozilla's hosts.
func firefoxHTTPClient() *http.Client {
	c := newKernelClient()
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 8 {
			return errors.New("重定向次数过多")
		}
		if !firefoxHostAllowed(req.URL) {
			return fmt.Errorf("下载地址被重定向到非 Mozilla 官方地址：%s", req.URL.Host)
		}
		return nil
	}
	return c
}

var ffRelPath = regexp.MustCompile(`/pub/firefox/releases/([^/]+)/(win64/[^/]+/[^/]+\.exe)$`)

// resolveFirefox follows the bouncer to the real installer URL and returns it
// with its version and the path used in SHA512SUMS.
func resolveFirefox(ctx context.Context, cl *http.Client, channel string) (dl, ver, sumPath string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, firefoxBouncerURL(channel), nil)
	if err != nil {
		return "", "", "", err
	}
	resp, err := cl.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", "", "", ctx.Err()
		}
		return "", "", "", &netErr{err}
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", "", "", &httpErr{code: resp.StatusCode, what: "Firefox 安装包"}
	}
	final := resp.Request.URL
	if !firefoxHostAllowed(final) {
		return "", "", "", fmt.Errorf("下载地址不是 Mozilla 官方 HTTPS 地址：%s", final)
	}
	p, _ := url.PathUnescape(final.EscapedPath())
	m := ffRelPath.FindStringSubmatch(p)
	if m == nil {
		return "", "", "", fmt.Errorf("无法识别的 Firefox 下载地址：%s", final)
	}
	return final.String(), m[1], m[2], nil
}

// firefoxExpectedSum reads Mozilla's SHA512SUMS for ver and returns the hash of sumPath.
func firefoxExpectedSum(ctx context.Context, cl *http.Client, ver, sumPath string) (string, error) {
	b, code, err := kernelGet(ctx, cl, firefoxSumsBase+url.PathEscape(ver)+"/SHA512SUMS", 16<<20)
	if err != nil {
		return "", err
	}
	if code != 200 {
		return "", &httpErr{code: code, what: "SHA512SUMS 校验文件"}
	}
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		f := strings.SplitN(strings.TrimSpace(sc.Text()), "  ", 2)
		if len(f) == 2 && f[1] == sumPath && len(f[0]) == 128 {
			return strings.ToLower(f[0]), nil
		}
	}
	return "", fmt.Errorf("SHA512SUMS 中没有 %s", sumPath)
}

func fileSHA512(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha512.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// runFirefoxInstaller runs Mozilla's full installer silently into dir.
func runFirefoxInstaller(ctx context.Context, setup, dir string) error {
	if runtime.GOOS != "windows" {
		return errors.New("一键安装 Firefox 仅支持 Windows；其他系统请用系统的软件包管理器安装")
	}
	cmd := exec.CommandContext(ctx, setup, "/S", "/InstallDirectoryPath="+dir,
		"/DesktopShortcut=false", "/StartMenuShortcut=false", "/TaskbarShortcut=false",
		"/PrivateBrowsingShortcut=false", "/MaintenanceService=false", "/RegisterDefaultAgent=false")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("安装程序失败：%v %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// FirefoxInstallStart installs channel ("esr" | "release"); onDone gets firefox.exe.
func firefoxInstallStart(channel string, onDone func(path string)) (KernelProgress, error) {
	k := ffInst
	_, ch := firefoxProduct(channel)
	k.mu.Lock()
	if k.st.Running {
		k.mu.Unlock()
		return k.snapshot(), errors.New("Firefox 正在安装")
	}
	ctx, cancel := context.WithCancel(context.Background())
	k.cancel = cancel
	k.st = KernelProgress{Phase: "resolve", Running: true, Source: ch, ETA: -1, Message: "正在获取官方下载地址…"}
	k.mu.Unlock()
	k.patch(true, func(*KernelProgress) {})
	go func() {
		defer cancel()
		exe, err := firefoxInstallRun(ctx, k, ch)
		switch {
		case err == nil:
			if onDone != nil {
				onDone(exe)
			}
			k.patch(true, func(s *KernelProgress) {
				s.Phase, s.Running, s.Path, s.Speed, s.ETA = "done", false, exe, 0, 0
				s.Pct, s.Message, s.Hint, s.Error = 100, "安装完成，已设为 Firefox 内核", "", ""
			})
		case errors.Is(err, context.Canceled):
			k.patch(true, func(s *KernelProgress) {
				s.Phase, s.Running, s.Speed, s.ETA = "cancelled", false, 0, -1
				s.Message, s.Error, s.Hint = "已取消安装", "", "已下载的部分会保留，重新安装时从断点继续"
			})
		default:
			msg, hint := kernelErrHint(err, "official")
			if strings.Contains(err.Error(), "SHA512") || strings.Contains(err.Error(), "官方") {
				msg, hint = err.Error(), "安装包未通过校验，已删除。请检查网络是否被劫持，或从官网手动下载"
			}
			k.patch(true, func(s *KernelProgress) {
				s.Phase, s.Running, s.Speed, s.ETA = "error", false, 0, -1
				s.Message, s.Hint, s.Error = msg, hint, err.Error()
			})
		}
	}()
	return k.snapshot(), nil
}

func firefoxInstallRun(ctx context.Context, k *kernelInstaller, channel string) (string, error) {
	root := firefoxInstallRoot()
	tmp := filepath.Join(root, ".tmp")
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return "", fmt.Errorf("无法创建目录: %w", err)
	}
	cl := firefoxHTTPClient()
	dl, ver, sumPath, err := resolveFirefox(ctx, cl, channel)
	if err != nil {
		return "", err
	}
	k.patch(true, func(s *KernelProgress) { s.Version, s.URL, s.Revision = ver, dl, ver })
	want, err := firefoxExpectedSum(ctx, cl, ver, sumPath)
	if err != nil {
		return "", err
	}
	if free, ok := diskFree(root); ok && free < 600<<20 {
		return "", &diskErr{need: 600 << 20, free: free}
	}
	setup := filepath.Join(tmp, safeName(path.Base(sumPath)))
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(kernelRetryDelay):
			}
		}
		lastErr = k.download(ctx, cl, dl, setup, 0)
		if lastErr == nil || errors.Is(lastErr, context.Canceled) || !retryable(lastErr) {
			break
		}
	}
	if lastErr != nil {
		return "", lastErr
	}
	k.phase("verify", "正在校验 SHA-512…")
	got, err := fileSHA512(setup)
	if err != nil {
		return "", err
	}
	if got != want {
		_ = os.Remove(setup)
		return "", fmt.Errorf("SHA512 校验失败：安装包与 Mozilla 公布的不一致")
	}
	k.phase("extract", "正在静默安装 Firefox…")
	dir := filepath.Join(root, channel)
	if err := firefoxRunInstaller(ctx, setup, dir); err != nil {
		return "", err
	}
	exe := filepath.Join(dir, "firefox.exe")
	if fi, err := os.Stat(exe); err != nil || fi.Size() == 0 {
		return "", errors.New("安装后没有找到 firefox.exe")
	}
	_ = os.Remove(setup)
	ffVerCache.Delete(exe)
	return exe, nil
}
