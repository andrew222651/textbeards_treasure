package main

import (
	"fmt"
	"os"
	"time"

	"pirates/internal/audio"
	"pirates/internal/game"
	"pirates/internal/score"
	"pirates/internal/tui"
)

func main() {
	config := game.Config{}
	scoreStore, err := configureScores(&config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer scoreStore.Close()

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
	repairSound := audio.NewRepairPlayer()
	defer repairSound.Close()
	config.OnRepair = repairSound.Play
	tradeSound := audio.NewTradePlayer()
	defer tradeSound.Close()
	config.OnTrade = tradeSound.Play
	splashSound := audio.NewSplashPlayer()
	defer splashSound.Close()
	config.OnEnemySunk = splashSound.Play
	music := audio.NewMusicPlayer()
	mute := audio.NewMuteController(music)
	sound.SetMutedFunc(mute.Muted)
	repairSound.SetMutedFunc(mute.Muted)
	tradeSound.SetMutedFunc(mute.Muted)
	splashSound.SetMutedFunc(mute.Muted)
	config.OnMuteChange = mute.SetMuted
	music.Start()
	defer music.Stop()
	config.OnPortStateChange = music.SetInPort

	g := game.New(config)
	if err := tui.Run(g); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func configureScores(config *game.Config) (*score.Store, error) {
	store, err := score.OpenDefault()
	if err != nil {
		return nil, fmt.Errorf("open high score database: %w", err)
	}
	highScore, err := store.HighScore()
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("read high score: %w", err)
	}

	config.HighScore = highScore
	config.OnScoreFinalized = func(scoreValue int) (int, error) {
		highScore, _, err := store.SaveIfRecord(scoreValue)
		return highScore, err
	}
	return store, nil
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
