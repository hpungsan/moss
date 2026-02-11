# UI Design Spec (Moss v0)

## Summary

Local web UI for Moss (`moss serve`). Browse, search, and delete capsules from a browser. Read-only plus delete — no store/update/append. Built on Go stdlib `net/http`, `html/template`, htmx, and `embed.FS`. No build step, no external CDN loads, no auth. Handlers call the same `internal/ops` layer as MCP, so behavior is identical.

**Related docs:**
- [Capsule API spec](../capsule/DESIGN.md) — MCP tool behaviors, capsule structure
- [Setup](../SETUP.md) — Config system, paths
- [Capsule backlog](../capsule/BACKLOG.md) — Planned features

---

# 1) Goals and non-goals

## Goals

* **Browse capsules:** List, filter, paginate, view details without touching the CLI or MCP.
* **Search:** Full-text search with FTS5 snippets, same ranking as `capsule_search`.
* **Inspect:** View rendered capsule markdown with metadata sidebar.
* **Clean up:** Delete individual capsules, purge soft-deleted capsules.
* **Cross-workspace view:** Inventory page for everything stored.
* **Localhost:** Binds to `127.0.0.1` by default. No remote access by default.
* **No build step:** All assets embedded via `embed.FS`. `go build` produces a single binary.
* **Reuse ops layer:** Handlers call `internal/ops` functions directly — same validation, same errors, same behavior as MCP.

## Non-goals (v0)

* **Write operations:** No store, update, or append from the UI. Use MCP or CLI.
* **Authentication / authorization:** Localhost-only, single-user.
* **Real-time updates:** No WebSockets, no SSE. Refresh the page.
* **Mobile-optimized layout:** Desktop-first. Functional on mobile, not designed for it.
* **Bulk operations:** No multi-select delete or bulk update.
* **Compose UI:** No visual capsule composition.
* **Export/import UI:** Use CLI or MCP.
* **Theming / dark mode:** Single light theme. Can be added later via CSS variables.

---

# 2) Architecture

## 2.1 Layer diagram

```
Browser
  │
  ▼
moss serve (net/http)
  │
  ├── GET /capsules         → handlers.HandleList()
  ├── GET /capsules/search  → handlers.HandleSearch()
  ├── GET /capsules/{id}    → handlers.HandleDetail()
  ├── DELETE /capsules/{id} → handlers.HandleDelete()
  ├── ...
  │
  ▼
internal/ops
  │
  ▼
internal/db
  │
  ▼
SQLite (~/.moss/moss.db)
```

HTTP handlers are thin: parse request, build ops input, call ops function, render template or JSON. No business logic in handlers.

## 2.2 Package structure

```
internal/web/
├── server.go         # HTTP server setup, router, graceful shutdown
├── handlers.go       # Route handlers (one function per route)
├── render.go         # Template rendering helpers, error rendering
├── templates/        # html/template files (embedded)
│   ├── layout.html       # Base layout (head, nav, footer, htmx)
│   ├── list.html         # Capsule list with filters
│   ├── detail.html       # Single capsule view
│   ├── search.html       # Search results
│   ├── inventory.html    # Cross-workspace inventory
│   └── error.html        # Error page
└── static/           # Static assets (embedded)
    ├── htmx.min.js       # htmx (vendored, no CDN)
    ├── app.js            # Event delegation (form submit prevention, go-back navigation)
    └── style.css         # Minimal CSS
```

## 2.3 Technology stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| HTTP server | Go `net/http` (Go 1.22+ path params) | No dependencies. `{id}` routing built in. |
| Templates | `html/template` | Auto-escaping (XSS safe). Stdlib. |
| Interactivity | htmx (vendored) | Delete, search, pagination without JS framework. |
| Markdown rendering | `github.com/yuin/goldmark` | Pure Go. Safe HTML output. |
| Asset embedding | `embed.FS` | Single binary. No file path issues. |
| Ops layer | `internal/ops` | Same functions as MCP handlers. |

## 2.4 Ops reuse

MCP handlers and web handlers both call the same ops functions:

```
MCP handler  ──┐
               ├──► ops.List(ctx, db, ListInput{...}) ──► db ──► SQLite
Web handler  ──┘
```

This guarantees identical behavior: same validation, same pagination limits, same error codes, same FTS5 ranking. The web layer is purely a presentation concern.

---

# 3) Routes

## Route summary

