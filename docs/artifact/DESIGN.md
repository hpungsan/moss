# Artifacts: Structured Orchestration State for Code Consumers

> Machine-readable typed data for code-based orchestration.
> Companion to Capsules (human/LLM handoffs).

## Quick Reference

| Tool | Purpose |
|------|---------|
| `artifact_store` | Create/replace artifact with typed `data` + optional `text` |
| `artifact_list` | Query by filters, returns `data` (not just metadata) |
| `artifact_fetch` | Get full artifact by id or name |
| `artifact_compose` | Bundle `text` views into markdown for LLM consumption |
| `artifact_delete` | Soft delete single artifact |
| `artifact_bulk_delete` | Delete all matching filters |
| `artifact_bulk_update` | Update metadata (phase, role, tags, TTL) by filter |
| `artifact_touch` | Extend TTL without modifying content |

---

## Background

### The Three Consumers

Moss serves different consumers with different needs:

| Consumer | Needs | Current Solution |
|----------|-------|------------------|
| **Humans** | Readable markdown | Capsules (6-section) |
| **LLMs** | Readable markdown | Capsules (6-section) |
| **Code** | Structured JSON, typed fields | ❌ None |

Orchestration layers (workflow engines, coordinators) are code, not LLMs. Code needs:
- Typed fields for sorting (`.relevance`, `.confidence`)
- Structured arrays for fan-in (`files[]`, `concerns[]`)
- Schema validation (Zod)
- Deterministic operations (dedupe by `.path`)

Capsules don't fit because:
- 6-section markdown can't be sorted/filtered programmatically
- Parsing markdown to extract fields is fragile
- `allow_thin` is a workaround, not a solution

### Solution: Artifacts

A third Moss type designed for **code consumers**:

```
Capsules ──→ Humans, LLMs (markdown handoffs)
Artifacts ─→ Code (structured orchestration state)
      │
      └───→ also has text view for LLM subagents
```

---

## Design Principles

### 1. Data is Truth, Text is View

```
data (typed JSON) ──→ caller validates by schema per kind
        │
        ▼
text (markdown) ────→ rendered view for LLMs (optional, caller provides)
```

Code operates on `data`. LLMs consume `text`.

### 2. Typed via `kind`

Each artifact kind has a registered schema:

| Kind | Consumer | Schema | Text Required |
|------|----------|--------|---------------|
| `explorer-finding` | orchestration | `ExplorerOutputSchema` | Yes (composed for stitcher) |
| `verifier-output` | orchestration | `VerifierOutputSchema` | Yes (composed for implementer) |
| `design-spec` | implementer | `DesignSpecSchema` | Yes (composed for implementer) |
| `run-record` | orchestration | `RunRecordSchema` | No (fetched directly) |
| `step-result` | crash recovery | `StepResultSchema` | No (fetched directly) |
| `dlq-entry` | resume flow | `DlqEntrySchema` | No (fetched directly) |

**Text rendering:** Only required for kinds that are composed via `artifact_compose`. Kinds fetched directly (`run-record`, `step-result`, `dlq-entry`) can omit `text` to save tokens.

### 3. Orchestration-Native

Artifacts have `run_id`, `phase`, `role` for workflow coordination — same as Capsules, because both are workflow state.

### 4. Lifecycle-Aware

Artifacts support TTL via `ttl_seconds`. Caller specifies TTL on store; Moss computes `expires_at` and handles expiration.

```typescript
artifact_store({
  kind: "explorer-finding",
  data: output,
  ttl_seconds: 3600,  // expires in 1 hour (null = no expiry)
});
```

**Expiration behavior:** Lazy filter + opportunistic batch purge.

```
Read/list/fetch:
  - Filter out expired (expires_at < now)
  - Don't delete on read path (keep reads fast)

Write (throttled):
  - If last_purge > 5 min ago: soft-delete expired active artifacts (best-effort, LIMIT 100)
  - Non-blocking, best-effort cleanup
```

No background job. Purge piggybacks on writes.

**Purge implementation (example):**
```sql
-- Mark expired artifacts as deleted, but keep them queryable with include_deleted/include_expired.
UPDATE artifacts
SET deleted_at = ?, updated_at = ?
WHERE rowid IN (
  SELECT rowid FROM artifacts
  WHERE deleted_at IS NULL AND expires_at IS NOT NULL AND expires_at < ?
  LIMIT 100
);
```

