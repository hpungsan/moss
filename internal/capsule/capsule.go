package capsule

// Capsule represents a distilled context snapshot for AI session handoffs.
// Fields correspond to the schema in DESIGN.md §9.
type Capsule struct {
	// ID is a ULID that uniquely identifies this capsule
	ID string

	// WorkspaceRaw is the original workspace string as provided by the user
	WorkspaceRaw string

	// WorkspaceNorm is the normalized workspace (lowercased, trimmed, collapsed spaces)
	WorkspaceNorm string

	// NameRaw is the original name as provided by the user (nullable)
	NameRaw *string

	// NameNorm is the normalized name (nullable)
	NameNorm *string

	// Title is an optional human-readable title
	Title *string

	// CapsuleText is the main content of the capsule
	CapsuleText string

	// CapsuleChars is the character count (runes, not bytes)
	CapsuleChars int

	// TokensEstimate is the estimated token count for LLM context budgeting
	TokensEstimate int

	// Tags is a list of tags for categorization (stored as JSON in DB)
	Tags []string

	// Source indicates where the capsule originated (e.g., "claude-code", "manual")
	Source *string

	// CreatedAt is the Unix timestamp when the capsule was created
	CreatedAt int64

	// UpdatedAt is the Unix timestamp when the capsule was last updated
	UpdatedAt int64

	// DeletedAt is the Unix timestamp for soft delete (nullable)
	DeletedAt *int64
}
