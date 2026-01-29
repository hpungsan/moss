package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hpungsan/moss/internal/config"
	_ "modernc.org/sqlite"
)

// CurrentSchemaVersion is the latest schema version.
// Bump this when adding migrations.
const CurrentSchemaVersion = 1

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

	// Open database with pragmas in connection string (applies to all connections)
	dbPath := filepath.Join(baseDir, "moss.db")
	dsn := dbPath + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify WAL mode is active
	if err := verifyWALMode(db); err != nil {
		db.Close()
		return nil, err
	}

	// Run migrations (this creates the file if it doesn't exist)
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	// Set file permissions after file exists (best-effort)
	_ = os.Chmod(dbPath, 0600)

	return db, nil
}

// ConfigurePool applies connection pool settings from config.
// Only sets limits if explicitly configured (non-zero values).
// Call after Init if you need to tune pool behavior for contention.
func ConfigurePool(db *sql.DB, cfg *config.Config) {
	if cfg == nil {
		return
	}
	if cfg.DBMaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	}
	if cfg.DBMaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	}
}

// migrate applies schema migrations based on user_version.
func migrate(db *sql.DB) error {
	version, err := GetUserVersion(db)
	if err != nil {
		return err
	}

	// Migration 0 -> 1: Initial schema (v1)
	if version < 1 {
		schema := `
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
		  run_id          TEXT,
		  phase           TEXT,
		  role            TEXT,
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

		CREATE INDEX IF NOT EXISTS idx_capsules_run_id
		ON capsules(run_id, phase, role)
		WHERE run_id IS NOT NULL AND deleted_at IS NULL;

		CREATE INDEX IF NOT EXISTS idx_capsules_workspace_run_id
		ON capsules(workspace_norm, run_id, updated_at DESC)
		WHERE run_id IS NOT NULL AND deleted_at IS NULL;

		CREATE INDEX IF NOT EXISTS idx_capsules_phase
		ON capsules(phase)
		WHERE phase IS NOT NULL AND deleted_at IS NULL;

		CREATE INDEX IF NOT EXISTS idx_capsules_role
		ON capsules(role)
		WHERE role IS NOT NULL AND deleted_at IS NULL;
		`
		if _, err := db.Exec(schema); err != nil {
			return fmt.Errorf("migration 1 failed: %w", err)
		}
		if err := SetUserVersion(db, 1); err != nil {
			return err
		}
	}

	// Future migrations go here:
	// if version < 2 { ... }

	return nil
}

// verifyWALMode checks that WAL mode is active (set via connection string).
func verifyWALMode(db *sql.DB) error {
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
