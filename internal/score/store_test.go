package score

import (
	"path/filepath"
	"testing"
)

func TestDefaultDatabasePathUsesEnvOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scores.sqlite")
	t.Setenv(EnvDatabasePath, path)

	got, err := DefaultDatabasePath()
	if err != nil {
		t.Fatalf("default database path: %v", err)
	}
	if got != path {
		t.Fatalf("expected env database path %q, got %q", path, got)
	}
}

func TestDefaultDatabasePathUsesTextbeardsTreasureConfigDir(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv(EnvDatabasePath, "")
	t.Setenv("XDG_CONFIG_HOME", configDir)

	got, err := DefaultDatabasePath()
	if err != nil {
		t.Fatalf("default database path: %v", err)
	}

	want := filepath.Join(configDir, "textbeards_treasure", "high_scores.sqlite")
	if got != want {
		t.Fatalf("expected default database path %q, got %q", want, got)
	}
}

func TestStoreStartsWithNoHighScore(t *testing.T) {
	store := openTestStore(t)
	defer store.Close()

	highScore, err := store.HighScore()
	if err != nil {
		t.Fatalf("read high score: %v", err)
	}
	if highScore != 0 {
		t.Fatalf("expected no high score to read as 0, got %d", highScore)
	}
}

func TestSaveIfRecordOnlyUpdatesNewRecords(t *testing.T) {
	store := openTestStore(t)
	defer store.Close()

	highScore, saved, err := store.SaveIfRecord(100)
	if err != nil {
		t.Fatalf("save first score: %v", err)
	}
	if !saved || highScore != 100 {
		t.Fatalf("expected first score to become high score, saved=%v high=%d", saved, highScore)
	}

	highScore, saved, err = store.SaveIfRecord(80)
	if err != nil {
		t.Fatalf("save lower score: %v", err)
	}
	if saved || highScore != 100 {
		t.Fatalf("expected lower score not to replace high score, saved=%v high=%d", saved, highScore)
	}

	highScore, saved, err = store.SaveIfRecord(125)
	if err != nil {
		t.Fatalf("save higher score: %v", err)
	}
	if !saved || highScore != 125 {
		t.Fatalf("expected higher score to replace high score, saved=%v high=%d", saved, highScore)
	}
}

func TestStorePersistsHighScoreOnDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scores.sqlite")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open score store: %v", err)
	}
	if _, _, err := store.SaveIfRecord(140); err != nil {
		t.Fatalf("save score: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close score store: %v", err)
	}

	store, err = Open(path)
	if err != nil {
		t.Fatalf("reopen score store: %v", err)
	}
	defer store.Close()

	highScore, err := store.HighScore()
	if err != nil {
		t.Fatalf("read persisted high score: %v", err)
	}
	if highScore != 140 {
		t.Fatalf("expected persisted high score 140, got %d", highScore)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "scores.sqlite"))
	if err != nil {
		t.Fatalf("open score store: %v", err)
	}
	return store
}
