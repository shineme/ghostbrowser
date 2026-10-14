package main

// One-click install of an open-source Chromium kernel (Windows x64) from the
// official Chromium snapshot builds, a domestic mirror of the same builds
// (npmmirror), or a custom URL. Downloads are streamed and resumable,
// extraction rejects zip-slip entries, and progress is pushed to the UI as
// "kernel:install" events.

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Snapshot roots; both serve <rev>/chrome-win.zip. Vars so tests can swap them.
var (
	kernelOfficialBase = "https://commondatastorage.googleapis.com/chromium-browser-snapshots/Win_x64/"
	kernelMirrorBase   = "https://registry.npmmirror.com/-/binary/chromium-browser-snapshots/Win_x64/"
	kernelRetryDelay   = 2 * time.Second
)

const kernelZipName = "chrome-win.zip"

// KernelProgress is the install state shown in Settings and pushed as events.
type KernelProgress struct {
	Phase    string  `json:"phase"` // idle|resolve|download|extract|verify|done|error|cancelled
	Running  bool    `json:"running"`
	Source   string  `json:"source"` // mirror|official|custom
	Revision string  `json:"revision,omitempty"`
	Version  string  `json:"version,omitempty"`
	Path     string  `json:"path,omitempty"` // chrome.exe after success
	URL      string  `json:"url,omitempty"`
	Done     int64   `json:"done"`
	Total    int64   `json:"total"`
	Pct      float64 `json:"pct"`
	Speed    int64   `json:"speed"` // bytes/s
	ETA      int64   `json:"eta"`   // seconds, -1 unknown
	Resumed  bool    `json:"resumed,omitempty"`
	Message  string  `json:"message"`
	Hint     string  `json:"hint,omitempty"`
	Error    string  `json:"error,omitempty"`
}

// InstalledKernel is one kernel under %APPDATA%/GhostBrowser/kernels.
type InstalledKernel struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Path      string `json:"path"` // chrome.exe
	Dir       string `json:"dir"`
	Size      int64  `json:"size"`
	Installed int64  `json:"installed"` // unix ms
	Active    bool   `json:"active"`
}

type kernelInstaller struct {
	mu     sync.Mutex
	st     KernelProgress
	cancel context.CancelFunc
	emit   func(KernelProgress)
	last   time.Time
}

var kernelInst = &kernelInstaller{st: KernelProgress{Phase: "idle", ETA: -1}}

func kernelsDir() string { return filepath.Join(appDataDir(), "kernels") }

func (k *kernelInstaller) snapshot() KernelProgress { k.mu.Lock(); defer k.mu.Unlock(); return k.st }

// patch updates the state; force=false throttles events to ~7/s.
func (k *kernelInstaller) patch(force bool, f func(*KernelProgress)) {
	k.mu.Lock()
	f(&k.st)
	s := k.st
	emit := k.emit
	send := force || time.Since(k.last) >= 150*time.Millisecond
	if send {
		k.last = time.Now()
	}
	k.mu.Unlock()
	if send && emit != nil {
		emit(s)
	}
}

func (k *kernelInstaller) phase(p, msg string) {
	k.patch(true, func(s *KernelProgress) {
		s.Phase, s.Message, s.Hint, s.Error = p, msg, "", ""
		s.Done, s.Total, s.Pct, s.Speed, s.ETA = 0, 0, 0, 0, -1
	})
}

// Start launches an install in the background. onDone receives chrome.exe.
func (k *kernelInstaller) Start(root, source, custom string, onDone func(path string)) (KernelProgress, error) {
	k.mu.Lock()
	if k.st.Running {
		k.mu.Unlock()
		return k.snapshot(), errors.New("已有内核正在安装")
	}
	ctx, cancel := context.WithCancel(context.Background())
	k.cancel = cancel
	k.st = KernelProgress{Phase: "resolve", Running: true, Source: source, ETA: -1, Message: "正在解析最新版本…"}
	k.mu.Unlock()
	k.patch(true, func(*KernelProgress) {})
	go func() {
		defer cancel()
		path, err := k.run(ctx, root, source, custom)
		switch {
		case err == nil:
			if onDone != nil {
				onDone(path)
			}
			k.patch(true, func(s *KernelProgress) {
				s.Phase, s.Running, s.Path, s.Speed, s.ETA = "done", false, path, 0, 0
				s.Pct, s.Message, s.Hint, s.Error = 100, "安装完成，已设为默认开源内核", "", ""
			})
		case errors.Is(err, context.Canceled):
			k.patch(true, func(s *KernelProgress) {
				s.Phase, s.Running, s.Speed, s.ETA = "cancelled", false, 0, -1
				s.Message, s.Error = "已取消安装", ""
				s.Hint = "已下载的部分会保留，重新安装时从断点继续"
			})
		default:
			msg, hint := kernelErrHint(err, source)
			k.patch(true, func(s *KernelProgress) {
				s.Phase, s.Running, s.Speed, s.ETA = "error", false, 0, -1
				s.Message, s.Hint, s.Error = msg, hint, err.Error()
			})
		}
	}()
	return k.snapshot(), nil
}

