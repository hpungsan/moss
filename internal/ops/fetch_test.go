package ops

import (
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

func TestFetch_ByID(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule first
	storeOutput, err := Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("test"),
		CapsuleText: validCapsuleText,
		Tags:        []string{"tag1"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Fetch by ID
	includeText := true
	output, err := Fetch(database, FetchInput{
		ID:          storeOutput.ID,
		IncludeText: &includeText,
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if output.ID != storeOutput.ID {
		t.Errorf("ID = %q, want %q", output.ID, storeOutput.ID)
	}
	if output.CapsuleText == "" {
		t.Error("CapsuleText should not be empty")
	}
	if output.FetchKey.MossCapsule != "test" {
		t.Errorf("FetchKey.MossCapsule = %q, want %q", output.FetchKey.MossCapsule, "test")
	}
}

func TestFetch_ByName(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule first
	storeOutput, err := Store(database, cfg, StoreInput{
		Workspace:   "myworkspace",
		Name:        stringPtr("auth"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Fetch by name
	includeText := true
	output, err := Fetch(database, FetchInput{
		Workspace:   "myworkspace",
		Name:        "auth",
		IncludeText: &includeText,
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if output.ID != storeOutput.ID {
		t.Errorf("ID = %q, want %q", output.ID, storeOutput.ID)
	}
	if output.FetchKey.MossWorkspace != "myworkspace" {
		t.Errorf("FetchKey.MossWorkspace = %q, want %q", output.FetchKey.MossWorkspace, "myworkspace")
	}
}

func TestFetch_ByName_DefaultWorkspace(t *testing.T) {
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

	// Fetch without specifying workspace
	includeText := true
	output, err := Fetch(database, FetchInput{
		Name:        "test",
		IncludeText: &includeText,
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if output.ID != storeOutput.ID {
		t.Errorf("ID = %q, want %q", output.ID, storeOutput.ID)
	}
}

func TestFetch_NotFound_ByID(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	_, err = Fetch(database, FetchInput{
		ID: "nonexistent",
	})
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("Fetch should return ErrNotFound, got: %v", err)
	}
}

func TestFetch_NotFound_ByName(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	_, err = Fetch(database, FetchInput{
		Workspace: "default",
		Name:      "nonexistent",
	})
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("Fetch should return ErrNotFound, got: %v", err)
	}
}

func TestFetch_AmbiguousAddressing(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	_, err = Fetch(database, FetchInput{
		ID:   "some-id",
		Name: "some-name",
	})
	if !errors.Is(err, errors.ErrAmbiguousAddressing) {
		t.Errorf("Fetch should return ErrAmbiguousAddressing, got: %v", err)
	}
}

func TestFetch_IncludeText_False(t *testing.T) {
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

	// Fetch without text
	includeText := false
	output, err := Fetch(database, FetchInput{
		ID:          storeOutput.ID,
		IncludeText: &includeText,
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if output.CapsuleText != "" {
		t.Errorf("CapsuleText = %q, want empty (IncludeText=false)", output.CapsuleText)
	}
	// Other fields should still be present
	if output.ID == "" {
		t.Error("ID should not be empty")
	}
	if output.CapsuleChars == 0 {
		t.Error("CapsuleChars should not be 0")
	}
}

func TestFetch_IncludeDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store and delete a capsule
	storeOutput, err := Store(database, cfg, StoreInput{
		Name:        stringPtr("deleted-test"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	if err := db.SoftDelete(database, storeOutput.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Fetch without include_deleted should fail
	_, err = Fetch(database, FetchInput{
		ID:             storeOutput.ID,
		IncludeDeleted: false,
	})
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("Fetch should return ErrNotFound, got: %v", err)
	}

	// Fetch with include_deleted should succeed
	output, err := Fetch(database, FetchInput{
		ID:             storeOutput.ID,
		IncludeDeleted: true,
	})
	if err != nil {
		t.Fatalf("Fetch with IncludeDeleted failed: %v", err)
	}
	if output.ID != storeOutput.ID {
		t.Errorf("ID = %q, want %q", output.ID, storeOutput.ID)
	}
	if output.DeletedAt == nil {
		t.Error("DeletedAt should be set for deleted capsule")
	}
}

func TestFetch_UnnamedCapsule_FetchKey(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store unnamed capsule
	storeOutput, err := Store(database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Fetch
	includeText := true
	output, err := Fetch(database, FetchInput{
		ID:          storeOutput.ID,
		IncludeText: &includeText,
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// FetchKey should use ID for unnamed capsule
	if output.FetchKey.MossID != storeOutput.ID {
		t.Errorf("FetchKey.MossID = %q, want %q", output.FetchKey.MossID, storeOutput.ID)
	}
	if output.FetchKey.MossCapsule != "" {
		t.Errorf("FetchKey.MossCapsule = %q, want empty (unnamed)", output.FetchKey.MossCapsule)
	}
}

func TestFetch_DefaultsToIncludeText(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule first
	storeOutput, err := Store(database, cfg, StoreInput{
		Name:        stringPtr("test"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Fetch without IncludeText set should include text by default
	output, err := Fetch(database, FetchInput{
		ID: storeOutput.ID,
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if output.CapsuleText == "" {
		t.Error("CapsuleText should not be empty (default include_text=true)")
	}
}
