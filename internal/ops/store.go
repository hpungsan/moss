package ops

import (
	"context"
	"crypto/rand"
	"database/sql"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

// StoreMode controls collision behavior.
type StoreMode string

const (
	StoreModeError   StoreMode = "error"   // default: fail on name collision
	StoreModeReplace StoreMode = "replace" // overwrite existing
)

// StoreInput contains parameters for the Store operation.
type StoreInput struct {
	Workspace   string  // default: "default"
	Name        *string // optional
	Title       *string // default: same as name, or nil
	CapsuleText string  // required
	Tags        []string
	Source      *string
	RunID       *string   // orchestration run ID
	Phase       *string   // workflow phase
	Role        *string   // agent role
	Mode        StoreMode // default: StoreModeError
	AllowThin   bool
}

// StoreOutput contains the result of the Store operation.
type StoreOutput struct {
	ID       string   `json:"id"`
	FetchKey FetchKey `json:"fetch_key"`
}

// Store creates or replaces a capsule.
func Store(ctx context.Context, database *sql.DB, cfg *config.Config, input StoreInput) (*StoreOutput, error) {
	// Validate required fields
	if input.CapsuleText == "" {
		return nil, errors.NewInvalidRequest("capsule_text is required")
	}

	// Apply defaults
	if strings.TrimSpace(input.Workspace) == "" {
		input.Workspace = "default"
	}
	input.RunID = cleanOptionalString(input.RunID)
	input.Phase = cleanOptionalString(input.Phase)
	input.Role = cleanOptionalString(input.Role)
	if input.Mode == "" {
		input.Mode = StoreModeError
	}
	if input.Mode != StoreModeError && input.Mode != StoreModeReplace {
		return nil, errors.NewInvalidRequest("mode must be one of: error, replace")
	}

	// Normalize workspace
	workspaceNorm := capsule.Normalize(input.Workspace)
	if workspaceNorm == "" {
		return nil, errors.NewInvalidRequest("workspace must not be empty")
	}

	// Normalize name if provided
	var nameRaw, nameNorm *string
	if input.Name != nil {
		normalized := capsule.Normalize(*input.Name)
		if normalized == "" {
			return nil, errors.NewInvalidRequest("name must not be empty (omit it for unnamed capsules)")
		}
		nameRaw = input.Name
		nameNorm = &normalized
	}

	// Default title to name if not provided
	title := input.Title
	if title == nil && nameRaw != nil {
		title = nameRaw
	}

	// Lint content
	lintResult := capsule.Lint(capsule.LintInput{
		CapsuleText: input.CapsuleText,
		MaxChars:    cfg.CapsuleMaxChars,
		AllowThin:   input.AllowThin,
	})

	if lintResult.TooLarge {
		return nil, errors.NewCapsuleTooLarge(lintResult.MaxChars, lintResult.ActualChars)
	}

	if len(lintResult.MissingSections) > 0 {
		return nil, errors.NewCapsuleTooThin(lintResult.MissingSections)
	}

	// Compute metrics
	capsuleChars := capsule.CountChars(input.CapsuleText)
	tokensEstimate := capsule.EstimateTokens(input.CapsuleText)
	now := time.Now().Unix()

	// Generate ULID for new capsule (may be discarded if upsert updates existing)
	id, err := generateULID()
	if err != nil {
		return nil, errors.NewInternal(err)
	}

	c := &capsule.Capsule{
		ID:             id,
		WorkspaceRaw:   input.Workspace,
		WorkspaceNorm:  workspaceNorm,
		NameRaw:        nameRaw,
		NameNorm:       nameNorm,
		Title:          title,
		CapsuleText:    input.CapsuleText,
		CapsuleChars:   capsuleChars,
		TokensEstimate: tokensEstimate,
		Tags:           input.Tags,
		Source:         input.Source,
		RunID:          input.RunID,
		Phase:          input.Phase,
		Role:           input.Role,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Build name for fetch key
	name := ""
	if nameRaw != nil {
		name = *nameRaw
	}

	if input.Mode == StoreModeReplace {
		// Use atomic UPSERT to avoid race conditions between concurrent callers.
		// If a capsule with the same (workspace, name) exists, it updates that capsule.
		// Otherwise, it inserts a new capsule.
		result, err := db.Upsert(database, c)
		if err != nil {
			return nil, err
		}

		return &StoreOutput{
			ID:       result.ID,
			FetchKey: BuildFetchKey(input.Workspace, name, result.ID),
		}, nil
	}

	// mode:error - Insert and fail on conflict
	if err := db.Insert(database, c); err != nil {
		return nil, err
	}

	return &StoreOutput{
		ID:       id,
		FetchKey: BuildFetchKey(input.Workspace, name, id),
	}, nil
}

// generateULID generates a new ULID.
func generateULID() (string, error) {
	entropy := ulid.Monotonic(rand.Reader, 0)
	id, err := ulid.New(ulid.Timestamp(time.Now()), entropy)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
