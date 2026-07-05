# Textbeard's Treasure

A Golang TUI game.

Control a pirate ship with `W`, `A`, `S`, and `D`.

- `W`: move forward in the current heading
- `S`: move backward opposite the current heading at half forward speed
- `A`: rotate the ship left by 45 degrees
- `D`: rotate the ship right by 45 degrees
- If no movement keys are pressed, the ship stays still.
- `1`: select cannonballs, or select rum while in port
- `2`: select grape shot, or select sugar while in port
- `3`: select tobacco while in port
- `Q`: fire the selected cannon load from the left side of the ship, with a 3-second cooldown
- `E`: fire the selected cannon load from the right side of the ship, with a 3-second cooldown
- `[` / `]`: decrease / increase port trade quantity while in port
- `B`: buy the selected good and quantity while in port
- `X`: sell the selected good and quantity while in port
- `R`: repair the ship to full hit points while in port for 25 gold
- `U`: buy the port's one-time ship upgrade while in port for 1000 gold


Music by Matthew Pablo


## Run the program

```sh
go run ./cmd/pirates
```

Press `Esc` or `Ctrl-C` to quit.

Music playback is best-effort through the first available seek-capable local
audio command: `ffplay`, `mpv`, or `play`. Cannon fire sounds can also use
`pw-play`, `paplay`, `ogg123`, `aplay`, or `canberra-gtk-play`. If none is
installed, the game continues silently.


## Terminal support

Works best with Ghostty


## Run the tests

```sh
go test ./...
```

## Run the Ghostty E2E test

This optional test launches the real Ghostty GUI, sends real key events with
`xdotool`, and checks ship movement telemetry. It is skipped by default.

Requirements:

- Linux with X11 and `DISPLAY` set
- `xdotool` installed

```sh
GHOSTTY_E2E=1 GHOSTTY_BIN=/path/to/ghostty go test ./internal/e2e -run Ghostty -count=1
```

