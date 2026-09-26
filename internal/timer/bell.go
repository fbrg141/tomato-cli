package timer

import (
	"os/exec"
	"runtime"

	"github.com/charmbracelet/bubbletea"
)

// Bell is the tea.Cmd fired whenever a period ends (naturally or skipped).
// Terminals frequently mute the BEL character (VS Code, iTerm2 and friends
// default to a silent or visual bell), so the end of a period also plays a
// real system sound. Best effort: if no player is available, the BEL alone
// still goes out with the frame.
func Bell() tea.Msg {
	playSound()
	return nil
}

func playSound() {
	switch runtime.GOOS {
	case "darwin":
		// Ships with macOS; pleasant and short.
		exec.Command("afplay", "/System/Library/Sounds/Glass.aiff").Run()
	case "linux":
		for _, c := range [][2]string{
			{"canberra-gtk-play", "-i bell"},
			{"paplay", "/usr/share/sounds/freedesktop/stereo/bell.oga"},
			{"aplay", "/usr/share/sounds/alsa/Front_Center.wav"},
		} {
			name, args := c[0], c[1]
			if path, err := exec.LookPath(name); err == nil {
				exec.Command(path, args).Run()
				return
			}
		}
	case "windows":
		exec.Command("powershell", "-NoProfile", "-Command",
			"[console]::beep(880,300)").Run()
	}
}