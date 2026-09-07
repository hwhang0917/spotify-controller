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
		Title:     "vibe-music",
		Width:     1200,
		Height:    800,
		MinWidth:  960,
		MinHeight: 640,
		// match the canvas so the window never flashes white before the splash paints
		BackgroundColour: &options.RGBA{R: 250, G: 250, B: 250, A: 255},
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
