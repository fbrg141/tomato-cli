// Package orb renders the animated breathing orb and big countdown digits.
package orb

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Mode selects the color ramp: focus work or pause.
type Mode int

const (
	Work Mode = iota
	Pause
)

// Palettes from dim (low intensity) to bright (high intensity).
var ramps = [2][]string{
	Work: {
		"#2b0505", "#4a0a0a", "#6b1010", "#8f1913", "#b32517",
		"#d63d1e", "#ea5c2b", "#f6823f", "#ffb257", "#ffd98a",
	},
	Pause: {
		"#041d1d", "#06302f", "#094a48", "#0d6a66", "#12908a",
		"#1ab5ab", "#3fd6c9", "#74e7da", "#aef3ea", "#e2fffa",
	},
}

// glyphKey identifies a styled braille glyph: mode, ramp index, dot bits.
type glyphKey struct {
	mode Mode
	ramp int
	bits int
}

// glyphCache memoizes styled glyphs so frames don't construct thousands of
// lipgloss styles per second. Filled lazily so the color profile is already
// detected by the time the first glyph is rendered.
var (
	glyphMu    sync.Mutex
	glyphCache = map[glyphKey]string{}

	// faintStyle dims overlaid content where the orb passes over it,
	// producing the transparency effect.
	faintStyle = lipgloss.NewStyle().Faint(true)
)

func styledGlyph(mode Mode, rampIdx, bits int) string {
	k := glyphKey{mode: mode, ramp: rampIdx, bits: bits}
	glyphMu.Lock()
	defer glyphMu.Unlock()
	if s, ok := glyphCache[k]; ok {
		return s
	}
	s := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ramps[mode][rampIdx])).
		Render(string(rune(0x2800 + bits)))
	glyphCache[k] = s
	return s
}

// Overlay is pre-rendered content composited over the frame: styled lines
// whose visible runes replace the orb cells behind them (spaces are
// transparent). Row is the first overlay row, 0 being the top of the frame.
type Overlay struct {
	Row   int
	Lines []string
}

type styledCell struct {
	s string // full ANSI-wrapped cell substring
	r rune   // the visible rune
}

// splitStyledCells splits a styled line into one substring per visible rune,
// keeping the escape sequences that style it. Concatenating the cells
// reproduces the original line.
func splitStyledCells(s string) []styledCell {
	var (
		cells []styledCell
		cur   strings.Builder
		inEsc bool
	)
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEsc = true
			cur.WriteRune(r)
		case inEsc:
			cur.WriteRune(r)
			if r == 'm' {
				inEsc = false
			}
		default:
			cur.WriteRune(r)
			cells = append(cells, styledCell{s: cur.String(), r: r})
			cur.Reset()
		}
	}
	if cur.Len() > 0 && len(cells) > 0 {
		// trailing escape junk: keep it with the last cell
		cells[len(cells)-1].s += cur.String()
	}
	return cells
}

