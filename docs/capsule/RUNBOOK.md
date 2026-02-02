# Capsule Runbook

Operational guide for using the Capsule primitive (via Moss): connect via MCP/CLI, operate capsules, and troubleshoot capsule-specific issues.

**Prereq:** Install and verify Moss first via [SETUP.md](../SETUP.md).

**Note:** Examples use `moss` assuming it’s on your `PATH`. If you built locally without installing, use `./bin/moss` (or the absolute path to the binary).

---

## Claude Code Integration (Capsule MCP tools)

Add Moss as an MCP server via `.mcp.json` in your project root. This is what exposes the `capsule_*` MCP tools to Claude Code.

### Configuration

Create `.mcp.json` in your project root:

```json
{
  "mcpServers": {
    "moss": {
      "command": "/path/to/moss"
    }
  }
}
```

Replace `/path/to/moss` with the actual path to your built binary:
- If installed via `go install`: use `moss` (must be in PATH)
- If built locally: use absolute path like `/Users/you/moss/bin/moss`

### Permissions for Multi-Agent Orchestration

By default, Claude Code prompts for approval on each MCP tool call. For autonomous multi-agent workflows (swarms, pipelines), pre-approve Moss tools in `~/.claude/settings.json`:

```json
{
  "permissions": {
    "allow": [
      "mcp__moss__*"
    ]
  }
}
```

This allows all Moss MCP tools without prompting. For finer control:

| Permission | Effect |
|------------|--------|
| `mcp__moss__*` | All Moss tools (recommended for orchestration) |
| `mcp__moss__capsule_fetch` | Only capsule_fetch |
| `mcp__moss__capsule_store` | Only capsule_store |
| `mcp__moss__capsule_list` | Only capsule_list |

**Why this matters:** In swarm patterns, workers run autonomously in background. Manual approval would block the workflow. Pre-approving Moss lets agents share context without human intervention.

### Verify Integration

1. **Restart Claude Code** (or start a new session) to load the MCP server
2. Ask: "Use `capsule_inventory` to list all capsules"
3. Expected: Tool call succeeds with `items: []` (empty store) or list of existing capsules


### Available Tools

| Tool | Description |
|------|-------------|
| `capsule_store` | Create a new capsule |
| `capsule_fetch` | Retrieve a capsule by ID or name |
| `capsule_fetch_many` | Batch fetch multiple capsules |
| `capsule_update` | Update an existing capsule |
| `capsule_delete` | Soft-delete a capsule |
| `capsule_latest` | Get most recent capsule in workspace |
| `capsule_list` | List capsules in a workspace |
| `capsule_inventory` | List all capsules across workspaces |
| `capsule_search` | Full-text search across capsules |
| `capsule_export` | Export capsules to JSONL file |
| `capsule_import` | Import capsules from JSONL file |
| `capsule_purge` | Permanently delete soft-deleted capsules |
| `capsule_bulk_delete` | Soft-delete multiple capsules by filter |
| `capsule_bulk_update` | Update metadata on multiple capsules |
| `capsule_compose` | Assemble multiple capsules into bundle |

---

## CLI Usage

The CLI provides direct command-line access to capsule operations. Output is JSON.

### Commands

```bash
# Store a capsule (reads from stdin)
echo "## Objective
..." | moss store --name=auth --workspace=myproject

# Fetch by name
moss fetch --name=auth --workspace=myproject

# Fetch by ID
moss fetch 01KFPRNV1JEK4F870H1K84XS6S

# Update (metadata only)
moss update --name=auth --title="New Title"

# Update with new content (from stdin)
echo "## Objective
..." | moss update --name=auth

# Delete (soft delete)
moss delete --name=auth

# List capsules in workspace
moss list --workspace=myproject

# List all capsules
moss inventory

# Get latest in workspace
moss latest --workspace=myproject --include-text

# Export to file (default-safe location)
moss export --path=~/.moss/exports/backup.jsonl

# Import from file
moss import --path=~/.moss/exports/backup.jsonl --mode=replace

# Purge deleted capsules
moss purge --older-than=7d

# List MCP tools with enabled/disabled status
moss tools
```

