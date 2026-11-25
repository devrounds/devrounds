package flow

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"unicode/utf8"

	_ "modernc.org/sqlite"
)

const (
	retention  = 90 * 24 * time.Hour
	maxCommand = 2000
)

// Record is one command a shell ran.
type Record struct {
	Time     time.Time
	Command  string
	Dir      string
	Project  string
	ExitCode int
	Duration time.Duration
}

// Store is the local command history. It lives in one SQLite file and is never sent anywhere.
type Store struct {
	db   *sql.DB
	Path string
}

// DefaultPath is where the history file lives. DEVROUNDS_FLOW_DB overrides it.
func DefaultPath() (string, error) {
	if p := os.Getenv("DEVROUNDS_FLOW_DB"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	var base string
	switch runtime.GOOS {
	case "windows":
		base = os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = filepath.Join(home, "AppData", "Local")
		}
	case "darwin":
		base = filepath.Join(home, "Library", "Application Support")
	default:
		base = os.Getenv("XDG_DATA_HOME")
		if base == "" {
			base = filepath.Join(home, ".local", "share")
		}
	}
	return filepath.Join(base, "devrounds", "flow.db"), nil
}

// Open opens the store, creating the folder, file and tables when they are missing.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	for _, stmt := range []string{
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
		`CREATE TABLE IF NOT EXISTS commands (
id INTEGER PRIMARY KEY AUTOINCREMENT,
ts INTEGER NOT NULL,
command TEXT NOT NULL,
dir TEXT NOT NULL,
project TEXT NOT NULL,
exit_code INTEGER NOT NULL,
duration_ms INTEGER NOT NULL
)`,
		"CREATE INDEX IF NOT EXISTS idx_commands_ts ON commands(ts)",
		"CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)",
	} {
		if _, err := db.Exec(stmt); err != nil {
			db.Close()
			return nil, fmt.Errorf("flow store: %w", err)
		}
	}
	_ = os.Chmod(path, 0o600)
	return &Store{db: db, Path: path}, nil
}

// Close closes the store.
func (s *Store) Close() error { return s.db.Close() }

// Enabled reports whether recording is on. It is on unless flow off was run.
func (s *Store) Enabled() bool {
	var v string
	err := s.db.QueryRow(`SELECT value FROM meta WHERE key = 'recording'`).Scan(&v)
	return err != nil || v != "off"
}

// SetEnabled turns recording on or off.
func (s *Store) SetEnabled(on bool) error {
	v := "on"
	if !on {
		v = "off"
	}
	_, err := s.db.Exec(`INSERT INTO meta(key, value) VALUES('recording', ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`, v)
	return err
}

// Add records a command. It returns false when the command was not recorded:
// recording is off, the command is one that must never be recorded, or it is
// older than the retention period. Secrets are removed before anything is stored.
func (s *Store) Add(r Record) (bool, error) {
	if !s.Enabled() {
		return false, nil
	}
	cmd, ok := Sanitize(r.Command)
	if !ok {
		return false, nil
	}
	if utf8.RuneCountInString(cmd) > maxCommand {
		cmd = string([]rune(cmd)[:maxCommand])
	}
	if r.Time.IsZero() {
		r.Time = time.Now()
	}
	cutoff := time.Now().Add(-retention)
	if r.Time.Before(cutoff) {
		return false, nil
	}
	if _, err := s.db.Exec(`DELETE FROM commands WHERE ts < ?`, cutoff.Unix()); err != nil {
		return false, err
	}
	_, err := s.db.Exec(
		`INSERT INTO commands(ts, command, dir, project, exit_code, duration_ms) VALUES(?, ?, ?, ?, ?, ?)`,
		r.Time.Unix(), cmd, r.Dir, r.Project, r.ExitCode, r.Duration.Milliseconds())
	if err != nil {
		return false, err
	}
	return true, nil
}

// Recent returns the newest commands first.
func (s *Store) Recent(limit int) ([]Record, error) {
	rows, err := s.db.Query(`SELECT ts, command, dir, project, exit_code, duration_ms
FROM commands ORDER BY ts DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		var ts, ms int64
		var r Record
		if err := rows.Scan(&ts, &r.Command, &r.Dir, &r.Project, &r.ExitCode, &ms); err != nil {
			return nil, err
		}
		r.Time = time.Unix(ts, 0)
		r.Duration = time.Duration(ms) * time.Millisecond
		out = append(out, r)
	}
	return out, rows.Err()
}

// Count returns how many commands are recorded.
func (s *Store) Count() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM commands`).Scan(&n)
	return n, err
}

// Forget deletes every recorded command.
func (s *Store) Forget() error {
	_, err := s.db.Exec(`DELETE FROM commands`)
	return err
}

// Retention is how long commands are kept.
func Retention() time.Duration { return retention }
