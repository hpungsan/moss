package ops

import (
	"context"
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
	Items    []ComposeRef    // required, 1-50 items
	Format   string          // "markdown" (default) or "json"
	Sections []string        // only include these sections (exact match, case-insensitive)
	StoreAs  *ComposeStoreAs // optional: persist result
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
	ID          string `json:"id"`
	Workspace   string `json:"workspace"`
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"display_name"` // computed: title > name > id
	Text        string `json:"text"`
	Chars       int    `json:"chars"`
}

// ComposeBundle is the JSON format output structure.
type ComposeBundle struct {
	Parts []ComposePart `json:"parts"`
}

// Compose assembles multiple capsules into a single bundle.
// All-or-nothing: fails if any capsule is missing.
func Compose(ctx context.Context, database *sql.DB, cfg *config.Config, input ComposeInput) (*ComposeOutput, error) {
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

	// Validate sections
	if len(input.Sections) > 0 {
		for i, s := range input.Sections {
			if strings.TrimSpace(s) == "" {
				return nil, errors.NewInvalidRequest(
					fmt.Sprintf("sections[%d]: section name must not be empty", i))
			}
		}
	}

	// Reject JSON format with store_as (JSON output lacks section headers, so lint would fail)
	if format == "json" && input.StoreAs != nil {
		return nil, errors.NewInvalidRequest("cannot use format:\"json\" with store_as; JSON output is not a valid capsule structure")
	}

	// Open a read-only transaction so all reads share a single point-in-time snapshot.
	tx, err := database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		if ctx.Err() != nil {
			return nil, errors.NewCancelled("compose")
		}
		return nil, errors.NewInternal(err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Fetch all capsules (all-or-nothing)
	parts := make([]ComposePart, 0, len(input.Items))
	estimatedChars := 0
	for i, ref := range input.Items {
		select {
		case <-ctx.Done():
			return nil, errors.NewCancelled("compose")
		default:
		}

		// Validate addressing for this ref
		addr, err := ValidateAddress(ref.ID, ref.Workspace, ref.Name)
		if err != nil {
			return nil, fmt.Errorf("items[%d]: %w", i, err)
		}

		// Fetch capsule
		var c *capsule.Capsule
		if addr.ByID {
			c, err = db.GetByID(ctx, tx, addr.ID, false)
		} else {
			c, err = db.GetByName(ctx, tx, addr.Workspace, addr.Name, false)
		}
		if err != nil {
			return nil, fmt.Errorf("items[%d]: %w", i, err)
		}

		partText := c.CapsuleText
		partChars := c.CapsuleChars
		if len(input.Sections) > 0 {
			partText = filterSections(partText, input.Sections)
			partChars = capsule.CountChars(partText)
		}

		// Early size check (conservative estimate without formatting overhead).
		// When sections filtering is enabled, estimate based on filtered text to avoid false positives.
		estimatedChars += partChars
		if estimatedChars > cfg.CapsuleMaxChars {
			return nil, errors.NewComposeTooLarge(cfg.CapsuleMaxChars, estimatedChars)
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

		// Skip empty parts when section filtering produces no content
		if len(input.Sections) > 0 && partText == "" {
			continue
		}

		parts = append(parts, ComposePart{
			ID:          c.ID,
			Workspace:   c.WorkspaceRaw,
			Name:        name,
			DisplayName: displayName,
			Text:        partText,
			Chars:       partChars,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.NewInternal(err)
	}

	// Assemble bundle based on format
	var bundleText string
	if format == "markdown" {
		bundleText = assembleMarkdown(parts)
	} else {
		var err error
		bundleText, err = assembleJSON(parts)
		if err != nil {
			return nil, err
		}
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
		if bundleText == "" {
			return nil, errors.NewInvalidRequest("cannot store empty bundle (sections filter matched no content)")
		}

		storeResult, err := Store(ctx, database, cfg, StoreInput{
			Workspace:   input.StoreAs.Workspace,
			Name:        &input.StoreAs.Name,
			CapsuleText: bundleText,
			Mode:        input.StoreAs.Mode,
			AllowThin:   len(input.Sections) > 0,
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
		sb.WriteString(part.DisplayName)
		sb.WriteString("\n\n")
		sb.WriteString(part.Text)
	}
	return sb.String()
}

// assembleJSON creates JSON format: {"parts": [...]}
func assembleJSON(parts []ComposePart) (string, error) {
	bundle := ComposeBundle{Parts: parts}
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return "", errors.NewInternal(err)
	}
	return string(data), nil
}

// filterSections extracts only the requested sections from capsule text.
// Sections are matched by exact name (case-insensitive), in the order specified
// by the caller. Placeholder sections are skipped. If no sections are found
// (e.g., thin capsule without markdown headers), the original text is returned.
func filterSections(text string, sections []string) string {
	parsed := capsule.ParseSections(text)
	if len(parsed) == 0 {
		return text // thin capsule, no markdown headers — pass through unchanged
	}

	var sb strings.Builder
	found := false
	for _, name := range sections {
		sec := capsule.FindSectionExact(parsed, name)
		if sec == nil || sec.IsPlaceholder {
			continue
		}
		if found {
			sb.WriteString("\n")
		}
		sb.WriteString(text[sec.HeaderStart:sec.ContentEnd])
		found = true
	}

	return sb.String()
}
