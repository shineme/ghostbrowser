package main

// Terminal launcher (终端): opens Claude Code / Codex CLI / Gemini CLI or a
// plain shell in a new console window, optionally inheriting a profile's
// timezone, language and proxy so command-line tools leave the same footprint
// as the profile's browser. Credentialed or SOCKS proxies go through the local
// relay (internal/undetectable) so every CLI gets a plain http://127.0.0.1
// proxy; the relay lives as long as the terminal process.

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"ghostbrowser/internal/undetectable"
)

// TerminalTool is one launchable CLI and whether it is installed.
type TerminalTool struct {
	ID         string `json:"id"`   // claude | codex | gemini | shell
	Name       string `json:"name"` // display name
	Bin        string `json:"bin"`  // command typed in the terminal
	Installed  bool   `json:"installed"`
	Version    string `json:"version"`
	Path       string `json:"path"`
	InstallURL string `json:"installUrl"`
	InstallCmd string `json:"installCmd"`
	Desc       string `json:"desc"`
}

type terminalToolDef struct {
	id, name, bin, url, install, desc string
}

var terminalToolDefs = []terminalToolDef{
	{"claude", "Claude Code", "claude", "https://docs.anthropic.com/en/docs/claude-code/setup", "npm install -g @anthropic-ai/claude-code", "Anthropic 的编程智能体"},
	{"codex", "Codex CLI", "codex", "https://github.com/openai/codex", "npm install -g @openai/codex", "OpenAI 的终端编程助手"},
	{"gemini", "Gemini CLI", "gemini", "https://github.com/google-gemini/gemini-cli", "npm install -g @google/gemini-cli", "Google 的开源命令行智能体"},
}

// shellTool is the plain terminal: Windows PowerShell on Windows, $SHELL elsewhere.
func shellTool() terminalToolDef {
	if runtime.GOOS == "windows" {
		return terminalToolDef{"shell", "PowerShell", "powershell.exe", "https://learn.microsoft.com/powershell/", "", "Windows PowerShell 纯终端"}
	}
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "sh"
	}
	return terminalToolDef{"shell", "终端", filepath.Base(sh), "", "", "系统默认 Shell 纯终端"}
}

func toolDef(id string) (terminalToolDef, bool) {
	if id == "shell" {
		return shellTool(), true
	}
	for _, d := range terminalToolDefs {
		if d.id == id {
			return d, true
		}
	}
	return terminalToolDef{}, false
}

var versionRe = regexp.MustCompile(`\d+\.\d+(?:\.\d+)?(?:[-+.][0-9A-Za-z.]+)?`)

// parseVersion pulls a version number out of `--version` output.
func parseVersion(out string) string {
	if m := versionRe.FindString(out); m != "" {
		return m
	}
	return strings.TrimSpace(firstLine(out))
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}

// lookPathIn is exec.LookPath against an explicit PATH (tests use a fake one).
func lookPathIn(bin, pathEnv string) string {
	exts := []string{""}
	if runtime.GOOS == "windows" && filepath.Ext(bin) == "" {
		pe := os.Getenv("PATHEXT")
		if pe == "" {
			pe = ".COM;.EXE;.BAT;.CMD"
		}
		exts = strings.Split(strings.ToLower(pe), ";")
	}
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}
		for _, e := range exts {
			p := filepath.Join(dir, bin+e)
			if st, err := os.Stat(p); err == nil && !st.IsDir() && (runtime.GOOS == "windows" || st.Mode()&0o111 != 0) {
				return p
			}
		}
	}
	return ""
}

// detectTools checks every CLI on pathEnv in parallel (version calls time out after 6 s).
func detectTools(pathEnv string) []TerminalTool {
	defs := append(append([]terminalToolDef{}, terminalToolDefs...), shellTool())
	out := make([]TerminalTool, len(defs))
	var wg sync.WaitGroup
	for i, d := range defs {
		out[i] = TerminalTool{ID: d.id, Name: d.name, Bin: d.bin, InstallURL: d.url, InstallCmd: d.install, Desc: d.desc}
		wg.Add(1)
		go func(t *TerminalTool) {
			defer wg.Done()
			p := lookPathIn(t.Bin, pathEnv)
			if p == "" {
				return
			}
			t.Installed, t.Path = true, p
			if t.ID == "shell" && runtime.GOOS == "windows" {
				return // powershell --version opens nothing useful; it is always there
			}
			ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, p, "--version")
			cmd.Env = mergeEnv(os.Environ(), map[string]string{"PATH": pathEnv, "NO_COLOR": "1"}, runtime.GOOS == "windows")
			hideConsole(cmd)
			b, _ := cmd.CombinedOutput()
			t.Version = parseVersion(string(b))
		}(&out[i])
	}
	wg.Wait()
	return out
}

