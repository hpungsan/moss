package ops

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

func newTestCapsuleForExport(id, workspaceRaw, text string) *capsule.Capsule {
	now := time.Now().Unix()
	return &capsule.Capsule{
		ID:             id,
		WorkspaceRaw:   workspaceRaw,
		WorkspaceNorm:  capsule.Normalize(workspaceRaw),
		CapsuleText:    text,
		CapsuleChars:   capsule.CountChars(text),
		TokensEstimate: capsule.EstimateTokens(text),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func TestExport_HappyPath(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Create capsules
	c1 := newTestCapsuleForExport("01EXP001", "default", "Content 1")
	c1.CreatedAt = 1000
	c2 := newTestCapsuleForExport("01EXP002", "default", "Content 2")
	c2.CreatedAt = 2000

	for _, c := range []*capsule.Capsule{c1, c2} {
		if err := db.Insert(database, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	output, err := Export(database, ExportInput{Path: exportPath})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if output.Path != exportPath {
		t.Errorf("Path = %q, want %q", output.Path, exportPath)
	}
	if output.Count != 2 {
		t.Errorf("Count = %d, want 2", output.Count)
	}
	if output.ExportedAt == 0 {
		t.Error("ExportedAt should be set")
	}

	// Verify file contents
	file, err := os.Open(exportPath)
	if err != nil {
		t.Fatalf("Failed to open export file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}

	// Should have header + 2 capsules = 3 lines
	if lines != 3 {
		t.Errorf("lines = %d, want 3 (header + 2 capsules)", lines)
	}
}

func TestExport_HeaderLine(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	output, err := Export(database, ExportInput{Path: exportPath})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Read first line (header)
	file, err := os.Open(exportPath)
	if err != nil {
		t.Fatalf("Failed to open export file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatal("Failed to read header line")
	}

	var header ExportHeader
	if err := json.Unmarshal(scanner.Bytes(), &header); err != nil {
		t.Fatalf("Failed to parse header: %v", err)
	}

	if !header.MossExport {
		t.Error("_moss_export should be true")
	}
	if header.SchemaVersion != "1.0" {
		t.Errorf("schema_version = %q, want 1.0", header.SchemaVersion)
	}
	if header.ExportedAt != output.ExportedAt {
		t.Errorf("exported_at = %d, want %d", header.ExportedAt, output.ExportedAt)
	}
}

func TestExport_JSONLFormat(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	c := newTestCapsuleForExport("01EXP003", "default", "Test content")
	c.NameRaw = stringPtr("test-name")
	c.NameNorm = stringPtr("test-name")
	c.Title = stringPtr("Test Title")
	c.Tags = []string{"tag1", "tag2"}
	c.Source = stringPtr("test")

	if err := db.Insert(database, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	_, err = Export(database, ExportInput{Path: exportPath})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Read second line (first capsule)
	file, err := os.Open(exportPath)
	if err != nil {
		t.Fatalf("Failed to open export file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan() // Skip header
	if !scanner.Scan() {
		t.Fatal("Failed to read capsule line")
	}

	var record capsule.ExportRecord
	if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
		t.Fatalf("Failed to parse capsule: %v", err)
	}

	if record.ID != c.ID {
		t.Errorf("ID = %q, want %q", record.ID, c.ID)
	}
	if record.WorkspaceRaw != c.WorkspaceRaw {
		t.Errorf("WorkspaceRaw = %q, want %q", record.WorkspaceRaw, c.WorkspaceRaw)
	}
	if record.NameRaw == nil || *record.NameRaw != "test-name" {
		t.Errorf("NameRaw = %v, want test-name", record.NameRaw)
	}
	if len(record.Tags) != 2 || record.Tags[0] != "tag1" {
		t.Errorf("Tags = %v, want [tag1 tag2]", record.Tags)
	}
}

func TestExport_WorkspaceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	c1 := newTestCapsuleForExport("01EXP004", "target", "In target")
	c2 := newTestCapsuleForExport("01EXP005", "other", "In other")

	for _, c := range []*capsule.Capsule{c1, c2} {
		if err := db.Insert(database, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	ws := "target"
	output, err := Export(database, ExportInput{
		Path:      exportPath,
		Workspace: &ws,
	})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if output.Count != 1 {
		t.Errorf("Count = %d, want 1", output.Count)
	}
}

func TestExport_IncludeDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	c1 := newTestCapsuleForExport("01EXP006", "default", "Active")
	c2 := newTestCapsuleForExport("01EXP007", "default", "Deleted")

	for _, c := range []*capsule.Capsule{c1, c2} {
		if err := db.Insert(database, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}
	if err := db.SoftDelete(database, c2.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Without includeDeleted
	exportPath1 := filepath.Join(tmpDir, "export1.jsonl")
	output1, err := Export(database, ExportInput{
		Path:           exportPath1,
		IncludeDeleted: false,
	})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	if output1.Count != 1 {
		t.Errorf("without includeDeleted: Count = %d, want 1", output1.Count)
	}

	// With includeDeleted
	exportPath2 := filepath.Join(tmpDir, "export2.jsonl")
	output2, err := Export(database, ExportInput{
		Path:           exportPath2,
		IncludeDeleted: true,
	})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	if output2.Count != 2 {
		t.Errorf("with includeDeleted: Count = %d, want 2", output2.Count)
	}
}

func TestExport_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	output, err := Export(database, ExportInput{Path: exportPath})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if output.Count != 0 {
		t.Errorf("Count = %d, want 0", output.Count)
	}

	// File should still exist with just header
	file, err := os.Open(exportPath)
	if err != nil {
		t.Fatalf("Failed to open export file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	if lines != 1 {
		t.Errorf("lines = %d, want 1 (header only)", lines)
	}
}

func TestExport_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	_, err = Export(database, ExportInput{Path: exportPath})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	info, err := os.Stat(exportPath)
	if err != nil {
		t.Fatalf("Failed to stat export file: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestExport_DefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Override HOME for test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	output, err := Export(database, ExportInput{})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Path should be under ~/.moss/exports/
	expectedDir := filepath.Join(tmpDir, ".moss", "exports")
	if !strings.HasPrefix(output.Path, expectedDir) {
		t.Errorf("Path = %q, should start with %q", output.Path, expectedDir)
	}

	// Path should contain "all-" for unfiltered export
	if !strings.Contains(filepath.Base(output.Path), "all-") {
		t.Errorf("Path = %q, should contain 'all-'", output.Path)
	}

	// File should exist
	if _, err := os.Stat(output.Path); os.IsNotExist(err) {
		t.Error("Export file should exist at default path")
	}
}

func TestExport_DefaultPathWithWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Override HOME for test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	ws := "MyWorkspace"
	output, err := Export(database, ExportInput{Workspace: &ws})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Path should contain normalized workspace name
	if !strings.Contains(filepath.Base(output.Path), "myworkspace-") {
		t.Errorf("Path = %q, should contain 'myworkspace-'", output.Path)
	}
}

func TestExport_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Export to a nested path that doesn't exist
	exportPath := filepath.Join(tmpDir, "nested", "dir", "export.jsonl")
	_, err = Export(database, ExportInput{Path: exportPath})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// File should exist
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Error("Export file should exist")
	}
}

func TestExport_PreservesOrder(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Create capsules with specific created_at order
	c1 := newTestCapsuleForExport("01EXP008", "default", "First")
	c1.CreatedAt = 1000
	c2 := newTestCapsuleForExport("01EXP009", "default", "Second")
	c2.CreatedAt = 2000
	c3 := newTestCapsuleForExport("01EXP00A", "default", "Third")
	c3.CreatedAt = 3000

	// Insert in random order
	for _, c := range []*capsule.Capsule{c3, c1, c2} {
		if err := db.Insert(database, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	_, err = Export(database, ExportInput{Path: exportPath})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Read capsules and verify order
	file, err := os.Open(exportPath)
	if err != nil {
		t.Fatalf("Failed to open export file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan() // Skip header

	var ids []string
	for scanner.Scan() {
		var record capsule.ExportRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatalf("Failed to parse capsule: %v", err)
		}
		ids = append(ids, record.ID)
	}

	// Should be ordered by created_at ASC
	if len(ids) != 3 || ids[0] != "01EXP008" || ids[1] != "01EXP009" || ids[2] != "01EXP00A" {
		t.Errorf("IDs = %v, want [01EXP008 01EXP009 01EXP00A] (created_at order)", ids)
	}
}

func TestExport_PathTraversalRejected(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	tests := []struct {
		name string
		path string
	}{
		{"traversal with ..", "/tmp/../../../etc/cron.d/malicious.jsonl"},
		{"relative traversal", "../../../etc/passwd.jsonl"},
		{"hidden traversal", "/tmp/safe/../../etc/shadow.jsonl"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Export(database, ExportInput{Path: tc.path})
			if err == nil {
				t.Error("Expected error for path traversal, got nil")
			}
			if !errors.Is(err, errors.ErrInvalidRequest) {
				t.Errorf("Expected ErrInvalidRequest, got: %v", err)
			}
		})
	}
}

func TestExport_RequiresJSONLExtension(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	_, err = Export(database, ExportInput{Path: filepath.Join(tmpDir, "export.txt")})
	if err == nil {
		t.Error("Expected error for non-.jsonl extension, got nil")
	}
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("Expected ErrInvalidRequest, got: %v", err)
	}
}
