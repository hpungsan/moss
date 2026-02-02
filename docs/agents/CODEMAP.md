# Codemap

## Directory Structure

```
moss/
├── cmd/
│   └── moss/
│       ├── main.go                # Entrypoint (MCP server or CLI routing)
│       └── cli.go                 # CLI app with 10 commands (urfave/cli/v2)
├── internal/
│   ├── capsule/
│   │   ├── capsule.go             # Capsule struct
│   │   ├── summary.go             # CapsuleSummary struct (browse operations)
│   │   ├── normalize.go           # Normalize, CountChars, EstimateTokens
│   │   ├── lint.go                # Section detection, size validation
│   │   └── export.go              # ExportRecord, ToCapsule, CapsuleToExportRecord
│   ├── config/
│   │   └── config.go              # Config loader (~/.moss/config.json)
│   ├── db/
│   │   ├── db.go                  # Init, schema, WAL setup
│   │   └── queries.go             # Querier interface, Insert, GetByID, GetByName,
│   │                              # UpdateByID, SoftDelete, ListByWorkspace, ListAll,
│   │                              # GetLatestSummary, GetLatestFull, SearchFullText,
│   │                              # StreamForExport, UpdateFull, FindUniqueName,
│   │                              # PurgeDeleted, BulkSoftDelete, BulkUpdate
│   ├── errors/
│   │   └── errors.go              # MossError, error codes (400/404/409/413/422/499/500)
│   ├── mcp/
│   │   ├── decode.go              # Generic decode[T] helper for MCP requests
│   │   ├── handlers.go            # Tool handlers calling ops functions
│   │   ├── server.go              # NewServer, Run (stdio transport)
│   │   └── tools.go               # 15 tool definitions with JSON schemas
│   └── ops/
│       ├── ops.go                 # Address validation, FetchKey
│       ├── store.go               # Store operation (create/replace)
│       ├── fetch.go               # Fetch operation
│       ├── fetch_many.go          # FetchMany operation (batch fetch)
│       ├── update.go              # Update operation
│       ├── delete.go              # Delete operation (soft delete)
│       ├── list.go                # List operation (workspace-scoped)
│       ├── inventory.go           # Inventory operation (global)
│       ├── search.go              # Search operation (FTS5 full-text search)
│       ├── latest.go              # Latest operation
│       ├── export.go              # Export to JSONL
│       ├── import.go              # Import from JSONL
│       ├── purge.go               # Purge soft-deleted capsules
│       ├── bulk_delete.go         # Bulk soft-delete by filter
│       ├── bulk_update.go         # Bulk metadata update by filter
│       ├── compose.go             # Compose multiple capsules into bundle
│       ├── pathcheck.go           # Path validation for import/export security
│       ├── fileopen_unix.go       # O_NOFOLLOW file open (Unix/Darwin/Linux)
│       └── fileopen_windows.go    # File open fallback (Windows)
├── docs/
│   ├── README.md                  # Concepts, use cases, workflow overview
│   ├── capsule/
│   │   ├── DESIGN.md              # Capsule API spec + implementation
│   │   ├── BACKLOG.md             # Post-v1 capsule features
│   │   └── RUNBOOK.md             # Capsule operations guide
│   ├── agents/
│   │   ├── CODEMAP.md             # This file
│   │   ├── MOSS_CC.md             # Claude Code integration
│   │   └── upgrade.md             # Agent upgrade notes
│   ├── SETUP.md                   # Installation and paths
│   └── integrations/
│       └── claude-code.md         # Claude Code MCP integration
├── .github/
│   └── workflows/
│       └── ci.yml                 # CI pipeline
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
├── CLAUDE.md                      # Claude Code instructions
└── README.md
```

## Key Paths

| Path | Purpose |
|------|---------|
| `internal/capsule/` | Capsule struct, normalization, linting (6 required sections), export record conversion |
| `internal/db/` | SQLite init, schema, CRUD + browse queries, Querier interface for transactions |
| `internal/config/` | Config loading from ~/.moss/config.json |
| `internal/errors/` | Structured errors with codes (400/404/409/413/422/499/500) |
| `internal/mcp/` | MCP server exposing 15 tools via stdio transport |
| `internal/ops/` | Business logic: Store, Fetch, FetchMany, Update, Delete, List, Inventory, Search, Latest, Export, Import, Purge, BulkDelete, BulkUpdate, Compose |
| `docs/capsule/DESIGN.md` | Capsule API spec |

## Notes

- `internal/` packages are not importable outside this module (Go convention)
- DB file: `~/.moss/moss.db`
- Config file: `~/.moss/config.json`
- Exports dir: `~/.moss/exports/` (created by Export, default output location)
- All operations use soft delete (deleted_at timestamp)
