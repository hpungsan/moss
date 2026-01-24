package capsule

import (
	"strings"
	"testing"
)

// validMarkdownCapsule contains all 6 required sections using markdown headers.
const validMarkdownCapsule = `## Objective
Build a user authentication system.

## Current status
Database schema is complete.

## Decisions
- Using JWT for tokens
- Session timeout: 24 hours

## Next actions
- Implement login endpoint
- Add password hashing

## Key locations
- cmd/auth/main.go
- internal/auth/handler.go

## Open questions
- Should we support OAuth?
`

// validColonCapsule contains all 6 required sections using colon-style.
const validColonCapsule = `Objective: Build a user authentication system.

Status: Database schema is complete.

Constraints: Using JWT for tokens.

Action items: Implement login endpoint.

Files: cmd/auth/main.go

Risks: Should we support OAuth?
`

// validJSONCapsule contains all 6 required sections as JSON.
const validJSONCapsule = `{
	"objective": "Build auth system",
	"status": "DB complete",
	"decisions": ["JWT tokens"],
	"next steps": ["login endpoint"],
	"locations": ["cmd/auth"],
	"questions": ["OAuth support?"]
}`

func TestLint_ValidMarkdown(t *testing.T) {
	input := LintInput{
		CapsuleText: validMarkdownCapsule,
		MaxChars:    12000,
		AllowThin:   false,
	}

	result := Lint(input)

	if !result.Valid {
		t.Errorf("Valid = false, want true; missing sections: %v", result.MissingSections)
	}
	if len(result.MissingSections) != 0 {
		t.Errorf("MissingSections = %v, want empty", result.MissingSections)
	}
}

func TestLint_ValidColon(t *testing.T) {
	input := LintInput{
		CapsuleText: validColonCapsule,
		MaxChars:    12000,
		AllowThin:   false,
	}

	result := Lint(input)

	if !result.Valid {
		t.Errorf("Valid = false, want true; missing sections: %v", result.MissingSections)
	}
}

func TestLint_ValidJSON(t *testing.T) {
	input := LintInput{
		CapsuleText: validJSONCapsule,
		MaxChars:    12000,
		AllowThin:   false,
	}

	result := Lint(input)

	if !result.Valid {
		t.Errorf("Valid = false, want true; missing sections: %v", result.MissingSections)
	}
}

func TestLint_MissingSections(t *testing.T) {
	// Only has Objective
	input := LintInput{
		CapsuleText: "## Objective\nBuild something.",
		MaxChars:    12000,
		AllowThin:   false,
	}

	result := Lint(input)

	if result.Valid {
		t.Error("Valid = true, want false")
	}

	// Should be missing 5 sections
	if len(result.MissingSections) != 5 {
		t.Errorf("MissingSections count = %d, want 5; got: %v", len(result.MissingSections), result.MissingSections)
	}

	// Check specific missing sections
	expected := map[string]bool{
		"Current status": true,
		"Decisions":      true,
		"Next actions":   true,
		"Key locations":  true,
		"Open questions": true,
	}
	for _, missing := range result.MissingSections {
		if !expected[missing] {
			t.Errorf("Unexpected missing section: %q", missing)
		}
	}
}

func TestLint_AllowThin(t *testing.T) {
	// Only has Objective, but allow_thin=true
	input := LintInput{
		CapsuleText: "## Objective\nBuild something.",
		MaxChars:    12000,
		AllowThin:   true,
	}

	result := Lint(input)

	if !result.Valid {
		t.Error("Valid = false, want true (allow_thin=true)")
	}
	if len(result.MissingSections) != 0 {
		t.Errorf("MissingSections = %v, want empty (allow_thin bypasses)", result.MissingSections)
	}
}

func TestLint_TooLarge(t *testing.T) {
	// Create text that exceeds limit
	largeText := strings.Repeat("x", 15000)
	input := LintInput{
		CapsuleText: largeText,
		MaxChars:    12000,
		AllowThin:   true, // Allow thin to isolate size check
	}

	result := Lint(input)

	if result.Valid {
		t.Error("Valid = true, want false (too large)")
	}
	if !result.TooLarge {
		t.Error("TooLarge = false, want true")
	}
	if result.ActualChars != 15000 {
		t.Errorf("ActualChars = %d, want 15000", result.ActualChars)
	}
}

func TestLint_SizeWithUTF8(t *testing.T) {
	// UTF-8 characters should count as single characters
	// "日本語" is 3 characters (9 bytes)
	input := LintInput{
		CapsuleText: "日本語",
		MaxChars:    5,
		AllowThin:   true,
	}

	result := Lint(input)

	if !result.Valid {
		t.Error("Valid = false, want true (3 chars <= 5 limit)")
	}
	if result.ActualChars != 3 {
		t.Errorf("ActualChars = %d, want 3", result.ActualChars)
	}
}

