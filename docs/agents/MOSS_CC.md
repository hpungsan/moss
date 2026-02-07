# Moss + Claude Code

How Moss Capsules integrate with Claude Code's built-in Teams, Tasks, and SendMessage for context persistence across sessions, agents, and swarms.

> **Prerequisite:** This doc assumes the `moss-capsule` skill is installed (`.claude/skills/moss-capsule/`). The skill covers capsule format, addressing, errors, and tool reference. This doc covers **integration patterns** — how capsules complement Claude Code's built-in coordination tools.

## Two Layers

Claude Code ships with a **coordination layer** (Teams + Tasks + SendMessage). Moss adds a **knowledge layer** (Capsules) on top.

| Layer | Built-in Tools | What It Handles |
|-------|---------------|-----------------|
| **Coordination** | `TeamCreate`, `Task`, `TaskCreate/Update/List/Get`, `SendMessage`, `TeamDelete` | Who does what, in what order, tell each other things |
| **Knowledge** | `capsule_store`, `capsule_fetch`, `capsule_list`, `capsule_search`, ... | What was decided, why, where things are, what's still open |

When a team shuts down and `TeamDelete` runs, the coordination layer is gone — tasks deleted, messages gone, team config removed. Capsules persist in `~/.moss/moss.db` indefinitely.

### What Tasks Carry vs What They Don't

Tasks have `subject`, `description`, `metadata`, `status`, `owner`, `blockedBy/blocks`. That covers coordination. But tasks don't carry:

- Decisions made during implementation and **why**
- Constraints discovered along the way
- Key file locations, commands, gotchas
- Open questions that affect other tasks or future work

**Capsules fill this gap.** They carry structured context (6 enforced sections — see skill `SKILL.md`) that survives:

- Session restarts and `/clear` commands
- Team shutdown (`TeamDelete`)
- Days or weeks between work sessions
- Agent and subagent boundaries

---

## SendMessage vs Capsules

Claude Code's `SendMessage` handles inter-agent communication within a team. Capsules handle persistent structured context.

| Aspect | SendMessage | Capsule |
|--------|------------|---------|
| Structure | Free text | 6 required sections (lint-enforced) |
| Persistence | Session-bound (dies with team) | Survives sessions, teams, `/clear`, days |
| Quality | None | Lint + size limits |
| Queryability | None (auto-delivered) | `capsule_list`, `capsule_search`, `capsule_inventory` |
| Scoping | Recipient name | `run_id`, `phase`, `role`, `workspace`, `tags` |
| Batch retrieval | N/A | `capsule_list` → `capsule_fetch_many` |

**When to use:**
- **SendMessage**: Transient coordination ("done", "found a bug in auth.go", "need help with X")
- **Capsule**: Context that outlives the conversation (decisions, key locations, next actions)

---

## Patterns

### Pattern 1: Parallel Specialists

Multiple agents review in parallel. Each stores structured findings. Leader gathers them.

```
# Leader sets up the team
TeamCreate { team_name: "pr-review" }
TaskCreate { subject: "Security review", description: "Review PR for vulnerabilities" }
TaskCreate { subject: "Performance review", description: "Review PR for perf issues" }

# Spawn teammates (run in background so they work in parallel)
Task {
  team_name: "pr-review", name: "sec-reviewer",
  subagent_type: "general-purpose",
  prompt: "Claim your task, do the review, store findings as a capsule
           with run_id 'pr-review-42' and role 'security-reviewer'.
           Then SendMessage the leader that you're done.",
  run_in_background: true
}
Task {
  team_name: "pr-review", name: "perf-reviewer",
  subagent_type: "general-purpose",
  prompt: "Claim your task, do the review, store findings as a capsule
           with run_id 'pr-review-42' and role 'performance-reviewer'.
           Then SendMessage the leader that you're done.",
  run_in_background: true
}
```

Each teammate stores findings (see skill `examples.md` for capsule format):