func (k *kernelInstaller) Cancel() {
	k.mu.Lock()
	c := k.cancel
	k.mu.Unlock()
	if c != nil {
		c()
	}
}

func (k *kernelInstaller) run(ctx context.Context, root, source, custom string) (string, error) {
	tmp := filepath.Join(root, ".tmp")
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return "", fmt.Errorf("无法创建内核目录: %w", err)
	}
	cl := newKernelClient()
	zipURL, rev, size, err := resolveKernel(ctx, cl, source, custom)
	if err != nil {
		return "", err
	}
	k.patch(true, func(s *KernelProgress) { s.Revision, s.URL, s.Total = rev, zipURL, size })

	// Room for the archive plus the unpacked tree (~1.8x the zip).
	need := int64(1200 << 20)
	if size > 0 {
		need = size*4 + 100<<20 // snapshot zip ~360 MB unpacks to ~860 MB
	}
	if free, ok := diskFree(root); ok && free < need {
		return "", &diskErr{need: need, free: free}
	}

	zipPath := filepath.Join(tmp, "chromium-"+safeName(rev)+".zip")
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			k.patch(true, func(s *KernelProgress) {
				s.Message = fmt.Sprintf("网络中断，%d 秒后自动重试（第 %d 次）…", int(kernelRetryDelay.Seconds()), attempt)
			})
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(kernelRetryDelay):
			}
		}
		lastErr = k.download(ctx, cl, zipURL, zipPath, size)
		if lastErr == nil || errors.Is(lastErr, context.Canceled) || !retryable(lastErr) {
			break
		}
	}
	if lastErr != nil {
		return "", lastErr
	}

	k.phase("extract", "正在解压…")
	xdir := filepath.Join(tmp, "x-"+safeName(rev))
	_ = os.RemoveAll(xdir)
	defer os.RemoveAll(xdir)
	if err := k.extract(ctx, zipPath, xdir); err != nil {
		if errors.Is(err, errZipCorrupt) {
			_ = os.Remove(zipPath) // force a fresh download next time
		}
		return "", err
	}

	k.phase("verify", "正在校验内核…")
	exeDir, err := findChromeExeDir(xdir, 3)
	if err != nil {
		return "", err
	}
	ver := kernelVersionIn(exeDir)
	label := ver
	if label == "" {
		label = "r" + safeName(rev)
	}
	final := filepath.Join(root, "chromium-"+label)
	if err := os.RemoveAll(final); err != nil {
		return "", fmt.Errorf("旧版本目录被占用，请先关闭使用该内核的环境: %w", err)
	}
	if err := os.Rename(exeDir, final); err != nil {
		return "", fmt.Errorf("移动内核失败: %w", err)
	}
	exe := filepath.Join(final, "chrome.exe")
	if fi, err := os.Stat(exe); err != nil || fi.Size() == 0 {
		return "", errors.New("校验失败：chrome.exe 不存在或为空")
	}
	_ = os.Remove(zipPath)
	k.patch(true, func(s *KernelProgress) { s.Version = ver })
	return exe, nil
}

// ---- resolve ----

func resolveKernel(ctx context.Context, cl *http.Client, source, custom string) (zipURL, rev string, size int64, err error) {
	switch source {
	case "official":
		return resolveSnapshotBase(ctx, cl, kernelOfficialBase)
	case "custom":
		u := strings.TrimSpace(custom)
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			return "", "", 0, errors.New("自定义下载地址需以 http:// 或 https:// 开头")
		}
		if strings.HasSuffix(strings.ToLower(strings.SplitN(u, "?", 2)[0]), ".zip") {
			name := filepath.Base(strings.SplitN(u, "?", 2)[0])
			rev = strings.TrimSuffix(name, filepath.Ext(name))
			if m := regexp.MustCompile(`/(\d{5,})/`).FindStringSubmatch(u); m != nil {
				rev = m[1]
			}
			return u, rev, 0, nil
		}
		return resolveSnapshotBase(ctx, cl, u)
	default:
		return resolveSnapshotBase(ctx, cl, kernelMirrorBase)
	}
}

