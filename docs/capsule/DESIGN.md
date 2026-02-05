# Capsule Design Spec (Moss v1)

## Summary

Capsule type spec for Moss: 16 MCP tools, CLI parity, capsule linting (6 sections), soft-delete, export/import, FTS5 full-text search, orchestration fields (`run_id`, `phase`, `role`).

---

## 0) One-line summary

A **Capsule** is a *strictly size-bounded* distilled handoff across Claude Code, Codex, etc. Moss provides MCP tools to manage capsules (**`capsule_store`/`capsule_fetch`/`capsule_update`/`capsule_delete`**), with **batch fetch (`capsule_fetch_many`)**, **`capsule_latest`/`capsule_list`**, **global `capsule_inventory`**, **`capsule_export`/`capsule_import`** for portability, **soft-delete** for safety, and **guardrails (lint + size limits)** to prevent both **context bloat** *and* **low-value capsules**.

**Related docs:**
- [README.md](../README.md) — Concepts and use cases
- [BACKLOG.md](BACKLOG.md) — Future features
- [RUNBOOK.md](RUNBOOK.md) — Build, configure, run, troubleshoot

---

# 1) Goals and non-goals

## Goals

* **Portable handoff:** Session A → capsule store → Session B across IDE/chat tools.
* **Orchestration-ready:** batch fetch, deterministic composition, and run-scoping for multi-agent workflows.
* **Low-bloat by design:** store/load only a **capsule** (distilled state). No full history by default.
* **High value density:** enforce a **capsule quality bar** (lint rules) so what you fetch is useful.
* **Explicit only:** no auto-save or auto-load.
* **Local-first:** single local process + local DB file.
* **Fast & simple:** minimal actions; no heavy infra.
* **Human-friendly addressing:** stable `name` handles so you can "fetch `auth`".
* **Discoverability:** one call to see **everything stored** (IDs + names) without loading capsule text.
* **Cross-client traceability:** store `source` (claude-code, codex, etc.).

## Non-goals (not in v1.0)

* Server-side summarization (the user/agent distills; the system stores)
* Repo indexing / RAG
* Multi-user / hosted SaaS / roles
* Integrations (GitHub, Notion, etc.)
* Semantic/vector search (FTS5 keyword search available; embeddings deferred)
* Rich evidence store (optional later)
* Editing capsule addressing keys (name/workspace changes)

---

# 2) Core concept: Capsule (compacted context)

A **Capsule** is the *current distilled state* intended to be pasted into a fresh chat with minimal bloat.

## Capsule contract (required structure)

A capsule must include 6 required sections (see §3.2 for list and validation).

Optional: **1–3 tiny snippets** only if critical (receipts, not transcript).

## Hard low-bloat guardrail

* Enforce `capsule_max_chars`
* If too big: **reject** (default)

---

# 3) Quality: “Capsule must not lose value”

Capsule quality is enforced via **workflow + validation**, without needing LLM intelligence.

## 3.1 Distill-before-save workflow (recommended)

Before `capsule_store` or `capsule_update`, the client (you or agent) produces a capsule:

> “Distill into a capsule under X chars. State not story. Must include Objective, Current status, Decisions/Constraints, Next actions, Key locations, Open questions/Risks. Max 1–3 tiny snippets only if critical. If too long, compress—do not omit decisions or next actions.”

The system stores the result.

## 3.2 Capsule linter (non-LLM)

On `capsule_store` and on `capsule_update` (when capsule content changes), the system validates minimum structure.

### Lint rules

| Rule | Error |
|------|-------|
| Size ≤ `capsule_max_chars` | 413 CAPSULE_TOO_LARGE |
| Required section headers present | 422 CAPSULE_TOO_THIN |

Required sections (detected by markdown header or JSON key):
1. Objective
2. Current status
3. Decisions / constraints
4. Next actions
5. Key locations
6. Open questions / risks

Content within sections is not validated — agents are trusted to provide useful content.

`allow_thin: true` bypasses section check (escape hatch).

