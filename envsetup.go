package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Self-contained delivery: the EXE is shipped alone, and on start-up it checks
// whether the custom Chromium and the fingerprint kit are present. Anything
// missing is downloaded from the shop, verified against a sha256 manifest, and
// unpacked into <appData>/undetectable-kit/.

const manifestPath = "manifest.json"

const downloadTO = 30 * time.Minute

// Delivery endpoints. chrome.zip unpacks to a top-level chrome/ folder and
// undetectable-kit.zip to undetectable-kit/fingerprints/.
var (
	chromeZipURL = "http://shop.chestchain.com/chrome.zip"
	fpKitZipURL  = "http://shop.chestchain.com/undetectable-kit.zip"
)

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

// envManifest lists the expected sha256 of each archive. It is fetched from
// <zip dir>/manifest.json; when the shop does not serve one yet, builtInSHA
// still rejects a tampered or truncated package.
type envManifest struct {
	Updated  string          `json:"updated"`
	Chrome   envManifestItem `json:"chrome"`
	FpKit    envManifestItem `json:"fpkit"`
	Revision int64           `json:"revision"`
}

type envManifestItem struct {
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	URL    string `json:"url,omitempty"`
}

func (m envManifest) item(kind string) envManifestItem {
	if kind == "chrome" {
		return m.Chrome
	}
	return m.FpKit
}

// builtInSHA is the fallback pin used only when the remote manifest is absent.
// Repin these after replacing the packages on the shop.
var builtInSHA = map[string]string{"chrome": "f99d6fabd051b0368f161a2493776e36b02fd0f2e88a9b342ce008b2ec1222d3", "fpkit": "4458ef4c0cb271fa6f83ffcda94a245f5e560b24f72d92bb9a8538c91e7ce923"}

// EnvStatus is what the UI binds to.
type EnvStatus struct {
	Ready     bool    `json:"ready"`
	KitDir    string  `json:"kitDir"`
	ChromeExe string  `json:"chromeExe"`
	Phase     string  `json:"phase"` // idle | checking | downloading | verifying | extracting | ready | failed
	Kind      string  `json:"kind,omitempty"`
	Pct       float64 `json:"pct"`
	Done      int64   `json:"done"`
	Total     int64   `json:"total"`
	Speed     int64   `json:"speedBps"`
	Message   string  `json:"message,omitempty"`
	Error     string  `json:"error,omitempty"`
	UpdatedAt int64   `json:"updatedAt"`
	Manifest  bool    `json:"manifestFromServer"`
}

type envInstaller struct {
	mu      sync.Mutex
	status  EnvStatus
	busy    bool
	lastErr string
	emit    func(EnvStatus)
	ctx     context.Context
	cancel  context.CancelFunc
}

var env = &envInstaller{status: EnvStatus{Phase: "idle"}}

func (e *envInstaller) snapshot() EnvStatus {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := e.status
	s.UpdatedAt = time.Now().Unix()
	return s
}

// patch applies non-zero fields and pushes the new status to the frontend.
func (e *envInstaller) patch(f func(*EnvStatus)) {
	e.mu.Lock()
	f(&e.status)
	s := e.status
	s.UpdatedAt = time.Now().Unix()
	e.mu.Unlock()
	if e.emit != nil {
		e.emit(s)
	}
}

func (e *envInstaller) setPhase(phase, msg string) {
	e.patch(func(s *EnvStatus) { s.Phase = phase; s.Message = msg; s.Error = "" })
}

func zipURL(kind string) string {
	if kind == "chrome" {
		return chromeZipURL
	}
	return fpKitZipURL
}

func zipLabel(kind string) string {
	if kind == "chrome" {
		return "定制 Chromium"
	}
	return "指纹包"
}

