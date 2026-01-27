package ops

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

// ComposeInput contains parameters for the Compose operation.
type ComposeInput struct {
	Items   []ComposeRef    // required, 1-50 items
	Format  string          // "markdown" (default) or "json"
	StoreAs *ComposeStoreAs // optional: persist result
}

// ComposeRef identifies a capsule by ID or by workspace+name.
type ComposeRef struct {
	ID        string `json:"id,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// ComposeStoreAs specifies how to persist the composed bundle.
type ComposeStoreAs struct {
	Workspace string    // default: "default"
	Name      string    // required
	Mode      StoreMode // default: StoreModeError
}

// ComposeOutput contains the result of the Compose operation.
type ComposeOutput struct {
	BundleText  string       `json:"bundle_text"`
	BundleChars int          `json:"bundle_chars"`
	PartsCount  int          `json:"parts_count"`
	Stored      *StoreOutput `json:"stored,omitempty"` // only if store_as
}

// ComposePart represents a single capsule in the composed bundle.
type ComposePart struct {
	ID        string `json:"id"`
	Workspace string `json:"workspace"`
	Name      string `json:"name,omitempty"`
	Title     string `json:"title,omitempty"`
	Text      string `json:"text"`
	Chars     int    `json:"chars"`
}

// ComposeBundle is the JSON format output structure.
type ComposeBundle struct {
	Parts []ComposePart `json:"parts"`
}

// Compose assembles multiple capsules into a single bundle.
// All-or-nothing: fails if any capsule is missing.
func Compose(database *sql.DB, cfg *config.Config, input ComposeInput) (*ComposeOutput, error) {
	// Validate items count
	if len(input.Items) == 0 {
		return nil, errors.NewInvalidRequest("items is required and must not be empty")
	}
	if len(input.Items) > MaxFetchManyItems {
		return nil, errors.NewInvalidRequest(
			fmt.Sprintf("too many items: %d (max %d)", len(input.Items), MaxFetchManyItems))
	}

	// Validate format
	format := input.Format
	if format == "" {
		format = "markdown"
	}
	if format != "markdown" && format != "json" {
		return nil, errors.NewInvalidRequest("format must be one of: markdown, json")
	}

	// Reject JSON format with store_as (JSON output lacks section headers, so lint would fail)
	if format == "json" && input.StoreAs != nil {
		return nil, errors.NewInvalidRequest("cannot use format:\"json\" with store_as; JSON output is not a valid capsule structure")
	}

	// Fetch all capsules (all-or-nothing)
	parts := make([]ComposePart, 0, len(input.Items))
	for i, ref := range input.Items {
		// Validate addressing for this ref
		addr, err := ValidateAddress(ref.ID, ref.Workspace, ref.Name)
		if err != nil {
			return nil, fmt.Errorf("items[%d]: %w", i, err)
		}

		// Fetch capsule
		var c *capsule.Capsule
		if addr.ByID {
			c, err = db.GetByID(database, addr.ID, false)
		} else {
			c, err = db.GetByName(database, addr.Workspace, addr.Name, false)
		}
		if err != nil {
			// Return error with context about which item failed
			if mossErr, ok := err.(*errors.MossError); ok {
				return nil, mossErr
			}
			return nil, err
		}

		// Build part with display name priority: title > name > id
		displayName := c.ID
		if c.NameRaw != nil {
			displayName = *c.NameRaw
		}
		if c.Title != nil {
			displayName = *c.Title
		}

		name := ""
		if c.NameRaw != nil {
			name = *c.NameRaw
		}

		parts = append(parts, ComposePart{
			ID:        c.ID,
			Workspace: c.WorkspaceRaw,
			Name:      name,
			Title:     displayName, // Use display name (title > name > id)
			Text:      c.CapsuleText,
			Chars:     c.CapsuleChars,
		})
	}

	// Assemble bundle based on format
	var bundleText string
	if format == "markdown" {
		bundleText = assembleMarkdown(parts)
	} else {
		bundleText = assembleJSON(parts)
	}

	bundleChars := capsule.CountChars(bundleText)

	// Check size limit
	if bundleChars > cfg.CapsuleMaxChars {
		return nil, errors.NewComposeTooLarge(cfg.CapsuleMaxChars, bundleChars)
	}

	output := &ComposeOutput{
		BundleText:  bundleText,
		BundleChars: bundleChars,
		PartsCount:  len(parts),
	}

	// Optionally store the result
	if input.StoreAs != nil {
		if input.StoreAs.Name == "" {
			return nil, errors.NewInvalidRequest("store_as.name is required")
		}

		storeResult, err := Store(database, cfg, StoreInput{
			Workspace:   input.StoreAs.Workspace,
			Name:        &input.StoreAs.Name,
			CapsuleText: bundleText,
			Mode:        input.StoreAs.Mode,
			AllowThin:   false, // Lint the composed result
		})
		if err != nil {
			return nil, err
		}
		output.Stored = storeResult
	}

	return output, nil
}

// assembleMarkdown creates markdown format: ## heading\n\ntext\n\n---\n\n...
func assembleMarkdown(parts []ComposePart) string {
	var sb strings.Builder
	for i, part := range parts {
		if i > 0 {
			sb.WriteString("\n\n---\n\n")
		}
		sb.WriteString("## ")
		sb.WriteString(part.Title)
		sb.WriteString("\n\n")
		sb.WriteString(part.Text)
	}
	return sb.String()
}

// assembleJSON creates JSON format: {"parts": [...]}
func assembleJSON(parts []ComposePart) string {
	// Restore original title (not display name) for JSON output
	bundle := ComposeBundle{
		Parts: parts,
	}
	data, _ := json.MarshalIndent(bundle, "", "  ")
	return string(data)
}
