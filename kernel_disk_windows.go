//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// diskFree reports free bytes for the volume holding dir.
func diskFree(dir string) (int64, bool) {
	_ = os.MkdirAll(dir, 0o755)
	p, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return 0, false
	}
	var avail, total, free uint64
	if windows.GetDiskFreeSpaceEx(p, &avail, &total, &free) != nil {
		return 0, false
	}
	return int64(avail), true
}
