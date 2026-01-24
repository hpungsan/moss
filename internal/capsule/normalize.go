package capsule

import (
	"math"
	"regexp"
	"strings"
	"unicode/utf8"
)

// whitespaceRegex matches one or more whitespace characters
var whitespaceRegex = regexp.MustCompile(`\s+`)

// Normalize normalizes a string per DESIGN.md §4.2:
// 1. Trim leading/trailing whitespace
// 2. Lowercase
// 3. Collapse internal whitespace to single spaces
func Normalize(s string) string {
	// Trim leading/trailing whitespace
	s = strings.TrimSpace(s)

	// Lowercase
	s = strings.ToLower(s)

	// Collapse internal whitespace to single spaces
	s = whitespaceRegex.ReplaceAllString(s, " ")

	return s
}

// CountChars returns the character count as runes (not bytes).
// This correctly handles multi-byte UTF-8 characters.
func CountChars(text string) int {
	return utf8.RuneCountInString(text)
}

// EstimateTokens estimates token count using a word-based heuristic.
// Uses 1.3x multiplier on word count as per DESIGN.md.
func EstimateTokens(text string) int {
	words := strings.Fields(strings.TrimSpace(text))
	return int(math.Ceil(float64(len(words)) * 1.3))
}