**Query flags:**
- `include_expired: false` (default): exclude expired artifacts from results
- `include_expired: true`: include expired artifacts (for debugging, audit)
- `include_deleted: false` (default): exclude soft-deleted artifacts
- `include_deleted: true`: include deleted artifacts (for recovery, audit)

**Flag interaction:** `include_expired` and `include_deleted` are independent filters. To inspect an artifact that is both expired and soft-deleted (e.g., an expired artifact that was soft-deleted during a name-collision insert), callers must set both flags.

**Expired artifact collision rule:** Expired artifacts don't block new stores. On `artifact_store`, if existing artifact with same (workspace_norm, name_norm) is expired:

1. **Soft-delete** the expired artifact (set `deleted_at = now`)
2. **Insert** new artifact with fresh ID, `version = 1`, `created_at = now`

This preserves auditability — expired artifacts remain queryable with `include_deleted: true` for debugging and compliance. The alternative (update-in-place) would lose creation timestamp and conflate "new artifact" with "TTL extension."

**Why soft-delete, not hard-delete:**
- Audit trail preserved (who created what, when)
- `include_deleted: true` can recover/inspect expired artifacts
- Consistent with explicit `artifact_delete` behavior
- No special "expired but not deleted" state to reason about

---

## Schema

### Artifact Fields

```typescript
interface Artifact<T = unknown> {
  // Identity
  id: string;                    // ULID, auto-generated
  workspace: string;             // namespace as provided (default: "default")
  workspace_norm: string;        // normalized for uniqueness/lookup
  name?: string;                 // unique handle as provided
  name_norm?: string;            // normalized for uniqueness/lookup

  // Content
  kind: string;                  // artifact type: "explorer-finding", "run-record", etc.
  data: T;                       // structured content (caller validates per kind)
  text?: string;                 // rendered view for LLMs (optional, caller provides)

  // Orchestration
  run_id?: string;               // groups artifacts for one workflow run
  phase?: string;                // workflow stage: "exploring", "implementing", "verifying"
  role?: string;                 // agent role: "code-explorer", "test-explorer", "verifier"
  tags?: string[];               // categorization

  // Lifecycle
  version: number;               // optimistic concurrency (starts at 1, incremented on update)
  ttl_seconds?: number;          // time-to-live (null = no expiry)
  expires_at?: number;           // computed: store_time + ttl_seconds
  created_at: number;            // Unix timestamp
  updated_at: number;
  deleted_at?: number;           // soft delete
}
```

### Name/Workspace Normalization

Artifacts use raw + normalized addressing, consistent with Capsules. This prevents "Auth" vs "auth" vs "AUTH" collisions while preserving original casing for display.

**Normalization rules:**
1. Trim leading/trailing whitespace
2. Lowercase
3. Collapse internal whitespace to single spaces

```
"  My Workspace  " → "my workspace"
"AUTH_SYSTEM"      → "auth_system"
"run-123-Explorer" → "run-123-explorer"
```

**Lookup uses normalized; display uses raw.** When you store `name: "Code-Explorer"`, queries match on `name_norm: "code-explorer"`, but responses return the original `name: "Code-Explorer"`.

**Unique constraint:** `UNIQUE(workspace_norm, name_norm)` — two artifacts with names that normalize to the same value cannot coexist in the same workspace.

### Kind Schemas (Example)

Caller validates data before storing. Moss stores `kind` as metadata but does not validate:

```typescript
const ArtifactKinds = {
  "explorer-finding": z.object({
    files: z.array(z.object({
      path: z.string(),
      relevance: z.enum(["high", "medium", "low"]),
      summary: z.string(),
    })),
    patterns: z.array(z.string()),
    concerns: z.array(z.string()),
    confidence: z.number().min(0).max(1),
  }),

  "verifier-output": z.object({
    verdict: z.enum(["pass", "concerns", "blocked"]),
    issues: z.array(z.object({
      id: z.string(),
      file: z.string(),
      line: z.number().optional(),  // excluded from comparison (drifts)
      category: z.string(),
      message: z.string(),
      severity: z.enum(["error", "warning", "info"]),
    })),
    summary: z.string(),
  }),

  // See FINN.md for full RunRecord schema
  "run-record": z.object({
    run_id: z.string(),
    status: z.enum(["RUNNING", "OK", "BLOCKED", "FAILED"]),
    workflow: z.enum(["plan", "feat", "fix"]),
    args: z.record(z.unknown()),
    repo_hash: z.string(),
    config: z.object({
      rounds: z.number(),
      retries: z.number(),
      timeout_ms: z.number(),
    }),
    steps: z.array(z.object({
      step_id: z.string(),
      step_instance_id: z.string(),
      name: z.string(),
      status: z.string(),
      events: z.array(z.unknown()),
      artifact_ids: z.array(z.string()),
      error_code: z.string().optional(),
    })),
    created_at: z.string(),
    updated_at: z.string(),
    last_error: z.string().optional(),
    resume_from: z.string().optional(),
  }),

  "design-spec": z.object({
    objective: z.string(),
    approach: z.string(),
    files_to_modify: z.array(z.string()),
    files_to_create: z.array(z.string()),
    constraints: z.array(z.string()),
    test_strategy: z.string(),
  }),

  "dlq-entry": z.object({
    workflow: z.enum(["plan", "feat", "fix"]),
    task: z.string(),
    failed_step: z.string(),
    inputs: z.record(z.unknown()),
    partial_results: z.array(z.string()).optional(),
    retry_count: z.number(),
    last_error: z.string(),
    relevant_files: z.array(z.string()).optional(),
  }),

  "step-result": z.object({
    step_instance_id: z.string(),
    status: z.enum(["OK", "BLOCKED", "FAILED"]),
    artifact_ids: z.array(z.string()),
    actions: z.array(z.object({
      action_id: z.string(),
      path: z.string(),
      op: z.enum(["edit", "create", "delete", "external"]),
      pre_hash: z.string().optional(),
      post_hash: z.string().optional(),
      external_ref: z.string().optional(),
    })),
    error_code: z.string().optional(),
    note: z.string().optional(),
  }),

  // Escape hatch for untyped data
  "freeform": z.record(z.unknown()),
};
```

---

## SQLite Schema

### Artifacts Table

```sql
CREATE TABLE artifacts (
    id              TEXT PRIMARY KEY,  -- ULID

    -- Identity (raw + normalized, consistent with capsules)
    workspace_raw   TEXT NOT NULL DEFAULT 'default',  -- as provided
    workspace_norm  TEXT NOT NULL DEFAULT 'default',  -- normalized for lookup
    name_raw        TEXT,              -- optional unique handle, as provided
    name_norm       TEXT,              -- normalized for uniqueness

    -- Content
    kind            TEXT NOT NULL,     -- artifact type
    data_json       TEXT NOT NULL,     -- structured JSON
    text            TEXT,              -- rendered markdown (nullable, caller provides)
    data_chars      INTEGER NOT NULL,  -- JSON character count
    text_chars      INTEGER,           -- text character count (nullable)

    -- Orchestration
    run_id          TEXT,
    phase           TEXT,
    role            TEXT,
    tags_json       TEXT,              -- JSON array

    -- Lifecycle
    version         INTEGER NOT NULL DEFAULT 1,  -- optimistic concurrency
    ttl_seconds     INTEGER,           -- null = no expiry
    expires_at      INTEGER,           -- computed: created_at + ttl_seconds
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    deleted_at      INTEGER,           -- soft delete

    -- Uniqueness on normalized values only
    UNIQUE(workspace_norm, name_norm) WHERE name_norm IS NOT NULL AND deleted_at IS NULL
);

-- Indexes (use normalized columns for queries)
CREATE INDEX idx_artifacts_run_id ON artifacts(run_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_artifacts_workspace_kind ON artifacts(workspace_norm, kind) WHERE deleted_at IS NULL;
CREATE INDEX idx_artifacts_expires ON artifacts(expires_at) WHERE expires_at IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_artifacts_name_prefix ON artifacts(workspace_norm, name_norm) WHERE name_norm IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_artifacts_updated ON artifacts(updated_at DESC) WHERE deleted_at IS NULL;  -- default list ordering
```

### FTS5 for Text Search

