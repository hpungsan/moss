# Agent Backlog

Future enhancements for AI agent skills and orchestration patterns.

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
capsule_store {
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
capsule_latest { workspace: "pr-reviews", phase: "review" }

// Or fetch specific PR
capsule_fetch { workspace: "pr-reviews", name: "pr-123" }

// Parse "Next actions" section for callouts
// Present to user for triage selection
```

**Benefits:**
- No manual copy-paste between sessions
- Full findings preserved (user might want to fix more later)
- Audit trail of what was reviewed
- Can query past reviews: `capsule_list { workspace: "pr-reviews" }`

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
- Query by role: `capsule_list { role: "impl-review" }`

**Costs:**
- 3 capsules per workflow instead of 1
- Cleanup burden (delete 3 vs 1, or use `capsule_bulk_update` to archive)
- More context on `capsule_list`

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
capsule_list { run_id: "fix-auth", role: "plan" }         # just the plan
capsule_list { run_id: "fix-auth", role: "design-review" } # just design feedback
capsule_inventory { role: "impl-review" }                  # all impl reviews across all runs
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

---

## featurev3: Feedback Loops + Moss Checkpoints

Evolution of featurev2. Two changes: targeted feedback loops between steps, and Moss checkpoints to track loop state.

### Current featurev2 flow (single-pass)

```
Parse → Context → Design Review → Adjust (once) → Implement → Verify → Store → Report
```

Single-pass: design reviewer flags concerns, you note them and move on. Verifier finds issues, you discuss with user. No automated retry.

### featurev3 flow (loops)

```
Parse → Context → [Design Review ⇄ Adjust] → Checkpoint → [Implement ⇄ Verify] → Store → Report
```

Two loops:

1. **Design review loop (Steps 3-4):** design-verifierv2 flags concerns → adjust plan → re-review → repeat until APPROVE. Exit: verdict is APPROVE, or max iterations reached (escalate to user).

2. **Implement-verify loop (Steps 5-6):** implement → impl-verifierv2 evaluates → if PARTIAL/NOT_VERIFIED, fix flagged issues → re-verify → repeat until VERIFIED. Exit: verdict is VERIFIED, or max iterations reached (escalate to user).

### Moss checkpoint

Store a capsule **after design review loop exits** (between the two loops). This captures the "reviewed and ready" state: finalized plan + design feedback + adjustments.

Why here:
- Implementation is the longest step and most likely to be interrupted
- If session dies mid-implementation, resume from a reviewed plan, not a best-effort WIP snapshot
- Each implement-verify iteration can update the capsule with what was tried and what failed
- Discoverable later: "what was the reviewed plan for feature X?"

### Why not Ralph Loop

Ralph Loop is brute-force: same prompt repeated, no memory, context comes only from file state. It works for well-defined tasks but iterations are blind.

featurev3 loops are targeted:
- **Feedback-driven:** verifier says "issue X at file:line" → fix X specifically
- **Structured state:** orchestrator knows which iteration, what was flagged, what was attempted
- **Separate loop boundaries:** design and implementation loops have distinct entry/exit criteria
- **Escalation:** after N iterations, stop and ask the user instead of grinding

### Open questions

- Max iterations per loop before escalating to user? (2-3 seems right)
- Should implement-verify loop update the Moss checkpoint each iteration, or only on interruption?
- Should design review loop also get a Moss checkpoint, or is it short enough to not need one?
