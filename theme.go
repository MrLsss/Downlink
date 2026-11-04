package main

import "github.com/egoist/mygo/ui"

// palette holds the tones the views paint with. The window is glass, so the
// page itself stays clear: only the sidebar dims the material, a field or a
// row lifts a chip above it, and the one accent comes from the theme.
type palette struct {
	scrim    ui.Color // the sidebar's tone over the window's glass
	chip     ui.Color // a field the content is written in
	hover    ui.Color // a row the pointer is on
	selected ui.Color // the row the window is on
	shadow   ui.Color // under what floats over the page
	ok       ui.Color
	warn     ui.Color
	danger   ui.Color
	muted    ui.Color
}

func paletteFor(t *ui.Theme) palette {
	if t.Dark {
		return palette{
			scrim:    ui.RGBA(0, 0, 0, 0.30),
			chip:     ui.RGBA(255, 255, 255, 0.06),
			hover:    ui.RGBA(255, 255, 255, 0.08),
			selected: ui.RGBA(255, 255, 255, 0.14),
			shadow:   ui.RGBA(0, 0, 0, 0.40),
			ok:       ui.Hex("#5BD6A0"),
			warn:     ui.Hex("#F2C14E"),
			danger:   ui.Hex("#F2736B"),
			muted:    t.TextMuted,
		}
	}
	return palette{
		scrim:    ui.RGBA(0, 0, 0, 0.06),
		chip:     ui.RGBA(0, 0, 0, 0.045),
		hover:    ui.RGBA(0, 0, 0, 0.055),
		selected: ui.RGBA(0, 0, 0, 0.10),
		shadow:   ui.RGBA(0, 0, 0, 0.14),
		ok:       ui.Hex("#1E9E6A"),
		warn:     ui.Hex("#A9761A"),
		danger:   ui.Hex("#C4403A"),
		muted:    ui.Hex("#6B6B73"),
	}
}
