// Package orb renders the animated breathing orb and big countdown digits.
package orb

import (
	"fmt"
	"math"
	"strings"
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

// Frame renders the orb as braille art. t is animation time in seconds.
// width/height are in terminal cells; dim lowers brightness (paused state).
func Frame(mode Mode, t float64, width, height int, dim bool) string {
	if width < 3 {
		width = 3
	}
	if height < 2 {
		height = 2
	}

	maxR := math.Min(float64(height*4), float64(width*2)) * 0.42
	breath := 1 + 0.06*math.Sin(t*2*math.Pi/8) // 8s breathing cycle
	R := maxR * breath

	// Center in dot coordinates (each cell is 2 dots wide, 4 dots tall).
	cx := float64(width) // width*2 dots / 2
	cy := float64(height) * 2.0

	// Shimmer particles on slow orbits.
	type particle struct{ x, y float64 }
	parts := make([]particle, 7)
	for i := range parts {
		ang := t*(0.25+0.06*float64(i)) + float64(i)*2*math.Pi/7
		r := R*1.18 + 2.0*math.Sin(t*0.5+float64(i)*1.3)
		parts[i] = particle{
			cx + r*math.Cos(ang),
			cy + r*0.95*math.Sin(ang),
		}
	}

	ramp := ramps[mode]
	var out strings.Builder
	for y := 0; y < height; y++ {
		var line strings.Builder
		for x := 0; x < width; x++ {
			bits := 0
			best := 0
			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 2; dx++ {
					px := float64(x*2 + dx)
					py := float64(y*4 + dy)
					d := math.Hypot(px-cx, py-cy)

					v := 0.0
					if d < R {
						q := d / R
						v = (1 - q*q) * (1 - 0.25*q)
					}
					// Glowing rim.
					rim := d - R*1.04
					v += 0.6 * math.Exp(-rim*rim/1.2)
					// Particle glows.
					for _, p := range parts {
						pd := math.Hypot(px-p.x, py-p.y)
						v += 0.35 * math.Exp(-pd*pd/1.5)
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
				line.WriteByte(' ')
				continue
			}
			if best > len(ramp)-1 {
				best = len(ramp) - 1
			}
			ch := string(rune(0x2800 + bits))
			line.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color(ramp[best])).Render(ch))
		}
		out.WriteString(strings.TrimRight(line.String(), " ") + "\n")
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
	dimC := lipgloss.NewStyle().Foreground(lipgloss.Color(ramp[4])).Bold(true)

	var rows [5]strings.Builder
	for i, ch := range seq {
		g := font[ch]
		st := bright
		if ch == ':' {
			st = dimC
		}
		for r := 0; r < 5; r++ {
			rows[r].WriteString(st.Render(g[r]))
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
