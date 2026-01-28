package ops

import (
	"context"
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
)

func TestLatest_HappyPath(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	stored, err := Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("test"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Latest
	output, err := Latest(context.Background(), database, LatestInput{
		Workspace: "default",
	})
	if err != nil {
		t.Fatalf("Latest failed: %v", err)
	}

	if output.Item == nil {
		t.Fatal("Item should not be nil")
	}
	if output.Item.ID != stored.ID {
		t.Errorf("ID = %q, want %q", output.Item.ID, stored.ID)
	}
}

func TestLatest_DefaultWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store in default workspace
	stored, err := Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Latest with empty workspace
	output, err := Latest(context.Background(), database, LatestInput{
		Workspace: "",
	})
	if err != nil {
		t.Fatalf("Latest failed: %v", err)
	}

	if output.Item == nil {
		t.Fatal("Item should not be nil")
	}
	if output.Item.ID != stored.ID {
		t.Errorf("ID = %q, want %q", output.Item.ID, stored.ID)
	}
}

func TestLatest_ReturnsSummaryByDefault(t *testing.T) {
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

	// Latest without include_text (default: false)
	output, err := Latest(context.Background(), database, LatestInput{
		Workspace: "default",
	})
	if err != nil {
		t.Fatalf("Latest failed: %v", err)
	}

	if output.Item == nil {
		t.Fatal("Item should not be nil")
	}
	// CapsuleText should be empty
	if output.Item.CapsuleText != "" {
		t.Errorf("CapsuleText should be empty, got %d chars", len(output.Item.CapsuleText))
	}
	// Summary fields should be present
	if output.Item.ID == "" {
		t.Error("ID should not be empty")
	}
	if output.Item.Name == nil || *output.Item.Name != "test" {
		t.Error("Name should be 'test'")
	}
	if output.Item.CapsuleChars == 0 {
		t.Error("CapsuleChars should not be 0")
	}
	if len(output.Item.Tags) != 1 || output.Item.Tags[0] != "tag1" {
		t.Errorf("Tags = %v, want [tag1]", output.Item.Tags)
	}
}

func TestLatest_IncludeText(t *testing.T) {
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

	// Latest with include_text=true
	includeText := true
	output, err := Latest(context.Background(), database, LatestInput{
		Workspace:   "default",
		IncludeText: &includeText,
	})
	if err != nil {
		t.Fatalf("Latest failed: %v", err)
	}

	if output.Item == nil {
		t.Fatal("Item should not be nil")
	}
	if output.Item.CapsuleText == "" {
		t.Error("CapsuleText should not be empty when include_text=true")
	}
}

func TestLatest_IncludeTextFalse(t *testing.T) {
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

	// Latest with include_text=false
	includeText := false
	output, err := Latest(context.Background(), database, LatestInput{
		Workspace:   "default",
		IncludeText: &includeText,
	})
	if err != nil {
		t.Fatalf("Latest failed: %v", err)
	}

	if output.Item == nil {
		t.Fatal("Item should not be nil")
	}
	if output.Item.CapsuleText != "" {
		t.Errorf("CapsuleText should be empty when include_text=false, got %d chars", len(output.Item.CapsuleText))
	}
}

func TestLatest_EmptyWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Latest on empty workspace
	output, err := Latest(context.Background(), database, LatestInput{
		Workspace: "empty",
	})
	if err != nil {
		t.Fatalf("Latest failed: %v", err)
	}

	if output.Item != nil {
		t.Errorf("Item = %v, want nil for empty workspace", output.Item)
	}
}

