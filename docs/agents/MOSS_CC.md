# Moss + Claude Code

How Moss Capsules integrate with Claude Code for context persistence across sessions, tasks, and swarms.

> **For full swarm orchestration reference:** See [`dev/swarm/SKILL.md`](../../dev/swarm/SKILL.md) for Teams, Teammates, Inboxes, Backends, and orchestration patterns.

## The Two Primitives

| Primitive | Purpose | Persistence |
|-----------|---------|-------------|
| **Tasks** | Coordination: what to do, dependencies, status | `~/.claude/tasks/` |
| **Capsules** | Context: why, decisions, key locations, open questions | `~/.moss/moss.db` |

Tasks tell agents **what** to do. Capsules tell agents **why** and **how**.

## Why Capsules?

Claude Code's Tasks evolved from Todos to support longer projects across multiple sessions and subagents. Tasks handle coordination (status, dependencies, blockers) but don't share:
- Decisions made during implementation
- Constraints discovered along the way
- Key file locations and commands
- Open questions that affect other tasks

**Capsules fill this gap.** They carry structured context that survives:
- Session restarts
- `/clear` commands
- Days between work sessions
- Subagent boundaries

## Capsules vs Inboxes

For swarm orchestration, Claude Code uses Inbox messages for inter-agent communication. Moss Capsules provide a better solution for **persistent structured context**.

| Aspect | Inbox | Capsule |
|--------|-------|---------|
| Structure | Unstructured text | 6 required sections |
| Persistence | Session-bound | Survives sessions, /clear, days |
| Quality | None | Lint + size limits |
| Queryability | Manual file reading | `list`, `inventory`, `fetch_many` |
| Scoping | None | `run_id`, `phase`, `role` |
| Batch retrieval | Read N files | Single `fetch_many` |

**When to use:**
- **Inbox**: Transient coordination ("done", "shutdown approved", "found bug")
- **Capsule**: Context handoffs (decisions, key locations, next actions)

---

## Moss in Swarm Patterns

The orchestration patterns in [`dev/swarm/SKILL.md`](../../dev/swarm/SKILL.md) use Inbox messages for worker coordination. Here's where Moss Capsules provide better context sharing:

### Pattern 1: Parallel Specialists

**SKILL.md approach:** Workers send findings to team-lead via Inbox.
```javascript
Teammate({ operation: "write", target_agent_id: "team-lead", value: "findings..." })
```

**Problem:** Findings are unstructured text, lost after session ends.

**With Moss:**
```
// Each specialist stores structured findings
security-reviewer:
  moss.store {
    name: "security-findings",
    run_id: "pr-review-123",
    role: "security-reviewer",
    capsule_text: "## Objective\nReview PR for security...\n## Current status\nFound 2 issues..."
  }

performance-reviewer:
  moss.store {
    name: "perf-findings",
    run_id: "pr-review-123",
    role: "performance-reviewer",
    capsule_text: "## Objective\nReview PR for performance..."
  }

// Leader gathers ALL findings in one call
team-lead:
  moss.fetch_many { run_id: "pr-review-123" }
  // Gets structured, queryable results from all specialists
```

**Benefits:**
- Structured findings (6 sections ensure completeness)
- Single `fetch_many` vs reading N inbox files
- Findings persist for later reference
- `run_id` scopes to this specific review

---

### Pattern 2: Pipeline (Sequential Dependencies)

**SKILL.md approach:** Each stage completes task, next stage starts fresh.

**Problem:** Context lost between stages. Stage 2 doesn't know Stage 1's decisions.