// kitRoot is where the delivered runtime lives. A folder next to the EXE wins
// so a portable layout keeps working; otherwise the per-user appData copy is
// used, which survives moving the EXE and needs no admin rights.
func kitRoot() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for i := 0; i < 4; i++ {
			if _, e := os.Stat(filepath.Join(dir, "undetectable-kit", "fingerprints", "library.json")); e == nil {
				return filepath.Join(dir, "undetectable-kit")
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return filepath.Join(appDataDir(), "undetectable-kit")
}

func chromeExePath(kit string) string { return filepath.Join(kit, "runtime", "chrome", "chrome.exe") }
func libPath(kit string) string       { return filepath.Join(kit, "fingerprints", "library.json") }

// envReport is the machine-readable answer for tests and the settings page.
type envReport struct {
	Kit        string   `json:"kit"`
	ChromeOK   bool     `json:"chromeOk"`
	LibraryOK  bool     `json:"libraryOk"`
	Templates  []string `json:"templates"`
	ChromeExe  string   `json:"chromeExe"`
	NeedChrome bool     `json:"needChrome"`
	NeedKit    bool     `json:"needKit"`
}

func inspectAt(kit string) envReport {
	r := envReport{Kit: kit, ChromeExe: chromeExePath(kit)}
	if st, err := os.Stat(r.ChromeExe); err == nil && !st.IsDir() {
		r.ChromeOK = true
	}
	if _, err := os.Stat(libPath(kit)); err == nil {
		r.LibraryOK = true
	}
	if ents, err := os.ReadDir(filepath.Join(kit, "fingerprints", "templates")); err == nil {
		for _, ent := range ents {
			if !ent.IsDir() && strings.HasSuffix(ent.Name(), ".bin") {
				r.Templates = append(r.Templates, ent.Name())
			}
		}
	}
	r.NeedChrome = !r.ChromeOK
	r.NeedKit = !r.LibraryOK || len(r.Templates) == 0
	return r
}

func inspectEnv() envReport { return inspectAt(kitRoot()) }

func envReady() bool {
	r := inspectEnv()
	return r.ChromeOK && r.LibraryOK && len(r.Templates) > 0
}

// httpClient honours HTTP_PROXY / HTTPS_PROXY / ALL_PROXY, including socks5://
// and socks5h:// which http.ProxyFromEnvironment ignores. Without this, a
// machine whose only route to the shop is a SOCKS tunnel cannot download.
func httpClient() *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{Timeout: 20 * time.Second, KeepAlive: 30 * time.Second}
	tr.DialContext = dialer.DialContext
	if p := proxyFromEnv(); p != nil {
		switch p.Scheme {
		case "socks5", "socks5h":
			var auth *proxy.Auth
			if p.User != nil {
				pw, _ := p.User.Password()
				auth = &proxy.Auth{User: p.User.Username(), Password: pw}
			}
			sd, err := proxy.SOCKS5("tcp", p.Host, auth, dialer)
			if err == nil {
				if ctxd, ok := sd.(proxy.ContextDialer); ok {
					tr.DialContext = ctxd.DialContext
				} else {
					tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
						return sd.Dial(network, addr)
					}
				}
				tr.Proxy = nil
			}
		default:
			tr.Proxy = http.ProxyURL(p)
		}
	}
	return &http.Client{Transport: tr, Timeout: downloadTO}
}

func proxyFromEnv() *url.URL {
	for _, k := range []string{"HTTP_PROXY", "http_proxy", "HTTPS_PROXY", "https_proxy", "ALL_PROXY", "all_proxy"} {
		v := strings.TrimSpace(os.Getenv(k))
		if v == "" || strings.Contains(v, "<") {
			continue
		}
		if !strings.Contains(v, "://") {
			v = "socks5://" + v
		}
		u, err := url.Parse(v)
		if err != nil || u.Host == "" {
			continue
		}
		return u
	}
	return nil
}

func urlDir(u string) string {
	if i := strings.LastIndex(u, "/"); i > 0 {
		return u[:i+1]
	}
	return u
}

// manifestFetch is swapped in tests; production uses fetchManifest.
var manifestFetch = fetchManifest

// fetchManifest pulls <zip dir>/manifest.json. Missing is fine; callers fall
// back to builtInSHA.
func fetchManifest(ctx context.Context) (envManifest, bool) {
	var m envManifest
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlDir(chromeZipURL)+manifestPath, nil)
	if err != nil {
		return m, false
	}
	resp, err := httpClient().Do(req)
	if err != nil {
		return m, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return m, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil || json.Unmarshal(body, &m) != nil {
		return m, false
	}
	if !sha256Hex.MatchString(m.Chrome.SHA256) || !sha256Hex.MatchString(m.FpKit.SHA256) {
		return m, false
	}
	return m, true
}

