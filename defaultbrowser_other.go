//go:build !windows

package main

import "errors"

var errDefaultBrowserUnsupported = errors.New("只有 Windows 版可以设为系统默认浏览器")

func defaultBrowserSupported() bool            { return false }
func allowForegroundHandoff()                  {}
func registerDefaultBrowser(string) error      { return errDefaultBrowserUnsupported }
func unregisterDefaultBrowser() error          { return errDefaultBrowserUnsupported }
func defaultBrowserState() (bool, bool, error) { return false, false, errDefaultBrowserUnsupported }
func ensureDefaultBrowserRegistration() string { return "" }
func openDefaultAppsSettings() error           { return errDefaultBrowserUnsupported }
