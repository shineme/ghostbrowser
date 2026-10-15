package main

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Managed proxy core: mihomo (MetaCubeX/mihomo, GPL-3.0) run as a separate,
// downloaded executable — its code is never vendored or linked. Browsers only
// speak http / socks5, so every subscription node of another protocol (vless,
// vmess, ss, trojan, hysteria2, tuic, anytls…) is given one local mixed
// listener on 127.0.0.1:<port> whose traffic goes straight to that node
// (`listeners: … proxy: <node>`). One shared process serves every node in use;
// it is reloaded through its REST API when the set changes and stopped when
// nothing uses it any more.
//
// Users of a node hold a "lease" (profile:<id> while a browser runs, check:<n>
// during a probe). The core config lives in appDataDir()/core/.

const coreRepo = "MetaCubeX/mihomo"

type CoreStatus struct {
	Installed   bool    `json:"installed"`
	Path        string  `json:"path"`
	Version     string  `json:"version"`
	Managed     bool    `json:"managed"` // downloaded by GhostBrowser into its data folder
	Running     bool    `json:"running"`
	PID         int     `json:"pid"`
	Listeners   int     `json:"listeners"`
	Leases      int     `json:"leases"`
	Error       string  `json:"error"`
	Downloading bool    `json:"downloading"`
	Progress    float64 `json:"progress"` // 0–100 while downloading
	Dir         string  `json:"dir"`
}

type coreConf struct {
	Path string `json:"path"` // user-picked or downloaded executable
}

type coreManager struct {
	mu       sync.Mutex
	dir      string // "" = appDataDir()/core
	cmd      *exec.Cmd
	exited   chan struct{}
	ctl      string // 127.0.0.1:<port>
	secret   string
	ports    map[string]int   // node ID -> listener port
	nodes    map[string]Proxy // node ID -> node (RawNode)
	leases   map[string]map[string]bool
	applied  string // hash of the config the process runs
	version  string
	verPath  string
	lastErr  string
	dl       bool
	dlPct    float64
	readyFor time.Duration
}

var coreMgr = &coreManager{}

var errCoreMissing = fmt.Errorf("需要内核")

func (c *coreManager) folder() string {
	d := c.dir
	if d == "" {
		d = filepath.Join(appDataDir(), "core")
	}
	_ = os.MkdirAll(d, 0o755)
	return d
}

func coreExeName() string {
	if runtime.GOOS == "windows" {
		return "mihomo.exe"
	}
	return "mihomo"
}

func (c *coreManager) loadConf() coreConf {
	var cf coreConf
	if b, err := os.ReadFile(filepath.Join(c.folder(), "core.json")); err == nil {
		_ = json.Unmarshal(b, &cf)
	}
	return cf
}

func (c *coreManager) saveConf(cf coreConf) error {
	b, _ := json.MarshalIndent(cf, "", "  ")
	return os.WriteFile(filepath.Join(c.folder(), "core.json"), b, 0o600)
}

// binPath: GHOST_MIHOMO (tests), the saved path, then the downloaded copy.
func (c *coreManager) binPath() string {
	if p := os.Getenv("GHOST_MIHOMO"); p != "" && fileExists(p) {
		return p
	}
	if p := c.loadConf().Path; p != "" && fileExists(p) {
		return p
	}
	if p := filepath.Join(c.folder(), coreExeName()); fileExists(p) {
		return p
	}
	return ""
}

var reCoreVer = regexp.MustCompile(`(?i)(mihomo|clash)[^\n]*?\b(v?\d+\.\d+\.\d+[\w.-]*)`)

// coreVersion runs `<bin> -v` and accepts only a mihomo / Clash Meta binary.
func coreVersion(bin string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-v")
	hideConsole(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) == 0 {
		return "", fmt.Errorf("无法运行内核：%v", err)
	}
	m := reCoreVer.FindStringSubmatch(string(out))
	if m == nil {
		return "", fmt.Errorf("不是 mihomo 内核（-v 输出：%s）", strings.TrimSpace(firstLine(string(out))))
	}
	return m[2], nil
}

func (c *coreManager) versionOf(bin string) string {
	c.mu.Lock()
	if c.verPath == bin && c.version != "" {
		v := c.version
		c.mu.Unlock()
		return v
	}
	c.mu.Unlock()
	v, err := coreVersion(bin)
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		c.lastErr = err.Error()
		return ""
	}
	c.version, c.verPath = v, bin
	return v
}