If lint fails: **422 CAPSULE_TOO_THIN** with details about what's missing.

This prevents saving "fluffy" capsules that don't rehydrate well.

---

# 4) Naming model + normalization

Each capsule has:

* `id` (immutable, system-generated; recommended **ULID**)
* `name` (optional; stable handle unique within workspace)

**ID generation (ULID):**
* Timestamp-based prefix (sortable by creation time)
* Random suffix (collision-resistant)
* 26 characters, base32 encoded (e.g., `01ARZ3NDEKTSV4RRFFQ69G5FAV`)

## 4.1 Default workspace

* If `workspace` omitted on actions that need it → use `"default"`

## 4.2 Normalized addressing (collision-proof)

To avoid `Auth` vs `auth` ambiguity, Moss stores raw + normalized:
* `workspace_raw`, `workspace_norm`
* `name_raw`, `name_norm`

Normalization rules:
1. Trim leading/trailing whitespace
2. Lowercase
3. Collapse internal whitespace to single spaces

Examples:
```
"StartupA"        → "startupa"
"  My Project  "  → "my project"
"LOUD_NAME"       → "loud_name"
```

Display uses raw; lookup uses normalized.

## 4.3 Deterministic resolution rule

For `capsule_fetch`/`capsule_update`/`capsule_delete`:

* Must specify **exactly one** addressing mode:

  * `id`
  * OR (`workspace` + `name`)
* If both provided → **400 AMBIGUOUS_ADDRESSING**

Uniqueness enforced on `(workspace_norm, name_norm)` when name is present.

---

# 5) External interface (MCP-first)

## 5.1 MCP tools

**Separate tools per operation** (MCP-idiomatic):

| Tool | Description |
|------|-------------|
| `capsule_store` | Create new capsule (supports upsert via `mode`) |
| `capsule_fetch` | Read capsule by id OR by name |
| `capsule_fetch_many` | Batch fetch multiple capsules |
| `capsule_update` | Update capsule content/metadata |
| `capsule_delete` | Soft delete (recoverable) |
| `capsule_latest` | Most recent capsule in workspace |
| `capsule_list` | List capsule summaries in workspace |
| `capsule_inventory` | List capsule summaries globally |
| `capsule_search` | Full-text search across capsules |
| `capsule_export` | JSONL backup |
| `capsule_import` | JSONL restore |
| `capsule_purge` | Permanently delete soft-deleted |
| `capsule_bulk_delete` | Soft-delete multiple capsules by filter |
| `capsule_bulk_update` | Update metadata on multiple capsules |
| `capsule_compose` | Assemble multiple capsules into bundle |
| `capsule_append` | Append content to a specific section |

Each tool has a focused schema — no `action` dispatch needed.

### Output bloat rules

* `capsule_list` **never** returns `capsule_text`
* `capsule_inventory` **never** returns `capsule_text`
* `capsule_latest` returns **summary by default**; requires `include_text:true` for full capsule
* `capsule_fetch` returns full capsule text (explicit load operation)

  * Optional: support `include_text:false` as a “peek” without bloat

---

# 6) MCP tool behaviors

Tool schemas are defined in code (`internal/mcp/tools.go`). This section documents key behaviors.

## 6.1 `capsule_store`

**Required:** `capsule_text`

**Optional:** `workspace` (default: "default"), `name`, `title`, `tags`, `source`, `run_id`, `phase`, `role`, `mode` ("error"|"replace"), `allow_thin`

**Orchestration fields**: `run_id`, `phase`, `role` enable multi-agent workflow scoping (e.g., `run_id: "pr-review-abc123"`, `phase: "design"`, `role: "design-intent"`).

**Behaviors:**
- `mode:"error"` + name collision → **409 NAME_ALREADY_EXISTS**
- `mode:"replace"` + name collision → overwrite (preserve `id`)
- Too large → **413 CAPSULE_TOO_LARGE**
- Lint fails → **422 CAPSULE_TOO_THIN**
- Soft-deleted capsules don't participate in name uniqueness

