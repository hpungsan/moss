package mcp

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/errors"
	"github.com/hpungsan/moss/internal/ops"
)

// Handlers holds dependencies for MCP tool handlers.
type Handlers struct {
	db  *sql.DB
	cfg *config.Config
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(db *sql.DB, cfg *config.Config) *Handlers {
	return &Handlers{db: db, cfg: cfg}
}

// Request types for each tool

// StoreRequest represents the arguments for store.
type StoreRequest struct {
	Workspace   string   `json:"workspace"`
	Name        *string  `json:"name,omitempty"`
	Title       *string  `json:"title,omitempty"`
	CapsuleText string   `json:"capsule_text"`
	Tags        []string `json:"tags,omitempty"`
	Source      *string  `json:"source,omitempty"`
	RunID       *string  `json:"run_id,omitempty"`
	Phase       *string  `json:"phase,omitempty"`
	Role        *string  `json:"role,omitempty"`
	Mode        string   `json:"mode,omitempty"`
	AllowThin   bool     `json:"allow_thin,omitempty"`
}

// FetchRequest represents the arguments for fetch.
type FetchRequest struct {
	ID             string `json:"id,omitempty"`
	Workspace      string `json:"workspace,omitempty"`
	Name           string `json:"name,omitempty"`
	IncludeDeleted bool   `json:"include_deleted,omitempty"`
	IncludeText    *bool  `json:"include_text,omitempty"`
}

// FetchManyRequest represents the arguments for fetch_many.
type FetchManyRequest struct {
	Items          []FetchManyRef `json:"items"`
	IncludeText    *bool          `json:"include_text,omitempty"`
	IncludeDeleted bool           `json:"include_deleted,omitempty"`
}

// FetchManyRef identifies a capsule in fetch_many.
type FetchManyRef struct {
	ID        string `json:"id,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// UpdateRequest represents the arguments for update.
type UpdateRequest struct {
	ID          string    `json:"id,omitempty"`
	Workspace   string    `json:"workspace,omitempty"`
	Name        string    `json:"name,omitempty"`
	CapsuleText *string   `json:"capsule_text,omitempty"`
	Title       *string   `json:"title,omitempty"`
	Tags        *[]string `json:"tags,omitempty"`
	Source      *string   `json:"source,omitempty"`
	RunID       *string   `json:"run_id,omitempty"`
	Phase       *string   `json:"phase,omitempty"`
	Role        *string   `json:"role,omitempty"`
	AllowThin   bool      `json:"allow_thin,omitempty"`
}

// DeleteRequest represents the arguments for delete.
type DeleteRequest struct {
	ID        string `json:"id,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// LatestRequest represents the arguments for latest.
type LatestRequest struct {
	Workspace      string  `json:"workspace,omitempty"`
	RunID          *string `json:"run_id,omitempty"`
	Phase          *string `json:"phase,omitempty"`
	Role           *string `json:"role,omitempty"`
	IncludeText    *bool   `json:"include_text,omitempty"`
	IncludeDeleted bool    `json:"include_deleted,omitempty"`
}

// ListRequest represents the arguments for list.
type ListRequest struct {
	Workspace      string  `json:"workspace,omitempty"`
	RunID          *string `json:"run_id,omitempty"`
	Phase          *string `json:"phase,omitempty"`
	Role           *string `json:"role,omitempty"`
	Limit          int     `json:"limit,omitempty"`
	Offset         int     `json:"offset,omitempty"`
	IncludeDeleted bool    `json:"include_deleted,omitempty"`
}

// InventoryRequest represents the arguments for inventory.
type InventoryRequest struct {
	Workspace      *string `json:"workspace,omitempty"`
	Tag            *string `json:"tag,omitempty"`
	NamePrefix     *string `json:"name_prefix,omitempty"`
	RunID          *string `json:"run_id,omitempty"`
	Phase          *string `json:"phase,omitempty"`
	Role           *string `json:"role,omitempty"`
	Limit          int     `json:"limit,omitempty"`
	Offset         int     `json:"offset,omitempty"`
	IncludeDeleted bool    `json:"include_deleted,omitempty"`
}

// ExportRequest represents the arguments for export.
type ExportRequest struct {
	Path           string  `json:"path,omitempty"`
	Workspace      *string `json:"workspace,omitempty"`
	IncludeDeleted bool    `json:"include_deleted,omitempty"`
}

// ImportRequest represents the arguments for import.
type ImportRequest struct {
	Path string `json:"path"`
	Mode string `json:"mode,omitempty"`
}

// PurgeRequest represents the arguments for purge.
type PurgeRequest struct {
	Workspace     *string `json:"workspace,omitempty"`
	OlderThanDays *int    `json:"older_than_days,omitempty"`
}

// Handler implementations

// HandleStore handles the store tool call.
func (h *Handlers) HandleStore(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decode[StoreRequest](req)
	if err != nil {
		return errorResult(errors.NewInvalidRequest(err.Error())), nil
	}

	// Map to ops input
	mode := ops.StoreModeError
	if input.Mode == "replace" {
		mode = ops.StoreModeReplace
	}

	result, err := ops.Store(h.db, h.cfg, ops.StoreInput{
		Workspace:   input.Workspace,
		Name:        input.Name,
		Title:       input.Title,
		CapsuleText: input.CapsuleText,
		Tags:        input.Tags,
		Source:      input.Source,
		RunID:       input.RunID,
		Phase:       input.Phase,
		Role:        input.Role,
		Mode:        mode,
		AllowThin:   input.AllowThin,
	})
	if err != nil {
		return errorResult(err), nil
	}

	return successResult(result)
}

// HandleFetch handles the fetch tool call.
func (h *Handlers) HandleFetch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decode[FetchRequest](req)
	if err != nil {
		return errorResult(errors.NewInvalidRequest(err.Error())), nil
	}

	result, err := ops.Fetch(h.db, ops.FetchInput{
		ID:             input.ID,
		Workspace:      input.Workspace,
		Name:           input.Name,
		IncludeDeleted: input.IncludeDeleted,
		IncludeText:    input.IncludeText,
	})
	if err != nil {
		return errorResult(err), nil
	}

	return successResult(result)
}

// HandleFetchMany handles the fetch_many tool call.
func (h *Handlers) HandleFetchMany(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decode[FetchManyRequest](req)
	if err != nil {
		return errorResult(errors.NewInvalidRequest(err.Error())), nil
	}

	// Convert refs
	refs := make([]ops.FetchManyRef, len(input.Items))
	for i, item := range input.Items {
		refs[i] = ops.FetchManyRef{
			ID:        item.ID,
			Workspace: item.Workspace,
			Name:      item.Name,
		}
	}

	result, err := ops.FetchMany(h.db, ops.FetchManyInput{
		Items:          refs,
		IncludeText:    input.IncludeText,
		IncludeDeleted: input.IncludeDeleted,
	})
	if err != nil {
		return errorResult(err), nil
	}

	return successResult(result)
}

// HandleUpdate handles the update tool call.
func (h *Handlers) HandleUpdate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decode[UpdateRequest](req)
	if err != nil {
		return errorResult(errors.NewInvalidRequest(err.Error())), nil
	}

	result, err := ops.Update(h.db, h.cfg, ops.UpdateInput{
		ID:          input.ID,
		Workspace:   input.Workspace,
		Name:        input.Name,
		CapsuleText: input.CapsuleText,
		Title:       input.Title,
		Tags:        input.Tags,
		Source:      input.Source,
		RunID:       input.RunID,
		Phase:       input.Phase,
		Role:        input.Role,
		AllowThin:   input.AllowThin,
	})
	if err != nil {
		return errorResult(err), nil
	}

	return successResult(result)
}

