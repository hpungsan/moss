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
go build | go run . | go test ./...

## Paths
- DB: `~/.moss/moss.db`
- Tasks: `~/.claude/tasks/` (CC Tasks integration, see `docs/agents/TASKS.md`)

## Docs
| Doc | Purpose |
|-----|---------|
| `devDocs/moss/OVERVIEW.md` | Concepts, use cases |
| `devDocs/moss/v1.0/DESIGN.md` | API spec + implementation details (v1.0) |
| `devDocs/moss/v1.0/BACKLOG.md` | Post-v1.0 features |
| `docs/agents/TASKS.md` | CC Tasks integration |
| `docs/agents/CODEMAP.md` | Directory structure |
