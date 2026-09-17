package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"tomato/internal/setup"
	"tomato/internal/timer"
)

type screen int

const (
	setupScreen screen = iota
	timerScreen
)

// model switches between the setup form and the timer.
type model struct {
	width, height int
	screen        screen
	setup         setup.Model
	timer         timer.Model
	timerOn       bool
}

func newModel(cfg timer.Config, startNow bool, workStr, pauseStr string) model {
	m := model{
		setup: setup.New(workStr, pauseStr, cfg.Cycles),
	}
	if startNow {
		m.timer = timer.New(cfg)
		m.timerOn = true
		m.screen = timerScreen
	}
	return m
}

func (m model) Init() tea.Cmd {
	if m.timerOn {
		return m.timer.Init()
	}
	return m.setup.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.setup.SetSize(m.width, m.height)
		if m.timerOn {
			m.timer.SetSize(m.width, m.height)
		}
		return m, nil

	case setup.StartMsg:
		m.timer = timer.New(msg.Config)
		m.timer.SetSize(m.width, m.height)
		m.timerOn = true
		m.screen = timerScreen
		return m, m.timer.Init()
	}

	if m.screen == setupScreen {
		var cmd tea.Cmd
		m.setup, cmd = m.setup.Update(msg)
		if m.setup.QuitRequested() {
			return m, tea.Quit
		}
		return m, cmd
	}

	if m.timerOn {
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		if m.timer.QuitRequested() {
			return m, tea.Quit
		}
		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	if m.screen == setupScreen || !m.timerOn {
		return m.setup.View()
	}
	return m.timer.View()
}

func main() {
	var (
		workF     = flag.String("work", "", "work length in minutes or mm:ss (default 30)")
		workS     = flag.String("w", "", "alias for -work")
		pauseF    = flag.String("pause", "", "pause length in minutes or mm:ss (default 5)")
		pauseS    = flag.String("p", "", "alias for -pause")
		cycles    = flag.Int("cycles", 0, "number of work intervals, 0 = infinite")
		cyclesS   = flag.Int("n", 0, "alias for -cycles")
		startNow  = flag.Bool("go", false, "skip the setup screen and start immediately")
		startNowS = flag.Bool("g", false, "alias for -go")
	)
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, `tomato — a pomodoro timer for your terminal

usage:
  tomato                    open the setup screen (work 30 / pause 5 / ∞)
  tomato -w 45 -p 10 -n 4   prefill the setup screen
  tomato -g -w 45           start immediately, no setup screen

flags:`)
		flag.PrintDefaults()
	}
	flag.Parse()

	go_ := *startNow || *startNowS
	cyc := *cycles
	if cyc == 0 {
		cyc = *cyclesS
	}

	work := pick(*workF, *workS)
	pause := pick(*pauseF, *pauseS)

	cfg, err := resolve(work, pause, cyc)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tomato:", err)
		fmt.Fprintln(os.Stderr, "use minutes (30) or mm:ss (1:30)")
		os.Exit(2)
	}

	p := tea.NewProgram(newModel(cfg, go_, work, pause), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tomato:", err)
		os.Exit(1)
	}
}

func pick(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// resolve turns flag strings (possibly empty) into a validated Config.
func resolve(workStr, pauseStr string, cycles int) (timer.Config, error) {
	cfg := timer.Config{Work: 30 * time.Minute, Pause: 5 * time.Minute, Cycles: cycles}
	if workStr != "" {
		d, err := timer.ParseDur(workStr)
		if err != nil {
			return cfg, fmt.Errorf("invalid -work %q", workStr)
		}
		cfg.Work = d
	}
	if pauseStr != "" {
		d, err := timer.ParseDur(pauseStr)
		if err != nil {
			return cfg, fmt.Errorf("invalid -pause %q", pauseStr)
		}
		cfg.Pause = d
	}
	if cfg.Work <= 0 || cfg.Pause <= 0 {
		return cfg, fmt.Errorf("work and pause must be positive")
	}
	return cfg, nil
}
