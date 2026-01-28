package ops

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

// BulkDeleteInput contains parameters for the BulkDelete operation.
type BulkDeleteInput struct {
	Workspace  *string
	Tag        *string
	NamePrefix *string
	RunID      *string
	Phase      *string
	Role       *string
}

// BulkDeleteOutput contains the result of the BulkDelete operation.
type BulkDeleteOutput struct {
	Deleted int    `json:"deleted"`
	Message string `json:"message"`
}

// BulkDelete soft-deletes all active capsules matching the given filters.
// At least one filter must be provided (safety guard).
func BulkDelete(ctx context.Context, database *sql.DB, input BulkDeleteInput) (*BulkDeleteOutput, error) {
	// Phase 1: at least one filter must be non-nil
	if !hasAnyFilter(input) {
		return nil, errors.NewInvalidRequest("at least one filter is required")
	}

	// Normalize filters
	var filters db.InventoryFilters
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
	if input.NamePrefix != nil {
		prefix := capsule.Normalize(*input.NamePrefix)
		if prefix != "" {
			filters.NamePrefix = &prefix
		}
	}
	filters.RunID = cleanOptionalString(input.RunID)
	filters.Phase = cleanOptionalString(input.Phase)
	filters.Role = cleanOptionalString(input.Role)

	// Phase 2: at least one filter must be non-empty after normalization
	if !hasAnyEffectiveFilter(filters) {
		return nil, errors.NewInvalidRequest("at least one filter must be non-empty after normalization")
	}

	count, err := db.BulkSoftDelete(ctx, database, filters)
	if err != nil {
		return nil, err
	}

	return &BulkDeleteOutput{
		Deleted: count,
		Message: formatBulkDeleteMessage(count, filters),
	}, nil
}

// hasAnyFilter checks if any filter field is non-nil.
func hasAnyFilter(input BulkDeleteInput) bool {
	return input.Workspace != nil ||
		input.Tag != nil ||
		input.NamePrefix != nil ||
		input.RunID != nil ||
		input.Phase != nil ||
		input.Role != nil
}

// hasAnyEffectiveFilter checks if any filter field is non-nil after normalization.
func hasAnyEffectiveFilter(filters db.InventoryFilters) bool {
	return filters.Workspace != nil ||
		filters.Tag != nil ||
		filters.NamePrefix != nil ||
		filters.RunID != nil ||
		filters.Phase != nil ||
		filters.Role != nil
}

// formatBulkDeleteMessage creates a human-readable message for the bulk delete result.
func formatBulkDeleteMessage(count int, filters db.InventoryFilters) string {
	if count == 0 {
		return "No active capsules matched the filters"
	}

	capsuleWord := "capsule"
	if count > 1 {
		capsuleWord = "capsules"
	}

	msg := fmt.Sprintf("Soft-deleted %d %s", count, capsuleWord)

	var parts []string
	if filters.Workspace != nil {
		parts = append(parts, fmt.Sprintf("workspace=%q", *filters.Workspace))
	}
	if filters.Tag != nil {
		parts = append(parts, fmt.Sprintf("tag=%q", *filters.Tag))
	}
	if filters.NamePrefix != nil {
		parts = append(parts, fmt.Sprintf("name_prefix=%q", *filters.NamePrefix))
	}
	if filters.RunID != nil {
		parts = append(parts, fmt.Sprintf("run_id=%q", *filters.RunID))
	}
	if filters.Phase != nil {
		parts = append(parts, fmt.Sprintf("phase=%q", *filters.Phase))
	}
	if filters.Role != nil {
		parts = append(parts, fmt.Sprintf("role=%q", *filters.Role))
	}

	if len(parts) > 0 {
		msg += " matching " + strings.Join(parts, ", ")
	}

	return msg
}
