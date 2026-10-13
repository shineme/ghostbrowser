//go:build !windows

package undetectable

import (
	"fmt"
	"os"
)

func lockProfile(_ string) (*os.File, error) {
	return nil, fmt.Errorf("this bundled native Chromium requires Windows")
}
