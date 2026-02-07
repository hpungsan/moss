package capsule

import (
	"regexp"
	"slices"
	"strings"
)

// Section represents a parsed section boundary.
type Section struct {
	Header        string // Full header line "## Design Reviews"
	HeaderName    string // Just the name part "Design Reviews"
	Canonical     string // Canonical name if matched, empty for custom
	HeaderStart   int    // Byte offset of header start
	HeaderEnd     int    // Byte offset after header line (including \n)
	ContentStart  int    // Byte offset where content starts
	ContentEnd    int    // Byte offset where content ends (before next section or EOF)
	IsPlaceholder bool   // True if content is only placeholder
}

// headerPattern matches markdown headers (h1-h6) at the start of a line.
// Groups: full match, hash symbols, header text
// Note: We match until end of line but don't consume the newline itself.
// Trailing spaces/tabs on the header line are trimmed by the [^\n]+ group.
var headerPattern = regexp.MustCompile(`(?m)^(#{1,6})\s+([^\n]+?)[ \t]*$`)

// fencePattern matches fenced code block delimiters (``` or ~~~) at the start of a line,
// allowing 0-3 spaces of indentation per CommonMark spec. Captures the fence characters
// (backticks or tildes) separately from leading whitespace.
var fencePattern = regexp.MustCompile("(?m)^[ ]{0,3}(`{3,}|~{3,})")

// fencedRanges returns byte offset ranges [start, end) for fenced code blocks in text.
// Properly pairs opening and closing fences: closing fence must use the same character
// (backtick or tilde) and be at least as long as the opening fence.
func fencedRanges(text string) [][2]int {
	matches := fencePattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) < 2 {
		return nil
	}

	var ranges [][2]int
	var openChar byte
	var openLen int
	var openStart int
	inFence := false

	for _, match := range matches {
		// match indices: [fullStart, fullEnd, fenceCharsStart, fenceCharsEnd]
		fenceChars := text[match[2]:match[3]]
		char := fenceChars[0]
		fenceLen := len(fenceChars)

		if !inFence {
			// Opening fence
			openChar = char
			openLen = fenceLen
			openStart = match[0]
			inFence = true
		} else if char == openChar && fenceLen >= openLen {
			// Valid closing fence: same character, at least as long
			ranges = append(ranges, [2]int{openStart, match[1]})
			inFence = false
		}
		// Otherwise: different char or shorter fence inside open block — skip
	}
	return ranges
}

// insideFence returns true if byte offset pos falls inside any fenced range.
func insideFence(pos int, ranges [][2]int) bool {
	for _, r := range ranges {
		if pos >= r[0] && pos < r[1] {
			return true
		}
	}
	return false
}

// placeholderPatterns are common placeholder values (case-insensitive, after trimming).
var placeholderPatterns = []string{
	"(pending)",
	"(none)",
	"(empty)",
	"(tbd)",
	"(n/a)",
	"tbd",
	"n/a",
	"none",
	"pending",
	"-",
}

// ParseSections finds all markdown section headers and their boundaries.
// Returns nil if no sections found (e.g., JSON format capsule).
// Headers inside fenced code blocks (``` or ~~~) are ignored.
func ParseSections(text string) []Section {
	allMatches := headerPattern.FindAllStringSubmatchIndex(text, -1)
	if len(allMatches) == 0 {
		return nil
	}

	// Filter out matches inside fenced code blocks
	fences := fencedRanges(text)
	matches := allMatches
	if len(fences) > 0 {
		matches = make([][]int, 0, len(allMatches))
		for _, m := range allMatches {
			if !insideFence(m[0], fences) {
				matches = append(matches, m)
			}
		}
		if len(matches) == 0 {
			return nil
		}
	}

	sections := make([]Section, len(matches))
	for i, match := range matches {
		// match indices: [fullStart, fullEnd, hashStart, hashEnd, nameStart, nameEnd]
		headerStart := match[0]
		headerEnd := match[1]
		headerName := text[match[4]:match[5]]
		fullHeader := text[match[0]:match[1]]

		// Content starts after the header line (skip the newline following the header)
		contentStart := headerEnd
		if contentStart < len(text) && text[contentStart] == '\n' {
			contentStart++
		}

		// Content ends at next section start or EOF
		var contentEnd int
		if i+1 < len(matches) {
			contentEnd = matches[i+1][0]
		} else {
			contentEnd = len(text)
		}

		// Check for canonical match
		canonical := MatchCanonical(headerName)

		// Extract content and check if placeholder
		content := ""
		if contentStart < contentEnd {
			content = text[contentStart:contentEnd]
		}
		isPlaceholder := isPlaceholderContent(content)

		sections[i] = Section{
			Header:        fullHeader,
			HeaderName:    headerName,
			Canonical:     canonical,
			HeaderStart:   headerStart,
			HeaderEnd:     headerEnd,
			ContentStart:  contentStart,
			ContentEnd:    contentEnd,
			IsPlaceholder: isPlaceholder,
		}
	}

	return sections
}

// FindSection finds a section by name (synonym-aware, case-insensitive).
// First checks if input matches a canonical section synonym, then falls back
// to exact case-insensitive match on header name.
// Used by lint and other operations that benefit from flexible matching.
func FindSection(sections []Section, name string) *Section {
	if len(sections) == 0 {
		return nil
	}

	nameLower := strings.ToLower(strings.TrimSpace(name))

	// First try canonical match (synonym-aware)
	canonical := MatchCanonical(name)
	if canonical != "" {
		for i := range sections {
			if sections[i].Canonical == canonical {
				return &sections[i]
			}
		}
	}

	// Fall back to exact case-insensitive header name match
	for i := range sections {
		if strings.ToLower(sections[i].HeaderName) == nameLower {
			return &sections[i]
		}
	}

	return nil
}

// FindSectionExact finds a section by exact header name (case-insensitive).
// No synonym resolution — returns the section whose HeaderName matches exactly.
// Used by capsule_append where precise targeting is required.
func FindSectionExact(sections []Section, name string) *Section {
	if len(sections) == 0 {
		return nil
	}

	nameLower := strings.ToLower(strings.TrimSpace(name))

	for i := range sections {
		if strings.ToLower(strings.TrimSpace(sections[i].HeaderName)) == nameLower {
			return &sections[i]
		}
	}

	return nil
}

// SectionNames returns the list of header names from parsed sections.
// Useful for error messages listing available sections.
func SectionNames(sections []Section) []string {
	names := make([]string, len(sections))
	for i, s := range sections {
		names[i] = s.HeaderName
	}
	return names
}

// InsertContent inserts content into a section (replace if placeholder, else append).
// Returns the modified text.
func InsertContent(text string, section *Section, content string) string {
	if section.IsPlaceholder {
		// Replace: remove placeholder content entirely
		// The placeholder content may include blank lines, so we trim them
		return text[:section.ContentStart] + content + "\n" + text[section.ContentEnd:]
	}

	// Append: normalize to blank line separator
	// Trim trailing whitespace from existing content to ensure consistent formatting
	existingContent := strings.TrimRight(text[section.ContentStart:section.ContentEnd], " \t\n")
	return text[:section.ContentStart] + existingContent + "\n\n" + content + "\n" + text[section.ContentEnd:]
}

// isPlaceholderContent checks if content is only placeholder text.
// Content with any non-placeholder text returns false.
func isPlaceholderContent(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return true
	}

	trimmedLower := strings.ToLower(trimmed)
	return slices.Contains(placeholderPatterns, trimmedLower)
}
