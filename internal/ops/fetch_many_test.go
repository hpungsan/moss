package ops

import (
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
)

func TestFetchMany_AllFound_ByID(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store 3 capsules
	var ids []string
	for _, name := range []string{"cap1", "cap2", "cap3"} {
		stored, err := Store(database, cfg, StoreInput{
			Workspace:   "default",
			Name:        stringPtr(name),
			CapsuleText: validCapsuleText,
		})
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}
		ids = append(ids, stored.ID)
	}

	// FetchMany by ID
	output, err := FetchMany(database, FetchManyInput{
		Items: []FetchManyRef{
			{ID: ids[0]},
			{ID: ids[1]},
			{ID: ids[2]},
		},
	})
	if err != nil {
		t.Fatalf("FetchMany failed: %v", err)
	}

	if len(output.Items) != 3 {
		t.Errorf("len(Items) = %d, want 3", len(output.Items))
	}
	if len(output.Errors) != 0 {
		t.Errorf("len(Errors) = %d, want 0", len(output.Errors))
	}
}

func TestFetchMany_AllFound_ByName(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store capsules
	_, err = Store(database, cfg, StoreInput{
		Workspace:   "ws1",
		Name:        stringPtr("auth"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	_, err = Store(database, cfg, StoreInput{
		Workspace:   "ws2",
		Name:        stringPtr("config"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// FetchMany by name
	output, err := FetchMany(database, FetchManyInput{
		Items: []FetchManyRef{
			{Workspace: "ws1", Name: "auth"},
			{Workspace: "ws2", Name: "config"},
		},
	})
	if err != nil {
		t.Fatalf("FetchMany failed: %v", err)
	}

	if len(output.Items) != 2 {
		t.Errorf("len(Items) = %d, want 2", len(output.Items))
	}
	if len(output.Errors) != 0 {
		t.Errorf("len(Errors) = %d, want 0", len(output.Errors))
	}
}

func TestFetchMany_PartialSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store one capsule
	stored, err := Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("exists"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// FetchMany - one exists, one doesn't
	output, err := FetchMany(database, FetchManyInput{
		Items: []FetchManyRef{
			{ID: stored.ID},
			{ID: "nonexistent"},
		},
	})
	if err != nil {
		t.Fatalf("FetchMany failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(output.Items))
	}
	if len(output.Errors) != 1 {
		t.Errorf("len(Errors) = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Code != "NOT_FOUND" {
		t.Errorf("Error code = %q, want NOT_FOUND", output.Errors[0].Code)
	}
}

func TestFetchMany_AllNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// FetchMany - none exist
	output, err := FetchMany(database, FetchManyInput{
		Items: []FetchManyRef{
			{ID: "nonexistent1"},
			{ID: "nonexistent2"},
		},
	})
	if err != nil {
		t.Fatalf("FetchMany failed: %v", err)
	}

	if len(output.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(output.Items))
	}
	if len(output.Errors) != 2 {
		t.Errorf("len(Errors) = %d, want 2", len(output.Errors))
	}
	// Verify Items is empty array, not nil
	if output.Items == nil {
		t.Error("Items should be empty array, not nil")
	}
}

func TestFetchMany_IncludeText_True(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	stored, err := Store(database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// FetchMany with include_text=true (default)
	output, err := FetchMany(database, FetchManyInput{
		Items: []FetchManyRef{{ID: stored.ID}},
	})
	if err != nil {
		t.Fatalf("FetchMany failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(output.Items))
	}
	if output.Items[0].CapsuleText == "" {
		t.Error("CapsuleText should not be empty when include_text=true (default)")
	}
}

func TestFetchMany_IncludeText_False(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	stored, err := Store(database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// FetchMany with include_text=false
	includeText := false
	output, err := FetchMany(database, FetchManyInput{
		Items:       []FetchManyRef{{ID: stored.ID}},
		IncludeText: &includeText,
	})
	if err != nil {
		t.Fatalf("FetchMany failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(output.Items))
	}
	if output.Items[0].CapsuleText != "" {
		t.Errorf("CapsuleText should be empty when include_text=false, got %d chars", len(output.Items[0].CapsuleText))
	}
	// Other fields should still be present
	if output.Items[0].ID == "" {
		t.Error("ID should not be empty")
	}
	if output.Items[0].CapsuleChars == 0 {
		t.Error("CapsuleChars should not be 0")
	}
}

func TestFetchMany_MixedAddressing(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store named capsule
	named, err := Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("named"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Store unnamed capsule
	unnamed, err := Store(database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// FetchMany with mixed addressing
	output, err := FetchMany(database, FetchManyInput{
		Items: []FetchManyRef{
			{ID: named.ID},                        // by ID
			{Workspace: "default", Name: "named"}, // by name
			{ID: unnamed.ID},                      // by ID
		},
	})
	if err != nil {
		t.Fatalf("FetchMany failed: %v", err)
	}

	// First two refs point to same capsule, but both should be returned
	if len(output.Items) != 3 {
		t.Errorf("len(Items) = %d, want 3", len(output.Items))
	}
	if len(output.Errors) != 0 {
		t.Errorf("len(Errors) = %d, want 0", len(output.Errors))
	}
}

func TestFetchMany_AmbiguousAddressing(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// FetchMany with ambiguous ref (both ID and name)
	output, err := FetchMany(database, FetchManyInput{
		Items: []FetchManyRef{
			{ID: "some-id", Name: "some-name"},
		},
	})
	if err != nil {
		t.Fatalf("FetchMany failed: %v", err)
	}

	if len(output.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(output.Items))
	}
	if len(output.Errors) != 1 {
		t.Errorf("len(Errors) = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Code != "AMBIGUOUS_ADDRESSING" {
		t.Errorf("Error code = %q, want AMBIGUOUS_ADDRESSING", output.Errors[0].Code)
	}
}

func TestFetchMany_InvalidRef(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// FetchMany with empty ref (neither ID nor name)
	output, err := FetchMany(database, FetchManyInput{
		Items: []FetchManyRef{
			{}, // empty
		},
	})
	if err != nil {
		t.Fatalf("FetchMany failed: %v", err)
	}

	if len(output.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(output.Items))
	}
	if len(output.Errors) != 1 {
		t.Errorf("len(Errors) = %d, want 1", len(output.Errors))
	}
	if output.Errors[0].Code != "INVALID_REQUEST" {
		t.Errorf("Error code = %q, want INVALID_REQUEST", output.Errors[0].Code)
	}
}

func TestFetchMany_EmptyInput(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// FetchMany with empty items
	output, err := FetchMany(database, FetchManyInput{
		Items: []FetchManyRef{},
	})
	if err != nil {
		t.Fatalf("FetchMany failed: %v", err)
	}

	if len(output.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(output.Items))
	}
	if len(output.Errors) != 0 {
		t.Errorf("len(Errors) = %d, want 0", len(output.Errors))
	}
	if output.Items == nil {
		t.Error("Items should be empty array, not nil")
	}
	if output.Errors == nil {
		t.Error("Errors should be empty array, not nil")
	}
}

func TestFetchMany_TaskLink(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store named capsule
	named, err := Store(database, cfg, StoreInput{
		Workspace:   "myworkspace",
		Name:        stringPtr("auth"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Store unnamed capsule
	unnamed, err := Store(database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	output, err := FetchMany(database, FetchManyInput{
		Items: []FetchManyRef{
			{ID: named.ID},
			{ID: unnamed.ID},
		},
	})
	if err != nil {
		t.Fatalf("FetchMany failed: %v", err)
	}

	if len(output.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(output.Items))
	}

	// Find named item
	var namedItem, unnamedItem *FetchManyItem
	for i := range output.Items {
		if output.Items[i].ID == named.ID {
			namedItem = &output.Items[i]
		}
		if output.Items[i].ID == unnamed.ID {
			unnamedItem = &output.Items[i]
		}
	}

	// Verify named capsule TaskLink
	if namedItem == nil {
		t.Fatal("named item not found")
	}
	if namedItem.TaskLink.MossCapsule != "auth" {
		t.Errorf("TaskLink.MossCapsule = %q, want 'auth'", namedItem.TaskLink.MossCapsule)
	}
	if namedItem.TaskLink.MossWorkspace != "myworkspace" {
		t.Errorf("TaskLink.MossWorkspace = %q, want 'myworkspace'", namedItem.TaskLink.MossWorkspace)
	}

	// Verify unnamed capsule TaskLink
	if unnamedItem == nil {
		t.Fatal("unnamed item not found")
	}
	if unnamedItem.TaskLink.MossID != unnamed.ID {
		t.Errorf("TaskLink.MossID = %q, want %q", unnamedItem.TaskLink.MossID, unnamed.ID)
	}
	if unnamedItem.TaskLink.MossCapsule != "" {
		t.Errorf("TaskLink.MossCapsule = %q, want empty (unnamed)", unnamedItem.TaskLink.MossCapsule)
	}
}

func TestFetchMany_DefaultWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store in default workspace
	_, err = Store(database, cfg, StoreInput{
		Name:        stringPtr("test"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// FetchMany without workspace (should default to "default")
	output, err := FetchMany(database, FetchManyInput{
		Items: []FetchManyRef{
			{Name: "test"},
		},
	})
	if err != nil {
		t.Fatalf("FetchMany failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(output.Items))
	}
}

func TestFetchMany_ErrorPreservesRef(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// FetchMany with non-existent refs
	output, err := FetchMany(database, FetchManyInput{
		Items: []FetchManyRef{
			{ID: "id-not-found"},
			{Workspace: "ws", Name: "name-not-found"},
		},
	})
	if err != nil {
		t.Fatalf("FetchMany failed: %v", err)
	}

	if len(output.Errors) != 2 {
		t.Fatalf("len(Errors) = %d, want 2", len(output.Errors))
	}

	// Verify ref is preserved in errors
	if output.Errors[0].Ref.ID != "id-not-found" {
		t.Errorf("Error[0].Ref.ID = %q, want 'id-not-found'", output.Errors[0].Ref.ID)
	}
	if output.Errors[1].Ref.Name != "name-not-found" {
		t.Errorf("Error[1].Ref.Name = %q, want 'name-not-found'", output.Errors[1].Ref.Name)
	}
	if output.Errors[1].Ref.Workspace != "ws" {
		t.Errorf("Error[1].Ref.Workspace = %q, want 'ws'", output.Errors[1].Ref.Workspace)
	}
}