func TestLint_SynonymDetection(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		canonical  string
		shouldFind bool
	}{
		// Objective synonyms
		{"Objective via Goal", "## Goal\nTest", "Objective", true},
		{"Objective via Purpose", "Purpose: Test", "Objective", true},

		// Current status synonyms
		{"Current status via Status", "## Status\nTest", "Current status", true},
		{"Current status via State", "State: Test", "Current status", true},
		{"Current status via Where we are", "## Where we are\nTest", "Current status", true},

		// Decisions synonyms
		{"Decisions via Constraints", "## Constraints\nTest", "Decisions", true},
		{"Decisions via Choices", "Choices: Test", "Decisions", true},
		{"Decisions via Decisions/constraints", "## Decisions/constraints\nTest", "Decisions", true},

		// Next actions synonyms
		{"Next actions via Next steps", "## Next steps\nTest", "Next actions", true},
		{"Next actions via Action items", "Action items: Test", "Next actions", true},
		{"Next actions via TODO", "## TODO\nTest", "Next actions", true},
		{"Next actions via Tasks", "Tasks: Test", "Next actions", true},

		// Key locations synonyms
		{"Key locations via Locations", "## Locations\nTest", "Key locations", true},
		{"Key locations via Files", "Files: Test", "Key locations", true},
		{"Key locations via Paths", "## Paths\nTest", "Key locations", true},
		{"Key locations via References", "References: Test", "Key locations", true},

		// Open questions synonyms
		{"Open questions via Questions", "## Questions\nTest", "Open questions", true},
		{"Open questions via Risks", "Risks: Test", "Open questions", true},
		{"Open questions via Unknowns", "## Unknowns\nTest", "Open questions", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synonyms := sectionSynonyms[tt.canonical]
			found := hasSection(tt.text, synonyms)
			if found != tt.shouldFind {
				t.Errorf("hasSection() = %v, want %v", found, tt.shouldFind)
			}
		})
	}
}

func TestLint_CaseInsensitivity(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"Uppercase", "## OBJECTIVE\nTest"},
		{"Lowercase", "## objective\nTest"},
		{"Mixed case", "## Objective\nTest"},
		{"Colon uppercase", "OBJECTIVE: Test"},
		{"Colon mixed", "Objective: Test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synonyms := sectionSynonyms["Objective"]
			if !hasSection(tt.text, synonyms) {
				t.Errorf("hasSection() = false, want true for %q", tt.text)
			}
		})
	}
}

func TestLint_MarkdownHeaderLevels(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"H1", "# Objective\nTest"},
		{"H2", "## Objective\nTest"},
		{"H3", "### Objective\nTest"},
		{"H4", "#### Objective\nTest"},
		{"H5", "##### Objective\nTest"},
		{"H6", "###### Objective\nTest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synonyms := sectionSynonyms["Objective"]
			if !hasSectionMarkdown(tt.text, synonyms) {
				t.Errorf("hasSectionMarkdown() = false, want true for %q", tt.text)
			}
		})
	}
}

func TestLint_NoFalsePositives(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"Word in middle of line", "This objective is clear."},
		{"Word without header marker", "Objective of this project"},
		{"Partial match in word", "Subjective opinion"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synonyms := sectionSynonyms["Objective"]
			if hasSection(tt.text, synonyms) {
				t.Errorf("hasSection() = true, want false for %q (false positive)", tt.text)
			}
		})
	}
}

func TestLint_JSONFormat(t *testing.T) {
	tests := []struct {
		name       string
		json       string
		canonical  string
		shouldFind bool
	}{
		{"lowercase key", `{"objective": "test"}`, "Objective", true},
		{"uppercase key", `{"OBJECTIVE": "test"}`, "Objective", true},
		{"synonym key", `{"goal": "test"}`, "Objective", true},
		{"next steps key", `{"next steps": "test"}`, "Next actions", true},
		{"non-matching key", `{"foo": "test"}`, "Objective", false},
		{"invalid json", `{invalid}`, "Objective", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synonyms := sectionSynonyms[tt.canonical]
			found := hasSectionJSON(tt.json, synonyms)
			if found != tt.shouldFind {
				t.Errorf("hasSectionJSON() = %v, want %v", found, tt.shouldFind)
			}
		})
	}
}

func TestLint_EmptyInput(t *testing.T) {
	input := LintInput{
		CapsuleText: "",
		MaxChars:    12000,
		AllowThin:   false,
	}

	result := Lint(input)

	if result.Valid {
		t.Error("Valid = true, want false (empty input missing all sections)")
	}
	if len(result.MissingSections) != 6 {
		t.Errorf("MissingSections count = %d, want 6", len(result.MissingSections))
	}
	if result.ActualChars != 0 {
		t.Errorf("ActualChars = %d, want 0", result.ActualChars)
	}
}

func TestLint_ZeroMaxChars(t *testing.T) {
	// MaxChars=0 should not trigger size check (unlimited)
	input := LintInput{
		CapsuleText: strings.Repeat("x", 50000),
		MaxChars:    0,
		AllowThin:   true,
	}

	result := Lint(input)

	if !result.Valid {
		t.Error("Valid = false, want true (MaxChars=0 means no limit)")
	}
	if result.TooLarge {
		t.Error("TooLarge = true, want false (MaxChars=0 means no limit)")
	}
}
