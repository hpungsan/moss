package ops

import (
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

func TestDelete_ByID(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	storeOutput, err := Store(database, cfg, StoreInput{
		Name:        stringPtr("test"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Delete by ID
	output, err := Delete(database, DeleteInput{
		ID: storeOutput.ID,
	})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if !output.Deleted {
		t.Error("Deleted = false, want true")
	}
	if output.ID != storeOutput.ID {
		t.Errorf("ID = %q, want %q", output.ID, storeOutput.ID)
	}

	// Verify capsule is no longer accessible
	_, err = Fetch(database, FetchInput{ID: storeOutput.ID, IncludeDeleted: false})
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("Fetch after delete should return ErrNotFound, got: %v", err)
	}
}

func TestDelete_ByName(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	storeOutput, err := Store(database, cfg, StoreInput{
		Workspace:   "myworkspace",
		Name:        stringPtr("auth"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Delete by name
	output, err := Delete(database, DeleteInput{
		Workspace: "myworkspace",
		Name:      "auth",
	})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if !output.Deleted {
		t.Error("Deleted = false, want true")
	}
	if output.ID != storeOutput.ID {
		t.Errorf("ID = %q, want %q", output.ID, storeOutput.ID)
	}
}

func TestDelete_NotFound_ByID(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	_, err = Delete(database, DeleteInput{
		ID: "nonexistent",
	})
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("Delete should return ErrNotFound, got: %v", err)
	}
}

func TestDelete_NotFound_ByName(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	_, err = Delete(database, DeleteInput{
		Workspace: "default",
		Name:      "nonexistent",
	})
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("Delete should return ErrNotFound, got: %v", err)
	}
}

func TestDelete_AmbiguousAddressing(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	_, err = Delete(database, DeleteInput{
		ID:   "some-id",
		Name: "some-name",
	})
	if !errors.Is(err, errors.ErrAmbiguousAddressing) {
		t.Errorf("Delete should return ErrAmbiguousAddressing, got: %v", err)
	}
}

func TestDelete_AlreadyDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store and delete
	storeOutput, err := Store(database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	_, err = Delete(database, DeleteInput{ID: storeOutput.ID})
	if err != nil {
		t.Fatalf("First Delete failed: %v", err)
	}

	// Try to delete again
	_, err = Delete(database, DeleteInput{ID: storeOutput.ID})
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("Second Delete should return ErrNotFound, got: %v", err)
	}
}

func TestDelete_NameSlotFreed(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store with name
	storeOutput, err := Store(database, cfg, StoreInput{
		Name:        stringPtr("reusable"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Delete
	_, err = Delete(database, DeleteInput{ID: storeOutput.ID})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Store with same name should succeed
	storeOutput2, err := Store(database, cfg, StoreInput{
		Name:        stringPtr("reusable"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store after delete failed: %v", err)
	}

	// IDs should be different
	if storeOutput2.ID == storeOutput.ID {
		t.Error("New capsule should have different ID")
	}
}

func TestDelete_DefaultWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store in default workspace
	storeOutput, err := Store(database, cfg, StoreInput{
		Name:        stringPtr("test"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Delete without specifying workspace (should default to "default")
	output, err := Delete(database, DeleteInput{
		Name: "test",
	})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if output.ID != storeOutput.ID {
		t.Errorf("ID = %q, want %q", output.ID, storeOutput.ID)
	}
}
