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

// runTicks delivers n ticks spaced tickInterval apart on the wall clock,
// as the real program would.
func runTicks(m Model, n int) Model {
	base := time.Now()
	for i := 1; i <= n; i++ {
		m, _ = m.Update(tickMsg(base.Add(time.Duration(i) * tickInterval)))
	}
	return m
}

func TestTransitionsFullCycle(t *testing.T) {
	m := New(Config{Work: 3 * time.Second, Pause: 2 * time.Second, Cycles: 2})
	// 2 cycles = 10s of wall time; tick count derived from the tick rate.
	n := int(10*time.Second/tickInterval) + 20
	m = runTicks(m, n)
	if !m.done {
		t.Fatalf("expected done after 2 cycles, got remaining=%v interval=%d", m.remaining, m.interval)
	}
	if m.interval != 2 {
		t.Fatalf("expected 2 completed intervals, got %d", m.interval)
	}
}

func TestInfiniteCycles(t *testing.T) {
	m := New(Config{Work: time.Second, Pause: time.Second, Cycles: 0})
	n := 200
	m = runTicks(m, n)
	if m.done {
		t.Fatal("infinite mode should never be done")
	}
	// Each work interval k ends at k*work + (k-1)*pause of wall time.
	elapsed := time.Duration(n-1) * tickInterval
	want := int((elapsed + time.Second) / (2 * time.Second))
	if m.interval != want {
		t.Fatalf("after %v of wall time, expected %d completed intervals, got %d", elapsed, want, m.interval)
	}
}

func TestPauseFreezesCountdown(t *testing.T) {
	m := New(Config{Work: time.Second, Pause: time.Second, Cycles: 0})
	m, _ = m.Update(teaKey(" "))
	before := m.remaining
	m = runTicks(m, 20)
	if !m.paused || m.remaining != before {
		t.Fatalf("paused timer should freeze; remaining before=%v after=%v", before, m.remaining)
	}
	// The countdown is frozen but the orb should drift slowly, not stop dead:
	// 20 ticks ≈ 0.67s of wall time → ~0.08s of animation at 12% speed.
	if m.animT <= 0 || m.animT >= 0.2 {
		t.Fatalf("animation should drift slowly while paused, got animT=%v", m.animT)
	}
}

// Ticks may arrive late or in bursts; the countdown must follow wall time.
func TestWallClockTickingIgnoresTickJitter(t *testing.T) {
	m := New(Config{Work: time.Second, Pause: time.Second, Cycles: 0})
	base := time.Now()
	// First tick anchors the clock (no time elapses), then irregular gaps
	// totaling exactly 1s — the work phase should end on the last tick.
	gaps := []time.Duration{
		10 * time.Millisecond,
		240 * time.Millisecond,
		700 * time.Millisecond,
		50 * time.Millisecond,
		10 * time.Millisecond,
	}
	now := base
	for _, g := range gaps {
		now = now.Add(g)
		m, _ = m.Update(tickMsg(now))
	}
	if m.phase != pausePhase {
		t.Fatalf("expected pause phase after 1s of wall time, phase=%v remaining=%v", m.phase, m.remaining)
	}
	if m.interval != 1 {
		t.Fatalf("expected 1 completed interval, got %d", m.interval)
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