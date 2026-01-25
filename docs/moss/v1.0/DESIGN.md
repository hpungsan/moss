# [MOSS] Moss v1.0 Design Spec

## 0) One-line summary

Moss is a **local "context capsule" store** that lets you **store/fetch/update/delete** a *strictly size-bounded* distilled handoff across Claude Code, Codex, etc., with **batch fetch (`fetch_many`)**, **upsert mode**, **latest/list**, **global inventory**, **human-friendly `name` handles**, **export/import** for portability, **soft-delete** for safety, and **guardrails (capsule lint + size limits)** to prevent both **context bloat** *and* **low-value capsules**.

**Related docs:**
- [OVERVIEW.md](../OVERVIEW.md) — Concepts and use cases
- [BACKLOG.md](BACKLOG.md) — Post-v1.0 features

---

# 1) Goals and non-goals

## Goals

* **Portable handoff:** Session A → Moss → Session B across IDE/chat tools.
* **Orchestration-ready:** batch fetch, deterministic composition, and run-scoping for multi-agent workflows.
* **Low-bloat by design:** store/load only a **capsule** (distilled state). No full history by default.
* **High value density:** enforce a **capsule quality bar** (lint rules) so what you fetch is useful.
* **Explicit only:** Moss never auto-saves or auto-loads.
* **Local-first:** single local process + local DB file.
* **Fast & simple:** minimal actions; no heavy infra.
* **Human-friendly addressing:** stable `name` handles so you can "fetch `auth`".
* **Discoverability:** one call to see **everything stored** (IDs + names) without loading capsule text.
* **Cross-client traceability:** store `source` (claude-code, codex, etc.).

## Non-goals (not in v1.0)

