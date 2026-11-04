package main

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func channel(v uint8) float64 { return float64(v) / 255 }

// luminance is the relative brightness of a colour, the way the web measures
// contrast.
func luminance(c ui.Color) float64 {
	lin := func(v uint8) float64 {
		s := channel(v)
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

// contrast is the ratio between two colours: 1 for the same colour, 21 for
// black against white.
func contrast(a, b ui.Color) float64 {
	hi, lo := luminance(a), luminance(b)
	if lo > hi {
		hi, lo = lo, hi
	}
	return (hi + 0.05) / (lo + 0.05)
}

// over draws a translucent colour on top of base, which is what it ends up
// looking like.
func over(c, base ui.Color) ui.Color {
	mix := func(fg, bg uint8) uint8 {
		a := float64(c.A) / 255
		return uint8(math.Round(float64(fg)*a + float64(bg)*(1-a)))
	}
	return ui.RGBA(mix(c.R, base.R), mix(c.G, base.G), mix(c.B, base.B), 255)
}

// TestSmallTextStaysReadable keeps the quiet greys of both appearances above
// the contrast small print needs on the panes and on the page.
func TestSmallTextStaysReadable(t *testing.T) {
	for _, th := range []*ui.Theme{ui.LightTheme(), ui.DarkTheme()} {
		pal := paletteFor(th)
		surfaces := map[string]ui.Color{
			"page":  th.Background,
			"panel": th.Surface,
		}
		for surface, base := range surfaces {
			ink := over(pal.muted, base)
			if r := contrast(ink, base); r < 4.5 {
				t.Errorf("dark=%v muted text on the %s reaches only %.2f:1", th.Dark, surface, r)
			}
		}
	}
}
