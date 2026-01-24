package ops

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

func newTestCapsuleForImport(id, workspaceRaw, text string) *capsule.Capsule {
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

func writeExportFile(t *testing.T, path string, records []capsule.ExportRecord) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create export file: %v", err)
	}
	defer file.Close()

	// Write header
	header := ExportHeader{
		MossExport:    true,
		SchemaVersion: "1.0",
		ExportedAt:    time.Now().Unix(),
	}
	headerJSON, _ := json.Marshal(header)
	if _, err := file.Write(headerJSON); err != nil {
		t.Fatalf("Failed to write header: %v", err)
	}
	if _, err := file.Write([]byte("\n")); err != nil {
		t.Fatalf("Failed to write newline: %v", err)
	}

	// Write records
	for _, record := range records {
		recordJSON, _ := json.Marshal(record)
		if _, err := file.Write(recordJSON); err != nil {
			t.Fatalf("Failed to write record: %v", err)
		}
		if _, err := file.Write([]byte("\n")); err != nil {
			t.Fatalf("Failed to write newline: %v", err)
		}
	}
}

func TestImport_HappyPath_ModeError(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Create export file with 2 capsules
	records := []capsule.ExportRecord{
		{
			ID:           "01IMP001",
			WorkspaceRaw: "default",
			CapsuleText:  "Content 1",
			CreatedAt:    1000,
			UpdatedAt:    1000,
		},
		{
			ID:           "01IMP002",
			WorkspaceRaw: "default",
			NameRaw:      stringPtr("named"),
			CapsuleText:  "Content 2",
			CreatedAt:    2000,
			UpdatedAt:    2000,
		},
	}

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	writeExportFile(t, exportPath, records)

	output, err := Import(database, ImportInput{
		Path: exportPath,
		Mode: ImportModeError,
	})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if output.Imported != 2 {
		t.Errorf("Imported = %d, want 2", output.Imported)
	}
	if len(output.Errors) != 0 {
		t.Errorf("Errors = %v, want empty", output.Errors)
	}

	// Verify capsules exist
	c1, err := db.GetByID(database, "01IMP001", false)
	if err != nil {
		t.Errorf("Capsule 1 should exist: %v", err)
	}
	if c1.CapsuleText != "Content 1" {
		t.Errorf("CapsuleText = %q, want 'Content 1'", c1.CapsuleText)
	}

	c2, err := db.GetByName(database, "default", "named", false)
	if err != nil {
		t.Errorf("Capsule 2 should exist by name: %v", err)
	}
	if c2.ID != "01IMP002" {
		t.Errorf("ID = %q, want 01IMP002", c2.ID)
	}
}

func TestImport_SkipsHeader(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	records := []capsule.ExportRecord{
		{
			ID:           "01IMP003",
			WorkspaceRaw: "default",
			CapsuleText:  "Content",
			CreatedAt:    1000,
			UpdatedAt:    1000,
		},
	}

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	writeExportFile(t, exportPath, records)

	output, err := Import(database, ImportInput{Path: exportPath})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if output.Imported != 1 {
		t.Errorf("Imported = %d, want 1 (header skipped)", output.Imported)
	}
}

