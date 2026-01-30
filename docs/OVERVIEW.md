# Moss Overview

Moss is a local store for **capsules**: distilled context snapshots for AI session handoffs and multi-agent workflows.

A capsule is **not** a chat log. It’s a structured summary that preserves:
- what you’re doing (objective)
- where things stand (status)
- what choices were made (decisions)
- what’s next (actions)
- where the relevant code/docs live (key locations)
- unresolved risks/questions

## When Moss Helps Most

- **Session handoff**: stop, store, later fetch and continue (across tools like Claude Code/Codex).
- **Parallel work (fan-out/fan-in)**: multiple agents work in isolation; each stores results; orchestrator fetches many and synthesizes.
- **Pipeline workflows**: chain capsules across phases (research → plan → implement → review) without re-discovering context.
- **Re-review / audit**: find prior decisions and verify what was addressed before changing direction.

## Core Workflow (Typical)

1. **Store** a capsule when you reach a stable checkpoint.
2. **Browse** (`list` / `inventory`) to find candidates without pulling full text.
3. **Fetch** the exact capsule(s) you need.
4. **Search** to find prior art by content.
5. **Export/import** for backup and portability.

## Tools at a Glance

High-signal mental model:
- `store` / `update` / `delete` change state
- `fetch` loads full capsule text (unless `include_text:false`)
- `list` / `inventory` are summaries only
- `search` finds by content (FTS5) and returns ranked snippets
- `fetch_many` is for fan-in; `compose` is for bundling capsules into one artifact

## Multi-Agent Metadata (Optional)

Use these when you want workflow scoping:
- `run_id`: group capsules for one run/work item (e.g. a PR)
- `phase`: stage in the workflow (research/implement/review)
- `role`: who produced it (architect/security-reviewer/etc.)

This enables patterns like “latest review capsule for run X” or “inventory all security reviews”.

## Next Docs

- [README.md](../README.md) — product intro + quick start
- [DESIGN.md](DESIGN.md) — full API/spec and precise behaviors
- [RUNBOOK.md](RUNBOOK.md) — install/config/run/troubleshoot
- [agents/MOSS_CC.md](agents/MOSS_CC.md) — integration patterns for agent workflows
- [examples/capsule.md](../examples/capsule.md) — capsule structure example
