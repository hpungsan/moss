package mcp

import (
	"context"
	"database/sql"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/hpungsan/moss/internal/config"
)

// KnownTypes lists all valid type names.
var KnownTypes = []string{"capsule"}

// toolEntry pairs a tool definition with a handler factory.
type toolEntry struct {
	def     mcp.Tool
	handler func(*Handlers) server.ToolHandlerFunc
}

// toolRegistry maps tool names to their definitions and handler factories.
var toolRegistry = map[string]toolEntry{
	"capsule_store": {
		def:     storeToolDef,
		handler: func(h *Handlers) server.ToolHandlerFunc { return h.HandleStore },
	},
	"capsule_fetch": {
		def:     fetchToolDef,
		handler: func(h *Handlers) server.ToolHandlerFunc { return h.HandleFetch },
	},
	"capsule_fetch_many": {
		def:     fetchManyToolDef,
		handler: func(h *Handlers) server.ToolHandlerFunc { return h.HandleFetchMany },
	},
	"capsule_update": {
		def:     updateToolDef,
		handler: func(h *Handlers) server.ToolHandlerFunc { return h.HandleUpdate },
	},
	"capsule_delete": {
		def:     deleteToolDef,
		handler: func(h *Handlers) server.ToolHandlerFunc { return h.HandleDelete },
	},
	"capsule_latest": {
		def:     latestToolDef,
		handler: func(h *Handlers) server.ToolHandlerFunc { return h.HandleLatest },
	},
	"capsule_list": {
		def:     listToolDef,
		handler: func(h *Handlers) server.ToolHandlerFunc { return h.HandleList },
	},
	"capsule_inventory": {
		def:     inventoryToolDef,
		handler: func(h *Handlers) server.ToolHandlerFunc { return h.HandleInventory },
	},
	"capsule_search": {
		def:     searchToolDef,
		handler: func(h *Handlers) server.ToolHandlerFunc { return h.HandleSearch },
	},
	"capsule_export": {
		def:     exportToolDef,
		handler: func(h *Handlers) server.ToolHandlerFunc { return h.HandleExport },
	},
	"capsule_import": {
		def:     importToolDef,
		handler: func(h *Handlers) server.ToolHandlerFunc { return h.HandleImport },
	},
	"capsule_purge": {
		def:     purgeToolDef,
		handler: func(h *Handlers) server.ToolHandlerFunc { return h.HandlePurge },
	},
	"capsule_bulk_delete": {
		def:     bulkDeleteToolDef,
		handler: func(h *Handlers) server.ToolHandlerFunc { return h.HandleBulkDelete },
	},
	"capsule_bulk_update": {
		def:     bulkUpdateToolDef,
		handler: func(h *Handlers) server.ToolHandlerFunc { return h.HandleBulkUpdate },
	},
	"capsule_compose": {
		def:     composeToolDef,
		handler: func(h *Handlers) server.ToolHandlerFunc { return h.HandleCompose },
	},
	"capsule_append": {
		def:     appendToolDef,
		handler: func(h *Handlers) server.ToolHandlerFunc { return h.HandleAppend },
	},
}

// AllToolNames returns a list of all valid tool names.
func AllToolNames() []string {
	names := make([]string, 0, len(toolRegistry))
	for name := range toolRegistry {
		names = append(names, name)
	}
	return names
}

// ValidateDisabledTools returns a list of unknown tool names from the given list.
func ValidateDisabledTools(names []string) []string {
	unknown := make([]string, 0)
	for _, name := range names {
		if _, ok := toolRegistry[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	return unknown
}

// ValidateDisabledTypes returns a list of unknown type names from the given list.
func ValidateDisabledTypes(names []string) []string {
	known := make(map[string]bool, len(KnownTypes))
	for _, t := range KnownTypes {
		known[t] = true
	}

	unknown := make([]string, 0)
	for _, name := range names {
		if !known[name] {
			unknown = append(unknown, name)
		}
	}
	return unknown
}

// GetTypeForTool extracts the type name from a tool name.
// Tool names follow the pattern "type_action" (e.g., "capsule_store" → "capsule").
func GetTypeForTool(toolName string) string {
	if idx := strings.Index(toolName, "_"); idx > 0 {
		return toolName[:idx]
	}
	return ""
}

// ExpandTypesToTools returns all tool names belonging to the given types.
func ExpandTypesToTools(types []string) []string {
	if len(types) == 0 {
		return nil
	}

	// Build set of types for O(1) lookup
	typeSet := make(map[string]bool, len(types))
	for _, t := range types {
		typeSet[t] = true
	}

	// Collect tools belonging to disabled types
	tools := make([]string, 0)
	for name := range toolRegistry {
		typ := GetTypeForTool(name)
		if typeSet[typ] {
			tools = append(tools, name)
		}
	}
	return tools
}

// NewServer creates a new MCP server with Moss tools registered.
// Tools listed in cfg.DisabledTools or belonging to cfg.DisabledTypes
// are excluded from registration.
func NewServer(db *sql.DB, cfg *config.Config, version string) *server.MCPServer {
	s := server.NewMCPServer(
		"moss",
		version,
		server.WithToolCapabilities(true),
	)

	h := NewHandlers(db, cfg)

	// Build set of disabled tools: first expand types, then add individual tools
	disabled := make(map[string]bool)
	for _, tool := range ExpandTypesToTools(cfg.DisabledTypes) {
		disabled[tool] = true
	}
	for _, name := range cfg.DisabledTools {
		disabled[name] = true
	}

	// Register tools (skip disabled)
	for name, entry := range toolRegistry {
		if disabled[name] {
			continue
		}
		s.AddTool(entry.def, entry.handler(h))
	}

	return s
}

// Run starts the MCP server using stdio transport.
func Run(db *sql.DB, cfg *config.Config, version string) error {
	s := NewServer(db, cfg, version)
	return server.ServeStdio(s)
}

// ToolHandlerFunc is the signature for tool handlers.
type ToolHandlerFunc func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