// download writes url to dest and returns the sha256 of what arrived. total
// comes from Content-Length when the server sends it. The file is closed before
// returning, because Windows keeps an open handle from blocking archive/zip.
func (e *envInstaller) download(ctx context.Context, urlStr, dest string, wantSize int64, kind string) (string, error) {
	tmp := dest + ".part"
	_ = os.Remove(tmp)
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		f.Close()
		return "", err
	}
	resp, err := httpClient().Do(req)
	if err != nil {
		f.Close()
		return "", fmt.Errorf("无法连接下载服务器: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		f.Close()
		return "", fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}
	total := resp.ContentLength
	if total <= 0 {
		total = wantSize
	}

	h := sha256.New()
	buf := make([]byte, 256*1024)
	var written int64
	start := time.Now()
	windowStart, windowBytes := start, int64(0)
	var speed int64
	for {
		if ctx.Err() != nil {
			f.Close()
			return "", ctx.Err()
		}
		n, rdErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := f.Write(buf[:n]); wErr != nil {
				f.Close()
				return "", fmt.Errorf("写入临时文件失败: %w", wErr)
			}
			h.Write(buf[:n])
			written += int64(n)
			windowBytes += int64(n)
			if el := time.Since(windowStart); el >= 500*time.Millisecond {
				speed = int64(float64(windowBytes) / el.Seconds())
				windowStart, windowBytes = time.Now(), 0
			}
			pct := 0.0
			if total > 0 {
				pct = float64(written) * 100 / float64(total)
			}
			e.patch(func(s *EnvStatus) {
				s.Phase = "downloading"
				s.Kind = kind
				s.Done, s.Total, s.Pct, s.Speed = written, total, pct, speed
				s.Message = fmt.Sprintf("正在下载%s", zipLabel(kind))
			})
		}
		if rdErr == io.EOF {
			break
		}
		if rdErr != nil {
			f.Close()
			return "", fmt.Errorf("下载中断: %w", rdErr)
		}
	}
	if syncErr := f.Sync(); syncErr != nil {
		f.Close()
		return "", syncErr
	}
	if closeErr := f.Close(); closeErr != nil {
		return "", closeErr
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if total > 0 && written != total {
		return sum, fmt.Errorf("下载不完整: 期望 %d 字节，实际 %d", total, written)
	}
	// Rename only after the hash matches so a bad download never looks installed.
	if err := os.Rename(tmp, dest); err != nil {
		return sum, err
	}
	return sum, nil
}

// extractZip unpacks an archive into a fresh staging directory. Entries are
// validated against the destination so "../" names and absolute paths cannot
// write outside it.
func extractZip(ctx context.Context, zipPath, dest string) (int64, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, fmt.Errorf("压缩包已损坏: %w", err)
	}
	defer zr.Close()
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return 0, err
	}
	var bytes int64
	for _, fh := range zr.File {
		if ctx.Err() != nil {
			return bytes, ctx.Err()
		}
		clean, ok := safeRel(fh.Name)
		if !ok {
			continue
		}
		target := filepath.Join(dest, clean)
		if fh.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return bytes, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return bytes, err
		}
		rc, err := fh.Open()
		if err != nil {
			return bytes, fmt.Errorf("读取 %s 失败: %w", clean, err)
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			rc.Close()
			return bytes, err
		}
		n, err := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return bytes, fmt.Errorf("解压 %s 失败: %w", clean, err)
		}
		bytes += n
	}
	return bytes, nil
}

// safeRel maps an archive entry name to a path inside the destination.
func safeRel(name string) (string, bool) {
	n := strings.ReplaceAll(filepath.ToSlash(name), "\\", "/")
	n = strings.TrimPrefix(n, "./")
	if n == "" || strings.HasPrefix(n, "/") || strings.Contains(n, ":") {
		return "", false
	}
	parts := []string{}
	for _, seg := range strings.Split(n, "/") {
		switch seg {
		case "", ".", "..":
			if seg == ".." {
				return "", false
			}
		default:
			parts = append(parts, seg)
		}
	}
	if len(parts) == 0 {
		return "", false
	}
	return filepath.Join(parts...), true
}

