package main

import (
	"encoding/base64"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"ghostbrowser/internal/undetectable"
)

func TestTerminalLocaleEnv(t *testing.T) {
	cases := map[string]string{"zh-CN": "zh_CN.UTF-8", "en-US": "en_US.UTF-8", "pt_BR": "pt_BR.UTF-8", "ja": "ja.UTF-8", "zh-Hant": "zh_TW.UTF-8", "sr-Latn-RS": "sr_RS.UTF-8", "": ""}
	for in, want := range cases {
		if got := posixLocale(in); got != want {
			t.Errorf("posixLocale(%q) = %q, want %q", in, got, want)
		}
	}
	env := terminalLocaleEnv("Asia/Tokyo", "ja-JP")
	if env["TZ"] != "Asia/Tokyo" || env["LANG"] != "ja_JP.UTF-8" || env["LC_ALL"] != "ja_JP.UTF-8" || env["LANGUAGE"] != "ja_JP" {
		t.Fatalf("env %v", env)
	}
	if e := terminalLocaleEnv("", ""); len(e) != 0 {
		t.Fatalf("empty should add nothing: %v", e)
	}
}

func TestTerminalProxyPlan(t *testing.T) {
	if p := planTerminalProxy(nil); p.Direct != "" || p.Upstream != "" {
		t.Fatal("nil proxy")
	}
	if p := planTerminalProxy(&Proxy{Type: "none", Host: "x", Port: 1}); p.Direct != "" || p.Upstream != "" {
		t.Fatal("type none")
	}
	p := planTerminalProxy(&Proxy{Type: "http", Host: "1.2.3.4", Port: 8080})
	if p.Direct != "http://1.2.3.4:8080" || p.Upstream != "" {
		t.Fatalf("plain http %+v", p)
	}
	p = planTerminalProxy(&Proxy{Type: "http", Host: "1.2.3.4", Port: 8080, User: "u", Pass: "p@ss"})
	if p.Direct != "" || !strings.HasPrefix(p.Upstream, "http://u:p%40ss@1.2.3.4:8080") || strings.Contains(p.Display, "p@ss") {
		t.Fatalf("auth http %+v", p)
	}
	p = planTerminalProxy(&Proxy{Type: "socks5", Host: "5.6.7.8", Port: 1080})
	if p.Direct != "" || p.Upstream != "socks5://5.6.7.8:1080" || p.Display != "SOCKS5 5.6.7.8:1080" {
		t.Fatalf("socks %+v", p)
	}
	env := terminalProxyEnv("http://127.0.0.1:5555")
	for _, k := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		if env[k] != "http://127.0.0.1:5555" {
			t.Errorf("%s = %q", k, env[k])
		}
	}
	if !strings.Contains(env["NO_PROXY"], "127.0.0.1") {
		t.Error("NO_PROXY")
	}
}

