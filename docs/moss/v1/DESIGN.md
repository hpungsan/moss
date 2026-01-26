# Moss v1 Design Spec

## Version History

- **v1.0**: Initial release — 11 MCP tools, CLI, capsule linting (6 sections), soft-delete, export/import

---

## 0) One-line summary

Moss is a **local "context capsule" store** that lets you **store/fetch/update/delete** a *strictly size-bounded* distilled handoff across Claude Code, Codex, etc., with **batch fetch (`fetch_many`)**, **upsert mode**, **latest/list**, **global inventory**, **human-friendly `name` handles**, **export/import** for portability, **soft-delete** for safety, and **guardrails (capsule lint + size limits)** to prevent both **context bloat** *and* **low-value capsules**.

**Related docs:**
- [OVERVIEW.md](../OVERVIEW.md) — Concepts and use cases
- [BACKLOG.md](BACKLOG.md) — Future features

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

# 6) MCP tool behaviors

Tool schemas are defined in code (`internal/mcp/tools.go`). This section documents key behaviors.

## 6.1 `moss.store`

**Required:** `capsule_text`

**Optional:** `workspace` (default: "default"), `name`, `title`, `tags`, `source`, `mode` ("error"|"replace"), `allow_thin`

**Behaviors:**
- `mode:"error"` + name collision → **409 NAME_ALREADY_EXISTS**
- `mode:"replace"` + name collision → overwrite (preserve `id`)
- Too large → **413 CAPSULE_TOO_LARGE**
- Lint fails → **422 CAPSULE_TOO_THIN**
- Soft-deleted capsules don't participate in name uniqueness

**Output:** `{ id, task_link }` — `task_link` provides ready-to-use metadata for Claude Code Tasks integration.

---

## 6.2 `moss.fetch`

**Addressing:** `id` OR (`workspace` + `name`) — not both

**Optional:** `include_deleted`, `include_text` (default: true)

**Behaviors:**
- Default excludes soft-deleted → **404 NOT_FOUND**
- `include_deleted:true` makes soft-deleted visible
- `include_text:false` returns summary only (peek)

---

## 6.3 `moss.fetch_many`

Batch fetch multiple capsules.

**Required:** `items` array (each addressed by `id` OR `workspace`+`name`)

**Optional:** `include_text`, `include_deleted`

**Behaviors:**
- Partial success allowed — missing items in `errors` array
- Each item includes `task_link`

---

## 6.4 `moss.update`

**Addressing:** `id` OR (`workspace` + `name`)

**Editable:** `capsule_text`, `title`, `tags`, `source`

**Immutable:** `id`, `workspace`, `name` — to "rename", delete and re-store

**Behaviors:**
- Missing → **404 NOT_FOUND**
- Too large → **413 CAPSULE_TOO_LARGE**
- Lint fails → **422 CAPSULE_TOO_THIN**
- No fields → **400 INVALID_REQUEST**

---

## 6.5 `moss.delete`

Soft-deletes by setting `deleted_at`. Capsule recoverable via `include_deleted` or export/import.

---

## 6.6 `moss.latest`

Returns most recent capsule in workspace.

**Optional:** `include_text` (default: false), `include_deleted`

---

## 6.7 `moss.list`

List summaries in workspace. **Never returns `capsule_text`.**

**Optional:** `limit` (default: 20, max: 100), `offset`, `include_deleted`

---

## 6.8 `moss.inventory`

Global list across all workspaces. **Never returns `capsule_text`.**

**Optional filters:** `workspace`, `tag`, `name_prefix`, `include_deleted`, `limit` (default: 100, max: 500), `offset`

---

## 6.9 `moss.export`

Export to JSONL file.

**Optional:** `path` (default: `~/.moss/exports/<workspace>-<timestamp>.jsonl`), `workspace`, `include_deleted`

---

## 6.10 `moss.import`

Import from JSONL file.

**Required:** `path`

**Optional:** `mode` — "error" (default, atomic fail on collision), "replace" (overwrite), "rename" (auto-suffix)

**Important:** `*_norm` fields are recomputed on import; don't trust incoming values.

---

## 6.11 `moss.purge`

Permanently delete soft-deleted capsules.

**Optional:** `workspace`, `older_than_days`

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
* `created_at INTEGER NOT NULL`
* `updated_at INTEGER NOT NULL`
* `deleted_at INTEGER NULL` — soft delete timestamp (null = active)

## Indexes / constraints

* Unique name handles: `UNIQUE(workspace_norm, name_norm)` excluding soft-deleted
* Fast list/latest: `INDEX(workspace_norm, updated_at DESC)` excluding soft-deleted

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
