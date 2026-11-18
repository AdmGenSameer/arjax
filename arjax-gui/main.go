package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "Arjax — Universal Package Manager",
		Width:            1140,
		Height:           740,
		MinWidth:         920,
		MinHeight:        620,
		DisableResize:    false,
		Fullscreen:       false,
		Frameless:        false,
		StartHidden:      false,
		AlwaysOnTop:      false,
		BackgroundColour: &options.RGBA{R: 13, G: 14, B: 20, A: 255},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: app.IconHandler(),
		},
		Linux: &linux.Options{
			ProgramName: "arjax-gui",
		},
	})
	if err != nil {
		panic(err)
	}
}

