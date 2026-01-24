package ops

import (
	"database/sql"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/db"
)

// FetchInput contains parameters for the Fetch operation.
type FetchInput struct {
	ID             string
	Workspace      string
	Name           string
	IncludeDeleted bool
	IncludeText    *bool // default: true (nil means default)
}

// FetchOutput contains the result of the Fetch operation.
type FetchOutput struct {
	capsule.Capsule          // embedded (copy, not pointer)
	TaskLink        TaskLink `json:"task_link"`
}

// Fetch retrieves a capsule by ID or name.
func Fetch(database *sql.DB, input FetchInput) (*FetchOutput, error) {
	// Validate address
	addr, err := ValidateAddress(input.ID, input.Workspace, input.Name)
	if err != nil {
		return nil, err
	}

	// Fetch capsule
	var c *capsule.Capsule
	if addr.ByID {
		c, err = db.GetByID(database, addr.ID, input.IncludeDeleted)
	} else {
		c, err = db.GetByName(database, addr.Workspace, addr.Name, input.IncludeDeleted)
	}
	if err != nil {
		return nil, err
	}

	// Create output with a copy of the capsule
	output := &FetchOutput{
		Capsule: *c, // copy, not pointer
	}

	includeText := true
	if input.IncludeText != nil {
		includeText = *input.IncludeText
	}
	if !includeText {
		output.CapsuleText = ""
	}

	// Build task link
	name := ""
	if c.NameRaw != nil {
		name = *c.NameRaw
	}
	output.TaskLink = BuildTaskLink(c.WorkspaceRaw, name, c.ID)

	return output, nil
}
