package capsule

import (
	"strings"
	"testing"
)

var testCapsule = `## Objective
Test the append functionality

## Current status
In progress

## Decisions
(pending)

## Next actions
- Write tests
- Implement feature

## Key locations
- internal/capsule/sections.go

## Open questions
None
`

func TestParseSections_StandardCapsule(t *testing.T) {
	sections := ParseSections(testCapsule)
	if len(sections) != 6 {
		t.Fatalf("Expected 6 sections, got %d", len(sections))
	}

	// Verify section names
	expected := []string{"Objective", "Current status", "Decisions", "Next actions", "Key locations", "Open questions"}
	for i, want := range expected {
		if sections[i].HeaderName != want {
			t.Errorf("Section %d: HeaderName = %q, want %q", i, sections[i].HeaderName, want)
		}
	}
}

func TestParseSections_CanonicalMatch(t *testing.T) {
	sections := ParseSections(testCapsule)

	// All sections should have canonical matches
	for i, s := range sections {
		if s.Canonical == "" {
			t.Errorf("Section %d (%s) has no canonical match", i, s.HeaderName)
		}
	}
}

func TestParseSections_CustomSection(t *testing.T) {
	text := `## Objective
Goal here

## Design Reviews
Round 1: APPROVE

## Status
Done
`
	sections := ParseSections(text)
	if len(sections) != 3 {
		t.Fatalf("Expected 3 sections, got %d", len(sections))
	}

	// Design Reviews is custom (no canonical match)
	if sections[1].Canonical != "" {
		t.Errorf("Design Reviews should have no canonical match, got %q", sections[1].Canonical)
	}

	// Status should match canonical "Current status"
	if sections[2].Canonical != "Current status" {
		t.Errorf("Status canonical = %q, want 'Current status'", sections[2].Canonical)
	}
}

func TestParseSections_HeaderLevels(t *testing.T) {
	text := `# Objective
Level 1

### Subsection
Level 3

###### Deep
Level 6
`
	sections := ParseSections(text)
	if len(sections) != 3 {
		t.Fatalf("Expected 3 sections, got %d", len(sections))
	}

	if sections[0].HeaderName != "Objective" {
		t.Errorf("Section 0 name = %q, want 'Objective'", sections[0].HeaderName)
	}
	if sections[1].HeaderName != "Subsection" {
		t.Errorf("Section 1 name = %q, want 'Subsection'", sections[1].HeaderName)
	}
	if sections[2].HeaderName != "Deep" {
		t.Errorf("Section 2 name = %q, want 'Deep'", sections[2].HeaderName)
	}
}

func TestParseSections_NoSections(t *testing.T) {
	// JSON format capsule - no markdown headers
	text := `{"objective": "test", "status": "done"}`
	sections := ParseSections(text)
	if sections != nil {
		t.Errorf("Expected nil for non-markdown text, got %d sections", len(sections))
	}
}

func TestParseSections_PlaceholderDetection(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		isPlaceholder bool
	}{
		{"pending parentheses", "## Status\n(pending)\n", true},
		{"pending lowercase", "## Status\npending\n", true},
		{"tbd uppercase", "## Status\nTBD\n", true},
		{"tbd mixed case", "## Status\nTbD\n", true},
		{"n/a", "## Status\nN/A\n", true},
		{"none", "## Status\nnone\n", true},
		{"dash", "## Status\n-\n", true},
		{"empty", "## Status\n\n", true},
		{"whitespace only", "## Status\n   \n", true},
		{"real content", "## Status\nIn progress\n", false},
		{"pending with more", "## Status\npending\nactual content\n", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sections := ParseSections(tc.text)
			if len(sections) == 0 {
				t.Fatal("No sections parsed")
			}
			if sections[0].IsPlaceholder != tc.isPlaceholder {
				t.Errorf("IsPlaceholder = %v, want %v", sections[0].IsPlaceholder, tc.isPlaceholder)
			}
		})
	}
}

func TestFindSection_ByCanonical(t *testing.T) {
	sections := ParseSections(testCapsule)

	// Find by synonym "Status" → should match "Current status"
	s := FindSection(sections, "Status")
	if s == nil {
		t.Fatal("FindSection returned nil for 'Status'")
	}
	if s.HeaderName != "Current status" {
		t.Errorf("Found section = %q, want 'Current status'", s.HeaderName)
	}
}

