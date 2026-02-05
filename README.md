# Moss

Local context store for AI session handoffs, multi-agent orchestration, and cross-tool context sharing.

## The Problem

AI coding sessions lose context when you switch tools or start fresh. Copy-pasting full chat history is bloated and noisy. Moss solves this with **types**—structured context objects optimized for different consumers.

## Types

| Type | Consumer | Format |
|-----------|----------|--------|
| **Capsule** | LLMs | Markdown (6 sections) |

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

See [Claude Code Integration](docs/integrations/claude-code.md) for full setup including skills and subagents.

## Capsule

A distilled context snapshot for LLM consumption. Not a chat log—a structured summary with 6 required sections: Objective, Status, Decisions, Next actions, Key locations, Open questions.

```
capsule_store { "workspace": "myproject", "name": "auth", "capsule_text": "## Objective\n..." }
capsule_fetch { "workspace": "myproject", "name": "auth" }
```

| Tool | Description |
|------|-------------|
| `capsule_store` | Create a new capsule |
| `capsule_fetch` | Retrieve by ID or name |
| `capsule_fetch_many` | Batch fetch multiple |
| `capsule_update` | Update existing capsule |
| `capsule_append` | Append to a section |
| `capsule_delete` | Soft-delete (recoverable) |
| `capsule_latest` | Most recent in workspace |
| `capsule_list` | List capsules in workspace |
| `capsule_inventory` | List all capsules globally |
| `capsule_search` | Full-text search |
| `capsule_compose` | Assemble multiple capsules |
| `capsule_export` | JSONL backup |
| `capsule_import` | JSONL restore |
| `capsule_purge` | Permanent delete |
| `capsule_bulk_delete` | Soft-delete by filter |
| `capsule_bulk_update` | Update metadata by filter |

**Customize tools:** Disable tools you don't need via config. See [Tool Filtering](docs/SETUP.md#tool-filtering).

See [Capsule Runbook](docs/capsule/RUNBOOK.md) for full usage, addressing modes, and error handling.

## CLI

The CLI mirrors MCP capsule operations for debugging and scripting. Note: orchestration fields (`run_id`, `phase`, `role`) are MCP-only.

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
- **Type-specific:** Each type optimized for its consumer (LLM vs code)

## Documentation

- [Overview & Use Cases](docs/README.md)
- [Moss Setup](docs/SETUP.md) — Installation and paths
- [Capsule Design Spec](docs/capsule/DESIGN.md)
- [Capsule Runbook](docs/capsule/RUNBOOK.md) — Operations, configuration, troubleshooting
- [Capsule Backlog](docs/capsule/BACKLOG.md) — Future capsule features

## License

MIT
