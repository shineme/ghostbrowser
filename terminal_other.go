//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func hideConsole(cmd *exec.Cmd) {}

func killTree(pid int) {
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
	}
}

func hasWindowsTerminal() bool { return false }

// startTerminalWindow (Linux): opens the first terminal emulator found and
// runs the CLI, then the user's shell. macOS is not supported yet.
func startTerminalWindow(host, dir, title string, env map[string]string, bin string) (*exec.Cmd, bool, error) {
	if runtime.GOOS == "darwin" {
		return nil, false, fmt.Errorf("终端启动器目前只支持 Windows 和 Linux")
	}
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "sh"
	}
	script := "exec " + sh
	if bin != "" {
		script = bin + "; exec " + sh
	}
	var cmd *exec.Cmd
	for _, t := range []string{"x-terminal-emulator", "gnome-terminal", "konsole", "xfce4-terminal", "xterm"} {
		p, err := exec.LookPath(t)
		if err != nil {
			continue
		}
		if t == "gnome-terminal" {
			cmd = exec.Command(p, "--title="+title, "--", "sh", "-c", script)
		} else {
			cmd = exec.Command(p, "-T", title, "-e", "sh", "-c", script)
		}
		break
	}
	if cmd == nil {
		return nil, false, fmt.Errorf("没有找到终端程序（x-terminal-emulator / gnome-terminal / konsole / xterm）")
	}
	cmd.Dir = dir
	cmd.Env = mergeEnv(os.Environ(), env, false)
	if err := cmd.Start(); err != nil {
		return nil, false, err
	}
	return cmd, !strings.Contains(cmd.Path, "gnome-terminal"), nil
}