```
# sec-reviewer does:
capsule_store {
  name: "security-findings",
  run_id: "pr-review-42",
  role: "security-reviewer",
  capsule_text: "## Objective\nSecurity review of PR #42...
## Current status\nFound 2 issues...
## Decisions\n- SQL injection in user query needs parameterization...
## Next actions\n- Fix parameterized query in db/users.go:47...
## Key locations\n- db/users.go:47, handlers/auth.go:112...
## Open questions\n- Is the admin endpoint intentionally unprotected?"
}
TaskUpdate { taskId: "1", status: "completed" }
SendMessage {
  type: "message", recipient: "team-lead",
  content: "Done. Findings in capsule 'security-findings'.",
  summary: "Security review complete"
}
```

Leader gathers all findings without context bloat:

```
# Step 1: Browse (summaries only, no text loaded)
capsule_list { run_id: "pr-review-42" }

# Step 2: Fetch only what you need
capsule_fetch_many { items: [
  { workspace: "default", name: "security-findings" },
  { workspace: "default", name: "perf-findings" }
]}

# Or compose into a single bundle
capsule_compose { items: [
  { workspace: "default", name: "security-findings" },
  { workspace: "default", name: "perf-findings" }
]}
```

**Why Moss matters here:** Without capsules, findings go through `SendMessage` — unstructured, lost when the team shuts down. With capsules, findings are structured, queryable by `run_id`, and available next week if someone asks "what did we find in PR #42?"

---

### Pattern 2: Pipeline (Sequential Dependencies)

Each stage inherits context from the previous one via capsules. Tasks handle ordering via `blockedBy`.

```
# Create pipeline tasks with dependencies
TaskCreate { subject: "Research auth options" }          # Task 1
TaskCreate { subject: "Design auth plan" }               # Task 2
TaskCreate { subject: "Implement auth" }                 # Task 3
TaskUpdate { taskId: "2", addBlockedBy: ["1"] }
TaskUpdate { taskId: "3", addBlockedBy: ["2"] }
```

Stage 1 (Research) stores decisions:

```
capsule_store {
  name: "research", run_id: "feature-oauth", phase: "research",
  capsule_text: "## Objective\nEvaluate OAuth providers...
## Decisions\n- Auth0 over Firebase: better Go SDK, lower latency...
## Key locations\n- Existing session logic at internal/auth/session.go..."
}
TaskUpdate { taskId: "1", status: "completed" }
```

Stage 2 (Plan) picks up where research left off:

```
capsule_fetch { workspace: "default", name: "research" }
# Now knows WHY Auth0, WHERE existing code lives

capsule_store {
  name: "plan", run_id: "feature-oauth", phase: "plan",
  capsule_text: "## Decisions\n- 3-phase rollout based on research..."
}
```

Stage 3 (Implement) gets the full chain:

```
capsule_list { run_id: "feature-oauth" }
capsule_fetch_many { items: [
  { workspace: "default", name: "research" },
  { workspace: "default", name: "plan" }
]}
# Has complete decision history: WHY Auth0, WHAT the plan is, WHERE to start
```

**Why Moss matters here:** Task descriptions are static — written at creation time, before research happened. Capsules carry the evolving context forward through each stage.

---

### Pattern 3: Swarm (Self-Organizing Workers)

Workers claim tasks from a shared queue and share discoveries via capsules so others don't duplicate work.

```
# Workers check what others found before starting
worker-2 (about to review payment.go):
  # Option A: Browse by run_id
  capsule_list { run_id: "codebase-review" }
  # Sees worker-1 already stored "user-model-review"

  # Option B: Search for relevant findings
  capsule_search { query: "payment OR validation", run_id: "codebase-review" }
  # Finds capsules mentioning payment even if named something else

  capsule_fetch { workspace: "default", name: "user-model-review" }
  # Learns: "Found shared validation logic that affects payment.go"
  # Can account for this instead of rediscovering it

# After finishing, stores own findings
worker-2:
  capsule_store {
    name: "payment-review", run_id: "codebase-review",
    capsule_text: "## Decisions\n- Shared validation from user model confirmed...
## Open questions\n- Rate limiting logic duplicated across 3 files..."
  }
```

