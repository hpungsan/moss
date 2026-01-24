# Claude Code Tasks + Moss Capsules

How Moss integrates with Claude Code's task system for agent handoffs, multi-agent orchestration, and swarms.

## The Two Primitives

| Primitive | Purpose | Persistence |
|-----------|---------|-------------|
| **Tasks** | Coordination: what to do, dependencies, status | `~/.claude/tasks/<id>/` |
| **Capsules** | Context: why, decisions, key locations, open questions | `~/.moss/moss.db` |

Tasks tell agents **what** to do. Capsules tell agents **why** and **how**.

## Why Both?

Claude Code's task system enables agent swarms—multiple sub-agents working in parallel with isolated 200k token context windows. Each sub-agent focuses on one task without polluting others' context.

But isolation creates a problem: **sub-agents can't share knowledge**.

Tasks only share status (`pending` → `in_progress` → `completed`). They don't share:
- Decisions made during implementation
- Constraints discovered along the way
- Key file locations and commands
- Open questions that affect other tasks

**Moss Capsules fill this gap.** They carry structured context between agents.

## Swarm Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     ORCHESTRATOR                            │
│  - Creates task graph with dependencies                     │
│  - Stores base context capsule                              │
│  - Spawns sub-agents for unblocked tasks                    │
│  - Gathers results via fetch_many                           │
└─────────────────────────────────────────────────────────────┘
        │                    │                    │
        ▼                    ▼                    ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│  SUB-AGENT 1  │  │  SUB-AGENT 2  │  │  SUB-AGENT 3  │
│  (200k ctx)   │  │  (200k ctx)   │  │  (200k ctx)   │
│               │  │               │  │               │
│ fetch capsule │  │ fetch capsule │  │ fetch capsule │
│ do work       │  │ do work       │  │ do work       │
│ store result  │  │ store result  │  │ store result  │
└───────────────┘  └───────────────┘  └───────────────┘
```

### Coordination Layer (Tasks)

- Dependency graph enforces execution order
- `blockedBy` prevents premature starts
- Parallel execution of independent tasks
- Status broadcasts across sessions

### Context Layer (Capsules)

- Structured handoff format (6 required sections)
- Size-bounded to prevent bloat
- Survives /clear, session restarts, days between work
- Queryable by workspace, name, or ID

## Patterns

### Pattern 1: Simple Handoff

Single agent hands off to another (different session, tool, or time).

```
Session A:
  1. Do work
  2. moss.store { name: "auth-progress", content: "..." }
  3. End session

Session B:
  1. moss.fetch { name: "auth-progress" }
  2. Continue work with full context
```

### Pattern 2: Fan-Out / Fan-In

Orchestrator spawns parallel workers, gathers results.

```
Orchestrator:
  1. moss.store { name: "base-context", content: "..." }
  2. Create tasks: auth, database, api (no dependencies)
  3. Spawn 3 sub-agents

Sub-agents (parallel):
  1. moss.fetch { name: "base-context" }
  2. Do assigned work
  3. moss.store { name: "auth-result" | "db-result" | "api-result" }
  4. Mark task completed

Orchestrator:
  1. moss.fetch_many { names: ["auth-result", "db-result", "api-result"] }
  2. Synthesize results
  3. Continue or spawn next wave
```

### Pattern 3: Pipeline with Context Chain

Sequential tasks where each builds on previous context.

```
Task 1 (Research):
  1. Investigate codebase
  2. moss.store { name: "research", content: "findings + recommendations" }
  3. Complete task

Task 2 (blocked by Task 1):
  1. moss.fetch { name: "research" }
  2. Implement based on findings
  3. moss.store { name: "implementation", content: "what was built + decisions" }
  4. Complete task

Task 3 (blocked by Task 2):
  1. moss.fetch { name: "implementation" }
  2. Write tests based on implementation details
  3. Complete task
