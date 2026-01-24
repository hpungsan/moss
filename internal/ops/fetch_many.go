package ops

import (
	"database/sql"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

// FetchManyInput contains parameters for the FetchMany operation.
type FetchManyInput struct {
	Items       []FetchManyRef
	IncludeText *bool // default: true
}

// FetchManyRef identifies a capsule by ID or by workspace+name.
type FetchManyRef struct {
	ID        string `json:"id,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// FetchManyOutput contains the result of the FetchMany operation.
type FetchManyOutput struct {
	Items  []FetchManyItem  `json:"items"`
	Errors []FetchManyError `json:"errors"`
}

// FetchManyItem contains a fetched capsule with metadata.
type FetchManyItem struct {
	ID             string   `json:"id"`
	Workspace      string   `json:"workspace"`
	WorkspaceNorm  string   `json:"workspace_norm"`
	Name           *string  `json:"name,omitempty"`
	NameNorm       *string  `json:"name_norm,omitempty"`
	Title          *string  `json:"title,omitempty"`
	CapsuleText    string   `json:"capsule_text,omitempty"` // omitempty - only when include_text=true
	CapsuleChars   int      `json:"capsule_chars"`
	TokensEstimate int      `json:"tokens_estimate"`
	Tags           []string `json:"tags,omitempty"`
	Source         *string  `json:"source,omitempty"`
	CreatedAt      int64    `json:"created_at"`
	UpdatedAt      int64    `json:"updated_at"`
	DeletedAt      *int64   `json:"deleted_at,omitempty"`
	TaskLink       TaskLink `json:"task_link"`
}

// FetchManyError represents an error for a specific ref.
type FetchManyError struct {
	Ref     FetchManyRef `json:"ref"`
	Code    string       `json:"code"`
	Message string       `json:"message"`
}

// FetchMany retrieves multiple capsules by ID or name.
// Returns partial success with items and errors arrays.
func FetchMany(database *sql.DB, input FetchManyInput) (*FetchManyOutput, error) {
	// Determine include_text (default: true)
	includeText := true
	if input.IncludeText != nil {
		includeText = *input.IncludeText
	}

	var items []FetchManyItem
	var errs []FetchManyError

	for _, ref := range input.Items {
		// Validate addressing for this ref
		addr, err := ValidateAddress(ref.ID, ref.Workspace, ref.Name)
		if err != nil {
			errs = append(errs, refToError(ref, err))
			continue
		}

		// Fetch capsule
		var c *capsule.Capsule
		if addr.ByID {
			c, err = db.GetByID(database, addr.ID, false)
		} else {
			c, err = db.GetByName(database, addr.Workspace, addr.Name, false)
		}
		if err != nil {
			errs = append(errs, refToError(ref, err))
			continue
		}

		// Build item
		item := capsuleToItem(c, includeText)
		items = append(items, item)
	}

	// Ensure we return empty arrays rather than nil
	if items == nil {
		items = []FetchManyItem{}
	}
	if errs == nil {
		errs = []FetchManyError{}
	}

	return &FetchManyOutput{
		Items:  items,
		Errors: errs,
	}, nil
}

// refToError converts a fetch error to a FetchManyError.
func refToError(ref FetchManyRef, err error) FetchManyError {
	var code, message string

	// Extract code and message from MossError
	if mossErr, ok := err.(*errors.MossError); ok {
		code = string(mossErr.Code)
		message = mossErr.Message
	} else {
		code = "INTERNAL_ERROR"
		message = err.Error()
	}

	return FetchManyError{
		Ref:     ref,
		Code:    code,
		Message: message,
	}
}

// capsuleToItem converts a Capsule to FetchManyItem.
func capsuleToItem(c *capsule.Capsule, includeText bool) FetchManyItem {
	text := ""
	if includeText {
		text = c.CapsuleText
	}

	name := ""
	if c.NameRaw != nil {
		name = *c.NameRaw
	}

	return FetchManyItem{
		ID:             c.ID,
		Workspace:      c.WorkspaceRaw,
		WorkspaceNorm:  c.WorkspaceNorm,
		Name:           c.NameRaw,
		NameNorm:       c.NameNorm,
		Title:          c.Title,
		CapsuleText:    text,
		CapsuleChars:   c.CapsuleChars,
		TokensEstimate: c.TokensEstimate,
		Tags:           c.Tags,
		Source:         c.Source,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
		DeletedAt:      c.DeletedAt,
		TaskLink:       BuildTaskLink(c.WorkspaceRaw, name, c.ID),
	}
}