Leader synthesizes:

```
capsule_list { run_id: "codebase-review" }
capsule_fetch_many { items: [all capsule refs from list] }
# Or compose into a single bundle for a summary
capsule_compose { items: [...], store_as: { name: "review-summary" } }
```

**Why Moss matters here:** `SendMessage` is point-to-point — Worker-2 can't read what Worker-1 sent to the leader. Capsules are a shared knowledge pool any agent can browse and fetch.

---

### Pattern 4: Research + Implementation (Across Sessions)

Research happens in one session or subagent. Implementation may happen hours or days later.

```
# Session A / researcher agent:
capsule_store {
  name: "caching-research", phase: "research",
  tags: ["redis", "caching", "performance"],
  capsule_text: "## Decisions\n- Redis over Memcached: better persistence, pub/sub...
## Key locations\n- Existing cache config at config/cache.yml..."
}

# Session B (days later, different session):
capsule_search { query: "caching redis" }
# Find it even if you forgot the exact name
capsule_fetch { workspace: "default", name: "caching-research" }
# Full research context even though the original session is long gone

# Session C (review, different person):
capsule_fetch { workspace: "default", name: "caching-research" }
# Can verify implementation matches research recommendations
```

**Why Moss matters here:** `SendMessage` and Tasks die with the session. There's no built-in mechanism to carry structured context across session boundaries. This is the simplest and most common Moss use case — no teams required.

---

### Pattern 5: Coordinated Multi-File Refactoring

Multiple workers refactor different parts. A downstream agent needs to know what changed and why.

```
# model-worker stores refactoring decisions
capsule_store {
  name: "user-refactor", run_id: "auth-refactor", role: "model-worker",
  capsule_text: "## Decisions\n- Extracted AuthenticatableUser concern
- Changed signature: authenticate(password) → authenticate!(password)
## Key locations\n- New concern at app/models/concerns/authenticatable_user.rb"
}

# controller-worker stores its decisions
capsule_store {
  name: "session-refactor", run_id: "auth-refactor", role: "controller-worker",
  capsule_text: "## Decisions\n- Now uses User.authenticate! (bang method)
## Open questions\n- Should we add rate limiting to SessionsController?"
}

# spec-worker (blocked by both) gathers all context
capsule_list { run_id: "auth-refactor" }
capsule_fetch_many { items: [
  { workspace: "default", name: "user-refactor" },
  { workspace: "default", name: "session-refactor" }
]}
# Knows: new concern location, method signature change, open rate-limiting question
```

---

## Non-Swarm Use Cases

### Session Continuity

No teams needed — just persist context across sessions.

```
Session A:
  1. Do work, make decisions
  2. capsule_store { name: "auth-progress", capsule_text: "..." }
  3. End session (or /clear)

Session B (hours/days later):
  1. capsule_fetch { workspace: "default", name: "auth-progress" }
  2. Continue with full context — decisions, locations, open questions
```

### Tasks + Capsules

Use Claude Code's built-in Tasks for coordination, Moss for context.

```
1. TaskCreate { subject: "Implement auth" }
2. Do work, make decisions
3. capsule_store { name: "auth-context", ... }
   # Response includes fetch_key: { moss_workspace: "default", moss_capsule: "auth-context" }
4. TaskUpdate { taskId: "1", metadata: { moss_workspace: "default", moss_capsule: "auth-context" } }
5. Later: TaskGet → read metadata → capsule_fetch
```

### Cross-Run Knowledge

Query capsules from prior workflow runs to inform new ones.

```
# Starting a new OAuth implementation — check prior art
capsule_inventory { phase: "research", tag: "oauth" }
# Returns capsules from ANY prior run tagged "oauth" in research phase

# Full-text search across all capsules (see skill reference.md § Full-Text Search)
capsule_search { query: "OAuth provider comparison", phase: "research" }

# Fetch relevant prior research
capsule_fetch { workspace: "feature-oauth-v1", name: "research" }
# Decisions from months ago:
# "Auth0 rejected due to cost, Google OAuth chosen for user-facing..."
```

