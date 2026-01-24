package ops

import (
	"database/sql"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/db"
)

// LatestInput contains parameters for the Latest operation.
type LatestInput struct {
	Workspace      string // required, defaults to "default"
	IncludeText    *bool  // default: false (summary only)
	IncludeDeleted bool
}

// LatestOutput contains the result of the Latest operation.
type LatestOutput struct {
	Item *LatestItem `json:"item"` // nil if workspace is empty
}

// LatestItem contains the latest capsule with optional text.
type LatestItem struct {
	capsule.CapsuleSummary          // embedded summary
	CapsuleText            string   `json:"capsule_text,omitempty"` // only if include_text
	TaskLink               TaskLink `json:"task_link"`
}

// Latest retrieves the most recent capsule in a workspace.
func Latest(database *sql.DB, input LatestInput) (*LatestOutput, error) {
	// Normalize workspace
	workspace := capsule.Normalize(input.Workspace)
	if workspace == "" {
		workspace = "default"
	}

	// Determine include_text (default: false)
	includeText := false
	if input.IncludeText != nil {
		includeText = *input.IncludeText
	}

	// Query database based on include_text
	if includeText {
		// Fetch full capsule with text
		c, err := db.GetLatestFull(database, workspace, input.IncludeDeleted)
		if err != nil {
			return nil, err
		}
		if c == nil {
			return &LatestOutput{Item: nil}, nil
		}

		// Build task link
		name := ""
		if c.NameRaw != nil {
			name = *c.NameRaw
		}

		return &LatestOutput{
			Item: &LatestItem{
				CapsuleSummary: c.ToSummary(),
				CapsuleText:    c.CapsuleText,
				TaskLink:       BuildTaskLink(c.WorkspaceRaw, name, c.ID),
			},
		}, nil
	}

	// Fetch summary only (no text)
	s, err := db.GetLatestSummary(database, workspace, input.IncludeDeleted)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return &LatestOutput{Item: nil}, nil
	}

	// Build task link
	name := ""
	if s.Name != nil {
		name = *s.Name
	}

	return &LatestOutput{
		Item: &LatestItem{
			CapsuleSummary: *s,
			CapsuleText:    "", // omitted via omitempty
			TaskLink:       BuildTaskLink(s.Workspace, name, s.ID),
		},
	}, nil
}
