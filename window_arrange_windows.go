//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	procSPI          = user32.NewProc("SystemParametersInfoW")
	procDpiForSystem = user32.NewProc("GetDpiForSystem")
)

// osWorkArea returns the primary monitor's work area (taskbar excluded),
// converted from physical pixels to the DIPs Chromium's window API uses.
func osWorkArea() (Rect, bool) {
	var r struct{ L, T, R, B int32 }
	const spiGetWorkArea = 0x0030
	if ok, _, _ := procSPI.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&r)), 0); ok == 0 {
		return Rect{}, false
	}
	dpi := uintptr(96)
	if procDpiForSystem.Find() == nil {
		if d, _, _ := procDpiForSystem.Call(); d >= 96 {
			dpi = d
		}
	}
	sc := func(v int32) int { return int(int64(v) * 96 / int64(dpi)) }
	return Rect{sc(r.L), sc(r.T), sc(r.R) - sc(r.L), sc(r.B) - sc(r.T)}, true
}