func (c *coreManager) Status() CoreStatus {
	bin := c.binPath()
	st := CoreStatus{Path: bin, Dir: c.folder()}
	if bin != "" {
		st.Version = c.versionOf(bin)
		st.Installed = st.Version != ""
		st.Managed = filepath.Dir(bin) == c.folder()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	st.Running = c.cmd != nil && c.cmd.Process != nil
	if st.Running {
		st.PID = c.cmd.Process.Pid
	}
	st.Listeners, st.Leases = len(c.ports), len(c.leases)
	st.Downloading, st.Progress = c.dl, c.dlPct
	if bin != "" && !st.Installed {
		st.Error = c.lastErr
	}
	return st
}

// SetPath verifies and saves a user-picked executable ("" = back to the downloaded copy).
func (c *coreManager) SetPath(p string) (CoreStatus, error) {
	p = strings.Trim(strings.TrimSpace(p), `"`)
	if p != "" {
		if !fileExists(p) {
			return c.Status(), fmt.Errorf("文件不存在：%s", p)
		}
		if _, err := coreVersion(p); err != nil {
			return c.Status(), err
		}
	}
	if err := c.saveConf(coreConf{Path: p}); err != nil {
		return c.Status(), err
	}
	c.mu.Lock()
	c.version, c.verPath = "", ""
	c.mu.Unlock()
	return c.Status(), nil
}

// ---------------------------------------------------------------- download

type ghRelease struct {
	Tag    string `json:"tag_name"`
	Assets []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
		Size int64  `json:"size"`
	} `json:"assets"`
}

// pickCoreAsset chooses the plain build for this OS / CPU; on amd64 the
// "compatible" build, which runs on every x86-64 CPU.
func pickCoreAsset(rel ghRelease, goos, goarch string) (string, string, int64) {
	ext := ".gz"
	if goos == "windows" {
		ext = ".zip"
	}
	want := []string{fmt.Sprintf("mihomo-%s-%s-%s%s", goos, goarch, rel.Tag, ext)}
	if goarch == "amd64" {
		want = append([]string{fmt.Sprintf("mihomo-%s-amd64-compatible-%s%s", goos, rel.Tag, ext)}, want...)
	}
	for _, w := range want {
		for _, a := range rel.Assets {
			if a.Name == w {
				return a.Name, a.URL, a.Size
			}
		}
	}
	return "", "", 0
}

var coreReleaseAPI = "https://api.github.com/repos/" + coreRepo + "/releases/latest"

func coreHTTP(px *url.URL, timeout time.Duration) *http.Client {
	tr := &http.Transport{Proxy: nil}
	if px != nil {
		tr.Proxy = http.ProxyURL(px)
	}
	return &http.Client{Timeout: timeout, Transport: tr}
}