```sql
CREATE VIRTUAL TABLE artifacts_fts USING fts5(
    text,
    content='artifacts',
    content_rowid='rowid',
    prefix='2 3 4'
);

-- Sync triggers (same pattern as capsules)
CREATE TRIGGER artifacts_fts_insert AFTER INSERT ON artifacts
WHEN NEW.text IS NOT NULL BEGIN
    INSERT INTO artifacts_fts(rowid, text) VALUES (NEW.rowid, NEW.text);
END;

CREATE TRIGGER artifacts_fts_delete AFTER DELETE ON artifacts
WHEN OLD.text IS NOT NULL BEGIN
    INSERT INTO artifacts_fts(artifacts_fts, rowid, text) VALUES ('delete', OLD.rowid, OLD.text);
END;

-- Split UPDATE into two triggers to avoid indexing empty text when NEW.text is NULL.
CREATE TRIGGER artifacts_fts_update_delete AFTER UPDATE OF text ON artifacts
WHEN OLD.text IS NOT NULL BEGIN
    INSERT INTO artifacts_fts(artifacts_fts, rowid, text) VALUES ('delete', OLD.rowid, OLD.text);
END;

CREATE TRIGGER artifacts_fts_update_insert AFTER UPDATE OF text ON artifacts
WHEN NEW.text IS NOT NULL BEGIN
    INSERT INTO artifacts_fts(rowid, text) VALUES (NEW.rowid, NEW.text);
END;
```

---

## MCP Tools

### `artifact_store` — Store structured data

```typescript
artifact_store({
  workspace?: string,          // default: "default" (normalized for lookup, raw preserved)
  name?: string,               // unique handle (normalized for uniqueness, raw preserved)
  kind: string,                // required: "explorer-finding", etc.
  data: Record<string, unknown>, // required: structured content
  text?: string,               // optional: rendered markdown for LLM consumption (caller provides)
  run_id?: string,
  phase?: string,
  role?: string,
  tags?: string[],
  ttl_seconds?: number,        // time-to-live in seconds (null = no expiry)
  expected_version?: number,   // optimistic locking: reject if current version != expected
  mode?: "error" | "replace",  // default: "error"
})
```

**Returns:** `{ id, workspace, name, kind, version, data_chars, text_chars, expires_at }`

