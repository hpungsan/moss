package ops

import (
	"context"
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
)

func TestInventory_NoFilters(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store capsules in different workspaces
	for _, ws := range []string{"ws1", "ws2", "ws3"} {
		_, err := Store(context.Background(), database, cfg, StoreInput{
			Workspace:   ws,
			CapsuleText: validCapsuleText,
		})
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}
	}

	// Inventory without filters
	output, err := Inventory(context.Background(), database, InventoryInput{})
	if err != nil {
		t.Fatalf("Inventory failed: %v", err)
	}

	if len(output.Items) != 3 {
		t.Errorf("len(Items) = %d, want 3", len(output.Items))
	}
	if output.Pagination.Total != 3 {
		t.Errorf("Total = %d, want 3", output.Pagination.Total)
	}
	if output.Sort != "updated_at_desc" {
		t.Errorf("Sort = %q, want 'updated_at_desc'", output.Sort)
	}
}

func TestInventory_WorkspaceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store in different workspaces
	_, err = Store(context.Background(), database, cfg, StoreInput{Workspace: "alpha", CapsuleText: validCapsuleText})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	_, err = Store(context.Background(), database, cfg, StoreInput{Workspace: "beta", CapsuleText: validCapsuleText})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Filter by workspace
	workspace := "alpha"
	output, err := Inventory(context.Background(), database, InventoryInput{Workspace: &workspace})
	if err != nil {
		t.Fatalf("Inventory failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(output.Items))
	}
	if output.Items[0].WorkspaceNorm != "alpha" {
		t.Errorf("WorkspaceNorm = %q, want 'alpha'", output.Items[0].WorkspaceNorm)
	}
}

func TestInventory_TagFilter(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store with matching tag
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
		Tags:        []string{"important", "urgent"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Store without matching tag
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
		Tags:        []string{"other"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Store with no tags
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Filter by tag
	tag := "important"
	output, err := Inventory(context.Background(), database, InventoryInput{Tag: &tag})
	if err != nil {
		t.Fatalf("Inventory failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(output.Items))
	}
}

func TestInventory_NamePrefixFilter(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store with matching prefix
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("auth-login"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("auth-logout"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Store without matching prefix
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("other"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Filter by name prefix
	prefix := "auth"
	output, err := Inventory(context.Background(), database, InventoryInput{NamePrefix: &prefix})
	if err != nil {
		t.Fatalf("Inventory failed: %v", err)
	}

	if len(output.Items) != 2 {
		t.Errorf("len(Items) = %d, want 2", len(output.Items))
	}
}

func TestInventory_MultipleFilters(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store matching all filters
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "project",
		Name:        stringPtr("auth-login"),
		CapsuleText: validCapsuleText,
		Tags:        []string{"important"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Store matching only workspace
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "project",
		Name:        stringPtr("other"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Store in different workspace
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "different",
		Name:        stringPtr("auth-test"),
		CapsuleText: validCapsuleText,
		Tags:        []string{"important"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Filter by workspace + tag + prefix
	workspace := "project"
	tag := "important"
	prefix := "auth"
	output, err := Inventory(context.Background(), database, InventoryInput{
		Workspace:  &workspace,
		Tag:        &tag,
		NamePrefix: &prefix,
	})
	if err != nil {
		t.Fatalf("Inventory failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(output.Items))
	}
}

func TestInventory_Pagination(t *testing.T) {
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

	// First page
	page1, err := Inventory(context.Background(), database, InventoryInput{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("Inventory page 1 failed: %v", err)
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

	// Last page
	page3, err := Inventory(context.Background(), database, InventoryInput{Limit: 2, Offset: 4})
	if err != nil {
		t.Fatalf("Inventory page 3 failed: %v", err)
	}

	if len(page3.Items) != 1 {
		t.Errorf("page3 len = %d, want 1", len(page3.Items))
	}
	if page3.Pagination.HasMore {
		t.Error("HasMore = true, want false")
	}
}

func TestInventory_LimitBounds(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Test default limit
	output, err := Inventory(context.Background(), database, InventoryInput{Limit: 0})
	if err != nil {
		t.Fatalf("Inventory failed: %v", err)
	}
	if output.Pagination.Limit != DefaultInventoryLimit {
		t.Errorf("Limit = %d, want %d", output.Pagination.Limit, DefaultInventoryLimit)
	}

	// Test max limit
	output, err = Inventory(context.Background(), database, InventoryInput{Limit: 1000})
	if err != nil {
		t.Fatalf("Inventory failed: %v", err)
	}
	if output.Pagination.Limit != MaxInventoryLimit {
		t.Errorf("Limit = %d, want %d", output.Pagination.Limit, MaxInventoryLimit)
	}
}

func TestInventory_ReturnsSummaries(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "myworkspace",
		Name:        stringPtr("test"),
		CapsuleText: validCapsuleText,
		Tags:        []string{"tag1"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Inventory and verify summary fields
	output, err := Inventory(context.Background(), database, InventoryInput{})
	if err != nil {
		t.Fatalf("Inventory failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(output.Items))
	}

	item := output.Items[0]
	if item.ID == "" {
		t.Error("ID should not be empty")
	}
	if item.Workspace != "myworkspace" {
		t.Errorf("Workspace = %q, want 'myworkspace'", item.Workspace)
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

func TestInventory_IncludeDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store and delete
	stored, err := Store(context.Background(), database, cfg, StoreInput{CapsuleText: validCapsuleText})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	if err := db.SoftDelete(context.Background(), database, stored.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Store active
	_, err = Store(context.Background(), database, cfg, StoreInput{CapsuleText: validCapsuleText})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Without includeDeleted
	output, err := Inventory(context.Background(), database, InventoryInput{IncludeDeleted: false})
	if err != nil {
		t.Fatalf("Inventory failed: %v", err)
	}
	if output.Pagination.Total != 1 {
		t.Errorf("Total = %d, want 1", output.Pagination.Total)
	}

	// With includeDeleted
	output, err = Inventory(context.Background(), database, InventoryInput{IncludeDeleted: true})
	if err != nil {
		t.Fatalf("Inventory failed: %v", err)
	}
	if output.Pagination.Total != 2 {
		t.Errorf("Total = %d, want 2", output.Pagination.Total)
	}
}

func TestInventory_EmptyFilters(t *testing.T) {
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
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Empty string filters should be treated as no filter
	emptyString := ""
	output, err := Inventory(context.Background(), database, InventoryInput{
		Workspace:  &emptyString,
		Tag:        &emptyString,
		NamePrefix: &emptyString,
	})
	if err != nil {
		t.Fatalf("Inventory failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(output.Items))
	}
}

func TestInventory_WhitespaceOnlyFilters(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule with a real tag
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
		Tags:        []string{"real-tag"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Whitespace-only tag filter should be treated as no filter (not as literal " ")
	whitespaceTag := "   "
	output, err := Inventory(context.Background(), database, InventoryInput{
		Tag: &whitespaceTag,
	})
	if err != nil {
		t.Fatalf("Inventory failed: %v", err)
	}

	// Should return all capsules (whitespace filter is ignored after trimming)
	if len(output.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1 (whitespace tag filter should be ignored)", len(output.Items))
	}
}
