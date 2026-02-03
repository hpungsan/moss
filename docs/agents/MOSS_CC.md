# Moss + Claude Code

How Moss Capsules integrate with Claude Code for context persistence across sessions, tasks, and swarms.

> **For full swarm orchestration reference:** See [`dev/vault/skillVault/swarm/SKILL.md`](../../dev/vault/skillVault/swarm/SKILL.md) for Teams, Teammates, Inboxes, Backends, and orchestration patterns.

## The Two Building Blocks

| Concept | Purpose | Persistence |
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
| Queryability | Manual file reading | `capsule_list`, `capsule_inventory`, `capsule_fetch_many` |
| Scoping | None | `run_id`, `phase`, `role` |
| Batch retrieval | Read N files | `capsule_list` → `capsule_fetch_many` |

**When to use:**
- **Inbox**: Transient coordination ("done", "shutdown approved", "found bug")
- **Capsule**: Context handoffs (decisions, key locations, next actions)

---

## Moss in Swarm Patterns

The orchestration patterns in [`dev/vault/skillVault/swarm/SKILL.md`](../../dev/vault/skillVault/swarm/SKILL.md) use Inbox messages for worker coordination. Here's where Moss Capsules provide better context sharing:

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
  capsule_store {
    name: "security-findings",
    run_id: "pr-review-123",
    role: "security-reviewer",
    capsule_text: "## Objective\nReview PR for security...\n## Current status\nFound 2 issues..."
  }

performance-reviewer:
  capsule_store {
    name: "perf-findings",
    run_id: "pr-review-123",
    role: "performance-reviewer",
    capsule_text: "## Objective\nReview PR for performance..."
  }

// Leader gathers ALL findings
team-lead:
  // Step 1: See what's available (no text, fast)
  capsule_list { run_id: "pr-review-123" }
  // Returns: [{ name: "security-findings", ... }, { name: "perf-findings", ... }]

  // Step 2: Fetch the ones you need
  capsule_fetch_many { items: [
    { workspace: "default", name: "security-findings" },
    { workspace: "default", name: "perf-findings" }
  ]}
  // Gets structured, queryable results from all specialists
```

**Benefits:**
- Structured findings (6 sections ensure completeness)
- Browse first, then fetch (prevents accidental context bloat)
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
  capsule_store {
    name: "research",
    run_id: "feature-oauth",
    phase: "research",
    capsule_text: "## Decisions\n- Use Auth0 over Firebase because...\n## Key locations\n- Existing auth at src/auth/..."
  }
  TaskUpdate { taskId: "1", status: "completed" }

Stage 2 (Plan) - blocked by Stage 1:
  capsule_fetch { name: "research" }
  // Now has: WHY Auth0, WHERE existing code lives
  // Create plan informed by research decisions
  capsule_store {
    name: "plan",
    run_id: "feature-oauth",
    phase: "plan",
    capsule_text: "## Decisions\n- Based on research, implement OAuth in 3 phases..."
  }

Stage 3 (Implement) - blocked by Stage 2:
  // Get all context from this workflow
  capsule_list { run_id: "feature-oauth" }
  capsule_fetch_many { items: [
    { workspace: "default", name: "research" },
    { workspace: "default", name: "plan" }
  ]}
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
  capsule_store {
    name: "user-model-review",
    run_id: "codebase-review",
    capsule_text: "## Decisions\n- Found shared validation logic that affects payment.rb..."
  }

worker-2 (reviewing payment.rb):
  // Before starting, check what others found
  capsule_list { run_id: "codebase-review" }
  // Sees worker-1 found shared validation logic
  capsule_fetch { name: "user-model-review" }
  // Can now account for the dependency

// Leader synthesizes all findings
team-lead:
  capsules = capsule_list { run_id: "codebase-review" }
  capsule_fetch_many { items: capsules.map(c => { workspace: c.workspace, name: c.name }) }
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
  capsule_store {
    name: "caching-research",
    phase: "research",
    capsule_text: "## Decisions\n- Redis over Memcached because...\n- Cache invalidation strategy: write-through\n## Key locations\n- Existing cache config at config/cache.yml"
  }

// Implementation phase (could be hours/days later)
implementer:
  capsule_fetch { name: "caching-research" }
  // Has full research context even if researcher session is gone

// Review phase (could be different person/session)
reviewer:
  capsule_fetch { name: "caching-research" }
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
  capsule_store {
    name: "oauth-plan",
    phase: "plan",
    role: "architect",
    capsule_text: "## Objective\nAdd OAuth2 authentication\n## Decisions\n- Use Auth0\n- Store tokens in httpOnly cookies\n## Next actions\n1. Install auth0 SDK\n2. Create callback route..."
  }
  // Capsule is validated: has all 6 required sections
  Teammate({ operation: "write", target_agent_id: "team-lead", value: "Plan ready: capsule_fetch { name: 'oauth-plan' }" })

team-lead:
  capsule_fetch { name: "oauth-plan" }
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
  capsule_store {
    name: "user-refactor",
    run_id: "auth-refactor",
    role: "model-worker",
    capsule_text: "## Decisions\n- Extracted to AuthenticatableUser concern\n- Changed method signature: authenticate(password) -> authenticate!(password)\n## Key locations\n- New concern at app/models/concerns/authenticatable_user.rb"
  }

controller-worker (refactoring Session controller):
  capsule_store {
    name: "session-refactor",
    run_id: "auth-refactor",
    role: "controller-worker",
    capsule_text: "## Decisions\n- Now uses User.authenticate! (note the bang)\n## Open questions\n- Should we add rate limiting here?"
  }

spec-worker (blocked by both):
  // Gather all refactoring context
  capsule_list { run_id: "auth-refactor" }
  capsule_fetch_many { items: [
    { workspace: "default", name: "user-refactor" },
    { workspace: "default", name: "session-refactor" }
  ]}
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
  2. capsule_store { name: "auth-progress", ... }
  3. End session (or /clear)

Session B (hours/days later):
  1. capsule_fetch { name: "auth-progress" }
  2. Continue with full context
```

