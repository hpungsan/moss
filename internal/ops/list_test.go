package ops

import (
	"context"
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
)

func TestList_HappyPath(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store 3 capsules
	for _, name := range []string{"cap1", "cap2", "cap3"} {
		_, err := Store(context.Background(), database, cfg, StoreInput{
			Workspace:   "default",
			Name:        stringPtr(name),
			CapsuleText: validCapsuleText,
		})
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}
	}

	// List
	output, err := List(context.Background(), database, ListInput{
		Workspace: "default",
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(output.Items) != 3 {
		t.Errorf("len(Items) = %d, want 3", len(output.Items))
	}
	if output.Pagination.Total != 3 {
		t.Errorf("Total = %d, want 3", output.Pagination.Total)
	}
	if output.Pagination.HasMore {
		t.Error("HasMore = true, want false")
	}
	if output.Sort != "updated_at_desc" {
		t.Errorf("Sort = %q, want 'updated_at_desc'", output.Sort)
	}
}

func TestList_DefaultWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store in default workspace
	_, err = Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// List with empty workspace (should default to "default")
	output, err := List(context.Background(), database, ListInput{
		Workspace: "",
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(output.Items))
	}
}

func TestList_Pagination(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store 5 capsules
	for i := 0; i < 5; i++ {
		_, err := Store(context.Background(), database, cfg, StoreInput{
			Workspace:   "default",
			CapsuleText: validCapsuleText,
		})
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}
	}

	// Get first page
	page1, err := List(context.Background(), database, ListInput{
		Workspace: "default",
		Limit:     2,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("List page 1 failed: %v", err)
	}

	if len(page1.Items) != 2 {
		t.Errorf("page1 len = %d, want 2", len(page1.Items))
	}
	if page1.Pagination.Total != 5 {
		t.Errorf("Total = %d, want 5", page1.Pagination.Total)
	}
	if !page1.Pagination.HasMore {
		t.Error("HasMore = false, want true")
	}
	if page1.Pagination.Limit != 2 {
		t.Errorf("Limit = %d, want 2", page1.Pagination.Limit)
	}
	if page1.Pagination.Offset != 0 {
		t.Errorf("Offset = %d, want 0", page1.Pagination.Offset)
	}

	// Get second page
	page2, err := List(context.Background(), database, ListInput{
		Workspace: "default",
		Limit:     2,
		Offset:    2,
	})
	if err != nil {
		t.Fatalf("List page 2 failed: %v", err)
	}

	if len(page2.Items) != 2 {
		t.Errorf("page2 len = %d, want 2", len(page2.Items))
	}
	if !page2.Pagination.HasMore {
		t.Error("HasMore = false, want true")
	}

	// Get third page
	page3, err := List(context.Background(), database, ListInput{
		Workspace: "default",
		Limit:     2,
		Offset:    4,
	})
	if err != nil {
		t.Fatalf("List page 3 failed: %v", err)
	}

	if len(page3.Items) != 1 {
		t.Errorf("page3 len = %d, want 1", len(page3.Items))
	}
	if page3.Pagination.HasMore {
		t.Error("HasMore = true, want false")
	}
}

func TestList_LimitBounds(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Test default limit (when 0 or negative)
	output, err := List(context.Background(), database, ListInput{
		Workspace: "default",
		Limit:     0,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if output.Pagination.Limit != DefaultListLimit {
		t.Errorf("Limit = %d, want %d", output.Pagination.Limit, DefaultListLimit)
	}

	// Test max limit (when exceeds max)
	output, err = List(context.Background(), database, ListInput{
		Workspace: "default",
		Limit:     1000,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if output.Pagination.Limit != MaxListLimit {
		t.Errorf("Limit = %d, want %d", output.Pagination.Limit, MaxListLimit)
	}
}

func TestList_EmptyWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	output, err := List(context.Background(), database, ListInput{
		Workspace: "empty",
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(output.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(output.Items))
	}
	if output.Pagination.Total != 0 {
		t.Errorf("Total = %d, want 0", output.Pagination.Total)
	}
	if output.Pagination.HasMore {
		t.Error("HasMore = true, want false")
	}
	// Verify Items is empty array, not nil
	if output.Items == nil {
		t.Error("Items should be empty array, not nil")
	}
}

func TestList_ReturnsSummaries(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("test"),
		CapsuleText: validCapsuleText,
		Tags:        []string{"tag1"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// List and verify summary fields
	output, err := List(context.Background(), database, ListInput{
		Workspace: "default",
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(output.Items))
	}

	item := output.Items[0]
	if item.ID == "" {
		t.Error("ID should not be empty")
	}
	if item.Workspace != "default" {
		t.Errorf("Workspace = %q, want 'default'", item.Workspace)
	}
	if item.Name == nil || *item.Name != "test" {
		t.Error("Name should be 'test'")
	}
	if item.CapsuleChars == 0 {
		t.Error("CapsuleChars should not be 0")
	}
	if len(item.Tags) != 1 || item.Tags[0] != "tag1" {
		t.Errorf("Tags = %v, want [tag1]", item.Tags)
	}
}

func TestList_IncludeDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store and delete a capsule
	stored, err := Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	if err := db.SoftDelete(database, stored.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Store active capsule
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Without includeDeleted
	output, err := List(context.Background(), database, ListInput{
		Workspace:      "default",
		IncludeDeleted: false,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if output.Pagination.Total != 1 {
		t.Errorf("Total = %d, want 1", output.Pagination.Total)
	}

	// With includeDeleted
	output, err = List(context.Background(), database, ListInput{
		Workspace:      "default",
		IncludeDeleted: true,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if output.Pagination.Total != 2 {
		t.Errorf("Total = %d, want 2", output.Pagination.Total)
	}
}

func TestList_NegativeOffset(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Negative offset should be treated as 0
	output, err := List(context.Background(), database, ListInput{
		Workspace: "default",
		Offset:    -10,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if output.Pagination.Offset != 0 {
		t.Errorf("Offset = %d, want 0", output.Pagination.Offset)
	}
}
