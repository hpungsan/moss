# Agent Backlog

Future enhancements for AI agent skills and orchestration patterns.

---

## CLI Improvements

### Unknown Command Handling

Currently, `moss invalidcommand` falls through to MCP server mode and hangs waiting for stdin. Should show an error instead.

**Current behavior:**
```
moss invalidcommand  # hangs (starts MCP server, waits for JSON-RPC)
```

**Expected behavior:**
```
moss invalidcommand  # error: unknown command "invalidcommand"
                     # Run 'moss --help' for usage.
```

**Fix:** In `main.go`, if there's an argument that's not a known command and stdin is a terminal, show error instead of starting MCP mode.

---

## Skill Integrations

### PR Review → Fix Handoff

Integrate Moss into the `/pr-review` → `/fix` skill workflow for seamless cross-session handoffs.

**Current flow:**
1. Session 1: `/pr-review <pr>` → outputs findings to user
2. User manually copies issues they want to fix
3. Session 2: `/fix` → user pastes issues

**Enhanced flow:**
1. Session 1: `/pr-review <pr>` → outputs findings to user AND stores in Moss
2. Session 2: `/fix` → fetches from Moss, user selects which to fix

**pr-review stores:**
```
store {
  workspace: "pr-reviews",
  name: "pr-<number>",
  run_id: "pr-<number>",
  phase: "review",
  tags: ["pr-review"],
  capsule_text: |
    ## Objective
    PR #<number> review findings

    ## Status
    Review complete. <N> blockers, <M> warnings, <P> notes.

    ## Decisions
    - Verdict: APPROVE | REQUEST_CHANGES

    ## Next actions
    - [BLOCKER] <desc> — `path:line`
    - [WARNING] <desc> — `path:line`
    - ...

    ## Key locations
    - <changed files from PR>

    ## Open questions
    - (none, or unresolved items)
}
```

**/fix fetches:**
```
// Check for recent pr-review findings
latest { workspace: "pr-reviews", phase: "review" }

// Or fetch specific PR
fetch { workspace: "pr-reviews", name: "pr-123" }

// Parse "Next actions" section for callouts
// Present to user for triage selection
```

**Benefits:**
- No manual copy-paste between sessions
- Full findings preserved (user might want to fix more later)
- Audit trail of what was reviewed
- Can query past reviews: `list { workspace: "pr-reviews" }`

**Cleanup:** After `/fix` completes, optionally delete the pr-review capsule or keep for history.

---

## Skill Orchestration Patterns

Patterns for skills that use Moss with subagents. Currently, `/feature` and `/fix` skills use a simple pattern (single capsule, subagents return inline). These patterns enable richer orchestration when needed.

### Subagent Persistence

Have subagents store their own capsules instead of returning findings inline.

**Current (simple):**
```
skill stores: fix-<slug>
  ↓
design-verifier → returns feedback inline
  ↓
impl-verifier → returns verdict inline, updates phase
```

**With persistence:**
```
skill stores: fix-<slug> (role: "plan")
  ↓
design-verifier stores: fix-<slug>-design (role: "design-review")
  ↓
impl-verifier stores: fix-<slug>-impl (role: "impl-review")
```

**Benefits:**
- Full audit trail (each agent's analysis persisted)
- Crash resilience (can resume after session dies)
- Cross-session visibility ("what did verifiers find last time?")
- Query by role: `list { role: "impl-review" }`

**Costs:**
- 3 capsules per workflow instead of 1
- Cleanup burden (delete 3 vs 1, or use `bulk_update` to archive)
- More context on `list`

**When to upgrade:**
- Security-sensitive changes (audit trail required)
- Team handoffs (others need to see reasoning)
- Complex multi-session features
- Debugging failed verifications

### Role Field

Use `role` to distinguish capsule purposes within a workflow.

**Roles for skill workflows:**
- `plan` — main skill's implementation plan
- `design-review` — design-verifier findings
- `impl-review` — impl-verifier findings

**Query patterns:**
```
list { run_id: "fix-auth", role: "plan" }         # just the plan
list { run_id: "fix-auth", role: "design-review" } # just design feedback
inventory { role: "impl-review" }                  # all impl reviews across all runs
```

**Note:** Role is only useful if subagents persist. For single-capsule workflows, role adds no value.

### Conditional Persistence

Persist subagent findings only for high-severity items:

```
# In skill, after triage
if severity == BLOCKER:
  # Tell subagents to persist findings
  spawn design-verifier with persist: true
else:
  # Inline only (current behavior)
  spawn design-verifier
```

Gets audit trails where they matter most without overhead for simple fixes.
