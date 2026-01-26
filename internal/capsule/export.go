package capsule

// ExportRecord represents a capsule record in JSONL export format.
// It is used for parsing export files during import.
type ExportRecord struct {
	// Header detection field - true only for header line
	MossExport bool `json:"_moss_export,omitempty"`

	// Header fields (only present in header line)
	SchemaVersion string `json:"schema_version,omitempty"`
	ExportedAt    int64  `json:"exported_at,omitempty"`

	// Capsule fields
	ID             string   `json:"id"`
	WorkspaceRaw   string   `json:"workspace_raw"`
	WorkspaceNorm  string   `json:"workspace_norm"` // IGNORED on import, recomputed
	NameRaw        *string  `json:"name_raw"`
	NameNorm       *string  `json:"name_norm"` // IGNORED on import, recomputed
	Title          *string  `json:"title"`
	CapsuleText    string   `json:"capsule_text"`
	CapsuleChars   int      `json:"capsule_chars"`   // IGNORED on import, recomputed
	TokensEstimate int      `json:"tokens_estimate"` // IGNORED on import, recomputed
	Tags           []string `json:"tags"`
	Source         *string  `json:"source"`
	RunID          *string  `json:"run_id"`
	Phase          *string  `json:"phase"`
	Role           *string  `json:"role"`
	CreatedAt      int64    `json:"created_at"`
	UpdatedAt      int64    `json:"updated_at"`
	DeletedAt      *int64   `json:"deleted_at"`
}

// ToCapsule converts an ExportRecord to a Capsule, recomputing derived fields.
func (r *ExportRecord) ToCapsule() *Capsule {
	c := &Capsule{
		ID:             r.ID,
		WorkspaceRaw:   r.WorkspaceRaw,
		WorkspaceNorm:  Normalize(r.WorkspaceRaw), // Recompute
		NameRaw:        r.NameRaw,
		Title:          r.Title,
		CapsuleText:    r.CapsuleText,
		CapsuleChars:   CountChars(r.CapsuleText),     // Recompute
		TokensEstimate: EstimateTokens(r.CapsuleText), // Recompute
		Tags:           r.Tags,
		Source:         r.Source,
		RunID:          r.RunID,
		Phase:          r.Phase,
		Role:           r.Role,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
		DeletedAt:      r.DeletedAt,
	}

	// Recompute name_norm from name_raw
	if r.NameRaw != nil {
		norm := Normalize(*r.NameRaw)
		c.NameNorm = &norm
	}

	return c
}

// CapsuleToExportRecord converts a Capsule to an ExportRecord for export.
func CapsuleToExportRecord(c *Capsule) *ExportRecord {
	return &ExportRecord{
		ID:             c.ID,
		WorkspaceRaw:   c.WorkspaceRaw,
		WorkspaceNorm:  c.WorkspaceNorm,
		NameRaw:        c.NameRaw,
		NameNorm:       c.NameNorm,
		Title:          c.Title,
		CapsuleText:    c.CapsuleText,
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
	}
}