// resolveSnapshotBase understands both layouts: a LAST_CHANGE text file
// (Google storage) and a JSON directory listing (npmmirror binary mirror).
func resolveSnapshotBase(ctx context.Context, cl *http.Client, base string) (string, string, int64, error) {
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	if body, code, err := kernelGet(ctx, cl, base+"LAST_CHANGE", 64); err == nil && code == 200 {
		if rev := strings.TrimSpace(string(body)); regexp.MustCompile(`^\d+$`).MatchString(rev) {
			return base + rev + "/" + kernelZipName, rev, 0, nil
		}
	} else if errors.Is(err, context.Canceled) {
		return "", "", 0, err
	}
	body, code, err := kernelGet(ctx, cl, base, 8<<20)
	if err != nil {
		return "", "", 0, err
	}
	if code != 200 {
		return "", "", 0, &httpErr{code: code, what: "版本列表"}
	}
	type entry struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
		URL  string `json:"url"`
	}
	var list []entry
	if json.Unmarshal(body, &list) != nil {
		return "", "", 0, errors.New("无法解析版本列表：下载地址不是快照目录")
	}
	revs := []int{}
	for _, e := range list {
		if n, err := strconv.Atoi(strings.Trim(e.Name, "/")); err == nil {
			revs = append(revs, n)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(revs)))
	// The newest folders can still be uploading; take the first with a zip.
	for i, r := range revs {
		if i >= 8 {
			break
		}
		rs := strconv.Itoa(r)
		b, c, err := kernelGet(ctx, cl, base+rs+"/", 1<<20)
		if err != nil {
			return "", "", 0, err
		}
		if c != 200 {
			continue
		}
		var files []entry
		_ = json.Unmarshal(b, &files)
		for _, f := range files {
			if f.Name == kernelZipName {
				u := f.URL
				if u == "" {
					u = base + rs + "/" + kernelZipName
				}
				return u, rs, f.Size, nil
			}
		}
	}
	return "", "", 0, errors.New("没有找到可用的 Windows x64 快照版本")
}

func kernelGet(ctx context.Context, cl *http.Client, u string, limit int64) ([]byte, int, error) {
	c, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(c, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := cl.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, 0, ctx.Err()
		}
		return nil, 0, &netErr{err}
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return nil, resp.StatusCode, &netErr{err}
	}
	return b, resp.StatusCode, nil
}

// ---- download ----

func (k *kernelInstaller) download(ctx context.Context, cl *http.Client, u, dest string, want int64) error {
	if fi, err := os.Stat(dest); err == nil && fi.Size() > 0 && (want <= 0 || fi.Size() == want) {
		return nil // finished earlier (e.g. extraction failed afterwards)
	}
	part := dest + ".part"
	var have int64
	if fi, err := os.Stat(part); err == nil {
		have = fi.Size()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	if have > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", have))
	}
	resp, err := cl.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return &netErr{err}
	}
	defer resp.Body.Close()
	flags := os.O_CREATE | os.O_WRONLY
	total := want
	switch {
	case resp.StatusCode == http.StatusPartialContent && have > 0:
		flags |= os.O_APPEND
		if cr := resp.Header.Get("Content-Range"); cr != "" {
			if i := strings.LastIndex(cr, "/"); i >= 0 {
				if n, err := strconv.ParseInt(cr[i+1:], 10, 64); err == nil {
					total = n
				}
			}
		}
		if total <= 0 && resp.ContentLength > 0 {
			total = have + resp.ContentLength
		}
	case resp.StatusCode == http.StatusRequestedRangeNotSatisfiable && have > 0 && (want <= 0 || have == want):
		return os.Rename(part, dest)
	case resp.StatusCode == http.StatusOK:
		flags |= os.O_TRUNC
		have = 0
		if resp.ContentLength > 0 {
			total = resp.ContentLength
		}
	default:
		return &httpErr{code: resp.StatusCode, what: "内核压缩包"}
	}
	f, err := os.OpenFile(part, flags, 0o644)
	if err != nil {
		return fmt.Errorf("无法写入临时文件: %w", err)
	}
	resumed := have > 0
	k.patch(true, func(s *KernelProgress) {
		s.Phase, s.Done, s.Total, s.Resumed, s.Hint, s.Error = "download", have, total, resumed, "", ""
		s.Message = "正在下载 Chromium 内核…"
		if resumed {
			s.Message = "正在从断点继续下载…"
		}
	})
	buf := make([]byte, 256<<10)
	done := have
	winStart, winBytes := time.Now(), int64(0)
	var speed float64
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				return fmt.Errorf("写入失败（磁盘已满？）: %w", werr)
			}
			done += int64(n)
			winBytes += int64(n)
			if el := time.Since(winStart); el >= 400*time.Millisecond {
				cur := float64(winBytes) / el.Seconds()
				if speed == 0 {
					speed = cur
				} else {
					speed = speed*0.7 + cur*0.3
				}
				winStart, winBytes = time.Now(), 0
			}
			d, sp := done, speed
			k.patch(false, func(s *KernelProgress) {
				s.Done, s.Speed = d, int64(sp)
				if s.Total > 0 {
					s.Pct = float64(d) * 100 / float64(s.Total)
					if sp > 0 {
						s.ETA = int64(float64(s.Total-d) / sp)
					}
				}
			})
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return &netErr{rerr}
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	if total > 0 && done != total {
		return &netErr{fmt.Errorf("下载不完整: %d/%d 字节", done, total)}
	}
	k.patch(true, func(s *KernelProgress) { s.Done, s.Pct, s.ETA = done, 100, 0 })
	return os.Rename(part, dest)
}

