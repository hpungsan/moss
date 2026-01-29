package ops

import (
	"context"
	"database/sql"
	"fmt"
	"html"
	"strings"
	"unicode/utf8"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

// Search limits
const (
	DefaultSearchLimit = 20
	MaxSearchLimit     = 100
	MaxQueryLength     = db.MaxSearchQueryChars
	MaxSnippetChars    = 300
)

// SearchInput contains parameters for the Search operation.
type SearchInput struct {
	Query          string  // required
	Workspace      *string // optional filter
	Tag            *string // optional filter
	RunID          *string // optional filter
	Phase          *string // optional filter
	Role           *string // optional filter
	Limit          int     // default: 20, max: 100
	Offset         int     // default: 0
	IncludeDeleted bool
}

// SearchResultItem wraps a SummaryItem with a match snippet.
type SearchResultItem struct {
	SummaryItem
	// Snippet is HTML-safe: user-controlled content is escaped; only <b>...</b>
	// highlight tags are present.
	Snippet string `json:"snippet"` // Match context (~300 chars max, <b> highlights)
}

// SearchOutput contains the result of the Search operation.
type SearchOutput struct {
	Items      []SearchResultItem `json:"items"`
	Pagination Pagination         `json:"pagination"`
	Sort       string             `json:"sort"` // "relevance"
}

// Search performs full-text search across capsules.
// Results are ranked by relevance (BM25) with title matches weighted 5x higher.
func Search(ctx context.Context, database *sql.DB, input SearchInput) (*SearchOutput, error) {
	// Validate query
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, errors.NewInvalidRequest("query is required")
	}
	if utf8.RuneCountInString(query) > MaxQueryLength {
		return nil, errors.NewInvalidRequest(fmt.Sprintf("query exceeds maximum length of %d characters", MaxQueryLength))
	}

	// Build filters
	var filters db.SearchFilters
	if input.Workspace != nil {
		workspace := capsule.Normalize(*input.Workspace)
		if workspace != "" {
			filters.Workspace = &workspace
		}
	}
	if input.Tag != nil {
		tag := strings.TrimSpace(*input.Tag)
		if tag != "" {
			filters.Tag = &tag
		}
	}
	filters.RunID = cleanOptionalString(input.RunID)
	filters.Phase = cleanOptionalString(input.Phase)
	filters.Role = cleanOptionalString(input.Role)

	// Apply limit defaults and bounds
	limit := input.Limit
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}

	// Ensure offset is non-negative
	offset := max(input.Offset, 0)

	// Query database
	results, total, err := db.SearchFullText(ctx, database, query, filters, limit, offset, input.IncludeDeleted)
	if err != nil {
		return nil, err
	}

	// Convert to output items
	items := make([]SearchResultItem, len(results))
	for i, r := range results {
		name := ""
		if r.Summary.Name != nil {
			name = *r.Summary.Name
		}

		// Process snippet:
		// 1. Escape user content to prevent XSS; convert internal markers to <b> tags
		// 2. Truncate to max length (preserves UTF-8 and closes unclosed tags)
		snippet := escapeSnippetHTML(r.Snippet)
		snippet = truncateSnippet(snippet, MaxSnippetChars)

		items[i] = SearchResultItem{
			SummaryItem: SummaryItem{
				CapsuleSummary: r.Summary,
				FetchKey:       BuildFetchKey(r.Summary.Workspace, name, r.Summary.ID),
			},
			Snippet: snippet,
		}
	}

	// Calculate has_more
	hasMore := offset+len(items) < total

	return &SearchOutput{
		Items: items,
		Pagination: Pagination{
			Limit:   limit,
			Offset:  offset,
			HasMore: hasMore,
			Total:   total,
		},
		Sort: "relevance",
	}, nil
}

// truncateSnippet truncates a snippet to approximately maxChars while:
// 1. Preserving valid UTF-8 (never splits multi-byte runes)
// 2. Preserving markup integrity (closes any open <b> tags)
// 3. Preferring word boundaries when possible
func truncateSnippet(s string, maxChars int) string {
	if maxChars <= 0 {
		return "..."
	}

	if len(s) <= maxChars {
		return s
	}

	// Find a safe truncation point that doesn't split UTF-8 runes
	truncateAt := maxChars
	for truncateAt > 0 && !utf8.RuneStart(s[truncateAt]) {
		truncateAt--
	}

	if truncateAt == 0 {
		// Edge case: couldn't find a safe point (shouldn't happen with valid UTF-8)
		return "..."
	}

	truncated := s[:truncateAt]

	// Avoid returning malformed HTML by trimming any partial tag/entity suffix.
	// At this point the only tags present should be <b> and </b>, and user content
	// may contain HTML entities (e.g., &lt;).
	if lastLT := strings.LastIndex(truncated, "<"); lastLT != -1 && !strings.Contains(truncated[lastLT:], ">") {
		truncated = truncated[:lastLT]
	}
	if lastAmp := strings.LastIndex(truncated, "&"); lastAmp != -1 && !strings.Contains(truncated[lastAmp:], ";") {
		truncated = truncated[:lastAmp]
	}

	// Try to cut at word boundary if we're not losing too much content
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > truncateAt/2 {
		truncated = truncated[:lastSpace]
	}

	// Fix unclosed <b> tags to maintain valid HTML structure
	// Count open and close tags
	openTags := strings.Count(truncated, "<b>")
	closeTags := strings.Count(truncated, "</b>")
	unclosedCount := openTags - closeTags

	// Close any unclosed <b> tags
	for range unclosedCount {
		truncated += "</b>"
	}

	return truncated + "..."
}

// escapeSnippetHTML escapes user content in a snippet while preserving our <b>
// highlight markers. This prevents XSS attacks from user-controlled capsule content.
//
// The snippet from SQLite FTS5 contains:
//   - User content (potentially malicious HTML/JS)
//   - Our markers: <b>, </b>, ...
//
// We need to escape the user content but preserve our markers.
func escapeSnippetHTML(s string) string {
	// Use unlikely placeholders that won't appear in normal content
	const (
		openPlaceholder  = "\x00MOSS_B_OPEN\x00"
		closePlaceholder = "\x00MOSS_B_CLOSE\x00"
		openMarker       = "[[[B]]]"
		closeMarker      = "[[[/B]]]"
	)

	// Step 1: Replace internal highlight markers with placeholders.
	// Markers come from SQLite snippet() start/end mark args in internal/db/queries.go.
	s = strings.ReplaceAll(s, openMarker, openPlaceholder)
	s = strings.ReplaceAll(s, closeMarker, closePlaceholder)

	// Step 2: Escape all HTML in user content
	s = html.EscapeString(s)

	// Step 3: Restore highlight tags (and only highlight tags).
	s = strings.ReplaceAll(s, openPlaceholder, "<b>")
	s = strings.ReplaceAll(s, closePlaceholder, "</b>")

	return s
}
