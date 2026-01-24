# Codemap

## Directory Structure

```
moss/
├── cmd/
│   └── moss/
│       └── main.go              # CLI entrypoint
├── internal/
│   ├── capsule/
│   │   ├── capsule.go           # Capsule struct
│   │   ├── normalize.go         # Normalize, CountChars, EstimateTokens
│   │   ├── normalize_test.go
│   │   └── lint.go              # Stub for Phase 2
│   ├── config/
│   │   └── config.go            # Config loader
│   └── db/
│       ├── db.go                # Init, schema, WAL setup
│       ├── db_test.go
│       └── queries.go           # CRUD stubs
├── devDocs/
│   └── moss/
│       ├── OVERVIEW.md          # Concepts, use cases
│       └── v1.0/
│           ├── DESIGN.md        # API spec + implementation details
│           └── BACKLOG.md       # Post-v1.0 features
├── docs/
│   └── agents/
│       ├── CODEMAP.md           # This file
│       └── TASKS.md             # Claude Code Tasks integration
├── .github/
│   └── workflows/
│       └── ci.yml               # CI pipeline
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
├── CLAUDE.md                    # Claude Code instructions
└── README.md
```

## Key Paths

| Path | Purpose |
|------|---------|
| `internal/capsule/` | Capsule struct, normalization, linting |
| `internal/db/` | SQLite init, schema, queries |
| `internal/config/` | Config loading from ~/.moss/config.json |
| `devDocs/moss/v1.0/DESIGN.md` | Full v1.0 spec |

## Notes

- `internal/` packages are not importable outside this module (Go convention)
- DB file: `~/.moss/moss.db`
- Config file: `~/.moss/config.json`