func TestFindSection_CaseInsensitive(t *testing.T) {
	sections := ParseSections(testCapsule)

	tests := []string{"OBJECTIVE", "objective", "Objective", "OBJective"}
	for _, name := range tests {
		s := FindSection(sections, name)
		if s == nil || s.HeaderName != "Objective" {
			t.Errorf("FindSection(%q) failed", name)
		}
	}
}

func TestFindSection_CustomSection(t *testing.T) {
	text := `## Objective
Goal

## Design Reviews
Round 1
`
	sections := ParseSections(text)

	// Find custom section by exact name
	s := FindSection(sections, "Design Reviews")
	if s == nil {
		t.Fatal("FindSection returned nil for 'Design Reviews'")
	}
	if s.HeaderName != "Design Reviews" {
		t.Errorf("Found section = %q, want 'Design Reviews'", s.HeaderName)
	}

	// Case insensitive
	s = FindSection(sections, "design reviews")
	if s == nil {
		t.Fatal("FindSection case insensitive failed")
	}
}

func TestFindSection_NotFound(t *testing.T) {
	sections := ParseSections(testCapsule)

	s := FindSection(sections, "Nonexistent")
	if s != nil {
		t.Errorf("FindSection should return nil for nonexistent section, got %q", s.HeaderName)
	}
}

func TestInsertContent_Append(t *testing.T) {
	text := `## Objective
Goal here

## Status
In progress
`
	sections := ParseSections(text)
	s := FindSection(sections, "Status")

	result := InsertContent(text, s, "Update: almost done")

	// Verify the content was appended with blank line separator
	expected := `## Objective
Goal here

## Status
In progress

Update: almost done
`
	if result != expected {
		t.Errorf("InsertContent result:\n%s\n\nExpected:\n%s", result, expected)
	}
}

func TestInsertContent_ReplacePlaceholder(t *testing.T) {
	text := `## Objective
Goal here

## Status
(pending)
`
	sections := ParseSections(text)
	s := FindSection(sections, "Status")

	result := InsertContent(text, s, "Completed successfully")

	expected := `## Objective
Goal here

## Status
Completed successfully
`
	if result != expected {
		t.Errorf("InsertContent result:\n%s\n\nExpected:\n%s", result, expected)
	}
}

func TestInsertContent_LastSection(t *testing.T) {
	text := `## Objective
Goal here

## Status
In progress`

	sections := ParseSections(text)
	s := FindSection(sections, "Status")

	result := InsertContent(text, s, "Update: done")

	expected := `## Objective
Goal here

## Status
In progress

Update: done
`
	if result != expected {
		t.Errorf("InsertContent result:\n%s\n\nExpected:\n%s", result, expected)
	}
}

func TestInsertContent_EmptySection(t *testing.T) {
	text := `## Objective
Goal here

## Status

## Decisions
Made some
`
	sections := ParseSections(text)
	s := FindSection(sections, "Status")

	result := InsertContent(text, s, "Now in progress")

	expected := `## Objective
Goal here

## Status
Now in progress
## Decisions
Made some
`
	if result != expected {
		t.Errorf("InsertContent result:\n%s\n\nExpected:\n%s", result, expected)
	}
}

func TestInsertContent_MiddleSection(t *testing.T) {
	// Test appending to a middle section (not last, not placeholder)
	// Note: blank line separator is added BEFORE new content, not after
	text := `## Objective
Goal here

## Status
In progress

## Decisions
Decision 1
`
	sections := ParseSections(text)
	s := FindSection(sections, "Status")

	result := InsertContent(text, s, "Update: blocked")

	// The blank line from original is consumed; new content gets \n\n before it, \n after
	expected := `## Objective
Goal here

## Status
In progress

Update: blocked
## Decisions
Decision 1
`
	if result != expected {
		t.Errorf("InsertContent result:\n%s\n\nExpected:\n%s", result, expected)
	}
}

func TestInsertContent_MiddleSectionNoGap(t *testing.T) {
	// Test appending to middle section with no blank line before next section
	text := `## Objective
Goal
## Status
Done
## Decisions
Made
`
	sections := ParseSections(text)
	s := FindSection(sections, "Status")

	result := InsertContent(text, s, "Extra info")

	// Blank line is added as separator, original tight gap preserved after
	expected := `## Objective
Goal
## Status
Done

Extra info
## Decisions
Made
`
	if result != expected {
		t.Errorf("InsertContent result:\n%s\n\nExpected:\n%s", result, expected)
	}
}

