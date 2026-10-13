//go:build windows

package undetectable

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
)

func lockProfile(dir string) (*os.File, error) {
	lockDir := filepath.Join(filepath.Dir(dir), ".native-locks")
	if err := os.MkdirAll(lockDir, 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(lockDir, filepath.Base(dir)+".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	var ov windows.Overlapped
	err = windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &ov)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("environment is already being launched by another process: %w", err)
	}
	return f, nil
}
