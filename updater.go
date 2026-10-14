package main

// In-app updater. Sources, in order:
//  1. Settings.UpdateURL: any latest.json URL,
//  2. the sync server (ghostsync.php ?r=/api/update/latest, token auth),
//  3. Settings.UpdateGitHub: a PUBLIC GitHub repo whose latest release has a
//     latest.json asset (a private repo needs a token, so it is not supported).
//
// latest.json = {version, notes, url, sha256, sig}. The client refuses to
// install unless the download's sha256 matches AND sig is a valid Ed25519
// signature (internal/updsig) by the key embedded below.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"ghostbrowser/internal/updsig"
)

// updatePubKey verifies releases. Generate your own pair with
// `go run ./cmd/sign-release keygen` and replace it here (or build with
// -ldflags "-X main.updatePubKey=..."); keep the private key offline.
var updatePubKey = "fqRcE1qCO6zRoCvBddOH+KtWV3Gb+kQjKsaVKDpXNqg="

func appVersion() string { return mcpAppVersion() }

type latestDoc struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Sig     string `json:"sig"`
}

// UpdateInfo is what the UI shows.
type UpdateInfo struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Notes     string `json:"notes"`
	Available bool   `json:"available"`
	Source    string `json:"source"` // custom | server | github | ""
	CheckedAt int64  `json:"checkedAt"`
	Error     string `json:"error,omitempty"`
}

// UpdateProgress is emitted as "update:progress".
type UpdateProgress struct {
	Phase string  `json:"phase"` // idle | downloading | verifying | installing | restarting | error
	Got   int64   `json:"got"`
	Total int64   `json:"total"`
	Pct   float64 `json:"pct"`
	Error string  `json:"error,omitempty"`
}

type updater struct {
	mu       sync.Mutex
	info     UpdateInfo
	doc      *latestDoc
	docBase  string // URL latest.json came from (relative url resolution)
	src      string
	prog     UpdateProgress
	busy     bool
	emit     func(name string, v any)
	hc       *http.Client
	exePath  func() (string, error)
	relaunch func(exe string) error
	quit     func()
}

var upd = &updater{hc: &http.Client{Timeout: 0}, exePath: os.Executable, prog: UpdateProgress{Phase: "idle"}}

// ---- version compare ----

// compareAppVersions returns -1, 0, 1. "v1.2.3", "1.2", "1.2.3-beta" (a
// pre-release sorts below the release).
func compareAppVersions(a, b string) int {
	pa, preA := splitVersion(a)
	pb, preB := splitVersion(b)
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	switch {
	case preA == preB:
		return 0
	case preA == "":
		return 1
	case preB == "":
		return -1
	case preA < preB:
		return -1
	}
	return 1
}

func splitVersion(v string) ([]int, string) {
	v = strings.TrimPrefix(strings.TrimSpace(strings.ToLower(v)), "v")
	pre := ""
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		pre, v = v[i+1:], v[:i]
	}
	out := []int{}
	for _, p := range strings.Split(v, ".") {
		n, _ := strconv.Atoi(p)
		out = append(out, n)
	}
	return out, pre
}

// ---- sources ----

func authGet(ctx context.Context, hc *http.Client, u, token string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "GhostBrowser/"+appVersion())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Ghost-Token", token)
	}
	return hc.Do(req)
}

