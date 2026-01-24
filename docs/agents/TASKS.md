# Claude Code Tasks

Cross-session task coordination primitive for Claude Code.

## Overview

Tasks replace the old TodoWrite tool. They enable Claude to track and coordinate work across multiple sessions, subagents, and context windows.

## Storage

Tasks are persisted to the filesystem:

```
~/.claude/tasks/
└── <task-list-id>/
    ├── 1.json
    ├── 2.json
    └── ...
```

Each task is a JSON file:

```json
{
  "id": "1",
  "subject": "Implement auth flow",
  "description": "Add OAuth2 authentication with JWT tokens",
  "activeForm": "Implementing auth flow",
  "status": "pending",
  "blocks": ["3"],
  "blockedBy": ["2"]
}
```

## Task Fields

| Field | Description |
|-------|-------------|
| `id` | Unique identifier within the task list |
| `subject` | Brief title (imperative form: "Run tests") |
| `description` | Detailed requirements and context |
| `activeForm` | Present continuous form for spinner ("Running tests") |
| `status` | `pending` → `in_progress` → `completed` |
| `blocks` | Task IDs that cannot start until this completes |
| `blockedBy` | Task IDs that must complete before this can start |

## Tools

- **TaskCreate** - Create a new task
- **TaskGet** - Retrieve task by ID
- **TaskUpdate** - Update status, dependencies, or details
- **TaskList** - List all tasks in current task list

## Cross-Session Collaboration

Share a task list across sessions using the environment variable:

```bash
# Use existing task list
CLAUDE_CODE_TASK_LIST_ID=d4f552c2-1995-4c5d-bc4f-b99036b715d9 claude

# Use friendly name (creates new if doesn't exist)
CLAUDE_CODE_TASK_LIST_ID=moss-project claude

# Works with headless mode
CLAUDE_CODE_TASK_LIST_ID=moss-project claude -p "continue implementation"

# Works with Agent SDK
# Pass task list ID in environment when spawning agents
```

When one session updates a task, the change is broadcasted to all sessions working on the same task list.

## Relation to Moss Capsules

Tasks and Capsules are complementary:

| Primitive | Purpose |
|-----------|---------|
| **Tasks** | Coordination: what to do, dependencies, status |
| **Capsules** | Context: why, decisions made, key locations, open questions |

A session can pick up both a task list AND a capsule for full continuity.

### Workspace = Task List Convention

Align Moss workspace with Claude Code task list ID for natural scoping:

```bash
CLAUDE_CODE_TASK_LIST_ID=moss-project claude
```

Agent stores capsules to matching workspace:
```bash
moss.store { "workspace": "moss-project", "name": "auth", ... }
```

Query all capsules for a task list:
```bash
moss.list { "workspace": "moss-project" }
```

### Linking Tasks to Capsules

Tasks can reference capsules via metadata:

```json
{
  "id": "1",
  "subject": "Implement auth flow",
  "status": "in_progress",
  "metadata": {
    "moss_capsule": "auth-handoff",
    "moss_workspace": "default"
  }
}
```

Moss `moss.store` and `moss.fetch` responses include a `task_link` field - a ready-to-use blob:

```json
{
  "id": "01J...ULID",
  "task_link": {
    "moss_capsule": "auth-handoff",
    "moss_workspace": "default"
  }
}
```

Workflow:
1. Store capsule → get `task_link` in response
2. Create/update task with `metadata` = `task_link`
3. Later session picks up task, reads `metadata.moss_capsule`
4. Calls `moss.fetch { workspace: "...", name: "..." }`
5. Has full coordination (task) + context (capsule)

## Source

Based on Claude Code team announcement (Jan 2025).
