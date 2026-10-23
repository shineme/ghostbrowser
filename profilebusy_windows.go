//go:build windows

package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

// profileDirHeld (Windows): Chromium keeps "<user data dir>\lockfile" open
// with read-only sharing (and Firefox "parent.lock" with none) for as long as
// any browser process runs on the directory; opening it for writing then
// fails with a sharing violation. A leftover file that opens is stale.
func profileDirHeld(dir string) bool {
	for _, n := range []string{"lockfile", "parent.lock"} {
		p := filepath.Join(dir, n)
		if _, err := os.Lstat(p); err != nil {
			continue
		}
		f, err := os.OpenFile(p, os.O_RDWR, 0)
		if err == nil {
			f.Close()
			continue
		}
		if errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return true
		}
	}
	return false
}

// leftoverPIDs lists browser processes whose command line runs on dir
// (chrome.exe of every kernel and its crashpad handler, firefox.exe).
func leftoverPIDs(dir string) []int {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	ps := `$ErrorActionPreference='SilentlyContinue'; ConvertTo-Json -Compress -InputObject @(Get-CimInstance Win32_Process | Where-Object { $_.CommandLine } | Select-Object ProcessId,CommandLine)`
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	hideConsole(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var rows []struct {
		ProcessId   int
		CommandLine string
	}
	if json.Unmarshal([]byte(strings.TrimSpace(string(out))), &rows) != nil {
		return nil
	}
	var pids []int
	for _, r := range rows {
		if r.ProcessId > 0 && r.ProcessId != os.Getpid() && browserCmdline(r.CommandLine) && cmdlineHasDir(r.CommandLine, dir) {
			pids = append(pids, r.ProcessId)
		}
	}
	return pids
}