**Output:** `{ id, fetch_key }` — `fetch_key` provides ready-to-use metadata for Claude Code Tasks integration.

---

## 6.2 `capsule_fetch`

**Addressing:** `id` OR (`workspace` + `name`) — not both

**Optional:** `include_deleted`, `include_text` (default: true)

**Behaviors:**
- Default excludes soft-deleted → **404 NOT_FOUND**
- `include_deleted:true` makes soft-deleted visible
- `include_text:false` returns summary only (peek)

---

## 6.3 `capsule_fetch_many`

Batch fetch multiple capsules in a single call. Useful for fan-in patterns where an orchestrator gathers results from parallel workers.

**Required:** `items` array (max 50, each addressed by `id` OR `workspace`+`name`)

**Optional:** `include_text` (default: true), `include_deleted`

**Behaviors:**
- Partial success: found items in `items` array, failures in `errors` array
- Each item includes `fetch_key` for subsequent operations
- Mixed addressing allowed (some by id, some by name)
- Too many items (>50) → **400 INVALID_REQUEST**

**Output:**
```json
{
  "items": [
    {
      "id": "01J...",
      "workspace": "default",
      "name": "auth",
      "title": "Auth Implementation",
      "capsule_text": "## Objective\n...",
      "capsule_chars": 2400,
      "tokens_estimate": 600,
      "fetch_key": { "moss_capsule": "auth", "moss_workspace": "default" }
    }
  ],
  "errors": [
    {
      "ref": { "workspace": "default", "name": "missing" },
      "code": "NOT_FOUND",
      "message": "capsule not found: default/missing"
    }
  ]
}
```

---

## 6.4 `capsule_update`

**Addressing:** `id` OR (`workspace` + `name`)

**Editable:** `capsule_text`, `title`, `tags`, `source`, `run_id`, `phase`, `role`

**Immutable:** `id`, `workspace`, `name` — to "rename", delete and re-store

**Behaviors:**
- Missing → **404 NOT_FOUND**
- Too large → **413 CAPSULE_TOO_LARGE**
- Lint fails → **422 CAPSULE_TOO_THIN**
- No fields → **400 INVALID_REQUEST**

---

## 6.5 `capsule_delete`

Soft-deletes by setting `deleted_at` and bumping `updated_at` to reflect deletion in "latest" ordering. Capsule recoverable via `include_deleted` or export/import.

---

## 6.6 `capsule_latest`

Returns most recent capsule in workspace.

**Optional:** `include_text` (default: false), `include_deleted`, `run_id`, `phase`, `role`

**Filters**: Use `run_id`/`phase`/`role` to get "latest design capsule from this run".

---

## 6.7 `capsule_list`

List summaries in workspace. **Never returns `capsule_text`.**

**Optional:** `limit` (default: 20, max: 100), `offset`, `include_deleted`, `run_id`, `phase`, `role`

**Filters**: `run_id`/`phase`/`role` narrow results to capsules in specific workflow contexts.

---

## 6.8 `capsule_inventory`

Global list across all workspaces. **Never returns `capsule_text`.**

**Optional filters:** `workspace`, `tag`, `name_prefix`, `run_id`, `phase`, `role`, `include_deleted`, `limit` (default: 100, max: 500), `offset`

---

## 6.9 `capsule_search`

Full-text search across capsules using SQLite FTS5. Returns results ranked by relevance with match snippets.

**Required:** `query` (max 1000 chars)

**Optional filters:** `workspace`, `tag`, `run_id`, `phase`, `role`, `include_deleted`, `limit` (default: 20, max: 100), `offset`

**Query syntax (FTS5):**
- Simple words: `authentication` (matches anywhere)
- Phrases: `"user authentication"` (exact match)
- Prefix: `auth*` (matches auth, authentication, authorize...)
- Boolean: `JWT OR OAuth`, `Redis AND cache`, `NOT deprecated`