// ---------- environment ----------

// TerminalRequest is what the 终端 page asks to launch.
type TerminalRequest struct {
	Tool      string `json:"tool"`
	Dir       string `json:"dir"`
	EnvMode   string `json:"envMode"` // none | profile | custom
	ProfileID string `json:"profileId"`
	TZ        string `json:"tz"`
	Lang      string `json:"lang"`
	Host      string `json:"host"` // console | wt
}

// TerminalPrefs remembers the last choices on the 终端 page.
type TerminalPrefs struct {
	Tool      string `json:"tool"`
	Dir       string `json:"dir"`
	EnvMode   string `json:"envMode"`
	ProfileID string `json:"profileId"`
	TZ        string `json:"tz"`
	Lang      string `json:"lang"`
	Host      string `json:"host"`
}

// posixLocale turns a BCP 47 tag (zh-CN, en-US, pt-BR) into LANG form (zh_CN.UTF-8).
func posixLocale(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return ""
	}
	parts := strings.Split(strings.ReplaceAll(tag, "_", "-"), "-")
	lang := strings.ToLower(parts[0])
	region := ""
	for _, p := range parts[1:] {
		if len(p) == 2 {
			region = strings.ToUpper(p)
			break
		}
	}
	// zh-Hans / zh-Hant without a region
	if region == "" && lang == "zh" {
		region = "CN"
		if strings.Contains(strings.ToLower(tag), "hant") {
			region = "TW"
		}
	}
	if region == "" {
		return lang + ".UTF-8"
	}
	return lang + "_" + region + ".UTF-8"
}

// terminalLocaleEnv is TZ / LANG / LC_ALL for an IANA zone and a language tag.
func terminalLocaleEnv(tz, lang string) map[string]string {
	env := map[string]string{}
	if tz = strings.TrimSpace(tz); tz != "" {
		env["TZ"] = tz
	}
	if l := posixLocale(lang); l != "" {
		env["LANG"], env["LC_ALL"] = l, l
		env["LANGUAGE"] = strings.SplitN(l, ".", 2)[0]
	}
	return env
}

// terminalProxyPlan says how a profile proxy reaches the CLIs: Direct is the
// URL to export as is; Upstream is set when a local relay is needed
// (credentials, or SOCKS which many CLIs can't speak in HTTP(S)_PROXY).
type terminalProxyPlan struct {
	Direct   string
	Upstream string
	Display  string // shown in the UI, never with the password
}

func planTerminalProxy(px *Proxy) terminalProxyPlan {
	if px == nil || px.Type == "none" || px.Type == "" || px.Host == "" || px.Port == 0 {
		return terminalProxyPlan{}
	}
	u := px.URL()
	disp := fmt.Sprintf("%s %s:%d", strings.ToUpper(px.Type), px.Host, px.Port)
	if strings.HasPrefix(px.Type, "socks") {
		u.Scheme = "socks5"
		return terminalProxyPlan{Upstream: u.String(), Display: disp}
	}
	if undetectable.NeedsRelay(u.String()) {
		return terminalProxyPlan{Upstream: u.String(), Display: disp}
	}
	return terminalProxyPlan{Direct: u.String(), Display: disp}
}

// terminalProxyEnv exports one proxy URL under every name CLIs read.
func terminalProxyEnv(proxyURL string) map[string]string {
	if proxyURL == "" {
		return map[string]string{}
	}
	env := map[string]string{}
	for _, k := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY"} {
		env[k] = proxyURL
		env[strings.ToLower(k)] = proxyURL
	}
	env["NO_PROXY"] = "localhost,127.0.0.1,::1"
	env["no_proxy"] = env["NO_PROXY"]
	return env
}

// mergeEnv overlays extra on base. fold = case-insensitive keys (Windows).
func mergeEnv(base []string, extra map[string]string, fold bool) []string {
	norm := func(k string) string {
		if fold {
			return strings.ToUpper(k)
		}
		return k
	}
	over := map[string]bool{}
	for k := range extra {
		over[norm(k)] = true
	}
	out := make([]string, 0, len(base)+len(extra))
	for _, kv := range base {
		k := kv
		if i := strings.Index(kv, "="); i > 0 {
			k = kv[:i]
		}
		if !over[norm(k)] {
			out = append(out, kv)
		}
	}
	keys := make([]string, 0, len(extra))
	seen := map[string]bool{}
	for k := range extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if fold && seen[norm(k)] {
			continue // http_proxy and HTTP_PROXY are one variable on Windows
		}
		seen[norm(k)] = true
		out = append(out, k+"="+extra[k])
	}
	return out
}

