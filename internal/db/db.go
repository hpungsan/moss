package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Schema DDL from DESIGN.md §9
const schema = `
CREATE TABLE IF NOT EXISTS capsules (
  id              TEXT PRIMARY KEY,
  workspace_raw   TEXT NOT NULL,
  workspace_norm  TEXT NOT NULL,
  name_raw        TEXT,
  name_norm       TEXT,
  title           TEXT,
  capsule_text    TEXT NOT NULL,
  capsule_chars   INTEGER NOT NULL,
  tokens_estimate INTEGER NOT NULL,
  tags_json       TEXT,
  source          TEXT,
  created_at      INTEGER NOT NULL,
  updated_at      INTEGER NOT NULL,
  deleted_at      INTEGER
);

CREATE INDEX IF NOT EXISTS idx_capsules_workspace_updated
ON capsules(workspace_norm, updated_at DESC)
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_capsules_workspace_name_norm
ON capsules(workspace_norm, name_norm)
WHERE name_norm IS NOT NULL AND deleted_at IS NULL;
`

// Init initializes the SQLite database at baseDir/moss.db.
// The baseDir parameter allows tests to use t.TempDir() instead of ~/.moss.
func Init(baseDir string) (*sql.DB, error) {
	// Create base directory with restricted permissions
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}
	// Explicit chmod (best-effort, may not work on all platforms)
	_ = os.Chmod(baseDir, 0700)

	// Create exports subdirectory
	exportsDir := filepath.Join(baseDir, "exports")
	if err := os.MkdirAll(exportsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create exports directory: %w", err)
	}
	_ = os.Chmod(exportsDir, 0700)

	// Open database
	dbPath := filepath.Join(baseDir, "moss.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set file permissions (best-effort)
	_ = os.Chmod(dbPath, 0600)

	// Configure connection
	if err := configureDB(db); err != nil {
		db.Close()
		return nil, err
	}

	// Create schema
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return db, nil
}

// configureDB sets pragmas for WAL mode and busy timeout.
func configureDB(db *sql.DB) error {
	// Set busy timeout (5 seconds)
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return fmt.Errorf("failed to set busy_timeout: %w", err)
	}

	// Enable WAL mode
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Verify WAL mode is active
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		return fmt.Errorf("failed to verify journal mode: %w", err)
	}
	if journalMode != "wal" {
		return fmt.Errorf("expected WAL mode, got %s", journalMode)
	}

	return nil
}

// GetUserVersion returns the current schema version (user_version pragma).
func GetUserVersion(db *sql.DB) (int, error) {
	var version int
	if err := db.QueryRow("PRAGMA user_version;").Scan(&version); err != nil {
		return 0, fmt.Errorf("failed to get user_version: %w", err)
	}
	return version, nil
}

// SetUserVersion sets the schema version (user_version pragma).
func SetUserVersion(db *sql.DB, version int) error {
	_, err := db.Exec(fmt.Sprintf("PRAGMA user_version=%d", version))
	if err != nil {
		return fmt.Errorf("failed to set user_version: %w", err)
	}
	return nil
}
