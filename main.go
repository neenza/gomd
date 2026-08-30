package main

import (
	"embed"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Check for initial file argument from CLI
	var initialFilePath string
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, "-") && len(arg) > 0 {
			initialFilePath = arg
			break
		}
	}

	// Create an instance of the app structure
	app := NewApp(initialFilePath)

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "gomd",
		Width:     1120,
		Height:    760,
		MinWidth:  640,
		MinHeight: 400,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 17, G: 20, B: 24, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			WebviewGpuPolicy: linux.WebviewGpuPolicyOnDemand,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
