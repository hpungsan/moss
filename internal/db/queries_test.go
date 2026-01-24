package db

import (
	"testing"
	"time"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/errors"
)

// newTestCapsule creates a capsule with default values for testing.
func newTestCapsule(id, workspaceRaw, text string) *capsule.Capsule {
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

// stringPtr returns a pointer to the given string.
func stringPtr(s string) *string {
	return &s
}

func TestInsertAndGetByID(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	c := newTestCapsule("01ABC123", "default", "Test content")
	c.NameRaw = stringPtr("test-capsule")
	c.NameNorm = stringPtr(capsule.Normalize("test-capsule"))
	c.Title = stringPtr("Test Title")
	c.Tags = []string{"tag1", "tag2"}
	c.Source = stringPtr("test")

	// Insert
	if err := Insert(db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// GetByID
	retrieved, err := GetByID(db, "01ABC123", false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	// Verify fields
	if retrieved.ID != c.ID {
		t.Errorf("ID = %q, want %q", retrieved.ID, c.ID)
	}
	if retrieved.WorkspaceRaw != c.WorkspaceRaw {
		t.Errorf("WorkspaceRaw = %q, want %q", retrieved.WorkspaceRaw, c.WorkspaceRaw)
	}
	if retrieved.WorkspaceNorm != c.WorkspaceNorm {
		t.Errorf("WorkspaceNorm = %q, want %q", retrieved.WorkspaceNorm, c.WorkspaceNorm)
	}
	if *retrieved.NameRaw != *c.NameRaw {
		t.Errorf("NameRaw = %q, want %q", *retrieved.NameRaw, *c.NameRaw)
	}
	if *retrieved.NameNorm != *c.NameNorm {
		t.Errorf("NameNorm = %q, want %q", *retrieved.NameNorm, *c.NameNorm)
	}
	if *retrieved.Title != *c.Title {
		t.Errorf("Title = %q, want %q", *retrieved.Title, *c.Title)
	}
	if retrieved.CapsuleText != c.CapsuleText {
		t.Errorf("CapsuleText = %q, want %q", retrieved.CapsuleText, c.CapsuleText)
	}
	if len(retrieved.Tags) != 2 || retrieved.Tags[0] != "tag1" {
		t.Errorf("Tags = %v, want %v", retrieved.Tags, c.Tags)
	}
	if *retrieved.Source != *c.Source {
		t.Errorf("Source = %q, want %q", *retrieved.Source, *c.Source)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	_, err = GetByID(db, "nonexistent", false)
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("GetByID should return ErrNotFound, got: %v", err)
	}
}

func TestGetByName(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	c := newTestCapsule("01DEF456", "MyWorkspace", "Content here")
	c.NameRaw = stringPtr("Auth System")
	c.NameNorm = stringPtr(capsule.Normalize("Auth System"))

	if err := Insert(db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// GetByName with normalized values
	retrieved, err := GetByName(db, "myworkspace", "auth system", false)
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}

	if retrieved.ID != c.ID {
		t.Errorf("ID = %q, want %q", retrieved.ID, c.ID)
	}
}

func TestGetByName_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	_, err = GetByName(db, "default", "nonexistent", false)
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("GetByName should return ErrNotFound, got: %v", err)
	}
}

func TestCheckNameExists(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Check non-existent name
	exists, err := CheckNameExists(db, "default", "auth")
	if err != nil {
		t.Fatalf("CheckNameExists failed: %v", err)
	}
	if exists {
		t.Error("CheckNameExists = true, want false")
	}

	// Insert capsule with name
	c := newTestCapsule("01GHI789", "default", "Content")
	c.NameRaw = stringPtr("auth")
	c.NameNorm = stringPtr("auth")
	if err := Insert(db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Check existing name
	exists, err = CheckNameExists(db, "default", "auth")
	if err != nil {
		t.Fatalf("CheckNameExists failed: %v", err)
	}
	if !exists {
		t.Error("CheckNameExists = false, want true")
	}
}

func TestUpdateByID(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert original
	c := newTestCapsule("01JKL012", "default", "Original content")
	c.NameRaw = stringPtr("update-test")
	c.NameNorm = stringPtr("update-test")
	c.Title = stringPtr("Original Title")
	c.Tags = []string{"old"}
	c.Source = stringPtr("original")
	if err := Insert(db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Modify capsule
	c.CapsuleText = "Updated content"
	c.CapsuleChars = capsule.CountChars(c.CapsuleText)
	c.TokensEstimate = capsule.EstimateTokens(c.CapsuleText)
	c.Title = stringPtr("Updated Title")
	c.Tags = []string{"new1", "new2"}
	c.Source = stringPtr("updated")

	// Update
	beforeUpdate := time.Now().Unix()
	if err := UpdateByID(db, c); err != nil {
		t.Fatalf("UpdateByID failed: %v", err)
	}

	// Verify updated_at is set to current time
	if c.UpdatedAt < beforeUpdate {
		t.Errorf("UpdatedAt = %d, should be >= %d", c.UpdatedAt, beforeUpdate)
	}

	// Retrieve and verify
	retrieved, err := GetByID(db, c.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.CapsuleText != "Updated content" {
		t.Errorf("CapsuleText = %q, want %q", retrieved.CapsuleText, "Updated content")
	}
	if *retrieved.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", *retrieved.Title, "Updated Title")
	}
	if len(retrieved.Tags) != 2 || retrieved.Tags[0] != "new1" {
		t.Errorf("Tags = %v, want [new1 new2]", retrieved.Tags)
	}

	// Verify ID was NOT changed
	if retrieved.ID != c.ID {
		t.Errorf("ID changed from %q to %q", c.ID, retrieved.ID)
	}
}

func TestUpdateByID_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	c := newTestCapsule("nonexistent", "default", "Content")
	err = UpdateByID(db, c)
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("UpdateByID should return ErrNotFound, got: %v", err)
	}
}

func TestSoftDelete(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert
	c := newTestCapsule("01MNO345", "default", "Content to delete")
	c.NameRaw = stringPtr("delete-test")
	c.NameNorm = stringPtr("delete-test")
	if err := Insert(db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Verify exists
	_, err = GetByID(db, c.ID, false)
	if err != nil {
		t.Fatalf("GetByID before delete failed: %v", err)
	}

	// Soft delete
	if err := SoftDelete(db, c.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Verify not found without includeDeleted
	_, err = GetByID(db, c.ID, false)
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("GetByID after delete should return ErrNotFound, got: %v", err)
	}

	// Verify found with includeDeleted
	retrieved, err := GetByID(db, c.ID, true)
	if err != nil {
		t.Fatalf("GetByID with includeDeleted failed: %v", err)
	}
	if retrieved.DeletedAt == nil {
		t.Error("DeletedAt should be set")
	}

	// Verify name slot is now free
	exists, err := CheckNameExists(db, "default", "delete-test")
	if err != nil {
		t.Fatalf("CheckNameExists failed: %v", err)
	}
	if exists {
		t.Error("Deleted capsule name should be available for reuse")
	}
}

func TestSoftDelete_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	err = SoftDelete(db, "nonexistent")
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("SoftDelete should return ErrNotFound, got: %v", err)
	}
}