| Method | Route | Ops call | Response |
|--------|-------|----------|----------|
| GET | `/` | — | 302 → `/capsules` |
| GET | `/capsules` | `ops.List` | HTML page (list + filters) |
| GET | `/capsules/search` | `ops.Search` | HTML page (results + snippets) |
| GET | `/capsules/inventory` | `ops.Inventory` | HTML page (cross-workspace) |
| GET | `/capsules/{id}` | `ops.Fetch` | HTML page (detail + rendered markdown) |
| DELETE | `/capsules/{id}` | `ops.Delete` | htmx: `HX-Redirect`. JSON: `{"deleted": true, "id": "..."}` |
| POST | `/capsules/purge` | `ops.Purge` | Requires `confirm=true`. Returns count. (No UI control yet.) |

Static routes (not listed above): `GET /static/*` serves embedded CSS and JS.

---

## 3.1 `GET /`

Redirect to the capsule list.

**Response:** `302 Found`, `Location: /capsules`

No template. No ops call.

---

## 3.2 `GET /capsules`

List capsules in a workspace with filters and pagination.

**Query params:**

| Param | Type | Default | Maps to |
|-------|------|---------|---------|
| `workspace` | string | `"default"` | `ListInput.Workspace` |
| `run_id` | string | — | `ListInput.RunID` |
| `phase` | string | — | `ListInput.Phase` |
| `role` | string | — | `ListInput.Role` |
| `include_deleted` | bool | `false` | `ListInput.IncludeDeleted` |
| `limit` | int | 20 | `ListInput.Limit` (max: 100) |
| `offset` | int | 0 | `ListInput.Offset` |

**Ops call:** `ops.List(ctx, db, ListInput{...})`

**Template:** `list.html`

**Page contents:**
- Workspace selector (text input, pre-filled with current workspace)
- Filter sidebar: `run_id`, `phase`, `role`, `include_deleted` checkbox, Apply button
- Capsule table: name/ID, title, chars, created, updated, actions (delete button)
- Each row links to `/capsules/{id}` (with `?include_deleted=true` appended when the deleted filter is active)
- Delete button per row (htmx DELETE, requires confirmation)
- Pagination controls (prev/next, showing offset/total) with URL-encoded filter values

**htmx behavior:**
- Filter form uses `hx-get="/capsules"` with `hx-push-url="true"` — submitted via Apply button (not auto-submit on change)
- Delete button uses `hx-delete="/capsules/{id}"` with `hx-confirm="Delete this capsule?"` — on success, htmx follows `HX-Redirect` to reload the list

**Error cases:**
- Invalid `limit`/`offset`: non-integers silently fall back to defaults (limit=20, offset=0)
- All other errors → error page with status + message

---

## 3.3 `GET /capsules/search`

Full-text search across capsules.

**Query params:**

| Param | Type | Default | Maps to |
|-------|------|---------|---------|
| `q` | string | — (required) | `SearchInput.Query` |
| `workspace` | string | — | `SearchInput.Workspace` |
| `tag` | string | — | `SearchInput.Tag` |
| `run_id` | string | — | `SearchInput.RunID` |
| `phase` | string | — | `SearchInput.Phase` |
| `role` | string | — | `SearchInput.Role` |
| `include_deleted` | bool | `false` | `SearchInput.IncludeDeleted` |
| `limit` | int | 20 | `SearchInput.Limit` (max: 100) |
| `offset` | int | 0 | `SearchInput.Offset` |

**Ops call:** `ops.Search(ctx, db, SearchInput{...})`

If `q` is empty, render the search page with an empty search box (no ops call).

**Template:** `search.html`

**Page contents:**
- Search input box (auto-focused)
- Filter inputs: `workspace`, `tag`, `run_id`, `phase`, `role` (`include_deleted` accepted by handler but not exposed as a UI control)
- Results as cards: name/ID, workspace badge, snippet (HTML-safe, `<b>` highlights from FTS5), chars, tags
- Each result links to `/capsules/{id}` (with `?include_deleted=true` appended when the deleted filter is active)
- Pagination controls with URL-encoded filter values

**htmx behavior:**
- The `<form>` has `data-no-submit` to prevent native form submission (handled by `app.js` event delegation, CSP-compatible)
- Search input uses `hx-get="/capsules/search"` with `hx-trigger="input changed delay:300ms, search"` for debounced search-as-you-type (also fires on Enter/clear)
- Search input uses `hx-target="#results"` to swap only the results section. Handler detects `HX-Target: results` and renders only the `search-results` template block (not the full page content), preventing form duplication.
- `hx-push-url="true"` to keep URL shareable
- Filter field values are included in the htmx request via `hx-include` on the search input

**Error cases:**
- Query > 1000 chars → 400 error page
- Invalid FTS5 syntax → 400 error page with message from ops
- Empty query → render empty search page (no error)

