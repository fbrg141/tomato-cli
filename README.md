# tomato

A pomodoro timer for your terminal, with a breathing orb.

Built with Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Usage

```sh
tomato                    # open the setup screen (work 30 / pause 5 / ∞)
tomato -w 45 -p 10 -n 4   # prefill the setup screen
tomato -g -w 45           # skip setup, start immediately
```

- **work** — focus interval length, in minutes (`30`) or `mm:ss` (`1:30`)
- **pause** — break length, same format
- **cycles** — number of work intervals; leave empty / 0 to run until you quit

## Keys

| Key | Action |
|---|---|
| `tab` / `↑↓` | move between setup fields |
| `enter` | start the timer |
| `space` | pause / resume |
| `q` / `esc` | quit |

The terminal bell rings when a work or pause interval ends.

## Build

```sh
go build -o tomato .
```

## Layout

```
main.go               flags, screen switching
internal/setup/       setup form screen
internal/timer/       session state machine (work/pause × cycles)
internal/orb/         braille orb renderer + big countdown digits
```