func TestImport_RecomputesNorms(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Create record with wrong norms and metrics
	records := []capsule.ExportRecord{
		{
			ID:             "01IMP004",
			WorkspaceRaw:   "MyWorkspace",
			WorkspaceNorm:  "wrong-norm", // Will be recomputed
			NameRaw:        stringPtr("MyName"),
			NameNorm:       stringPtr("wrong"), // Will be recomputed
			CapsuleText:    "Hello world",
			CapsuleChars:   999, // Will be recomputed
			TokensEstimate: 999, // Will be recomputed
			CreatedAt:      1000,
			UpdatedAt:      1000,
		},
	}

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	writeExportFile(t, exportPath, records)

	_, err = Import(database, ImportInput{Path: exportPath})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	c, err := db.GetByID(database, "01IMP004", false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if c.WorkspaceNorm != "myworkspace" {
		t.Errorf("WorkspaceNorm = %q, want myworkspace", c.WorkspaceNorm)
	}
	if c.NameNorm == nil || *c.NameNorm != "myname" {
		t.Errorf("NameNorm = %v, want myname", c.NameNorm)
	}
	if c.CapsuleChars != 11 {
		t.Errorf("CapsuleChars = %d, want 11", c.CapsuleChars)
	}
}

func TestImport_ModeError_RollsBackOnIDCollision(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Pre-insert a capsule
	existing := newTestCapsuleForImport("01IMP005", "default", "Existing")
	if err := db.Insert(database, existing); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Try to import with same ID
	records := []capsule.ExportRecord{
		{
			ID:           "01IMP006",
			WorkspaceRaw: "default",
			CapsuleText:  "New 1",
			CreatedAt:    1000,
			UpdatedAt:    1000,
		},
		{
			ID:           "01IMP005", // Collision!
			WorkspaceRaw: "default",
			CapsuleText:  "Collision",
			CreatedAt:    2000,
			UpdatedAt:    2000,
		},
	}

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	writeExportFile(t, exportPath, records)

	output, err := Import(database, ImportInput{
		Path: exportPath,
		Mode: ImportModeError,
	})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Should have rolled back
	if output.Imported != 0 {
		t.Errorf("Imported = %d, want 0 (rolled back)", output.Imported)
	}
	if len(output.Errors) != 1 {
		t.Errorf("Errors = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Code != "ID_COLLISION" {
		t.Errorf("Error code = %q, want ID_COLLISION", output.Errors[0].Code)
	}

	// First capsule should NOT have been inserted (atomic rollback)
	_, err = db.GetByID(database, "01IMP006", false)
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("First capsule should not exist (rolled back): %v", err)
	}
}

func TestImport_ModeError_RollsBackOnNameCollision(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Pre-insert a named capsule
	existing := newTestCapsuleForImport("01IMP007", "default", "Existing")
	existing.NameRaw = stringPtr("taken")
	existing.NameNorm = stringPtr("taken")
	if err := db.Insert(database, existing); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Try to import with same name
	records := []capsule.ExportRecord{
		{
			ID:           "01IMP008",
			WorkspaceRaw: "default",
			NameRaw:      stringPtr("taken"), // Collision!
			CapsuleText:  "New with taken name",
			CreatedAt:    1000,
			UpdatedAt:    1000,
		},
	}

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	writeExportFile(t, exportPath, records)

	output, err := Import(database, ImportInput{
		Path: exportPath,
		Mode: ImportModeError,
	})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if output.Imported != 0 {
		t.Errorf("Imported = %d, want 0", output.Imported)
	}
	if len(output.Errors) != 1 || output.Errors[0].Code != "NAME_COLLISION" {
		t.Errorf("Expected NAME_COLLISION error, got: %v", output.Errors)
	}
}

func TestImport_ModeReplace_UpdatesOnIDCollision(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Pre-insert a capsule
	existing := newTestCapsuleForImport("01IMP009", "default", "Old content")
	if err := db.Insert(database, existing); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Import with same ID, new content
	records := []capsule.ExportRecord{
		{
			ID:           "01IMP009",
			WorkspaceRaw: "default",
			CapsuleText:  "Updated content",
			CreatedAt:    1000,
			UpdatedAt:    2000,
		},
	}

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	writeExportFile(t, exportPath, records)

	output, err := Import(database, ImportInput{
		Path: exportPath,
		Mode: ImportModeReplace,
	})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if output.Imported != 1 {
		t.Errorf("Imported = %d, want 1", output.Imported)
	}

	// Verify content was updated
	c, err := db.GetByID(database, "01IMP009", false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if c.CapsuleText != "Updated content" {
		t.Errorf("CapsuleText = %q, want 'Updated content'", c.CapsuleText)
	}
}

func TestImport_ModeReplace_UpdatesOnNameCollision(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Pre-insert a named capsule
	existing := newTestCapsuleForImport("01IMP010", "default", "Old content")
	existing.NameRaw = stringPtr("myname")
	existing.NameNorm = stringPtr("myname")
	if err := db.Insert(database, existing); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Import with different ID but same name
	records := []capsule.ExportRecord{
		{
			ID:           "01IMP011", // Different ID
			WorkspaceRaw: "default",
			NameRaw:      stringPtr("myname"), // Same name
			CapsuleText:  "New content",
			CreatedAt:    1000,
			UpdatedAt:    2000,
		},
	}

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	writeExportFile(t, exportPath, records)

	output, err := Import(database, ImportInput{
		Path: exportPath,
		Mode: ImportModeReplace,
	})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if output.Imported != 1 {
		t.Errorf("Imported = %d, want 1", output.Imported)
	}

	// Verify existing capsule was updated (ID preserved)
	c, err := db.GetByID(database, "01IMP010", false)
	if err != nil {
		t.Fatalf("Original ID should still exist: %v", err)
	}
	if c.CapsuleText != "New content" {
		t.Errorf("CapsuleText = %q, want 'New content'", c.CapsuleText)
	}

	// New ID should NOT exist
	_, err = db.GetByID(database, "01IMP011", false)
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("New ID should not exist: %v", err)
	}
}

func TestImport_ModeReplace_IgnoresDeletedNameCollision(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Pre-insert a named capsule, then soft-delete it.
	deleted := newTestCapsuleForImport("01IMP0D1", "default", "Old deleted content")
	deleted.NameRaw = stringPtr("myname")
	deleted.NameNorm = stringPtr("myname")
	if err := db.Insert(database, deleted); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if err := db.SoftDelete(database, deleted.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Import an active capsule with the same name but different ID.
	records := []capsule.ExportRecord{
		{
			ID:           "01IMP0D2", // Different ID
			WorkspaceRaw: "default",
			NameRaw:      stringPtr("myname"), // Same name as deleted capsule
			CapsuleText:  "New content",
			CreatedAt:    1000,
			UpdatedAt:    2000,
			DeletedAt:    nil, // Active
		},
	}

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	writeExportFile(t, exportPath, records)

	output, err := Import(database, ImportInput{
		Path: exportPath,
		Mode: ImportModeReplace,
	})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if output.Imported != 1 {
		t.Errorf("Imported = %d, want 1", output.Imported)
	}
	if output.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", output.Skipped)
	}

	// Deleted capsule should remain deleted.
	cDel, err := db.GetByID(database, "01IMP0D1", true)
	if err != nil {
		t.Fatalf("GetByID(deleted) failed: %v", err)
	}
	if cDel.DeletedAt == nil {
		t.Error("Deleted capsule should remain soft-deleted")
	}
	if cDel.CapsuleText != "Old deleted content" {
		t.Errorf("Deleted capsule text = %q, want %q", cDel.CapsuleText, "Old deleted content")
	}

	// New capsule ID should exist and be active.
	cNew, err := db.GetByID(database, "01IMP0D2", false)
	if err != nil {
		t.Fatalf("New ID should exist: %v", err)
	}
	if cNew.DeletedAt != nil {
		t.Error("New capsule should be active")
	}
	if cNew.CapsuleText != "New content" {
		t.Errorf("New capsule text = %q, want %q", cNew.CapsuleText, "New content")
	}
}

func TestImport_ModeReplace_ErrorsOnAmbiguousCollision(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Pre-insert two capsules
	c1 := newTestCapsuleForImport("01IMP012", "default", "Content 1")
	c1.NameRaw = stringPtr("name1")
	c1.NameNorm = stringPtr("name1")
	if err := db.Insert(database, c1); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	c2 := newTestCapsuleForImport("01IMP013", "default", "Content 2")
	c2.NameRaw = stringPtr("name2")
	c2.NameNorm = stringPtr("name2")
	if err := db.Insert(database, c2); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Import with ID matching c1 but name matching c2 (ambiguous!)
	records := []capsule.ExportRecord{
		{
			ID:           "01IMP012", // Matches c1
			WorkspaceRaw: "default",
			NameRaw:      stringPtr("name2"), // Matches c2
			CapsuleText:  "Ambiguous",
			CreatedAt:    1000,
			UpdatedAt:    2000,
		},
	}

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	writeExportFile(t, exportPath, records)

	output, err := Import(database, ImportInput{
		Path: exportPath,
		Mode: ImportModeReplace,
	})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if output.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", output.Skipped)
	}
	if len(output.Errors) != 1 || output.Errors[0].Code != "AMBIGUOUS_COLLISION" {
		t.Errorf("Expected AMBIGUOUS_COLLISION error, got: %v", output.Errors)
	}
}

