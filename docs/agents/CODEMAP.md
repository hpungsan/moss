# Codemap

## Directory Structure

```
moss/
├── cmd/
│   └── moss/
│       └── main.go                # CLI entrypoint
├── internal/
│   ├── capsule/
│   │   ├── capsule.go             # Capsule struct
│   │   ├── normalize.go           # Normalize, CountChars, EstimateTokens
│   │   ├── normalize_test.go
│   │   ├── lint.go                # Section detection, size validation
│   │   └── lint_test.go
│   ├── config/
│   │   ├── config.go              # Config loader
│   │   └── config_test.go
│   ├── db/
│   │   ├── db.go                  # Init, schema, WAL setup
│   │   ├── db_test.go
│   │   ├── queries.go             # Insert, GetByID, GetByName, UpdateByID, SoftDelete
│   │   └── queries_test.go
│   ├── errors/
│   │   ├── errors.go              # MossError, error codes, constructors
│   │   └── errors_test.go
│   └── ops/
│       ├── ops.go                 # Address validation, TaskLink
│       ├── ops_test.go
│       ├── store.go               # Store operation (create/replace)
│       ├── store_test.go
│       ├── fetch.go               # Fetch operation
│       ├── fetch_test.go
│       ├── update.go              # Update operation
│       ├── update_test.go
│       ├── delete.go              # Delete operation (soft delete)
│       └── delete_test.go
├── devDocs/
│   ├── moss/
│   │   ├── OVERVIEW.md            # Concepts, use cases
│   │   └── v1.0/
│   │       ├── DESIGN.md          # API spec + implementation details
│   │       └── BACKLOG.md         # Post-v1.0 features
│   └── build/
│       └── v1.0/
│           └── BUILD.md           # Build phases
├── docs/
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
| `internal/capsule/` | Capsule struct, normalization, linting (6 required sections) |
| `internal/db/` | SQLite init, schema, CRUD queries |
| `internal/config/` | Config loading from ~/.moss/config.json |
| `internal/errors/` | Structured errors with codes (400/404/409/413/422/500) |
| `internal/ops/` | Business logic: Store, Fetch, Update, Delete |
| `devDocs/moss/v1.0/DESIGN.md` | Full v1.0 spec |

## Notes

- `internal/` packages are not importable outside this module (Go convention)
- DB file: `~/.moss/moss.db`
- Config file: `~/.moss/config.json`
- All operations use soft delete (deleted_at timestamp)