func fetchJSON(ctx context.Context, hc *http.Client, u, token string, out any) error {
	resp, err := authGet(ctx, hc, u, token)
	if err != nil {
		return fmt.Errorf("连接失败：%v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != 200 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(body, &e)
		if e.Error != "" {
			return fmt.Errorf("HTTP %d：%s", resp.StatusCode, e.Error)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("返回内容不是有效的 JSON")
	}
	return nil
}

// sameHost: send the sync token only to the sync server itself.
func sameHost(a, b string) bool {
	ua, e1 := url.Parse(a)
	ub, e2 := url.Parse(b)
	return e1 == nil && e2 == nil && ua.Host != "" && strings.EqualFold(ua.Host, ub.Host)
}

// fetchLatest asks the configured sources and returns the first latest.json.
func (u *updater) fetchLatest(ctx context.Context, st Settings) (*latestDoc, string, string, error) {
	var errs []string
	try := func(src, link, token string) (*latestDoc, error) {
		var d latestDoc
		if err := fetchJSON(ctx, u.hc, link, token, &d); err != nil {
			return nil, err
		}
		if strings.TrimSpace(d.Version) == "" {
			return nil, errors.New("latest.json 缺少 version")
		}
		return &d, nil
	}
	if s := strings.TrimSpace(st.UpdateURL); s != "" {
		tok := ""
		if st.SyncServer != "" && sameHost(s, st.SyncServer) {
			tok = st.SyncToken
		}
		d, err := try("custom", s, tok)
		if err == nil {
			return d, "custom", s, nil
		}
		errs = append(errs, "自定义地址："+err.Error())
	}
	if st.SyncServer != "" && st.SyncToken != "" {
		link := syncURL(strings.TrimRight(st.SyncServer, "/"), "/api/update/latest")
		d, err := try("server", link, st.SyncToken)
		if err == nil {
			return d, "server", link, nil
		}
		errs = append(errs, "同步服务器："+err.Error())
	}
	if repo := strings.Trim(strings.TrimSpace(st.UpdateGitHub), "/"); repo != "" {
		var rel struct {
			Assets []struct {
				Name string `json:"name"`
				URL  string `json:"browser_download_url"`
			} `json:"assets"`
		}
		if err := fetchJSON(ctx, u.hc, "https://api.github.com/repos/"+repo+"/releases/latest", "", &rel); err != nil {
			errs = append(errs, "GitHub："+err.Error())
		} else {
			found := false
			for _, a := range rel.Assets {
				if a.Name == "latest.json" {
					found = true
					d, err := try("github", a.URL, "")
					if err == nil {
						return d, "github", a.URL, nil
					}
					errs = append(errs, "GitHub："+err.Error())
				}
			}
			if !found {
				errs = append(errs, "GitHub：最新 Release 没有 latest.json 附件")
			}
		}
	}
	if len(errs) == 0 {
		return nil, "", "", errors.New("没有配置更新源：请先设置同步服务器，或在 设置 → 更新 填写更新地址")
	}
	return nil, "", "", errors.New(strings.Join(errs, "；"))
}

// downloadURL resolves latest.json's url: a bare file name means a file in
// update/ on the sync server; a relative path is relative to latest.json.
func downloadURL(st Settings, src, base, ref string) (string, string) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", ""
	}
	if pu, err := url.Parse(ref); err == nil && pu.Scheme != "" {
		if pu.Scheme != "https" && pu.Scheme != "http" {
			return "", ""
		}
		tok := ""
		if st.SyncServer != "" && sameHost(ref, st.SyncServer) {
			tok = st.SyncToken
		}
		return ref, tok
	}
	if src == "server" && !strings.ContainsAny(ref, "/\\?#") {
		return syncURL(strings.TrimRight(st.SyncServer, "/"), "/api/update/file/"+ref), st.SyncToken
	}
	b, err := url.Parse(base)
	if err != nil {
		return "", ""
	}
	r, err := b.Parse(ref)
	if err != nil {
		return "", ""
	}
	tok := ""
	if st.SyncServer != "" && sameHost(r.String(), st.SyncServer) {
		tok = st.SyncToken
	}
	return r.String(), tok
}

// Check fetches latest.json and records whether it is newer.
func (u *updater) Check(st Settings) UpdateInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	info := UpdateInfo{Current: appVersion(), CheckedAt: time.Now().Unix()}
	d, src, base, err := u.fetchLatest(ctx, st)
	u.mu.Lock()
	defer u.mu.Unlock()
	if err != nil {
		info.Error = err.Error()
		// keep a previously found update visible
		if u.doc != nil {
			info.Latest, info.Notes, info.Available, info.Source = u.info.Latest, u.info.Notes, u.info.Available, u.info.Source
		}
		u.info = info
		return info
	}
	info.Latest, info.Notes, info.Source = strings.TrimPrefix(d.Version, "v"), d.Notes, src
	info.Available = compareAppVersions(d.Version, appVersion()) > 0
	u.doc, u.src, u.docBase, u.info = d, src, base, info
	if info.Available && u.emit != nil {
		go u.emit("update:available", info)
	}
	return info
}

func (u *updater) Info() UpdateInfo {
	u.mu.Lock()
	defer u.mu.Unlock()
	i := u.info
	if i.Current == "" {
		i.Current = appVersion()
	}
	return i
}

func (u *updater) Progress() UpdateProgress {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.prog
}

func (u *updater) setProg(p UpdateProgress) {
	u.mu.Lock()
	u.prog = p
	emit := u.emit
	u.mu.Unlock()
	if emit != nil {
		emit("update:progress", p)
	}
}

// download fetches the release into dst, reporting progress, and verifies
// sha256 and the signature. dst is removed on any failure.
func (u *updater) download(st Settings, d *latestDoc, src, base, dst string) error {
	pub, err := updsig.ParsePublic(updatePubKey)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(d.SHA256)) != 64 || strings.TrimSpace(d.Sig) == "" {
		return errors.New("latest.json 缺少 sha256 或 sig，已拒绝安装")
	}
	// Check the signature before downloading anything large.
	if err := updsig.Verify(pub, d.Version, d.SHA256, d.Sig); err != nil {
		return err
	}
	link, tok := downloadURL(st, src, base, d.URL)
	if link == "" {
		return errors.New("latest.json 的 url 无效")
	}
	resp, err := authGet(context.Background(), u.hc, link, tok)
	if err != nil {
		return fmt.Errorf("下载失败：%v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("下载失败：HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("无法写入更新文件：%v", err)
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(dst)
		}
	}()
	h := sha256.New()
	total := resp.ContentLength
	var got int64
	last := time.Time{}
	buf := make([]byte, 256<<10)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				return fmt.Errorf("写入失败：%v", werr)
			}
			h.Write(buf[:n])
			got += int64(n)
			if time.Since(last) > 120*time.Millisecond {
				last = time.Now()
				p := UpdateProgress{Phase: "downloading", Got: got, Total: total}
				if total > 0 {
					p.Pct = float64(got) * 100 / float64(total)
				}
				u.setProg(p)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			return fmt.Errorf("下载中断：%v", rerr)
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	u.setProg(UpdateProgress{Phase: "verifying", Got: got, Total: got, Pct: 100})
	sum := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(sum, strings.TrimSpace(d.SHA256)) {
		return errors.New("更新包 SHA-256 校验失败：下载的文件与发布信息不一致，已拒绝安装")
	}
	if err := updsig.Verify(pub, d.Version, sum, d.Sig); err != nil {
		return err
	}
	ok = true
	return nil
}

