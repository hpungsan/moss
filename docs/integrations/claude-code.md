# Moss + Claude Code Setup

How to use Moss with Claude Code sessions and subagents.

## Prerequisites

1. Moss is built and available (see [SETUP](../SETUP.md))
2. Claude Code is installed

## MCP Configuration

Add Moss to your project's `.mcp.json` or global `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "moss": {
      "command": "/path/to/moss"
    }
  }
}
```

Replace `/path/to/moss` with your actual binary path (e.g., `$GOPATH/bin/moss` if installed via `go install`).

## Main Session

The main Claude Code session has access to all MCP tools automatically via the config above. No extra setup needed — just use Moss tools directly:

```
capsule_store { name: "my-context", capsule_text: "..." }
capsule_fetch { name: "my-context" }
capsule_list {}
```

## Subagents

Claude Code subagents (spawned via the Task tool) **don't have MCP access by default**. To give a subagent Moss access, add MCP tools and the `moss-capsule` skill to its agent definition frontmatter.

### Agent Frontmatter

```yaml
---
name: my-agent
description: What this agent does
tools: Bash, Read, Glob, Grep, mcp__moss__capsule_store
skills:
  - moss-capsule
---
```

**`tools`** — Add the specific `mcp__moss__capsule_*` tools the agent needs. Only grant what's necessary.

**`skills`** — Adding `moss-capsule` gives the agent capsule skill instructions (capsule format, addressing modes, error handling). Requires the skill files in `.claude/skills/moss-capsule/`.

### Available MCP Tools

| Tool | Purpose |
|------|---------|
| `mcp__moss__capsule_store` | Store or replace a capsule |
| `mcp__moss__capsule_fetch` | Fetch a single capsule by ID or name |
| `mcp__moss__capsule_fetch_many` | Batch fetch multiple capsules |
| `mcp__moss__capsule_update` | Update an existing capsule |
| `mcp__moss__capsule_delete` | Soft-delete a capsule |
| `mcp__moss__capsule_list` | List capsules in a workspace |
| `mcp__moss__capsule_inventory` | List capsules across all workspaces |
| `mcp__moss__capsule_latest` | Get the most recently updated capsule |
| `mcp__moss__capsule_compose` | Assemble multiple capsules into a bundle |
| `mcp__moss__capsule_export` | Export capsules to JSONL |
| `mcp__moss__capsule_import` | Import capsules from JSONL |
| `mcp__moss__capsule_purge` | Permanently delete soft-deleted capsules |

### Example: Reviewer That Stores Findings

```yaml
---
name: security-reviewer
description: Security review for vulnerabilities
tools: Bash, Read, Glob, Grep, mcp__moss__capsule_store
skills:
  - moss-capsule
model: opus
---

You are Security Reviewer. After completing your review, store
your findings as a Moss capsule using the capsule_store tool.
```

This agent can read code with Bash/Read/Glob/Grep and persist structured findings via `mcp__moss__capsule_store`. The `moss-capsule` skill teaches it capsule format and conventions.

### Example: Agent That Reads Prior Context

```yaml
---
name: implementer
description: Implements features using prior research context
tools: Bash, Read, Glob, Grep, Edit, Write, mcp__moss__capsule_fetch, mcp__moss__capsule_list
skills:
  - moss-capsule
---

You are an implementer. Before starting work, check for existing
context capsules using capsule_list and capsule_fetch.
```

### Common Patterns

**Store only** — Most subagents just need `mcp__moss__capsule_store` to save their output.

**Read only** — Agents that consume context need `mcp__moss__capsule_fetch` and optionally `mcp__moss__capsule_list` for discovery.

**Read + write** — Agents that both consume and produce context need both capsule_fetch and capsule_store tools.

## Moss Capsule Skill

The `moss-capsule` skill teaches Claude Code how to use Moss tools correctly. It covers capsule format (6 required sections), addressing modes, output bloat rules, and error handling.

To set up the skill, create `.claude/skills/moss-capsule/` in your project with a `SKILL.md` file. See the [Moss skill repository](https://github.com/hpungsan/moss) for a reference implementation, or write your own based on the [DESIGN spec](../capsule/DESIGN.md).

## Further Reading

- [MOSS_CC.md](../agents/MOSS_CC.md) — Capsule patterns for sessions, tasks, and swarms
- [SETUP](../SETUP.md) — Installation and paths
- [DESIGN](../capsule/DESIGN.md) — Capsule API spec