```

### Pattern 4: Hierarchical Swarms

Sub-agents spawn their own sub-agents (nested orchestration).

```
Level 0 (Main):
  moss.store { workspace: "project", name: "base" }
  Create tasks: auth-subsystem, db-subsystem
  Spawn sub-agents

Level 1 (Auth Sub-Agent):
  moss.fetch { workspace: "project", name: "base" }
  moss.store { workspace: "project", name: "auth-base" }
  Create sub-tasks: login, logout, sessions, tokens
  Spawn own sub-agents

Level 2 (Login Sub-Agent):
  moss.fetch { workspace: "project", name: "auth-base" }
  Do focused work
  moss.store { workspace: "project", name: "auth-login-result" }
```

## Task-Capsule Linking

### Workspace Convention

Align Moss workspace with task list ID:

```bash
CLAUDE_CODE_TASK_LIST_ID=my-project claude
```

```
moss.store { "workspace": "my-project", "name": "auth", ... }
moss.list { "workspace": "my-project" }
```

All capsules for a swarm live in one queryable namespace.

### Metadata Linking

Tasks can reference their capsule via metadata:

```json
{
  "id": "1",
  "subject": "Implement auth flow",
  "status": "in_progress",
  "metadata": {
    "moss_capsule": "auth-handoff",
    "moss_workspace": "my-project"
  }
}
```

Moss responses include a ready-to-use `task_link`:

```json
{
  "id": "01J...ULID",
  "task_link": {
    "moss_capsule": "auth-handoff",
    "moss_workspace": "my-project"
  }
}
```

Workflow:
1. `moss.store` → get `task_link` in response
2. `TaskUpdate` with `metadata` = `task_link`
3. Later agent picks up task, reads `metadata.moss_capsule`
4. `moss.fetch` → has full context

## Task System Reference

### Storage

```
~/.claude/tasks/
└── <task-list-id>/
    ├── 1.json
    ├── 2.json
    └── ...
```

### Task Fields

| Field | Description |
|-------|-------------|
| `id` | Unique identifier within task list |
| `subject` | Brief title (imperative: "Run tests") |
| `description` | Detailed requirements |
| `activeForm` | Present continuous for spinner ("Running tests") |
| `status` | `pending` → `in_progress` → `completed` |
| `blocks` | Task IDs that cannot start until this completes |
| `blockedBy` | Task IDs that must complete before this can start |
| `metadata` | Arbitrary key-value (use for capsule links) |

### Tools

- **TaskCreate** - Create new task
- **TaskGet** - Retrieve by ID
- **TaskUpdate** - Update status, dependencies, metadata
- **TaskList** - List all tasks

### Cross-Session Setup

```bash
# Named task list (persistent)
CLAUDE_CODE_TASK_LIST_ID=my-project claude

# Headless mode
CLAUDE_CODE_TASK_LIST_ID=my-project claude -p "continue implementation"

# Agent SDK - pass in environment when spawning
```

Updates broadcast to all sessions on the same task list.

## Context Persistence vs Reanchoring

Before tasks, "Ralph"-style reanchoring was common: re-explaining context after /clear or session restart. Tasks externalize coordination state, but not semantic context.

Capsules externalize semantic context:
- **Objective** - what you're trying to accomplish
- **Current status** - where things stand
- **Decisions/constraints** - choices made and why
- **Next actions** - what to do next
- **Key locations** - files, URLs, commands
- **Open questions** - unresolved issues

Together, Tasks + Capsules replace ad-hoc reanchoring with structured persistence.

## Quick Reference

| Need | Use |
|------|-----|
| Track what to do | Tasks |
| Enforce execution order | `blockedBy` / `blocks` |
| Share status across sessions | Task list ID |
| Preserve decisions and reasoning | Capsules |
| Hand off context to sub-agent | `moss.store` → `moss.fetch` |
| Gather results from parallel agents | `moss.fetch_many` |
| Query all context for a swarm | `moss.list { workspace }` |
| Link task to its context | `metadata.moss_capsule` |
