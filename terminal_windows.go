//go:build windows

package main

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

const (
	createNewConsole = 0x00000010
	createNoWindow   = 0x08000000
)

// hideConsole keeps helper calls (version checks) from flashing a window.
func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

// killTree ends the console and everything started in it (the CLI, node, ...).
func killTree(pid int) {
	c := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid))
	hideConsole(c)
	if c.Run() != nil {
		if p, err := os.FindProcess(pid); err == nil {
			_ = p.Kill()
		}
	}
}

// wtPath is Windows Terminal's launcher when installed.
func wtPath() string {
	p, err := exec.LookPath("wt.exe")
	if err != nil {
		return ""
	}
	return p
}

// startTerminalWindow opens a new console. PowerShell gets its own console
// window (CREATE_NEW_CONSOLE) so GhostBrowser keeps the process and the proxy
// relay alive exactly as long as the window. Windows Terminal hands the tab
// to its own process, so that window is not tracked.
func startTerminalWindow(host, dir, title string, env map[string]string, bin string) (*exec.Cmd, bool, error) {
	wt := ""
	if host == "wt" {
		wt = wtPath()
	}
	exe, args := windowsLaunchArgs(host, wt, dir, terminalScript(title, env, bin))
	cmd := exec.Command(exe, args...)
	cmd.Dir = dir
	cmd.Env = mergeEnv(os.Environ(), env, true)
	if wt == "" {
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNewConsole}
	} else {
		hideConsole(cmd)
	}
	if err := cmd.Start(); err != nil {
		return nil, false, err
	}
	if wt != "" {
		go cmd.Wait() // wt.exe returns at once
		return cmd, false, nil
	}
	return cmd, true, nil
}

func hasWindowsTerminal() bool { return wtPath() != "" }
