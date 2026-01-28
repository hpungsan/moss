package ops

import (
	"context"
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
)

func TestBulkDelete_WorkspaceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store in ws1 and ws2
	_, err = Store(context.Background(), database, cfg, StoreInput{Workspace: "ws1", CapsuleText: validCapsuleText})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	stored2, err := Store(context.Background(), database, cfg, StoreInput{Workspace: "ws2", CapsuleText: validCapsuleText})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Bulk delete ws1
	ws := "ws1"
	output, err := BulkDelete(context.Background(), database, BulkDeleteInput{Workspace: &ws})
	if err != nil {
		t.Fatalf("BulkDelete failed: %v", err)
	}

	if output.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1", output.Deleted)
	}

	// ws2 capsule should still be active
	_, err = db.GetByID(context.Background(), database, stored2.ID, false)
	if err != nil {
		t.Errorf("ws2 capsule should still be active: %v", err)
	}
}

func TestBulkDelete_TagFilter(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store with tag "x" and tag "y"
	_, err = Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
		Tags:        []string{"x"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	storedY, err := Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
		Tags:        []string{"y"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Bulk delete tag "x"
	tag := "x"
	output, err := BulkDelete(context.Background(), database, BulkDeleteInput{Tag: &tag})
	if err != nil {
		t.Fatalf("BulkDelete failed: %v", err)
	}

	if output.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1", output.Deleted)
	}

	// tag "y" capsule should still be active
	_, err = db.GetByID(context.Background(), database, storedY.ID, false)
	if err != nil {
		t.Errorf("tag y capsule should still be active: %v", err)
	}
}

func TestBulkDelete_NamePrefixFilter(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store auth-login, auth-logout, other
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Name:        stringPtr("auth-login"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Name:        stringPtr("auth-logout"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	storedOther, err := Store(context.Background(), database, cfg, StoreInput{
		Name:        stringPtr("other"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Bulk delete prefix "auth"
	prefix := "auth"
	output, err := BulkDelete(context.Background(), database, BulkDeleteInput{NamePrefix: &prefix})
	if err != nil {
		t.Fatalf("BulkDelete failed: %v", err)
	}

	if output.Deleted != 2 {
		t.Errorf("Deleted = %d, want 2", output.Deleted)
	}

	// "other" capsule should still be active
	_, err = db.GetByID(context.Background(), database, storedOther.ID, false)
	if err != nil {
		t.Errorf("other capsule should still be active: %v", err)
	}
}

func TestBulkDelete_RunIDFilter(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	r1 := "r1"
	r2 := "r2"
	_, err = Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
		RunID:       &r1,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	storedR2, err := Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
		RunID:       &r2,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Bulk delete run_id "r1"
	output, err := BulkDelete(context.Background(), database, BulkDeleteInput{RunID: &r1})
	if err != nil {
		t.Fatalf("BulkDelete failed: %v", err)
	}

	if output.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1", output.Deleted)
	}

	// r2 capsule should still be active
	_, err = db.GetByID(context.Background(), database, storedR2.ID, false)
	if err != nil {
		t.Errorf("r2 capsule should still be active: %v", err)
	}
}

func TestBulkDelete_PhaseFilter(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	research := "research"
	implement := "implement"
	_, err = Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
		Phase:       &research,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	storedImpl, err := Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
		Phase:       &implement,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Bulk delete phase "research"
	output, err := BulkDelete(context.Background(), database, BulkDeleteInput{Phase: &research})
	if err != nil {
		t.Fatalf("BulkDelete failed: %v", err)
	}

	if output.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1", output.Deleted)
	}

	// "implement" capsule should still be active
	_, err = db.GetByID(context.Background(), database, storedImpl.ID, false)
	if err != nil {
		t.Errorf("implement capsule should still be active: %v", err)
	}
}

func TestBulkDelete_RoleFilter(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	qa := "qa"
	dev := "dev"
	_, err = Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
		Role:        &qa,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	storedDev, err := Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
		Role:        &dev,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Bulk delete role "qa"
	output, err := BulkDelete(context.Background(), database, BulkDeleteInput{Role: &qa})
	if err != nil {
		t.Fatalf("BulkDelete failed: %v", err)
	}

	if output.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1", output.Deleted)
	}

	// "dev" capsule should still be active
	_, err = db.GetByID(context.Background(), database, storedDev.ID, false)
	if err != nil {
		t.Errorf("dev capsule should still be active: %v", err)
	}
}

func TestBulkDelete_MultipleFilters(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store matching both workspace + tag
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "project",
		CapsuleText: validCapsuleText,
		Tags:        []string{"cleanup"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Store matching only workspace (not tag)
	storedNoTag, err := Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "project",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Store matching only tag (not workspace)
	storedOtherWS, err := Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "other",
		CapsuleText: validCapsuleText,
		Tags:        []string{"cleanup"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Bulk delete workspace="project" AND tag="cleanup"
	ws := "project"
	tag := "cleanup"
	output, err := BulkDelete(context.Background(), database, BulkDeleteInput{
		Workspace: &ws,
		Tag:       &tag,
	})
	if err != nil {
		t.Fatalf("BulkDelete failed: %v", err)
	}

	if output.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1 (AND semantics)", output.Deleted)
	}

	// Non-matching capsules should still be active
	_, err = db.GetByID(context.Background(), database, storedNoTag.ID, false)
	if err != nil {
		t.Errorf("project capsule without tag should still be active: %v", err)
	}
	_, err = db.GetByID(context.Background(), database, storedOtherWS.ID, false)
	if err != nil {
		t.Errorf("other workspace capsule should still be active: %v", err)
	}
}

func TestBulkDelete_NoFiltersError(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	_, err = BulkDelete(context.Background(), database, BulkDeleteInput{})
	if err == nil {
		t.Fatal("Expected error for no filters, got nil")
	}

	want := "INVALID_REQUEST: at least one filter is required"
	if err.Error() != want {
		t.Errorf("Error = %q, want %q", err.Error(), want)
	}
}

func TestBulkDelete_WhitespaceOnlyFiltersError(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	ws := "   "
	tag := "   "
	_, err = BulkDelete(context.Background(), database, BulkDeleteInput{
		Workspace: &ws,
		Tag:       &tag,
	})
	if err == nil {
		t.Fatal("Expected error for whitespace-only filters, got nil")
	}

	want := "INVALID_REQUEST: at least one filter must be non-empty after normalization"
	if err.Error() != want {
		t.Errorf("Error = %q, want %q", err.Error(), want)
	}
}

func TestBulkDelete_SkipsAlreadyDeleted(t *testing.T) {
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

	// Soft-delete the first one manually
	if err := db.SoftDelete(context.Background(), database, stored1.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Bulk delete same workspace — should only affect the active one
	ws := "target"
	output, err := BulkDelete(context.Background(), database, BulkDeleteInput{Workspace: &ws})
	if err != nil {
		t.Fatalf("BulkDelete failed: %v", err)
	}

	if output.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1 (should skip already-deleted)", output.Deleted)
	}
}

func TestBulkDelete_NoMatchesReturnsZero(t *testing.T) {
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

	// Bulk delete ws2 — no matches
	ws := "ws2"
	output, err := BulkDelete(context.Background(), database, BulkDeleteInput{Workspace: &ws})
	if err != nil {
		t.Fatalf("BulkDelete failed: %v", err)
	}

	if output.Deleted != 0 {
		t.Errorf("Deleted = %d, want 0", output.Deleted)
	}
	if output.Message != "No active capsules matched the filters" {
		t.Errorf("Message = %q, want 'No active capsules matched the filters'", output.Message)
	}
}
