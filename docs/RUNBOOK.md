# Moss v1 Runbook

Operational guide for building, configuring, and running Moss.

## Building

### From Source

```bash
git clone https://github.com/hpungsan/moss.git
cd moss
```

Then choose one:

| Goal | Command |
|------|---------|
| Build locally | `go build -o bin/moss ./cmd/moss` |
| Install to PATH | `go install ./cmd/moss` |

**Alternative: Using Makefile**

| Goal | Command |
|------|---------|
| Build locally | `make build` → `bin/moss` |
| Install to PATH | `make install` → `$GOPATH/bin/moss` |
| Build with version | `make build-release VERSION=1.0.0` |
| Cross-compile all platforms | `make build-all VERSION=1.0.0` |
| Cross-compile + checksums | `make build-checksums VERSION=1.0.0` |

### Pre-built Binaries

Download from [GitHub Releases](https://github.com/hpungsan/moss/releases):

| Platform | Binary |
|----------|--------|
| macOS (Apple Silicon) | `moss-darwin-arm64` |
| macOS (Intel) | `moss-darwin-amd64` |
| Linux (x64) | `moss-linux-amd64` |
| Linux (ARM64) | `moss-linux-arm64` |
| Windows (x64) | `moss-windows-amd64.exe` |

```bash
# Example: macOS Apple Silicon
curl -LO https://github.com/hpungsan/moss/releases/latest/download/moss-darwin-arm64
chmod +x moss-darwin-arm64
sudo mv moss-darwin-arm64 /usr/local/bin/moss
```

### Verify Build

```bash
# Check version
./moss --version

# Show help
./moss --help

# Test MCP protocol (piped input = MCP server mode)
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./moss
# Expected: {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05",...}}
```

---

## Claude Code Integration

Add Moss as an MCP server via `.mcp.json` in your project root.

### Configuration

Create `.mcp.json` in your project root:

```json
{
  "mcpServers": {
    "moss": {
      "command": "/path/to/moss",
      "args": ["mcp"]
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
| `mcp__moss__fetch` | Only fetch |
| `mcp__moss__store` | Only store |
| `mcp__moss__list` | Only list |

**Why this matters:** In swarm patterns, workers run autonomously in background. Manual approval would block the workflow. Pre-approving Moss lets agents share context without human intervention.

### Verify Integration

1. **Restart Claude Code** (or start a new session) to load the MCP server
2. Ask: "Use inventory to list all capsules"
3. Expected: Tool call succeeds with `items: []` (empty store) or list of existing capsules


### Available Tools

| Tool | Description |
|------|-------------|
| `store` | Create a new capsule |
| `fetch` | Retrieve a capsule by ID or name |
| `fetch_many` | Batch fetch multiple capsules |
| `update` | Update an existing capsule |
| `delete` | Soft-delete a capsule |
| `latest` | Get most recent capsule in workspace |
| `list` | List capsules in a workspace |
| `inventory` | List all capsules across workspaces |
| `export` | Export capsules to JSONL file |
| `import` | Import capsules from JSONL file |
| `purge` | Permanently delete soft-deleted capsules |
| `bulk_delete` | Soft-delete multiple capsules by filter |
| `bulk_update` | Update metadata on multiple capsules |
| `compose` | Assemble multiple capsules into bundle |

---

## CLI Usage

The CLI provides direct command-line access to all Moss operations. Output is JSON.

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

Location: `~/.moss/config.json`

```json
{
  "capsule_max_chars": 12000,
  "allowed_paths": [],
  "allow_unsafe_paths": false
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `capsule_max_chars` | 12000 | Maximum characters per capsule (~3k tokens) |
| `allowed_paths` | `[]` | Additional directories allowed for import/export |
| `allow_unsafe_paths` | `false` | Bypass directory restrictions (symlink checks still apply) |

If the file doesn't exist, defaults are used.

### Import/Export Path Security

By default, `export` and `import` are restricted to `~/.moss/exports/` to prevent accidental writes to sensitive locations.

**To allow additional directories:**

```json
{
  "allowed_paths": ["/tmp/moss-backups", "/home/user/capsule-exports"]
}
```

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
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./moss
```

Expected: JSON response listing 14 tools.

### 2. Inventory (Empty Store)

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"inventory","arguments":{}}}' | ./moss
```

Expected: `{"items":[],"pagination":{"limit":100,"offset":0,"has_more":false,"total":0},"sort":"updated_at_desc"}`

### 3. Store and Fetch

```bash
# Store a capsule
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"store","arguments":{"capsule_text":"## Objective\nTest\n## Current status\nTesting\n## Decisions\nNone\n## Next actions\nVerify\n## Key locations\n./test\n## Open questions\nNone","name":"test","workspace":"default"}}}' | ./moss

# Fetch it back
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"fetch","arguments":{"workspace":"default","name":"test"}}}' | ./moss
```

### 4. Error Cases

**Missing sections (422):**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"store","arguments":{"capsule_text":"too short"}}}' | ./moss
```

Expected: `isError: true` with `code: "CAPSULE_TOO_THIN"`

**Ambiguous addressing (400):**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"fetch","arguments":{"id":"01ABC","workspace":"default","name":"test"}}}' | ./moss
```