func TestSoftDelete_AlreadyDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert and delete
	c := newTestCapsule("01PQR678", "default", "Content")
	if err := Insert(db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if err := SoftDelete(db, c.ID); err != nil {
		t.Fatalf("First SoftDelete failed: %v", err)
	}

	// Try to delete again
	err = SoftDelete(db, c.ID)
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("Second SoftDelete should return ErrNotFound, got: %v", err)
	}
}

func TestInsert_UnnamedCapsule(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Capsule without name
	c := newTestCapsule("01STU901", "default", "Unnamed content")
	// NameRaw and NameNorm are nil by default

	if err := Insert(db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Retrieve
	retrieved, err := GetByID(db, c.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.NameRaw != nil {
		t.Errorf("NameRaw = %v, want nil", retrieved.NameRaw)
	}
	if retrieved.NameNorm != nil {
		t.Errorf("NameNorm = %v, want nil", retrieved.NameNorm)
	}
}

func TestInsert_EmptyTags(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	c := newTestCapsule("01VWX234", "default", "Content")
	// Tags is nil/empty by default

	if err := Insert(db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	retrieved, err := GetByID(db, c.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if len(retrieved.Tags) != 0 {
		t.Errorf("Tags = %v, want empty", retrieved.Tags)
	}
}

func TestGetByName_IncludeDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	c := newTestCapsule("01YZA567", "default", "Content")
	c.NameRaw = stringPtr("deleted-test")
	c.NameNorm = stringPtr("deleted-test")
	if err := Insert(db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	if err := SoftDelete(db, c.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Should not find without includeDeleted
	_, err = GetByName(db, "default", "deleted-test", false)
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("GetByName should return ErrNotFound, got: %v", err)
	}

	// Should find with includeDeleted
	retrieved, err := GetByName(db, "default", "deleted-test", true)
	if err != nil {
		t.Fatalf("GetByName with includeDeleted failed: %v", err)
	}
	if retrieved.ID != c.ID {
		t.Errorf("ID = %q, want %q", retrieved.ID, c.ID)
	}
}

func TestGetByName_IncludeDeleted_PrefersActive(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Create a named capsule and delete it.
	old := newTestCapsule("01OLD567", "default", "Old content")
	old.NameRaw = stringPtr("reuse")
	old.NameNorm = stringPtr("reuse")
	if err := Insert(db, old); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if err := SoftDelete(db, old.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Create a new active capsule with the same name.
	newer := newTestCapsule("01NEW567", "default", "New content")
	newer.NameRaw = stringPtr("reuse")
	newer.NameNorm = stringPtr("reuse")
	if err := Insert(db, newer); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// includeDeleted:true should still return the active capsule (spec preference).
	retrieved, err := GetByName(db, "default", "reuse", true)
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}
	if retrieved.ID != newer.ID {
		t.Errorf("ID = %q, want %q (prefer active)", retrieved.ID, newer.ID)
	}
	if retrieved.DeletedAt != nil {
		t.Errorf("DeletedAt = %v, want nil (active capsule)", *retrieved.DeletedAt)
	}
}
