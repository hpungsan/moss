# Moss Documentation

Reference documentation for Moss types and integrations.

## Overview

Moss is a local store for **types**: structured context objects optimized for different consumers.

The primary type today is the **capsule**: a distilled context snapshot for AI session handoffs and multi-agent workflows. A capsule is **not** a chat log — it’s a structured summary that preserves:

- what you’re doing (objective)
- where things stand (status)
- what choices were made (decisions)
- what’s next (actions)
- where relevant code/docs live (key locations)
- unresolved risks/questions

### When Moss Helps Most

- **Session handoff:** stop, store, later fetch and continue (across tools like Claude Code/Codex)
- **Parallel work (fan-out/fan-in):** agents work in isolation; orchestrator fetches many and synthesizes
- **Pipeline workflows:** chain checkpoints across phases (research → plan → implement → review)
- **Re-review / audit:** find prior decisions and verify what was addressed before changing direction

### Core Workflow

1. **Store** at stable checkpoints
2. **Browse** (`list` / `inventory`) to find candidates without pulling full text
3. **Fetch** the exact capsule(s) you need
4. **Search** to find prior art by content
5. **Export/import** for backup and portability

## Types

Moss stores **types**—structured context objects optimized for different consumers.

| Type | Consumer | Format | Docs |
|-----------|----------|--------|------|
| **Capsule** | LLMs | Markdown (6 sections) | [capsule/](capsule/) |

### Capsule

Distilled context snapshots for LLM consumption. Markdown-based with 6 required sections (Objective, Status, Decisions, Next actions, Key locations, Open questions). Validated on store.

- [DESIGN.md](capsule/DESIGN.md) — API spec, tool reference, error codes
- [RUNBOOK.md](capsule/RUNBOOK.md) — Operations guide, configuration, troubleshooting
- [BACKLOG.md](capsule/BACKLOG.md) — Future features

## Integration

| Doc | Purpose |
|-----|---------|
| [SETUP.md](SETUP.md) | Installation and paths |
| [integrations/claude-code.md](integrations/claude-code.md) | Claude Code MCP setup, skills, subagents |
| [agents/MOSS_CC.md](agents/MOSS_CC.md) | Capsule patterns for sessions, tasks, swarms |
| [agents/CODEMAP.md](agents/CODEMAP.md) | File-level codebase map |

## Quick Links

- [Project README](../README.md) — Installation, quick start
- [Example capsule](../examples/capsule.md) — 6-section format reference
