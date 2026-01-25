# Codemap

## Directory Structure

```
moss/
├── cmd/
│   └── moss/
│       └── main.go                # CLI entrypoint
├── internal/
│   ├── capsule/
│   │   ├── capsule.go             # Capsule struct, CapsuleSummary
│   │   ├── normalize.go           # Normalize, CountChars, EstimateTokens
│   │   ├── lint.go                # Section detection, size validation
│   │   └── export.go              # ExportRecord, ToCapsule, CapsuleToExportRecord
│   ├── config/
│   │   └── config.go              # Config loader (~/.moss/config.json)
│   ├── db/
│   │   ├── db.go                  # Init, schema, WAL setup
│   │   └── queries.go             # Querier interface, Insert, GetByID, GetByName,
│   │                              # UpdateByID, SoftDelete, ListByWorkspace, ListAll,
│   │                              # GetLatestSummary, GetLatestFull, StreamForExport,
│   │                              # UpdateFull, FindUniqueName, PurgeDeleted
│   ├── errors/
│   │   └── errors.go              # MossError, error codes (400/404/409/413/422/500)
│   └── ops/
│       ├── ops.go                 # Address validation, TaskLink
│       ├── store.go               # Store operation (create/replace)
│       ├── fetch.go               # Fetch, FetchMany operations
│       ├── update.go              # Update operation
│       ├── delete.go              # Delete operation (soft delete)
│       ├── list.go                # List operation (workspace-scoped)
│       ├── inventory.go           # Inventory operation (global)
│       ├── latest.go              # Latest operation
│       ├── export.go              # Export to JSONL
│       ├── import.go              # Import from JSONL
│       └── purge.go               # Purge soft-deleted capsules
├── docs/
│   ├── moss/
│   │   ├── OVERVIEW.md            # Concepts, use cases
│   │   └── v1.0/
│   │       ├── DESIGN.md          # API spec + implementation details
│   │       └── BACKLOG.md         # Post-v1.0 features
│   └── agents/
│       ├── CODEMAP.md             # This file
│       └── TASKS.md               # Claude Code Tasks integration
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
| `internal/errors/` | Structured errors with codes (400/404/409/413/422/500) |
| `internal/ops/` | Business logic: Store, Fetch, FetchMany, Update, Delete, List, Inventory, Latest, Export, Import, Purge |
| `docs/moss/v1.0/DESIGN.md` | Full v1.0 spec |

## Notes

- `internal/` packages are not importable outside this module (Go convention)
- DB file: `~/.moss/moss.db`
- Config file: `~/.moss/config.json`
- Exports dir: `~/.moss/exports/` (created by Export, default output location)
- All operations use soft delete (deleted_at timestamp)