// Frame renders the orb as braille art. t is animation time in seconds.
// width/height are in terminal cells (the frame has exactly height lines);
// circleRows is the vertical budget for the circle itself — the circle is
// centered in the top circleRows rows, sized to it, and the frame may
// extend further down so breathing/deformation can wash over content
// composited there. Pass circleRows == height for the classic centered
// orb. dim lowers brightness (paused state). progress is the current
// phase's completion in [0,1] — it drives an end-of-phase heartbeat; pass
// 0 to disable. Overlays are composited over the orb (see Overlay).
func Frame(mode Mode, t float64, width, height, circleRows int, dim bool, progress float64, overlays ...Overlay) string {
	if width < 3 {
		width = 3
	}
	if height < 2 {
		height = 2
	}
	if circleRows <= 0 || circleRows > height {
		circleRows = height
	}
	if progress < 0 {
		progress = 0
	} else if progress > 1 {
		progress = 1
	}

	// Urgency: in the last 10% of a phase the heartbeat ramps up to full.
	urgency := 0.0
	if progress > 0.9 {
		urgency = (progress - 0.9) * 10
		if urgency > 1 {
			urgency = 1
		}
	}

	maxR := math.Min(float64(circleRows*4), float64(width*2)) * 0.41
	// Slow 8s breathing, plus a quick heartbeat near the end of a phase.
	breath := 1 + 0.06*math.Sin(t*2*math.Pi/8) +
		urgency*0.05*math.Sin(t*2*math.Pi/0.9)
	R := maxR * breath
	// Heartbeat flash: a pulse of light from the core on every beat.
	beat := 0.0
	if urgency > 0 {
		beat = urgency * 0.3 * math.Max(0, math.Sin(t*2*math.Pi/0.9))
	}

	// Circle center in dot coordinates (each cell is 2 dots wide, 4 dots
	// tall): the middle of the top circleRows rows.
	cx := float64(width) // width*2 dots / 2
	cy := float64(circleRows) * 2.0

	// A comet of light circles the rim once every 12s, dragging a tail.
	cometA := t * 2 * math.Pi / 12
	cometAmp := 0.55
	if mode == Pause {
		cometAmp = 0.40
	}

	// Shimmer particles on slow orbits, each trailing behind itself.
	type particle struct{ x, y, tx, ty float64 }
	parts := make([]particle, 5)
	for i := range parts {
		ang := t*(0.25+0.06*float64(i)) + float64(i)*2*math.Pi/5
		r := R*1.10 + 1.5*math.Sin(t*0.5+float64(i)*1.3)
		ta := ang - 0.18
		parts[i] = particle{
			cx + r*math.Cos(ang), cy + r*0.95*math.Sin(ang),
			cx + r*math.Cos(ta), cy + r*0.95*math.Sin(ta),
		}
	}

	var out strings.Builder
	for y := 0; y < height; y++ {
		cells := make([]string, width)
		for x := 0; x < width; x++ {
			bits := 0
			best := 0
			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 2; dx++ {
					px := float64(x*2 + dx)
					py := float64(y*4 + dy)
					d := math.Hypot(px-cx, py-cy)
					th := math.Atan2(py-cy, px-cx)

					// Organic edge: low-frequency angular wobble rides on
					// top of the breathing, and the blob grows agitated
					// (a fast tremble) as the phase nears its end.
					deform := 0.05*math.Sin(2*th+t*0.7) +
						0.035*math.Sin(3*th-t*0.5+1.3) +
						urgency*0.03*math.Sin(4*th+t*2.4)
					edge := R * (1 + deform)

					v := 0.0
					if d < edge {
						q := d / edge
						v = (1 - q*q) * (1 - 0.25*q) + beat*(1-q)
					}

					// Swirling plasma bands inside the orb: two
					// counter-rotating waves, fading toward core and rim.
					if q := d / edge; q < 1.25 {
						win := q * (1.25 - q)
						v += 0.22 * math.Sin(3*th+t*0.9+q*4.0) * win
						v += 0.10 * math.Sin(5*th-t*0.6-q*6.0) * win
					}

					// Glowing rim, following the deformed edge.
					rim := d - edge*1.04
					v += 0.6 * math.Exp(-rim*rim/1.2)

					// Comet streak racing around the rim; the tail
					// widens behind the head.
					dd := math.Mod(th-cometA, 2*math.Pi)
					if dd > math.Pi {
						dd -= 2 * math.Pi
					} else if dd < -math.Pi {
						dd += 2 * math.Pi
					}
					s2 := 0.05
					if dd < 0 {
						s2 = 0.05 - 0.20*dd/math.Pi
					}
					v += cometAmp * math.Exp(-dd*dd/s2) * math.Exp(-rim*rim/2.5)

					// Particle glows and their tails.
					for _, p := range parts {
						pd := math.Hypot(px-p.x, py-p.y)
						v += 0.35 * math.Exp(-pd*pd/1.5)
						td := math.Hypot(px-p.tx, py-p.ty)
						v += 0.15 * math.Exp(-td*td/1.2)
					}

					if v > 0.10 {
						bits |= brailleBit[dy][dx]
					}
					if idx := int(v * 9); idx > best {
						best = idx
					}
				}
			}
			if bits == 0 {
				cells[x] = " "
				continue
			}
			if best > 9 {
				best = 9
			}
			cells[x] = styledGlyph(mode, best, bits)
		}

		// Composite overlays on top of the orb: visible runes replace
		// the cells behind them, spaces stay transparent.
		for _, ov := range overlays {
			i := y - ov.Row
			if i < 0 || i >= len(ov.Lines) {
				continue
			}
			oc := splitStyledCells(ov.Lines[i])
			if len(oc) == 0 {
				continue
			}
			pad := (width - len(oc)) / 2
			for j, c := range oc {
				if c.r == ' ' {
					continue // transparent: orb shows through
				}
				x := pad + j
				if x < 0 || x >= width {
					continue
				}
				if cells[x] == " " {
					cells[x] = c.s
				} else {
					// The orb passes over the timer: the glyph
					// shines through it, dimmed.
					cells[x] = faintStyle.Render(c.s)
				}
			}
		}
		out.WriteString(strings.Join(cells, "") + "\n")
	}

	s := out.String()
	if dim {
		s = lipgloss.NewStyle().Faint(true).Render(strings.TrimRight(s, "\n"))
	}
	return strings.TrimRight(s, "\n")
}

var brailleBit = [4][2]int{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

// 5-row block font for the big countdown.
var font = map[rune][5]string{
	'0': {"█████", "█   █", "█   █", "█   █", "█████"},
	'1': {"  █  ", " ██  ", "  █  ", "  █  ", " ███ "},
	'2': {"█████", "    █", "█████", "█    ", "█████"},
	'3': {"█████", "    █", " ███ ", "    █", "█████"},
	'4': {"█   █", "█   █", "█████", "    █", "    █"},
	'5': {"█████", "█    ", "█████", "    █", "█████"},
	'6': {"█████", "█    ", "█████", "█   █", "█████"},
	'7': {"█████", "    █", "    █", "    █", "    █"},
	'8': {"█████", "█   █", "█████", "█   █", "█████"},
	'9': {"█████", "█   █", "█████", "    █", "█████"},
	':': {"     ", "  █  ", "     ", "  █  ", "     "},
}

// Digits renders the big MM:SS countdown.
func Digits(mode Mode, remaining time.Duration) string {
	s := int(math.Ceil(remaining.Seconds()))
	if s < 0 {
		s = 0
	}
	if s > 99*60+59 {
		s = 99*60 + 59
	}
	seq := fmt.Sprintf("%02d:%02d", s/60, s%60)

	ramp := ramps[mode]
	bright := lipgloss.NewStyle().Foreground(lipgloss.Color(ramp[len(ramp)-1])).Bold(true)

	var rows [5]strings.Builder
	for i, ch := range seq {
		g := font[ch]
		for r := 0; r < 5; r++ {
			rows[r].WriteString(bright.Render(g[r]))
			if i < len(seq)-1 {
				rows[r].WriteByte(' ')
			}
		}
	}
	var out strings.Builder
	for r := 0; r < 5; r++ {
		out.WriteString(rows[r].String())
		if r < 4 {
			out.WriteByte('\n')
		}
	}
	return out.String()
}