### Common Flags

| Flag | Description |
|------|-------------|
| `--workspace, -w` | Workspace name (default: "default") |
| `--name, -n` | Capsule name |
| `--include-deleted` | Include soft-deleted capsules |
| `--limit, -l` | Max items to return |
| `--offset, -o` | Items to skip |

### Mode vs MCP

- **No arguments (terminal)**: Shows banner and usage hint
- **No arguments (piped input)**: Starts MCP server (stdio transport)
- **Subcommand**: Runs CLI command (e.g., `moss store`, `moss fetch`)
- **--help / --version**: Shows help or version

---

## Configuration

### Config File

Moss loads config from two locations:

| Location | Scope | Priority |
|----------|-------|----------|
| `~/.moss/config.json` | Global (user) | Lower |
| `.moss/config.json` | Repo (project) | Higher |

**Repo config discovery:** Moss walks upward from the current working directory to find the nearest `.moss/config.json`. This means running from a subdirectory (e.g., `src/`) still finds the repo root config.

**Merge behavior:**
- Scalars: repo overrides global (if non-zero)
- Booleans: OR (either true → true)
- Arrays (`allowed_paths`, `disabled_tools`): merged and deduplicated

```json
{
  "capsule_max_chars": 12000,
  "allowed_paths": [],
  "allow_unsafe_paths": false,
  "db_max_open_conns": 0,
  "db_max_idle_conns": 0,
  "disabled_tools": []
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `capsule_max_chars` | 12000 | Maximum characters per capsule (~3k tokens) |
| `allowed_paths` | `[]` | Additional directories allowed for import/export |
| `allow_unsafe_paths` | `false` | Bypass directory restrictions (symlink checks still apply) |
| `db_max_open_conns` | 0 | Max open DB connections (0 = unlimited; set to 1 if you hit "database is locked") |
| `db_max_idle_conns` | 0 | Max idle DB connections (0 = default; typically match `db_max_open_conns`) |
| `disabled_tools` | `[]` | MCP tool names to exclude from registration |

If the file doesn't exist, defaults are used.

### Tool Filtering

Disable specific MCP tools by adding their names to `disabled_tools`. This is useful for hiding destructive tools like `capsule_purge` or `capsule_bulk_delete` from agents.

```json
{
  "disabled_tools": ["capsule_purge", "capsule_bulk_delete", "capsule_bulk_update"]
}
```

**Behavior:**
- All 15 tools are enabled by default (see [Available Tools](#available-tools))
- Disabled tools are not registered with the MCP server
- Unknown tool names trigger a warning on startup
- New tools added in future versions are auto-enabled (blocklist approach)

### Import/Export Path Security

By default, `capsule_export` and `capsule_import` (and the CLI `moss export` / `moss import`) are restricted to `~/.moss/exports/` to prevent accidental writes to sensitive locations.

**To allow additional directories:**

```json
{
  "allowed_paths": ["/tmp/moss-backups", "/home/user/capsule-exports"]
}
```

Note: `allowed_paths` entries must be absolute paths (relative paths are ignored).

**To bypass directory restrictions (not recommended):**

```json
{
  "allow_unsafe_paths": true
}
```

**Security checks performed:**
- `.jsonl` extension required
- Directory traversal (`..`) rejected
- Subdirectories not allowed: files must be directly in an allowed directory (prevents TOCTOU attacks)
- Symlink files rejected (`O_NOFOLLOW` on Unix; validation check on all platforms)
- Parent directory symlinks rejected

### Database

Location: `~/.moss/moss.db` (SQLite)

The database is created automatically on first run. Recommended permissions:
- Directory `~/.moss/`: `0700`
- Database file: `0600`

### Exports

Default export location: `~/.moss/exports/`

Files are named: `<workspace>-<timestamp>.jsonl` or `all-<timestamp>.jsonl`

---

## Verification Tests

### 1. List Tools

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | moss
```

