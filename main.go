package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	// This program asks a binary for its version by running it with "version".
	// Answering that here would open a window in place of a version, and the
	// probe that asked would start this program again.
	if os.Getenv(probeEnv) != "" {
		fmt.Fprintln(os.Stderr, "Downlink 不是 tdl")
		os.Exit(2)
	}

	mygo.Theme.SetSource(mygo.ThemeSystem)
	setupLog()

	a := newApp()
	mygo.App.WhenReady(func() {
		a.load()
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:          "Downlink",
			Width:          1120,
			Height:         764,
			MinWidth:       940,
			MinHeight:      620,
			StateKey:       "downlink",
			TitleBarStyle:  mygo.TitleBarHiddenInset,
			TitleBarHeight: 44,
			Vibrancy:       mygo.VibrancyGlass,
			Content:        ui.View(a.view),
		})
		a.win.OnClose(func(e *mygo.CloseEvent) { a.shutdown() })
		a.win.OnClosed(func() { mygo.App.Quit() })
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// setupLog writes what the program reports to a file beside its settings, so a
// failure that leaves no window behind can still be read.
func setupLog() {
	dir := dataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	log.SetOutput(io.MultiWriter(os.Stderr, f))
}
