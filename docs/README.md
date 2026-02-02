# Moss Documentation

Reference documentation for Moss primitives and integrations.

## Overview

Moss is a local store for **primitives**: typed context objects optimized for different consumers.

The primary primitive today is the **capsule**: a distilled context snapshot for AI session handoffs and multi-agent workflows. A capsule is **not** a chat log — it’s a structured summary that preserves:

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

## Primitives

Moss stores typed **primitives**—context objects optimized for different consumers.

| Primitive | Consumer | Format | Docs |
|-----------|----------|--------|------|
| **Capsule** | LLMs | Markdown (6 sections) | [capsule/](capsule/) |
| **Artifact** | Code/orchestration | JSON (structured) | [artifact/](artifact/) |

### Capsule

Distilled context snapshots for LLM consumption. Markdown-based with 6 required sections (Objective, Status, Decisions, Next actions, Key locations, Open questions). Validated on store.

- [DESIGN.md](capsule/DESIGN.md) — API spec, tool reference, error codes
- [RUNBOOK.md](capsule/RUNBOOK.md) — Operations guide, configuration, troubleshooting
- [BACKLOG.md](capsule/BACKLOG.md) — Future features

### Artifact

Structured JSON data for code and orchestration. Schema-validated, optimized for programmatic access. Enables deterministic fan-in (sort, filter, dedupe by fields).

- [DESIGN.md](artifact/DESIGN.md) — API spec
- [BACKLOG.md](artifact/BACKLOG.md) — Future features

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
