package ops

import (
	"strings"
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

// validCapsuleText contains all 6 required sections.
const validCapsuleText = `## Objective
Build a user authentication system.

## Current status
Database schema is complete.

## Decisions
Using JWT for tokens.

## Next actions
Implement login endpoint.

## Key locations
cmd/auth/main.go

## Open questions
Should we support OAuth?
`

func stringPtr(s string) *string {
	return &s
}

func TestStore_HappyPath_Named(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	input := StoreInput{
		Workspace:   "myworkspace",
		Name:        stringPtr("auth"),
		CapsuleText: validCapsuleText,
		Tags:        []string{"tag1", "tag2"},
		Source:      stringPtr("test"),
	}

	output, err := Store(database, cfg, input)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	if output.ID == "" {
		t.Error("ID should not be empty")
	}
	if len(output.ID) != 26 {
		t.Errorf("ID length = %d, want 26 (ULID)", len(output.ID))
	}
	if output.FetchKey.MossCapsule != "auth" {
		t.Errorf("FetchKey.MossCapsule = %q, want %q", output.FetchKey.MossCapsule, "auth")
	}
	if output.FetchKey.MossWorkspace != "myworkspace" {
		t.Errorf("FetchKey.MossWorkspace = %q, want %q", output.FetchKey.MossWorkspace, "myworkspace")
	}
}

func TestStore_HappyPath_Unnamed(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	input := StoreInput{
		CapsuleText: validCapsuleText,
	}

	output, err := Store(database, cfg, input)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	if output.FetchKey.MossID == "" {
		t.Error("FetchKey.MossID should not be empty for unnamed capsule")
	}
	if output.FetchKey.MossCapsule != "" {
		t.Errorf("FetchKey.MossCapsule = %q, want empty (unnamed)", output.FetchKey.MossCapsule)
	}
}

func TestStore_DefaultWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	input := StoreInput{
		Name:        stringPtr("test"),
		CapsuleText: validCapsuleText,
	}

	output, err := Store(database, cfg, input)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Verify capsule is stored in default workspace
	capsule, err := db.GetByName(database, "default", "test", false)
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}
	if capsule.ID != output.ID {
		t.Errorf("Capsule ID mismatch")
	}
}

func TestStore_CapsuleTextRequired(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	input := StoreInput{
		Name: stringPtr("test"),
		// CapsuleText missing
	}

	_, err = Store(database, cfg, input)
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("Store should return ErrInvalidRequest, got: %v", err)
	}
}

func TestStore_TooLarge(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := &config.Config{CapsuleMaxChars: 100}

	// Create text that exceeds 100 chars
	largeText := strings.Repeat("x", 150)
	input := StoreInput{
		CapsuleText: largeText,
		AllowThin:   true, // Bypass section check to isolate size check
	}

	_, err = Store(database, cfg, input)
	if !errors.Is(err, errors.ErrCapsuleTooLarge) {
		t.Errorf("Store should return ErrCapsuleTooLarge, got: %v", err)
	}
}

func TestStore_TooThin(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Only has one section
	input := StoreInput{
		CapsuleText: "## Objective\nBuild something.",
		AllowThin:   false,
	}

	_, err = Store(database, cfg, input)
	if !errors.Is(err, errors.ErrCapsuleTooThin) {
		t.Errorf("Store should return ErrCapsuleTooThin, got: %v", err)
	}
}

func TestStore_AllowThin(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Only has one section, but allow_thin=true
	input := StoreInput{
		CapsuleText: "## Objective\nBuild something.",
		AllowThin:   true,
	}

	output, err := Store(database, cfg, input)
	if err != nil {
		t.Fatalf("Store with AllowThin should succeed: %v", err)
	}
	if output.ID == "" {
		t.Error("ID should not be empty")
	}
}

func TestStore_NameCollision_ModeError(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// First store
	input := StoreInput{
		Workspace:   "default",
		Name:        stringPtr("auth"),
		CapsuleText: validCapsuleText,
		Mode:        StoreModeError,
	}

	_, err = Store(database, cfg, input)
	if err != nil {
		t.Fatalf("First Store failed: %v", err)
	}

	// Second store with same name
	_, err = Store(database, cfg, input)
	if !errors.Is(err, errors.ErrNameAlreadyExists) {
		t.Errorf("Store should return ErrNameAlreadyExists, got: %v", err)
	}
}

func TestStore_NameCollision_ModeReplace(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// First store
	input := StoreInput{
		Workspace:   "default",
		Name:        stringPtr("auth"),
		CapsuleText: validCapsuleText,
		Tags:        []string{"v1"},
	}

	output1, err := Store(database, cfg, input)
	if err != nil {
		t.Fatalf("First Store failed: %v", err)
	}

	// Second store with same name and mode:replace
	input.CapsuleText = validCapsuleText + "\nUpdated content."
	input.Tags = []string{"v2"}
	input.Mode = StoreModeReplace

	output2, err := Store(database, cfg, input)
	if err != nil {
		t.Fatalf("Second Store with mode:replace failed: %v", err)
	}

	// ID should be preserved
	if output2.ID != output1.ID {
		t.Errorf("ID changed from %q to %q; mode:replace should preserve ID", output1.ID, output2.ID)
	}

	// Verify content was updated
	capsule, err := db.GetByID(database, output1.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if !strings.Contains(capsule.CapsuleText, "Updated content") {
		t.Error("CapsuleText was not updated")
	}
	if len(capsule.Tags) != 1 || capsule.Tags[0] != "v2" {
		t.Errorf("Tags = %v, want [v2]", capsule.Tags)
	}
}

func TestStore_TitleDefaultsToName(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	input := StoreInput{
		Name:        stringPtr("Auth System"),
		CapsuleText: validCapsuleText,
		// Title not provided
	}

	output, err := Store(database, cfg, input)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	capsule, err := db.GetByID(database, output.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if capsule.Title == nil || *capsule.Title != "Auth System" {
		t.Errorf("Title = %v, want %q (default to name)", capsule.Title, "Auth System")
	}
}

func TestStore_ExplicitTitle(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	input := StoreInput{
		Name:        stringPtr("auth"),
		Title:       stringPtr("User Authentication System"),
		CapsuleText: validCapsuleText,
	}

	output, err := Store(database, cfg, input)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	capsule, err := db.GetByID(database, output.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if capsule.Title == nil || *capsule.Title != "User Authentication System" {
		t.Errorf("Title = %v, want %q", capsule.Title, "User Authentication System")
	}
}

func TestStore_NormalizesWorkspaceAndName(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	input := StoreInput{
		Workspace:   "  My Workspace  ",
		Name:        stringPtr("  Auth System  "),
		CapsuleText: validCapsuleText,
	}

	output, err := Store(database, cfg, input)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	capsule, err := db.GetByID(database, output.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if capsule.WorkspaceNorm != "my workspace" {
		t.Errorf("WorkspaceNorm = %q, want %q", capsule.WorkspaceNorm, "my workspace")
	}
	if capsule.NameNorm == nil || *capsule.NameNorm != "auth system" {
		t.Errorf("NameNorm = %v, want %q", capsule.NameNorm, "auth system")
	}
}
