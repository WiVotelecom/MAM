// Package store provides a lightweight SQLite-backed configuration store for
// NetInsight. It keeps admin-defined connectivity targets so they persist
// across restarts. It uses a pure-Go SQLite driver so the resulting binary
// stays fully static and air-gap friendly.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Target is an admin-defined endpoint that NetInsight probes for reachability.
type Target struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Type      string    `json:"type"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("record not found")

// Store wraps a SQLite database handle.
type Store struct {
	db *sql.DB
}

// Open opens (and if necessary creates) the SQLite database at path and applies
// the schema. Use ":memory:" for an in-memory database (handy for tests).
func Open(path string) (*Store, error) {
	dsn := path
	if path != ":memory:" {
		dsn = path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// modernc's sqlite allows a single writer; keep the pool small to avoid
	// "database is locked" surprises for in-memory databases.
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS targets (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL,
    host       TEXT    NOT NULL,
    port       INTEGER NOT NULL DEFAULT 0,
    type       TEXT    NOT NULL DEFAULT 'tcp',
    enabled    INTEGER NOT NULL DEFAULT 1,
    created_at TEXT    NOT NULL
);`
	_, err := s.db.Exec(schema)
	return err
}

// AddTarget validates and inserts a new target, returning the stored record.
func (s *Store) AddTarget(t Target) (Target, error) {
	if err := ValidateTarget(t); err != nil {
		return Target{}, err
	}
	t.Type = strings.ToLower(strings.TrimSpace(t.Type))
	now := time.Now().UTC()
	res, err := s.db.Exec(
		`INSERT INTO targets (name, host, port, type, enabled, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		t.Name, t.Host, t.Port, t.Type, boolToInt(t.Enabled), now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Target{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Target{}, err
	}
	t.ID = id
	t.CreatedAt = now
	return t, nil
}

// ListTargets returns all configured targets ordered by id.
func (s *Store) ListTargets() ([]Target, error) {
	rows, err := s.db.Query(`SELECT id, name, host, port, type, enabled, created_at FROM targets ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Target
	for rows.Next() {
		var (
			t       Target
			enabled int
			created string
		)
		if err := rows.Scan(&t.ID, &t.Name, &t.Host, &t.Port, &t.Type, &enabled, &created); err != nil {
			return nil, err
		}
		t.Enabled = enabled != 0
		t.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, t)
	}
	return out, rows.Err()
}

// DeleteTarget removes a target by id.
func (s *Store) DeleteTarget(id int64) error {
	res, err := s.db.Exec(`DELETE FROM targets WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ValidateTarget checks the required fields of a target.
func ValidateTarget(t Target) error {
	if strings.TrimSpace(t.Name) == "" {
		return errors.New("target name is required")
	}
	if strings.TrimSpace(t.Host) == "" {
		return errors.New("target host is required")
	}
	if t.Port < 0 || t.Port > 65535 {
		return fmt.Errorf("invalid port %d", t.Port)
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
