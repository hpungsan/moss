package ops

import (
	"context"
	"database/sql"

	"github.com/hpungsan/moss/internal/db"
)

// DeleteInput contains parameters for the Delete operation.
type DeleteInput struct {
	ID        string
	Workspace string
	Name      string
}

// DeleteOutput contains the result of the Delete operation.
type DeleteOutput struct {
	Deleted bool   `json:"deleted"`
	ID      string `json:"id"`
}

// Delete soft-deletes a capsule.
func Delete(ctx context.Context, database *sql.DB, input DeleteInput) (*DeleteOutput, error) {
	// Validate address
	addr, err := ValidateAddress(input.ID, input.Workspace, input.Name)
	if err != nil {
		return nil, err
	}

	// Fetch existing (active only) to get the ID if addressed by name
	var capsuleID string
	if addr.ByID {
		capsuleID = addr.ID
		// Verify it exists (GetByID will return ErrNotFound if not)
		_, err = db.GetByID(ctx, database, addr.ID, false)
		if err != nil {
			return nil, err
		}
	} else {
		c, err := db.GetByName(ctx, database, addr.Workspace, addr.Name, false)
		if err != nil {
			return nil, err
		}
		capsuleID = c.ID
	}

	// Soft delete
	if err := db.SoftDelete(ctx, database, capsuleID); err != nil {
		return nil, err
	}

	return &DeleteOutput{
		Deleted: true,
		ID:      capsuleID,
	}, nil
}
