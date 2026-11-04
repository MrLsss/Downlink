package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// shortenPath puts the home directory back under ~ and cuts what is still long.
func shortenPath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && home != "/" {
		if rest, ok := strings.CutPrefix(p, home); ok {
			p = "~" + rest
		}
	}
	if len([]rune(p)) <= 64 {
		return p
	}
	r := []rune(p)
	return "…" + string(r[len(r)-63:])
}

// humanBytes writes a size the way tdl does, in binary units.
func humanBytes(n int64) string {
	if n < 1024 {
		return itoa(n) + " B"
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	v := float64(n)
	i := -1
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", v, units[i])
}

// humanRate writes a speed, empty while there is none.
func humanRate(n int64) string {
	if n <= 0 {
		return ""
	}
	return humanBytes(n) + "/s"
}

// humanClock writes a duration as minutes and seconds.
func humanClock(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