func coreGet(u string, max int64, progress func(done, total int64)) ([]byte, error) {
	try := func(px *url.URL) ([]byte, error) {
		req, _ := http.NewRequest("GET", u, nil)
		req.Header.Set("User-Agent", "GhostBrowser")
		resp, err := coreHTTP(px, 10*time.Minute).Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		var buf bytes.Buffer
		total := resp.ContentLength
		chunk := make([]byte, 64<<10)
		for {
			n, err := resp.Body.Read(chunk)
			buf.Write(chunk[:n])
			if int64(buf.Len()) > max {
				return nil, fmt.Errorf("文件过大")
			}
			if progress != nil {
				progress(int64(buf.Len()), total)
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
		}
		return buf.Bytes(), nil
	}
	b, err := try(nil)
	if err == nil {
		return b, nil
	}
	if px := mihomoListenerURL(); px != nil {
		if b2, err2 := try(px); err2 == nil {
			return b2, nil
		}
	}
	return nil, err
}

// extractCore pulls the executable out of a release .zip or .gz.
func extractCore(name string, data []byte) ([]byte, error) {
	if strings.HasSuffix(name, ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			base := strings.ToLower(filepath.Base(f.Name))
			if strings.HasPrefix(base, "mihomo") && !f.FileInfo().IsDir() {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(io.LimitReader(rc, 200<<20))
			}
		}
		return nil, fmt.Errorf("压缩包里没有 mihomo 可执行文件")
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	return io.ReadAll(io.LimitReader(gz, 200<<20))
}

// Download fetches the latest mihomo release for this computer from GitHub,
// verifies it with -v and installs it into the data folder.
func (c *coreManager) Download(emit func(CoreStatus)) (CoreStatus, error) {
	c.mu.Lock()
	if c.dl {
		c.mu.Unlock()
		return c.Status(), fmt.Errorf("正在下载中")
	}
	c.dl, c.dlPct = true, 0
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.dl = false
		c.mu.Unlock()
		if emit != nil {
			emit(c.Status())
		}
	}()
	fail := func(err error) (CoreStatus, error) {
		return c.Status(), fmt.Errorf("下载 mihomo 失败：%v。可以从 https://github.com/%s/releases 手动下载后在这里选择", err, coreRepo)
	}
	b, err := coreGet(coreReleaseAPI, 4<<20, nil)
	if err != nil {
		return fail(err)
	}
	var rel ghRelease
	if err := json.Unmarshal(b, &rel); err != nil || rel.Tag == "" {
		return fail(fmt.Errorf("无法读取版本信息"))
	}
	name, u, _ := pickCoreAsset(rel, runtime.GOOS, runtime.GOARCH)
	if u == "" {
		return fail(fmt.Errorf("没有适合 %s/%s 的版本", runtime.GOOS, runtime.GOARCH))
	}
	last := time.Time{}
	data, err := coreGet(u, 120<<20, func(done, total int64) {
		if total <= 0 || time.Since(last) < 250*time.Millisecond {
			return
		}
		last = time.Now()
		c.mu.Lock()
		c.dlPct = float64(done) * 100 / float64(total)
		c.mu.Unlock()
		if emit != nil {
			emit(c.Status())
		}
	})
	if err != nil {
		return fail(err)
	}
	exe, err := extractCore(name, data)
	if err != nil {
		return fail(err)
	}
	dst := filepath.Join(c.folder(), coreExeName())
	tmp := dst + ".new"
	if err := os.WriteFile(tmp, exe, 0o755); err != nil {
		return fail(err)
	}
	if _, err := coreVersion(tmp); err != nil {
		_ = os.Remove(tmp)
		return fail(err)
	}
	c.mu.Lock()
	running := c.cmd != nil
	c.mu.Unlock()
	if running { // replace the binary under a running core: stop, swap, restart on next use
		c.StopAll()
	}
	_ = os.Remove(dst)
	if err := os.Rename(tmp, dst); err != nil {
		return fail(err)
	}
	if err := c.saveConf(coreConf{Path: ""}); err != nil {
		return fail(err)
	}
	c.mu.Lock()
	c.version, c.verPath, c.lastErr = "", "", ""
	c.mu.Unlock()
	return c.Status(), nil
}

// ---------------------------------------------------------------- leases

// coreNodeName is the name a node gets inside the generated config.
func coreNodeName(id string) string { return "gb-" + id }

