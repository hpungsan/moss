# Capsule Backlog

Features and enhancements for the Capsule primitive.

---

## Candidates

### Optimistic Concurrency

Add `if_updated_at` to `capsule_update`:

```json
{
  "name": "auth",
  "capsule_text": "...",
  "if_updated_at": 1737260500
}
```

Rejects if capsule was modified since timestamp (prevents overwrites).

**Context:** `capsule_update` is a read-modify-write operation (`capsule_fetch` then `UpdateByID`) and can lose concurrent updates. `capsule_delete` also does a name→id read before `SoftDelete`. In the common swarm pattern, capsules are treated as agent-owned (writers usually don't target the same capsule), but the system does not enforce this, and humans/CLIs/orchestrators can still collide. Since `capsule_update` replaces the full `capsule_text`, optimistic concurrency mainly prevents silent clobbering and forces a retry; it won’t merge concurrent edits. Defer until a concrete collision-prone workflow emerges.

### Multi-Run Queries

Allow `run_id` filter to accept an array for querying across multiple runs:

```json
capsule_inventory { "run_id": ["run-001", "run-002"] }
```

Use case: Comparing capsules from related runs or aggregating results from parallel workflows.

### Run-Scoped Purge

Add `run_id` filter to `capsule_purge` for cleaning up completed workflows:

```json
capsule_purge { "run_id": "pr-review-abc123" }
```

Permanently deletes all capsules (including active) matching the run. Requires explicit confirmation param to prevent accidents.

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

### `restore` Tool

Recover soft-deleted capsules:

```json
{ "workspace": "default", "name": "auth" }
```

Currently: use export/import.

### Capsule Lineage (`based_on`)

Track which capsules informed the creation of a new capsule:

```json
capsule_store {
  "name": "implementation-plan",
  "based_on": [
    { "workspace": "default", "name": "research-findings" },
    { "workspace": "default", "name": "security-review" }
  ],
  "capsule_text": "..."
}
```

**Storage:** New `based_on` JSON column (nullable array of refs).

**Behavior:**
- `capsule_fetch` returns `based_on` array if present
- `capsule_list`/`capsule_inventory` include `based_on` in summaries
- No validation that referenced capsules exist (allows cross-workspace refs, deleted capsules)

**Use case:** Pipeline traceability. "This plan was based on that research." Audit trail for "why was this decision made?"

### `clone` Tool

Create new capsule based on existing:

```json
clone {
  "source": { "workspace": "default", "name": "oauth-research-v1" },
  "target": { "name": "oauth-research-v2", "run_id": "pr-200" },
  "set": { "phase": "research" }
}
```

**Parameters:**
- `source` — capsule ref (id OR workspace+name)
- `target` — new capsule ref (workspace defaults to source workspace)
- `set` — optional overrides for metadata fields (run_id, phase, role, tags)

**Behavior:**
- Copies `capsule_text` from source
- Sets `based_on: [source]` automatically (if lineage feature exists)
- Applies `set` overrides
- New capsule gets fresh id/timestamps

**Use case:** New workflow run that builds on prior research. Preserves lineage, avoids manual copy-paste.

### Real-Time Capsule Awareness (`watch`)

Long-poll tool that blocks until new capsules match a filter:

```json
watch {
  "workspace": "default",
  "run_id": "pr-123",
  "timeout_ms": 30000
}
```

**Returns** when:
- New capsule stored matching filter
- Existing capsule updated matching filter
- Timeout reached (returns empty)

**Output:**
```json
{
  "event": "created",  // or "updated", "timeout"
  "capsule": { /* summary if event != timeout */ }
}
```

**Use case:** Swarm workers waiting for dependencies without polling. Leader waiting for worker outputs. Pipeline stages waiting for upstream completion.

**Alternatives considered:**
- SSE/WebSocket (requires HTTP transport, more complex)
- Polling with `list` (works, but wasteful for long waits)

**Implementation:** SQLite doesn't support LISTEN/NOTIFY. Options:
1. Poll internally with backoff, return on change detection
2. File watcher on DB (platform-specific)
3. In-memory subscription registry (process-local only)

> **Decision:** Defer until polling proves insufficient. Most swarm patterns don't need sub-second awareness.

### `stats` Tool

Quick overview of capsule distribution without fetching content:

```json
stats { "workspace": "default" }
```

**Output:**
```json
{
  "capsule_count": 47,
  "total_chars": 142000,
  "by_phase": { "research": 12, "plan": 8, "implement": 20, "review": 7 },
  "by_role": { "security-reviewer": 5, "architect": 3 },
  "oldest": "2025-01-15T10:00:00Z",
  "newest": "2025-01-25T14:30:00Z"
}
```

**Parameters:**
- `workspace` — optional filter
- `run_id` — optional filter

**Use case:** Understand capsule usage patterns, detect bloat, see workflow distribution across phases/roles.

### `diff` Tool

Compare two capsules section-by-section:

```json
diff {
  "a": { "workspace": "default", "name": "plan-v1" },
  "b": { "workspace": "default", "name": "plan-v2" }
}
```

**Output:**
```json
{
  "sections": {
    "Objective": "unchanged",
    "Current status": "modified",
    "Decisions": "modified",
    "Next actions": "modified",
    "Key locations": "unchanged",
    "Open questions": "removed_content"
  },
  "summary": {
    "unchanged_sections": 2,
    "modified_sections": 3,
    "a_chars": 2400,
    "b_chars": 2850
  }
}
```

**Optional:** `include_diff: true` to include line-level diff for modified sections.

**Use case:** See what changed between pipeline stages, plan revisions, or before/after refactoring.

---

## Future Ideas

### Search Enhancements

#### `has_more` Only Mode

Replace COUNT query with `limit+1` fetch for faster pagination when exact totals aren't needed.

**Current behavior:** Returns exact `total` count (extra query).
**Enhancement:** Add `count_total: false` option to skip COUNT and use `limit+1` for `has_more` detection.

#### Safe Query Mode

Add `safe_query: true` option that auto-escapes special FTS5 characters in user input.

**Current behavior:** `query` is passed directly to FTS5 (supports full syntax but can error).
**Enhancement:** When `safe_query: true`, escape quotes/operators so plain text "just works".

#### Code-Aware Tokenization

Configure FTS5 tokenizer to handle code paths and symbols better:
- `tokenchars='_/-.'` to keep `src/auth/session.go` as searchable unit
- Trade-off: affects word boundaries for natural language

**Decision:** Defer until real usage patterns emerge. Document workaround: use AND (e.g., `src AND auth AND session`).

#### Configurable Snippet Length

Add `snippet_tokens` parameter (default: 64, max: 128) for FTS5.
Add `snippet_chars` parameter (default: 300, max: 500) for Go truncation.

### Semantic Search (Vector)

Embeddings-based similarity search for finding capsules by meaning rather than exact name.

**Use case:** "Find the capsule where I decided to use JWT" — without knowing the capsule name.

```
capsule_search { "query": "JWT authentication decision", "limit": 5 }
```

**Implementation options:**
- Local embeddings (e.g., `sentence-transformers` via Python sidecar, or Go-native like `go-embeddings`)
- SQLite `sqlite-vec` extension for vector storage
- Optional: remote embedding API (OpenAI, Voyage) with local vector cache

**Considerations:**
- Embedding on store vs lazy indexing
- Re-index on update
- Storage overhead (~1.5KB per capsule for 384-dim embeddings)
- Hybrid search: combine FTS5 keyword + semantic ranking

> **Implementation note: FTS5 first, vector later.**
>
> FTS5 gets 80% of search value with 10% of complexity:
> - No embedding model or vector storage needed
> - SQLite native, sub-millisecond queries, works offline
> - Capsules are semi-structured text — keywords like "Redis", "JWT", "Auth0" appear literally
>
> Vector search adds value when:
> - Users don't know exact terminology used in capsules
> - Need "similar to this" queries
> - Cross-run knowledge discovery with poor tagging discipline
> - Capsule count exceeds 500+ and keyword search isn't finding things
>
> **Recommended path:** Filters (done) → FTS5 (done) → Vector (when users hit the wall with keyword search)

### Versioning

Keep last N revisions of a capsule (extends `fetch`):

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

### MCP Server Graceful Shutdown (HTTP Transport)

**Stdio**: Already handled. `mcp-go`'s `ServeStdio()` catches SIGTERM/SIGINT and shuts down gracefully.

**HTTP**: Will need explicit shutdown handling when REST API is added. Use the transport server's `Shutdown(ctx)` (e.g., `server.NewSSEServer(...).Shutdown(ctx)` / `server.NewStreamableHTTPServer(...).Shutdown(ctx)`) with a timeout context.

### Import: Reuse ULID Entropy Source

`internal/ops/import.go` creates a new `ulid.Monotonic()` per call to `generateNewULID()`. Could be reused for minor efficiency gain.

### Import: Increase Scanner Buffer Size

`internal/ops/import.go` uses `bufio.NewScanner()` with default 64KB line limit. If `capsule_max_chars` is increased significantly (e.g., 50K+), large export records could be silently truncated. Consider using `scanner.Buffer()` to set explicit limit matching max capsule size + overhead.

### Export: Atomic Replace on Windows

`internal/ops/export.go` writes exports to a temp file and then finalizes via rename. On Windows, `os.Rename` fails if the destination already exists, and a delete+rename fallback is not atomic and can lose the existing file if the rename fails (locked file, AV, perms, etc.).

**Current behavior (safe):** if the destination exists on Windows, export fails and preserves the existing file.

**Desired behavior:** implement an atomic replace strategy on Windows (e.g., via Win32 `ReplaceFile` / `MoveFileEx` patterns) so exports can overwrite existing files without data-loss risk.

### Database: Dedicated Write Connection

For high-concurrency workloads, separate read and write connection pools to reduce lock contention without globally serializing reads.

**Current mitigation:** Config knobs `db_max_open_conns` and `db_max_idle_conns` allow users to tune pool behavior if they hit "database is locked" errors.

**Desired behavior:** Dedicated write connection (`SetMaxOpenConns(1)`) for writes only; default pool for concurrent reads. Avoids serializing read throughput while eliminating write contention.

### Bulk Update: `skip_unchanged` Option

`bulk_update` always bumps `updated_at` even if values are already at the target ("touched" semantics). This can cause churn in `list`/`latest` ordering.

**Current behavior (keep as default):** Consistent with single-item `update`, simple "rows matched" count semantics.

**Desired behavior:** Add opt-in `skip_unchanged: true` flag. When set, add WHERE predicates to exclude rows already at target values (`phase IS NULL OR phase != ?` for set, `phase IS NOT NULL` for clear). Tags comparison works via `tags_json IS NULL OR tags_json != ?` since JSON is deterministically serialized.

### Database: FTS Migration Lock Contention

FTS5 migration (`internal/db/db.go`) runs `INSERT INTO capsules_fts(capsules_fts) VALUES('rebuild')` without an explicit transaction or advisory lock. If multiple Moss processes start concurrently (e.g., parallel CI jobs, container restarts), they can fight over SQLite locks and repeatedly trigger rebuilds.

**Current behavior:** Works correctly for single-process use (typical). Concurrent starts may see transient lock errors and retry.

**Desired behavior:** Wrap migration in an exclusive transaction or use an advisory lock file to serialize concurrent startup migrations.

---

## Considered & Deferred

### Import/Export: Full openat() Path Traversal

For complete TOCTOU protection, each directory component could be opened with `openat(O_NOFOLLOW|O_DIRECTORY)` before opening the final file.

**Decision:** Instead of complex `openat()` traversal, we disallow subdirectories entirely—files must be directly in allowed directories. This eliminates the attack surface (no intermediate components to swap) while keeping the implementation simple. Combined with `O_NOFOLLOW` on the final component, this provides complete symlink protection.

### Content Lint Checks

Warn or reject if capsule sections lack actual content:
- ≥1 Next action item
- ≥1 Decision/constraint item
- ≥1 Key location pattern

**Decision:** Not in v1.0. Headers provide structure; content quality is agent's responsibility. Can add warnings in future if thin capsules become a problem.
