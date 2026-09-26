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
| `enter` | skip the current period |
| `q` / `esc` | quit |

When a period ends, a system sound plays and the terminal bell rings (skipping with `enter` also sounds, since the period is over). If your terminal mutes bells, you'll still hear the sound.

## Install

```sh
go install github.com/fbrg141/tomato-cli@latest
```

Or grab a prebuilt binary from a [release](https://github.com/fbrg141/tomato-cli/releases).

## Build

```sh
go build -o tomato .
```

To cut a release (binaries + checksums on GitHub Releases, via CI):

```sh
git tag v0.1.0 && git push origin v0.1.0
```

Local dry run:

```sh
goreleaser release --snapshot --clean
```

## Layout

```
main.go               flags, screen switching
internal/setup/       setup form screen
internal/timer/       session state machine (work/pause × cycles)
internal/orb/         braille orb renderer + big countdown digits
.goreleaser.yaml      release config (binaries + GitHub Releases)
```