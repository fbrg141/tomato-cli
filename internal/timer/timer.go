// Package timer holds the session state machine: work/pause phases,
// optional cycle count, pause/resume, and the tick driver.
package timer

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fbrg141/tomato-cli/internal/orb"
)

// Config is a parsed session configuration.
type Config struct {
	Work   time.Duration
	Pause  time.Duration
	Cycles int // number of work intervals; 0 = infinite
}

type phase int

const (
	workPhase phase = iota
	pausePhase
)

const tickInterval = time.Second / 30

type tickMsg time.Time

// Model is the timer screen.
type Model struct {
	cfg       Config
	phase     phase
	remaining time.Duration
	animT     float64   // animation clock, driven by wall-clock deltas
	lastTick  time.Time // wall clock of the previously received tick

	interval   int // completed work intervals
	paused     bool
	done       bool
	bellFrames int
	quit       bool

	width, height int
}

// New builds a timer model in the work phase.
func New(cfg Config) Model {
	return Model{
		cfg:       cfg,
		phase:     workPhase,
		remaining: cfg.Work,
	}
}

func (m *Model) SetSize(w, h int)   { m.width, m.height = w, h }
func (m Model) QuitRequested() bool { return m.quit }

func tickEvery() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Init() tea.Cmd { return tickEvery() }

// Update handles messages. Always reschedules the tick while running.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if m.bellFrames > 0 {
			m.bellFrames--
		}
		// Advance on wall-clock deltas so the animation and countdown
		// stay smooth and accurate even when ticks arrive late or in bursts.
		var dt time.Duration
		if !m.lastTick.IsZero() {
			if d := time.Time(msg).Sub(m.lastTick); d > 0 {
				dt = d
			}
		}
		m.lastTick = time.Time(msg)
		if !m.paused && !m.done {
			m.animT += dt.Seconds()
			m.remaining -= dt
			if m.remaining <= 0 {
				m.transition()
			}
		} else if m.paused {
			// Dreamy slow drift while resting instead of a hard freeze.
			m.animT += dt.Seconds() * 0.12
		}
		return m, tickEvery()

	case tea.KeyMsg:
		switch msg.String() {
		case " ", "space":
			if !m.done {
				m.paused = !m.paused
			}
		case "q":
			m.quit = true
		default:
			if m.done {
				m.quit = true
			}
		}
	}
	return m, nil
}

// transition fires at the end of a phase; rings the bell and advances.
func (m *Model) transition() {
	m.bellFrames = 2 // \a rendered for ~2 frames
	if m.phase == workPhase {
		m.interval++
		if m.cfg.Cycles > 0 && m.interval >= m.cfg.Cycles {
			m.done = true
			return
		}
		m.phase = pausePhase
		m.remaining = m.cfg.Pause
	} else {
		m.phase = workPhase
		m.remaining = m.cfg.Work
	}
}

