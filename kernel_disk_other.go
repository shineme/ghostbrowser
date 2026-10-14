//go:build !windows

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

// diskFree reports free bytes for the volume holding dir.
func diskFree(dir string) (int64, bool) {
	_ = os.MkdirAll(dir, 0o755)
	var st unix.Statfs_t
	if unix.Statfs(dir, &st) != nil {
		return 0, false
	}
	return int64(st.Bavail) * int64(st.Bsize), true
}