**Behaviors:**
- Title matches weighted 5x higher than body (BM25 ranking)
- Returns `snippet` field with match context (~300 chars, `<b>` highlights, HTML-escaped user content)
- Empty results returns `[]`, not error
- Query > 1000 chars → **400 INVALID_REQUEST**
- Invalid FTS5 syntax → **400 INVALID_REQUEST**

**Output:**
```json
{
  "items": [
    {
      "id": "01J...",
      "workspace": "default",
      "name": "auth",
      "snippet": "...using <b>JWT</b> for authentication...",
      "fetch_key": { "moss_capsule": "auth", "moss_workspace": "default" }
    }
  ],
  "pagination": { "limit": 20, "offset": 0, "has_more": false, "total": 1 },
  "sort": "relevance"
}
```

---

## 6.10 `capsule_export`

Export to JSONL file.

**Optional:** `path` (default: `~/.moss/exports/<workspace>-<timestamp>.jsonl`), `workspace`, `include_deleted`

---

## 6.11 `capsule_import`

Import from JSONL file.

**Required:** `path`

**Optional:** `mode` — "error" (default, atomic fail on collision), "replace" (overwrite), "rename" (auto-suffix)

**Important:** `*_norm` fields are recomputed on import; don't trust incoming values.

---

## 6.12 `capsule_purge`

Permanently delete soft-deleted capsules.

**Optional:** `workspace`, `older_than_days`

---

## 6.13 `capsule_compose`

Assemble multiple capsules into a single bundle. All-or-nothing: fails if any capsule is missing.

**Required:** `items` array (each addressed by `id` OR `workspace`+`name`)

**Optional:** `format` ("markdown"|"json", default: "markdown"), `store_as` (persist result)

**Format options:**
- `markdown`: `## <display_name>\n\n<text>\n\n---\n\n...`
- `json`: `{ "parts": [{ "id", "workspace", "name", "display_name", "text", "chars" }, ...] }`

**Display name:** computed as title > name > id (always present)

**Behaviors:**
- All-or-nothing: if any item missing → **404 NOT_FOUND**
- Too large → **413 COMPOSE_TOO_LARGE**
- `format:"json"` + `store_as` → **400 INVALID_REQUEST** (JSON lacks section headers)
- If `store_as` provided: lint + store via `capsule_store` operation
- `store_as.name` required when `store_as` provided

**Output:**
```json
{
  "bundle_text": "## cap1\n\n...\n\n---\n\n## cap2\n\n...",
  "bundle_chars": 3241,
  "parts_count": 2,
  "stored": { "id": "01J...", "fetch_key": {...} }  // only if store_as
}
```

---

## 6.14 `capsule_bulk_delete`

Soft-delete multiple active capsules matching filters. Requires at least one filter (safety guard). Only targets active capsules (`deleted_at IS NULL` is hardcoded).

**Optional filters:** `workspace`, `tag`, `name_prefix`, `run_id`, `phase`, `role`

**Safety:** At least one filter must be provided and non-empty after normalization. Calling with no filters or only whitespace filters → **400 INVALID_REQUEST**.

**Behaviors:**
- Filters use AND semantics (all provided filters must match)
- Already soft-deleted capsules are not affected
- Returns count of 0 with no error if no capsules match
- Single atomic UPDATE query (no explicit transaction needed)

**Output:**
```json
{
  "deleted": 3,
  "message": "Soft-deleted 3 capsules matching workspace=\"project\", tag=\"stale\""
}
```

---

## 6.15 `capsule_bulk_update`

Update metadata (phase, role, tags) on multiple active capsules matching filters. Requires at least one filter AND at least one update field (safety guard). Only targets active capsules (`deleted_at IS NULL` is hardcoded).

**Optional filters:** `workspace`, `tag`, `name_prefix`, `run_id`, `phase`, `role`

**Update fields:** `set_phase`, `set_role`, `set_tags` (prefixed with `set_` to distinguish from filter fields)

**Safety:**
- At least one filter must be provided and non-empty after normalization.
- At least one update field must be provided (empty values are allowed to support explicit clearing).
- Calling with no filters or no update fields → **400 INVALID_REQUEST**.

