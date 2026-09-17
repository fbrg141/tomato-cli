package orb

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Frame must emit exactly `height` lines, each exactly `width` cells wide, so
// the caller can center the orb as a single block. Per-line trimming would
// make the orb wobble horizontally as particles extend lines asymmetrically.
func TestFrameFullWidthLines(t *testing.T) {
	for _, sz := range [][2]int{{80, 10}, {100, 16}, {70, 7}} {
		w, h := sz[0], sz[1]
		s := Frame(Work, 1.0, w, h, h, false, 0.5)
		lines := strings.Split(s, "\n")
		if len(lines) != h {
			t.Errorf("w=%d h=%d: got %d lines, want %d", w, h, len(lines), h)
		}
		for i, ln := range lines {
			if got := len([]rune(ln)); got != w {
				t.Errorf("w=%d h=%d: line %d has %d cells, want %d", w, h, i, got, w)
			}
		}
	}
}

// The rendered orb must stay horizontally centered across animation
// frames: the glyph centroid has to remain on the middle column (no
// wobble from particles extending lines asymmetrically).
func TestOrbDoesNotWobble(t *testing.T) {
	const w, h = 100, 12
	for _, at := range []float64{0, 0.37, 1.9, 4.2, 7.7} {
		lines := strings.Split(Frame(Work, at, w, h, h, false, 0.5), "\n")
		var sum, count float64
		for _, ln := range lines {
			for x, r := range []rune(ln) {
				if r != ' ' {
					sum += float64(x)
					count++
				}
			}
		}
		if count == 0 {
			t.Fatal("no orb glyphs rendered")
		}
		centroid := sum / count
		if d := centroid - float64(w)/2; d < -2.5 || d > 2.5 {
			t.Errorf("t=%v: orb centroid at column %.1f, %.1f cells off center", at, centroid, d)
		}
	}
}

// The end-of-phase heartbeat must be visible: rendering at the same instant
// with and without urgency should differ, and out-of-range progress must be
// clamped rather than distorting the orb.
func TestUrgencyChangesFrame(t *testing.T) {
	calm := Frame(Work, 4.0, 80, 12, 12, false, 0.5)
	urgent := Frame(Work, 4.0, 80, 12, 12, false, 0.99)
	if calm == urgent {
		t.Error("high phase progress should alter the orb (heartbeat)")
	}
	for _, p := range []float64{-1, 0, 1, 1.5} {
		if s := Frame(Work, 4.0, 80, 12, 12, false, p); s == "" {
			t.Errorf("progress=%v: empty frame", p)
		}
	}
}

// Overlaid content must shine through the orb dimmed where the orb covers
// it, and render at full brightness where it doesn't.
func TestOverlayTransparency(t *testing.T) {
	// Force a color profile so styling is observable; restore afterwards.
	r := lipgloss.DefaultRenderer()
	old := r.ColorProfile()
	r.SetColorProfile(termenv.TrueColor)
	defer r.SetColorProfile(old)

	digits := strings.Split(Digits(Work, 90*time.Second), "\n")
	// Frame 16 rows tall, circle budgeted to the top 10 rows: rows 4-8 are
	// covered by the orb, rows 13+ are empty.
	covered := Frame(Work, 1.0, 80, 16, 10, false, 0.5, Overlay{Row: 4, Lines: digits})
	if !strings.Contains(covered, "\x1b[2m") {
		t.Error("digits behind orb glyphs should be dimmed (transparency)")
	}
	clear := Frame(Work, 1.0, 80, 16, 10, false, 0.5, Overlay{Row: 13, Lines: digits})
	if strings.Contains(clear, "\x1b[2m") {
		t.Error("digits over empty cells should render at full brightness")
	}
	// The frame must still be full-width lines in both cases.
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	for name, s := range map[string]string{"covered": covered, "clear": clear} {
		for i, ln := range strings.Split(ansi.ReplaceAllString(s, ""), "\n") {
			if got := len([]rune(ln)); got != 80 {
				t.Errorf("%s: line %d has %d cells, want 80", name, i, got)
			}
		}
	}
}

func TestDigitsFormat(t *testing.T) {
	got := Digits(Work, 90*time.Second)
	lines := strings.Split(got, "\n")
	if len(lines) != 5 {
		t.Fatalf("Digits should render 5 rows, got %d", len(lines))
	}
	// 01:30 — 5 glyphs, 4 separators.
	if w := len([]rune(lines[0])); w != 5*5+4 {
		t.Errorf("Digits row width %d, want %d", w, 5*5+4)
	}
}