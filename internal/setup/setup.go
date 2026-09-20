// Package setup is the pre-timer form screen: work, pause, cycles.
package setup

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fbrg141/tomato-cli/internal/timer"
)

// StartMsg is emitted when the form is validated and the timer should start.
type StartMsg struct {
	Config timer.Config
}

// Model is the setup screen.
type Model struct {
	inputs []textinput.Model
	focus  int
	err    string
	width  int
	height int
	quit   bool
}

// New builds the form, prefilled from flags ("" means use placeholder default).
func New(work, pause string, cycles int) Model {
	w := work
	if w == "" {
		w = "30"
	}
	p := pause
	if p == "" {
		p = "5"
	}
	c := ""
	if cycles > 0 {
		c = strconv.Itoa(cycles)
	}

	m := Model{focus: 0}
	for i, v := range []string{w, p, c} {
		in := textinput.New()
		in.SetValue(v)
		in.CharLimit = 8
		in.Width = 16
		in.Prompt = ""
		m.inputs = append(m.inputs, in)
		_ = i
	}
	m.inputs[0].Focus()
	return m
}

func (m *Model) SetSize(w, h int)   { m.width, m.height = w, h }
func (m Model) QuitRequested() bool { return m.quit }

func (m Model) Init() tea.Cmd { return textinput.Blink }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			m.quit = true
			return m, nil
		case "esc":
			m.quit = true
			return m, nil
		case "tab", "down":
			m.inputs[m.focus].Blur()
			m.focus = (m.focus + 1) % len(m.inputs)
			m.inputs[m.focus].Focus()
			return m, nil
		case "shift+tab", "up":
			m.inputs[m.focus].Blur()
			m.focus = (m.focus - 1 + len(m.inputs)) % len(m.inputs)
			m.inputs[m.focus].Focus()
			return m, nil
		case "enter":
			return m, m.tryStart()
		}
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	for i := range m.inputs {
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) tryStart() tea.Cmd {
	cfg, err := m.parse()
	if err != nil {
		m.err = err.Error()
		return nil
	}
	m.err = ""
	return func() tea.Msg { return StartMsg{Config: cfg} }
}

// parse validates the form values into a Config.
func (m Model) parse() (timer.Config, error) {
	w, err := timer.ParseDur(m.inputs[0].Value())
	if err != nil {
		return timer.Config{}, errBad("work", m.inputs[0].Value())
	}
	p, err := timer.ParseDur(m.inputs[1].Value())
	if err != nil {
		return timer.Config{}, errBad("pause", m.inputs[1].Value())
	}
	cycles := 0 // infinite
	if v := strings.TrimSpace(m.inputs[2].Value()); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return timer.Config{}, errBad("cycles", v)
		}
		cycles = n
	}
	return timer.Config{Work: w, Pause: p, Cycles: cycles}, nil
}

func errBad(field, val string) error {
	return fmt.Errorf("%s %q is not valid — use minutes (30) or mm:ss (1:30)", field, val)
}

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	boxW := 44
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#b32517")).
		Padding(1, 3).
		Width(boxW)

	title := titleStyle.Render("tomato") +
		dimStyle.Render(" — a pomodoro timer")

	var fields strings.Builder
	labels := []string{"work", "pause", "cycles"}
	hints := []string{"minutes or mm:ss", "minutes or mm:ss", "empty = ∞"}
	for i, in := range m.inputs {
		cursor := dimStyle.Render("   ")
		if i == m.focus {
			cursor = keyStyle.Render(" ▶ ")
		}
		fields.WriteString(cursor +
			labelStyle.Render(fmt.Sprintf("%-6s", labels[i])) +
			in.View() +
			dimStyle.Render(" "+hints[i]) + "\n\n")
	}

	var inner strings.Builder
	inner.WriteString(title + "\n\n")
	inner.WriteString(fields.String())
	if m.err != "" {
		inner.WriteString(errStyle.Render("✗ "+m.err) + "\n\n")
	}

	legend := keyStyle.Render("[tab/↑↓]") + dimStyle.Render(" field   ") +
		keyStyle.Render("[enter]") + dimStyle.Render(" start   ") +
		keyStyle.Render("[esc]") + dimStyle.Render(" quit")

	card := box.Render(inner.String() + "\n" + legend)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, card)
}

// Styles.
var (
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6b3d")).Bold(true)
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#e8dcc8"))
	keyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd98a")).Bold(true)
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5c5c"))
)
