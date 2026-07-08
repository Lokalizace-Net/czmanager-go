package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

// Version je verze aplikace. Vkládá se při buildu přes ldflags:
//   -ldflags "-X main.Version=v1.6.1"
// Když se nevloží (např. lokální dev build), zůstane "dev".
var Version = "dev"

func main() {
	// Create an instance of the app structure
	app := NewApp()
	app.version = Version

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "CZManager",
		Width:     1280,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 18, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		// Drag & drop souborů (Manuální instalace - přetažení ZIPu)
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		// Windows specific options
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
		// macOS specific options
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            true,
				UseToolbar:                 false,
			},
			Appearance: mac.NSAppearanceNameDarkAqua,
		},
		// Linux specific options
		Linux: &linux.Options{
			ProgramName: "CZManager",
			Icon:        icon,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
