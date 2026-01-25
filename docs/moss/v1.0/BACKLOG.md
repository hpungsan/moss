# Moss Post-v1.0 Backlog

Features and enhancements deferred from v1.0.

---

## v1.1 Candidates

### Orchestration Fields

Add `run_id`, `phase`, `role` for multi-agent workflow scoping:

```json
{
  "run_id": "pr-review-abc123",
  "phase": "design",
  "role": "design-intent",
  ...
}
```

(Example for `moss.store` tool)

- `run_id` — groups capsules from single command run
- `phase` — e.g., `base`, `design`, `qa`, `security`, `docs`, `final`
- `role` — e.g., `design-intent`, `design-principles`

**Database schema additions:**

```sql
-- New columns in capsules table
run_id TEXT NULL,
phase TEXT NULL,
role TEXT NULL

-- Index for run-scoped queries
CREATE INDEX IF NOT EXISTS idx_capsules_run_id
ON capsules(run_id, phase, role)
WHERE run_id IS NOT NULL AND deleted_at IS NULL;
```

Enables filtering on `moss.list` and `moss.inventory`:

```json
{ "workspace": "startupA", "run_id": "pr-review-abc123", "phase": "design", "limit": 20 }
```

```json
{ "run_id": "pr-review-abc123", "phase": "design", "role": "design-intent", "limit": 200 }
```

**Example flow: PR review with design gate + detail reviewers**

```
1. Parent stores base context:
   moss.store { workspace: "phinn", name: "pr-123-base", run_id: "run-abc", phase: "base", ... }

2. Design agents fetch base:
   moss.fetch { workspace: "phinn", name: "pr-123-base" }

3. Each design agent stores findings:
   moss.store { name: "pr-123-design-intent", run_id: "run-abc", phase: "design", role: "design-intent", ... }
   moss.store { name: "pr-123-design-principles", run_id: "run-abc", phase: "design", role: "design-principles", ... }

4. Parent composes design outputs:
   moss.compose { items: [...], store_as: { name: "pr-123-design" } }

5. QA/Sec/Docs agents batch-fetch:
   moss.fetch_many { items: [{ name: "pr-123-base" }, { name: "pr-123-design" }] }

6. Browse run artifacts:
   moss.list { workspace: "phinn", run_id: "run-abc" }
```

### `moss.compose` Tool

Deterministic assembly of multiple capsules into one bundle.

```json
{
  "items": [
    { "workspace": "phinn", "name": "pr-123-base" },
    { "workspace": "phinn", "name": "pr-123-design" }
  ],
  "format": "markdown",
  "store_as": { "workspace": "phinn", "name": "pr-123-postgate", "mode": "replace" }
}
```

**Parameters:**
- `items` — ordered list of capsule refs (by id OR by workspace+name)
- `format` — `"markdown"` (default) or `"json"`
- `store_as` — optional; if provided, persists the composed output as a new capsule

**Format options:**
- `markdown`: adds `## <title or name>` headers between capsules
- `json`: `{ "parts": [{ "name": "...", "text": "..." }, ...] }`

**Behavior:**
- Deterministic assembly (no LLM)
- Enforce `capsule_max_chars` on output → **413 COMPOSE_TOO_LARGE** if exceeded
- Lint only if `store_as` is provided
- Partial failure: if any item is missing, return error (no partial compose)

**Output (no store_as):**
```json
{
  "bundle_text": "## pr-123-base\n\nObjective: ...\n\n---\n\n## pr-123-design\n\nObjective: ...",
  "bundle_chars": 3241,
  "parts_count": 2
}
```

**Output (with store_as):**
```json
{
  "bundle_text": "...",
  "bundle_chars": 3241,
  "parts_count": 2,
  "stored": { "id": "01J...ULID" }
}
```

### Optimistic Concurrency

Add `if_updated_at` to `moss.update`:

```json
{
  "name": "auth",
  "capsule_text": "...",
  "if_updated_at": 1737260500
}
```

Rejects if capsule was modified since timestamp (prevents overwrites).

### REST API

HTTP interface for debugging (localhost only, bind to `127.0.0.1`).

Resource: `/capsules`

- `POST /capsules` → store
- `GET /capsules/{id}` → fetch by id
- `GET /capsules/by-name?workspace=...&name=...&include_text=true|false` → fetch by name
- `PUT /capsules/{id}` → update content/context by id
- `PUT /capsules/by-name?workspace=...&name=...` → update content/context by name
- `DELETE /capsules/{id}` → delete by id
- `GET /capsules/latest?workspace=...&include_text=true|false`
- `GET /capsules?workspace=...&limit=...&offset=...` → list (summaries only)
- `GET /capsules/inventory?...` → inventory (summaries only)

### CLI Completeness

Add missing commands:

```bash
moss update --name=auth < updated.md
moss latest [--workspace=X] [--include-text]
```

Currently: use MCP or delete+store for update; no CLI for latest.

### Real Tokenizer

Replace word-count heuristic with model-specific tokenizer (e.g., tiktoken).

### `moss.restore` Tool

Recover soft-deleted capsules:

```json
{ "workspace": "default", "name": "auth" }
```

Currently: use export/import.

---

## v1.2+ Ideas

### `moss.search` Tool

SQLite FTS5 for full-text search across capsules:

```json
{ "query": "authentication JWT" }
```

### Semantic Search

Embeddings-based similarity search.

### Versioning

Keep last N revisions of a capsule (extends `moss.fetch`):

```json
{ "name": "auth", "version": -1 }  // previous version
```

### Evidence Store

Optional snippets/transcript refs with "expand" semantics:

```json
{
  "capsule_text": "...",
  "evidence": [
    { "type": "snippet", "ref": "src/auth.ts:42-50", "content": "..." }
  ]
}
```

---

## Minor Improvements

### Context Propagation to Ops Layer

Pass `context.Context` through MCP handlers to ops functions. Currently handlers accept context but don't pass it to the ops layer, preventing cancellation of long-running operations (e.g., large imports).

**Scope:** Modify all 11 ops functions to accept `context.Context` as first parameter, propagate to db layer.

### MCP Server Graceful Shutdown

Add graceful shutdown handling to the MCP server. Currently `server.ServeStdio()` has no shutdown hook. For stdio transport this is low priority since process exit handles cleanup, but would be needed for HTTP transport (REST API).

### Import: Reuse ULID Entropy Source

`internal/ops/import.go` creates a new `ulid.Monotonic()` per call to `generateNewULID()`. Could be reused for minor efficiency gain.

### Export: Clean Up Partial File on Failure

`internal/ops/export.go` leaves incomplete file on disk if iteration fails mid-export. Could delete partial file on error.

### Import: Handle ulid.MustNew Panic

`ulid.MustNew` panics if entropy source fails. Could use `ulid.New` which returns an error. Low priority since `crypto/rand.Reader` failure indicates system-level issues.

---

## Considered & Deferred

### Content Lint Checks

Warn or reject if capsule sections lack actual content:
- ≥1 Next action item
- ≥1 Decision/constraint item
- ≥1 Key location pattern

**Decision:** Not in v1.0. Headers provide structure; content quality is agent's responsibility. Can add warnings in future if thin capsules become a problem.
