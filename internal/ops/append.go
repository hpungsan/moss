package ops

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

// AppendInput contains parameters for the Append operation.
type AppendInput struct {
	// Addressing
	ID        string
	Workspace string
	Name      string

	// Required fields
	Section string // target section header
	Content string // text to append
}

// AppendOutput contains the result of the Append operation.
type AppendOutput struct {
	ID         string    `json:"id"`
	FetchKey   *FetchKey `json:"fetch_key,omitempty"`
	SectionHit string    `json:"section_hit"` // actual header matched
	Replaced   bool      `json:"replaced"`    // true if placeholder was replaced
}

// Append adds content to a specific section of a capsule.
// It finds the section by header (case-insensitive, synonym-aware) and either
// replaces placeholder content or appends after existing content.
func Append(ctx context.Context, database *sql.DB, cfg *config.Config, input AppendInput) (*AppendOutput, error) {
	// Validate address
	addr, err := ValidateAddress(input.ID, input.Workspace, input.Name)
	if err != nil {
		return nil, err
	}

	// Validate section
	if strings.TrimSpace(input.Section) == "" {
		return nil, errors.NewInvalidRequest("section is required")
	}

	// Validate content
	if strings.TrimSpace(input.Content) == "" {
		return nil, errors.NewInvalidRequest("content is required")
	}

	// Fetch existing capsule (active only)
	var c *capsule.Capsule
	if addr.ByID {
		c, err = db.GetByID(ctx, database, addr.ID, false)
	} else {
		c, err = db.GetByName(ctx, database, addr.Workspace, addr.Name, false)
	}
	if err != nil {
		return nil, err
	}

	// Parse sections
	sections := capsule.ParseSections(c.CapsuleText)
	if len(sections) == 0 {
		return nil, errors.NewInvalidRequest("capsule_append requires markdown format (no sections found)")
	}

	// Find target section (exact match, no synonym resolution)
	section := capsule.FindSectionExact(sections, input.Section)
	if section == nil {
		available := capsule.SectionNames(sections)
		return nil, errors.NewInvalidRequest(fmt.Sprintf("section %q not found; available: %v", input.Section, available))
	}

	// Insert content
	replaced := section.IsPlaceholder
	newText := capsule.InsertContent(c.CapsuleText, section, input.Content)

	// Check size limit
	newChars := capsule.CountChars(newText)
	if cfg.CapsuleMaxChars > 0 && newChars > cfg.CapsuleMaxChars {
		return nil, errors.NewCapsuleTooLarge(cfg.CapsuleMaxChars, newChars)
	}

	// Update capsule fields
	c.CapsuleText = newText
	c.CapsuleChars = newChars
	c.TokensEstimate = capsule.EstimateTokens(newText)

	// Persist update
	if err := db.UpdateByID(ctx, database, c); err != nil {
		return nil, err
	}

	// Build output
	output := &AppendOutput{
		ID:         c.ID,
		SectionHit: section.Header,
		Replaced:   replaced,
	}

	// Include fetch_key only for named capsules
	if c.NameRaw != nil {
		fk := BuildFetchKey(c.WorkspaceRaw, *c.NameRaw, c.ID)
		output.FetchKey = &fk
	}

	return output, nil
}