**With Moss:**
```
Stage 1 (Research):
  // Do research
  moss.store {
    name: "research",
    run_id: "feature-oauth",
    phase: "research",
    capsule_text: "## Decisions\n- Use Auth0 over Firebase because...\n## Key locations\n- Existing auth at src/auth/..."
  }
  TaskUpdate { taskId: "1", status: "completed" }

Stage 2 (Plan) - blocked by Stage 1:
  moss.fetch { name: "research" }
  // Now has: WHY Auth0, WHERE existing code lives
  // Create plan informed by research decisions
  moss.store {
    name: "plan",
    run_id: "feature-oauth",
    phase: "plan",
    capsule_text: "## Decisions\n- Based on research, implement OAuth in 3 phases..."
  }

Stage 3 (Implement) - blocked by Stage 2:
  moss.fetch_many { run_id: "feature-oauth" }
  // Gets BOTH research AND plan context
  // Implements with full decision history
```

**Benefits:**
- Each stage inherits previous decisions
- No re-discovery of constraints
- Full audit trail via `phase` filtering

---

### Pattern 3: Swarm (Self-Organizing)

**SKILL.md approach:** Workers race to claim tasks, send findings via Inbox.

**Problem:** Workers can't see what other workers discovered. Duplicate work possible.

**With Moss:**
```
// Workers share discoveries in real-time
worker-1 (reviewing user.rb):
  moss.store {
    name: "user-model-review",
    run_id: "codebase-review",
    capsule_text: "## Decisions\n- Found shared validation logic that affects payment.rb..."
  }

worker-2 (reviewing payment.rb):
  // Before starting, check what others found
  moss.list { run_id: "codebase-review" }
  // Sees worker-1 found shared validation logic
  moss.fetch { name: "user-model-review" }
  // Can now account for the dependency

// Leader synthesizes all findings
team-lead:
  moss.inventory { run_id: "codebase-review" }
  moss.fetch_many { ... }
```

**Benefits:**
- Workers can check related discoveries
- Avoid duplicate/conflicting findings
- Leader gets complete picture

---

### Pattern 4: Research + Implementation

**SKILL.md approach:** Research subagent returns result, implementation uses it.

**Problem:** Research result is ephemeral. Can't reference it later.

**With Moss:**
```
// Research phase
researcher:
  moss.store {
    name: "caching-research",
    phase: "research",
    capsule_text: "## Decisions\n- Redis over Memcached because...\n- Cache invalidation strategy: write-through\n## Key locations\n- Existing cache config at config/cache.yml"
  }

// Implementation phase (could be hours/days later)
implementer:
  moss.fetch { name: "caching-research" }
  // Has full research context even if researcher session is gone

// Review phase (could be different person/session)
reviewer:
  moss.fetch { name: "caching-research" }
  // Can verify implementation matches research recommendations
```

**Benefits:**
- Research survives session boundaries
- Multiple phases can reference same research
- Audit trail for "why was this approach chosen?"

---

### Pattern 5: Plan Approval Workflow

**SKILL.md approach:** Architect sends plan via Inbox, leader approves/rejects.

**Problem:** Plan is unstructured text. No verification it covers all requirements.

**With Moss:**
```
architect:
  moss.store {
    name: "oauth-plan",
    phase: "plan",
    role: "architect",
    capsule_text: "## Objective\nAdd OAuth2 authentication\n## Decisions\n- Use Auth0\n- Store tokens in httpOnly cookies\n## Next actions\n1. Install auth0 SDK\n2. Create callback route..."
  }
  // Capsule is validated: has all 6 required sections
  Teammate({ operation: "write", target_agent_id: "team-lead", value: "Plan ready: moss.fetch { name: 'oauth-plan' }" })

team-lead:
  moss.fetch { name: "oauth-plan" }
  // Structured plan with guaranteed sections
  // Can approve knowing it's complete
```

**Benefits:**
- Plans validated for completeness (lint rules)
- Structured format ensures nothing missed
- Plan persists for implementation reference

---

### Pattern 6: Coordinated Multi-File Refactoring

**SKILL.md approach:** Workers refactor different files, specs depend on both completing.

**Problem:** Spec writer doesn't know what decisions each refactorer made.

