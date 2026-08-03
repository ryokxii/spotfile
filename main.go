package main

import (
	"embed"
	"io"
	"log"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
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
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