func TestParseSections_ContentBoundaries(t *testing.T) {
	text := `## Objective
Line 1
Line 2

## Status
In progress
`
	sections := ParseSections(text)

	// Verify Objective content boundaries
	obj := sections[0]
	content := text[obj.ContentStart:obj.ContentEnd]
	expected := "Line 1\nLine 2\n\n"
	if content != expected {
		t.Errorf("Objective content = %q, want %q", content, expected)
	}

	// Verify Status content boundaries
	status := sections[1]
	content = text[status.ContentStart:status.ContentEnd]
	expected = "In progress\n"
	if content != expected {
		t.Errorf("Status content = %q, want %q", content, expected)
	}
}

func TestSynonymMatching(t *testing.T) {
	// Test various synonym matches
	tests := []struct {
		input    string
		wantName string
	}{
		{"Status", "Current status"},
		{"state", "Current status"},
		{"where we are", "Current status"},
		{"objective", "Objective"},
		{"goal", "Objective"},
		{"next steps", "Next actions"},
		{"todo", "Next actions"},
		{"action items", "Next actions"},
		{"questions", "Open questions"},
		{"risks", "Open questions"},
		{"unknowns", "Open questions"},
		{"files", "Key locations"},
		{"paths", "Key locations"},
		{"references", "Key locations"},
		{"constraints", "Decisions"},
		{"choices", "Decisions"},
	}

	// Build a capsule with canonical sections
	text := `## Objective
Goal

## Current status
Status here

## Decisions
Decisions made

## Next actions
Actions

## Key locations
Locations

## Open questions
Questions
`
	sections := ParseSections(text)

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			s := FindSection(sections, tc.input)
			if s == nil {
				t.Fatalf("FindSection(%q) returned nil", tc.input)
			}
			if s.HeaderName != tc.wantName {
				t.Errorf("FindSection(%q) = %q, want %q", tc.input, s.HeaderName, tc.wantName)
			}
		})
	}
}

func TestFindSectionExact(t *testing.T) {
	text := `## Objective
Goal

## Current status
In progress

## Design Reviews
Round 1
`
	sections := ParseSections(text)

	// Exact match works
	s := FindSectionExact(sections, "Objective")
	if s == nil || s.HeaderName != "Objective" {
		t.Errorf("FindSectionExact('Objective') failed")
	}

	// Case-insensitive exact match works
	s = FindSectionExact(sections, "objective")
	if s == nil || s.HeaderName != "Objective" {
		t.Errorf("FindSectionExact('objective') case-insensitive failed")
	}

	// Exact match with spaces
	s = FindSectionExact(sections, "Current status")
	if s == nil || s.HeaderName != "Current status" {
		t.Errorf("FindSectionExact('Current status') failed")
	}

	// Custom section works
	s = FindSectionExact(sections, "Design Reviews")
	if s == nil || s.HeaderName != "Design Reviews" {
		t.Errorf("FindSectionExact('Design Reviews') failed")
	}

	// Synonym does NOT match (this is the key difference from FindSection)
	s = FindSectionExact(sections, "Status")
	if s != nil {
		t.Errorf("FindSectionExact('Status') should return nil (no synonym matching), got %q", s.HeaderName)
	}

	s = FindSectionExact(sections, "goal")
	if s != nil {
		t.Errorf("FindSectionExact('goal') should return nil (no synonym matching), got %q", s.HeaderName)
	}

	// Not found
	s = FindSectionExact(sections, "Nonexistent")
	if s != nil {
		t.Errorf("FindSectionExact('Nonexistent') should return nil")
	}
}

func TestParseSections_IgnoresHeadersInFencedCodeBlocks(t *testing.T) {
	text := "## Objective\nGoal here\n\n## Key locations\n```go\n// ## Decisions\nfunc main() {}\n```\n\n## Decisions\nUsed JWT\n"
	sections := ParseSections(text)

	// Should find 3 real sections, NOT the ## Decisions inside the code fence
	if len(sections) != 3 {
		t.Fatalf("Expected 3 sections, got %d", len(sections))
	}

	expected := []string{"Objective", "Key locations", "Decisions"}
	for i, want := range expected {
		if sections[i].HeaderName != want {
			t.Errorf("Section %d: HeaderName = %q, want %q", i, sections[i].HeaderName, want)
		}
	}

	// Key locations content should include the full code fence block
	klContent := text[sections[1].ContentStart:sections[1].ContentEnd]
	if !strings.Contains(klContent, "```go") {
		t.Error("Key locations content should include the code fence")
	}
	if !strings.Contains(klContent, "func main()") {
		t.Error("Key locations content should include code inside the fence")
	}
}

