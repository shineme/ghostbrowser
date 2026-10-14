//go:build !windows

package main

// osWorkArea has no native implementation off Windows; the Wails screen hook
// (or a 1920×1040 default) is used instead.
func osWorkArea() (Rect, bool) { return Rect{}, false }