// winEnvBlock builds a CreateProcess UNICODE environment block: entries
// sorted case-insensitively by name (as Windows expects), each NUL-terminated,
// the block ending with an extra NUL. Entries without '=' are dropped;
// later duplicates (case-insensitive) win.
func winEnvBlock(env []string) []uint16 {
	idx := map[string]int{}
	var kept []string
	var names []string
	for _, kv := range env {
		n, ok := envName(kv)
		if !ok || strings.ContainsRune(kv, 0) {
			continue
		}
		k := strings.ToUpper(n)
		if j, dup := idx[k]; dup {
			kept[j] = kv
			continue
		}
		idx[k] = len(kept)
		kept = append(kept, kv)
		names = append(names, k)
	}
	order := make([]int, len(kept))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return names[order[a]] < names[order[b]] })
	var out []uint16
	for _, i := range order {
		out = append(out, utf16.Encode([]rune(kept[i]))...)
		out = append(out, 0)
	}
	if len(out) == 0 {
		out = append(out, 0)
	}
	return append(out, 0)
}

// envName is the variable name of a "NAME=value" entry. Names may start with
// '=' (cmd's per-drive "=C:=C:\dir" entries), so the separator is searched
// from the second character.
func envName(kv string) (string, bool) {
	if len(kv) < 2 {
		return "", false
	}
	i := strings.IndexByte(kv[1:], '=')
	if i < 0 {
		return "", false
	}
	return kv[:i+1], true
}

// resolveTerminalDir checks a typed working directory; "" = the home folder.
func resolveTerminalDir(dir string) (string, error) {
	dir = strings.Trim(strings.TrimSpace(dir), "\"")
	if dir == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("无法确定用户目录")
		}
		return h, nil
	}
	if strings.HasPrefix(dir, "~") {
		if h, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(h, strings.TrimPrefix(strings.TrimPrefix(dir, "~"), string(filepath.Separator)))
		}
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("路径无效：%s", dir)
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("目录不存在：%s", abs)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("不是文件夹：%s", abs)
	}
	return abs, nil
}

// ---------- command line ----------

// psQuote single-quotes a PowerShell string.
func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// terminalScript is the PowerShell run in the new window: window title, the
// environment (repeated here so Windows Terminal tabs, which may not inherit
// our process env, get it too) and the CLI itself.
func terminalScript(title string, env map[string]string, bin string) string {
	var b strings.Builder
	b.WriteString("$Host.UI.RawUI.WindowTitle = " + psQuote(title) + "\n")
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "${env:%s} = %s\n", k, psQuote(env[k]))
	}
	if bin != "" {
		b.WriteString("& " + psQuote(bin) + "\n")
	}
	return b.String()
}

