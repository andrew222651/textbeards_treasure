package main

import (
	"fmt"
	"os"
	"time"

	"pirates/internal/audio"
	"pirates/internal/game"
	"pirates/internal/tui"
)

func main() {
	config := game.Config{}
	closeTelemetry, err := configureTelemetry(&config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if closeTelemetry != nil {
		defer func() { _ = closeTelemetry() }()
	}
	sound := audio.NewCannonFirePlayer()
	defer sound.Close()
	config.OnCannonFire = sound.Play
	music := audio.NewMusicPlayer()
	music.Start()
	defer music.Stop()
	config.OnPortStateChange = music.SetInPort

	g := game.New(config)
	if err := tui.Run(g); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func configureTelemetry(config *game.Config) (func() error, error) {
	path := os.Getenv("PIRATES_POSITION_LOG")
	if path == "" {
		return nil, nil
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open position telemetry log: %w", err)
	}

	config.OnMove = func(position game.Position) {
		_, _ = fmt.Fprintf(file, "%d %.6f %.6f\n", time.Now().UnixNano(), position.X, position.Y)
	}
	return file.Close, nil
}
