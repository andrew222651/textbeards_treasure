package audio

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCannonFireSoundIsEmbedded(t *testing.T) {
	data, err := embeddedAssets.ReadFile(cannonFireAsset)
	if err != nil {
		t.Fatalf("expected embedded cannon sound: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected embedded cannon sound to be non-empty")
	}
}

func TestCannonFirePlayerPreparesTemporarySoundFile(t *testing.T) {
	player := NewCannonFirePlayer()
	path, ok := player.soundPath()
	if !ok {
		t.Fatal("expected sound path to be prepared")
	}
	if filepath.Ext(path) != ".ogg" {
		t.Fatalf("expected .ogg temp file, got %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected temp sound file to be readable: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected temp sound file to be non-empty")
	}

	if err := player.Close(); err != nil {
		t.Fatalf("expected close to remove temp file: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected temp sound file to be removed, stat err=%v", err)
	}
}

func TestMusicTracksAreEmbedded(t *testing.T) {
	for _, asset := range []string{defaultMusicAsset, tavernMusicAsset} {
		data, err := embeddedAssets.ReadFile(asset)
		if err != nil {
			t.Fatalf("expected embedded music asset %s: %v", asset, err)
		}
		if len(data) == 0 {
			t.Fatalf("expected embedded music asset %s to be non-empty", asset)
		}
	}
}

func TestMusicPlayerPreparesTemporaryMusicFiles(t *testing.T) {
	player := NewMusicPlayer()
	defaultPath, ok := player.musicPath(defaultMusicTrack)
	if !ok {
		t.Fatal("expected default music path to be prepared")
	}
	if filepath.Ext(defaultPath) != ".wav" {
		t.Fatalf("expected .wav temp file, got %q", defaultPath)
	}
	tavernPath, ok := player.musicPath(tavernMusicTrack)
	if !ok {
		t.Fatal("expected tavern music path to be prepared")
	}
	if filepath.Ext(tavernPath) != ".ogg" {
		t.Fatalf("expected .ogg temp file, got %q", tavernPath)
	}

	for _, path := range []string{defaultPath, tavernPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected temp music file %q to be readable: %v", path, err)
		}
		if len(data) == 0 {
			t.Fatalf("expected temp music file %q to be non-empty", path)
		}
	}

	player.Stop()
	for _, path := range []string{defaultPath, tavernPath} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected temp music file %q to be removed, stat err=%v", path, err)
		}
	}
}

func TestMusicPlayerStartStopWithoutAudioCommand(t *testing.T) {
	t.Setenv("PATH", "")
	player := NewMusicPlayer()
	player.Start()
	player.EnterPort()
	player.LeavePort()
	player.Stop()
}

func TestMusicPlayerLoopsPlaybackCommandUntilStopped(t *testing.T) {
	callsPath := installFakeFFPlay(t)

	player := NewMusicPlayer()
	player.Start()
	defer player.Stop()

	waitForRecordedCalls(t, callsPath, func(calls string) bool {
		return strings.Count(calls, "pirates-default-music-") >= 2
	})
	player.Stop()
	if _, err := os.Stat(player.paths[defaultMusicTrack]); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected temp music file to be removed after stop, stat err=%v", err)
	}
}

func TestMusicPlayerSwitchesToTavernAndBack(t *testing.T) {
	callsPath := installFakeFFPlay(t)

	player := NewMusicPlayer()
	player.Start()
	defer player.Stop()
	waitForRecordedCalls(t, callsPath, func(calls string) bool {
		return strings.Contains(calls, "pirates-default-music-")
	})

	player.EnterPort()
	waitForRecordedCalls(t, callsPath, func(calls string) bool {
		return strings.Contains(calls, "pirates-tavern-music-")
	})

	player.LeavePort()
	waitForRecordedCalls(t, callsPath, func(calls string) bool {
		return strings.Count(calls, "pirates-default-music-") >= 2
	})
}

func TestMusicPlayerResumesInterruptedTrackFromLastOffset(t *testing.T) {
	callsPath := installBlockingFakeFFPlay(t)

	player := NewMusicPlayer()
	player.Start()
	defer player.Stop()
	waitForRecordedCalls(t, callsPath, func(calls string) bool {
		return strings.Contains(calls, "pirates-default-music-")
	})

	player.EnterPort()
	waitForRecordedCalls(t, callsPath, func(calls string) bool {
		return strings.Contains(calls, "pirates-tavern-music-")
	})
	waitForCondition(t, func() bool {
		return player.trackOffset(defaultMusicTrack) > 0
	})

	player.LeavePort()
	waitForRecordedCalls(t, callsPath, func(calls string) bool {
		for _, line := range strings.Split(calls, "\n") {
			if strings.Contains(line, "pirates-default-music-") && strings.Contains(line, "-ss") {
				return true
			}
		}
		return false
	})
}

func TestMusicCommandArgsIncludeSeekOffsets(t *testing.T) {
	offset := 1500 * time.Millisecond
	if got := strings.Join(musicCommands()[0].argsFor("song.wav", offset), " "); !strings.Contains(got, "-ss 1.500 song.wav") {
		t.Fatalf("expected ffplay seek args before path, got %q", got)
	}
	if got := strings.Join(musicCommands()[1].argsFor("song.wav", offset), " "); !strings.Contains(got, "--start=1.500 song.wav") {
		t.Fatalf("expected mpv seek args before path, got %q", got)
	}
	if got := strings.Join(musicCommands()[2].argsFor("song.wav", offset), " "); !strings.Contains(got, "song.wav trim 1.500") {
		t.Fatalf("expected SoX play seek args after path, got %q", got)
	}
}

func installFakeFFPlay(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	callsPath := filepath.Join(dir, "calls")
	fakeFFPlay := filepath.Join(dir, "ffplay")
	if err := os.WriteFile(fakeFFPlay, []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$CALLS_PATH\"\n"), 0o755); err != nil {
		t.Fatalf("write fake ffplay: %v", err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("CALLS_PATH", callsPath)
	return callsPath
}

func installBlockingFakeFFPlay(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	callsPath := filepath.Join(dir, "calls")
	blockPath := filepath.Join(dir, "block")
	if err := syscall.Mkfifo(blockPath, 0o600); err != nil {
		t.Fatalf("create blocking fifo: %v", err)
	}
	fakeFFPlay := filepath.Join(dir, "ffplay")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$CALLS_PATH\"\nread _ < \"$BLOCK_PATH\"\n"
	if err := os.WriteFile(fakeFFPlay, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake ffplay: %v", err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("CALLS_PATH", callsPath)
	t.Setenv("BLOCK_PATH", blockPath)
	return callsPath
}

func waitForRecordedCalls(t *testing.T, callsPath string, done func(string) bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, _ := os.ReadFile(callsPath)
		calls := string(data)
		if done(calls) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	data, _ := os.ReadFile(callsPath)
	t.Fatalf("timed out waiting for recorded calls, got %q", data)
}

func waitForCondition(t *testing.T, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if done() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timed out waiting for condition")
}
