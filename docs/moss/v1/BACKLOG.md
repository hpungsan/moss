# Moss Post-v1 Backlog

Features and enhancements for future versions.

---

## v1.1 Candidates

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

### CLI Enhancements

v1 CLI outputs JSON only. Future enhancements:

- **Orchestration flags** — `--run-id`, `--phase`, `--role` for store, update, list, inventory, latest commands (MCP has these; CLI deferred since orchestration is primarily for multi-agent workflows)
- **Table formatting** for `list` and `inventory` commands (human-readable output)
- **Color output** for better terminal readability
- **Shell completion** (bash, zsh, fish)
- **Interactive mode** for guided capsule operations

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

### MCP Server Graceful Shutdown (HTTP Transport)

**Stdio**: Already handled. `mcp-go`'s `ServeStdio()` catches SIGTERM/SIGINT and shuts down gracefully.

**HTTP**: Will need explicit shutdown handling when REST API is added (v1.1+). Use the transport server’s `Shutdown(ctx)` (e.g., `server.NewSSEServer(...).Shutdown(ctx)` / `server.NewStreamableHTTPServer(...).Shutdown(ctx)`) with a timeout context.

### Import: Reuse ULID Entropy Source

`internal/ops/import.go` creates a new `ulid.Monotonic()` per call to `generateNewULID()`. Could be reused for minor efficiency gain.

### Import: Increase Scanner Buffer Size

`internal/ops/import.go` uses `bufio.NewScanner()` with default 64KB line limit. If `capsule_max_chars` is increased significantly (e.g., 50K+), large export records could be silently truncated. Consider using `scanner.Buffer()` to set explicit limit matching max capsule size + overhead.

---

## Considered & Deferred

### Content Lint Checks

Warn or reject if capsule sections lack actual content:
- ≥1 Next action item
- ≥1 Decision/constraint item
- ≥1 Key location pattern

**Decision:** Not in v1.0. Headers provide structure; content quality is agent's responsibility. Can add warnings in future if thin capsules become a problem.
