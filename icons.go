package main

import "github.com/egoist/mygo/ui"

func icon(shapes string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">` + shapes + `</svg>`))
}

var (
	// iconLogo is the mark of the app icon: an arrow coming down into a tray.
	iconLogo = icon(`<path d="M12 3.6v9.8"/><path d="M8.2 9.6 12 13.4l3.8-3.8"/><path d="M4.4 15.2v2.8a2.4 2.4 0 0 0 2.4 2.4h10.4a2.4 2.4 0 0 0 2.4-2.4v-2.8"/>`)

	iconDownload = iconLogo
	iconHistory  = icon(`<path d="M3.8 12a8.2 8.2 0 1 0 2.6-6"/><path d="M3.4 4.2v4.4h4.4"/><path d="M12 8.2V12l2.8 1.8"/>`)
	iconSettings = icon(`<path d="M4 7.2h9"/><path d="M17 7.2h3"/><circle cx="15" cy="7.2" r="2.2"/><path d="M4 16.8h3"/><path d="M11 16.8h9"/><circle cx="9" cy="16.8" r="2.2"/>`)

	iconStop  = icon(`<rect x="6.6" y="6.6" width="10.8" height="10.8" rx="2.4" fill="currentColor" stroke="none"/>`)
	iconTrash = icon(`<path d="M4.6 7h14.8"/><path d="M9.4 7V5.2a1.6 1.6 0 0 1 1.6-1.6h2a1.6 1.6 0 0 1 1.6 1.6V7"/><path d="M6.6 7l.8 11.2a2 2 0 0 0 2 1.8h5.2a2 2 0 0 0 2-1.8L17.4 7"/>`)
	iconCheck = icon(`<path d="M5 12.6 9.6 17 19 6.8"/>`)
	iconAlert = icon(`<path d="M12 4.4 3.2 20h17.6z"/><path d="M12 9.8v4.6"/><circle cx="12" cy="17.2" r="0.6" fill="currentColor"/>`)
)