**With Moss:**
```
model-worker (refactoring User model):
  moss.store {
    name: "user-refactor",
    run_id: "auth-refactor",
    role: "model-worker",
    capsule_text: "## Decisions\n- Extracted to AuthenticatableUser concern\n- Changed method signature: authenticate(password) -> authenticate!(password)\n## Key locations\n- New concern at app/models/concerns/authenticatable_user.rb"
  }

controller-worker (refactoring Session controller):
  moss.store {
    name: "session-refactor",
    run_id: "auth-refactor",
    role: "controller-worker",
    capsule_text: "## Decisions\n- Now uses User.authenticate! (note the bang)\n## Open questions\n- Should we add rate limiting here?"
  }

spec-worker (blocked by both):
  moss.fetch_many { run_id: "auth-refactor" }
  // Knows:
  // - New concern location
  // - Method signature changed to authenticate!
  // - Open question about rate limiting
  // Can write accurate specs
```

**Benefits:**
- Spec writer has full decision context
- Knows about method signature changes
- Aware of open questions to address

---

## Use Cases (Non-Swarm)

### Simple: Session Continuity

No swarms needed—just persist context across sessions.

```
Session A:
  1. Do work
  2. moss.store { name: "auth-progress", ... }
  3. End session (or /clear)

Session B (hours/days later):
  1. moss.fetch { name: "auth-progress" }
  2. Continue with full context
```

### Intermediate: Tasks + Capsules

Track work with Tasks, persist context with Capsules.

```
1. TaskCreate { subject: "Implement auth" }
2. Do work, make decisions
3. moss.store { name: "auth-context", ... } → get fetch_key
4. TaskUpdate { taskId: "1", metadata: fetch_key }
5. Later: pick up task, read metadata, moss.fetch
```

---

## Orchestration Fields

Scope capsules to specific workflow runs:

| Field | Purpose | Example |
|-------|---------|---------|
| `run_id` | Identify workflow run | `"pr-review-123"` |
| `phase` | Workflow stage | `"research"`, `"implement"`, `"review"` |
| `role` | Agent role | `"security-reviewer"`, `"architect"` |

**Store with scope:**
```json
moss.store { "run_id": "pr-123", "phase": "security", "role": "reviewer", ... }
```

**Query by scope:**
```
moss.list { run_id: "pr-123" }                    // All capsules from this run
moss.list { run_id: "pr-123", phase: "research" } // Just research phase
moss.latest { run_id: "pr-123", role: "architect" } // Latest from architect
```

---

## Task-Capsule Linking

### Workspace Convention

Align Moss workspace with task list ID:

```bash
CLAUDE_CODE_TASK_LIST_ID=my-project claude
```

```
moss.store { workspace: "my-project", name: "auth", ... }
moss.list { workspace: "my-project" }
```

All capsules for a project live in one queryable namespace.

### fetch_key

Moss responses include `fetch_key` for direct Task metadata linking:

```json
{
  "id": "01J...ULID",
  "fetch_key": {
    "workspace": "my-project",
    "name": "auth"
  }
}
```

**Workflow:**
1. `moss.store` → get `fetch_key`
2. `TaskUpdate { metadata: fetch_key }`
3. Later: read `metadata`, `moss.fetch`

---

## Quick Reference

| Need | Use |
|------|-----|
| Track what to do | Tasks |
| Enforce execution order | `blockedBy` / `blocks` |
| Persist decisions and reasoning | Capsules |
| Context across sessions | `moss.store` → `moss.fetch` |
| Gather parallel results | `moss.fetch_many` |
| Scope to workflow run | `run_id` filter |
| Filter by workflow stage | `phase` filter |
| Filter by agent role | `role` filter |
| Link task to context | `fetch_key` in metadata |
| Transient agent messages | Inbox (see [`dev/swarm/SKILL.md`](../../dev/swarm/SKILL.md)) |
