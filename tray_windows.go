//go:build windows

package main

// System tray (Windows): github.com/energye/systray runs its own hidden
// window and message loop on a locked OS thread, next to Wails' loop.

import (
	_ "embed"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/energye/systray"
)

//go:embed build/windows/icon.ico
var trayIconICO []byte

const traySlots = 10

var trayOnce sync.Once

func trayAvailable() bool { return true }

func startTray(h trayHost) {
	trayOnce.Do(func() {
		go func() {
			runtime.LockOSThread()
			systray.Run(func() { trayReady(h) }, func() {})
		}()
	})
}

func stopTray() { systray.Quit() }

func trayReady(h trayHost) {
	systray.SetIcon(trayIconICO)
	systray.SetTitle("GhostBrowser")
	systray.SetTooltip("GhostBrowser")
	systray.SetOnClick(func(systray.IMenu) { h.ShowWindow() })
	systray.SetOnDClick(func(systray.IMenu) { h.ShowWindow() })

	show := systray.AddMenuItem("显示主窗口", "")
	show.Click(h.ShowWindow)
	systray.AddSeparator()
	head := systray.AddMenuItem("没有运行中的环境", "")
	head.Disable()
	slots := make([]*systray.MenuItem, traySlots)
	ids := make([]string, traySlots)
	var mu sync.Mutex
	for i := range slots {
		i := i
		slots[i] = systray.AddMenuItem("", "切到前台")
		slots[i].Click(func() {
			mu.Lock()
			id := ids[i]
			mu.Unlock()
			if id != "" {
				h.FocusProfile(id)
			}
		})
		slots[i].Hide()
	}
	more := systray.AddMenuItem("", "")
	more.Disable()
	more.Hide()
	systray.AddSeparator()
	stopAll := systray.AddMenuItem("全部关闭", "关闭所有运行中的环境")
	stopAll.Click(h.StopAll)
	systray.AddSeparator()
	quit := systray.AddMenuItem("退出", "关闭所有环境并退出 GhostBrowser")
	quit.Click(h.QuitApp)

	last := "\x00"
	for {
		items := h.RunningItems()
		key := fmt.Sprint(items)
		if key != last {
			last = key
			if len(items) == 0 {
				head.SetTitle("没有运行中的环境")
				stopAll.Disable()
				systray.SetTooltip("GhostBrowser")
			} else {
				head.SetTitle(fmt.Sprintf("运行中的环境（%d）", len(items)))
				stopAll.Enable()
				systray.SetTooltip(fmt.Sprintf("GhostBrowser · 运行中 %d 个环境", len(items)))
			}
			mu.Lock()
			for i := range slots {
				if i < len(items) {
					ids[i] = items[i].ID
					slots[i].SetTitle("    " + trayClip(items[i].Name, 32))
					slots[i].Show()
				} else {
					ids[i] = ""
					slots[i].Hide()
				}
			}
			mu.Unlock()
			if len(items) > traySlots {
				more.SetTitle(fmt.Sprintf("    …还有 %d 个", len(items)-traySlots))
				more.Show()
			} else {
				more.Hide()
			}
		}
		time.Sleep(2 * time.Second)
	}
}
