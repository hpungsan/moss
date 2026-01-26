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

// ParsedAddress represents a validated capsule address.
type ParsedAddress struct {
	ByID      bool
	ID        string
	Workspace string // normalized, defaulted to "default" for name-mode
	Name      string // normalized
}

// ValidateAddress validates addressing parameters and returns a normalized ParsedAddress.
// Rules:
// - Must specify exactly one addressing mode: id OR (workspace + name)
// - If id provided with name or workspace → ErrAmbiguousAddressing
// - If neither id nor name provided → ErrInvalidRequest
func ValidateAddress(id, workspace, name string) (*ParsedAddress, error) {
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
		return &ParsedAddress{
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

	return &ParsedAddress{
		ByID:      false,
		Workspace: workspaceNorm,
		Name:      nameNorm,
	}, nil
}

func cleanOptionalString(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

// FetchKey provides an address for fetching a capsule.
// Either (MossCapsule + MossWorkspace) or MossID is populated.
type FetchKey struct {
	MossCapsule   string `json:"moss_capsule,omitempty"`
	MossWorkspace string `json:"moss_workspace,omitempty"`
	MossID        string `json:"moss_id,omitempty"`
}

// BuildFetchKey creates a FetchKey for the given capsule identifiers.
// If name present: {moss_capsule: name, moss_workspace: workspace}
// If unnamed: {moss_id: id}
func BuildFetchKey(workspace, name, id string) FetchKey {
	if name != "" {
		return FetchKey{
			MossCapsule:   name,
			MossWorkspace: workspace,
		}
	}
	return FetchKey{
		MossID: id,
	}
}

// SummaryItem wraps a CapsuleSummary with a FetchKey for list/inventory responses.
type SummaryItem struct {
	capsule.CapsuleSummary
	FetchKey FetchKey `json:"fetch_key"`
}

// SummaryToItem converts a CapsuleSummary to a SummaryItem with fetch_key.
func SummaryToItem(s capsule.CapsuleSummary) SummaryItem {
	name := ""
	if s.Name != nil {
		name = *s.Name
	}
	return SummaryItem{
		CapsuleSummary: s,
		FetchKey:       BuildFetchKey(s.Workspace, name, s.ID),
	}
}

// SummariesToItems converts a slice of CapsuleSummary to SummaryItems.
func SummariesToItems(summaries []capsule.CapsuleSummary) []SummaryItem {
	items := make([]SummaryItem, len(summaries))
	for i, s := range summaries {
		items[i] = SummaryToItem(s)
	}
	return items
}
