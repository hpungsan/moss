package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	// Use temp directory for test isolation
	tmpDir := t.TempDir()

	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer db.Close()

	// Verify database file was created
	dbPath := filepath.Join(tmpDir, "moss.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("database file not created at %s", dbPath)
	}

	// Verify exports directory was created
	exportsDir := filepath.Join(tmpDir, "exports")
	info, err := os.Stat(exportsDir)
	if os.IsNotExist(err) {
		t.Errorf("exports directory not created at %s", exportsDir)
	} else if !info.IsDir() {
		t.Errorf("exports path is not a directory")
	}

	// Verify WAL mode is active
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %s, want wal", journalMode)
	}

	// Verify schema was created by checking for capsules table
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='capsules'").Scan(&tableName)
	if err != nil {
		t.Fatalf("capsules table not found: %v", err)
	}
	if tableName != "capsules" {
		t.Errorf("table name = %s, want capsules", tableName)
	}
}

func TestInit_CreatesDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "nested", "path", ".moss")

	db, err := Init(baseDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer db.Close()

	// Verify nested directories were created
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		t.Errorf("base directory not created at %s", baseDir)
	}
}

func TestUserVersion(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer db.Close()

	// After Init, version should be CurrentSchemaVersion (migration ran)
	version, err := GetUserVersion(db)
	if err != nil {
		t.Fatalf("GetUserVersion() error = %v", err)
	}
	if version != CurrentSchemaVersion {
		t.Errorf("user_version after Init = %d, want %d", version, CurrentSchemaVersion)
	}

	// Test setting a higher version
	if err := SetUserVersion(db, 99); err != nil {
		t.Fatalf("SetUserVersion() error = %v", err)
	}

	// Verify version was set
	version, err = GetUserVersion(db)
	if err != nil {
		t.Fatalf("GetUserVersion() error = %v", err)
	}
	if version != 99 {
		t.Errorf("user_version = %d, want 99", version)
	}
}

func TestInit_MigrationIdempotent(t *testing.T) {
	tmpDir := t.TempDir()

	// First Init
	db1, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("first Init() error = %v", err)
	}
	db1.Close()

	// Second Init on same DB should succeed (migrations skip if already applied)
	db2, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("second Init() error = %v", err)
	}
	defer db2.Close()

	// Version should still be CurrentSchemaVersion
	version, err := GetUserVersion(db2)
	if err != nil {
		t.Fatalf("GetUserVersion() error = %v", err)
	}
	if version != CurrentSchemaVersion {
		t.Errorf("user_version after second Init = %d, want %d", version, CurrentSchemaVersion)
	}
}

func TestInit_SchemaIndexes(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer db.Close()

	// Verify all indexes were created
	indexes := []string{
		"idx_capsules_workspace_updated",
		"idx_capsules_workspace_name_norm",
		"idx_capsules_run_id",
		"idx_capsules_workspace_run_id",
		"idx_capsules_phase",
		"idx_capsules_role",
	}

	for _, idx := range indexes {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&name)
		if err != nil {
			t.Errorf("index %s not found: %v", idx, err)
		}
	}
}
