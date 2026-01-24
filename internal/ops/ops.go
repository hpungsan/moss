package ops

import (
	"strings"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/errors"
)

// Pagination limits
const (
	DefaultListLimit      = 20
	MaxListLimit          = 100
	DefaultInventoryLimit = 100
	MaxInventoryLimit     = 500
	MaxFetchManyItems     = 50
)

// Pagination contains pagination metadata for list operations.
type Pagination struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
	Total   int  `json:"total"`
}

// Address represents a validated capsule address.
type Address struct {
	ByID      bool
	ID        string
	Workspace string // normalized, defaulted to "default" for name-mode
	Name      string // normalized
}

// ValidateAddress validates addressing parameters and returns a normalized Address.
// Rules:
// - Must specify exactly one addressing mode: id OR (workspace + name)
// - If id provided with name or workspace → ErrAmbiguousAddressing
// - If neither id nor name provided → ErrInvalidRequest
func ValidateAddress(id, workspace, name string) (*Address, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	workspace = strings.TrimSpace(workspace)

	hasID := id != ""
	hasName := name != ""
	hasWorkspace := workspace != ""

	// Strict: id must be alone, no other addressing fields
	if hasID && (hasName || hasWorkspace) {
		return nil, errors.NewAmbiguousAddressing()
	}

	if !hasID && !hasName {
		return nil, errors.NewInvalidRequest("must specify either id or name")
	}

	if hasID {
		return &Address{
			ByID: true,
			ID:   id,
		}, nil
	}

	// ByName mode
	workspaceNorm := capsule.Normalize(workspace)
	if workspaceNorm == "" {
		workspaceNorm = "default"
	}
	nameNorm := capsule.Normalize(name)
	if nameNorm == "" {
		return nil, errors.NewInvalidRequest("name must not be empty")
	}

	return &Address{
		ByID:      false,
		Workspace: workspaceNorm,
		Name:      nameNorm,
	}, nil
}

// TaskLink provides a reference for Claude Code Tasks integration.
// Either (MossCapsule + MossWorkspace) or MossID is populated.
type TaskLink struct {
	MossCapsule   string `json:"moss_capsule,omitempty"`
	MossWorkspace string `json:"moss_workspace,omitempty"`
	MossID        string `json:"moss_id,omitempty"`
}

// BuildTaskLink creates a TaskLink for the given capsule identifiers.
// If name present: {moss_capsule: name, moss_workspace: workspace}
// If unnamed: {moss_id: id}
func BuildTaskLink(workspace, name, id string) TaskLink {
	if name != "" {
		return TaskLink{
			MossCapsule:   name,
			MossWorkspace: workspace,
		}
	}
	return TaskLink{
		MossID: id,
	}
}
