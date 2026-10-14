//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	shcore               = syscall.NewLazyDLL("shcore.dll")
	procSPI              = user32.NewProc("SystemParametersInfoW")
	procDpiForSystem     = user32.NewProc("GetDpiForSystem")
	procMonitorFromPoint = user32.NewProc("MonitorFromPoint")
	procDpiForMonitor    = shcore.NewProc("GetDpiForMonitor")
)

// primaryDPI is the primary monitor's current effective DPI. GetDpiForSystem
// alone is the DPI the session started with, which is stale after the user
// changes the scale without signing out; GetDpiForMonitor (Windows 8.1+)
// reports the live value. Both return 96 for a DPI-unaware caller, whose
// SPI_GETWORKAREA result is already virtualised to 96 DPI, so the conversion
// stays correct either way.
func primaryDPI() uintptr {
	if procMonitorFromPoint.Find() == nil && procDpiForMonitor.Find() == nil {
		const monitorDefaultToPrimary = 1
		const mdtEffectiveDPI = 0
		// MonitorFromPoint takes POINT{0,0} by value: one packed register on
		// 64-bit, two stack slots on 32-bit.
		var mon uintptr
		if unsafe.Sizeof(uintptr(0)) == 8 {
			mon, _, _ = procMonitorFromPoint.Call(0, monitorDefaultToPrimary)
		} else {
			mon, _, _ = procMonitorFromPoint.Call(0, 0, monitorDefaultToPrimary)
		}
		if mon != 0 {
			var dx, dy uint32
			if hr, _, _ := procDpiForMonitor.Call(mon, mdtEffectiveDPI, uintptr(unsafe.Pointer(&dx)), uintptr(unsafe.Pointer(&dy))); hr == 0 && dx >= 96 {
				return uintptr(dx)
			}
		}
	}
	if procDpiForSystem.Find() == nil {
		if d, _, _ := procDpiForSystem.Call(); d >= 96 {
			return d
		}
	}
	return 96
}

// osWorkArea returns the primary monitor's work area (taskbar excluded),
// converted from physical pixels to the DIPs Chromium's window API uses.
func osWorkArea() (Rect, bool) {
	var r struct{ L, T, R, B int32 }
	const spiGetWorkArea = 0x0030
	if ok, _, _ := procSPI.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&r)), 0); ok == 0 {
		return Rect{}, false
	}
	dpi := primaryDPI()
	sc := func(v int32) int { return int(int64(v) * 96 / int64(dpi)) }
	return Rect{sc(r.L), sc(r.T), sc(r.R) - sc(r.L), sc(r.B) - sc(r.T)}, true
}