// Acquire makes the given nodes reachable on local listeners and records the
// lease. Direct (http / socks5) nodes are ignored.
func (c *coreManager) Acquire(lease string, nodes ...Proxy) error {
	var need []Proxy
	for _, n := range nodes {
		if n.needsCore() {
			need = append(need, n)
		}
	}
	if len(need) == 0 {
		return nil
	}
	if c.binPath() == "" {
		return fmt.Errorf("%w：%s 节点需要代理内核 mihomo，请在「设置 → 代理内核 (mihomo)」一键下载或选择本地 mihomo 程序", errCoreMissing, strings.ToUpper(need[0].Protocol))
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ports == nil {
		c.ports, c.nodes, c.leases = map[string]int{}, map[string]Proxy{}, map[string]map[string]bool{}
	}
	if c.leases[lease] == nil {
		c.leases[lease] = map[string]bool{}
	}
	for _, n := range need {
		c.leases[lease][n.ID] = true
		if old, ok := c.nodes[n.ID]; !ok || old.RawNode != n.RawNode {
			c.nodes[n.ID] = n
		}
		if c.ports[n.ID] == 0 {
			c.ports[n.ID] = freePort()
		}
	}
	if err := c.applyLocked(); err != nil {
		for _, n := range need {
			delete(c.leases[lease], n.ID)
		}
		if len(c.leases[lease]) == 0 {
			delete(c.leases, lease)
		}
		c.pruneLocked()
		if len(c.leases) > 0 {
			_ = c.applyLocked() // put back what other users had
		} else {
			c.stopProcLocked()
		}
		return err
	}
	return nil
}

// Local returns the listener proxy that stands in for a leased core node.
func (c *coreManager) Local(n Proxy) (Proxy, bool) {
	c.mu.Lock()
	port := c.ports[n.ID]
	c.mu.Unlock()
	if port == 0 {
		return Proxy{}, false
	}
	l := n // keep name and last check (timezone, country…) for geo matching
	l.ID, l.CoreOf = "", n.ID
	l.Type, l.Host, l.Port, l.User, l.Pass = "socks5", "127.0.0.1", port, "", ""
	l.RawNode, l.Protocol, l.SubID, l.NodeKey, l.ChangeIP, l.RotateOnLaunch, l.Guessed = "", "", "", "", "", false, false
	return l, true
}

// Release drops a lease. Listeners of nodes nobody uses stay until the next
// reload (so running browsers are not disturbed); the core stops when the
// last lease is gone.
func (c *coreManager) Release(lease string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.leases[lease]; !ok {
		return
	}
	delete(c.leases, lease)
	c.pruneLocked()
	if len(c.leases) == 0 {
		c.stopProcLocked()
	}
}

func (c *coreManager) stopProcLocked() {
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		if c.exited != nil {
			select {
			case <-c.exited:
			case <-time.After(3 * time.Second):
			}
		}
	}
	c.cmd, c.exited, c.applied = nil, nil, ""
}

func (c *coreManager) pruneLocked() {
	used := map[string]bool{}
	for _, l := range c.leases {
		for id := range l {
			used[id] = true
		}
	}
	for id := range c.nodes {
		if !used[id] {
			delete(c.nodes, id)
			delete(c.ports, id)
		}
	}
}

// StopAll ends the core process and forgets every lease (app exit, binary swap).
func (c *coreManager) StopAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopProcLocked()
	c.leases, c.nodes, c.ports = map[string]map[string]bool{}, map[string]Proxy{}, map[string]int{}
}

// ---------------------------------------------------------------- config + process

// buildCoreConfig renders the mihomo config (JSON is valid YAML). No mixed
// port, no TUN, no DNS changes, LAN closed; every listener goes straight to
// its node and everything else would be DIRECT.
func buildCoreConfig(ctl, secret string, nodes map[string]Proxy, ports map[string]int) ([]byte, error) {
	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	proxies := []any{}
	listeners := []any{}
	for _, id := range ids {
		var m map[string]any
		if err := json.Unmarshal([]byte(nodes[id].RawNode), &m); err != nil {
			return nil, fmt.Errorf("节点「%s」数据损坏，请更新订阅", nodes[id].Name)
		}
		m["name"] = coreNodeName(id)
		proxies = append(proxies, m)
		listeners = append(listeners, map[string]any{"name": "in-" + id, "type": "mixed", "listen": "127.0.0.1", "port": ports[id], "proxy": coreNodeName(id), "udp": true})
	}
	cfg := map[string]any{
		"allow-lan":           false,
		"bind-address":        "127.0.0.1",
		"mode":                "rule",
		"log-level":           "warning",
		"ipv6":                true,
		"unified-delay":       true,
		"find-process-mode":   "off",
		"geo-auto-update":     false,
		"external-controller": ctl,
		"secret":              secret,
		"profile":             map[string]any{"store-selected": false, "store-fake-ip": false},
		"proxies":             proxies,
		"listeners":           listeners,
		"rules":               []any{"MATCH,DIRECT"},
	}
	return json.MarshalIndent(cfg, "", "  ")
}

func (c *coreManager) configPath() string { return filepath.Join(c.folder(), "config.yaml") }

