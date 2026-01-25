# Moss

Local context capsule store for portable AI handoffs, multi-agent orchestration, and cross-tool context sharing.

## The Problem

AI coding sessions lose context when you switch tools or start fresh. Copy-pasting full chat history is bloated and noisy. Moss solves this with **capsules**—distilled context snapshots that preserve what matters.

## What Moss Does

- **Hand off sessions:** Session A → Moss → Session B, across tools (Claude Code, Codex, etc.)
- **Orchestrate agents:** give parallel agents a shared context layer
- **Stay portable:** export/import for backup and cross-machine transfer

## Installation

### From Source

```bash
git clone https://github.com/hpungsan/moss.git
cd moss
go build ./cmd/moss
```

### Claude Code Integration

Add to `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "moss": {
      "command": "/path/to/moss"
    }
  }
}
```

Replace `/path/to/moss` with your actual binary path.

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

Capsules are size-bounded and linted to stay useful.

## Quick Start

### Store a Capsule

```
moss.store {
  "workspace": "myproject",
  "name": "auth",
  "capsule_text": "## Objective\nImplement JWT auth\n## Current status\nMiddleware done\n## Decisions\nUsing RS256\n## Next actions\nAdd refresh tokens\n## Key locations\nsrc/auth/\n## Open questions\nToken expiry policy?"
}
```

### Fetch by Name

```
moss.fetch { "workspace": "myproject", "name": "auth" }
```

### List All Capsules

```
moss.inventory {}
```

### Get Latest in Workspace

```
moss.latest { "workspace": "myproject", "include_text": true }
```

### Export for Backup

```
moss.export { "path": "/tmp/backup.jsonl" }
```

## MCP Tools

| Tool | Description |
|------|-------------|
| `moss.store` | Create a new capsule |
| `moss.fetch` | Retrieve by ID or name |
| `moss.fetch_many` | Batch fetch multiple |
| `moss.update` | Update existing capsule |
| `moss.delete` | Soft-delete (recoverable) |
| `moss.latest` | Most recent in workspace |
| `moss.list` | List capsules in workspace |
| `moss.inventory` | List all capsules globally |
| `moss.export` | JSONL backup |
| `moss.import` | JSONL restore |
| `moss.purge` | Permanent delete |

## Design Principles

- **Local-first:** SQLite at `~/.moss/moss.db`, no external services
- **MCP-first:** Native tool for AI agents, CLI for debugging
- **Explicit only:** No auto-save, no auto-load
- **Low-bloat:** Size limits + lint rules enforce quality

## Documentation

- [Overview & Use Cases](docs/moss/OVERVIEW.md)
- [Design Spec](docs/moss/v1.0/DESIGN.md)
- [Runbook](docs/moss/v1.0/RUNBOOK.md) — Installation, configuration, troubleshooting
- [Pairing Moss with Claude Code Tasks](docs/agents/TASKS.md)

## License

MIT