**Empty string semantics:** Empty string `""` means "clear the field" (set to NULL). This allows intentional field clearing. An empty `set_tags` array clears all tags.

**Behaviors:**
- Filters use AND semantics (all provided filters must match)
- Tags are replaced entirely (not merged)
- Always updates `updated_at` timestamp
- Already soft-deleted capsules are not affected
- Returns count of 0 with no error if no capsules match
- Single atomic UPDATE query (no explicit transaction needed)

**Output:**
```json
{
  "updated": 5,
  "message": "Updated 5 capsules matching workspace=\"project\"; set phase=\"archived\""
}
```

---

## 6.16 `capsule_append`

Append content to a specific section of a capsule without rewriting the entire document. Useful for accumulating history in workflows (design reviews, verification attempts, decisions).

**Addressing:** `id` OR (`workspace` + `name`)

**Required:** `section`, `content`

**Section matching:**
- Exact header name match (case-insensitive)
- No synonym resolution — use the header as written (e.g., `## Status` → `"Status"`)
- Error message lists available sections if not found

**Placeholder handling:** If section content is only a placeholder (`(pending)`, `TBD`, `N/A`, `-`, `none`, etc.), replaces it entirely. Otherwise appends after existing content with blank line separator.

**Behaviors:**
- Markdown format required → **400 INVALID_REQUEST** if no sections found (e.g., JSON capsule)
- Section not found → **400 INVALID_REQUEST** with section name
- Empty/whitespace-only content → **400 INVALID_REQUEST**
- Result exceeds size limit → **413 CAPSULE_TOO_LARGE**
- No section lint (append may target custom sections not in required 6)
- Assumes LF line endings; CRLF files may parse incorrectly

**Output:**
```json
{
  "id": "01ABC...",
  "fetch_key": { "moss_capsule": "feat-auth", "moss_workspace": "feat" },
  "section_hit": "## Design Reviews",
  "replaced": false
}
```

`fetch_key` is omitted if capsule is unnamed (addressed by ID only). `replaced` is true if placeholder content was replaced, false if content was appended.

---

# 7) System architecture (minimal)

1. **Moss service** (single local process)

   * MCP handler (primary)
   * CLI for debugging (secondary)
   * normalization + validation + lint
   * persistence (SQLite)

2. **SQLite DB**

   * file: `~/.moss/moss.db`
   * file perms: recommend `0600`
   * directory perms: `~/.moss/` and `~/.moss/exports/` should be `0700`

3. **SQLite configuration**

   * Enable WAL mode + `busy_timeout` to avoid "database is locked" when MCP + CLI overlap
   * Use `PRAGMA user_version` for schema migrations (bump on schema changes)

No workers, queues, vector DB.

## 7.1 Context propagation and cancellation

All 16 ops functions accept `context.Context` as their first parameter. Context originates from the MCP request handler and propagates through the ops layer into database calls:

```
MCP handler → ops.Operation(ctx, ...) → db.Query(ctx, tx, ...)
```

**Cancellable operations:** Four loop-based operations check `ctx.Done()` on each iteration, enabling early abort for long-running batches:

| Operation | Cancellation point |
|-----------|-------------------|
| `capsule_fetch_many` | Before each item fetch |
| `capsule_compose` | Before each item fetch |
| `capsule_export` | Before each row write |
| `capsule_import` | Before each record insert (all 3 modes) |

**On cancellation:**
- The loop exits immediately and returns a **499 CANCELLED** error with the operation name (e.g., `"import cancelled"`)
- `capsule_import` runs within a transaction — cancellation triggers rollback with no partial writes
- `capsule_export` writes to a temp file and finalizes via atomic rename; failures clean up the temp file and preserve any existing destination file

**Single-query operations** (`capsule_store`, `capsule_fetch`, `capsule_update`, `capsule_delete`, `capsule_list`, `capsule_latest`, `capsule_inventory`, `capsule_purge`, `capsule_bulk_delete`, `capsule_bulk_update`, `capsule_append`) pass context to database calls but do not have explicit `ctx.Done()` loop checks, as they execute a bounded number of queries.

