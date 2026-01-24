package ops

import (
	"database/sql"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/db"
)

// InventoryInput contains parameters for the Inventory operation.
type InventoryInput struct {
	Workspace      *string // optional filter
	Tag            *string // optional filter
	NamePrefix     *string // optional filter
	Limit          int     // default: 100, max: 500
	Offset         int     // default: 0
	IncludeDeleted bool
}

// InventoryOutput contains the result of the Inventory operation.
type InventoryOutput struct {
	Items      []capsule.CapsuleSummary `json:"items"`
	Pagination Pagination               `json:"pagination"`
	Sort       string                   `json:"sort"`
}

// Inventory retrieves capsule summaries across all workspaces with optional filters.
func Inventory(database *sql.DB, input InventoryInput) (*InventoryOutput, error) {
	// Normalize filters if present
	var filters db.InventoryFilters
	if input.Workspace != nil {
		workspace := capsule.Normalize(*input.Workspace)
		if workspace != "" {
			filters.Workspace = &workspace
		}
	}
	if input.Tag != nil {
		tag := *input.Tag
		if tag != "" {
			filters.Tag = &tag
		}
	}
	if input.NamePrefix != nil {
		prefix := capsule.Normalize(*input.NamePrefix)
		if prefix != "" {
			filters.NamePrefix = &prefix
		}
	}

	// Apply limit defaults and bounds
	limit := input.Limit
	if limit <= 0 {
		limit = DefaultInventoryLimit
	}
	if limit > MaxInventoryLimit {
		limit = MaxInventoryLimit
	}

	// Ensure offset is non-negative
	offset := max(input.Offset, 0)

	// Query database
	summaries, total, err := db.ListAll(database, filters, limit, offset, input.IncludeDeleted)
	if err != nil {
		return nil, err
	}

	// Ensure we return an empty array rather than nil
	if summaries == nil {
		summaries = []capsule.CapsuleSummary{}
	}

	// Calculate has_more
	hasMore := offset+len(summaries) < total

	return &InventoryOutput{
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