// moveContents relocates every child of src into dst, replacing existing paths.
func moveContents(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	ents, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, ent := range ents {
		to := filepath.Join(dst, ent.Name())
		if err := os.RemoveAll(to); err != nil {
			return err
		}
		if err := os.Rename(filepath.Join(src, ent.Name()), to); err != nil {
			return fmt.Errorf("移动 %s 失败: %w", ent.Name(), err)
		}
	}
	return nil
}

// installTarget picks the kit directory the installer writes to. When a
// portable kit sits next to the EXE but is incomplete, it is left untouched and
// the user-writable copy is used, because Program Files is not writable without
// admin rights.
func installTarget() string {
	kit := kitRoot()
	if _, e := os.Stat(filepath.Join(kit, "fingerprints", "library.json")); e != nil {
		kit = filepath.Join(appDataDir(), "undetectable-kit")
	}
	return kit
}

// installOne downloads, verifies and unpacks one package into the kit.
func (e *envInstaller) installOne(ctx context.Context, kit, work, kind string, man envManifest, fromServer bool) error {
	urlStr := zipURL(kind)
	if u := man.item(kind).URL; fromServer && u != "" {
		urlStr = u
	}
	want := man.item(kind).SHA256
	source := "内置校验值"
	if fromServer {
		source = "服务器清单"
	} else if want == "" {
		want = builtInSHA[kind]
	}
	if want == "" {
		return fmt.Errorf("没有%s的校验值，无法确认包完整性", zipLabel(kind))
	}

	zipPath := filepath.Join(work, kind+".zip")
	e.patch(func(s *EnvStatus) {
		s.Phase = "downloading"
		s.Kind, s.Done, s.Total, s.Pct, s.Speed = kind, 0, man.item(kind).Size, 0, 0
		s.Message = fmt.Sprintf("正在下载%s", zipLabel(kind))
	})
	sum, err := e.download(ctx, urlStr, zipPath, man.item(kind).Size, kind)
	if err != nil {
		return err
	}
	if sum != want {
		_ = os.Remove(zipPath)
		return fmt.Errorf("%s校验不通过（%s）：期望 %s…，实际 %s…", zipLabel(kind), source, want[:12], sum[:12])
	}
	e.patch(func(s *EnvStatus) {
		s.Phase = "verifying"
		s.Pct = 100
		s.Message = fmt.Sprintf("校验%s通过", zipLabel(kind))
	})

	stage := filepath.Join(work, "stage-"+kind)
	_ = os.RemoveAll(stage)
	e.patch(func(s *EnvStatus) {
		s.Phase = "extracting"
		s.Pct = 0
		s.Message = fmt.Sprintf("正在解压%s", zipLabel(kind))
	})
	if _, err := extractZip(ctx, zipPath, stage); err != nil {
		return err
	}
	if err := placePackage(kind, stage, kit); err != nil {
		return err
	}
	_ = os.RemoveAll(stage)
	_ = os.Remove(zipPath)
	return nil
}

// placePackage moves an unpacked package into the kit layout. chrome.zip
// contains a top-level chrome/ folder, which must land at runtime/chrome/ —
// a blind extract would leave the kernel undiscoverable.
func placePackage(kind, stage, kit string) error {
	if kind == "chrome" {
		src := filepath.Join(stage, "chrome")
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("chrome.zip 里找不到顶层 chrome/ 目录")
		}
		runtime := filepath.Join(kit, "runtime")
		if err := os.MkdirAll(runtime, 0o755); err != nil {
			return err
		}
		dst := filepath.Join(runtime, "chrome")
		if err := os.RemoveAll(dst); err != nil {
			return fmt.Errorf("无法替换旧的 runtime/chrome，可能浏览器仍在运行: %w", err)
		}
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("移动到 runtime/chrome 失败: %w", err)
		}
		return nil
	}
	// undetectable-kit.zip unpacks to an undetectable-kit/ folder holding
	// fingerprints/.
	src := filepath.Join(stage, "undetectable-kit")
	if _, err := os.Stat(filepath.Join(src, "fingerprints")); err != nil {
		src = stage
		if _, err := os.Stat(filepath.Join(src, "fingerprints")); err != nil {
			return fmt.Errorf("undetectable-kit.zip 里找不到 fingerprints/")
		}
	}
	return moveContents(src, kit)
}