// HandleDelete handles the delete tool call.
func (h *Handlers) HandleDelete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decode[DeleteRequest](req)
	if err != nil {
		return errorResult(errors.NewInvalidRequest(err.Error())), nil
	}

	result, err := ops.Delete(h.db, ops.DeleteInput{
		ID:        input.ID,
		Workspace: input.Workspace,
		Name:      input.Name,
	})
	if err != nil {
		return errorResult(err), nil
	}

	return successResult(result)
}

// HandleLatest handles the latest tool call.
func (h *Handlers) HandleLatest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decode[LatestRequest](req)
	if err != nil {
		return errorResult(errors.NewInvalidRequest(err.Error())), nil
	}

	result, err := ops.Latest(h.db, ops.LatestInput{
		Workspace:      input.Workspace,
		RunID:          input.RunID,
		Phase:          input.Phase,
		Role:           input.Role,
		IncludeText:    input.IncludeText,
		IncludeDeleted: input.IncludeDeleted,
	})
	if err != nil {
		return errorResult(err), nil
	}

	return successResult(result)
}

// HandleList handles the list tool call.
func (h *Handlers) HandleList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decode[ListRequest](req)
	if err != nil {
		return errorResult(errors.NewInvalidRequest(err.Error())), nil
	}

	result, err := ops.List(h.db, ops.ListInput{
		Workspace:      input.Workspace,
		RunID:          input.RunID,
		Phase:          input.Phase,
		Role:           input.Role,
		Limit:          input.Limit,
		Offset:         input.Offset,
		IncludeDeleted: input.IncludeDeleted,
	})
	if err != nil {
		return errorResult(err), nil
	}

	return successResult(result)
}

