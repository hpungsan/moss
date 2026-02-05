package ops

import (
	"context"
	"strings"
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

func TestAppend_AppendToExisting(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	storeOutput, err := Store(context.Background(), database, cfg, StoreInput{
		Name:        stringPtr("test"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Append to Status section
	output, err := Append(context.Background(), database, cfg, AppendInput{
		ID:      storeOutput.ID,
		Section: "Status",
		Content: "Update: feature complete",
	})
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	if output.ID != storeOutput.ID {
		t.Errorf("ID = %q, want %q", output.ID, storeOutput.ID)
	}
	if output.Replaced {
		t.Error("Replaced should be false for append to existing content")
	}
	if !strings.Contains(output.SectionHit, "status") && !strings.Contains(output.SectionHit, "Status") {
		t.Errorf("SectionHit = %q, should contain 'status'", output.SectionHit)
	}

	// Verify content was appended
	includeText := true
	fetched, err := Fetch(context.Background(), database, FetchInput{ID: storeOutput.ID, IncludeText: &includeText})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if !strings.Contains(fetched.CapsuleText, "Update: feature complete") {
		t.Error("Appended content not found in capsule")
	}
}

func TestAppend_ReplacePlaceholder(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule with placeholder in Decisions
	capsuleText := `## Objective
Test placeholder replacement

## Current status
In progress

## Decisions
(pending)

## Next actions
- Test

## Key locations
- test.go

## Open questions
None
`
	storeOutput, err := Store(context.Background(), database, cfg, StoreInput{
		Name:        stringPtr("test"),
		CapsuleText: capsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Append to Decisions section (should replace placeholder)
	output, err := Append(context.Background(), database, cfg, AppendInput{
		ID:      storeOutput.ID,
		Section: "Decisions",
		Content: "Decided to use Go",
	})
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	if !output.Replaced {
		t.Error("Replaced should be true when replacing placeholder")
	}

	// Verify placeholder was replaced
	includeText := true
	fetched, err := Fetch(context.Background(), database, FetchInput{ID: storeOutput.ID, IncludeText: &includeText})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if strings.Contains(fetched.CapsuleText, "(pending)") {
		t.Error("Placeholder should have been replaced")
	}
	if !strings.Contains(fetched.CapsuleText, "Decided to use Go") {
		t.Error("New content not found")
	}
}

func TestAppend_ByName(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "test-ws",
		Name:        stringPtr("auth"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Append by name
	output, err := Append(context.Background(), database, cfg, AppendInput{
		Workspace: "test-ws",
		Name:      "auth",
		Section:   "Status",
		Content:   "Progress update",
	})
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	if output.FetchKey == nil {
		t.Error("FetchKey should be present for named capsule")
	} else if output.FetchKey.MossCapsule != "auth" {
		t.Errorf("FetchKey.MossCapsule = %q, want 'auth'", output.FetchKey.MossCapsule)
	}
}

func TestAppend_SectionNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	storeOutput, err := Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Try to append to nonexistent section
	_, err = Append(context.Background(), database, cfg, AppendInput{
		ID:      storeOutput.ID,
		Section: "Nonexistent Section",
		Content: "test",
	})
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("Expected ErrInvalidRequest, got: %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), "section not found") {
		t.Errorf("Error should mention 'section not found', got: %v", err)
	}
}

func TestAppend_SizeLimitExceeded(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := &config.Config{CapsuleMaxChars: 500}

	// Store a small capsule with allow_thin
	capsuleText := `## Objective
Test

## Status
OK
`
	storeOutput, err := Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: capsuleText,
		AllowThin:   true,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Try to append large content
	largeContent := strings.Repeat("x", 600)
	_, err = Append(context.Background(), database, cfg, AppendInput{
		ID:      storeOutput.ID,
		Section: "Status",
		Content: largeContent,
	})
	if !errors.Is(err, errors.ErrCapsuleTooLarge) {
		t.Errorf("Expected ErrCapsuleTooLarge, got: %v", err)
	}
}

func TestAppend_CustomSection(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule with custom section
	capsuleText := `## Objective
Test custom sections

## Current status
OK

## Decisions
Made

## Next actions
- Test

## Key locations
- here

## Open questions
None

## Design Reviews
Round 1: APPROVE
`
	storeOutput, err := Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: capsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Append to custom section
	output, err := Append(context.Background(), database, cfg, AppendInput{
		ID:      storeOutput.ID,
		Section: "Design Reviews",
		Content: "Round 2: APPROVE",
	})
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	if !strings.Contains(output.SectionHit, "Design Reviews") {
		t.Errorf("SectionHit = %q, should contain 'Design Reviews'", output.SectionHit)
	}

	// Verify content
	includeText := true
	fetched, err := Fetch(context.Background(), database, FetchInput{ID: storeOutput.ID, IncludeText: &includeText})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if !strings.Contains(fetched.CapsuleText, "Round 1: APPROVE") {
		t.Error("Original content should be preserved")
	}
	if !strings.Contains(fetched.CapsuleText, "Round 2: APPROVE") {
		t.Error("Appended content not found")
	}
}

func TestAppend_SynonymMatch(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	storeOutput, err := Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Append using synonym "goal" → should match "Objective"
	output, err := Append(context.Background(), database, cfg, AppendInput{
		ID:      storeOutput.ID,
		Section: "goal",
		Content: "Additional goal",
	})
	if err != nil {
		t.Fatalf("Append with synonym failed: %v", err)
	}

	// SectionHit should be the actual header
	if !strings.Contains(output.SectionHit, "Objective") {
		t.Errorf("SectionHit = %q, should contain 'Objective'", output.SectionHit)
	}
}

func TestAppend_CapsuleNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	_, err = Append(context.Background(), database, cfg, AppendInput{
		ID:      "nonexistent-id",
		Section: "Status",
		Content: "test",
	})
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestAppend_EmptySection(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	_, err = Append(context.Background(), database, cfg, AppendInput{
		ID:      "some-id",
		Section: "",
		Content: "test",
	})
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("Expected ErrInvalidRequest for empty section, got: %v", err)
	}
}

func TestAppend_EmptyContent(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	_, err = Append(context.Background(), database, cfg, AppendInput{
		ID:      "some-id",
		Section: "Status",
		Content: "",
	})
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("Expected ErrInvalidRequest for empty content, got: %v", err)
	}
}

func TestAppend_WhitespaceOnlyContent(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	_, err = Append(context.Background(), database, cfg, AppendInput{
		ID:      "some-id",
		Section: "Status",
		Content: "   \t\n  ",
	})
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("Expected ErrInvalidRequest for whitespace-only content, got: %v", err)
	}
}

func TestAppend_JSONFormatCapsule(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a JSON format capsule
	jsonCapsule := `{"objective": "test", "status": "in progress", "decisions": "none", "next_actions": "test", "key_locations": "here", "open_questions": "none"}`
	storeOutput, err := Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: jsonCapsule,
		AllowThin:   true, // JSON doesn't have markdown headers
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Try to append to JSON capsule
	_, err = Append(context.Background(), database, cfg, AppendInput{
		ID:      storeOutput.ID,
		Section: "status",
		Content: "update",
	})
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("Expected ErrInvalidRequest for JSON capsule, got: %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), "markdown format") {
		t.Errorf("Error should mention 'markdown format', got: %v", err)
	}
}

func TestAppend_UnnamedCapsule(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store an unnamed capsule
	storeOutput, err := Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
		// No name
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Append to unnamed capsule
	output, err := Append(context.Background(), database, cfg, AppendInput{
		ID:      storeOutput.ID,
		Section: "Status",
		Content: "Update",
	})
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// FetchKey should be nil for unnamed capsule
	if output.FetchKey != nil {
		t.Error("FetchKey should be nil for unnamed capsule")
	}
}

func TestAppend_SequentialAppends(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	storeOutput, err := Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// First append
	_, err = Append(context.Background(), database, cfg, AppendInput{
		ID:      storeOutput.ID,
		Section: "Status",
		Content: "Update 1",
	})
	if err != nil {
		t.Fatalf("First append failed: %v", err)
	}

	// Second append
	_, err = Append(context.Background(), database, cfg, AppendInput{
		ID:      storeOutput.ID,
		Section: "Status",
		Content: "Update 2",
	})
	if err != nil {
		t.Fatalf("Second append failed: %v", err)
	}

	// Third append
	_, err = Append(context.Background(), database, cfg, AppendInput{
		ID:      storeOutput.ID,
		Section: "Status",
		Content: "Update 3",
	})
	if err != nil {
		t.Fatalf("Third append failed: %v", err)
	}

	// Verify all appends are present
	includeText := true
	fetched, err := Fetch(context.Background(), database, FetchInput{ID: storeOutput.ID, IncludeText: &includeText})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	for i := 1; i <= 3; i++ {
		expected := "Update " + string(rune('0'+i))
		if !strings.Contains(fetched.CapsuleText, expected) {
			t.Errorf("Missing append %d content", i)
		}
	}
}

func TestAppend_ContentWithLeadingTrailingWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	storeOutput, err := Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Append content with leading/trailing whitespace
	contentWithWhitespace := "  Content with spaces  "
	_, err = Append(context.Background(), database, cfg, AppendInput{
		ID:      storeOutput.ID,
		Section: "Status",
		Content: contentWithWhitespace,
	})
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Verify content is preserved (not trimmed)
	includeText := true
	fetched, err := Fetch(context.Background(), database, FetchInput{ID: storeOutput.ID, IncludeText: &includeText})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if !strings.Contains(fetched.CapsuleText, contentWithWhitespace) {
		t.Error("Content whitespace should be preserved")
	}
}

func TestAppend_AmbiguousAddressing(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	_, err = Append(context.Background(), database, cfg, AppendInput{
		ID:      "some-id",
		Name:    "some-name",
		Section: "Status",
		Content: "test",
	})
	if !errors.Is(err, errors.ErrAmbiguousAddressing) {
		t.Errorf("Expected ErrAmbiguousAddressing, got: %v", err)
	}
}
