// Package timer holds the session state machine: work/pause phases,
// optional cycle count, pause/resume, and the tick driver.
package timer

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tomato/internal/orb"
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

const tickInterval = 100 * time.Millisecond

type tickMsg time.Time

// Model is the timer screen.
type Model struct {
	cfg       Config
	phase     phase
	remaining time.Duration
	elapsed   time.Duration // time in current phase
	animT     float64       // animation clock, frozen while paused

	interval   int // completed work intervals
	paused     bool
	done       bool
	focusHint  bool
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
		_ = msg
		if m.bellFrames > 0 {
			m.bellFrames--
		}
		if !m.paused && !m.done {
			m.animT += tickInterval.Seconds()
			m.remaining -= tickInterval
			m.elapsed += tickInterval
			if m.remaining <= 0 {
				m.transition()
			}
		}
		return m, tickEvery()

	case tea.KeyMsg:
		switch msg.String() {
		case " ", "space":
			if !m.done {
				m.paused = !m.paused
			}
		case "f", "F":
			m.focusHint = !m.focusHint
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
			m.elapsed = m.cfg.Work
			return
		}
		m.phase = pausePhase
		m.remaining = m.cfg.Pause
	} else {
		m.phase = workPhase
		m.remaining = m.cfg.Work
	}
	m.elapsed = 0
}

func (m Model) phaseDuration() time.Duration {
	if m.phase == workPhase {
		return m.cfg.Work
	}
	return m.cfg.Pause
}

func (m Model) View() string {
	if m.width < 70 || m.height < 24 {
		return dimStyle.Render("\n  terminal too small — need at least 70 columns x 24 rows.\n  resize the window and the timer will appear.\n")
	}

	mode := orb.Work
	if m.phase == pausePhase {
		mode = orb.Pause
	}

	var b strings.Builder
	if m.bellFrames > 0 {
		b.WriteString("\a")
	}

	// Layout: orb on top, label+digits+bar in the middle, legend at bottom.
	orbH := m.height - 11
	if orbH < 6 {
		orbH = 6
	}
	if orbH > 16 {
		orbH = 16
	}

	if m.done {
		return m.doneView()
	}

	orbArt := orb.Frame(mode, m.animT, m.width, orbH, m.paused)
	b.WriteString(centerLines(orbArt, m.width))
	b.WriteString("\n")

	label := "FOCUS"
	labelStyle := labelStyle
	if m.phase == pausePhase {
		label = "PAUSE"
		labelStyle = tealLabelStyle
	}
	b.WriteString(centerLines(labelStyle.Render(label), m.width))
	b.WriteString("\n")

	b.WriteString(centerLines(orb.Digits(mode, m.remaining), m.width))
	b.WriteString("\n")

	// Progress bar for the current phase.
	frac := 0.0
	if d := m.phaseDuration(); d > 0 {
		frac = m.elapsed.Seconds() / d.Seconds()
	}
	barW := 24
	filled := int(frac * float64(barW))
	if filled > barW {
		filled = barW
	}
	barStyle, fillStyle := tealBarStyle, tealFillStyle
	if m.phase == workPhase {
		barStyle, fillStyle = barStyleW, fillStyleW
	}
	bar := fillStyle.Render(strings.Repeat("█", filled)) +
		barStyle.Render(strings.Repeat("░", barW-filled))
	b.WriteString(centerLines(bar, m.width))
	b.WriteString("\n\n")

	if m.paused {
		b.WriteString(centerLines(pauseStyle.Render("❚❚  PAUSED — space to resume"), m.width))
		b.WriteString("\n")
	}
	if m.focusHint {
		b.WriteString(centerLines(focusStyle.Render(
			`focus: create a Shortcuts.app shortcut named "Tomato Focus" — planned, see GitHub issue #2`), m.width))
		b.WriteString("\n")
	}

	// Legend.
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
		{"f", "focus"},
		{"q", "quit"},
	} {
		legend.WriteString(keyStyle.Render("["+kv[0]+"] ") + dimStyle.Render(kv[1]) + "   ")
	}
	legend.WriteString(dimStyle.Render("interval ") +
		keyStyle.Render(fmt.Sprintf("%d/%s", cur, total)))
	b.WriteString(centerLines(legend.String(), m.width))

	return b.String()
}

func (m Model) doneView() string {
	var b strings.Builder
	if m.bellFrames > 0 {
		b.WriteString("\a")
	}
	orbArt := orb.Frame(orb.Work, m.animT, m.width, m.height-12, true)
	b.WriteString(centerLines(orbArt, m.width))
	b.WriteString("\n")
	title := doneStyle.Render("★  SESSION COMPLETE  ★")
	b.WriteString(centerLines(title, m.width))
	b.WriteString("\n\n")
	stats := dimStyle.Render(fmt.Sprintf("%d focus interval(s) · %s work · %s pause",
		m.cfg.Cycles, m.cfg.Work, m.cfg.Pause))
	b.WriteString(centerLines(stats, m.width))
	b.WriteString("\n\n")
	b.WriteString(centerLines(dimStyle.Render("press any key to exit"), m.width))
	return b.String()
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
	barStyleW      = lipgloss.NewStyle().Foreground(lipgloss.Color("#5a2410"))
	fillStyleW     = lipgloss.NewStyle().Foreground(lipgloss.Color("#f6823f"))
	tealBarStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#0d4a48"))
	tealFillStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#3fd6c9"))
	pauseStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd98a")).Bold(true)
	focusStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	doneStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd98a")).Bold(true)
)