---

## 3.4 `GET /capsules/inventory`

Cross-workspace capsule inventory.

**Query params:**

| Param | Type | Default | Maps to |
|-------|------|---------|---------|
| `workspace` | string | — | `InventoryInput.Workspace` |
| `tag` | string | — | `InventoryInput.Tag` |
| `name_prefix` | string | — | `InventoryInput.NamePrefix` |
| `run_id` | string | — | `InventoryInput.RunID` |
| `phase` | string | — | `InventoryInput.Phase` |
| `role` | string | — | `InventoryInput.Role` |
| `include_deleted` | bool | `false` | `InventoryInput.IncludeDeleted` |
| `limit` | int | 100 | `InventoryInput.Limit` (max: 500) |
| `offset` | int | 0 | `InventoryInput.Offset` |

**Ops call:** `ops.Inventory(ctx, db, InventoryInput{...})`

**Template:** `inventory.html`

**Page contents:**
- Filter bar: `workspace`, `tag`, `name_prefix`, `run_id`, `phase`, `role`, `include_deleted` checkbox
- Flat capsule table with workspace column visible (not grouped)
- Columns: name/ID, title, workspace, chars, created, updated
- Each row links to `/capsules/{id}` (with `?include_deleted=true` appended when the deleted filter is active)
- Pagination controls with URL-encoded filter values

**htmx behavior:**
- Filter form uses `hx-get="/capsules/inventory"` with `hx-push-url="true"` — submitted via Apply button (not auto-submit on change)

**Error cases:**
- Invalid `limit`/`offset`: non-integers silently fall back to defaults (limit=100, offset=0)

---

## 3.5 `GET /capsules/{id}`

View a single capsule with rendered markdown content.

**Path params:**

| Param | Type | Maps to |
|-------|------|---------|
| `id` | string | `FetchInput.ID` |

**Query params:**

| Param | Type | Default | Maps to |
|-------|------|---------|---------|
| `include_deleted` | bool | `false` | `FetchInput.IncludeDeleted` |

**Ops call:** `ops.Fetch(ctx, db, FetchInput{ID: id, IncludeText: ptr(true), IncludeDeleted: parseBoolParam(r, "include_deleted")})`

The capsule's `CapsuleText` is rendered from markdown to HTML using goldmark before passing to the template.

**Template:** `detail.html`

**Page contents:**
- Breadcrumb: Capsules → {name or id}
- Rendered capsule markdown (main content area)
- Metadata sidebar:
  - ID, workspace, name, title
  - Tags (as badge spans)
  - Source, run_id, phase, role
  - Chars, tokens estimate
  - Created at, updated at
  - Deleted at (if soft-deleted)
- Delete button (if not already deleted)

**htmx behavior:**
- Delete button uses `hx-delete="/capsules/{id}"` with `hx-confirm="Delete this capsule?"` — on success, htmx follows `HX-Redirect` back to `/capsules`

**Error cases:**
- Capsule not found → 404 error page
- Soft-deleted without `include_deleted=true` → 404 error page

---

## 3.6 `DELETE /capsules/{id}`

Soft-delete a capsule.

**Path params:**

| Param | Type | Maps to |
|-------|------|---------|
| `id` | string | `DeleteInput.ID` |

**Ops call:** `ops.Delete(ctx, db, DeleteInput{ID: id})`

**Response (content negotiation):**

| Condition | Response |
|-----------|----------|
| `HX-Request: true` | 200 + `HX-Redirect: /capsules` header |
| `Accept` contains `application/json` | `{"deleted": true, "id": "<id>"}` |
| Otherwise | 302 → `/capsules` |

**Error cases:**
- Not found → 404 (HTML error page or JSON error, per content negotiation)

---

## 3.7 `POST /capsules/purge`

Permanently delete all soft-deleted capsules. No UI control exists yet — the endpoint is available for form-driven workflows and tooling (e.g. `curl`).

**Form params:**

| Param | Type | Required | Maps to |
|-------|------|----------|---------|
| `confirm` | string | Yes (must be `"true"`) | — (safety guard) |
| `workspace` | string | No | `PurgeInput.Workspace` |
| `older_than_days` | int | No | `PurgeInput.OlderThanDays` |

**Ops call:** `ops.Purge(ctx, db, PurgeInput{...})`

If `confirm` is not `"true"`, return 400 with message `"confirm parameter must be \"true\""`.

**Response (content negotiation):**

