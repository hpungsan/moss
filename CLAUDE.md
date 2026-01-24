# Claude Code Instructions

## Project: Moss
Local context capsule store for AI session handoffs.

## Tech Stack
Go, SQLite (modernc.org/sqlite), MCP (github.com/mark3labs/mcp-go)

## Key Concepts
- **Capsule**: Distilled context snapshot (Objective, Status, Decisions, Next actions, Key locations, Open questions)
- **Workspace**: Namespace (default: "default")
- **Name**: Unique handle within workspace

## MCP Tools
`moss.store` `moss.fetch` `moss.fetch_many` `moss.update` `moss.delete` `moss.list` `moss.inventory` `moss.latest` `moss.export` `moss.import` `moss.purge`

## Guidelines
- MCP-first (CLI is secondary)
- Explicit only (no auto-save/load)
- Low-bloat (size limits + lint)

## Commands
```
go build ./...          # Build all
go test ./...           # Run tests
go test ./... -v        # Verbose tests
go fmt ./...            # Format code
go mod tidy             # Clean dependencies
golangci-lint run       # Lint
```

## Package Structure
```
internal/
├── capsule/     # Capsule type, normalize, lint (6 required sections)
├── config/      # Config loader (~/.moss/config.json)
├── db/          # SQLite init, migrations, queries (CRUD)
├── errors/      # MossError with codes (400/404/409/413/422/500)
└── ops/         # Business logic: Store, Fetch, Update, Delete
```

## Paths
- DB: `~/.moss/moss.db`
- Tasks: `~/.claude/tasks/` (CC Tasks integration, see `docs/agents/TASKS.md`)

## Docs
`docs/agents/` — supplementary reference docs for AI agents

| Doc | Purpose |
|-----|---------|
| `docs/moss/OVERVIEW.md` | Concepts, use cases |
| `docs/moss/v1.0/DESIGN.md` | API spec + implementation details (v1.0) |
| `docs/moss/v1.0/BACKLOG.md` | Post-v1.0 features |
| `docs/agents/CODEMAP.md` | File-level lookup table |
| `docs/agents/TASKS.md` | CC Tasks integration |

## Dev (gitignored)
| Doc | Purpose |
|-----|---------|
| `dev/build/v1.0/BUILD.md` | Build phases + task checklist |