// HandleInventory handles the inventory tool call.
func (h *Handlers) HandleInventory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decode[InventoryRequest](req)
	if err != nil {
		return errorResult(errors.NewInvalidRequest(err.Error())), nil
	}

	result, err := ops.Inventory(h.db, ops.InventoryInput{
		Workspace:      input.Workspace,
		Tag:            input.Tag,
		NamePrefix:     input.NamePrefix,
		RunID:          input.RunID,
		Phase:          input.Phase,
		Role:           input.Role,
		Limit:          input.Limit,
		Offset:         input.Offset,
		IncludeDeleted: input.IncludeDeleted,
	})
	if err != nil {
		return errorResult(err), nil
	}

	return successResult(result)
}

// HandleExport handles the export tool call.
func (h *Handlers) HandleExport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decode[ExportRequest](req)
	if err != nil {
		return errorResult(errors.NewInvalidRequest(err.Error())), nil
	}

	result, err := ops.Export(h.db, ops.ExportInput{
		Path:           input.Path,
		Workspace:      input.Workspace,
		IncludeDeleted: input.IncludeDeleted,
	})
	if err != nil {
		return errorResult(err), nil
	}

	return successResult(result)
}

// HandleImport handles the import tool call.
func (h *Handlers) HandleImport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decode[ImportRequest](req)
	if err != nil {
		return errorResult(errors.NewInvalidRequest(err.Error())), nil
	}

	// Map to ops input
	mode := ops.ImportModeError
	switch input.Mode {
	case "replace":
		mode = ops.ImportModeReplace
	case "rename":
		mode = ops.ImportModeRename
	}

	result, err := ops.Import(h.db, ops.ImportInput{
		Path: input.Path,
		Mode: mode,
	})
	if err != nil {
		return errorResult(err), nil
	}

	return successResult(result)
}

// HandlePurge handles the purge tool call.
func (h *Handlers) HandlePurge(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := decode[PurgeRequest](req)
	if err != nil {
		return errorResult(errors.NewInvalidRequest(err.Error())), nil
	}

	result, err := ops.Purge(h.db, ops.PurgeInput{
		Workspace:     input.Workspace,
		OlderThanDays: input.OlderThanDays,
	})
	if err != nil {
		return errorResult(err), nil
	}

	return successResult(result)
}

// Result helpers

// errorResult creates an MCP error result from any error.
// Uses IsError: true so MCP clients recognize failures properly.
// Note: Internal error details are not exposed to prevent leaking sensitive info.
func errorResult(err error) *mcp.CallToolResult {
	var payload map[string]any

	if mossErr, ok := err.(*errors.MossError); ok {
		errorObj := map[string]any{
			"code":    mossErr.Code,
			"message": mossErr.Message,
			"status":  mossErr.Status,
		}
		// Only include details for non-internal errors to avoid leaking
		// sensitive info like file paths or SQL errors
		if mossErr.Code != errors.ErrInternal && mossErr.Details != nil {
			errorObj["details"] = mossErr.Details
		}
		payload = map[string]any{"error": errorObj}
	} else {
		payload = map[string]any{
			"error": map[string]any{
				"code":    "INTERNAL",
				"message": "an internal error occurred",
				"status":  500,
			},
		}
	}

	content, _ := json.Marshal(payload)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: string(content)}},
		IsError: true,
	}
}

// successResult creates an MCP success result from any data.
func successResult(data any) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultJSON(data)
}