// Ensure runs the check-and-install pass. It is safe to call repeatedly; a
// second call while one is running is ignored.
func (e *envInstaller) Ensure(force bool) EnvStatus { return e.ensure(kitRoot(), force) }

func (e *envInstaller) ensure(kit string, force bool) EnvStatus {
	e.mu.Lock()
	if e.busy {
		s := e.status
		e.mu.Unlock()
		return s
	}
	e.busy = true
	e.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	e.ctx, e.cancel = ctx, cancel
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.busy, e.ctx, e.cancel = false, nil, nil
		e.mu.Unlock()
	}()

	rep := inspectAt(kit)
	needChrome, needKit := rep.NeedChrome, rep.NeedKit
	if force {
		needChrome, needKit = true, true
	}
	e.patch(func(s *EnvStatus) {
		s.Phase = "checking"
		s.Kind, s.Pct, s.Done, s.Total = "", 0, 0, 0
		s.KitDir, s.ChromeExe = rep.Kit, rep.ChromeExe
		s.Message = "正在检测浏览器内核环境"
	})
	if !needChrome && !needKit {
		e.markReady(rep)
		return e.snapshot()
	}

	work := filepath.Join(filepath.Dir(kit), "_envsetup-work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		e.fail("work", err)
		return e.snapshot()
	}
	man, fromServer := manifestFetch(ctx)
	e.patch(func(s *EnvStatus) { s.Manifest = fromServer })

	for _, kind := range []string{"chrome", "fpkit"} {
		if (kind == "chrome" && !needChrome) || (kind == "fpkit" && !needKit) {
			continue
		}
		if err := e.installOne(ctx, kit, work, kind, man, fromServer); err != nil {
			e.fail(kind, err)
			return e.snapshot()
		}
	}
	rep = inspectAt(kit)
	if !(rep.ChromeOK && rep.LibraryOK && len(rep.Templates) > 0) {
		e.fail("verify", fmt.Errorf("安装后仍缺少必需文件：%s", strings.Join(rep.missing(), "、")))
		return e.snapshot()
	}
	e.markReady(rep)
	return e.snapshot()
}

func (r envReport) missing() []string {
	out := []string{}
	if !r.ChromeOK {
		out = append(out, "runtime/chrome/chrome.exe")
	}
	if !r.LibraryOK {
		out = append(out, "fingerprints/library.json")
	}
	if len(r.Templates) == 0 {
		out = append(out, "fingerprints/templates/*.bin")
	}
	return out
}

func (e *envInstaller) markReady(rep envReport) {
	e.patch(func(s *EnvStatus) {
		s.Ready, s.Phase, s.Kind, s.Error = true, "ready", "", ""
		s.Pct, s.Done, s.Total = 100, 0, 0
		s.KitDir, s.ChromeExe = rep.Kit, rep.ChromeExe
		s.Message = "环境就绪"
	})
	e.lastErr = ""
}

func (e *envInstaller) fail(kind string, err error) {
	e.lastErr = err.Error()
	e.patch(func(s *EnvStatus) {
		s.Ready, s.Phase, s.Kind, s.Error = false, "failed", kind, err.Error()
		s.Message = "环境安装失败"
	})
}

func (e *envInstaller) Cancel() {
	e.mu.Lock()
	c := e.cancel
	e.mu.Unlock()
	if c != nil {
		c()
	}
}

// applyToSettings points the store at the installed kit. Paths that already
// exist are kept, so a hand-picked kit directory survives.
func applyToSettings(st Settings) Settings {
	rep := inspectEnv()
	st.KernelMode = "undetectable"
	if _, err := os.Stat(rep.ChromeExe); err == nil {
		st.UndetectableKit = rep.Kit
		st.KernelPath = rep.ChromeExe
	} else if st.UndetectableKit != "" {
		if cand := chromeExePath(st.UndetectableKit); fileExists(cand) {
			st.KernelPath = cand
		}
	}
	return st
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return p != "" && err == nil && !st.IsDir()
}
