//go:build windows

package main

import (
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const createNoWindow = 0x08000000

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

// startTerminalWindow opens a new console. The console host ("console") is
// powershell.exe in its own window, created with CreateProcess directly (see
// startConsoleProcess) so GhostBrowser keeps the process and the proxy relay
// alive exactly as long as the window. Windows Terminal hands the tab to its
// own process, so that window is not tracked.
func startTerminalWindow(host, dir, title string, env map[string]string, bin string) (*termProc, bool, error) {
	wt := ""
	if host == "wt" {
		wt = wtPath()
	}
	exe, args := windowsLaunchArgs(host, wt, dir, terminalScript(title, env, bin))
	if wt != "" {
		cmd := exec.Command(exe, args...)
		cmd.Dir = dir
		cmd.Env = mergeEnv(os.Environ(), env, true)
		hideConsole(cmd)
		if err := cmd.Start(); err != nil {
			return nil, false, err
		}
		pid := cmd.Process.Pid
		go cmd.Wait() // wt.exe returns at once
		return &termProc{pid: pid}, false, nil
	}
	p, err := startConsoleProcess(exe, args, dir, title, mergeEnv(os.Environ(), env, true))
	return p, err == nil, err
}

// startConsoleProcess starts exe in a NEW console window without os/exec.
//
// os/exec cannot be used here: syscall.StartProcess always sets
// STARTF_USESTDHANDLES and passes the Cmd's stdin/stdout/stderr, which are
// the NUL device when left nil (os/exec childStdin/writerDescriptor). With
// CREATE_NEW_CONSOLE the child then uses those NUL handles instead of the new
// console: PowerShell reads EOF from stdin and exits at once even with
// -NoExit, and everything it prints goes to NUL — the window flashes and
// closes. Here no std handles are passed (no STARTF_USESTDHANDLES) and no
// handles are inherited, so the child attaches to its own console.
func startConsoleProcess(exe string, args []string, dir, title string, env []string) (*termProc, error) {
	full, err := exec.LookPath(exe)
	if err != nil {
		return nil, err
	}
	appName, err := windows.UTF16PtrFromString(full)
	if err != nil {
		return nil, err
	}
	cmdLine, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(append([]string{full}, args...)))
	if err != nil {
		return nil, err
	}
	var cwd *uint16
	if dir != "" {
		if cwd, err = windows.UTF16PtrFromString(dir); err != nil {
			return nil, err
		}
	}
	block := winEnvBlock(env)
	si := &windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{}))}
	if title != "" {
		if t, e := windows.UTF16PtrFromString(title); e == nil {
			si.Title = t
		}
	}
	pi := &windows.ProcessInformation{}
	flags := uint32(windows.CREATE_NEW_CONSOLE | windows.CREATE_UNICODE_ENVIRONMENT)
	if err := windows.CreateProcess(appName, cmdLine, nil, nil, false, flags, &block[0], cwd, si, pi); err != nil {
		return nil, err
	}
	windows.CloseHandle(pi.Thread)
	h := pi.Process
	var once sync.Once
	return &termProc{pid: int(pi.ProcessId), wait: func() (int, error) {
		code := -1
		var werr error
		once.Do(func() {
			defer windows.CloseHandle(h)
			if _, werr = windows.WaitForSingleObject(h, windows.INFINITE); werr != nil {
				return
			}
			var c uint32
			if werr = windows.GetExitCodeProcess(h, &c); werr == nil {
				code = int(c)
			}
		})
		return code, werr
	}}, nil
}

func hasWindowsTerminal() bool { return wtPath() != "" }