func TestImport_ModeRename_AutoSuffixesName(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Pre-insert a named capsule
	existing := newTestCapsuleForImport("01IMP014", "default", "Existing")
	existing.NameRaw = stringPtr("auth")
	existing.NameNorm = stringPtr("auth")
	if err := db.Insert(database, existing); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Import with same name
	records := []capsule.ExportRecord{
		{
			ID:           "01IMP015",
			WorkspaceRaw: "default",
			NameRaw:      stringPtr("auth"), // Collision
			CapsuleText:  "New auth",
			CreatedAt:    1000,
			UpdatedAt:    2000,
		},
	}

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	writeExportFile(t, exportPath, records)

	output, err := Import(database, ImportInput{
		Path: exportPath,
		Mode: ImportModeRename,
	})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if output.Imported != 1 {
		t.Errorf("Imported = %d, want 1", output.Imported)
	}

	// Both capsules should exist
	_, err = db.GetByName(database, "default", "auth", false)
	if err != nil {
		t.Errorf("Original 'auth' should exist: %v", err)
	}

	renamed, err := db.GetByName(database, "default", "auth-1", false)
	if err != nil {
		t.Errorf("Renamed 'auth-1' should exist: %v", err)
	}
	if renamed.CapsuleText != "New auth" {
		t.Errorf("Renamed capsule has wrong content: %q", renamed.CapsuleText)
	}
}

