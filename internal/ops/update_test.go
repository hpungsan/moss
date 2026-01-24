package ops

import (
	"strings"
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

func TestUpdate_ByID(t *testing.T) {
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
		Tags:        []string{"v1"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Update by ID
	newText := validCapsuleText + "\nUpdated content."
	newTags := []string{"v2", "updated"}
	output, err := Update(database, cfg, UpdateInput{
		ID:          storeOutput.ID,
		CapsuleText: &newText,
		Tags:        &newTags,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if output.ID != storeOutput.ID {
		t.Errorf("ID = %q, want %q", output.ID, storeOutput.ID)
	}

	// Verify changes
	includeText := true
	fetched, err := Fetch(database, FetchInput{ID: storeOutput.ID, IncludeText: &includeText})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if !strings.Contains(fetched.CapsuleText, "Updated content") {
		t.Error("CapsuleText was not updated")
	}
	if len(fetched.Tags) != 2 || fetched.Tags[0] != "v2" {
		t.Errorf("Tags = %v, want [v2 updated]", fetched.Tags)
	}
}

func TestUpdate_ByName(t *testing.T) {
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

	// Update by name
	newTitle := "Updated Auth System"
	output, err := Update(database, cfg, UpdateInput{
		Workspace: "myworkspace",
		Name:      "auth",
		Title:     &newTitle,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if output.ID != storeOutput.ID {
		t.Errorf("ID = %q, want %q", output.ID, storeOutput.ID)
	}

	// Verify changes
	includeText := true
	fetched, err := Fetch(database, FetchInput{ID: storeOutput.ID, IncludeText: &includeText})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if fetched.Title == nil || *fetched.Title != "Updated Auth System" {
		t.Errorf("Title = %v, want %q", fetched.Title, "Updated Auth System")
	}
}

func TestUpdate_NoFieldsProvided(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	_, err = Update(database, cfg, UpdateInput{
		ID: "some-id",
		// No fields to update
	})
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("Update should return ErrInvalidRequest, got: %v", err)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	newTitle := "test"
	_, err = Update(database, cfg, UpdateInput{
		ID:    "nonexistent",
		Title: &newTitle,
	})
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("Update should return ErrNotFound, got: %v", err)
	}
}

func TestUpdate_CapsuleText_TooLarge(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := &config.Config{CapsuleMaxChars: 100}

	// Store with small limit but allow_thin
	storeOutput, err := Store(database, cfg, StoreInput{
		CapsuleText: "Small text",
		AllowThin:   true,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Try to update with large text
	largeText := strings.Repeat("x", 150)
	_, err = Update(database, cfg, UpdateInput{
		ID:          storeOutput.ID,
		CapsuleText: &largeText,
		AllowThin:   true,
	})
	if !errors.Is(err, errors.ErrCapsuleTooLarge) {
		t.Errorf("Update should return ErrCapsuleTooLarge, got: %v", err)
	}
}

func TestUpdate_CapsuleText_TooThin(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store with valid content
	storeOutput, err := Store(database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Try to update with thin content
	thinText := "## Objective\nOnly one section."
	_, err = Update(database, cfg, UpdateInput{
		ID:          storeOutput.ID,
		CapsuleText: &thinText,
		AllowThin:   false,
	})
	if !errors.Is(err, errors.ErrCapsuleTooThin) {
		t.Errorf("Update should return ErrCapsuleTooThin, got: %v", err)
	}
}

func TestUpdate_CapsuleText_AllowThin(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store with valid content
	storeOutput, err := Store(database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Update with thin content but allow_thin=true
	thinText := "## Objective\nOnly one section."
	_, err = Update(database, cfg, UpdateInput{
		ID:          storeOutput.ID,
		CapsuleText: &thinText,
		AllowThin:   true,
	})
	if err != nil {
		t.Fatalf("Update with AllowThin should succeed: %v", err)
	}

	// Verify
	includeText := true
	fetched, err := Fetch(database, FetchInput{ID: storeOutput.ID, IncludeText: &includeText})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if fetched.CapsuleText != thinText {
		t.Errorf("CapsuleText = %q, want %q", fetched.CapsuleText, thinText)
	}
}

func TestUpdate_PartialUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store with all fields
	storeOutput, err := Store(database, cfg, StoreInput{
		Name:        stringPtr("test"),
		Title:       stringPtr("Original Title"),
		CapsuleText: validCapsuleText,
		Tags:        []string{"original"},
		Source:      stringPtr("original-source"),
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Update only tags
	newTags := []string{"updated"}
	_, err = Update(database, cfg, UpdateInput{
		ID:   storeOutput.ID,
		Tags: &newTags,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify other fields are unchanged
	includeText := true
	fetched, err := Fetch(database, FetchInput{ID: storeOutput.ID, IncludeText: &includeText})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if *fetched.Title != "Original Title" {
		t.Errorf("Title = %q, want %q (unchanged)", *fetched.Title, "Original Title")
	}
	if *fetched.Source != "original-source" {
		t.Errorf("Source = %q, want %q (unchanged)", *fetched.Source, "original-source")
	}
	if len(fetched.Tags) != 1 || fetched.Tags[0] != "updated" {
		t.Errorf("Tags = %v, want [updated]", fetched.Tags)
	}
}

func TestUpdate_UpdatedAtChanges(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store
	storeOutput, err := Store(database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Get original updated_at
	includeText := true
	fetched1, err := Fetch(database, FetchInput{ID: storeOutput.ID, IncludeText: &includeText})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	originalUpdatedAt := fetched1.UpdatedAt

	// Update
	newSource := "updated"
	_, err = Update(database, cfg, UpdateInput{
		ID:     storeOutput.ID,
		Source: &newSource,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify updated_at changed
	fetched2, err := Fetch(database, FetchInput{ID: storeOutput.ID, IncludeText: &includeText})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if fetched2.UpdatedAt < originalUpdatedAt {
		t.Errorf("UpdatedAt decreased from %d to %d", originalUpdatedAt, fetched2.UpdatedAt)
	}
}

func TestUpdate_AmbiguousAddressing(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	newTitle := "test"
	_, err = Update(database, cfg, UpdateInput{
		ID:    "some-id",
		Name:  "some-name",
		Title: &newTitle,
	})
	if !errors.Is(err, errors.ErrAmbiguousAddressing) {
		t.Errorf("Update should return ErrAmbiguousAddressing, got: %v", err)
	}
}

func TestUpdate_ClearSource(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store with source
	storeOutput, err := Store(database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
		Source:      stringPtr("original"),
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Update to clear source (set to empty string)
	emptySource := ""
	_, err = Update(database, cfg, UpdateInput{
		ID:     storeOutput.ID,
		Source: &emptySource,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify source is updated to empty
	includeText := true
	fetched, err := Fetch(database, FetchInput{ID: storeOutput.ID, IncludeText: &includeText})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if fetched.Source == nil || *fetched.Source != "" {
		t.Errorf("Source = %v, want empty string", fetched.Source)
	}
}
