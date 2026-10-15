//go:build windows

package main

import (
	"os"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// winQuote must match what CreateProcess callers (x/sys ComposeCommandLine,
// syscall.EscapeArg) produce, so the logged command line is the real one.
func TestTerminalComposeCommandLine(t *testing.T) {
	exe, args := windowsLaunchArgs("console", "", `C:\Users\王 五\`, terminalScript("t", map[string]string{"TZ": "Asia/Tokyo"}, "codex"))
	if got, want := winCommandLine(exe, args), windows.ComposeCommandLine(append([]string{exe}, args...)); got != want {
		t.Fatalf("got %s\nwant %s", got, want)
	}
	for _, a := range []string{`a"b`, `C:\dir with space\`, "", "plain", `\\server\share\`} {
		if winQuote(a) != syscall.EscapeArg(a) {
			t.Errorf("winQuote(%q)=%s EscapeArg=%s", a, winQuote(a), syscall.EscapeArg(a))
		}
	}
}

// The console process must stay alive with its own console (no NUL stdin).
func TestTerminalConsoleProcessStays(t *testing.T) {
	p, err := startConsoleProcess("cmd.exe", []string{"/c", "echo ok & ping -n 3 127.0.0.1 >nul"}, "", "gb-test", mergeEnv(os.Environ(), map[string]string{"GB_T": "1"}, true))
	if err != nil {
		t.Fatal(err)
	}
	t0 := time.Now()
	code, err := p.wait()
	if err != nil || code != 0 || time.Since(t0) < time.Second {
		t.Fatalf("code=%d err=%v after %v", code, err, time.Since(t0))
	}
}
