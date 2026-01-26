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
	ID             string   `json:"id"`
	Workspace      string   `json:"workspace"`
	WorkspaceNorm  string   `json:"workspace_norm"`
	Name           *string  `json:"name,omitempty"`
	NameNorm       *string  `json:"name_norm,omitempty"`
	Title          *string  `json:"title,omitempty"`
	CapsuleText    string   `json:"capsule_text,omitempty"`
	CapsuleChars   int      `json:"capsule_chars"`
	TokensEstimate int      `json:"tokens_estimate"`
	Tags           []string `json:"tags,omitempty"`
	Source         *string  `json:"source,omitempty"`
	RunID          *string  `json:"run_id,omitempty"`
	Phase          *string  `json:"phase,omitempty"`
	Role           *string  `json:"role,omitempty"`
	CreatedAt      int64    `json:"created_at"`
	UpdatedAt      int64    `json:"updated_at"`
	DeletedAt      *int64   `json:"deleted_at,omitempty"`
	FetchKey       FetchKey `json:"fetch_key"`
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

	// Determine include_text (default: true)
	includeText := true
	if input.IncludeText != nil {
		includeText = *input.IncludeText
	}

	// Build task link
	name := ""
	if c.NameRaw != nil {
		name = *c.NameRaw
	}

	// Build output with explicit field mapping
	output := &FetchOutput{
		ID:             c.ID,
		Workspace:      c.WorkspaceRaw,
		WorkspaceNorm:  c.WorkspaceNorm,
		Name:           c.NameRaw,
		NameNorm:       c.NameNorm,
		Title:          c.Title,
		CapsuleChars:   c.CapsuleChars,
		TokensEstimate: c.TokensEstimate,
		Tags:           c.Tags,
		Source:         c.Source,
		RunID:          c.RunID,
		Phase:          c.Phase,
		Role:           c.Role,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
		DeletedAt:      c.DeletedAt,
		FetchKey:       BuildFetchKey(c.WorkspaceRaw, name, c.ID),
	}

	// Only include text if requested (omitempty handles the rest)
	if includeText {
		output.CapsuleText = c.CapsuleText
	}

	return output, nil
}
