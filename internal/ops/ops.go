package ops

import (
	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/errors"
)

// Address represents a validated capsule address.
type Address struct {
	ByID      bool
	ID        string
	Workspace string // normalized, defaulted to "default" for name-mode
	Name      string // normalized
}

// ValidateAddress validates addressing parameters and returns a normalized Address.
// Rules:
// - If id provided → ByID mode, ignore workspace/name
// - Else if name provided → ByName mode, default workspace to "default" if empty
// - If both id AND name provided → ErrAmbiguousAddressing
// - If neither id nor name provided → ErrInvalidRequest
func ValidateAddress(id, workspace, name string) (*Address, error) {
	hasID := id != ""
	hasName := name != ""

	if hasID && hasName {
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
	if workspace == "" {
		workspace = "default"
	}

	return &Address{
		ByID:      false,
		Workspace: capsule.Normalize(workspace),
		Name:      capsule.Normalize(name),
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
