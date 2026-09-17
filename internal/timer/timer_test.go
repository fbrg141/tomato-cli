package timer

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func teaKey(s string) tea.KeyMsg {
	if s == " " {
		return tea.KeyMsg{Type: tea.KeySpace}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestTransitionsFullCycle(t *testing.T) {
	m := New(Config{Work: 3 * time.Second, Pause: 2 * time.Second, Cycles: 2})
	tick := tickMsg(time.Now())
	for i := 0; i < 120; i++ {
		m, _ = m.Update(tick)
	}
	if !m.done {
		t.Fatalf("expected done after 2 cycles, got remaining=%v interval=%d", m.remaining, m.interval)
	}
	if m.interval != 2 {
		t.Fatalf("expected 2 completed intervals, got %d", m.interval)
	}
}

func TestInfiniteCycles(t *testing.T) {
	m := New(Config{Work: time.Second, Pause: time.Second, Cycles: 0})
	tick := tickMsg(time.Now())
	for i := 0; i < 100; i++ {
		m, _ = m.Update(tick)
	}
	if m.done {
		t.Fatal("infinite mode should never be done")
	}
	// 100 ticks = 10s = 10 phases; every other phase end is a work interval.
	if m.interval != 5 {
		t.Fatalf("expected 5 completed intervals after 10s, got %d", m.interval)
	}
}

func TestPauseFreezesCountdown(t *testing.T) {
	m := New(Config{Work: time.Second, Pause: time.Second, Cycles: 0})
	tick := tickMsg(time.Now())
	space := teaKey(" ")
	m, _ = m.Update(space)
	before := m.remaining
	for i := 0; i < 20; i++ {
		m, _ = m.Update(tick)
	}
	if !m.paused || m.remaining != before {
		t.Fatalf("paused timer should freeze; remaining before=%v after=%v", before, m.remaining)
	}
	if m.animT != 0 {
		t.Fatalf("animation clock should freeze while paused, got %v", m.animT)
	}
}

func TestParseDur(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"30", 30 * time.Minute, true},
		{"1:30", 90 * time.Second, true},
		{"0:03", 3 * time.Second, true},
		{"2.5", 150 * time.Second, true},
		{"", 0, false},
		{"abc", 0, false},
		{"1:75", 0, false},
	}
	for _, c := range cases {
		got, err := ParseDur(c.in)
		if c.ok && (err != nil || got != c.want) {
			t.Errorf("ParseDur(%q) = %v, %v; want %v", c.in, got, err, c.want)
		}
		if !c.ok && err == nil {
			t.Errorf("ParseDur(%q) should fail", c.in)
		}
	}
}
