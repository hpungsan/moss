# Moss

Local context capsule store for portable AI session handoffs.

## What is Moss?

Moss stores **capsules**—distilled, size-bounded context snapshots—for:
- **Session handoffs:** seamlessly continue work across AI tools (Claude Code, Codex, etc.)
- **Multi-agent orchestration:** share context between parallel agents

**Key features:**
- Store/fetch/update/delete capsules with human-friendly names
- Batch fetch (`fetch_many`) for multi-agent workflows
- Export/import (JSONL) for backup and portability
- Quality guardrails: lint rules enforce useful capsules, size limits prevent bloat
- Soft delete with purge for safety
- Local-first: SQLite storage at `~/.moss/moss.db`
- MCP-first interface with CLI for debugging

## Capsule Structure

Every capsule must include:
1. Objective
2. Current status
3. Decisions/constraints
4. Next actions
5. Key locations
6. Open questions/risks

## Quick Start

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
moss list --workspace=myproject
moss fetch --workspace=myproject --name=auth
moss inventory
moss export --workspace=myproject > backup.jsonl
```

## License

MIT
