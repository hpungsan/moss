package db

import (
	"context"
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
	if err := Insert(context.Background(), db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// GetByID
	retrieved, err := GetByID(context.Background(), db, "01ABC123", false)
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

	_, err = GetByID(context.Background(), db, "nonexistent", false)
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

	if err := Insert(context.Background(), db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// GetByName with normalized values
	retrieved, err := GetByName(context.Background(), db, "myworkspace", "auth system", false)
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

	_, err = GetByName(context.Background(), db, "default", "nonexistent", false)
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
	exists, err := CheckNameExists(context.Background(), db, "default", "auth")
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
	if err := Insert(context.Background(), db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Check existing name
	exists, err = CheckNameExists(context.Background(), db, "default", "auth")
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
	if err := Insert(context.Background(), db, c); err != nil {
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
	if err := UpdateByID(context.Background(), db, c); err != nil {
		t.Fatalf("UpdateByID failed: %v", err)
	}

	// Verify updated_at is set to current time
	if c.UpdatedAt < beforeUpdate {
		t.Errorf("UpdatedAt = %d, should be >= %d", c.UpdatedAt, beforeUpdate)
	}

	// Retrieve and verify
	retrieved, err := GetByID(context.Background(), db, c.ID, false)
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
	err = UpdateByID(context.Background(), db, c)
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
	if err := Insert(context.Background(), db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Verify exists
	_, err = GetByID(context.Background(), db, c.ID, false)
	if err != nil {
		t.Fatalf("GetByID before delete failed: %v", err)
	}

	// Soft delete
	if err := SoftDelete(context.Background(), db, c.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Verify not found without includeDeleted
	_, err = GetByID(context.Background(), db, c.ID, false)
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("GetByID after delete should return ErrNotFound, got: %v", err)
	}

	// Verify found with includeDeleted
	retrieved, err := GetByID(context.Background(), db, c.ID, true)
	if err != nil {
		t.Fatalf("GetByID with includeDeleted failed: %v", err)
	}
	if retrieved.DeletedAt == nil {
		t.Error("DeletedAt should be set")
	}

	// Verify name slot is now free
	exists, err := CheckNameExists(context.Background(), db, "default", "delete-test")
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

	err = SoftDelete(context.Background(), db, "nonexistent")
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
	if err := Insert(context.Background(), db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if err := SoftDelete(context.Background(), db, c.ID); err != nil {
		t.Fatalf("First SoftDelete failed: %v", err)
	}

	// Try to delete again
	err = SoftDelete(context.Background(), db, c.ID)
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

	if err := Insert(context.Background(), db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Retrieve
	retrieved, err := GetByID(context.Background(), db, c.ID, false)
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

	if err := Insert(context.Background(), db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	retrieved, err := GetByID(context.Background(), db, c.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if len(retrieved.Tags) != 0 {
		t.Errorf("Tags = %v, want empty", retrieved.Tags)
	}
}

func TestInsert_UniqueConstraint(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert first capsule with name
	c1 := newTestCapsule("01FIRST1", "default", "First content")
	c1.NameRaw = stringPtr("unique-name")
	c1.NameNorm = stringPtr("unique-name")
	if err := Insert(context.Background(), db, c1); err != nil {
		t.Fatalf("First Insert failed: %v", err)
	}

	// Try to insert second capsule with same name (different ID)
	c2 := newTestCapsule("01SECOND", "default", "Second content")
	c2.NameRaw = stringPtr("unique-name")
	c2.NameNorm = stringPtr("unique-name")
	err = Insert(context.Background(), db, c2)

	// Should return NAME_ALREADY_EXISTS (409) for named capsules
	if !errors.Is(err, errors.ErrNameAlreadyExists) {
		t.Errorf("Insert should return ErrNameAlreadyExists, got: %v", err)
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
	if err := Insert(context.Background(), db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	if err := SoftDelete(context.Background(), db, c.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Should not find without includeDeleted
	_, err = GetByName(context.Background(), db, "default", "deleted-test", false)
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("GetByName should return ErrNotFound, got: %v", err)
	}

	// Should find with includeDeleted
	retrieved, err := GetByName(context.Background(), db, "default", "deleted-test", true)
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
	if err := Insert(context.Background(), db, old); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if err := SoftDelete(context.Background(), db, old.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Create a new active capsule with the same name.
	newer := newTestCapsule("01NEW567", "default", "New content")
	newer.NameRaw = stringPtr("reuse")
	newer.NameNorm = stringPtr("reuse")
	if err := Insert(context.Background(), db, newer); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// includeDeleted:true should still return the active capsule (spec preference).
	retrieved, err := GetByName(context.Background(), db, "default", "reuse", true)
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

// =============================================================================
// ListByWorkspace Tests
// =============================================================================

func TestListByWorkspace_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert 3 capsules in "default" workspace
	for i, id := range []string{"01AAA001", "01AAA002", "01AAA003"} {
		c := newTestCapsule(id, "default", "Content "+id)
		c.NameRaw = stringPtr("cap-" + id)
		c.NameNorm = stringPtr("cap-" + id)
		c.UpdatedAt = int64(1000 + i) // Ensure ordering
		if err := Insert(context.Background(), db, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	// Insert 1 capsule in different workspace
	other := newTestCapsule("01BBB001", "other", "Other content")
	if err := Insert(context.Background(), db, other); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// List default workspace
	summaries, total, err := ListByWorkspace(context.Background(), db, "default", ListFilters{}, 10, 0, false)
	if err != nil {
		t.Fatalf("ListByWorkspace failed: %v", err)
	}

	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(summaries) != 3 {
		t.Errorf("len(summaries) = %d, want 3", len(summaries))
	}

	// Verify ordering (most recent first)
	if summaries[0].ID != "01AAA003" {
		t.Errorf("first summary ID = %q, want 01AAA003", summaries[0].ID)
	}
}

func TestListByWorkspace_Pagination(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert 5 capsules
	for i := 0; i < 5; i++ {
		id := "01CCC00" + string(rune('1'+i))
		c := newTestCapsule(id, "default", "Content")
		c.UpdatedAt = int64(1000 + i)
		if err := Insert(context.Background(), db, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	// Get first page (limit 2)
	page1, total, err := ListByWorkspace(context.Background(), db, "default", ListFilters{}, 2, 0, false)
	if err != nil {
		t.Fatalf("ListByWorkspace page 1 failed: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(page1) != 2 {
		t.Errorf("page1 len = %d, want 2", len(page1))
	}

	// Get second page (offset 2)
	page2, _, err := ListByWorkspace(context.Background(), db, "default", ListFilters{}, 2, 2, false)
	if err != nil {
		t.Fatalf("ListByWorkspace page 2 failed: %v", err)
	}
	if len(page2) != 2 {
		t.Errorf("page2 len = %d, want 2", len(page2))
	}

	// Verify no overlap
	if page1[0].ID == page2[0].ID {
		t.Error("pages should not overlap")
	}
}

func TestListByWorkspace_StableOrdering(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert capsules with same updated_at but different IDs
	sameTime := int64(1000)
	ids := []string{"01DDD003", "01DDD001", "01DDD002"} // Not in order
	for _, id := range ids {
		c := newTestCapsule(id, "default", "Content")
		c.UpdatedAt = sameTime
		if err := Insert(context.Background(), db, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	summaries, _, err := ListByWorkspace(context.Background(), db, "default", ListFilters{}, 10, 0, false)
	if err != nil {
		t.Fatalf("ListByWorkspace failed: %v", err)
	}

	// Should be ordered by ID DESC when updated_at is same
	if summaries[0].ID != "01DDD003" {
		t.Errorf("first ID = %q, want 01DDD003", summaries[0].ID)
	}
	if summaries[1].ID != "01DDD002" {
		t.Errorf("second ID = %q, want 01DDD002", summaries[1].ID)
	}
	if summaries[2].ID != "01DDD001" {
		t.Errorf("third ID = %q, want 01DDD001", summaries[2].ID)
	}
}

func TestListByWorkspace_IncludeDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert and delete one capsule
	c := newTestCapsule("01EEE001", "default", "Content")
	if err := Insert(context.Background(), db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if err := SoftDelete(context.Background(), db, c.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Insert active capsule
	active := newTestCapsule("01EEE002", "default", "Active")
	if err := Insert(context.Background(), db, active); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Without includeDeleted
	_, total, err := ListByWorkspace(context.Background(), db, "default", ListFilters{}, 10, 0, false)
	if err != nil {
		t.Fatalf("ListByWorkspace failed: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}

	// With includeDeleted
	summaries, total, err := ListByWorkspace(context.Background(), db, "default", ListFilters{}, 10, 0, true)
	if err != nil {
		t.Fatalf("ListByWorkspace failed: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(summaries) != 2 {
		t.Errorf("len(summaries) = %d, want 2", len(summaries))
	}
}

func TestListByWorkspace_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	summaries, total, err := ListByWorkspace(context.Background(), db, "nonexistent", ListFilters{}, 10, 0, false)
	if err != nil {
		t.Fatalf("ListByWorkspace failed: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(summaries) != 0 {
		t.Errorf("len(summaries) = %d, want 0", len(summaries))
	}
}

// =============================================================================
// ListAll Tests
// =============================================================================

func TestListAll_NoFilters(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert capsules in different workspaces
	for i, ws := range []string{"ws1", "ws2", "ws3"} {
		c := newTestCapsule("01FFF00"+string(rune('1'+i)), ws, "Content")
		c.UpdatedAt = int64(1000 + i)
		if err := Insert(context.Background(), db, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	summaries, total, err := ListAll(context.Background(), db, InventoryFilters{}, 10, 0, false)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(summaries) != 3 {
		t.Errorf("len(summaries) = %d, want 3", len(summaries))
	}
}

func TestListAll_WorkspaceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert capsules
	c1 := newTestCapsule("01GGG001", "alpha", "Content")
	c2 := newTestCapsule("01GGG002", "beta", "Content")
	if err := Insert(context.Background(), db, c1); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if err := Insert(context.Background(), db, c2); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	workspace := "alpha"
	summaries, total, err := ListAll(context.Background(), db, InventoryFilters{Workspace: &workspace}, 10, 0, false)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if summaries[0].WorkspaceNorm != "alpha" {
		t.Errorf("workspace = %q, want alpha", summaries[0].WorkspaceNorm)
	}
}

func TestListAll_TagFilter(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Capsule with matching tag
	c1 := newTestCapsule("01HHH001", "default", "Content")
	c1.Tags = []string{"important", "urgent"}
	if err := Insert(context.Background(), db, c1); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Capsule without matching tag
	c2 := newTestCapsule("01HHH002", "default", "Content")
	c2.Tags = []string{"other"}
	if err := Insert(context.Background(), db, c2); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Capsule with no tags
	c3 := newTestCapsule("01HHH003", "default", "Content")
	if err := Insert(context.Background(), db, c3); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	tag := "important"
	summaries, total, err := ListAll(context.Background(), db, InventoryFilters{Tag: &tag}, 10, 0, false)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if summaries[0].ID != "01HHH001" {
		t.Errorf("ID = %q, want 01HHH001", summaries[0].ID)
	}
}

func TestListAll_NamePrefixFilter_EscapesWildcards(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Capsule with literal % in name
	c1 := newTestCapsule("01WILD01", "default", "Content")
	c1.NameRaw = stringPtr("test%percent")
	c1.NameNorm = stringPtr("test%percent")
	if err := Insert(context.Background(), db, c1); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Capsule with literal _ in name
	c2 := newTestCapsule("01WILD02", "default", "Content")
	c2.NameRaw = stringPtr("test_underscore")
	c2.NameNorm = stringPtr("test_underscore")
	if err := Insert(context.Background(), db, c2); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Capsule that would match unescaped % wildcard
	c3 := newTestCapsule("01WILD03", "default", "Content")
	c3.NameRaw = stringPtr("testANYTHING")
	c3.NameNorm = stringPtr("testanything")
	if err := Insert(context.Background(), db, c3); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Search for literal "test%" - should only match c1, not c3
	prefix := "test%"
	summaries, total, err := ListAll(context.Background(), db, InventoryFilters{NamePrefix: &prefix}, 10, 0, false)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if total != 1 {
		t.Errorf("total = %d, want 1 (only literal %% match)", total)
	}
	if len(summaries) != 1 {
		t.Errorf("len(summaries) = %d, want 1", len(summaries))
	}
	if len(summaries) > 0 && summaries[0].ID != "01WILD01" {
		t.Errorf("ID = %q, want 01WILD01", summaries[0].ID)
	}

	// Search for literal "test_" - should only match c2, not c3
	prefix = "test_"
	summaries, total, err = ListAll(context.Background(), db, InventoryFilters{NamePrefix: &prefix}, 10, 0, false)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if total != 1 {
		t.Errorf("total = %d, want 1 (only literal _ match)", total)
	}
	if len(summaries) > 0 && summaries[0].ID != "01WILD02" {
		t.Errorf("ID = %q, want 01WILD02", summaries[0].ID)
	}
}

func TestListAll_NamePrefixFilter(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Capsule matching prefix
	c1 := newTestCapsule("01III001", "default", "Content")
	c1.NameRaw = stringPtr("auth-login")
	c1.NameNorm = stringPtr("auth-login")
	if err := Insert(context.Background(), db, c1); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	c2 := newTestCapsule("01III002", "default", "Content")
	c2.NameRaw = stringPtr("auth-logout")
	c2.NameNorm = stringPtr("auth-logout")
	if err := Insert(context.Background(), db, c2); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Capsule not matching prefix
	c3 := newTestCapsule("01III003", "default", "Content")
	c3.NameRaw = stringPtr("other")
	c3.NameNorm = stringPtr("other")
	if err := Insert(context.Background(), db, c3); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	prefix := "auth"
	summaries, total, err := ListAll(context.Background(), db, InventoryFilters{NamePrefix: &prefix}, 10, 0, false)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(summaries) != 2 {
		t.Errorf("len(summaries) = %d, want 2", len(summaries))
	}
}

func TestListAll_Pagination(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert 5 capsules
	for i := 0; i < 5; i++ {
		c := newTestCapsule("01JJJ00"+string(rune('1'+i)), "default", "Content")
		c.UpdatedAt = int64(1000 + i)
		if err := Insert(context.Background(), db, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	// First page
	page1, total, err := ListAll(context.Background(), db, InventoryFilters{}, 2, 0, false)
	if err != nil {
		t.Fatalf("ListAll page 1 failed: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(page1) != 2 {
		t.Errorf("page1 len = %d, want 2", len(page1))
	}

	// Third page (partial)
	page3, _, err := ListAll(context.Background(), db, InventoryFilters{}, 2, 4, false)
	if err != nil {
		t.Fatalf("ListAll page 3 failed: %v", err)
	}
	if len(page3) != 1 {
		t.Errorf("page3 len = %d, want 1", len(page3))
	}
}

// =============================================================================
// GetLatestSummary Tests
// =============================================================================

func TestGetLatestSummary_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert 3 capsules with different updated_at
	c1 := newTestCapsule("01KKK001", "default", "First")
	c1.UpdatedAt = 1000
	if err := Insert(context.Background(), db, c1); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	c2 := newTestCapsule("01KKK002", "default", "Second")
	c2.UpdatedAt = 2000
	if err := Insert(context.Background(), db, c2); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	c3 := newTestCapsule("01KKK003", "default", "Third")
	c3.UpdatedAt = 1500
	if err := Insert(context.Background(), db, c3); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	summary, err := GetLatestSummary(context.Background(), db, "default", LatestFilters{}, false)
	if err != nil {
		t.Fatalf("GetLatestSummary failed: %v", err)
	}

	if summary == nil {
		t.Fatal("summary should not be nil")
	}
	if summary.ID != "01KKK002" {
		t.Errorf("ID = %q, want 01KKK002 (most recent)", summary.ID)
	}
}

func TestGetLatestSummary_EmptyWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	summary, err := GetLatestSummary(context.Background(), db, "empty", LatestFilters{}, false)
	if err != nil {
		t.Fatalf("GetLatestSummary failed: %v", err)
	}

	if summary != nil {
		t.Errorf("summary = %v, want nil for empty workspace", summary)
	}
}

func TestGetLatestSummary_StableOrdering(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert capsules with same updated_at
	sameTime := int64(1000)
	c1 := newTestCapsule("01LLL001", "default", "First")
	c1.UpdatedAt = sameTime
	if err := Insert(context.Background(), db, c1); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	c2 := newTestCapsule("01LLL003", "default", "Second")
	c2.UpdatedAt = sameTime
	if err := Insert(context.Background(), db, c2); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	summary, err := GetLatestSummary(context.Background(), db, "default", LatestFilters{}, false)
	if err != nil {
		t.Fatalf("GetLatestSummary failed: %v", err)
	}

	// Should return higher ID when updated_at is same
	if summary.ID != "01LLL003" {
		t.Errorf("ID = %q, want 01LLL003 (higher ID as tiebreaker)", summary.ID)
	}
}

func TestGetLatestSummary_IncludeDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert and delete a recent capsule
	c1 := newTestCapsule("01MMM001", "default", "Deleted")
	c1.UpdatedAt = 2000
	if err := Insert(context.Background(), db, c1); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if err := SoftDelete(context.Background(), db, c1.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Insert older active capsule
	c2 := newTestCapsule("01MMM002", "default", "Active")
	c2.UpdatedAt = 1000
	if err := Insert(context.Background(), db, c2); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Without includeDeleted
	summary, err := GetLatestSummary(context.Background(), db, "default", LatestFilters{}, false)
	if err != nil {
		t.Fatalf("GetLatestSummary failed: %v", err)
	}
	if summary.ID != "01MMM002" {
		t.Errorf("ID = %q, want 01MMM002 (active)", summary.ID)
	}

	// With includeDeleted - should return deleted one since it's more recent
	summary, err = GetLatestSummary(context.Background(), db, "default", LatestFilters{}, true)
	if err != nil {
		t.Fatalf("GetLatestSummary failed: %v", err)
	}
	if summary.ID != "01MMM001" {
		t.Errorf("ID = %q, want 01MMM001 (deleted but more recent)", summary.ID)
	}
}

// =============================================================================
// GetLatestFull Tests
// =============================================================================

func TestGetLatestFull_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	c := newTestCapsule("01NNN001", "default", "Full capsule content")
	if err := Insert(context.Background(), db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	capsule, err := GetLatestFull(context.Background(), db, "default", LatestFilters{}, false)
	if err != nil {
		t.Fatalf("GetLatestFull failed: %v", err)
	}

	if capsule == nil {
		t.Fatal("capsule should not be nil")
	}
	if capsule.CapsuleText != "Full capsule content" {
		t.Errorf("CapsuleText = %q, want 'Full capsule content'", capsule.CapsuleText)
	}
}

func TestGetLatestFull_EmptyWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	capsule, err := GetLatestFull(context.Background(), db, "empty", LatestFilters{}, false)
	if err != nil {
		t.Fatalf("GetLatestFull failed: %v", err)
	}

	if capsule != nil {
		t.Errorf("capsule = %v, want nil for empty workspace", capsule)
	}
}

// =============================================================================
// StreamForExport Tests
// =============================================================================

func TestStreamForExport_All(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert capsules in different workspaces
	c1 := newTestCapsule("01EXP001", "ws1", "Content 1")
	c1.CreatedAt = 1000
	c2 := newTestCapsule("01EXP002", "ws2", "Content 2")
	c2.CreatedAt = 2000
	c3 := newTestCapsule("01EXP003", "ws1", "Content 3")
	c3.CreatedAt = 3000

	for _, c := range []*capsule.Capsule{c1, c2, c3} {
		if err := Insert(context.Background(), db, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	rows, err := StreamForExport(context.Background(), db, nil, false)
	if err != nil {
		t.Fatalf("StreamForExport failed: %v", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		c, err := ScanCapsuleFromRows(rows)
		if err != nil {
			t.Fatalf("ScanCapsuleFromRows failed: %v", err)
		}
		ids = append(ids, c.ID)
	}

	if len(ids) != 3 {
		t.Errorf("got %d capsules, want 3", len(ids))
	}
	// Should be ordered by created_at ASC
	if ids[0] != "01EXP001" || ids[1] != "01EXP002" || ids[2] != "01EXP003" {
		t.Errorf("wrong order: %v, want [01EXP001 01EXP002 01EXP003]", ids)
	}
}

func TestStreamForExport_WorkspaceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	c1 := newTestCapsule("01EXP101", "target", "Content 1")
	c2 := newTestCapsule("01EXP102", "other", "Content 2")
	c3 := newTestCapsule("01EXP103", "target", "Content 3")

	for _, c := range []*capsule.Capsule{c1, c2, c3} {
		if err := Insert(context.Background(), db, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	ws := "target"
	rows, err := StreamForExport(context.Background(), db, &ws, false)
	if err != nil {
		t.Fatalf("StreamForExport failed: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		c, err := ScanCapsuleFromRows(rows)
		if err != nil {
			t.Fatalf("ScanCapsuleFromRows failed: %v", err)
		}
		if c.WorkspaceNorm != "target" {
			t.Errorf("unexpected workspace: %s", c.WorkspaceNorm)
		}
		count++
	}

	if count != 2 {
		t.Errorf("got %d capsules, want 2", count)
	}
}

func TestStreamForExport_IncludeDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	c1 := newTestCapsule("01EXP201", "default", "Active")
	c2 := newTestCapsule("01EXP202", "default", "Deleted")

	if err := Insert(context.Background(), db, c1); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if err := Insert(context.Background(), db, c2); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if err := SoftDelete(context.Background(), db, c2.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Without includeDeleted
	rows, err := StreamForExport(context.Background(), db, nil, false)
	if err != nil {
		t.Fatalf("StreamForExport failed: %v", err)
	}
	count := 0
	for rows.Next() {
		count++
	}
	rows.Close()
	if count != 1 {
		t.Errorf("without includeDeleted: got %d, want 1", count)
	}

	// With includeDeleted
	rows, err = StreamForExport(context.Background(), db, nil, true)
	if err != nil {
		t.Fatalf("StreamForExport failed: %v", err)
	}
	count = 0
	for rows.Next() {
		count++
	}
	rows.Close()
	if count != 2 {
		t.Errorf("with includeDeleted: got %d, want 2", count)
	}
}

// =============================================================================
// UpdateFull Tests
// =============================================================================

func TestUpdateFull_AllFields(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert original
	c := newTestCapsule("01UPD001", "original-ws", "Original content")
	c.NameRaw = stringPtr("original-name")
	c.NameNorm = stringPtr("original-name")
	c.Title = stringPtr("Original Title")
	c.Tags = []string{"old"}
	c.Source = stringPtr("original")
	c.CreatedAt = 1000
	c.UpdatedAt = 1000

	if err := Insert(context.Background(), db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Update all fields
	c.WorkspaceRaw = "new-ws"
	c.WorkspaceNorm = "new-ws"
	c.NameRaw = stringPtr("new-name")
	c.NameNorm = stringPtr("new-name")
	c.Title = stringPtr("New Title")
	c.CapsuleText = "New content"
	c.CapsuleChars = 11
	c.TokensEstimate = 3
	c.Tags = []string{"new1", "new2"}
	c.Source = stringPtr("updated")
	c.CreatedAt = 500  // Can change
	c.UpdatedAt = 2000 // Can change
	deletedAt := int64(3000)
	c.DeletedAt = &deletedAt

	if err := UpdateFull(context.Background(), db, c); err != nil {
		t.Fatalf("UpdateFull failed: %v", err)
	}

	// Retrieve and verify all fields
	retrieved, err := GetByID(context.Background(), db, c.ID, true)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.WorkspaceRaw != "new-ws" {
		t.Errorf("WorkspaceRaw = %q, want new-ws", retrieved.WorkspaceRaw)
	}
	if *retrieved.NameRaw != "new-name" {
		t.Errorf("NameRaw = %q, want new-name", *retrieved.NameRaw)
	}
	if *retrieved.Title != "New Title" {
		t.Errorf("Title = %q, want New Title", *retrieved.Title)
	}
	if retrieved.CapsuleText != "New content" {
		t.Errorf("CapsuleText = %q, want New content", retrieved.CapsuleText)
	}
	if retrieved.CreatedAt != 500 {
		t.Errorf("CreatedAt = %d, want 500", retrieved.CreatedAt)
	}
	if retrieved.UpdatedAt != 2000 {
		t.Errorf("UpdatedAt = %d, want 2000", retrieved.UpdatedAt)
	}
	if retrieved.DeletedAt == nil || *retrieved.DeletedAt != 3000 {
		t.Errorf("DeletedAt = %v, want 3000", retrieved.DeletedAt)
	}
}

func TestUpdateFull_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	c := newTestCapsule("nonexistent", "default", "Content")
	err = UpdateFull(context.Background(), db, c)
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("UpdateFull should return ErrNotFound, got: %v", err)
	}
}

// =============================================================================
// FindUniqueName Tests
// =============================================================================

func TestFindUniqueName_NoCollision(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	name, err := FindUniqueName(context.Background(), db, "default", "auth")
	if err != nil {
		t.Fatalf("FindUniqueName failed: %v", err)
	}
	if name != "auth" {
		t.Errorf("name = %q, want auth (no collision)", name)
	}
}

func TestFindUniqueName_FindsNextSuffix(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Create auth and auth-1
	c1 := newTestCapsule("01FUN001", "default", "Content")
	c1.NameRaw = stringPtr("auth")
	c1.NameNorm = stringPtr("auth")
	if err := Insert(context.Background(), db, c1); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	c2 := newTestCapsule("01FUN002", "default", "Content")
	c2.NameRaw = stringPtr("auth-1")
	c2.NameNorm = stringPtr("auth-1")
	if err := Insert(context.Background(), db, c2); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	name, err := FindUniqueName(context.Background(), db, "default", "auth")
	if err != nil {
		t.Fatalf("FindUniqueName failed: %v", err)
	}
	if name != "auth-2" {
		t.Errorf("name = %q, want auth-2", name)
	}
}

func TestFindUniqueName_SkipsGaps(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Create auth and auth-2 (skipping auth-1)
	c1 := newTestCapsule("01FUN101", "default", "Content")
	c1.NameRaw = stringPtr("auth")
	c1.NameNorm = stringPtr("auth")
	if err := Insert(context.Background(), db, c1); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	c2 := newTestCapsule("01FUN102", "default", "Content")
	c2.NameRaw = stringPtr("auth-2")
	c2.NameNorm = stringPtr("auth-2")
	if err := Insert(context.Background(), db, c2); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	name, err := FindUniqueName(context.Background(), db, "default", "auth")
	if err != nil {
		t.Fatalf("FindUniqueName failed: %v", err)
	}
	// Should find auth-1 (first gap)
	if name != "auth-1" {
		t.Errorf("name = %q, want auth-1 (first gap)", name)
	}
}

// =============================================================================
// PurgeDeleted Tests
// =============================================================================

func TestPurgeDeleted_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert and soft-delete some capsules
	c1 := newTestCapsule("01PUR001", "default", "Active")
	c2 := newTestCapsule("01PUR002", "default", "Deleted 1")
	c3 := newTestCapsule("01PUR003", "default", "Deleted 2")

	for _, c := range []*capsule.Capsule{c1, c2, c3} {
		if err := Insert(context.Background(), db, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	if err := SoftDelete(context.Background(), db, c2.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}
	if err := SoftDelete(context.Background(), db, c3.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Purge all deleted
	count, err := PurgeDeleted(context.Background(), db, nil, nil)
	if err != nil {
		t.Fatalf("PurgeDeleted failed: %v", err)
	}
	if count != 2 {
		t.Errorf("purged = %d, want 2", count)
	}

	// Verify active capsule still exists
	_, err = GetByID(context.Background(), db, c1.ID, false)
	if err != nil {
		t.Errorf("Active capsule should still exist: %v", err)
	}

	// Verify deleted capsules are gone
	_, err = GetByID(context.Background(), db, c2.ID, true)
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("Deleted capsule should be purged: %v", err)
	}
}

func TestPurgeDeleted_WorkspaceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Delete capsules in different workspaces
	c1 := newTestCapsule("01PUR101", "ws1", "Deleted in ws1")
	c2 := newTestCapsule("01PUR102", "ws2", "Deleted in ws2")

	for _, c := range []*capsule.Capsule{c1, c2} {
		if err := Insert(context.Background(), db, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
		if err := SoftDelete(context.Background(), db, c.ID); err != nil {
			t.Fatalf("SoftDelete failed: %v", err)
		}
	}

	// Purge only ws1
	ws := "ws1"
	count, err := PurgeDeleted(context.Background(), db, &ws, nil)
	if err != nil {
		t.Fatalf("PurgeDeleted failed: %v", err)
	}
	if count != 1 {
		t.Errorf("purged = %d, want 1", count)
	}

	// ws2 capsule should still exist (as deleted)
	_, err = GetByID(context.Background(), db, c2.ID, true)
	if err != nil {
		t.Errorf("ws2 capsule should still exist: %v", err)
	}
}

func TestPurgeDeleted_OlderThanDays(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Create capsules
	c1 := newTestCapsule("01PUR201", "default", "Recent")
	c2 := newTestCapsule("01PUR202", "default", "Old")

	for _, c := range []*capsule.Capsule{c1, c2} {
		if err := Insert(context.Background(), db, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	// Soft-delete both, but manipulate deleted_at for c2 to be old
	if err := SoftDelete(context.Background(), db, c1.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// For c2, manually set deleted_at to 10 days ago
	tenDaysAgo := time.Now().Unix() - (10 * 24 * 60 * 60)
	_, err = db.Exec("UPDATE capsules SET deleted_at = ? WHERE id = ?", tenDaysAgo, c2.ID)
	if err != nil {
		t.Fatalf("Failed to set old deleted_at: %v", err)
	}

	// Purge capsules deleted more than 7 days ago
	days := 7
	count, err := PurgeDeleted(context.Background(), db, nil, &days)
	if err != nil {
		t.Fatalf("PurgeDeleted failed: %v", err)
	}
	if count != 1 {
		t.Errorf("purged = %d, want 1 (only old one)", count)
	}

	// Recent deleted capsule should still exist
	_, err = GetByID(context.Background(), db, c1.ID, true)
	if err != nil {
		t.Errorf("Recent deleted capsule should still exist: %v", err)
	}

	// Old deleted capsule should be gone
	_, err = GetByID(context.Background(), db, c2.ID, true)
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("Old deleted capsule should be purged: %v", err)
	}
}

func TestPurgeDeleted_NoDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert only active capsules
	c := newTestCapsule("01PUR301", "default", "Active")
	if err := Insert(context.Background(), db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	count, err := PurgeDeleted(context.Background(), db, nil, nil)
	if err != nil {
		t.Fatalf("PurgeDeleted failed: %v", err)
	}
	if count != 0 {
		t.Errorf("purged = %d, want 0", count)
	}
}