Expected: JSON response listing 15 tools.

### 2. Inventory (Empty Store)

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"capsule_inventory","arguments":{}}}' | moss
```

Expected: `{"items":[],"pagination":{"limit":100,"offset":0,"has_more":false,"total":0},"sort":"updated_at_desc"}`

### 3. Store and Fetch

```bash
# Store a capsule
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"capsule_store","arguments":{"capsule_text":"## Objective\nTest\n## Current status\nTesting\n## Decisions\nNone\n## Next actions\nVerify\n## Key locations\n./test\n## Open questions\nNone","name":"test","workspace":"default"}}}' | moss

# Fetch it back
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"capsule_fetch","arguments":{"workspace":"default","name":"test"}}}' | moss
```

### 4. Error Cases

**Missing sections (422):**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"capsule_store","arguments":{"capsule_text":"too short"}}}' | moss
```

Expected: `isError: true` with `code: "CAPSULE_TOO_THIN"`

**Ambiguous addressing (400):**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"capsule_fetch","arguments":{"id":"01ABC","workspace":"default","name":"test"}}}' | moss
```

Expected: `isError: true` with `code: "AMBIGUOUS_ADDRESSING"`

---

## Common Operations

### Store a Capsule

```
capsule_store {
  "workspace": "myproject",
  "name": "auth",
  "capsule_text": "## Objective\n...\n## Current status\n...\n## Decisions\n...\n## Next actions\n...\n## Key locations\n...\n## Open questions\n..."
}
```

### Fetch by Name

```
capsule_fetch { "workspace": "myproject", "name": "auth" }
```

### Fetch by ID

```
capsule_fetch { "id": "01KFPRNV1JEK4F870H1K84XS6S" }
```

### Batch Fetch Multiple Capsules

```
capsule_fetch_many {
  "items": [
    { "workspace": "myproject", "name": "research" },
    { "workspace": "myproject", "name": "design" },
    { "id": "01KFPRNV1JEK4F870H1K84XS6S" }
  ]
}
```

Partial success is allowed — found capsules in `items`, failures in `errors`.

### List All Capsules

```
capsule_inventory {}
```

### Export for Backup

```
capsule_export { "path": "~/.moss/exports/moss-backup.jsonl" }
```

### Import from Backup

```
capsule_import { "path": "~/.moss/exports/moss-backup.jsonl", "mode": "error" }
```

### Compose Multiple Capsules

```
capsule_compose {
  "items": [
    { "workspace": "myproject", "name": "research" },
    { "workspace": "myproject", "name": "design" }
  ],
  "format": "markdown"
}
```

With optional storage:

```
capsule_compose {
  "items": [
    { "workspace": "myproject", "name": "research" },
    { "workspace": "myproject", "name": "design" }
  ],
  "store_as": {
    "workspace": "myproject",
    "name": "combined",
    "mode": "replace"
  }
}
```

**Note:** `store_as` requires `format:"markdown"` (the default). Using `format:"json"` with `store_as` returns an error because JSON output lacks section headers required for capsule lint.

### Search Capsules

```
capsule_search { "query": "authentication" }
```

With filters:

```
capsule_search {
  "query": "JWT OR OAuth",
  "workspace": "myproject",
  "phase": "design"
}
```

**Query syntax:**
- Simple: `authentication`
- Phrase: `"user authentication"`
- Prefix: `auth*`
- Boolean: `JWT OR OAuth`, `Redis AND cache`, `NOT deprecated`

Results are ranked by relevance (title matches weighted 5x higher). Snippets are HTML-safe: user content is escaped; only `<b>` highlight tags are present.

### Bulk Delete by Filter

```
capsule_bulk_delete { "workspace": "scratch" }
```

Expected:
```json
{
  "deleted": 5,
  "message": "Soft-deleted 5 capsules matching workspace=\"scratch\""
}
```

Multiple filters (AND semantics):

```
capsule_bulk_delete {
  "workspace": "myproject",
  "tag": "stale",
  "phase": "research"
}
```

At least one filter is required. Calling with no filters returns an error:

```
capsule_bulk_delete {}
```

Expected: `isError: true` with `code: "INVALID_REQUEST"`

Note: whitespace-only filters are treated as empty and rejected.

### Bulk Update by Filter

```
capsule_bulk_update { "workspace": "myproject", "set_phase": "archived" }
```

Expected:
```json
{
  "updated": 5,
  "message": "Updated 5 capsules matching workspace=\"myproject\"; set phase=\"archived\""
}
```

Multiple filters and updates:

```
capsule_bulk_update {
  "workspace": "myproject",
  "tag": "completed",
  "set_phase": "archived",
  "set_role": "done"
}
```

Clear a field with empty string:

```
capsule_bulk_update { "workspace": "scratch", "set_phase": "" }
```

At least one filter AND one update field is required:

```
capsule_bulk_update { "workspace": "test" }  // Error: no update fields
capsule_bulk_update { "set_phase": "done" }  // Error: no filters
```

Expected: `isError: true` with `code: "INVALID_REQUEST"`

Note: whitespace-only filters are treated as empty and rejected.

---

## Orchestration

Multi-agent workflows can use `run_id`, `phase`, and `role` to scope capsules.

### Store with Orchestration

```
capsule_store {
  "workspace": "myproject",
  "name": "design-intent",
  "run_id": "pr-review-abc123",
  "phase": "design",
  "role": "design-intent",
  "capsule_text": "## Objective\n..."
}
```

### Filter by Run ID

```
capsule_list {
  "workspace": "myproject",
  "run_id": "pr-review-abc123"
}
```

### Latest Design Capsule from Run

```
capsule_latest {
  "workspace": "myproject",
  "run_id": "pr-review-abc123",
  "phase": "design",
  "include_text": true
}
```

### Cross-Workspace Run Query

```
capsule_inventory {
  "run_id": "pr-review-abc123"
}
```

---

## Troubleshooting

### "database is locked"

The capsule store uses SQLite WAL mode to reduce lock contention. If you see this error:
1. Ensure only one MCP server instance is running
2. Check for stale lock files in `~/.moss/`

### Tool not found in Claude Code

1. Verify `~/.claude/settings.json` has correct path
2. Restart Claude Code after config changes
3. Check binary is executable: `chmod +x /path/to/moss`

### CAPSULE_TOO_THIN errors

Your capsule is missing required sections. Include all 6:
1. Objective
2. Current status
3. Decisions / constraints
4. Next actions
5. Key locations
6. Open questions / risks

Use `allow_thin: true` to bypass (not recommended for real capsules).

### CAPSULE_TOO_LARGE errors

Capsule exceeds `capsule_max_chars` (default: 12000). Options:
1. Compress the capsule content
2. Increase limit in `~/.moss/config.json`

### Import Collisions

- `mode: "error"` (default): Fails on any collision. Use when importing to empty store.
- `mode: "replace"`: Overwrites existing. Use for merging/syncing.
- `mode: "rename"`: Auto-suffixes names on collision. Use for preserving both versions.

### Reset Database

To start fresh, delete the database file:

```bash
rm ~/.moss/moss.db
```

A new empty database is created automatically on the next command.

---

## Logs and Debugging

The `moss` binary writes to stderr for errors. To capture:

```bash
moss 2>moss.log
```

For verbose protocol debugging, inspect the JSON-RPC messages directly:

```bash
# Wrap moss to log I/O
tee input.log | moss 2>error.log | tee output.log
```
