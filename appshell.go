package main

// App shell: close-to-tray lifecycle and the tray's view of the app.

import (
	"context"
	"sort"
	"sync/atomic"
	"unicode/utf8"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type trayItem struct{ ID, Name string }

// trayHost is what the tray menu needs from the app.
type trayHost interface {
	ShowWindow()
	RunningItems() []trayItem
	FocusProfile(id string)
	StopAll()
	QuitApp()
}

// appQuitting is set by a real quit (tray 退出, update restart) so
// OnBeforeClose lets the window close instead of hiding it.
var appQuitting atomic.Bool

func trayClip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}

// shouldHideOnClose decides OnBeforeClose: hide to tray when the tray runs,
// the setting is on (default) and this is not a real quit.
func shouldHideOnClose(st Settings, tray, quitting bool) bool {
	return tray && !quitting && !st.CloseToTrayOff
}

// beforeClose is Wails' OnBeforeClose: returning true keeps the app running.
func (a *App) beforeClose(ctx context.Context) bool {
	if a.store == nil || !shouldHideOnClose(a.store.GetSettings(), trayAvailable(), appQuitting.Load()) {
		return false
	}
	wruntime.WindowHide(ctx)
	return true
}

type appTray struct{ a *App }

func (t appTray) ShowWindow() {
	wruntime.WindowShow(t.a.ctx)
	wruntime.WindowUnminimise(t.a.ctx)
}

func (t appTray) RunningItems() []trayItem {
	runs := runReg.list()
	sort.Slice(runs, func(i, j int) bool { return runs[i].Started < runs[j].Started })
	out := []trayItem{}
	for _, r := range runs {
		name := r.ProfileID
		if p, ok := t.a.store.GetProfile(r.ProfileID); ok && p.Name != "" {
			name = p.Name
		}
		out = append(out, trayItem{ID: r.ProfileID, Name: name})
	}
	return out
}

func (t appTray) FocusProfile(id string) { _ = t.a.FocusProfile(id) }

func (t appTray) StopAll() {
	ids := []string{}
	for _, r := range runReg.list() {
		ids = append(ids, r.ProfileID)
	}
	if len(ids) > 0 {
		t.a.StopProfiles(ids)
	}
}

func (t appTray) QuitApp() { t.a.quitApp() }

// quitApp is a real quit: OnShutdown then stops profiles and drains sync.
func (a *App) quitApp() {
	appQuitting.Store(true)
	wruntime.Quit(a.ctx)
}
