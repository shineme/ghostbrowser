package main

import (
	"context"
	"embed"
	"os"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--mcp-stdio" {
		os.Exit(runMcpStdio())
	}
	handleAfterUpdate(os.Args[1:]) // updater: wait for the previous version to exit
	// default browser: started by a link click (--open-url / bare URL / file)
	coldLinks = parseOpenURLArgs(os.Args[1:])
	if len(coldLinks) > 0 {
		allowForegroundHandoff()
	}
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "GhostBrowser",
		Width:     1360,
		Height:    860,
		MinWidth:  1100,
		MinHeight: 680,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Frameless:   true,
		StartHidden: len(coldLinks) > 0, // link click: open it, keep the manager out of the way
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "ghostbrowser-7d3f2a91-manager",
			OnSecondInstanceLaunch: func(d options.SecondInstanceData) {
				app.onSecondInstance(d.Args)
			},
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		Windows: &windows.Options{
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               false,
			DisableFramelessWindowDecorations: true,
			Theme:                             windows.Dark,
		},
		OnStartup:     app.startup,
		OnBeforeClose: app.beforeClose, // close-to-tray (appshell.go)
		OnShutdown: func(_ context.Context) {
			stopTray()
			runReg.stopAll()
			coreMgr.StopAll() // mihomo core (core.go)
			globalAuto.Drain(90 * time.Second)
			globalAuto.FlushGlobal()
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
