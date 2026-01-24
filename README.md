# Moss

Local context capsule store for portable AI handoffs, multi-agent orchestration, and cross-tool context sharing.

## The Problem

AI coding sessions lose context when you switch tools or start fresh. Copy-pasting full chat history is bloated and noisy. Moss solves this with **capsules**—distilled context snapshots that preserve what matters.

## What Moss Does

- **Hand off sessions:** Session A → Moss → Session B, across tools (Claude Code, Codex, etc.)
- **Orchestrate agents:** give parallel agents a shared context layer
- **Stay portable:** export/import for backup and cross-machine transfer

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

### MCP Tools

```bash
# Store a capsule
moss.store { "workspace": "myproject", "name": "auth", "capsule_text": "..." }

# Fetch by name
moss.fetch { "workspace": "myproject", "name": "auth" }

# List capsules (summaries only)
moss.list { "workspace": "myproject" }

# See everything
moss.inventory {}
```

### CLI (debug)

```bash
moss store --name=auth < capsule.md
moss fetch --workspace=myproject --name=auth
moss list --workspace=myproject
moss export --workspace=myproject > backup.jsonl
```

## Design Principles

- **Local-first:** SQLite at `~/.moss/moss.db`, no external services
- **MCP-first:** Native tool for AI agents, CLI for debugging
- **Explicit only:** No auto-save, no auto-load
- **Low-bloat:** Size limits + lint rules enforce quality

## Examples

- [Pairing Moss with Claude Code Tasks](docs/agents/TASKS.md)

## License

MIT
