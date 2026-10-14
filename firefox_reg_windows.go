//go:build windows

package main

import (
	"strings"

	"golang.org/x/sys/windows/registry"
)

// firefoxFromRegistry reads the App Paths entry Firefox's installer writes.
func firefoxFromRegistry() string {
	for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		k, err := registry.OpenKey(root, `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\firefox.exe`, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		v, _, err := k.GetStringValue("")
		k.Close()
		if v = strings.Trim(strings.TrimSpace(v), "\""); err == nil && v != "" && fileExists(v) {
			return v
		}
	}
	return ""
}