// A credentialed proxy reaches the CLI as a local relay URL that forwards
// with the credentials.
func TestTerminalRelayForAuthProxy(t *testing.T) {
	// Upstream HTTP proxy that requires auth and answers CONNECT.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	gotAuth := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 4096)
		n, _ := c.Read(buf)
		req := string(buf[:n])
		for _, l := range strings.Split(req, "\r\n") {
			if strings.HasPrefix(strings.ToLower(l), "proxy-authorization:") {
				gotAuth <- strings.TrimSpace(l[len("proxy-authorization:"):])
			}
		}
		c.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	}()
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	px := &Proxy{Type: "http", Host: "127.0.0.1", User: "alice", Pass: "s3cret"}
	px.Port = atoiT(port)
	plan := planTerminalProxy(px)
	if plan.Upstream == "" {
		t.Fatal("auth proxy should use the relay")
	}
	r, err := undetectable.StartRelay(plan.Upstream)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	env := terminalProxyEnv(r.URL())
	if !strings.HasPrefix(env["HTTPS_PROXY"], "http://127.0.0.1:") || strings.Contains(env["HTTPS_PROXY"], "alice") {
		t.Fatalf("relay url %q", env["HTTPS_PROXY"])
	}
	c, err := net.Dial("tcp", r.Addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Write([]byte("CONNECT example.com:443 HTTP/1.1\r\nHost: example.com:443\r\n\r\n"))
	b := make([]byte, 256)
	n, _ := c.Read(b)
	if !strings.Contains(string(b[:n]), "200") {
		t.Fatalf("relay answer %q", b[:n])
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:s3cret"))
	if a := <-gotAuth; a != want {
		t.Fatalf("upstream auth %q want %q", a, want)
	}
}

func atoiT(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

func TestTerminalEnvFromProfile(t *testing.T) {
	s := testStore(t)
	s.PutProxy(Proxy{ID: "px1", Type: "socks5", Host: "9.9.9.9", Port: 1080, User: "u", Pass: "p"})
	fp := GenerateFingerprint("windows", "141")
	fp.Timezone, fp.Languages, fp.TimezoneFromIP, fp.LanguageFromIP, fp.GeoFromIP = "Europe/Berlin", []string{"de-DE", "de"}, false, false, false
	s.PutProfile(Profile{ID: "p1", Name: "德国店铺", ProxyID: "px1", Fingerprint: fp})
	env, plan, sess, err := s.buildTerminalEnv(TerminalRequest{EnvMode: "profile", ProfileID: "p1"})
	if err != nil {
		t.Fatal(err)
	}
	if env["TZ"] != "Europe/Berlin" || env["LANG"] != "de_DE.UTF-8" || env["LC_ALL"] != "de_DE.UTF-8" {
		t.Fatalf("env %v", env)
	}
	if plan.Upstream != "socks5://u:p@9.9.9.9:1080" || sess.Proxy != "SOCKS5 9.9.9.9:1080" || sess.ProfileName != "德国店铺" {
		t.Fatalf("plan %+v sess %+v", plan, sess)
	}
	env, plan, _, err = s.buildTerminalEnv(TerminalRequest{EnvMode: "custom", TZ: "Asia/Shanghai", Lang: "zh-CN"})
	if err != nil || env["TZ"] != "Asia/Shanghai" || env["LANG"] != "zh_CN.UTF-8" || plan.Upstream != "" {
		t.Fatalf("custom %v %+v %v", env, plan, err)
	}
	if _, _, _, err = s.buildTerminalEnv(TerminalRequest{EnvMode: "custom", TZ: "Mars/Base"}); err == nil {
		t.Fatal("invalid TZ accepted")
	}
	if _, _, _, err = s.buildTerminalEnv(TerminalRequest{EnvMode: "profile", ProfileID: "nope"}); err == nil {
		t.Fatal("missing profile accepted")
	}
	env, _, _, _ = s.buildTerminalEnv(TerminalRequest{EnvMode: "none"})
	if len(env) != 0 {
		t.Fatalf("none %v", env)
	}
}

func TestTerminalMergeEnv(t *testing.T) {
	base := []string{"PATH=/bin", "http_proxy=http://old", "TZ=UTC", "HOME=/h"}
	got := mergeEnv(base, map[string]string{"HTTP_PROXY": "http://new", "http_proxy": "http://new", "TZ": "Asia/Tokyo"}, true)
	j := strings.Join(got, "|")
	if strings.Contains(j, "old") || strings.Contains(j, "TZ=UTC") || !strings.Contains(j, "TZ=Asia/Tokyo") || !strings.Contains(j, "PATH=/bin") {
		t.Fatalf("fold merge %v", got)
	}
	if strings.Count(strings.ToUpper(j), "HTTP_PROXY=") != 1 {
		t.Fatalf("windows env should hold one HTTP_PROXY: %v", got)
	}
	got = mergeEnv(base, map[string]string{"HTTP_PROXY": "http://new"}, false)
	if !strings.Contains(strings.Join(got, "|"), "http_proxy=http://old") {
		t.Fatalf("case-sensitive merge dropped lowercase: %v", got)
	}
}

func TestTerminalDetectFakePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts")
	}
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write("claude", "echo '2.0.14 (Claude Code)'")
	write("codex", "echo 'codex-cli 0.46.0'")
	// gemini missing; a non-executable file must not count
	os.WriteFile(filepath.Join(dir, "gemini"), []byte("x"), 0o644)
	t.Setenv("SHELL", "/bin/sh")
	tools := detectTools(dir + string(os.PathListSeparator) + "/bin")
	by := map[string]TerminalTool{}
	for _, x := range tools {
		by[x.ID] = x
	}
	if !by["claude"].Installed || by["claude"].Version != "2.0.14" {
		t.Errorf("claude %+v", by["claude"])
	}
	if !by["codex"].Installed || by["codex"].Version != "0.46.0" {
		t.Errorf("codex %+v", by["codex"])
	}
	if by["gemini"].Installed || by["gemini"].InstallURL == "" || by["gemini"].InstallCmd == "" {
		t.Errorf("gemini %+v", by["gemini"])
	}
	if _, ok := by["shell"]; !ok {
		t.Error("shell card missing")
	}
	if len(tools) != 4 {
		t.Errorf("want 4 tools, got %d", len(tools))
	}
}

