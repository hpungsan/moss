# Moss Setup

Installation, configuration, and CLI usage. Primitive-specific operations live in the primitive runbooks.

## Install

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

Download from GitHub Releases:

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
# If installed to PATH:
moss --version
moss --help

# If built locally:
./bin/moss --version
./bin/moss --help

# Test MCP protocol (piped input = MCP server mode)
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./bin/moss
```

---

## Storage

| Path | Description |
|------|-------------|
| `~/.moss/moss.db` | SQLite database |
| `~/.moss/config.json` | Global config |
| `.moss/config.json` | Repo config (overrides global) |
| `~/.moss/exports/` | Default export location |

---

## Claude Code Integration

Add Moss as an MCP server via `.mcp.json` in your project root.

### MCP Server Configuration

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

---

## CLI Usage

The CLI provides direct command-line access to Moss operations. Output is JSON.

### Mode Detection

- **No arguments (terminal)**: Shows banner and usage hint
- **No arguments (piped input)**: Starts MCP server (stdio transport)
- **Subcommand**: Runs CLI command (e.g., `moss store`, `moss fetch`)
- **--help / --version**: Shows help or version

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

---

## Configuration

### Config Files

Moss loads config from two locations:

| Location | Scope | Priority |
|----------|-------|----------|
| `~/.moss/config.json` | Global (user) | Lower |
| `.moss/config.json` | Repo (project) | Higher |

**Repo config discovery:** Moss walks upward from the current working directory to find the nearest `.moss/config.json`. This means running from a subdirectory (e.g., `src/`) still finds the repo root config.

**Merge behavior:**
- Scalars: repo overrides global (if non-zero)
- Booleans: OR (either true → true)
- Arrays (`allowed_paths`, `disabled_tools`, `disabled_primitives`): merged and deduplicated

### Config Fields

```json
{
  "capsule_max_chars": 12000,
  "allowed_paths": [],
  "allow_unsafe_paths": false,
  "db_max_open_conns": 0,
  "db_max_idle_conns": 0,
  "disabled_tools": [],
  "disabled_primitives": []
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
| `disabled_primitives` | `[]` | Primitive names to disable entirely (e.g., `["capsule"]` disables all capsule tools) |

If the file doesn't exist, defaults are used.

### Tool Filtering

Disable specific MCP tools by adding their names to `disabled_tools`. This is useful for hiding destructive tools like `capsule_purge` or `capsule_bulk_delete` from agents.

```json
{
  "disabled_tools": ["capsule_purge", "capsule_bulk_delete", "capsule_bulk_update"]
}
```

**Behavior:**
- All tools are enabled by default
- Disabled tools are not registered with the MCP server
- Unknown tool names trigger a warning on startup
- New tools added in future versions are auto-enabled (blocklist approach)

### Primitive Filtering

Disable entire primitive types by adding their names to `disabled_primitives`. This disables all tools belonging to that primitive.

```json
{
  "disabled_primitives": ["capsule"]
}
```

Known primitives: `capsule` (artifact coming soon).

**Behavior:**
- Primitives are extracted from tool names (e.g., `capsule_store` → `capsule`)
- All tools belonging to disabled primitives are excluded from registration
- Unknown primitive names trigger a warning on startup
- Can be combined with `disabled_tools` for fine-grained control

**Example: disable primitive, but also block a specific tool:**

```json
{
  "disabled_primitives": ["capsule"],
  "disabled_tools": ["artifact_purge"]
}
```

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

---

## Verification Tests

If Moss is not installed in your PATH, replace `moss` with `./bin/moss` in the commands below.

### 1. List Tools

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | moss
```

Expected: JSON response listing available tools.

### 2. Inventory (Empty Store)

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"capsule_inventory","arguments":{}}}' | moss
```

Expected: `{"items":[],"pagination":{"limit":100,"offset":0,"has_more":false,"total":0},"sort":"updated_at_desc"}`

---

## Troubleshooting

### "database is locked"

Moss uses SQLite WAL mode to reduce lock contention. If you see this error:
1. Ensure only one MCP server instance is running
2. Check for stale lock files in `~/.moss/`

### Tool not found in Claude Code

1. Verify `.mcp.json` has correct path
2. Restart Claude Code after config changes
3. Check binary is executable: `chmod +x /path/to/moss`

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

---

## Primitive Runbooks

- Capsules: [docs/capsule/RUNBOOK.md](capsule/RUNBOOK.md)