**Normalization:** Both `workspace` and `name` are stored raw (as provided) and normalized (for lookup/uniqueness). See [Name/Workspace Normalization](#nameworkspace-normalization). Uniqueness is checked on normalized values — `name: "Code-Explorer"` and `name: "code-explorer"` are considered the same.

**Behavior:**
1. If `ttl_seconds` provided, compute `expires_at = now + ttl_seconds`
2. If `expected_version` provided: update path (see below)
3. If `expected_version` not provided: `mode` determines create vs update
4. On create: `version = 1`. On update: `version = current.version + 1`
5. Store artifact

**`expected_version` behavior (optimistic locking):**
- `expected_version` implies update — `mode` is ignored
- Artifact must exist: if not found → `NOT_FOUND` error
- Version must match: if `current.version != expected_version` → `VERSION_MISMATCH` error

**`mode` behavior (when `expected_version` not provided):**
- `"error"` (default): fail with `NAME_ALREADY_EXISTS` if name exists
- `"replace"`: overwrite if name exists, create if not

**Replace semantics:** Both `expected_version` updates and `mode:"replace"` use true replace — all fields are overwritten. Omitted optional fields (tags, phase, role, ttl_seconds, text) are cleared, not preserved. To preserve metadata, fetch first and copy supported fields:
```typescript
const current = await moss.artifact_fetch({ workspace: "runs", name: run_id });
await moss.artifact_store({
  workspace: current.workspace,
  name: current.name,
  kind: current.kind,
  data: newData,
  text: current.text,
  run_id: current.run_id,
  phase: current.phase,
  role: current.role,
  tags: current.tags,
  ttl_seconds: current.ttl_seconds,
  expected_version: current.version,
});
```

**Name omitted:** When `name` is not provided:
- Always creates new artifact with auto-generated ID
- `mode` is irrelevant (nothing to conflict with)
- `expected_version` is invalid → `INVALID_REQUEST` error

**Note:** Moss does not validate `data` or render `text`. Caller provides both; Moss stores what's given.

**Version semantics:**

| Operation | `version` | `updated_at` | Rationale |
|-----------|-----------|--------------|-----------|
| `artifact_store` (create) | = 1 | = now | Initial |
| `artifact_store` (update) | += 1 | = now | Content changed |
| `artifact_touch` | unchanged | = now | TTL extension isn't content change |
| `artifact_bulk_update` | unchanged | = now | Metadata change, not content |

**Design rationale:** `version` is for optimistic locking on **data/text** changes only. Metadata operations (TTL, phase, role, tags) don't bump version. This separates two concerns:

1. **Content versioning** (`version`): "Has the data I fetched been modified?" Used by orchestrators doing read-modify-write on `data`.
2. **Activity tracking** (`updated_at`): "When was this artifact last touched?" Used for recency queries and cache invalidation.

**Trade-off acknowledged:** Concurrent metadata updates (`artifact_touch`, `artifact_bulk_update`) can silently overwrite each other. This is acceptable because:

- Metadata ops are typically field-specific (`set_phase`, `set_tags`), not full replace
- Finn serializes writes via RunWriter queue — concurrent metadata updates don't happen in practice
- If concurrent metadata control is needed, callers can fetch + check `updated_at` + conditional update

**Alternative considered:** Bump `version` on any mutation. Rejected because it would cause spurious `VERSION_MISMATCH` errors when:
- Process A fetches artifact (v=1), plans data update
- Process B extends TTL (would bump to v=2)
- Process A tries to store with `expected_version: 1` → fails despite data being unchanged

The current design avoids this by keeping `version` strictly for data/text changes.

**Errors:** See [Error Codes](#error-codes) for all artifact errors.

### `artifact_list` — List with structured data

```typescript
artifact_list({
  workspace?: string,
  kind?: string,               // filter by kind
  run_id?: string,
  phase?: string,
  role?: string,
  tag?: string,
  include_expired?: boolean,   // default: false
  include_deleted?: boolean,   // default: false
  order_by?: "created_at" | "updated_at",  // default: "updated_at"
  limit?: number,              // default: 50, max: 100
  offset?: number,
})
```

**Returns:** Array of artifact summaries **including `data`**, sorted descending by `order_by` field:
```typescript
{
  items: [{
    id, workspace, name, kind,
    data,                      // ← structured data returned!
    version,                   // ← for optimistic locking
    run_id, phase, role, tags,
    data_chars, text_chars,
    expires_at, created_at, updated_at
  }],
  pagination: { limit, offset, has_more }
}
```

**Ordering:**
- `order_by: "updated_at"` (default): Most recently modified first. Best for run records, long-lived artifacts where activity matters.
- `order_by: "created_at"`: Most recently created first. Best for ephemeral artifacts like explorer findings where creation order matters.

**Note:** `text` is NOT included in list (use `fetch` for full text).

**Tag filtering:** `tag` filter matches if tags array contains exact value (case-sensitive). SQL: `EXISTS (SELECT 1 FROM json_each(tags_json) WHERE value = ?)`. Single tag filter only; for multi-tag queries, filter client-side.

### `artifact_fetch` — Get full artifact

```typescript
artifact_fetch({
  id?: string,                 // by ID (exact match)
  workspace?: string,          // by name (requires workspace, normalized for lookup)
  name?: string,               // normalized for lookup
  include_expired?: boolean,
  include_deleted?: boolean,
})
```

**Returns:** Full artifact including `data` and `text`. Returns the raw (as-provided) `workspace` and `name` values.

**Addressing:** Provide `id` OR `workspace + name`, not both. Returns `AMBIGUOUS_ADDRESSING` error if both provided.

**Normalization:** When addressing by `workspace + name`, both are normalized before lookup. Fetching `name: "Code-Explorer"` will find an artifact stored as `name: "code-explorer"` (and vice versa).

**Deleted vs expired:** `include_deleted: true` includes soft-deleted artifacts, but expired artifacts are still excluded unless `include_expired: true` is also set.

### `artifact_compose` — Bundle text views

```typescript
artifact_compose({
  items: [
    { id: "..." },
    { workspace: "...", name: "..." },
  ],
  format?: "markdown" | "json", // default: "markdown"
  store_as?: {                  // optional: store composed result
    workspace: string,
    name: string,
    kind: string,
    mode?: "error" | "replace",
  },
})
```

**Returns:**
- `format: "markdown"`: `{ bundle_text: "<structured sections>" }`
- `format: "json"`: `{ parts: [{ id, name, data, text }, ...] }`

**Markdown header format:** Each artifact rendered with structured header for stable, scannable context:
```markdown
## {kind}: {role} ({name})

{text}

---
```

**Fallbacks:**
- `role` missing → `## {kind} ({name})`
- `name` missing → `## {kind}: {role} ({id})`
- Both missing → `## {kind} ({id})`

Example: `## explorer-finding: code-explorer (run-123-code-explorer)`

**`store_as` behavior:** When `store_as` is provided, the composed result is stored as a new artifact:
- `data` = `{ sources: [<artifact IDs from items>] }` (provenance only)
- `text` = the composed `bundle_text`
- Returns the stored artifact's `{ id, workspace, name, kind, version }` in addition to the bundle

**Error:** Returns `COMPOSE_MISSING_TEXT` if any artifact in `items` has no `text`.

### `artifact_delete` — Soft delete

```typescript
artifact_delete({
  id?: string,
  workspace?: string,
  name?: string,
})
```

**Addressing:** Provide `id` OR `workspace + name`, not both. Returns `AMBIGUOUS_ADDRESSING` error if both provided.

### `artifact_bulk_delete` — Delete by filter

```typescript
artifact_bulk_delete({
  // Filters (at least one required)
  workspace?: string,
  kind?: string,
  run_id?: string,
  phase?: string,
  role?: string,
  tag?: string,
})
```

**Returns:** `{ deleted: N }`

### `artifact_bulk_update` — Update metadata by filter

```typescript
artifact_bulk_update({
  // Filters (at least one required)
  workspace?: string,
  kind?: string,
  run_id?: string,
  phase?: string,
  role?: string,
  tag?: string,

  // Updates (at least one required, prefixed with set_)
  set_phase?: string,            // empty string clears
  set_role?: string,
  set_tags?: string[],           // empty array clears
  set_ttl_seconds?: number,      // null clears (no expiry)
})
```

**Returns:** `{ updated: N }`

### `artifact_touch` — Extend TTL

```typescript
artifact_touch({
  id?: string,
  workspace?: string,
  name?: string,
  ttl_seconds: number,         // reset expiration
})
```

**Addressing:** Provide `id` OR `workspace + name`, not both.

---

## Error Codes

| Code | Cause | Used By |
|------|-------|---------|
| `VERSION_MISMATCH` | `expected_version` doesn't match current | store |
| `NAME_ALREADY_EXISTS` | `mode: "error"` and artifact with same name exists | store |
| `NOT_FOUND` | Artifact doesn't exist (or `expected_version` provided but not found) | store, fetch, delete, touch, compose |
| `INVALID_REQUEST` | Invalid parameter combination (e.g., `expected_version` without `name`) | store |
| `AMBIGUOUS_ADDRESSING` | Both `id` AND `workspace + name` provided | fetch, delete, touch |
| `DATA_TOO_LARGE` | `data` exceeds 50K chars | store |
| `TEXT_TOO_LARGE` | `text` exceeds 12K chars | store |
| `COMPOSE_MISSING_TEXT` | Artifact in items has no `text` | compose |
| `FILTER_REQUIRED` | No filters provided | bulk_delete, bulk_update |

---

## Usage Examples

### Explorer Fan-out

```typescript
// Explorer outputs structured JSON
const output = ExplorerOutputSchema.parse(explorerResult);

// Store as artifact (text required for compose)
await moss.artifact_store({
  workspace: "plan",
  name: `${run_id}-code-explorer`,
  kind: "explorer-finding",
  data: output,
  text: renderExplorerFinding(output),  // required if composing for LLM
  run_id,
  role: "code-explorer",
  phase: "exploring",
  ttl_seconds: 3600,  // 1 hour
});
```

### Fan-in (Code Operates on Structured Data)

```typescript
// Query artifacts — get structured data back
const findings = await moss.artifact_list({
  run_id,
  kind: "explorer-finding",
});

// Code operates on typed fields
const allFiles = findings.items.flatMap(f => f.data.files);

const sorted = allFiles.sort((a, b) =>
  relevanceRank[b.relevance] - relevanceRank[a.relevance]
);

const deduped = mergeByPath(sorted);
```

### Stitcher Consumption (LLM Consumes Rendered Text)

```typescript
// Compose text views for LLM
const bundle = await moss.artifact_compose({
  items: findings.items.map(f => ({ id: f.id })),
  format: "markdown",
});

// Stitcher receives markdown
const plan = await stitcher.run({
  context: bundle.bundle_text,
});
```

### Run Record

```typescript
await moss.artifact_store({
  workspace: "runs",
  name: run_id,
  kind: "run-record",
  data: runRecord,
  run_id,
  phase: runRecord.status === "OK" ? "complete" : "failed",
  ttl_seconds: runRecord.status === "OK"
    ? 7 * 24 * 3600    // 7 days for success
    : 30 * 24 * 3600,  // 30 days for failure
});
```

### Verifier Cross-Round Memory

```typescript
// Round 1: Store verifier output
await moss.artifact_store({
  workspace: "feat",
  name: `${run_id}-verifier-r1`,
  kind: "verifier-output",
  data: verifierOutput,
  run_id,
  role: "impl-verifier",
  phase: "verifying",
  ttl_seconds: 7200,  // 2 hours
});

// Round 2: Fetch previous round's concerns
const prev = await moss.artifact_fetch({
  workspace: "feat",
  name: `${run_id}-verifier-r1`,
});

// Verifier can compare issues
const prevIssues = prev.data.issues;
const newIssues = currentOutput.issues;
const resolved = prevIssues.filter(p => !newIssues.some(n => n.id === p.id));
```

### DLQ Entry

```typescript
// Store DLQ entry (no TTL = persistent)
await moss.artifact_store({
  workspace: "dlq",
  name: run_id,
  kind: "dlq-entry",
  data: {
    workflow: "feat",
    task: "add user auth",
    failed_step: "verify-r2",
    inputs: { plan_file: "plans/auth.md" },
    retry_count: 2,
    last_error: "THRASHING",
    relevant_files: ["src/auth.ts"],
  },
  run_id,
  // ttl_seconds: null (persistent until manually deleted)
});

// Resume: read structured data
const entry = await moss.artifact_fetch({ workspace: "dlq", name: run_id });
const { workflow, failed_step, inputs } = entry.data;
// Route deterministically
```

---

## Comparison: Capsules vs Artifacts

| Aspect | Capsules | Artifacts |
|--------|----------|-----------|
| **Purpose** | Session handoffs | Orchestration state |
| **Consumer** | Humans, LLMs | **Code**, then LLMs |
| **Content** | 6-section markdown | Typed JSON + rendered text |
| **Validation** | 6-section required | Per-kind schema |
| **Size limit** | 12K chars | 50K JSON + 12K text |
| **Lifecycle** | Short-lived | Per-run (TTL) |
| **Orchestration** | `run_id`, `phase`, `role` | `run_id`, `phase`, `role` |
| **Normalization** | raw + norm | raw + norm |
| **list() returns** | Metadata only | **Metadata + `data`** |
| **list() ordering** | `updated_at DESC` | `updated_at DESC` (configurable) |
| **Use case** | Claude Code skills | Orchestration workflows |

---

## Implementation Plan

### Package Structure

```
internal/
├── artifact/           # NEW
│   ├── artifact.go     # Artifact struct, ArtifactSummary
│   ├── normalize.go    # Reuse capsule.Normalize or shared util
│   └── artifact_test.go
├── db/
│   ├── db.go           # Add migration N (artifacts + artifacts_fts)
│   ├── queries.go      # Add artifact CRUD + list queries
│   └── ...
├── ops/
│   ├── artifact_store.go
│   ├── artifact_list.go
│   ├── artifact_fetch.go
│   ├── artifact_compose.go
│   ├── artifact_delete.go
│   ├── artifact_touch.go
│   └── artifact_*_test.go
└── mcp/
    ├── tools.go        # Add artifact_* tool definitions
    └── handlers.go     # Add artifact_* handlers
```

---

## Migration from Current Design

### FINN.md Updates

Replace Moss capsule references with artifacts:

```diff
- await moss.capsule_store({
-   workspace: "plan",
-   run_id,
-   role: "code-explorer",
-   capsule_text: renderExplorerCapsule(output, task),
- });

+ await moss.artifact_store({
+   workspace: "plan",
+   kind: "explorer-finding",
+   data: output,
+   text: renderExplorerText(output),  // caller renders
+   run_id,
+   role: "code-explorer",
+ });
```

### Capsules Still Used For

- Session handoffs between Claude Code sessions (existing skills)
- Human-readable summaries (if needed)

### Artifacts Used For

- All orchestration state
- Anything code needs to query/sort/filter

---

## Related Docs

- [BACKLOG.md](BACKLOG.md) — Future artifact features