func (m Model) View() string {
	var b strings.Builder
	if m.bellFrames > 0 {
		b.WriteString("\a")
	}
	if m.width < 70 || m.height < 24 {
		b.WriteString(dimStyle.Render("\n  terminal too small — need at least 70 columns x 24 rows.\n  resize the window and the timer will appear.\n"))
		return b.String()
	}
	if m.done {
		b.WriteString(m.doneView())
		return b.String()
	}

	mode := orb.Work
	if m.phase == pausePhase {
		mode = orb.Pause
	}

	// Layout: countdown dead-center on the screen, orb above it. The orb's
	// frame extends down over the counter rows so that when it breathes and
	// deforms it washes over the timer instead of being cut off — the
	// digits shine through the overlapping orb, dimmed. Legend at bottom.
	zone := m.height - 1 // rows above the legend
	counterH := 6       // label + digits
	if m.paused {
		counterH++
	}
	counterTop := (zone - counterH + 1) / 2

	// The circle is budgeted to reach the top of the countdown (it
	// hovers over the label, particles brushing it, and breathes down over
	// the digits); the frame keeps going past it so the orb expands over
	// the timer instead of being clipped.
	circleRows := counterTop + 1
	if circleRows > 26 {
		circleRows = 26
	}
	frameTop := counterTop - circleRows
	if frameTop < 0 {
		frameTop = 0
	}
	frameH := counterTop - frameTop + counterH + 2 // counter + two spare rows
	frameBot := frameTop + frameH - 1

	phaseDur := m.cfg.Work
	if m.phase == pausePhase {
		phaseDur = m.cfg.Pause
	}
	progress := 1 - m.remaining.Seconds()/phaseDur.Seconds()
	if progress < 0 {
		progress = 0
	} else if progress > 1 {
		progress = 1
	}

	// Counter composited over the orb frame: label, big digits, and the
	// paused hint. Where the orb covers them, they shine through dimmed.
	label := "FOCUS"
	labStyle := labelStyle
	if m.phase == pausePhase {
		label = "PAUSE"
		labStyle = tealLabelStyle
	}
	lines := []string{labStyle.Render(label)}
	lines = append(lines, strings.Split(orb.Digits(mode, m.remaining), "\n")...)
	if m.paused {
		lines = append(lines, centerLines(pauseStyle.Render("❚❚  PAUSED — space to resume"), m.width))
	}
	ov := orb.Overlay{Row: counterTop - frameTop, Lines: lines}

	b.WriteString(strings.Repeat("\n", frameTop))
	b.WriteString(orb.Frame(mode, m.animT, m.width, frameH, circleRows, m.paused, progress, ov))

	// Remaining blank rows, then the legend on the last row.
	b.WriteString("\n")
	for row := frameBot + 1; row < m.height-1; row++ {
		b.WriteString("\n")
	}
	b.WriteString(centerLines(m.legend(), m.width))
	return b.String()
}

func (m Model) legend() string {
	cur := m.interval + 1
	total := fmt.Sprint(m.cfg.Cycles)
	if m.cfg.Cycles == 0 {
		total = "∞"
	}
	if cur > m.cfg.Cycles && m.cfg.Cycles > 0 {
		cur = m.cfg.Cycles
	}
	var legend strings.Builder
	for _, kv := range [][2]string{
		{"space", "pause"},
		{"q", "quit"},
	} {
		legend.WriteString(keyStyle.Render("["+kv[0]+"] ") + dimStyle.Render(kv[1]) + "   ")
	}
	legend.WriteString(dimStyle.Render("interval ") +
		keyStyle.Render(fmt.Sprintf("%d/%s", cur, total)))
	return legend.String()
}

func (m Model) doneView() string {
	orbH := m.height - 12
	if orbH < 6 {
		orbH = 6
	}
	if orbH > 26 {
		orbH = 26
	}
	var block strings.Builder
	block.WriteString(orb.Frame(orb.Work, m.animT, m.width, orbH, orbH, true, 0))
	block.WriteString("\n")
	block.WriteString(centerLines(doneStyle.Render("★  SESSION COMPLETE  ★"), m.width))
	block.WriteString("\n\n")
	stats := dimStyle.Render(fmt.Sprintf("%d focus interval(s) · %s work · %s pause",
		m.interval, m.cfg.Work, m.cfg.Pause))
	block.WriteString(centerLines(stats, m.width))
	block.WriteString("\n\n")
	block.WriteString(centerLines(dimStyle.Render("press any key to exit"), m.width))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, block.String())
}

func centerLines(s string, width int) string {
	var out strings.Builder
	for _, line := range strings.Split(s, "\n") {
		out.WriteString(lipgloss.PlaceHorizontal(width, lipgloss.Center, line))
		out.WriteString("\n")
	}
	return strings.TrimSuffix(out.String(), "\n")
}

// Styles.
var (
	dimStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	keyStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd98a")).Bold(true)
	labelStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6b3d")).Bold(true).Padding(0, 1)
	tealLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#3fd6c9")).Bold(true).Padding(0, 1)
	pauseStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd98a")).Bold(true)
	doneStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd98a")).Bold(true)
)