// applyLocked writes the config and starts or reloads the core when it changed.
func (c *coreManager) applyLocked() error {
	if c.cmd != nil && c.exited != nil {
		select {
		case <-c.exited: // crashed: start over
			c.cmd, c.applied = nil, ""
		default:
		}
	}
	if c.cmd == nil {
		c.ctl, c.secret = fmt.Sprintf("127.0.0.1:%d", freePort()), randHex(16)
	}
	cfg, err := buildCoreConfig(c.ctl, c.secret, c.nodes, c.ports)
	if err != nil {
		return err
	}
	hash := string(cfg)
	if c.cmd != nil && hash == c.applied {
		return nil
	}
	if err := os.WriteFile(c.configPath(), cfg, 0o600); err != nil {
		return err
	}
	if c.cmd != nil {
		err := c.reloadLocked()
		if err == nil {
			c.applied = hash
			return c.waitPortsLocked()
		}
		if strings.HasPrefix(err.Error(), "mihomo HTTP") {
			// The core rejected the new config and keeps running the old one.
			return fmt.Errorf("内核拒绝了节点配置：%s", strings.TrimPrefix(err.Error(), "mihomo "))
		}
		c.killLocked() // controller unreachable: start over
	}
	if err := c.startLocked(); err != nil {
		return err
	}
	c.applied = hash
	return c.waitPortsLocked()
}

func (c *coreManager) killLocked() {
	c.stopProcLocked()
	c.ctl, c.secret = fmt.Sprintf("127.0.0.1:%d", freePort()), randHex(16)
	if cfg, err := buildCoreConfig(c.ctl, c.secret, c.nodes, c.ports); err == nil {
		_ = os.WriteFile(c.configPath(), cfg, 0o600)
	}
}

func (c *coreManager) logPath() string { return filepath.Join(c.folder(), "core.log") }

func (c *coreManager) startLocked() error {
	bin := c.binPath()
	if bin == "" {
		return fmt.Errorf("%w：请在「设置 → 代理内核 (mihomo)」一键下载", errCoreMissing)
	}
	logf, err := os.Create(c.logPath())
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, "-d", c.folder(), "-f", c.configPath())
	cmd.Dir = c.folder()
	cmd.Stdout, cmd.Stderr = logf, logf
	hideConsole(cmd) // Windows: CREATE_NO_WINDOW
	if err := cmd.Start(); err != nil {
		logf.Close()
		return fmt.Errorf("启动 mihomo 失败：%v", err)
	}
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); logf.Close(); close(exited) }()
	c.cmd, c.exited = cmd, exited
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-exited:
			c.cmd, c.exited = nil, nil
			return fmt.Errorf("mihomo 启动后立即退出：%s", c.logTail())
		default:
		}
		if c.api("GET", "/version", nil) == nil {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	c.killLocked()
	return fmt.Errorf("mihomo 启动超时：%s", c.logTail())
}

func (c *coreManager) logTail() string {
	b, _ := os.ReadFile(c.logPath())
	s := strings.TrimSpace(string(b))
	lines := strings.Split(s, "\n")
	if len(lines) > 3 {
		lines = lines[len(lines)-3:]
	}
	s = strings.Join(lines, " | ")
	if len(s) > 400 {
		s = s[len(s)-400:]
	}
	if s == "" {
		s = "没有日志输出"
	}
	return s
}

func (c *coreManager) api(method, path string, body any) error {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, "http://"+c.ctl+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.secret)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := coreHTTP(nil, 15*time.Second).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("mihomo HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func (c *coreManager) reloadLocked() error {
	return c.api("PUT", "/configs?force=true", map[string]string{"path": c.configPath()})
}