| Condition | Response |
|-----------|----------|
| `HX-Request: true` | HTML fragment with purge result message |
| `Accept` contains `application/json` | `{"purged": N, "message": "..."}` |
| Otherwise | 302 → `/capsules?include_deleted=true` |

**Error cases:**
- Missing `confirm=true` → 400
- Internal error → 500

---

# 4) Templates and htmx patterns

## 4.1 Template files

All templates are embedded via `embed.FS` and parsed at startup (not per-request). Each page template defines a `content` block that the layout wraps. For htmx requests (`HX-Request` header), only the `content` block is rendered to avoid duplicating the layout shell.

### `layout.html`

Base layout. Provides `<head>` (CSS, htmx, app.js), nav bar (Capsules, Inventory, Search), `<main id="main">` container for the content block, and footer with version.

### `list.html`

- Sidebar: workspace input, filter fields (`run_id`, `phase`, `role`), "Include deleted" checkbox
- Main: table of capsules with columns: Name/ID, Title, Chars, Created, Updated, Actions (delete button)
- Pagination: Prev / Next links with offset math
- Empty state when no capsules match filters

### `detail.html`

- Breadcrumb navigation back to list
- Two-column layout: rendered markdown (main), metadata sidebar (right)
- Markdown rendered server-side via goldmark into `template.HTML`
- Raw capsule text toggle (collapsible `<details>` element)
- Delete button at bottom of sidebar (hidden for already-deleted capsules)

### `search.html`

- Search box with debounced htmx trigger (300ms delay)
- Inline filter fields: workspace, tag, run_id, phase, role
- Results rendered as cards: title/name, workspace badge, FTS5 snippet with `<b>` highlights
- Snippets are HTML-safe (ops layer handles escaping)
- Empty state: "No results" or "Enter a search query"
- Results content is in a separate `search-results` template block, rendered independently for htmx partial swaps (prevents form duplication when `HX-Target: results`)

### `inventory.html`

- Horizontal filter bar: workspace, tag, name_prefix, run_id, phase, role, include deleted
- Table with workspace column visible (cross-workspace view)
- Same pagination pattern as list

### `error.html`

- Centered error display: HTTP status code, error message
- Two actions: "Back to capsules" link (`/capsules`) and "Go back" button (`history.back()` via `app.js` event delegation, CSP-compatible)
- No stack traces or internal details

## 4.2 htmx patterns

All htmx interactions use `hx-push-url="true"` to keep the URL bar in sync with the current view.

- **Search debounce:** Search input triggers on `input changed delay:300ms, search` and targets `#results` to swap only the results section. Includes all filter field values via `hx-include`. The search form uses `data-no-submit` (not `hx-get`) to prevent native form submission; `app.js` handles submit prevention via event delegation (CSP-compatible). Handler detects `HX-Target: results` and renders only the `search-results` template block.
- **Delete with confirmation:** Uses `hx-delete` with `hx-confirm` browser dialog. On success, server responds with `HX-Redirect: /capsules` and htmx navigates.
- **Filter forms (list, inventory):** Submit via Apply button using `hx-get` targeting `#main`. Server detects `HX-Request: true` and returns only the content block (not the full layout).
- **Pagination:** Standard `<a>` links with offset/limit query params. Filter values are URL-encoded via `urlquery`.
- **Purge (no UI yet):** Endpoint supports `hx-post` with `hx-confirm` dialog and hidden `confirm=true` field, but no template currently includes a purge control.

---

# 5) Configuration

## 5.1 Config fields

Defined in `internal/config/config.go` `Config` struct:

| Field | JSON key | Type | Default | Description |
|-------|----------|------|---------|-------------|
| `UIPort` | `ui_port` | `int` | `8314` | Port for `moss serve` |
| `UIBind` | `ui_bind` | `string` | `"127.0.0.1"` | Bind address for `moss serve` |

