package ops

import (
	"database/sql"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/db"
)

// ListInput contains parameters for the List operation.
type ListInput struct {
	Workspace      string // required, defaults to "default"
	Limit          int    // default: 20, max: 100
	Offset         int    // default: 0
	IncludeDeleted bool
}

// ListOutput contains the result of the List operation.
type ListOutput struct {
	Items      []capsule.CapsuleSummary `json:"items"`
	Pagination Pagination               `json:"pagination"`
	Sort       string                   `json:"sort"`
}

// List retrieves capsule summaries for a workspace with pagination.
func List(database *sql.DB, input ListInput) (*ListOutput, error) {
	// Normalize workspace
	workspace := capsule.Normalize(input.Workspace)
	if workspace == "" {
		workspace = "default"
	}

	// Apply limit defaults and bounds
	limit := input.Limit
	if limit <= 0 {
		limit = DefaultListLimit
	}
	if limit > MaxListLimit {
		limit = MaxListLimit
	}

	// Ensure offset is non-negative
	offset := max(input.Offset, 0)

	// Query database
	summaries, total, err := db.ListByWorkspace(database, workspace, limit, offset, input.IncludeDeleted)
	if err != nil {
		return nil, err
	}

	// Ensure we return an empty array rather than nil
	if summaries == nil {
		summaries = []capsule.CapsuleSummary{}
	}

	// Calculate has_more
	hasMore := offset+len(summaries) < total

	return &ListOutput{
		Items: summaries,
		Pagination: Pagination{
			Limit:   limit,
			Offset:  offset,
			HasMore: hasMore,
			Total:   total,
		},
		Sort: "updated_at_desc",
	}, nil
}
