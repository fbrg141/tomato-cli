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

	// Layout: orb dead-center on the screen, countdown right below it,
	// legend pinned to the bottom row.
	zone := m.height - 1 // rows above the legend
	extraRows := 0
	if m.paused {
		extraRows++
	}
	if m.focusHint {
		extraRows++
	}
	orbH := zone - 13 - extraRows // 13 = label + digits + breathing room
	if orbH > 26 {
		orbH = 26
	}
	orbTop := (zone - orbH + 1) / 2
	padBottom := m.height - 7 - orbTop - orbH - extraRows
	if padBottom < 0 {
		padBottom = 0
	}

	b.WriteString(strings.Repeat("\n", orbTop))
	b.WriteString(centerLines(orb.Frame(mode, m.animT, m.width, orbH, m.paused), m.width))
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
	if m.paused {
		b.WriteString("\n")
		b.WriteString(centerLines(pauseStyle.Render("❚❚  PAUSED — space to resume"), m.width))
	}
	if m.focusHint {
		b.WriteString("\n")
		b.WriteString(centerLines(focusStyle.Render(
			`focus: create a Shortcuts.app shortcut named "Tomato Focus" — planned, see GitHub issue #2`), m.width))
	}

	b.WriteString(strings.Repeat("\n", padBottom))
	b.WriteString("\n")
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
		{"f", "focus"},
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
	block.WriteString(centerLines(orb.Frame(orb.Work, m.animT, m.width, orbH, true), m.width))
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
	focusStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	doneStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd98a")).Bold(true)
)
