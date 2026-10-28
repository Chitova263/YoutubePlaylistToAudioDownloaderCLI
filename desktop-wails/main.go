package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "YouTube Playlist Downloader",
		Width:  960,
		Height: 640,
		MinWidth:  700,
		MinHeight: 480,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:          app.startup,
		OnShutdown:         app.shutdown,
		Bind:               []interface{}{app},
		Logger:             logger.NewDefaultLogger(),
		LogLevel:           logger.DEBUG,
		LogLevelProduction: logger.ERROR,
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})
	if err != nil {
		log.Fatal("Error:", err)
	}
}