Expected: `isError: true` with `code: "AMBIGUOUS_ADDRESSING"`

---

## Common Operations

### Store a Capsule

```
store {
  "workspace": "myproject",
  "name": "auth",
  "capsule_text": "## Objective\n...\n## Current status\n...\n## Decisions\n...\n## Next actions\n...\n## Key locations\n...\n## Open questions\n..."
}
```

### Fetch by Name

```
fetch { "workspace": "myproject", "name": "auth" }
```

### Fetch by ID

```
fetch { "id": "01KFPRNV1JEK4F870H1K84XS6S" }
```

### Batch Fetch Multiple Capsules

```
fetch_many {
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
inventory {}
```

### Export for Backup

```
export { "path": "~/.moss/exports/moss-backup.jsonl" }
```

### Import from Backup

```
import { "path": "~/.moss/exports/moss-backup.jsonl", "mode": "error" }
```

### Compose Multiple Capsules

```
compose {
  "items": [
    { "workspace": "myproject", "name": "research" },
    { "workspace": "myproject", "name": "design" }
  ],
  "format": "markdown"
}
```

With optional storage:

```
compose {
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

### Bulk Delete by Filter

```
bulk_delete { "workspace": "scratch" }
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
bulk_delete {
  "workspace": "myproject",
  "tag": "stale",
  "phase": "research"
}
```

At least one filter is required. Calling with no filters returns an error:

```
bulk_delete {}
```

Expected: `isError: true` with `code: "INVALID_REQUEST"`

### Bulk Update by Filter

```
bulk_update { "workspace": "myproject", "set_phase": "archived" }
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
bulk_update {
  "workspace": "myproject",
  "tag": "completed",
  "set_phase": "archived",
  "set_role": "done"
}
```

Clear a field with empty string:

```
bulk_update { "workspace": "scratch", "set_phase": "" }
```

At least one filter AND one update field is required:

```
bulk_update { "workspace": "test" }  // Error: no update fields
bulk_update { "set_phase": "done" }  // Error: no filters
```

Expected: `isError: true` with `code: "INVALID_REQUEST"`

---

## Orchestration

Multi-agent workflows can use `run_id`, `phase`, and `role` to scope capsules.

### Store with Orchestration

```
store {
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
list {
  "workspace": "myproject",
  "run_id": "pr-review-abc123"
}
```

### Latest Design Capsule from Run

```
latest {
  "workspace": "myproject",
  "run_id": "pr-review-abc123",
  "phase": "design",
  "include_text": true
}
```

### Cross-Workspace Run Query

```
inventory {
  "run_id": "pr-review-abc123"
}
```

---

## Troubleshooting

### "database is locked"

Moss uses SQLite WAL mode to prevent this. If you see this error:
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

Moss writes to stderr for errors. To capture:

```bash
./moss 2>moss.log
```

For verbose protocol debugging, inspect the JSON-RPC messages directly:

```bash
# Wrap moss to log I/O
tee input.log | ./moss 2>error.log | tee output.log
```