These follow the same config loading and merge behavior as existing fields (see [capsule DESIGN.md §8](../capsule/DESIGN.md#8-runtime-configuration)):
- Scalars: repo overrides global (if non-zero)
- Zero values fall back to defaults

**Example config (`~/.moss/config.json`):**

```json
{
  "capsule_max_chars": 12000,
  "ui_port": 8314,
  "ui_bind": "127.0.0.1"
}
```

## 5.2 Default port choice

Port 8314 chosen because:
- Not in common use (checked IANA registry)
- Easy to remember (no conflict with 8080, 8000, 3000, etc.)
- Above 1024 (no root needed)

---

# 6) CLI: `moss serve`

## 6.1 Command definition

```
moss serve [--port PORT] [--bind ADDRESS]
```

| Flag | Type | Default | Source |
|------|------|---------|--------|
| `--port` | int | 8314 | Config `ui_port`, then flag override |
| `--bind` | string | `127.0.0.1` | Config `ui_bind`, then flag override |

Flag precedence: CLI flag > repo config > global config > default.

## 6.2 Startup behavior

1. Load config (global + repo merge)
2. Open database (`~/.moss/moss.db`)
3. Apply flag overrides to config values
4. Create HTTP server with routes (address built via `net.JoinHostPort` for IPv6 safety)
5. Start listening
6. Log: `Moss UI running at http://{addr}`
7. If bind is exactly `0.0.0.0` or `::`, log warning: `WARNING: Server is binding to all interfaces and may be accessible from the network`

## 6.3 Graceful shutdown

- Listen for `SIGINT` and `SIGTERM`
- On signal: call `http.Server.Shutdown(ctx)` with 5-second timeout
- Log: `Shutting down...`
- Close database (via `defer db.Close()` in `main.go`, triggered on return from `serveCmd`)
- Exit 0

## 6.4 Registration

Registered as `serveCmd(db, cfg)` in the commands list in `cmd/moss/cli.go`.

---

# 7) Error handling

## 7.1 MossError to HTTP status

`MossError.Status` maps directly to HTTP status codes:

| MossError.Status | HTTP status | Used by |
|------------------|-------------|---------|
| 400 | 400 Bad Request | Invalid params, missing confirm |
| 404 | 404 Not Found | Capsule not found |
| 500 | 500 Internal Server Error | Unexpected errors |

Other MossError statuses (409, 413, 422, 499) are unlikely in the read-only UI but are handled the same way: `MossError.Status` becomes the HTTP response status.

## 7.2 Content negotiation

The error rendering function checks request context to determine response format:

| Condition | Response |
|-----------|----------|
| `HX-Request: true` | HTML fragment (error message only, for htmx swap) |
| `Accept` contains `application/json` | JSON: `{"error": {"code": "...", "message": "...", "status": N}}` |
| Otherwise | Full error page (`error.html` template) |

## 7.3 Security

- `MossError.Details` is never exposed in HTTP responses (may contain internal error strings)
- Stack traces are never rendered
- Error messages from ops are safe to display (they contain user-provided identifiers like capsule names, which are escaped by `html/template`)

---

# 8) Security

## 8.1 Network

- Default bind: `127.0.0.1` (localhost only)
- Warning logged if bound to `0.0.0.0` or `::`
- No TLS (localhost doesn't need it)
- No authentication (single-user, localhost assumption)
- No CORS headers (same-origin by default)

## 8.2 XSS prevention

- `html/template` auto-escapes all template variables
- Capsule markdown rendered by goldmark with `html.WithUnsafe()` **disabled** (default safe mode strips raw HTML from markdown)
- FTS5 search snippets are pre-escaped by the ops layer (only `<b>` tags for highlights); rendered in templates via the `trustedSnippet` template function (named to signal that only ops-produced content should be passed)
- htmx attributes use static values, no user-controlled injection points
- No `template.HTML` for user-supplied content (only for goldmark-rendered output and ops-generated snippets via `trustedSnippet`)

## 8.3 Destructive operations

- **Delete:** Requires `hx-confirm` browser confirmation dialog. Soft-delete only (recoverable via `include_deleted`).
- **Purge:** Requires `confirm=true` form parameter. Permanent. (No UI control yet; when added, should include `hx-confirm` browser dialog.)
- No CSRF tokens needed (localhost, no auth, no cookies with session state)

## 8.4 Asset security

- All assets embedded in binary (`embed.FS`): htmx, app.js, CSS
- No external CDN loads, no external scripts
- No inline JavaScript: event handlers use `app.js` (event delegation) and htmx attributes, not `onclick`/`onsubmit`/`javascript:` URIs
- Security headers on all responses:
  - `Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self'`
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`

---

# 9) Future considerations (not in v0)

These are explicitly deferred. Listed here to show they were considered and to avoid premature design decisions.

- **Dark mode:** Add CSS variables for theming. Toggle in nav.
- **Bulk delete:** Checkbox select + batch delete. Needs ops.BulkDelete integration.
- **Compose viewer:** Visual bundle preview. Needs ops.Compose integration.
- **Export/import UI:** File upload/download. Needs multipart form handling.
- **Update/append:** Edit capsule content in-browser. Needs ops.Update/Append integration.
- **Real-time:** SSE for live updates when MCP writes. Needs event bus.
- **Mobile layout:** Responsive CSS for small screens.
- **API token auth:** Required if binding to non-localhost.