### Intermediate: Tasks + Capsules

Track work with Tasks, persist context with Capsules.

```
1. TaskCreate { subject: "Implement auth" }
2. Do work, make decisions
3. capsule_store { name: "auth-context", ... } → get fetch_key
4. TaskUpdate { taskId: "1", metadata: fetch_key }
5. Later: pick up task, read metadata, fetch
```

### Advanced: Cross-Run Knowledge

Query capsules from *prior* workflow runs to inform new ones. Unlike `run_id` scoping (within a run), this leverages accumulated knowledge across runs.

```
// New OAuth implementation starting
// Option 1: Filter by metadata
capsule_inventory { phase: "research", tag: "oauth" }
// Returns capsules from ANY prior run tagged "oauth" in research phase

// Option 2: Full-text search (finds content, not just metadata)
capsule_search { query: "OAuth provider comparison", phase: "research" }
// Returns ranked results with HTML-safe snippets (<b> highlights, user content escaped):
// [{ name: "provider-comparison", snippet: "...<b>OAuth</b> <b>provider</b>...", ... }]

// Fetch relevant prior research
capsule_fetch { workspace: "feature-oauth-v1", name: "research" }
// Get decisions from 6 months ago:
// "## Decisions
//  - Auth0 rejected due to cost
//  - Google OAuth chosen for user-facing
//  - Service accounts use JWT..."

// Now implement with institutional knowledge
// Don't repeat the Auth0 evaluation—it's already documented
```

**Use cases:**
- **Onboarding**: New agent/session queries prior art before starting
- **Avoiding re-work**: Check if similar research exists
- **Pattern mining**: `capsule_inventory { phase: "security" }` → see all security reviews
- **Postmortems**: `capsule_inventory { tag: "incident" }` → find related incidents

**Tagging for discoverability:**
```
capsule_store {
  name: "oauth-research",
  run_id: "pr-123",
  phase: "research",
  tags: ["oauth", "auth", "google"],  // Searchable across runs
  capsule_text: "..."
}
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
capsule_store { "run_id": "pr-123", "phase": "security", "role": "reviewer", ... }
```

**Query by scope:**
```
capsule_list { run_id: "pr-123" }                    // All capsules from this run
capsule_list { run_id: "pr-123", phase: "research" } // Just research phase
capsule_latest { run_id: "pr-123", role: "architect" } // Latest from architect
capsule_search { query: "security", run_id: "pr-123" } // Search within run
```

---

## Task-Capsule Linking

### Workspace Convention

Align Moss workspace with task list ID:

```bash
CLAUDE_CODE_TASK_LIST_ID=my-project claude
```

```
capsule_store { workspace: "my-project", name: "auth", ... }
capsule_list { workspace: "my-project" }
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
1. `capsule_store` → get `fetch_key`
2. `TaskUpdate { metadata: fetch_key }`
3. Later: read `metadata`, `capsule_fetch`

---

## Quick Reference

| Need | Use |
|------|-----|
| Track what to do | Tasks |
| Enforce execution order | `blockedBy` / `blocks` |
| Persist decisions and reasoning | Capsules |
| Context across sessions | `capsule_store` → `capsule_fetch` |
| Gather parallel results | `capsule_list` → `capsule_fetch_many` |
| Scope to workflow run | `run_id` filter |
| Filter by workflow stage | `phase` filter |
| Filter by agent role | `role` filter |
| Link task to context | `fetch_key` in metadata |
| Query prior art across runs | `capsule_inventory` with `phase`/`tag` filters |
| Transient agent messages | Inbox (see [`dev/vault/skillVault/swarm/SKILL.md`](../../dev/vault/skillVault/swarm/SKILL.md)) |
