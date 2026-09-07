package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// Admin UI (Wails window). The guest UI lives in ./web and is served by Chi.
//
//go:embed all:frontend/dist
var adminAssets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "vibe-music",
		Width:  900,
		Height: 640,
		AssetServer: &assetserver.Options{
			Assets: adminAssets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind:       []interface{}{app},
	})
	if err != nil {
		log.Fatal(err)
	}
}
