# Moss

Local context capsule store for portable AI handoffs, multi-agent orchestration, and cross-tool context sharing.

## The Problem

AI coding sessions lose context when you switch tools or start fresh. Copy-pasting full chat history is bloated and noisy. Moss solves this with **capsules**—distilled context snapshots that preserve what matters.

## What Moss Does

- **Hand off sessions:** Session A → Moss → Session B, across tools (Claude Code, Codex, etc.)
- **Orchestrate agents:** give parallel agents a shared context layer
- **Stay portable:** export/import for backup and cross-machine transfer

## Installation

### Download Pre-built Binary

Binaries available on the [Releases](https://github.com/hpungsan/moss/releases) page:

| Platform | Binary |
|----------|--------|
| macOS (Apple Silicon) | `moss-darwin-arm64` |
| macOS (Intel) | `moss-darwin-amd64` |
| Linux (x64) | `moss-linux-amd64` |
| Linux (ARM64) | `moss-linux-arm64` |
| Windows (x64) | `moss-windows-amd64.exe` |

Download and install:
```bash
# Example: macOS Apple Silicon
curl -LO https://github.com/hpungsan/moss/releases/latest/download/moss-darwin-arm64
chmod +x moss-darwin-arm64
sudo mv moss-darwin-arm64 /usr/local/bin/moss
```

### From Source

```bash
git clone https://github.com/hpungsan/moss.git
cd moss
go build ./cmd/moss
```

### Claude Code Integration

See [Claude Code Setup](docs/setup/claude-code.md) for full setup including skills and subagents.

## Capsule Structure

A capsule is not a chat log. It's a structured summary:

| Section | Purpose |
|---------|---------|
| Objective | What you're trying to accomplish |
| Current status | Where things stand now |
| Decisions/constraints | Choices made and why |
| Next actions | What to do next |
| Key locations | Files, URLs, commands |
| Open questions | Unresolved issues |

Capsules are size-bounded and linted to stay useful. See [examples/capsule.md](examples/capsule.md) for a complete example.

**Orchestration fields** (MCP only): `run_id`, `phase`, `role` enable multi-agent workflow scoping.

## Quick Start

### Store a Capsule

```
store {
  "workspace": "myproject",
  "name": "auth",
  "capsule_text": "## Objective\nImplement JWT auth\n## Current status\nMiddleware done\n## Decisions\nUsing RS256\n## Next actions\nAdd refresh tokens\n## Key locations\nsrc/auth/\n## Open questions\nToken expiry policy?"
}
```

### Fetch by Name

```
fetch { "workspace": "myproject", "name": "auth" }
```

### List All Capsules

```
inventory {}
```

### Search Capsules

```
search { "query": "authentication", "workspace": "myproject" }
```

### Get Latest in Workspace

```
latest { "workspace": "myproject", "include_text": true }
```

### Export for Backup

```
export {}
```

## MCP Tools

| Tool | Description |
|------|-------------|
| `store` | Create a new capsule |
| `fetch` | Retrieve by ID or name |
| `fetch_many` | Batch fetch multiple |
| `update` | Update existing capsule |
| `delete` | Soft-delete (recoverable) |
| `latest` | Most recent in workspace |
| `list` | List capsules in workspace |
| `inventory` | List all capsules globally |
| `search` | Full-text search across capsules |
| `export` | JSONL backup |
| `import` | JSONL restore |
| `purge` | Permanent delete |
| `bulk_delete` | Soft-delete multiple by filter |
| `bulk_update` | Update metadata on multiple |
| `compose` | Assemble multiple capsules |

## CLI

The CLI mirrors MCP operations for debugging and scripting. Note: orchestration fields (`run_id`, `phase`, `role`) are MCP-only.

```bash
# Store (reads capsule from stdin)
echo "## Objective
..." | moss store --name=auth

# Fetch
moss fetch --name=auth

# List all
moss inventory

# Export
moss export
```

Run `moss --help` for all commands.

## Design Principles

- **Local-first:** SQLite at `~/.moss/moss.db`, no external services
- **MCP-first:** Native tool for AI agents, CLI for debugging
- **Explicit only:** No auto-save, no auto-load
- **Low-bloat:** Size limits + lint rules enforce quality

## Documentation

- [Overview & Use Cases](docs/OVERVIEW.md)
- [Design Spec](docs/DESIGN.md)
- [Runbook](docs/RUNBOOK.md) — Installation, configuration, troubleshooting
- [Backlog](docs/BACKLOG.md) — Future features

## License

MIT