func TestImport_ModeRename_GeneratesNewIDOnCollision(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Pre-insert a capsule
	existing := newTestCapsuleForImport("01IMP016", "default", "Existing")
	if err := db.Insert(database, existing); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Import with same ID
	records := []capsule.ExportRecord{
		{
			ID:           "01IMP016", // Collision
			WorkspaceRaw: "default",
			CapsuleText:  "New content",
			CreatedAt:    1000,
			UpdatedAt:    2000,
		},
	}

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	writeExportFile(t, exportPath, records)

	output, err := Import(database, ImportInput{
		Path: exportPath,
		Mode: ImportModeRename,
	})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if output.Imported != 1 {
		t.Errorf("Imported = %d, want 1", output.Imported)
	}

	// Original should still exist with old content
	original, err := db.GetByID(database, "01IMP016", false)
	if err != nil {
		t.Fatalf("Original should exist: %v", err)
	}
	if original.CapsuleText != "Existing" {
		t.Errorf("Original content should be preserved")
	}

	// There should now be 2 capsules
	summaries, total, err := db.ListByWorkspace(database, "default", 10, 0, false)
	if err != nil {
		t.Fatalf("ListByWorkspace failed: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}

	// Find the new one
	var newCapsule *capsule.CapsuleSummary
	for i := range summaries {
		if summaries[i].ID != "01IMP016" {
			newCapsule = &summaries[i]
			break
		}
	}
	if newCapsule == nil {
		t.Fatal("New capsule with generated ID should exist")
	}
	if len(newCapsule.ID) != 26 {
		t.Errorf("New ID should be valid ULID, got %q", newCapsule.ID)
	}
}

func TestImport_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	_, err = Import(database, ImportInput{
		Path: filepath.Join(tmpDir, "nonexistent.jsonl"),
	})
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("Import should return ErrNotFound, got: %v", err)
	}
}

func TestImport_MalformedJSON_ModeError(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Create file with malformed JSON
	exportPath := filepath.Join(tmpDir, "export.jsonl")
	file, err := os.Create(exportPath)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if _, err := file.WriteString(`{"_moss_export":true,"schema_version":"1.0","exported_at":1000}` + "\n"); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}
	if _, err := file.WriteString(`not valid json` + "\n"); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}
	file.Close()

	output, err := Import(database, ImportInput{
		Path: exportPath,
		Mode: ImportModeError,
	})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if output.Imported != 0 {
		t.Errorf("Imported = %d, want 0", output.Imported)
	}
	if len(output.Errors) != 1 || output.Errors[0].Code != "PARSE_ERROR" {
		t.Errorf("Expected PARSE_ERROR, got: %v", output.Errors)
	}
}

