# Moss Overview

## Why Moss?

AI coding sessions (Claude Code, Codex, etc.) lose context when you switch tools or start fresh. Copy-pasting full chat history is bloated and noisy. Moss solves this with **capsules**—distilled context snapshots that preserve what matters.

## Core Concept: Capsules

A capsule is not a chat log. It's a **structured summary** of current state:

| Section | Purpose |
|---------|---------|
| Objective | What you're trying to accomplish |
| Current status | Where things stand now |
| Decisions/constraints | Choices made and why |
| Next actions | What to do next |
| Key locations | Files, URLs, commands |
| Open questions | Unresolved issues |

Capsules are size-bounded and linted to ensure they stay useful.

## Use Cases

### Session Handoffs
```
Session A (Claude Code) → store capsule → Session B (Codex) fetches → continues work
```

### Multi-Agent Orchestration & Swarms

Sub-agents spawned via Task tool are **isolated**—each gets its own 200k context window. Tasks share status (`pending` → `done`), but not decisions, findings, or reasoning.

**Without Moss vs With Moss:**

| Pattern | Without Moss | With Moss |
|---------|--------------|-----------|
| **Fan-out** | Sub-agents start cold, duplicate discovery | `fetch` base capsule → shared starting point |
| **Fan-in** | Orchestrator only knows "done" | `fetch_many` results → synthesize findings |
| **Pipeline** | Each stage re-discovers prior work | Chain capsules: research → impl → test |
| **Re-review** | Start fresh, may miss what was addressed | `fetch` prior review → verify fixes |

**Swarm Architecture:**

```
┌─────────────────────────────────────────────────────────────┐
│                     ORCHESTRATOR                            │
│  - Creates task graph with dependencies                     │
│  - moss.store base context capsule                          │
│  - Spawns sub-agents for unblocked tasks                    │
│  - moss.fetch_many to gather results                        │
└─────────────────────────────────────────────────────────────┘
        │                    │                    │
        ▼                    ▼                    ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│  SUB-AGENT 1  │  │  SUB-AGENT 2  │  │  SUB-AGENT 3  │
│  (200k ctx)   │  │  (200k ctx)   │  │  (200k ctx)   │
│               │  │               │  │               │
│ moss.fetch    │  │ moss.fetch    │  │ moss.fetch    │
│ do work       │  │ do work       │  │ do work       │
│ moss.store    │  │ moss.store    │  │ moss.store    │
└───────────────┘  └───────────────┘  └───────────────┘
```

Moss bridges the isolation gap—sub-agents share structured context without polluting each other's windows.

### Integration with Claude Code Tasks

Claude Code Tasks handle **coordination** (what to do, dependencies, status). Moss Capsules handle **context** (why, decisions, key locations). They're complementary—Tasks are ephemeral work items, Capsules are durable knowledge artifacts.

See [TASKS.md](../../docs/agents/TASKS.md) for details on integration patterns.

## Design Principles

- **Local-first:** SQLite at `~/.moss/moss.db`, no external services
- **MCP-first:** Native tool for AI agents, CLI for debugging
- **Explicit only:** No auto-save, no auto-load
- **Low-bloat:** Size limits + lint rules enforce quality
- **Human-friendly:** Name capsules like `auth` or `pr-123-base`

## How It Works

MCP tools (one per operation):

| Tool | Purpose |
|------|---------|
| `moss.store` | Create capsule (with upsert mode) |
| `moss.fetch` | Load capsule by id or name |
| `moss.fetch_many` | Batch load multiple capsules |
| `moss.update` | Update capsule content |
| `moss.delete` | Soft delete (recoverable) |
| `moss.list` | List capsules in workspace |
| `moss.inventory` | List all capsules globally |
| `moss.latest` | Get most recent capsule |
| `moss.export` | JSONL backup |
| `moss.import` | JSONL restore |
| `moss.purge` | Permanently delete soft-deleted |

See [v1.0/DESIGN.md](v1.0/DESIGN.md) for full specification.
