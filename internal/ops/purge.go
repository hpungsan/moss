package ops

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hpungsan/moss/internal/db"
)

// PurgeInput contains parameters for the Purge operation.
type PurgeInput struct {
	Workspace     *string // optional filter by workspace
	OlderThanDays *int    // optional, only purge if deleted_at < (now - N days)
}

// PurgeOutput contains the result of the Purge operation.
type PurgeOutput struct {
	Purged  int    `json:"purged"`
	Message string `json:"message"`
}

// Purge permanently deletes soft-deleted capsules.
func Purge(ctx context.Context, database *sql.DB, input PurgeInput) (*PurgeOutput, error) {
	count, err := db.PurgeDeleted(database, input.Workspace, input.OlderThanDays)
	if err != nil {
		return nil, err
	}

	message := formatPurgeMessage(count, input.Workspace, input.OlderThanDays)

	return &PurgeOutput{
		Purged:  count,
		Message: message,
	}, nil
}

// formatPurgeMessage creates a human-readable message for the purge result.
func formatPurgeMessage(count int, workspace *string, olderThanDays *int) string {
	if count == 0 {
		return "No deleted capsules to purge"
	}

	capsuleWord := "capsule"
	if count > 1 {
		capsuleWord = "capsules"
	}

	msg := fmt.Sprintf("Permanently deleted %d %s", count, capsuleWord)

	if workspace != nil {
		msg += fmt.Sprintf(" from workspace %q", *workspace)
	}

	if olderThanDays != nil {
		msg += fmt.Sprintf(" (deleted more than %d days ago)", *olderThanDays)
	}

	return msg
}