// psEncode is the -EncodedCommand form (base64 of UTF-16LE): no quoting
// issues with cmd, wt's ';' separator or non-ASCII paths.
func psEncode(script string) string {
	u := utf16.Encode([]rune(script))
	b := make([]byte, len(u)*2)
	for i, c := range u {
		b[2*i], b[2*i+1] = byte(c), byte(c>>8)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// windowsLaunchArgs builds the program and arguments for a new console:
// host "wt" uses Windows Terminal (wtPath), otherwise powershell.exe in its own
// console window (CREATE_NEW_CONSOLE is set by the windows launcher).
func windowsLaunchArgs(host, wtPath, dir, script string) (string, []string) {
	ps := []string{"powershell.exe", "-NoLogo", "-NoExit", "-ExecutionPolicy", "Bypass", "-EncodedCommand", psEncode(script)}
	if host == "wt" && wtPath != "" {
		return wtPath, append([]string{"-w", "new", "new-tab", "--title", "GhostBrowser", "-d", dir}, ps...)
	}
	return ps[0], ps[1:]
}

// winQuote quotes one argument the way CommandLineToArgvW reads it
// (same rules as syscall.EscapeArg), for the command line shown in logs/tests.
func winQuote(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\"") {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	slashes := 0
	for _, c := range s {
		switch c {
		case '\\':
			slashes++
		case '"':
			b.WriteString(strings.Repeat(`\`, slashes*2+1))
			slashes = 0
		default:
			slashes = 0
		}
		if c != '\\' {
			b.WriteRune(c)
		} else {
			b.WriteByte('\\')
		}
	}
	b.WriteString(strings.Repeat(`\`, slashes))
	b.WriteByte('"')
	return b.String()
}

func winCommandLine(exe string, args []string) string {
	parts := []string{winQuote(exe)}
	for _, a := range args {
		parts = append(parts, winQuote(a))
	}
	return strings.Join(parts, " ")
}

// ---------- sessions ----------

// TerminalSession is one terminal window GhostBrowser opened.
type TerminalSession struct {
	ID          string `json:"id"`
	Tool        string `json:"tool"`
	Name        string `json:"name"`
	Dir         string `json:"dir"`
	ProfileID   string `json:"profileId"`
	ProfileName string `json:"profileName"`
	TZ          string `json:"tz"`
	Lang        string `json:"lang"`
	Proxy       string `json:"proxy"` // display only
	Relay       string `json:"relay"` // local relay address when one is used
	Host        string `json:"host"`
	PID         int    `json:"pid"`
	Tracked     bool   `json:"tracked"` // false: Windows Terminal owns the window; closing only drops the relay
	Started     int64  `json:"started"`
}

// termProc is a started terminal window process. On Windows it is created
// with CreateProcess directly (terminal_windows.go), elsewhere it wraps an
// exec.Cmd. wait blocks until the process exits and returns its exit code;
// it is nil when the window is owned by another process (Windows Terminal).
type termProc struct {
	pid  int
	wait func() (int, error)
}

// cmdProc wraps a started exec.Cmd.
func cmdProc(cmd *exec.Cmd) *termProc {
	return &termProc{pid: cmd.Process.Pid, wait: func() (int, error) {
		err := cmd.Wait()
		code := -1
		if cmd.ProcessState != nil {
			code = cmd.ProcessState.ExitCode()
		}
		return code, err
	}}
}

// TerminalExit reports a terminal window that closed right after launch
// (event "terminal:failed"), so the UI can say so instead of the window just vanishing.
type TerminalExit struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Code    int    `json:"code"`
	Millis  int64  `json:"millis"`
	Message string `json:"message"`
}

// earlyExitWindow: a tracked terminal that ends sooner than this after launch
// is reported as a failed launch.
const earlyExitWindow = 3 * time.Second

func earlyExitMessage(name string, code int, d time.Duration) string {
	return fmt.Sprintf("%s 终端窗口启动后 %.1f 秒就退出了（退出码 %d）。请在 PowerShell 中手动运行该命令查看报错，或改用 Windows Terminal 打开。", name, d.Seconds(), code)
}

type termEntry struct {
	s     TerminalSession
	proc  *termProc
	relay *undetectable.Relay
}

type termRegistry struct {
	mu      sync.Mutex
	m       map[string]*termEntry
	onEvent func()
	onFail  func(TerminalExit)
}

var terms = &termRegistry{m: map[string]*termEntry{}}

func (r *termRegistry) list() []TerminalSession {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]TerminalSession, 0, len(r.m))
	for _, e := range r.m {
		out = append(out, e.s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Started > out[j].Started })
	return out
}

func (r *termRegistry) notify() {
	if r.onEvent != nil {
		r.onEvent()
	}
}

func (r *termRegistry) drop(id string) *termEntry {
	r.mu.Lock()
	e := r.m[id]
	delete(r.m, id)
	r.mu.Unlock()
	if e != nil {
		e.relay.Close()
	}
	return e
}

func (r *termRegistry) close(id string) error {
	r.mu.Lock()
	e := r.m[id]
	r.mu.Unlock()
	if e == nil {
		return fmt.Errorf("终端已关闭")
	}
	if e.s.Tracked && e.proc != nil && e.proc.pid > 0 {
		killTree(e.proc.pid)
	}
	r.drop(id)
	r.notify()
	return nil
}

// watch waits for a tracked window to close, then drops its entry (and relay).
// A window gone within earlyExitWindow — and not closed by us — is reported.
func (r *termRegistry) watch(sess TerminalSession, proc *termProc, started time.Time) {
	code, _ := proc.wait()
	d := time.Since(started)
	if r.drop(sess.ID) == nil {
		return // closed from GhostBrowser
	}
	r.notify()
	if d < earlyExitWindow && r.onFail != nil {
		r.onFail(TerminalExit{ID: sess.ID, Name: sess.Name, Code: code, Millis: d.Milliseconds(), Message: earlyExitMessage(sess.Name, code, d)})
	}
}

func (r *termRegistry) closeAll() {
	for _, s := range r.list() {
		_ = r.close(s.ID)
	}
}

// buildTerminalEnv assembles the env overlay for a request.
// It returns the locale/proxy overlay, the proxy plan and display values.
func (s *Store) buildTerminalEnv(req TerminalRequest) (env map[string]string, plan terminalProxyPlan, sess TerminalSession, err error) {
	env = map[string]string{}
	switch req.EnvMode {
	case "profile":
		p, ok := s.GetProfile(req.ProfileID)
		if !ok {
			return nil, plan, sess, fmt.Errorf("所选环境不存在")
		}
		var px *Proxy
		if p.InlineProxy != nil {
			px = p.InlineProxy
		} else if p.ProxyID != "" {
			if pr, ok := s.GetProxy(p.ProxyID); ok {
				px = pr
			}
		}
		fp := p.Fingerprint
		if needsGeo(fp) {
			if m, e := s.matchFP(fp, px); e == nil {
				fp = m
			} else if px != nil && px.Timezone != "" && fp.TimezoneFromIP {
				fp.Timezone = px.Timezone // last checked exit
			}
		}
		for k, v := range terminalLocaleEnv(fp.Timezone, firstLang(fp.Languages)) {
			env[k] = v
		}
		plan = planTerminalProxy(px)
		sess.ProfileID, sess.ProfileName = p.ID, p.Name
		sess.TZ, sess.Lang = fp.Timezone, firstLang(fp.Languages)
	case "custom":
		for k, v := range terminalLocaleEnv(req.TZ, req.Lang) {
			env[k] = v
		}
		sess.TZ, sess.Lang = strings.TrimSpace(req.TZ), strings.TrimSpace(req.Lang)
	}
	if sess.TZ != "" {
		if _, e := time.LoadLocation(sess.TZ); e != nil {
			return nil, plan, sess, fmt.Errorf("时区无效：%s（请使用 IANA 名称，如 Asia/Shanghai）", sess.TZ)
		}
	}
	sess.Proxy = plan.Display
	return env, plan, sess, nil
}

func (s *Store) launchTerminal(req TerminalRequest) (TerminalSession, error) {
	def, ok := toolDef(req.Tool)
	if !ok {
		return TerminalSession{}, fmt.Errorf("未知的工具：%s", req.Tool)
	}
	dir, err := resolveTerminalDir(req.Dir)
	if err != nil {
		return TerminalSession{}, err
	}
	env, plan, sess, err := s.buildTerminalEnv(req)
	if err != nil {
		return TerminalSession{}, err
	}
	bin := ""
	if def.id != "shell" {
		if lookPathIn(def.bin, os.Getenv("PATH")) == "" {
			return TerminalSession{}, fmt.Errorf("没有找到 %s：请先安装（%s）", def.name, def.install)
		}
		bin = def.bin
	}
	var relay *undetectable.Relay
	proxyURL := plan.Direct
	if plan.Upstream != "" {
		if relay, err = undetectable.StartRelay(plan.Upstream); err != nil {
			return TerminalSession{}, fmt.Errorf("代理中转启动失败：%w", err)
		}
		proxyURL = relay.URL()
		sess.Relay = relay.Addr
	}
	for k, v := range terminalProxyEnv(proxyURL) {
		env[k] = v
	}
	title := "GhostBrowser · " + def.name
	if sess.ProfileName != "" {
		title += " · " + sess.ProfileName
	}
	host := req.Host
	if host != "wt" {
		host = "console"
	}
	proc, tracked, err := startTerminalWindow(host, dir, title, env, bin)
	if err != nil {
		relay.Close()
		return TerminalSession{}, fmt.Errorf("终端窗口启动失败：%w", err)
	}
	started := time.Now()
	sess.ID = randHex(6)
	sess.Tool, sess.Name, sess.Dir, sess.Host, sess.Tracked = def.id, def.name, dir, host, tracked
	sess.Started = started.Unix()
	if proc != nil {
		sess.PID = proc.pid
	}
	e := &termEntry{s: sess, proc: proc, relay: relay}
	terms.mu.Lock()
	terms.m[sess.ID] = e
	terms.mu.Unlock()
	if tracked && proc != nil && proc.wait != nil {
		go terms.watch(sess, proc, started)
	}
	return sess, nil
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = crand.Read(b)
	return fmt.Sprintf("%x", b)
}
func hostOS() string { return runtime.GOOS }
