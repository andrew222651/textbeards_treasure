package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const defaultGhosttyPath = "/tmp/Ghostty-1.3.1-x86_64.AppImage"

type positionSample struct {
	Time time.Time
	X    float64
	Y    float64
}

func TestGhosttyHeldWKeyMovesShipForwardUntilRelease(t *testing.T) {
	if os.Getenv("GHOSTTY_E2E") != "1" {
		t.Skip("set GHOSTTY_E2E=1 to run the Ghostty GUI end-to-end test")
	}
	if os.Getenv("DISPLAY") == "" {
		t.Skip("DISPLAY is required for the Ghostty GUI end-to-end test")
	}
	if _, err := exec.LookPath("xdotool"); err != nil {
		t.Skip("xdotool is required for the Ghostty GUI end-to-end test")
	}

	ghostty := os.Getenv("GHOSTTY_BIN")
	if ghostty == "" {
		ghostty = defaultGhosttyPath
	}
	if info, err := os.Stat(ghostty); err != nil || info.IsDir() {
		t.Fatalf("Ghostty executable %q is not available", ghostty)
	}

	repoRoot := findRepoRoot(t)
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "textbeards-treasure-e2e")
	positionLogPath := filepath.Join(tempDir, "position.log")
	title := fmt.Sprintf("textbeards-treasure-e2e-%d-%d", os.Getpid(), time.Now().UnixNano())

	build := exec.Command("go", "build", "-o", binPath, "./cmd/textbeards_treasure")
	build.Dir = repoRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build test binary: %v\n%s", err, output)
	}

	ghosttyCmd := exec.Command(
		ghostty,
		"--gtk-single-instance=false",
		"--title="+title,
		"--class="+title,
		"--window-width=80",
		"--window-height=24",
		"-e", "env", "TEXTBEARDS_TREASURE_POSITION_LOG="+positionLogPath, binPath,
	)
	ghosttyCmd.Dir = repoRoot
	if err := ghosttyCmd.Start(); err != nil {
		t.Fatalf("start Ghostty: %v", err)
	}
	defer func() {
		_ = ghosttyCmd.Process.Kill()
		_, _ = ghosttyCmd.Process.Wait()
	}()

	windowID := xdotoolOutput(t, 10*time.Second, "search", "--sync", "--name", title)
	windowID = firstLine(windowID)
	if windowID == "" {
		t.Fatal("xdotool did not find the Ghostty test window")
	}

	xdotool(t, 3*time.Second, "windowactivate", "--sync", windowID)
	xdotool(t, 3*time.Second, "windowfocus", "--sync", windowID)
	time.Sleep(300 * time.Millisecond)

	xdotool(t, 3*time.Second, "keydown", "w")
	holdStarted := time.Now()
	time.Sleep(700 * time.Millisecond)
	xdotool(t, 3*time.Second, "keyup", "w")
	keyUpCompleted := time.Now()
	time.Sleep(300 * time.Millisecond)

	xdotool(t, 3*time.Second, "key", "ctrl+c")
	done := make(chan error, 1)
	go func() { done <- ghosttyCmd.Wait() }()
	select {
	case <-time.After(3 * time.Second):
		t.Fatal("Ghostty did not exit after sending Ctrl-C")
	case err := <-done:
		if err != nil {
			t.Fatalf("Ghostty exited with error: %v", err)
		}
	}

	samples := readPositionSamples(t, positionLogPath)
	if len(samples) < 3 {
		t.Fatalf("expected at least 3 movement samples while holding W, got %d", len(samples))
	}

	if delay := samples[0].Time.Sub(holdStarted); delay > 200*time.Millisecond {
		t.Fatalf("expected movement to start promptly after W keydown; first sample delay was %s", delay)
	}

	first := samples[0]
	last := samples[len(samples)-1]
	if last.Y >= first.Y {
		t.Fatalf("expected held W to move ship forward; first sample %#v, last sample %#v", first, last)
	}
	if abs(last.X-first.X) > 0.25 {
		t.Fatalf("expected held W to keep X stable; first sample %#v, last sample %#v", first, last)
	}

	settleCutoff := keyUpCompleted.Add(150 * time.Millisecond)
	for _, sample := range samples {
		if sample.Time.After(settleCutoff) {
			t.Fatalf("expected movement to stop after key release; saw sample %s after keyup", sample.Time.Sub(keyUpCompleted))
		}
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root containing go.mod")
		}
		dir = parent
	}
}

func xdotool(t *testing.T, timeout time.Duration, args ...string) {
	t.Helper()
	_ = xdotoolOutput(t, timeout, args...)
}

func xdotoolOutput(t *testing.T, timeout time.Duration, args ...string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "xdotool", args...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("xdotool %s timed out", strings.Join(args, " "))
	}
	if err != nil {
		t.Fatalf("xdotool %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func readPositionSamples(t *testing.T, path string) []positionSample {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read position log: %v", err)
	}

	var samples []positionSample
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			t.Fatalf("expected position log line to have 3 fields, got %q", line)
		}

		timestamp, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			t.Fatalf("parse position timestamp %q: %v", fields[0], err)
		}
		x, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			t.Fatalf("parse position x %q: %v", fields[1], err)
		}
		y, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			t.Fatalf("parse position y %q: %v", fields[2], err)
		}

		samples = append(samples, positionSample{Time: time.Unix(0, timestamp), X: x, Y: y})
	}
	return samples
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return strings.TrimSpace(line)
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
