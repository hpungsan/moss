# Claude Code Instructions

## Project: Moss
Local context store for AI session handoffs and multi-agent orchestration.

## Tech Stack
Go, SQLite (modernc.org/sqlite), MCP (github.com/mark3labs/mcp-go), CLI (github.com/urfave/cli/v2)

## Key Concepts
- **Type**: Category of stored object (capsule)
- **Capsule**: Markdown-based context for LLM consumption (6 required sections)
- **Workspace**: Namespace (default: "default")
- **Name**: Unique handle within workspace
- **Orchestration**: `run_id`, `phase`, `role` for multi-agent workflow scoping

## MCP Tools

### Capsule
`capsule_store` `capsule_fetch` `capsule_fetch_many` `capsule_update` `capsule_delete` `capsule_list` `capsule_inventory` `capsule_search` `capsule_latest` `capsule_export` `capsule_import` `capsule_purge` `capsule_bulk_delete` `capsule_bulk_update` `capsule_compose` `capsule_append`

## Guidelines
- MCP-first (CLI is secondary)
- Explicit only (no auto-save/load)
- Low-bloat (size limits + lint)
- Type-specific (each type optimized for its consumer)

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

Also: `Makefile` (human convenience), `docs/capsule/RUNBOOK.md` (operational guide).

## Package Structure
```
cmd/moss/        # main.go (entrypoint), cli.go (CLI commands)
internal/
├── capsule/     # Capsule type, normalize, lint (6 required sections)
├── config/      # Config loader (~/.moss/config.json)
├── db/          # SQLite init, migrations, queries (CRUD)
├── errors/      # MossError with codes (400/404/409/413/422/499/500)
├── mcp/         # MCP server, tool definitions, handlers
└── ops/         # Business logic (capsule operations)
```

## Paths
- DB: `~/.moss/moss.db`
- Config: `~/.moss/config.json` (global), `.moss/config.json` (repo)
- Skills: `.claude/skills/moss-capsule/` (capsule skill)

## Docs
`docs/` — reference documentation

| Doc | Purpose |
|-----|---------|
| `docs/README.md` | Concepts, use cases, workflow overview |
| `docs/SETUP.md` | Installation and paths |
| `docs/capsule/DESIGN.md` | Capsule API spec + implementation |
| `docs/capsule/BACKLOG.md` | Post-v1 capsule features |
| `docs/capsule/RUNBOOK.md` | Capsule operations guide |
| `docs/integrations/claude-code.md` | Claude Code integration |
| `docs/agents/CODEMAP.md` | File-level lookup table |
| `docs/agents/MOSS_CC.md` | Claude Code patterns |