func TestImport_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Create original capsules
	c1 := newTestCapsuleForImport("01ROUND1", "default", "Content 1")
	c1.NameRaw = stringPtr("cap1")
	c1.NameNorm = stringPtr("cap1")
	c1.Title = stringPtr("Title 1")
	c1.Tags = []string{"tag1", "tag2"}
	c1.Source = stringPtr("test")
	c1.CreatedAt = 1000
	c1.UpdatedAt = 2000

	c2 := newTestCapsuleForImport("01ROUND2", "default", "Content 2")
	c2.CreatedAt = 3000
	c2.UpdatedAt = 4000

	for _, c := range []*capsule.Capsule{c1, c2} {
		if err := db.Insert(database, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	// Export
	exportPath := filepath.Join(tmpDir, "export.jsonl")
	exportOut, err := Export(database, ExportInput{Path: exportPath})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	if exportOut.Count != 2 {
		t.Errorf("Export count = %d, want 2", exportOut.Count)
	}

	// Delete original capsules
	if err := db.SoftDelete(database, c1.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}
	if err := db.SoftDelete(database, c2.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Purge to remove completely
	_, err = db.PurgeDeleted(database, nil, nil)
	if err != nil {
		t.Fatalf("PurgeDeleted failed: %v", err)
	}

	// Verify capsules are gone
	_, err = db.GetByID(database, c1.ID, true)
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("Capsule should be purged: %v", err)
	}

	// Import
	importOut, err := Import(database, ImportInput{Path: exportPath})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if importOut.Imported != 2 {
		t.Errorf("Import count = %d, want 2", importOut.Imported)
	}

	// Verify capsules are restored with correct data
	restored1, err := db.GetByID(database, c1.ID, false)
	if err != nil {
		t.Fatalf("Capsule 1 should be restored: %v", err)
	}
	if restored1.CapsuleText != "Content 1" {
		t.Errorf("CapsuleText = %q, want 'Content 1'", restored1.CapsuleText)
	}
	if restored1.NameRaw == nil || *restored1.NameRaw != "cap1" {
		t.Errorf("NameRaw = %v, want 'cap1'", restored1.NameRaw)
	}
	if restored1.Title == nil || *restored1.Title != "Title 1" {
		t.Errorf("Title = %v, want 'Title 1'", restored1.Title)
	}
	if len(restored1.Tags) != 2 || restored1.Tags[0] != "tag1" {
		t.Errorf("Tags = %v, want [tag1 tag2]", restored1.Tags)
	}
	// Timestamps should be preserved from export, not import time
	if restored1.CreatedAt != 1000 {
		t.Errorf("CreatedAt = %d, want 1000", restored1.CreatedAt)
	}
	if restored1.UpdatedAt != 2000 {
		t.Errorf("UpdatedAt = %d, want 2000", restored1.UpdatedAt)
	}

	restored2, err := db.GetByID(database, c2.ID, false)
	if err != nil {
		t.Fatalf("Capsule 2 should be restored: %v", err)
	}
	if restored2.CapsuleText != "Content 2" {
		t.Errorf("CapsuleText = %q, want 'Content 2'", restored2.CapsuleText)
	}
}

func TestImport_DefaultsToModeError(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Pre-insert a capsule
	existing := newTestCapsuleForImport("01IMP017", "default", "Existing")
	if err := db.Insert(database, existing); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Import with same ID, no mode specified
	records := []capsule.ExportRecord{
		{
			ID:           "01IMP017", // Collision
			WorkspaceRaw: "default",
			CapsuleText:  "New",
			CreatedAt:    1000,
			UpdatedAt:    1000,
		},
	}

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	writeExportFile(t, exportPath, records)

	output, err := Import(database, ImportInput{
		Path: exportPath,
		// Mode not specified, should default to error
	})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Should fail like mode:error
	if output.Imported != 0 {
		t.Errorf("Imported = %d, want 0 (default mode:error)", output.Imported)
	}
	if len(output.Errors) != 1 || output.Errors[0].Code != "ID_COLLISION" {
		t.Errorf("Expected ID_COLLISION error, got: %v", output.Errors)
	}
}

func TestImport_InvalidMode(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	_, err = Import(database, ImportInput{
		Path: filepath.Join(tmpDir, "any.jsonl"),
		Mode: "invalid",
	})
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("Import should return ErrInvalidRequest, got: %v", err)
	}
}

func TestImport_PathRequired(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	_, err = Import(database, ImportInput{})
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("Import should return ErrInvalidRequest, got: %v", err)
	}
}
