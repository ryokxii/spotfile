package main

import (
	"embed"
	"io"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

// filteredWriter silences the harmless "Unsolicited response received on idle
// HTTP channel" noise that Go's net/http transport logs when Wails' internal
// keep-alive connections receive a stale 400 from the webview bridge.
type filteredWriter struct{ w io.Writer }

func (fw filteredWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "Unsolicited response received on idle HTTP channel") {
		return len(p), nil
	}
	return fw.w.Write(p)
}

func main() {
	log.SetOutput(filteredWriter{os.Stderr})

	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "Spotfile",
		Width:     1440,
		Height:    900,
		MinWidth:  1000,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 12, G: 12, B: 15, A: 255},
		// Wails only enables the macOS zoom (green, full screen) button when Mac
		// options are present, so pass an empty set rather than nil.
		Mac:        &mac.Options{},
		Menu:       appMenu(),
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

// appMenu returns the standard macOS menu bar. The Window menu is where macOS
// adds "Enter Full Screen" (⌃⌘F), and the Edit menu makes ⌘C/⌘V/⌘A work in
// text fields. Windows keeps its native title bar without a menu bar.
func appMenu() *menu.Menu {
	if runtime.GOOS != "darwin" {
		return nil
	}
	return menu.NewMenuFromItems(menu.AppMenu(), menu.EditMenu(), menu.WindowMenu())
}