// ---- extract / verify ----

var errZipCorrupt = errors.New("压缩包已损坏")

// extract unpacks zipPath into dest. Any entry that would land outside dest
// (absolute path, drive letter, "..") aborts the install: a snapshot never
// contains one, so it means the archive was tampered with.
func (k *kernelInstaller) extract(ctx context.Context, zipPath, dest string) error {
	zr, err := zip.OpenReader(zipPath)
	if errors.Is(err, zip.ErrInsecurePath) {
		if zr != nil {
			zr.Close()
		}
		return errors.New("压缩包包含不安全的路径，已拒绝解压")
	}
	if err != nil {
		return fmt.Errorf("%w: %v", errZipCorrupt, err)
	}
	defer zr.Close()
	var total int64
	for _, fh := range zr.File {
		if _, ok := safeRel(fh.Name); !ok && strings.Trim(fh.Name, "/.") != "" {
			return fmt.Errorf("压缩包包含不安全的路径 %q，已拒绝解压", fh.Name)
		}
		total += int64(fh.UncompressedSize64)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	absDest, _ := filepath.Abs(dest)
	var done int64
	for _, fh := range zr.File {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		rel, ok := safeRel(fh.Name)
		if !ok {
			continue
		}
		target := filepath.Join(absDest, rel)
		if r, err := filepath.Rel(absDest, target); err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
			return fmt.Errorf("压缩包包含不安全的路径 %q，已拒绝解压", fh.Name)
		}
		if fh.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if fh.Mode()&os.ModeSymlink != 0 {
			continue // never follow links from an archive
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := fh.Open()
		if err != nil {
			return fmt.Errorf("%w: %v", errZipCorrupt, err)
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
		if err != nil {
			rc.Close()
			return err
		}
		n, err := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			if strings.Contains(err.Error(), "checksum") || errors.Is(err, zip.ErrChecksum) || errors.Is(err, zip.ErrFormat) {
				return fmt.Errorf("%w: %v", errZipCorrupt, err)
			}
			return fmt.Errorf("解压 %s 失败: %w", rel, err)
		}
		done += n
		d := done
		k.patch(false, func(s *KernelProgress) {
			s.Done, s.Total = d, total
			if total > 0 {
				s.Pct = float64(d) * 100 / float64(total)
			}
		})
	}
	return nil
}

func findChromeExeDir(root string, depth int) (string, error) {
	var found string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || found != "" {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(root, p)
		if d.IsDir() && rel != "." && strings.Count(rel, string(filepath.Separator))+1 > depth {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.EqualFold(d.Name(), "chrome.exe") {
			found = filepath.Dir(p)
			return filepath.SkipAll
		}
		return nil
	})
	if found == "" {
		return "", errors.New("校验失败：压缩包中没有 chrome.exe，可能不是 Windows 版 Chromium")
	}
	return found, nil
}

var kernelDirVerRe = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`)

// kernelVersionIn reads the full version from the version folder or
// <version>.manifest that ships next to chrome.exe.
func kernelVersionIn(dir string) string {
	best := ""
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		n := strings.TrimSuffix(e.Name(), ".manifest")
		if kernelDirVerRe.MatchString(n) && verLess(best, n) {
			best = n
		}
	}
	return best
}

func verLess(a, b string) bool {
	if a == "" {
		return true
	}
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		x, _ := strconv.Atoi(pa[i])
		y, _ := strconv.Atoi(pb[i])
		if x != y {
			return x < y
		}
	}
	return len(pa) < len(pb)
}

// ---- installed kernels ----

func listKernels(root, active string) []InstalledKernel {
	out := []InstalledKernel{}
	ents, _ := os.ReadDir(root)
	for _, e := range ents {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "chromium-") {
			continue
		}
		dir := filepath.Join(root, e.Name())
		exe := filepath.Join(dir, "chrome.exe")
		fi, err := os.Stat(exe)
		if err != nil {
			continue
		}
		var size int64
		_ = filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				if i, e := d.Info(); e == nil {
					size += i.Size()
				}
			}
			return nil
		})
		ver := kernelVersionIn(dir)
		if ver == "" {
			ver = strings.TrimPrefix(e.Name(), "chromium-")
		}
		out = append(out, InstalledKernel{Name: e.Name(), Version: ver, Path: exe, Dir: dir, Size: size,
			Installed: fi.ModTime().UnixMilli(), Active: active != "" && strings.EqualFold(filepath.Clean(active), filepath.Clean(exe))})
	}
	sort.Slice(out, func(i, j int) bool { return verLess(out[j].Version, out[i].Version) })
	return out
}

func deleteKernel(root, dir string) error {
	absRoot, _ := filepath.Abs(root)
	abs, _ := filepath.Abs(dir)
	rel, err := filepath.Rel(absRoot, abs)
	if err != nil || rel == "." || strings.Contains(rel, string(filepath.Separator)) || !strings.HasPrefix(rel, "chromium-") {
		return errors.New("只能删除一键安装的内核")
	}
	if err := os.RemoveAll(abs); err != nil {
		return fmt.Errorf("删除失败，请先关闭使用该内核的环境: %w", err)
	}
	return nil
}

// ---- helpers ----

// newKernelClient is swapped by tests (the env proxy cannot reach httptest).
var newKernelClient = kernelHTTPClient

func kernelHTTPClient() *http.Client {
	c := httpClient()
	c.Timeout = 0 // large file: rely on context + per-read stall detection by the transport
	if tr, ok := c.Transport.(*http.Transport); ok {
		tr.ResponseHeaderTimeout = 30 * time.Second
	}
	return c
}

func safeName(s string) string {
	return regexp.MustCompile(`[^A-Za-z0-9._-]+`).ReplaceAllString(s, "_")
}

type netErr struct{ err error }

func (e *netErr) Error() string { return "网络错误: " + e.err.Error() }
func (e *netErr) Unwrap() error { return e.err }

type httpErr struct {
	code int
	what string
}

func (e *httpErr) Error() string { return fmt.Sprintf("下载%s失败: HTTP %d", e.what, e.code) }

type diskErr struct{ need, free int64 }

func (e *diskErr) Error() string {
	return fmt.Sprintf("磁盘空间不足：需要约 %.1f GB，可用 %.1f GB", float64(e.need)/(1<<30), float64(e.free)/(1<<30))
}

func retryable(err error) bool {
	var ne *netErr
	var he *httpErr
	if errors.As(err, &he) {
		return he.code >= 500 || he.code == 429
	}
	return errors.As(err, &ne)
}

// kernelErrHint turns an error into a short Chinese message and a next step.
func kernelErrHint(err error, source string) (string, string) {
	switchTo := "请切换到「国内镜像」或填写自定义下载地址"
	if source == "mirror" {
		switchTo = "请检查网络或代理设置，也可以切换到「自定义地址」使用其他镜像"
	}
	var ne *netErr
	var he *httpErr
	var de *diskErr
	var oe *net.OpError
	switch {
	case errors.As(err, &de):
		return de.Error(), "清理系统盘空间后点击「重试」"
	case errors.As(err, &he) && (he.code == 403 || he.code == 404):
		return he.Error(), "该下载源无法访问此文件，" + switchTo
	case errors.As(err, &he):
		return he.Error(), "下载服务器暂时异常，稍后点击「重试」，或换一个下载源"
	case errors.As(err, &oe), errors.As(err, &ne):
		return "网络连接失败或被中断", "可能无法访问该下载源，" + switchTo + "；点击「重试」会从断点继续"
	case errors.Is(err, errZipCorrupt):
		return "压缩包已损坏", "已删除损坏的文件，点击「重试」重新下载"
	}
	return err.Error(), "点击「重试」，如仍失败请换一个下载源"
}