* Moss-generated summarization (Moss doesn't "think"; user/agent distills)
* Repo indexing / RAG
* Multi-user / hosted SaaS / roles
* Integrations (GitHub, Notion, etc.)
* Semantic search (optional later)
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

Moss ensures capsule usefulness via **workflow + validation**, without needing LLM intelligence.

## 3.1 Distill-before-save workflow (recommended)

Before `store` or `update`, the client (you or agent) produces a capsule:

> “Distill into Moss capsule under X chars. State not story. Must include Objective, Current status, Decisions/Constraints, Next actions, Key locations, Open questions/Risks. Max 1–3 tiny snippets only if critical. If too long, compress—do not omit decisions or next actions.”

Moss stores the result.

## 3.2 Moss capsule linter (non-LLM)

On `store` and on `update` (when capsule content changes), Moss validates minimum structure.

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

If lint fails: **422 CAPSULE_TOO_THIN** with details about what’s missing.

This prevents saving “fluffy” capsules that don’t rehydrate well.

### Section detection (implementation)

For lint validation, detect sections via three formats. All detection is **case-insensitive**.

**Formats to detect:**
1. Markdown headers: `## Objective`, `# Current Status`, etc.
2. Colon-style: `Objective:`, `Current status:`, etc.
3. JSON keys: if `capsule_text` is valid JSON, check for matching keys

**Section synonyms:**

| Canonical | Synonyms (any match counts) |
|-----------|----------------------------|
| Objective | Goal, Purpose |
| Current status | Status, State, Where we are |
| Decisions | Decisions / constraints, Decisions/constraints, Constraints, Choices |
| Next actions | Next steps, Action items, TODO, Tasks |
| Key locations | Locations, Files, Paths, References |
| Open questions | Open questions / risks, Open questions/risks, Questions, Risks, Unknowns |

Match any synonym in any format → section is present.

---

# 4) Naming model + normalization

Each capsule has:

* `id` (immutable, system-generated; recommended **ULID**)
* `name` (optional; stable handle unique within workspace)

**ID generation (ULID):**
* Timestamp-based prefix (sortable by creation time)
* Random suffix (collision-resistant)
* 26 characters, base32 encoded

```go
import "github.com/oklog/ulid/v2"

id := ulid.Make().String() // "01ARZ3NDEKTSV4RRFFQ69G5FAV"
```

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

For `fetch/update/delete`:

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
| `moss.store` | Create new capsule (supports upsert via `mode`) |
| `moss.fetch` | Read capsule by id OR by name |
| `moss.fetch_many` | Batch fetch multiple capsules |
| `moss.update` | Update capsule content/metadata |
| `moss.delete` | Soft delete (recoverable) |
| `moss.latest` | Most recent capsule in workspace |
| `moss.list` | List capsule summaries in workspace |
| `moss.inventory` | List capsule summaries globally |
| `moss.export` | JSONL backup |
| `moss.import` | JSONL restore |
| `moss.purge` | Permanently delete soft-deleted |

Each tool has a focused schema — no `action` dispatch needed.

### Output bloat rules

* `list` **never** returns `capsule_text`
* `inventory` **never** returns `capsule_text`
* `latest` returns **summary by default**; requires `include_text:true` for full capsule
* `fetch` returns full capsule text (explicit load primitive)

  * Optional: support `include_text:false` as a “peek” without bloat

---

# 6) MCP tool schemas

## 6.1 `moss.store`

```json
{
  "workspace": "startupA",
  "name": "auth",
  "title": "Auth + sessions",
  "capsule_text": "Objective: ...\nCurrent status: ...\nDecisions: ...\nNext actions: ...\nKey locations: ...\nOpen questions: ...",
  "tags": ["auth","sessions"],
  "source": "claude-code",
  "mode": "error",
  "allow_thin": false
}
```

**Required fields:**

* `capsule_text` — the capsule content

**Optional fields:**

* `name` — human-friendly handle. If omitted, capsule is only addressable by `id`. Name uniqueness rules only apply when name is present.
* `workspace` — namespace (default: `"default"`)
* `title` — display title (default: same as `name`, or null if unnamed)
* `tags` — array of tags
* `source` — source identifier (e.g., `"claude-code"`, `"codex"`)
* `mode` — `"error"` (default) or `"replace"`. If `"replace"` and an **active** capsule exists for `(workspace_norm,name_norm)`, overwrites it; otherwise creates a new capsule.
* `allow_thin` — `false` (default). If `true`, bypasses section presence check (escape hatch for edge cases).

Behavior:

* If `mode:"error"` (default) and an **active** capsule with `(workspace_norm,name_norm)` exists (`deleted_at IS NULL`) → **409 NAME_ALREADY_EXISTS**
* If `mode:"replace"` and an **active** capsule with `(workspace_norm,name_norm)` exists (`deleted_at IS NULL`) → overwrite (update `updated_at`, preserve `id`)
* If `mode:"replace"` and no active capsule exists for `(workspace_norm,name_norm)` → create a new capsule
* If too large → **413 CAPSULE_TOO_LARGE**
* If lint fails → **422 CAPSULE_TOO_THIN**

Soft-deleted capsules do not participate in name uniqueness.

Output:

```json
{
  "id": "01J...ULID",
  "task_link": {
    "moss_capsule": "auth",
    "moss_workspace": "startupA"
  }
}
```

The `task_link` field provides a ready-to-use blob for Claude Code Task metadata integration. Agents can copy this directly into a task's `metadata` field to link the task to this capsule.

**task_link for unnamed capsules:** If `name` is null, `task_link` uses the ID instead:
```json
{
  "id": "01J...ULID",
  "task_link": {
    "moss_id": "01J...ULID"
  }
}
```

---

## 6.2 `moss.fetch`

By id:

```json
{ "id": "01J...ULID" }
```

By name:

```json
{ "workspace": "startupA", "name": "auth" }
```

Include soft-deleted:

```json
{ "workspace": "startupA", "name": "auth", "include_deleted": true }
```

Optional peek:

```json
{ "workspace": "startupA", "name": "auth", "include_text": false }
```

Behavior:

* Default excludes soft-deleted capsules (`deleted_at IS NULL`) and returns **404 NOT_FOUND** if only a deleted capsule exists.
* `include_deleted:true` makes soft-deleted capsules visible for this action.
* If both an active and soft-deleted capsule exist for the same `(workspace_norm,name_norm)`, name-based fetch returns the **active** capsule; fetching a deleted variant requires addressing by `id` (or using `export include_deleted:true` to locate it).

Output (full capsule when include_text true/default):

```json
{
  "id": "01J...ULID",
  "workspace": "startupA",
  "workspace_norm": "startupa",
  "name": "auth",
  "name_norm": "auth",
  "title": "Auth + sessions",
  "capsule_text": "...",
  "capsule_chars": 1823,
  "tokens_estimate": 456,
  "tags": ["auth","sessions"],
  "source": "claude-code",
  "created_at": 1737260000,
  "updated_at": 1737260500,
  "task_link": {
    "moss_capsule": "auth",
    "moss_workspace": "startupA"
  }
}
```

---

## 6.3 `moss.fetch_many`
Batch fetch multiple capsules in one call.

```json
{
  "items": [
    { "workspace": "phinn", "name": "pr-123-base" },
    { "workspace": "phinn", "name": "pr-123-design" },
    { "id": "01J...ULID" }
  ],
  "include_text": true,
  "include_deleted": false
}
```

Behavior:

* Each item can be addressed by `id` OR by `(workspace, name)`
* Partial success allowed — missing items returned in `errors` array
* If `include_text: false`, returns summaries only
* If `include_deleted: true`, soft-deleted capsules are eligible to be returned
* Each returned item includes `task_link` for client integrations

Output:

```json
{
  "items": [
    { "id": "01J...", "workspace": "phinn", "name": "pr-123-base", "capsule_text": "...", ... },
    { "id": "01J...", "workspace": "phinn", "name": "pr-123-design", "capsule_text": "...", ... }
  ],
  "errors": [
    { "ref": { "id": "01J...ULID" }, "code": "NOT_FOUND", "message": "Capsule not found" }
  ]
}
```

---

## 6.4 `moss.update`

Updates capsule content and metadata. Cannot change addressing keys (name/workspace).

By name:

```json
{
  "workspace": "startupA",
  "name": "auth",
  "capsule_text": "Objective: ...\nCurrent status: ...\nDecisions: ...\nNext actions: ...\nKey locations: ...\nOpen questions: ...",
  "title": "Auth + sessions (updated)",
  "tags": ["auth", "sessions", "jwt"],
  "source": "codex",
  "allow_thin": false
}
```

By id:

```json
{ "id": "01J...ULID", "capsule_text": "...", "title": "Fixed title", "source": "codex" }
```

**Addressing fields** (used to identify the capsule, not modified):

* `id` — address by ID
* `workspace` + `name` — address by name

Provide exactly one addressing mode. These fields route the request; they are **not** updated.

**Editable fields:**

* `capsule_text` — capsule content (triggers lint validation)
* `title` — human-readable title
* `tags` — tag array
* `source` — source identifier

**Optional flags:**

* `allow_thin` — `false` (default). If `true`, bypasses section presence check.

**Note:** A capsule's `name`, `workspace`, and `id` cannot be changed after creation. To "rename" a capsule, delete and re-store.

Behavior:

* If target missing → **404 NOT_FOUND**
* If too large → **413 CAPSULE_TOO_LARGE**
* If lint fails → **422 CAPSULE_TOO_THIN**
* If no editable fields provided → **400 INVALID_REQUEST** ("no fields to update")

Output:

```json
{
  "id": "01J...ULID",
  "task_link": {
    "moss_capsule": "auth",
    "moss_workspace": "startupA"
  }
}
```

---

## 6.5 `moss.delete` (soft delete)

Soft-deletes a capsule by setting `deleted_at`.

```json
{ "workspace": "startupA", "name": "auth" }
```

Output:

```json
{ "deleted": true, "id": "01J...ULID" }
```

Soft-delete notes:

* Soft-deleted capsules are excluded from name uniqueness, so the same `(workspace,name)` can be stored again after delete.
* By default, read/browse actions exclude soft-deleted capsules; `include_deleted:true` is supported by: `fetch`, `fetch_many`, `latest`, `list`, `inventory`, `export`.
* `update` and `delete` by name operate on **active** capsules only (no `include_deleted` support).
* Recovery is via `fetch/include_deleted`, or `export include_deleted:true` + `import` (see “Export/import file format” under `export` / `import` below).

---

## 6.6 `moss.latest`

Summary only (default):

```json
{ "workspace": "startupA" }
```

Include text (explicit):

```json
{ "workspace": "startupA", "include_text": true }
```

Include soft-deleted:

```json
{ "workspace": "startupA", "include_deleted": true }
```

**Ordering with `include_deleted`:** When `include_deleted: true`, returns the single most recently updated capsule among **all** capsules (active + deleted) in the workspace. No preference for active over deleted — purely by `updated_at`.

### Capsule summary shape (browse/query ops)

Browse/query operations return capsule metadata without `capsule_text` using a shared summary shape:

* `id`
* `workspace`, `workspace_norm`
* `name` (optional), `name_norm` (optional)
* `title` (optional)
* `capsule_chars`, `tokens_estimate`
* `tags` (array, optional), `source` (optional)
* `created_at`, `updated_at`
* `deleted_at` (optional, only for soft-deleted capsules)

Default output (summary):

```json
{
  "item": {
    "id": "01J...ULID",
    "workspace": "startupA",
    "workspace_norm": "startupa",
    "name": "auth",
    "name_norm": "auth",
    "title": "Auth + sessions",
    "created_at": 1737260000,
    "updated_at": 1737260500,
    "tags": ["auth","sessions"],
    "source": "claude-code",
    "capsule_chars": 1823,
    "tokens_estimate": 456,
    "task_link": { "moss_capsule": "auth", "moss_workspace": "startupA" }
  }
}
```

If no capsules exist in the workspace, returns `{ "item": null }`.

---

## 6.7 `moss.list` (workspace scoped)

```json
{ "workspace": "startupA", "limit": 20, "offset": 0 }
```

Defaults: `limit: 20`, `offset: 0`. Max limit: `100`.

Include soft-deleted:

```json
{ "workspace": "startupA", "include_deleted": true }
```

Output (summaries only):

```json
{
  "items": [
    {
      "id":"01J...",
      "workspace":"startupA",
      "workspace_norm":"startupa",
      "name":"auth",
      "name_norm":"auth",
      "title":"Auth + sessions",
      "created_at":1737260000,
      "updated_at":1737260500,
      "tags":["auth"],
      "source":"claude-code",
      "capsule_chars":1823,
      "tokens_estimate":456
    }
  ],
  "pagination": { "limit": 20, "offset": 0, "has_more": false, "total": 1 },
  "sort": "updated_at_desc"
}
```

---

## 6.8 `moss.inventory` (global summaries only)

```json
{ "limit": 100, "offset": 0 }
```

Defaults: `limit: 100`, `offset: 0`. Max limit: `500`.

Optional filters:

```json
{ "workspace": "startupA", "tag": "auth", "name_prefix": "au", "include_deleted": true, "limit": 200, "offset": 0 }
```

Output:

```json
{
  "items": [
    { "id":"01J...","workspace":"startupA","workspace_norm":"startupa","name":"auth","name_norm":"auth","title":"Auth + sessions","created_at":1737260000,"updated_at":1737260500,"tags":["auth"],"source":"claude-code","capsule_chars":1823,"tokens_estimate":456 },
    { "id":"01J...","workspace":"startupB","workspace_norm":"startupb","title":"Brainstorm notes","created_at":1737200000,"updated_at":1737200000,"source":"codex","capsule_chars":921,"tokens_estimate":230 }
  ],
  "pagination": { "limit": 200, "offset": 0, "has_more": false, "total": 2 },
  "sort": "updated_at_desc"
}
```

Bloat rule: never returns `capsule_text`.

---

## 6.9 `moss.export`

Export capsules to JSONL file for backup/portability.

```json
{}
```

With explicit path:

```json
{ "path": "/tmp/moss-backup.jsonl" }
```

Export specific workspace:

```json
{ "workspace": "startupA" }
```

Include soft-deleted:

```json
{ "include_deleted": true }
```

**Parameters:**

* `path` (optional) — file path to write JSONL output
  * Default: `~/.moss/exports/<workspace>-<timestamp>.jsonl`
  * Example: `~/.moss/exports/default-2025-01-23T143022.jsonl`
  * Must have `.jsonl` extension
  * Must not contain directory traversal (`..`)
* `workspace` (optional) — filter to single workspace (default: all)
* `include_deleted` (optional) — include soft-deleted capsules

Output:

```json
{
  "path": "/Users/you/.moss/exports/default-2025-01-23T143022.jsonl",
  "count": 42,
  "exported_at": 1737260500
}
```

Response always includes actual `path` used. Directory created if needed.

### Export/import file format (JSONL)

Export and import use JSONL (newline-delimited JSON).

Each line is a complete capsule record:

```jsonl
{"id":"01ABC...","workspace_raw":"default","workspace_norm":"default","name_raw":"auth","name_norm":"auth","title":"Auth context","capsule_text":"Objective: ...","capsule_chars":1823,"tokens_estimate":456,"tags":["auth"],"source":"claude-code","created_at":1737260000,"updated_at":1737260500,"deleted_at":null}
{"id":"01DEF...","workspace_raw":"default","workspace_norm":"default","name_raw":"db","name_norm":"db","title":"Database schema","capsule_text":"Objective: ...","capsule_chars":2104,"tokens_estimate":512,"tags":[],"source":"cli","created_at":1737200000,"updated_at":1737200000,"deleted_at":null}
```

**Note:** Export uses `tags` as a JSON array for human readability. SQLite stores as `tags_json` (TEXT); export/import converts between formats.

First line may be a metadata header:

```jsonl
{"_moss_export":true,"schema_version":"1.0","exported_at":1737260500}
{"id":"01ABC...","workspace_raw":"default",...}
```

Import should detect and skip header lines (check for `_moss_export` key).

**File permissions:** recommend `0600` for backups containing sensitive context.

---

## 6.10 `moss.import`

Import capsules from JSONL file.

```json
{
  "path": "/tmp/moss-backup.jsonl",
  "mode": "error"
}
```

**Parameters:**

* `path` (required) — file path to read JSONL from
* `mode` (optional) — collision handling mode

**Collision modes:**

* `mode: "error"` (default) — fail on any `id` or `(workspace_norm, name_norm)` collision (atomic, no partial writes)
* `mode: "replace"` — overwrite on collision, preserve existing ID; fails if ambiguous (ID matches one row, name matches different row). Atomic.
* `mode: "rename"` — auto-suffix name on collision (e.g., `auth` → `auth-1`), generate new ID on ID collision. Atomic.

### Import normalization (important)

On import, **always recompute** `workspace_norm` and `name_norm` from `workspace_raw` and `name_raw`. Do not trust incoming `*_norm` fields — they may be stale, hand-edited, or from a different normalization version.

The `*_norm` fields in JSONL exports are informational only.

### Required fields per record

Each import record must contain:
* `id` — capsule identifier
* `workspace_raw` — workspace name (used to recompute `workspace_norm`)

Records missing `id` or `workspace_raw` are skipped with an `INVALID_RECORD` error.

Other fields are accepted if present:
* `capsule_text` is stored as-is (empty is allowed).
* `capsule_chars` and `tokens_estimate` are informational only (ignored on import; recomputed from `capsule_text`).
* `created_at`, `updated_at`, `deleted_at` are restored as provided.

### Import collision semantics (implementation guidance)

Collisions should be detected on:
* **ID collision:** imported `id` already exists in the DB
* **Name collision:** imported `(workspace_norm, name_norm)` conflicts with an existing **active** capsule (`deleted_at IS NULL`) — using the **recomputed** norm values

Suggested handling by mode:
* `mode:"error"`: fail the import on any collision (id or active name), ideally atomically (no partial writes).
* `mode:"replace"`: overwrite existing capsules on collision:
  * If colliding by `id`, update the existing row in-place (preserve `id`).
  * If colliding by active name with a different `id`, update the name-matched row in-place and ignore the imported `id` (preserve the existing `id`).
  * If a single incoming record collides by `id` with one row and by name with a different row, fail the import (ambiguous merge).
* `mode:"rename"`: on name collision, suffix deterministically (e.g., `auth` → `auth-1`, `auth-2`, …) until free, then insert; on `id` collision, generate a new `id` before inserting.

Output:

```json
{
  "imported": 42,
  "skipped": 0,
  "errors": []
}
```

---

## 6.11 `moss.purge`

Permanently delete soft-deleted capsules.

Purge all soft-deleted:

```json
{}
```

Purge in specific workspace:

```json
{ "workspace": "startupA" }
```

Purge older than N days:

```json
{ "older_than_days": 7 }
```

Output:

```json
{
  "purged": 5,
  "message": "Permanently deleted 5 capsules"
}
```

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

---

# 8) Runtime configuration

## 8.1) Configuration file

Location: `~/.moss/config.json`

```json
{
  "capsule_max_chars": 12000
}
```

**Defaults** (applied if config missing or field omitted):

| Field | Default | Description |
|-------|---------|-------------|
| `capsule_max_chars` | 12000 | Max characters per capsule (~3k tokens) |

---

## 8.2) CLI

CLI is a debug/human interface that shares the same SQLite DB as MCP and mirrors the MCP actions.

### Commands and flags

```bash
# Store (create) a capsule
moss store --name=auth < capsule.md
moss store --name=auth --workspace=myproject --title="Auth context" < capsule.md

# Fetch a capsule
moss fetch <id>
moss fetch --workspace=X --name=Y

# Update a capsule
moss update <id> --title="New title"
moss update --workspace=X --name=Y < updated.md

# Delete (soft delete) a capsule
moss delete <id>
moss delete --workspace=X --name=Y

# List capsules
moss list [--workspace=X] [--limit=N] [--offset=N] [--include-deleted]

# Show inventory
moss inventory [--workspace=X] [--tag=X] [--name-prefix=X]

# Get latest capsule
moss latest [--workspace=X] [--include-text] [--include-deleted]

# Export to JSONL file
moss export [--path=backup.jsonl] [--workspace=X] [--include-deleted]

# Import from JSONL file (path required)
moss import --path=backup.jsonl [--mode=error|replace|rename]

# Purge soft-deleted capsules
moss purge [--workspace=X] [--older-than=7d]
```

**Addressing:** Commands that operate on a single capsule (fetch, update, delete) accept either:
- Positional `<id>` argument, OR
- `--workspace` + `--name` flags

**store flags:**
| Flag | Required | Default |
|------|----------|---------|
| `--name` | no | (unnamed capsule) |
| `--workspace` | no | `"default"` |
| `--title` | no | same as name |
| `--tags` | no | (none) |
| `--mode` | no | `"error"` |
| `--allow-thin` | no | false |

### stdin reading

`moss store` and `moss update` read capsule content from stdin:
```bash
moss store --name=auth < capsule.md
cat capsule.md | moss store --name=auth
echo "## Objective..." | moss update --name=auth
```

### Output format

All CLI output is JSON for scriptability. Table formatting may be added in a future version.

**Example outputs:**

```bash
# store
{"id": "01J...", "task_link": {"moss_capsule": "auth", "moss_workspace": "default"}}

# delete
{"deleted": true, "id": "01J..."}

# list
{"items": [...], "pagination": {...}, "sort": "updated_at_desc"}

# fetch
{"id": "01J...", "workspace": "default", "name": "auth", "capsule_text": "...", ...}
```

### Error output

Errors print to stderr with exit code 1:
```
[CAPSULE_TOO_THIN] capsule missing required sections: [Decisions, Key locations]
```

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
* `created_at INTEGER NOT NULL`
* `updated_at INTEGER NOT NULL`
* `deleted_at INTEGER NULL` — soft delete timestamp (null = active)

## Indexes / constraints

* Unique name handles: `UNIQUE(workspace_norm, name_norm)` excluding soft-deleted
* Fast list/latest: `INDEX(workspace_norm, updated_at DESC)` excluding soft-deleted

### SQLite DDL

```sql
CREATE TABLE IF NOT EXISTS capsules (
  id              TEXT PRIMARY KEY,
  workspace_raw   TEXT NOT NULL,
  workspace_norm  TEXT NOT NULL,
  name_raw        TEXT,
  name_norm       TEXT,
  title           TEXT,
  capsule_text    TEXT NOT NULL,
  capsule_chars   INTEGER NOT NULL,
  tokens_estimate INTEGER NOT NULL,
  tags_json       TEXT,
  source          TEXT,
  created_at      INTEGER NOT NULL,
  updated_at      INTEGER NOT NULL,
  deleted_at      INTEGER
);

-- Fast list/latest (excludes soft-deleted)
CREATE INDEX IF NOT EXISTS idx_capsules_workspace_updated
ON capsules(workspace_norm, updated_at DESC)
WHERE deleted_at IS NULL;

-- Unique name handles (excludes soft-deleted)
CREATE UNIQUE INDEX IF NOT EXISTS idx_capsules_workspace_name_norm
ON capsules(workspace_norm, name_norm)
WHERE name_norm IS NOT NULL AND deleted_at IS NULL;
```

### Token estimation (for `tokens_estimate`)

```go
func estimateTokens(text string) int {
	words := strings.Fields(strings.TrimSpace(text))
	return int(math.Ceil(float64(len(words)) * 1.3))
}
```

Heuristic: ~1.3 tokens per word (English average). Used for summaries (`list`/`inventory`) and display.

---

# 10) Validation & constraints

## Required fields

* `workspace` required for `store/list/latest` and name addressing (defaults to `"default"` if omitted)
* `capsule_text` required for `store`; optional for `update` (lint runs only if provided)
* For `fetch/update/delete`: either `id` OR (`workspace`+`name`), not both

## Hard limits

* Default `capsule_max_chars`: **12,000** (configurable in `~/.moss/config.json`)
* `len(capsule_text) <= capsule_max_chars` else **413 CAPSULE_TOO_LARGE**

## Mode validation

* `store.mode` must be `"error"` (default) or `"replace"`
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
| NAME_ALREADY_EXISTS | 409 | Name collision on store with mode:"error" |
| CAPSULE_TOO_LARGE | 413 | Exceeds `capsule_max_chars` |
| CAPSULE_TOO_THIN | 422 | Missing required sections |
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

The `details` field varies by error code (e.g., `max_chars`/`actual_chars` for CAPSULE_TOO_LARGE).

---

# 12) Operational flows (value-preserving)

## Flow A — Save (Session A → Moss)

1. Distill to capsule under limit using the capsule contract
2. Call `store`
3. Moss validates size + lint, saves, returns id

## Flow B — Continue (Moss → Session B)

1. Call `fetch` by name or `latest include_text:true`
2. Paste `capsule_text` into new session
3. Continue work

## Flow C — Browse (no bloat)

* `inventory` to see everything stored (no text)
* `list` for workspace
* `latest` summary, then `fetch` to load

## Flow D — Refresh (rolling state)

* Distill updated state → `update`
* Capsule stays evergreen

## Flow E — Cleanup

* `delete`

## Flow F — Backup & Restore

```
1. Export all capsules:
   export { path: "/tmp/backup.jsonl" }

2. Export specific workspace:
   export { path: "/tmp/projectA.jsonl", workspace: "projectA" }

3. Restore to new machine:
   import { path: "/tmp/backup.jsonl", mode: "error" }

4. Merge with existing:
   import { path: "/tmp/backup.jsonl", mode: "replace" }
```
