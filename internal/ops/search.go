package ops

import (
	"context"
	"database/sql"
	"strings"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

// Search limits
const (
	DefaultSearchLimit = 20
	MaxSearchLimit     = 100
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
		items[i] = SearchResultItem{
			SummaryItem: SummaryItem{
				CapsuleSummary: r.Summary,
				FetchKey:       BuildFetchKey(r.Summary.Workspace, name, r.Summary.ID),
			},
			Snippet: truncateSnippet(r.Snippet, MaxSnippetChars),
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

// truncateSnippet truncates a snippet to maxChars while preserving word boundaries.
// Tries to cut at a word boundary to avoid mid-word truncation.
func truncateSnippet(s string, maxChars int) string {
	if len(s) <= maxChars {
		return s
	}

	// Find last space before maxChars
	truncated := s[:maxChars]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxChars/2 {
		// Cut at word boundary if we're not losing too much
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}