func TestLatest_ReturnsMostRecent(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store 3 capsules and track their IDs
	storedIDs := make(map[string]bool)
	for _, name := range []string{"first", "second", "third"} {
		stored, err := Store(context.Background(), database, cfg, StoreInput{
			Workspace:   "default",
			Name:        stringPtr(name),
			CapsuleText: validCapsuleText,
		})
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}
		storedIDs[stored.ID] = true
	}

	// Latest should return one of the stored capsules (the most recent by updated_at, id)
	output, err := Latest(context.Background(), database, LatestInput{
		Workspace: "default",
	})
	if err != nil {
		t.Fatalf("Latest failed: %v", err)
	}

	if output.Item == nil {
		t.Fatal("Item should not be nil")
	}
	if !storedIDs[output.Item.ID] {
		t.Errorf("ID = %q not in stored IDs", output.Item.ID)
	}

	// Call Latest again to verify deterministic ordering
	output2, err := Latest(context.Background(), database, LatestInput{
		Workspace: "default",
	})
	if err != nil {
		t.Fatalf("Latest failed: %v", err)
	}
	if output.Item.ID != output2.Item.ID {
		t.Errorf("Latest should be deterministic: first=%q, second=%q", output.Item.ID, output2.Item.ID)
	}
}

func TestLatest_FetchKey_Named(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store named capsule
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "myworkspace",
		Name:        stringPtr("auth"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	output, err := Latest(context.Background(), database, LatestInput{
		Workspace: "myworkspace",
	})
	if err != nil {
		t.Fatalf("Latest failed: %v", err)
	}

	if output.Item.FetchKey.MossCapsule != "auth" {
		t.Errorf("FetchKey.MossCapsule = %q, want 'auth'", output.Item.FetchKey.MossCapsule)
	}
	if output.Item.FetchKey.MossWorkspace != "myworkspace" {
		t.Errorf("FetchKey.MossWorkspace = %q, want 'myworkspace'", output.Item.FetchKey.MossWorkspace)
	}
}

func TestLatest_FetchKey_Unnamed(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store unnamed capsule
	stored, err := Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	output, err := Latest(context.Background(), database, LatestInput{
		Workspace: "default",
	})
	if err != nil {
		t.Fatalf("Latest failed: %v", err)
	}

	if output.Item.FetchKey.MossID != stored.ID {
		t.Errorf("FetchKey.MossID = %q, want %q", output.Item.FetchKey.MossID, stored.ID)
	}
	if output.Item.FetchKey.MossCapsule != "" {
		t.Errorf("FetchKey.MossCapsule = %q, want empty (unnamed)", output.Item.FetchKey.MossCapsule)
	}
}

func TestLatest_IncludeDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store older active capsule
	older, err := Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("older"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Manually set older capsule's updated_at to an earlier time
	_, err = database.Exec("UPDATE capsules SET updated_at = ? WHERE id = ?", 1000, older.ID)
	if err != nil {
		t.Fatalf("Failed to update timestamp: %v", err)
	}

	// Store and delete newer capsule
	newer, err := Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("newer"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Manually set newer capsule's updated_at to a later time
	_, err = database.Exec("UPDATE capsules SET updated_at = ? WHERE id = ?", 2000, newer.ID)
	if err != nil {
		t.Fatalf("Failed to update timestamp: %v", err)
	}

	if err := db.SoftDelete(context.Background(), database, newer.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Without includeDeleted - should return older active capsule
	output, err := Latest(context.Background(), database, LatestInput{
		Workspace:      "default",
		IncludeDeleted: false,
	})
	if err != nil {
		t.Fatalf("Latest failed: %v", err)
	}
	if output.Item == nil {
		t.Fatal("Item should not be nil")
	}
	if output.Item.ID != older.ID {
		t.Errorf("ID = %q, want %q (active)", output.Item.ID, older.ID)
	}

	// With includeDeleted - should return deleted but more recent
	output, err = Latest(context.Background(), database, LatestInput{
		Workspace:      "default",
		IncludeDeleted: true,
	})
	if err != nil {
		t.Fatalf("Latest failed: %v", err)
	}
	if output.Item == nil {
		t.Fatal("Item should not be nil")
	}
	if output.Item.ID != newer.ID {
		t.Errorf("ID = %q, want %q (deleted but more recent)", output.Item.ID, newer.ID)
	}
}
