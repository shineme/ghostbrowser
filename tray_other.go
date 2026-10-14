//go:build !windows

package main

// No tray outside Windows: closing the window quits as before.

func trayAvailable() bool { return false }
func startTray(trayHost)  {}
func stopTray()           {}