**Use cases:**
- **Onboarding**: New agent/session queries prior art before starting
- **Avoiding re-work**: Check if similar research already exists
- **Pattern mining**: `capsule_inventory { phase: "security" }` → all security reviews ever done
- **Postmortems**: `capsule_inventory { tag: "incident" }` → related incidents

---

## Orchestration Fields

Scope capsules to specific workflow runs (see skill `reference.md` § Multi-Agent Orchestration Fields):

| Field | Purpose | Example |
|-------|---------|---------|
| `run_id` | Identify workflow run | `"pr-review-42"` |
| `phase` | Workflow stage | `"research"`, `"plan"`, `"implement"`, `"review"` |
| `role` | Agent role | `"security-reviewer"`, `"architect"` |

**Query by scope:**
```
capsule_list { run_id: "pr-42" }                      # All capsules from this run
capsule_list { run_id: "pr-42", phase: "research" }   # Just research phase
capsule_latest { run_id: "pr-42", role: "architect" }  # Latest from architect
capsule_search { query: "SQL injection", run_id: "pr-42" }  # Search within run
```

---

## Task-Capsule Linking

### Workspace Convention

Align Moss workspace with team name for clean scoping:

```
TeamCreate { team_name: "my-project" }

capsule_store { workspace: "my-project", name: "auth", ... }
capsule_list { workspace: "my-project" }
```

All capsules for a project live in one queryable namespace.

### fetch_key

Moss responses include `fetch_key` for direct Task metadata linking (see skill `reference.md` § Output Fields):

```json
{
  "id": "01J...ULID",
  "fetch_key": {
    "moss_workspace": "my-project",
    "moss_capsule": "auth"
  }
}
```

**Workflow:**
1. `capsule_store` → get `fetch_key` from response
2. `TaskUpdate { taskId: "1", metadata: { moss_workspace: "my-project", moss_capsule: "auth" } }`
3. Later: `TaskGet` → read `metadata` → `capsule_fetch`

---

## Giving Subagents Moss Access

Teammates spawned via `Task` with `team_name` inherit MCP tools from the project config. For custom agent definitions (`.claude/agents/`), explicitly grant Moss tools in the frontmatter:

```yaml
---
name: security-reviewer
tools: Bash, Read, Glob, Grep, mcp__moss__capsule_store, mcp__moss__capsule_fetch
skills:
  - moss-capsule
---
```

See `docs/integrations/claude-code.md` for full subagent configuration and available MCP tool names.

---

## Quick Reference

| Need | Built-in | Moss |
|------|----------|------|
| Track what to do | `TaskCreate` / `TaskUpdate` / `TaskList` | — |
| Enforce execution order | `addBlockedBy` / `addBlocks` on `TaskUpdate` | — |
| Transient agent messages | `SendMessage` (DM or broadcast) | — |
| Spawn teammates | `Task` with `team_name` + `name` | — |
| Persist decisions | — | `capsule_store` |
| Context across sessions | — | `capsule_store` → `capsule_fetch` |
| Gather parallel results | — | `capsule_list` → `capsule_fetch_many` |
| Scope to workflow run | — | `run_id` filter on capsule tools |
| Filter by stage / role | — | `phase` / `role` filters |
| Link task to context | Task `metadata` field | `fetch_key` from capsule response |
| Search past knowledge | — | `capsule_search` / `capsule_inventory` |
| Compose multiple capsules | — | `capsule_compose` |
| Incrementally update | — | `capsule_append` |
| Batch cleanup | — | `capsule_bulk_delete` / `capsule_bulk_update` |

> **Capsule mechanics:** See skill `SKILL.md` for format, addressing, errors, and tool reference. See `examples.md` for store/fetch/orchestration examples. See `reference.md` for schema, search syntax, and batch operations.
