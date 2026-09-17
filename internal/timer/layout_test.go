package timer

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

// digitRows returns the first and last rendered line index of the big countdown.
func digitRows(lines []string) (top, bottom int) {
	top, bottom = -1, -1
	for i, ln := range lines {
		if strings.ContainsRune(ln, '█') {
			if top == -1 {
				top = i
			}
			bottom = i
		}
	}
	return top, bottom
}

// The countdown must sit dead-center of the screen with the orb above it and
// the legend on the last row; the view must fill the terminal exactly.
func TestCounterCenteredAtAllSizes(t *testing.T) {
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
			if legend := strings.TrimSpace(lines[h-1]); !strings.Contains(legend, "interval") {
				t.Errorf("w=%d h=%d paused=%v: legend not on last row: %q", w, h, paused, legend)
			}
			top, bottom := digitRows(lines)
			if top == -1 {
				t.Errorf("w=%d h=%d paused=%v: no countdown rendered", w, h, paused)
				continue
			}
			mid := (top + bottom) / 2
			center := (h - 1) / 2
			// ±1: with an even row count the geometric center falls between
			// rows, so exact centering is impossible.
			if mid-center < -1 || mid-center > 1 {
				t.Errorf("w=%d h=%d paused=%v: countdown mid row %d, want %d", w, h, paused, mid, center)
			}
			// The orb must render above the countdown.
			orbFound := false
			for i := 0; i < top; i++ {
				if strings.ContainsAny(lines[i], "⣿⠿⢿⣀⡿") {
					orbFound = true
					break
				}
			}
			if !orbFound {
				t.Errorf("w=%d h=%d paused=%v: no orb above countdown", w, h, paused)
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