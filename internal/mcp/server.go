package mcp

import (
	"database/sql"

	"github.com/mark3labs/mcp-go/server"

	"github.com/hpungsan/moss/internal/config"
)

// NewServer creates a new MCP server with all Moss tools registered.
func NewServer(db *sql.DB, cfg *config.Config, version string) *server.MCPServer {
	s := server.NewMCPServer(
		"moss",
		version,
		server.WithToolCapabilities(true),
	)

	h := NewHandlers(db, cfg)

	// Register all 15 tools
	s.AddTool(storeToolDef, h.HandleStore)
	s.AddTool(fetchToolDef, h.HandleFetch)
	s.AddTool(fetchManyToolDef, h.HandleFetchMany)
	s.AddTool(updateToolDef, h.HandleUpdate)
	s.AddTool(deleteToolDef, h.HandleDelete)
	s.AddTool(latestToolDef, h.HandleLatest)
	s.AddTool(listToolDef, h.HandleList)
	s.AddTool(inventoryToolDef, h.HandleInventory)
	s.AddTool(searchToolDef, h.HandleSearch)
	s.AddTool(exportToolDef, h.HandleExport)
	s.AddTool(importToolDef, h.HandleImport)
	s.AddTool(purgeToolDef, h.HandlePurge)
	s.AddTool(bulkDeleteToolDef, h.HandleBulkDelete)
	s.AddTool(bulkUpdateToolDef, h.HandleBulkUpdate)
	s.AddTool(composeToolDef, h.HandleCompose)

	return s
}

// Run starts the MCP server using stdio transport.
func Run(db *sql.DB, cfg *config.Config, version string) error {
	s := NewServer(db, cfg, version)
	return server.ServeStdio(s)
}
