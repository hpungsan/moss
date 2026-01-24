package capsule

// CapsuleSummary represents a capsule's metadata without the full text content.
// Used for browse operations (list, inventory, latest) to reduce data transfer.
type CapsuleSummary struct {
	// ID is a ULID that uniquely identifies this capsule
	ID string `json:"id"`

	// Workspace is the original workspace string as provided by the user
	Workspace string `json:"workspace"`

	// WorkspaceNorm is the normalized workspace (lowercased, trimmed, collapsed spaces)
	WorkspaceNorm string `json:"workspace_norm"`

	// Name is the original name as provided by the user (nullable)
	Name *string `json:"name,omitempty"`

	// NameNorm is the normalized name (nullable)
	NameNorm *string `json:"name_norm,omitempty"`

	// Title is an optional human-readable title
	Title *string `json:"title,omitempty"`

	// CapsuleChars is the character count (runes, not bytes)
	CapsuleChars int `json:"capsule_chars"`

	// TokensEstimate is the estimated token count for LLM context budgeting
	TokensEstimate int `json:"tokens_estimate"`

	// Tags is a list of tags for categorization
	Tags []string `json:"tags,omitempty"`

	// Source indicates where the capsule originated (e.g., "claude-code", "manual")
	Source *string `json:"source,omitempty"`

	// CreatedAt is the Unix timestamp when the capsule was created
	CreatedAt int64 `json:"created_at"`

	// UpdatedAt is the Unix timestamp when the capsule was last updated
	UpdatedAt int64 `json:"updated_at"`

	// DeletedAt is the Unix timestamp for soft delete (nullable)
	DeletedAt *int64 `json:"deleted_at,omitempty"`
}

// ToSummary converts a Capsule to a CapsuleSummary by stripping the text content.
func (c *Capsule) ToSummary() CapsuleSummary {
	return CapsuleSummary{
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
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
		DeletedAt:      c.DeletedAt,
	}
}