func TestParseSections_IgnoresHeadersInTildeFence(t *testing.T) {
	text := "## Objective\nGoal\n\n~~~\n## Fake Header\nContent\n~~~\n\n## Status\nDone\n"
	sections := ParseSections(text)

	if len(sections) != 2 {
		t.Fatalf("Expected 2 sections, got %d", len(sections))
	}
	if sections[0].HeaderName != "Objective" {
		t.Errorf("Section 0 = %q, want 'Objective'", sections[0].HeaderName)
	}
	if sections[1].HeaderName != "Status" {
		t.Errorf("Section 1 = %q, want 'Status'", sections[1].HeaderName)
	}
}

func TestParseSections_NoFences_Unchanged(t *testing.T) {
	// Ensure normal capsules without fences still work identically
	text := "## Objective\nGoal\n\n## Status\nDone\n"
	sections := ParseSections(text)
	if len(sections) != 2 {
		t.Fatalf("Expected 2 sections, got %d", len(sections))
	}
	if sections[0].HeaderName != "Objective" || sections[1].HeaderName != "Status" {
		t.Error("Sections should parse normally without fences")
	}
}

func TestParseSections_UnclosedFence_NoFiltering(t *testing.T) {
	// An unclosed fence (only one ```) should not filter anything
	text := "## Objective\nGoal\n\n```\n## Status\nDone\n"
	sections := ParseSections(text)

	// With only one fence delimiter, no ranges are produced — all headers visible
	if len(sections) != 2 {
		t.Fatalf("Expected 2 sections (unclosed fence = no filtering), got %d", len(sections))
	}
}

func TestParseSections_IndentedFence(t *testing.T) {
	// CommonMark allows 0-3 spaces before fence delimiter
	text := "## Objective\nGoal\n\n   ```\n## Fake\nContent\n   ```\n\n## Status\nDone\n"
	sections := ParseSections(text)

	if len(sections) != 2 {
		t.Fatalf("Expected 2 sections (indented fence should be recognized), got %d", len(sections))
	}
	if sections[0].HeaderName != "Objective" {
		t.Errorf("Section 0 = %q, want 'Objective'", sections[0].HeaderName)
	}
	if sections[1].HeaderName != "Status" {
		t.Errorf("Section 1 = %q, want 'Status'", sections[1].HeaderName)
	}
}

func TestParseSections_FenceTypeMustMatch(t *testing.T) {
	// Closing fence must use same character as opening fence.
	// Here ~~~ opens but ``` should NOT close it — ## Fake stays inside the fence.
	text := "## Objective\nGoal\n\n~~~\n## Fake\n```\nstill in fence\n~~~\n\n## Status\nDone\n"
	sections := ParseSections(text)

	if len(sections) != 2 {
		t.Fatalf("Expected 2 sections (mismatched fence type should not close), got %d", len(sections))
	}
	if sections[0].HeaderName != "Objective" {
		t.Errorf("Section 0 = %q, want 'Objective'", sections[0].HeaderName)
	}
	if sections[1].HeaderName != "Status" {
		t.Errorf("Section 1 = %q, want 'Status'", sections[1].HeaderName)
	}
}

func TestParseSections_ClosingFenceMustBeAtLeastAsLong(t *testing.T) {
	// Opening with ```` (4 backticks) — closing ``` (3) should NOT close it
	text := "## Objective\nGoal\n\n````\n## Fake\n```\nstill in fence\n````\n\n## Status\nDone\n"
	sections := ParseSections(text)

	if len(sections) != 2 {
		t.Fatalf("Expected 2 sections (shorter fence should not close), got %d", len(sections))
	}
	if sections[0].HeaderName != "Objective" {
		t.Errorf("Section 0 = %q, want 'Objective'", sections[0].HeaderName)
	}
	if sections[1].HeaderName != "Status" {
		t.Errorf("Section 1 = %q, want 'Status'", sections[1].HeaderName)
	}
}

func TestSectionNames(t *testing.T) {
	text := `## Objective
Goal

## Status
Done

## Custom Section
Content
`
	sections := ParseSections(text)
	names := SectionNames(sections)

	if len(names) != 3 {
		t.Fatalf("SectionNames returned %d names, want 3", len(names))
	}

	expected := []string{"Objective", "Status", "Custom Section"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("SectionNames[%d] = %q, want %q", i, names[i], want)
		}
	}
}