// waitPortsLocked waits until every listener accepts connections.
func (c *coreManager) waitPortsLocked() error {
	deadline := time.Now().Add(8 * time.Second)
	for id, port := range c.ports {
		for {
			conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 500*time.Millisecond)
			if err == nil {
				conn.Close()
				break
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("内核没有打开节点「%s」的本地端口 %d：%s", c.nodes[id].Name, port, c.logTail())
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	return nil
}

// ---------------------------------------------------------------- store glue

// resolveLaunchProxy returns the proxy a browser should use: core nodes are
// leased under "profile:<id>" and replaced by their local listener.
func resolveLaunchProxy(lease string, px *Proxy) (*Proxy, error) {
	if px == nil || !px.needsCore() {
		return px, nil
	}
	if err := coreMgr.Acquire(lease, *px); err != nil {
		return nil, err
	}
	l, ok := coreMgr.Local(*px)
	if !ok {
		return nil, fmt.Errorf("内核没有为节点「%s」分配端口", px.Name)
	}
	return &l, nil
}

// launchProxy picks the profile's proxy (inline first) and resolves core nodes.
func (s *Store) launchProxy(p *Profile) (*Proxy, error) {
	var px *Proxy
	if p.InlineProxy != nil {
		px = p.InlineProxy
	} else if p.ProxyID != "" {
		if pr, ok := s.GetProxy(p.ProxyID); ok {
			px = pr
		}
	}
	return resolveLaunchProxy("profile:"+p.ID, px)
}

// withCore runs fn against the local listener of a core node (temporary lease).
func withCore(p Proxy, fn func(local Proxy) Proxy) Proxy {
	lease := "check:" + randHex(6)
	defer coreMgr.Release(lease)
	if err := coreMgr.Acquire(lease, p); err != nil {
		p.Checked = time.Now().Unix()
		if strings.HasPrefix(err.Error(), errCoreMissing.Error()) {
			p.Status, p.Error = "unknown", "需要内核"
		} else {
			p.Status, p.Error = "fail", shortErr(err)
		}
		return p
	}
	l, ok := coreMgr.Local(p)
	if !ok {
		p.Status, p.Error = "fail", "内核没有分配端口"
		return p
	}
	r := fn(l)
	mergeCheck(&p, r)
	if r.SpeedAt != 0 {
		p.Speed, p.SpeedAt = r.Speed, r.SpeedAt
	}
	return p
}

// coreBatch leases every core node of a batch at once (one reload instead of
// one per node). When the core rejects the batch, each check falls back to
// its own lease, so one broken node cannot fail the rest. Returns the release.
func coreBatch(list []Proxy) func() {
	var core []Proxy
	for _, p := range list {
		if p.needsCore() {
			core = append(core, p)
		}
	}
	if len(core) < 2 || coreMgr.binPath() == "" {
		return func() {}
	}
	lease := "batch:" + randHex(6)
	_ = coreMgr.Acquire(lease, core...)
	return func() { coreMgr.Release(lease) }
}

// mergeCheck copies the result of a check onto the node record.
func mergeCheck(dst *Proxy, r Proxy) {
	dst.IP, dst.Country, dst.City, dst.Region, dst.Timezone = r.IP, r.Country, r.City, r.Region, r.Timezone
	dst.Lat, dst.Lon, dst.Latency, dst.Connect, dst.ISP = r.Lat, r.Lon, r.Latency, r.Connect, r.ISP
	if r.IPType != "" {
		dst.IPType = r.IPType
	}
	dst.Error, dst.Status, dst.Checked = r.Error, r.Status, r.Checked
}

// ---------------------------------------------------------------- app bindings

func (a *App) emitCore(st CoreStatus) {
	if a.ctx != nil && a.ctx.Value("events") != nil {
		wruntime.EventsEmit(a.ctx, "core:status", st)
	}
}

func (a *App) CoreStatus() CoreStatus { return coreMgr.Status() }

// DownloadCore installs the latest mihomo release from GitHub.
func (a *App) DownloadCore() (CoreStatus, error) {
	st, err := coreMgr.Download(a.emitCore)
	if err == nil {
		a.store.auditNamed("ui", "core.download", "", "mihomo "+st.Version, nil, st.Path)
	}
	return st, err
}

// SetCorePath uses a local mihomo executable ("" = the downloaded copy).
func (a *App) SetCorePath(path string) (CoreStatus, error) { return coreMgr.SetPath(path) }

// PickCorePath opens a file dialog for the mihomo executable.
func (a *App) PickCorePath() string {
	opts := wruntime.OpenDialogOptions{Title: "选择 mihomo 内核程序", DefaultDirectory: coreMgr.folder()}
	if runtime.GOOS == "windows" {
		opts.Filters = []wruntime.FileFilter{{DisplayName: "mihomo 程序 (*.exe)", Pattern: "*.exe"}}
	}
	f, _ := wruntime.OpenFileDialog(a.ctx, opts)
	return f
}

// StopCore stops the core process (running browsers on core nodes lose their connection).
func (a *App) StopCore() CoreStatus {
	coreMgr.StopAll()
	return coreMgr.Status()
}