func TestTerminalDir(t *testing.T) {
	d := t.TempDir()
	if got, err := resolveTerminalDir(" \"" + d + "\" "); err != nil || got != d {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := resolveTerminalDir(filepath.Join(d, "missing")); err == nil || !strings.Contains(err.Error(), "目录不存在") {
		t.Fatalf("missing: %v", err)
	}
	f := filepath.Join(d, "f.txt")
	os.WriteFile(f, []byte("x"), 0o644)
	if _, err := resolveTerminalDir(f); err == nil {
		t.Fatal("file accepted")
	}
	h, _ := os.UserHomeDir()
	if got, _ := resolveTerminalDir(""); got != h {
		t.Fatalf("empty = %q", got)
	}
}

func decodePS(t *testing.T, b64 string) string {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatal(err)
	}
	u := make([]uint16, len(raw)/2)
	for i := range u {
		u[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
	}
	return string(utf16.Decode(u))
}

func TestTerminalWindowsCommandLine(t *testing.T) {
	env := map[string]string{"TZ": "Asia/Tokyo", "LANG": "ja_JP.UTF-8", "HTTPS_PROXY": "http://127.0.0.1:5555"}
	script := terminalScript("GhostBrowser · Claude Code · 小王's 店", env, "claude")
	for _, want := range []string{
		"$Host.UI.RawUI.WindowTitle = 'GhostBrowser · Claude Code · 小王''s 店'",
		"${env:TZ} = 'Asia/Tokyo'", "${env:HTTPS_PROXY} = 'http://127.0.0.1:5555'", "& 'claude'",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script lacks %q:\n%s", want, script)
		}
	}
	if strings.Contains(terminalScript("t", nil, ""), "& ") {
		t.Error("plain shell should not run a command")
	}
	dir := `C:\Users\王 五\code\`
	exe, args := windowsLaunchArgs("console", "", dir, script)
	if exe != "powershell.exe" || args[0] != "-NoLogo" || args[1] != "-NoExit" || args[len(args)-2] != "-EncodedCommand" {
		t.Fatalf("console %s %q", exe, args)
	}
	if decodePS(t, args[len(args)-1]) != script {
		t.Fatal("encoded command does not round-trip")
	}
	exe, args = windowsLaunchArgs("wt", `C:\WT\wt.exe`, dir, script)
	cl := winCommandLine(exe, args)
	if exe != `C:\WT\wt.exe` || !strings.Contains(cl, `-d "C:\Users\王 五\code\\" powershell.exe -NoLogo -NoExit`) || strings.Contains(cl, ";") {
		t.Fatalf("wt cmdline %s", cl)
	}
	// wt asked for but not installed: falls back to a console window
	if exe, _ = windowsLaunchArgs("wt", "", dir, script); exe != "powershell.exe" {
		t.Fatal("wt fallback")
	}
	if winQuote(`a"b`) != `"a\"b"` || winQuote("") != `""` || winQuote("plain") != "plain" {
		t.Fatal("winQuote")
	}
}

func TestTerminalWinEnvBlock(t *testing.T) {
	env := []string{"Path=C:\\Windows", "=C:=C:\\work", "tz=old", "LANG=zh_CN.UTF-8", "TZ=Asia/Tokyo", "bogus", "ALL_PROXY=http://127.0.0.1:1", "名字=值"}
	b := winEnvBlock(env)
	if len(b) < 2 || b[len(b)-1] != 0 || b[len(b)-2] != 0 {
		t.Fatalf("block not double-NUL terminated: %v", b[max(0, len(b)-4):])
	}
	var got []string
	for _, part := range strings.Split(string(utf16.Decode(b[:len(b)-2])), "\x00") {
		got = append(got, part)
	}
	want := []string{"=C:=C:\\work", "ALL_PROXY=http://127.0.0.1:1", "LANG=zh_CN.UTF-8", "Path=C:\\Windows", "TZ=Asia/Tokyo", "名字=值"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("env block\n got %q\nwant %q", got, want)
	}
	// non-ASCII survives as UTF-16 (not bytes)
	if !strings.Contains(string(utf16.Decode(b)), "名字=值") {
		t.Fatal("utf-16")
	}
	if e := winEnvBlock(nil); len(e) != 2 || e[0] != 0 || e[1] != 0 {
		t.Fatalf("empty block %v", e)
	}
	// merged overlay ends up in the block once
	m := winEnvBlock(mergeEnv([]string{"HTTPS_PROXY=x", "Home=C:\\u"}, map[string]string{"https_proxy": "http://127.0.0.1:9", "HTTPS_PROXY": "http://127.0.0.1:9"}, true))
	if s := string(utf16.Decode(m)); strings.Count(strings.ToUpper(s), "HTTPS_PROXY=") != 1 || !strings.Contains(s, "127.0.0.1:9") {
		t.Fatalf("merged %q", s)
	}
}

func TestTerminalEarlyExitReported(t *testing.T) {
	r := &termRegistry{m: map[string]*termEntry{}}
	var got []TerminalExit
	r.onFail = func(x TerminalExit) { got = append(got, x) }
	sess := TerminalSession{ID: "a1", Name: "Codex CLI", Tracked: true}
	quick := &termProc{pid: 1, wait: func() (int, error) { return 1, nil }}
	r.m["a1"] = &termEntry{s: sess, proc: quick}
	r.watch(sess, quick, time.Now())
	if len(got) != 1 || got[0].Code != 1 || got[0].ID != "a1" || !strings.Contains(got[0].Message, "退出码 1") {
		t.Fatalf("early exit not reported: %+v", got)
	}
	if len(r.list()) != 0 {
		t.Fatal("entry not dropped")
	}
	// a long-lived window that closes later is not an error
	got = nil
	r.m["b2"] = &termEntry{s: TerminalSession{ID: "b2"}, proc: quick}
	r.watch(TerminalSession{ID: "b2"}, quick, time.Now().Add(-earlyExitWindow-time.Second))
	if len(got) != 0 {
		t.Fatal("late exit reported")
	}
	// closed from GhostBrowser (entry already dropped): silent
	r.watch(TerminalSession{ID: "gone"}, quick, time.Now())
	if len(got) != 0 {
		t.Fatal("user close reported")
	}
}
