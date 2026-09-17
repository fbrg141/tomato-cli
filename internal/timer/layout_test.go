package timer

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

// orbRows returns the first and last rendered line index containing orb glyphs.
func orbRows(lines []string) (top, bottom int) {
	top, bottom = -1, -1
	for i, ln := range lines {
		if strings.ContainsAny(ln, "⣿⠿⢿⣀⡿") {
			if top == -1 {
				top = i
			}
			bottom = i
		}
	}
	return top, bottom
}

// The orb must sit dead-center of the screen with the legend on the last row,
// and the view must fill the terminal exactly (no scrolling).
func TestOrbCenteredAtAllSizes(t *testing.T) {
	for _, sz := range [][2]int{{80, 24}, {80, 30}, {100, 40}, {120, 50}, {70, 25}} {
		w, h := sz[0], sz[1]
		for _, paused := range []bool{false, true} {
			m := New(Config{Work: 25 * time.Minute, Pause: 5 * time.Minute, Cycles: 0})
			m.SetSize(w, h)
			m.animT = 1.0
			m.paused = paused
			lines := strings.Split(ansiRe.ReplaceAllString(m.View(), ""), "\n")
			if len(lines) != h {
				t.Errorf("w=%d h=%d paused=%v: view has %d lines, want %d", w, h, paused, len(lines), h)
				continue
			}
			legend := strings.TrimSpace(lines[h-1])
			if !strings.Contains(legend, "interval") {
				t.Errorf("w=%d h=%d paused=%v: legend not on last row: %q", w, h, paused, legend)
			}
			top, bottom := orbRows(lines)
			if top == -1 {
				t.Errorf("w=%d h=%d paused=%v: no orb rendered", w, h, paused)
				continue
			}
			mid := (top + bottom) / 2
			center := (h - 1) / 2
			// ±1: with an even row count the geometric center falls between
			// rows, so exact centering is impossible.
			if mid-center < -1 || mid-center > 1 {
				t.Errorf("w=%d h=%d paused=%v: orb mid row %d, want %d", w, h, paused, mid, center)
			}
		}
	}
}

func TestDoneViewFillsScreen(t *testing.T) {
	m := New(Config{Work: time.Second, Pause: time.Second, Cycles: 1})
	m.SetSize(80, 30)
	m.done = true
	lines := strings.Split(ansiRe.ReplaceAllString(m.View(), ""), "\n")
	if len(lines) > 30 {
		t.Errorf("done view has %d lines, want <= 30", len(lines))
	}
	if !strings.Contains(strings.Join(lines, "\n"), "SESSION COMPLETE") {
		t.Error("done view missing title")
	}
}