// replaceExe swaps the running executable: exe -> exe.old, newFile -> exe.
// Windows allows renaming (not deleting) a running EXE.
func replaceExe(exe, newFile string) error {
	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		return fmt.Errorf("无法替换程序文件（%v）；请确认程序目录可写", err)
	}
	if err := os.Rename(newFile, exe); err != nil {
		_ = os.Rename(old, exe) // roll back
		return fmt.Errorf("无法写入新版本：%v", err)
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(exe, 0o755)
	}
	return nil
}

// cleanupOldExe removes exe.old left by the previous update (it may still be
// locked for a moment while the old process exits).
func cleanupOldExe() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	old := exe + ".old"
	for i := 0; i < 20; i++ {
		if _, err := os.Stat(old); os.IsNotExist(err) {
			return
		}
		if os.Remove(old) == nil {
			return
		}
		time.Sleep(time.Second)
	}
}

// Install downloads, verifies, replaces the EXE and restarts. Runs async;
// progress arrives as "update:progress".
func (u *updater) Install(st Settings) error {
	u.mu.Lock()
	if u.busy {
		u.mu.Unlock()
		return errors.New("正在更新中")
	}
	d, src, base := u.doc, u.src, u.docBase
	if d == nil || !u.info.Available {
		u.mu.Unlock()
		return errors.New("没有可用的更新，请先检查更新")
	}
	u.busy = true
	u.mu.Unlock()
	go func() {
		defer func() { u.mu.Lock(); u.busy = false; u.mu.Unlock() }()
		fail := func(err error) { u.setProg(UpdateProgress{Phase: "error", Error: err.Error()}) }
		exe, err := u.exePath()
		if err != nil {
			fail(err)
			return
		}
		if r, e := filepath.EvalSymlinks(exe); e == nil {
			exe = r
		}
		u.setProg(UpdateProgress{Phase: "downloading"})
		tmp := exe + ".download"
		if err := u.download(st, d, src, base, tmp); err != nil {
			fail(err)
			return
		}
		u.setProg(UpdateProgress{Phase: "installing", Pct: 100})
		if err := replaceExe(exe, tmp); err != nil {
			_ = os.Remove(tmp)
			fail(err)
			return
		}
		u.setProg(UpdateProgress{Phase: "restarting", Pct: 100})
		relaunch := u.relaunch
		if relaunch == nil {
			relaunch = relaunchAfterUpdate
		}
		if err := relaunch(exe); err != nil {
			fail(fmt.Errorf("新版本已安装，但自动重启失败（%v），请手动重新打开 GhostBrowser", err))
			return
		}
		if u.quit != nil {
			u.quit()
		}
	}()
	return nil
}

// relaunchAfterUpdate starts the new EXE, which waits for this process to
// exit (profiles stopped, sync drained) before it opens its window.
func relaunchAfterUpdate(exe string) error {
	cmd := exec.Command(exe, "--after-update", strconv.Itoa(os.Getpid()))
	cmd.Dir = filepath.Dir(exe)
	return cmd.Start()
}

// handleAfterUpdate: in a freshly updated process, wait (max 3 min) for the
// previous instance to exit so the local API port and data files are free.
func handleAfterUpdate(args []string) {
	for i, a := range args {
		if a == "--after-update" && i+1 < len(args) {
			pid, err := strconv.Atoi(args[i+1])
			if err == nil && pid > 0 && pid != os.Getpid() {
				waitPIDExit(pid, 3*time.Minute)
			}
		}
	}
}

func waitPIDExit(pid int, max time.Duration) {
	p, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		if runtime.GOOS == "windows" {
			_, _ = p.Wait() // works for any process handle on Windows
			return
		}
		for processAlive(p) {
			time.Sleep(300 * time.Millisecond)
		}
	}()
	select {
	case <-done:
	case <-time.After(max):
	}
}

// updateLoop checks at start-up and every 6 hours unless turned off.
func (u *updater) loop(s *Store) {
	time.Sleep(15 * time.Second)
	for {
		if st := s.GetSettings(); !st.UpdateAutoOff {
			u.Check(st)
		}
		time.Sleep(6 * time.Hour)
	}
}

func processAlive(p *os.Process) bool { return p.Signal(syscall.Signal(0)) == nil }
