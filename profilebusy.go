package main

// Refusing to upload a profile whose browser is still running. Chromium keeps
// writing caches, databases and memory-mapped files while it runs, so an
// upload taken from a live directory either fails ("文件 … 一直在变化") or
// stores torn files. Besides our own launcher sessions (runReg) this also
// catches a browser that outlived its window: a crashed launcher, a kernel
// left in the background, or a process GhostBrowser lost track of.

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

var errProfileRunning = errors.New("环境正在运行，请先关闭后再修复/上传")

const leftoverHint = "结束残留进程"

// leftoverError: the profile directory is held by a browser process that is
// not one of our running sessions. pids are the ones GhostBrowser can end.
type leftoverError struct{ pids []int }

func (e *leftoverError) Error() string {
	m := "检测到该环境仍有浏览器进程在后台运行（可能是残留进程），请在任务管理器结束后重试"
	if len(e.pids) > 0 {
		m += fmt.Sprintf("；找到 %d 个由 GhostBrowser 为这个环境启动的残留进程，可以点「%s」一键清理", len(e.pids), leftoverHint)
	}
	return m
}

// testHookProfileHeld replaces the process check in tests.
var testHookProfileHeld func(dir string) bool

func profileHeld(dir string) bool {
	if testHookProfileHeld != nil {
		return testHookProfileHeld(dir)
	}
	return profileDirHeld(dir)
}

// profileHeldError is nil when no browser process holds the directory.
func profileHeldError(dir string) error {
	if !profileHeld(dir) {
		return nil
	}
	return &leftoverError{pids: leftoverPIDs(dir)}
}

// profileBusy is the up-front check for 上传 / 修复云端.
func (s *Store) profileBusy(id string) error {
	if _, ok := runReg.get(id); ok {
		return errProfileRunning
	}
	_, dir, e := s.syncDir(id)
	if e != nil {
		return e
	}
	return profileHeldError(dir)
}

// waitProfileReleased waits up to max for the browser to let go of the
// directory (Chromium's helpers finish a moment after the main process).
func waitProfileReleased(dir string, max time.Duration) bool {
	deadline := time.Now().Add(max)
	for profileHeld(dir) {
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(500 * time.Millisecond)
	}
	return true
}

// KillLeftoverProcesses ends the browser processes GhostBrowser started for
// this profile that outlived its window (matched by the profile directory on
// their command line; never a running session, never another profile).
func (s *Store) KillLeftoverProcesses(id string) (int, error) {
	s.launchMu.Lock()
	defer s.launchMu.Unlock()
	if _, ok := runReg.get(id); ok {
		return 0, fmt.Errorf("环境正在运行，请先正常关闭它")
	}
	_, dir, e := s.syncDir(id)
	if e != nil {
		return 0, e
	}
	pids := leftoverPIDs(dir)
	for _, pid := range pids {
		if pid > 0 && pid != os.Getpid() {
			killTree(pid)
		}
	}
	if !waitProfileReleased(dir, 8*time.Second) {
		return len(pids), fmt.Errorf("已结束 %d 个进程，但环境目录仍被占用：请在任务管理器里结束剩下的浏览器进程（chrome.exe / firefox.exe）后重试", len(pids))
	}
	return len(pids), nil
}

// cmdlineHasDir reports whether a process command line names dir as a
// whole path (dir itself or something inside it), case-insensitively.
func cmdlineHasDir(cmd, dir string) bool {
	if dir == "" {
		return false
	}
	lc := strings.ToLower(strings.ReplaceAll(cmd, "/", `\`))
	ld := strings.TrimRight(strings.ToLower(strings.ReplaceAll(dir, "/", `\`)), `\`)
	for i := 0; ; {
		j := strings.Index(lc[i:], ld)
		if j < 0 {
			return false
		}
		end := i + j + len(ld)
		if end == len(lc) || strings.ContainsRune(`"' \`+"\t", rune(lc[end])) {
			return true
		}
		i = i + j + 1
	}
}

// browserCmdline: a command line that is a browser (or its crash handler)
// running on a user data directory.
func browserCmdline(cmd string) bool {
	l := strings.ToLower(cmd)
	return strings.Contains(l, "--user-data-dir") || strings.Contains(l, "--database=") || strings.Contains(l, "-profile")
}
