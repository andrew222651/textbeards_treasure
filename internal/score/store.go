package score

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const EnvDatabasePath = "PIRATES_SCORE_DB"

type Store struct {
	db *sql.DB
}

func DefaultDatabasePath() (string, error) {
	if path := os.Getenv(EnvDatabasePath); path != "" {
		return path, nil
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	return filepath.Join(dir, "pirates", "high_scores.sqlite"), nil
}

func OpenDefault() (*Store, error) {
	path, err := DefaultDatabasePath()
	if err != nil {
		return nil, err
	}
	return Open(path)
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("score database path is empty")
	}
	if shouldCreateParentDirectory(path) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("create score database directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open score database: %w", err)
	}
	store := &Store{db: db}
	if err := store.init(); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

func shouldCreateParentDirectory(path string) bool {
	return path != ":memory:" && !strings.HasPrefix(path, "file:")
}

func (s *Store) init() error {
	if s == nil || s.db == nil {
		return errors.New("score store is not open")
	}
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS high_scores (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			score INTEGER NOT NULL CHECK (score >= 0),
			recorded_at TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("initialize score database: %w", err)
	}
	return nil
}

func (s *Store) HighScore() (int, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("score store is not open")
	}

	var highScore int
	err := s.db.QueryRow(`SELECT score FROM high_scores WHERE id = 1`).Scan(&highScore)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read high score: %w", err)
	}
	return highScore, nil
}

func (s *Store) SaveIfRecord(score int) (int, bool, error) {
	if s == nil || s.db == nil {
		return 0, false, errors.New("score store is not open")
	}
	if score < 0 {
		score = 0
	}

	result, err := s.db.Exec(`
		INSERT INTO high_scores (id, score, recorded_at)
		VALUES (1, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			score = excluded.score,
			recorded_at = excluded.recorded_at
		WHERE excluded.score > high_scores.score
	`, score)
	if err != nil {
		return 0, false, fmt.Errorf("save high score: %w", err)
	}

	highScore, err := s.HighScore()
	if err != nil {
		return 0, false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return highScore, highScore == score, nil
	}
	return highScore, rowsAffected > 0, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
