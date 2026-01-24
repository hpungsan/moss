package ops

import (
	"database/sql"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

// UpdateInput contains parameters for the Update operation.
type UpdateInput struct {
	// Addressing
	ID        string
	Workspace string
	Name      string

	// Editable fields (nil = don't change)
	CapsuleText *string
	Title       *string
	Tags        *[]string
	Source      *string

	AllowThin bool
}

// UpdateOutput contains the result of the Update operation.
type UpdateOutput struct {
	ID       string   `json:"id"`
	TaskLink TaskLink `json:"task_link"`
}

// Update modifies an existing capsule.
func Update(database *sql.DB, cfg *config.Config, input UpdateInput) (*UpdateOutput, error) {
	// Validate address
	addr, err := ValidateAddress(input.ID, input.Workspace, input.Name)
	if err != nil {
		return nil, err
	}

	// Validate at least one editable field is provided
	if input.CapsuleText == nil && input.Title == nil && input.Tags == nil && input.Source == nil {
		return nil, errors.NewInvalidRequest("at least one editable field must be provided")
	}

	// Fetch existing capsule (active only)
	var c *capsule.Capsule
	if addr.ByID {
		c, err = db.GetByID(database, addr.ID, false)
	} else {
		c, err = db.GetByName(database, addr.Workspace, addr.Name, false)
	}
	if err != nil {
		return nil, err
	}

	// Apply updates
	if input.CapsuleText != nil {
		// Lint new content
		lintResult := capsule.Lint(capsule.LintInput{
			CapsuleText: *input.CapsuleText,
			MaxChars:    cfg.CapsuleMaxChars,
			AllowThin:   input.AllowThin,
		})

		if lintResult.TooLarge {
			return nil, errors.NewCapsuleTooLarge(lintResult.MaxChars, lintResult.ActualChars)
		}

		if len(lintResult.MissingSections) > 0 {
			return nil, errors.NewCapsuleTooThin(lintResult.MissingSections)
		}

		c.CapsuleText = *input.CapsuleText
		c.CapsuleChars = capsule.CountChars(*input.CapsuleText)
		c.TokensEstimate = capsule.EstimateTokens(*input.CapsuleText)
	}

	if input.Title != nil {
		c.Title = input.Title
	}

	if input.Tags != nil {
		c.Tags = *input.Tags
	}

	if input.Source != nil {
		c.Source = input.Source
	}

	// Persist update
	if err := db.UpdateByID(database, c); err != nil {
		return nil, err
	}

	// Build task link
	name := ""
	if c.NameRaw != nil {
		name = *c.NameRaw
	}

	return &UpdateOutput{
		ID:       c.ID,
		TaskLink: BuildTaskLink(c.WorkspaceRaw, name, c.ID),
	}, nil
}
