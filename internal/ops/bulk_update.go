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

// BulkUpdateInput contains parameters for the BulkUpdate operation.
type BulkUpdateInput struct {
	// Filters
	Workspace  *string
	Tag        *string
	NamePrefix *string
	RunID      *string
	Phase      *string
	Role       *string
	// Updates (set_ prefix to distinguish from filters)
	SetPhase *string
	SetRole  *string
	SetTags  *[]string
}

// BulkUpdateOutput contains the result of the BulkUpdate operation.
type BulkUpdateOutput struct {
	Updated int    `json:"updated"`
	Message string `json:"message"`
}

// BulkUpdate updates metadata on all active capsules matching the given filters.
// At least one filter and at least one update field must be provided (safety guard).
func BulkUpdate(ctx context.Context, database *sql.DB, input BulkUpdateInput) (*BulkUpdateOutput, error) {
	// Phase 1: at least one filter must be non-nil
	if !hasAnyBulkUpdateFilter(input) {
		return nil, errors.NewInvalidRequest("at least one filter is required")
	}

	// Phase 2: at least one update field must be non-nil
	if !hasAnyUpdateField(input) {
		return nil, errors.NewInvalidRequest("at least one update field is required")
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

	// Phase 3: at least one filter must be non-empty after normalization
	if !hasAnyEffectiveFilter(filters) {
		return nil, errors.NewInvalidRequest("at least one filter must be non-empty after normalization")
	}

	// Build update fields - pass raw values (empty string means "clear field")
	var fields db.BulkUpdateFields
	if input.SetPhase != nil {
		v := strings.TrimSpace(*input.SetPhase)
		fields.Phase = &v
	}
	if input.SetRole != nil {
		v := strings.TrimSpace(*input.SetRole)
		fields.Role = &v
	}
	if input.SetTags != nil {
		fields.Tags = input.SetTags
	}

	count, err := db.BulkUpdate(ctx, database, filters, fields)
	if err != nil {
		return nil, err
	}

	return &BulkUpdateOutput{
		Updated: count,
		Message: formatBulkUpdateMessage(count, filters, fields),
	}, nil
}

// hasAnyBulkUpdateFilter checks if any filter field is non-nil.
func hasAnyBulkUpdateFilter(input BulkUpdateInput) bool {
	return input.Workspace != nil ||
		input.Tag != nil ||
		input.NamePrefix != nil ||
		input.RunID != nil ||
		input.Phase != nil ||
		input.Role != nil
}

// hasAnyUpdateField checks if any update field is non-nil.
func hasAnyUpdateField(input BulkUpdateInput) bool {
	return input.SetPhase != nil ||
		input.SetRole != nil ||
		input.SetTags != nil
}

// formatBulkUpdateMessage creates a human-readable message for the bulk update result.
func formatBulkUpdateMessage(count int, filters db.InventoryFilters, fields db.BulkUpdateFields) string {
	if count == 0 {
		return "No active capsules matched the filters"
	}

	capsuleWord := "capsule"
	if count > 1 {
		capsuleWord = "capsules"
	}

	msg := fmt.Sprintf("Updated %d %s", count, capsuleWord)

	// Format filter description
	var filterParts []string
	if filters.Workspace != nil {
		filterParts = append(filterParts, fmt.Sprintf("workspace=%q", *filters.Workspace))
	}
	if filters.Tag != nil {
		filterParts = append(filterParts, fmt.Sprintf("tag=%q", *filters.Tag))
	}
	if filters.NamePrefix != nil {
		filterParts = append(filterParts, fmt.Sprintf("name_prefix=%q", *filters.NamePrefix))
	}
	if filters.RunID != nil {
		filterParts = append(filterParts, fmt.Sprintf("run_id=%q", *filters.RunID))
	}
	if filters.Phase != nil {
		filterParts = append(filterParts, fmt.Sprintf("phase=%q", *filters.Phase))
	}
	if filters.Role != nil {
		filterParts = append(filterParts, fmt.Sprintf("role=%q", *filters.Role))
	}

	if len(filterParts) > 0 {
		msg += " matching " + strings.Join(filterParts, ", ")
	}

	// Format update description
	var updateParts []string
	if fields.Phase != nil {
		if *fields.Phase == "" {
			updateParts = append(updateParts, "phase=null")
		} else {
			updateParts = append(updateParts, fmt.Sprintf("phase=%q", *fields.Phase))
		}
	}
	if fields.Role != nil {
		if *fields.Role == "" {
			updateParts = append(updateParts, "role=null")
		} else {
			updateParts = append(updateParts, fmt.Sprintf("role=%q", *fields.Role))
		}
	}
	if fields.Tags != nil {
		if len(*fields.Tags) == 0 {
			updateParts = append(updateParts, "tags=null")
		} else {
			updateParts = append(updateParts, fmt.Sprintf("tags=%v", *fields.Tags))
		}
	}

	if len(updateParts) > 0 {
		msg += "; set " + strings.Join(updateParts, ", ")
	}

	return msg
}
