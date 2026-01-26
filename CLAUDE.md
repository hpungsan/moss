# Claude Code Instructions

## Project: Moss
Local context capsule store for AI session handoffs.

## Tech Stack
Go, SQLite (modernc.org/sqlite), MCP (github.com/mark3labs/mcp-go), CLI (github.com/urfave/cli/v2)

## Key Concepts
- **Capsule**: Distilled context snapshot (Objective, Status, Decisions, Next actions, Key locations, Open questions)
- **Workspace**: Namespace (default: "default")
- **Name**: Unique handle within workspace
- **Orchestration**: `run_id`, `phase`, `role` for multi-agent workflow scoping

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
go test -v ./...        # Verbose tests
go fmt ./...            # Format code
golangci-lint run       # Lint
make build-all          # Cross-compile all platforms
make build-checksums    # Cross-compile + SHA256 checksums
```

## CLI
```
moss store --name=X < capsule.md   # Store capsule
moss fetch --name=X                # Fetch by name
moss fetch <id>                    # Fetch by ID
moss list                          # List in workspace
moss inventory                     # List all
moss --help                        # All commands
```

Also: `Makefile` (human convenience), `docs/moss/v1/RUNBOOK.md` (operational guide).

## Package Structure
```
cmd/moss/        # main.go (entrypoint), cli.go (CLI commands)
internal/
├── capsule/     # Capsule type, normalize, lint (6 required sections)
├── config/      # Config loader (~/.moss/config.json)
├── db/          # SQLite init, migrations, queries (CRUD)
├── errors/      # MossError with codes (400/404/409/413/422/500)
├── mcp/         # MCP server, tool definitions, handlers
└── ops/         # Business logic (11 operations)
```

## Paths
- DB: `~/.moss/moss.db`
- Tasks: `~/.claude/tasks/` (CC Tasks integration, see `docs/agents/MOSS_CC.md`)

## Docs
`docs/agents/` — supplementary reference docs for AI agents

| Doc | Purpose |
|-----|---------|
| `docs/moss/OVERVIEW.md` | Concepts, use cases |
| `docs/moss/v1/DESIGN.md` | API spec + implementation details (v1) |
| `docs/moss/v1/RUNBOOK.md` | Build, configure, run, troubleshoot |
| `docs/moss/v1/BACKLOG.md` | Post-v1 features |
| `docs/agents/CODEMAP.md` | File-level lookup table |
| `docs/agents/MOSS_CC.md` | Claude Code integration |

## Dev (gitignored)
| Doc | Purpose |
|-----|---------|
| `dev/build/v1/BUILD.md` | Build phases + task checklist |
