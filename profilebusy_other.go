//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// profileDirHeld (Linux/macOS): Chromium's SingletonLock is a symlink to
// "<host>-<pid>", Firefox's "lock" one to "<ip>:+<pid>"; the directory is held
// while that process is alive on this machine.
func profileDirHeld(dir string) bool {
	host, _ := os.Hostname()
	for _, n := range []string{"SingletonLock", "lock"} {
		t, err := os.Readlink(filepath.Join(dir, n))
		if err != nil {
			continue
		}
		i := strings.LastIndexAny(t, "-+")
		if i < 0 {
			continue
		}
		pid, err := strconv.Atoi(t[i+1:])
		if err != nil || pid <= 0 {
			continue
		}
		if n == "SingletonLock" && host != "" && t[:i] != host {
			continue // held on another machine (shared folder)
		}
		if pidAlive(pid) {
			return true
		}
	}
	return false
}

func pidAlive(pid int) bool {
	if b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat"); err == nil {
		if k := strings.LastIndexByte(string(b), ')'); k > 0 && k+2 < len(b) && (b[k+2] == 'Z' || b[k+2] == 'X') {
			return false // zombie: exited, not reaped yet
		}
	}
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// leftoverPIDs lists processes whose command line runs a browser on dir.
func leftoverPIDs(dir string) []int {
	ents, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var out []int
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid == os.Getpid() {
			continue
		}
		b, err := os.ReadFile("/proc/" + e.Name() + "/cmdline")
		if err != nil || len(b) == 0 {
			continue
		}
		cmd := strings.ReplaceAll(string(b), "\x00", " ")
		if browserCmdline(cmd) && cmdlineHasDir(cmd, dir) && pidAlive(pid) {
			out = append(out, pid)
		}
	}
	return out
}
