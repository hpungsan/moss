# Moss + Claude Code

How Moss Capsules integrate with Claude Code's built-in Teams, Tasks, and SendMessage.

> **Prerequisite:** This doc assumes the `moss-capsule` skill is installed (`.claude/skills/moss-capsule/`). The skill covers capsule format, addressing, errors, and tool reference. This doc covers **integration patterns**.

## What Moss Adds

Claude Code ships with coordination tools (Teams + Tasks + SendMessage). These handle who does what, in what order, and how agents talk to each other.

Moss adds **structured, queryable, persistent context** on top. Specifically:

- **Structure**: 6 lint-enforced sections vs free-text messages or ad-hoc files
- **Queryability**: Search, filter by `run_id`/`phase`/`role`, compose across capsules
- **Persistence**: Survives session restarts, `/clear`, `TeamDelete`, days between sessions

You can achieve basic persistence by writing files. Capsules are worth the overhead when you need to **find context again later** — across sessions, across agents, or across time.

### When Capsules Are Worth It

- Context that will be needed **across sessions** (decisions, research, key locations)
- Multi-agent workflows where agents need to **discover what others found** (not just receive messages)
- Workflows where you'll want to **query past work** by run, phase, role, or full-text search
- Long-running projects where structured context beats grepping through old files

### When They're Not

- Single-session team workflows where `SendMessage` or shared files suffice
- One-off tasks where the context won't be needed again
- Simple agent handoffs where a file or task description carries enough context

---

## SendMessage vs Capsules

| Aspect | SendMessage | Capsule |
|--------|------------|---------|
| Structure | Free text | 6 required sections (lint-enforced) |
| Persistence | Session-bound (dies with team) | Survives sessions, teams, `/clear`, days |
| Quality | None | Lint + size limits |
| Queryability | None (auto-delivered) | `capsule_list`, `capsule_search`, `capsule_inventory` |
| Scoping | Recipient name | `run_id`, `phase`, `role`, `workspace`, `tags` |
| Batch retrieval | N/A | `capsule_list` → `capsule_fetch_many` |

Use `SendMessage` for transient coordination ("done", "found a bug", "need help"). Use capsules when the context should outlive the conversation.

---

## Patterns

### Pattern 1: Session Continuity

The simplest use case — no teams required. Persist context across sessions.

```
Session A:
  1. Do work, make decisions
  2. capsule_store { name: "auth-progress", capsule_text: "..." }
  3. End session (or /clear)

Session B (hours/days later):
  1. capsule_fetch { workspace: "default", name: "auth-progress" }
  2. Continue with full context — decisions, locations, open questions
```

This is the core Moss value. Built-in tools (Tasks, SendMessage) are session-bound. Files persist but aren't searchable by metadata. Capsules give you `capsule_search { query: "auth" }` to find context you stored weeks ago, even if you forgot the exact name.

---

### Pattern 2: Cross-Run Knowledge

Query capsules from prior workflow runs to inform new ones.

```
# Starting a new OAuth implementation — check prior art
capsule_inventory { phase: "research", tag: "oauth" }
# Returns capsules from ANY prior run tagged "oauth" in research phase

# Full-text search across all capsules
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

### Pattern 3: Shared Knowledge Pool (Swarm)

`SendMessage` is point-to-point — Worker-2 can't read what Worker-1 sent to the leader. When agents need to build on each other's findings, capsules act as a shared knowledge pool.

For simple parallel work where each agent is independent, `SendMessage` is fine. Use capsules when workers might overlap or when findings need to be discoverable by other agents.

```
# worker-2 checks what others found before starting
capsule_list { run_id: "codebase-review" }
# Sees worker-1 already stored "user-model-review"

capsule_search { query: "payment OR validation", run_id: "codebase-review" }
# Finds capsules mentioning payment even if named differently

capsule_fetch { workspace: "default", name: "user-model-review" }
# Learns: "Found shared validation logic that affects payment.go"
# Can account for this instead of rediscovering it

# After finishing, stores own findings
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
# Or compose — filtered to just decisions and open questions
capsule_compose { items: [...], sections: ["Decisions", "Open questions"] }
# Or compose and store the filtered bundle
capsule_compose { items: [...], sections: ["Decisions", "Open questions"],
  store_as: { name: "review-summary" } }
```

---

### Pattern 4: Pipeline (Sequential Dependencies)

Tasks handle ordering via `blockedBy`. Capsules carry evolving context forward through each stage.

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

You could pass context through task descriptions or files instead. Capsules make it queryable — Stage 3 can `capsule_list { run_id: "feature-oauth" }` to discover all prior stages without knowing their names upfront.

---

### Pattern 5: Parallel Specialists

Multiple agents review in parallel. Each stores structured findings. Leader gathers them.

This pattern is most useful when findings will be referenced again later (e.g., recurring reviews, audits). For a one-off review where you just need the results now, `SendMessage` with structured text works fine.

```
# Leader ("team-lead") sets up the team
TeamCreate { team_name: "pr-review" }
TaskCreate { subject: "Security review", description: "Review PR for vulnerabilities" }
TaskCreate { subject: "Performance review", description: "Review PR for perf issues" }

# Spawn teammates
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

Leader gathers all findings:

```
capsule_list { run_id: "pr-review-42" }

capsule_fetch_many { items: [
  { workspace: "default", name: "security-findings" },
  { workspace: "default", name: "perf-findings" }
]}

# Or compose with sections filter — only decisions and open questions
capsule_compose { items: [
  { workspace: "default", name: "security-findings" },
  { workspace: "default", name: "perf-findings" }
], sections: ["Decisions", "Open questions"]}
```

The payoff comes later: `capsule_search { query: "SQL injection", run_id: "pr-review-42" }` works next month. SendMessage content is long gone by then.

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
| Persist decisions across sessions | — | `capsule_store` → `capsule_fetch` |
| Discoverable shared knowledge | — | `capsule_list` → `capsule_fetch_many` |
| Scope to workflow run | — | `run_id` filter on capsule tools |
| Filter by stage / role | — | `phase` / `role` filters |
| Link task to context | Task `metadata` field | `fetch_key` from capsule response |
| Search past knowledge | — | `capsule_search` / `capsule_inventory` |
| Compose multiple capsules | — | `capsule_compose` |
| Incrementally update | — | `capsule_append` |
| Batch cleanup | — | `capsule_bulk_delete` / `capsule_bulk_update` |

> **Capsule mechanics:** See skill `SKILL.md` for format, addressing, errors, and tool reference. See `examples.md` for store/fetch/orchestration examples. See `reference.md` for schema, search syntax, and batch operations.