---

# 8) Runtime configuration

## 8.1) Configuration file

Moss loads config from two locations (merged):

| Location | Scope | Priority |
|----------|-------|----------|
| `~/.moss/config.json` | Global (user) | Lower |
| `.moss/config.json` | Repo (project) | Higher |

**Repo config discovery:** Moss walks upward from the current working directory to find the nearest `.moss/config.json`. This means running from a subdirectory still finds the repo root config.

**Merge behavior:**
- Scalars: repo overrides global (if non-zero)
- Booleans: OR (either true → true)
- Arrays: merged and deduplicated

```json
{
  "capsule_max_chars": 12000,
  "allowed_paths": ["/tmp/my-exports"],
  "allow_unsafe_paths": false,
  "db_max_open_conns": 0,
  "db_max_idle_conns": 0,
  "disabled_tools": [],
  "disabled_types": []
}
```

**Defaults** (applied if config missing or field omitted):

| Field | Default | Description |
|-------|---------|-------------|
| `capsule_max_chars` | 12000 | Max characters per capsule (~3k tokens) |
| `allowed_paths` | `[]` | Additional directories allowed for import/export |
| `allow_unsafe_paths` | `false` | Bypass directory restrictions for import/export (symlink checks still apply) |
| `db_max_open_conns` | 0 | Max open DB connections (0 = unlimited; set to 1 if you hit "database is locked") |
| `db_max_idle_conns` | 0 | Max idle DB connections (0 = default; typically match `db_max_open_conns`) |
| `disabled_tools` | `[]` | MCP tool names to exclude from registration (see §5.1 for tool list) |
| `disabled_types` | `[]` | Type names to disable entirely (e.g., `["capsule"]` disables all capsule tools) |

### Import/export path security

By default, `capsule_export` and `capsule_import` operations are restricted to `~/.moss/exports/`. This prevents accidental writes to sensitive locations and limits exposure from symlink attacks.

**Restrictions enforced:**
- `.jsonl` extension required
- Directory traversal (`..`) rejected
- Subdirectories not allowed: files must be directly in an allowed directory (prevents TOCTOU attacks on directory components)
- Symlink files rejected (uses `O_NOFOLLOW` where supported) to prevent symlink-target reads/writes
- Parent directory symlinks rejected (defense-in-depth)
- Workspace names sanitized: path separators and `..` stripped from default export filenames
- Paths must be within `~/.moss/exports/` or a directory in `allowed_paths`

**Configuration options:**
- `allowed_paths`: Add directories to the allowlist (absolute paths only)
- `allow_unsafe_paths: true`: Bypass directory restrictions (escape hatch for advanced users; symlink restrictions, `.jsonl` extension, and traversal checks still apply)

---

## 8.2) CLI

CLI mirrors MCP operations for debugging and scripting. See [RUNBOOK.md](RUNBOOK.md) for commands, flags, and examples.

---

# 9) Storage design (SQLite)

## Table: `capsules`

* `id TEXT PRIMARY KEY`
* `workspace_raw TEXT NOT NULL`
* `workspace_norm TEXT NOT NULL`
* `name_raw TEXT NULL`
* `name_norm TEXT NULL`
* `title TEXT NULL`
* `capsule_text TEXT NOT NULL`
* `capsule_chars INTEGER NOT NULL`
* `tokens_estimate INTEGER NOT NULL` — heuristic: word count × 1.3
* `tags_json TEXT NULL`
* `source TEXT NULL`
* `run_id TEXT NULL` — orchestration run identifier
* `phase TEXT NULL` — workflow phase
* `role TEXT NULL` — agent role
* `created_at INTEGER NOT NULL`
* `updated_at INTEGER NOT NULL`
* `deleted_at INTEGER NULL` — soft delete timestamp (null = active)

## Indexes / constraints

