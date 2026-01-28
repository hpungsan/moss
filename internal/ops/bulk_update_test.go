package ops

import (
	"context"
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
)

func TestBulkUpdate_UpdatePhaseByWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store in ws1
	stored, err := Store(context.Background(), database, cfg, StoreInput{Workspace: "ws1", CapsuleText: validCapsuleText})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Bulk update phase in ws1
	ws := "ws1"
	newPhase := "archived"
	output, err := BulkUpdate(context.Background(), database, BulkUpdateInput{
		Workspace: &ws,
		SetPhase:  &newPhase,
	})
	if err != nil {
		t.Fatalf("BulkUpdate failed: %v", err)
	}

	if output.Updated != 1 {
		t.Errorf("Updated = %d, want 1", output.Updated)
	}

	// Verify phase changed
	c, err := db.GetByID(context.Background(), database, stored.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if c.Phase == nil || *c.Phase != "archived" {
		t.Errorf("Phase = %v, want 'archived'", c.Phase)
	}
}

func TestBulkUpdate_UpdateRoleByTag(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store with tag "x"
	stored, err := Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
		Tags:        []string{"x"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Bulk update role by tag
	tag := "x"
	newRole := "reviewer"
	output, err := BulkUpdate(context.Background(), database, BulkUpdateInput{
		Tag:     &tag,
		SetRole: &newRole,
	})
	if err != nil {
		t.Fatalf("BulkUpdate failed: %v", err)
	}

	if output.Updated != 1 {
		t.Errorf("Updated = %d, want 1", output.Updated)
	}

	// Verify role changed
	c, err := db.GetByID(context.Background(), database, stored.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if c.Role == nil || *c.Role != "reviewer" {
		t.Errorf("Role = %v, want 'reviewer'", c.Role)
	}
}

func TestBulkUpdate_UpdateTagsByRunID(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	r1 := "run-123"
	stored, err := Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
		RunID:       &r1,
		Tags:        []string{"old-tag"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Bulk update tags by run_id
	newTags := []string{"new-tag-1", "new-tag-2"}
	output, err := BulkUpdate(context.Background(), database, BulkUpdateInput{
		RunID:   &r1,
		SetTags: &newTags,
	})
	if err != nil {
		t.Fatalf("BulkUpdate failed: %v", err)
	}

	if output.Updated != 1 {
		t.Errorf("Updated = %d, want 1", output.Updated)
	}

	// Verify tags replaced
	c, err := db.GetByID(context.Background(), database, stored.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if len(c.Tags) != 2 || c.Tags[0] != "new-tag-1" || c.Tags[1] != "new-tag-2" {
		t.Errorf("Tags = %v, want [new-tag-1, new-tag-2]", c.Tags)
	}
}

func TestBulkUpdate_UpdateMultipleFields(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	stored, err := Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "project",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Bulk update phase + role
	ws := "project"
	newPhase := "review"
	newRole := "qa"
	output, err := BulkUpdate(context.Background(), database, BulkUpdateInput{
		Workspace: &ws,
		SetPhase:  &newPhase,
		SetRole:   &newRole,
	})
	if err != nil {
		t.Fatalf("BulkUpdate failed: %v", err)
	}

	if output.Updated != 1 {
		t.Errorf("Updated = %d, want 1", output.Updated)
	}

	// Verify both changed
	c, err := db.GetByID(context.Background(), database, stored.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if c.Phase == nil || *c.Phase != "review" {
		t.Errorf("Phase = %v, want 'review'", c.Phase)
	}
	if c.Role == nil || *c.Role != "qa" {
		t.Errorf("Role = %v, want 'qa'", c.Role)
	}
}

func TestBulkUpdate_MultipleFiltersAND(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store matching both workspace + tag
	stored1, err := Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "project",
		CapsuleText: validCapsuleText,
		Tags:        []string{"target"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Store matching only workspace (not tag)
	stored2, err := Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "project",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Bulk update workspace="project" AND tag="target"
	ws := "project"
	tag := "target"
	newPhase := "done"
	output, err := BulkUpdate(context.Background(), database, BulkUpdateInput{
		Workspace: &ws,
		Tag:       &tag,
		SetPhase:  &newPhase,
	})
	if err != nil {
		t.Fatalf("BulkUpdate failed: %v", err)
	}

	if output.Updated != 1 {
		t.Errorf("Updated = %d, want 1 (AND semantics)", output.Updated)
	}

	// Verify only matching capsule was updated
	c1, err := db.GetByID(context.Background(), database, stored1.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if c1.Phase == nil || *c1.Phase != "done" {
		t.Errorf("Phase = %v, want 'done'", c1.Phase)
	}

	c2, err := db.GetByID(context.Background(), database, stored2.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if c2.Phase != nil {
		t.Errorf("Phase = %v, want nil (should not be updated)", c2.Phase)
	}
}

func TestBulkUpdate_NoFiltersError(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	newPhase := "archived"
	_, err = BulkUpdate(context.Background(), database, BulkUpdateInput{
		SetPhase: &newPhase,
	})
	if err == nil {
		t.Fatal("Expected error for no filters, got nil")
	}

	want := "INVALID_REQUEST: at least one filter is required"
	if err.Error() != want {
		t.Errorf("Error = %q, want %q", err.Error(), want)
	}
}

func TestBulkUpdate_NoUpdatesError(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	ws := "test"
	_, err = BulkUpdate(context.Background(), database, BulkUpdateInput{
		Workspace: &ws,
	})
	if err == nil {
		t.Fatal("Expected error for no updates, got nil")
	}

	want := "INVALID_REQUEST: at least one update field is required"
	if err.Error() != want {
		t.Errorf("Error = %q, want %q", err.Error(), want)
	}
}

func TestBulkUpdate_WhitespaceFiltersError(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	ws := "   "
	newPhase := "archived"
	_, err = BulkUpdate(context.Background(), database, BulkUpdateInput{
		Workspace: &ws,
		SetPhase:  &newPhase,
	})
	if err == nil {
		t.Fatal("Expected error for whitespace-only filters, got nil")
	}

	want := "INVALID_REQUEST: at least one filter must be non-empty after normalization"
	if err.Error() != want {
		t.Errorf("Error = %q, want %q", err.Error(), want)
	}
}

func TestBulkUpdate_WhitespaceUpdatesError(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	ws := "test"
	emptyPhase := "   "
	emptyRole := "   "
	_, err = BulkUpdate(context.Background(), database, BulkUpdateInput{
		Workspace: &ws,
		SetPhase:  &emptyPhase,
		SetRole:   &emptyRole,
	})
	// This should NOT error - empty strings after trimming mean "clear field"
	// Only completely nil update fields should error
	if err != nil {
		t.Fatalf("BulkUpdate failed unexpectedly: %v", err)
	}
}

func TestBulkUpdate_SkipsDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store two capsules in same workspace
	stored1, err := Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "target",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "target",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Soft-delete the first one
	if err := db.SoftDelete(context.Background(), database, stored1.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Bulk update same workspace — should only affect the active one
	ws := "target"
	newPhase := "updated"
	output, err := BulkUpdate(context.Background(), database, BulkUpdateInput{
		Workspace: &ws,
		SetPhase:  &newPhase,
	})
	if err != nil {
		t.Fatalf("BulkUpdate failed: %v", err)
	}

	if output.Updated != 1 {
		t.Errorf("Updated = %d, want 1 (should skip deleted)", output.Updated)
	}
}

func TestBulkUpdate_NoMatchesReturnsZero(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store in ws1
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "ws1",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Bulk update ws2 — no matches
	ws := "ws2"
	newPhase := "archived"
	output, err := BulkUpdate(context.Background(), database, BulkUpdateInput{
		Workspace: &ws,
		SetPhase:  &newPhase,
	})
	if err != nil {
		t.Fatalf("BulkUpdate failed: %v", err)
	}

	if output.Updated != 0 {
		t.Errorf("Updated = %d, want 0", output.Updated)
	}
	if output.Message != "No active capsules matched the filters" {
		t.Errorf("Message = %q, want 'No active capsules matched the filters'", output.Message)
	}
}

func TestBulkUpdate_ClearsFieldWithEmptyString(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store with phase set
	phase := "design"
	stored, err := Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "project",
		CapsuleText: validCapsuleText,
		Phase:       &phase,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Verify phase is set
	c, err := db.GetByID(context.Background(), database, stored.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if c.Phase == nil || *c.Phase != "design" {
		t.Fatalf("Phase = %v, want 'design'", c.Phase)
	}

	// Bulk update with empty string to clear phase
	ws := "project"
	emptyPhase := ""
	output, err := BulkUpdate(context.Background(), database, BulkUpdateInput{
		Workspace: &ws,
		SetPhase:  &emptyPhase,
	})
	if err != nil {
		t.Fatalf("BulkUpdate failed: %v", err)
	}

	if output.Updated != 1 {
		t.Errorf("Updated = %d, want 1", output.Updated)
	}

	// Verify phase is now nil
	c, err = db.GetByID(context.Background(), database, stored.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if c.Phase != nil {
		t.Errorf("Phase = %v, want nil (cleared)", c.Phase)
	}
}
