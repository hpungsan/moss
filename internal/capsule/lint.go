package capsule

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
)

// LintInput contains parameters for linting a capsule.
type LintInput struct {
	CapsuleText string
	MaxChars    int
	AllowThin   bool
}

// LintResult contains the results of linting a capsule.
type LintResult struct {
	Valid           bool
	MissingSections []string // canonical names of missing sections
	TooLarge        bool
	ActualChars     int
	MaxChars        int
}

// canonicalSections lists the required sections in canonical order.
var canonicalSections = []string{
	"Objective",
	"Current status",
	"Decisions",
	"Next actions",
	"Key locations",
	"Open questions",
}

// sectionSynonyms maps canonical section names to all accepted synonyms (lowercase).
var sectionSynonyms = map[string][]string{
	"Objective":      {"objective", "goal", "purpose"},
	"Current status": {"current status", "status", "state", "where we are"},
	"Decisions":      {"decisions", "decisions / constraints", "decisions/constraints", "constraints", "choices"},
	"Next actions":   {"next actions", "next steps", "action items", "todo", "tasks"},
	"Key locations":  {"key locations", "locations", "files", "paths", "references"},
	"Open questions": {"open questions", "open questions / risks", "open questions/risks", "questions", "risks", "unknowns"},
}

// Lint validates capsule content and returns a LintResult.
func Lint(input LintInput) *LintResult {
	result := &LintResult{
		Valid:       true,
		ActualChars: CountChars(input.CapsuleText),
		MaxChars:    input.MaxChars,
	}

	// Check size
	if input.MaxChars > 0 && result.ActualChars > input.MaxChars {
		result.TooLarge = true
		result.Valid = false
	}

	// Check required sections (unless allow_thin)
	if !input.AllowThin {
		result.MissingSections = findMissingSections(input.CapsuleText)
		if len(result.MissingSections) > 0 {
			result.Valid = false
		}
	}

	return result
}

// findMissingSections returns a list of canonical section names that are missing.
func findMissingSections(text string) []string {
	var missing []string

	for _, canonical := range canonicalSections {
		synonyms := sectionSynonyms[canonical]
		if !hasSection(text, synonyms) {
			missing = append(missing, canonical)
		}
	}

	return missing
}

// hasSection checks if any synonym is found in the text using any supported format.
func hasSection(text string, synonyms []string) bool {
	// Try markdown headers first
	if hasSectionMarkdown(text, synonyms) {
		return true
	}

	// Try colon-style
	if hasSectionColon(text, synonyms) {
		return true
	}

	// Try JSON format
	if hasSectionJSON(text, synonyms) {
		return true
	}

	return false
}

// hasSectionMarkdown checks for markdown headers: ## Section or # Section etc.
// Pattern: ^#{1,6}\s*<synonym> (case-insensitive, at start of line)
func hasSectionMarkdown(text string, synonyms []string) bool {
	for _, synonym := range synonyms {
		// Build regex for this synonym
		// Escape special regex chars in synonym
		escaped := regexp.QuoteMeta(synonym)
		pattern := `(?im)^#{1,6}\s*` + escaped + `\s*$`
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

// hasSectionColon checks for colon-style headers: Section:
// Pattern: ^<synonym>\s*: (case-insensitive, at start of line)
func hasSectionColon(text string, synonyms []string) bool {
	for _, synonym := range synonyms {
		// Build regex for this synonym
		escaped := regexp.QuoteMeta(synonym)
		pattern := `(?im)^` + escaped + `\s*:`
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

// hasSectionJSON checks if the text is valid JSON and contains any synonym as a key.
func hasSectionJSON(text string, synonyms []string) bool {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return false
	}

	var obj map[string]any
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return false
	}

	// Check each key against synonyms (case-insensitive)
	for key := range obj {
		keyLower := strings.ToLower(key)
		if slices.Contains(synonyms, keyLower) {
			return true
		}
	}

	return false
}