* Unique name handles: `UNIQUE(workspace_norm, name_norm)` excluding soft-deleted
* Fast list/latest: `INDEX(workspace_norm, updated_at DESC)` excluding soft-deleted
* Orchestration queries: `INDEX(run_id, phase, role)` excluding soft-deleted, partial (run_id IS NOT NULL)

---

# 10) Validation & constraints

## Required fields

* `workspace` required for `capsule_store`/`capsule_list`/`capsule_latest` and name addressing (defaults to `"default"` if omitted)
* `capsule_text` required for `capsule_store`; optional for `capsule_update` (lint runs only if provided)
* For `capsule_fetch`/`capsule_update`/`capsule_delete`: either `id` OR (`workspace`+`name`), not both

## Hard limits

* Default `capsule_max_chars`: **12,000** (configurable in `~/.moss/config.json`)
* `len(capsule_text) <= capsule_max_chars` else **413 CAPSULE_TOO_LARGE**
* Import JSONL file size is capped (default: **25MB**) else **413 FILE_TOO_LARGE**

## Mode validation

* `capsule_store.mode` must be `"error"` (default) or `"replace"`
* `mode:"replace"` overwrites existing active capsule; creates new if none exists
* Soft-delete interaction: `mode:"replace"` targets active rows only (`deleted_at IS NULL`); if none exists, it creates a new capsule rather than reviving a deleted one.

## Lint

* Capsules must contain required section headers else **422 CAPSULE_TOO_THIN**
* `allow_thin: true` bypasses section check

See section 3.2 for lint rules, section 4.2 for normalization.

---

# 11) Error contract

| Code | Status | When |
|------|--------|------|
| AMBIGUOUS_ADDRESSING | 400 | Both `id` and `name` provided |
| INVALID_REQUEST | 400 | Invalid fields or malformed request |
| NOT_FOUND | 404 | Capsule doesn't exist (or is soft-deleted) |
| NAME_ALREADY_EXISTS | 409 | Name collision on capsule_store with mode:"error" |
| CAPSULE_TOO_LARGE | 413 | Exceeds `capsule_max_chars` |
| FILE_TOO_LARGE | 413 | Import file exceeds max size limit |
| COMPOSE_TOO_LARGE | 413 | Composed bundle exceeds `capsule_max_chars` |
| CAPSULE_TOO_THIN | 422 | Missing required sections |
| CANCELLED | 499 | Context cancelled during long-running operation |
| INTERNAL | 500 | Unexpected error |

Response format:

```json
{
  "error": {
    "code": "CAPSULE_TOO_THIN",
    "message": "Capsule missing required sections",
    "status": 422,
    "details": {
      "missing": ["Decisions", "Key locations"]
    }
  }
}
```

The `details` field varies by error code (e.g., `max_chars`/`actual_chars` for CAPSULE_TOO_LARGE; `max_bytes`/`actual_bytes` for FILE_TOO_LARGE).

---

# 12) Operational flows (value-preserving)

## Flow A — Save (Session A → Moss)

1. Distill to capsule under limit using the capsule contract
2. Call `capsule_store`
3. Moss validates size + lint, saves, returns id

## Flow B — Continue (Moss → Session B)

1. Call `capsule_fetch` by name or `capsule_latest include_text:true`
2. Paste `capsule_text` into new session
3. Continue work

## Flow C — Browse (no bloat)

* `capsule_inventory` to see everything stored (no text)
* `capsule_list` for workspace
* `capsule_latest` summary, then `capsule_fetch` to load

## Flow D — Refresh (rolling state)

* Distill updated state → `capsule_update`
* Capsule stays evergreen

## Flow E — Cleanup

* `capsule_delete`

## Flow F — Backup & Restore

```
1. Export all capsules:
   capsule_export { path: "~/.moss/exports/backup.jsonl" }

2. Export specific workspace:
   capsule_export { path: "~/.moss/exports/projectA.jsonl", workspace: "projectA" }

3. Restore to new machine:
   capsule_import { path: "~/.moss/exports/backup.jsonl", mode: "error" }

4. Merge with existing:
   capsule_import { path: "~/.moss/exports/backup.jsonl", mode: "replace" }
```
