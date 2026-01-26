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

# Test MCP protocol (no args = MCP server mode)
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./moss
# Expected: {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05",...}}
```

---

## Claude Code Integration

Add Moss as an MCP server in your Claude Code settings.

### Configuration

Edit `~/.claude/settings.json`:

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
- If built locally: use absolute path like `/Users/you/moss/moss`

### Verify Integration

1. Start a new Claude Code session
2. Ask: "Use moss.inventory to list all capsules"
3. Expected: Tool call succeeds with `items: []` (empty store) or list of existing capsules

### Available Tools

| Tool | Description |
|------|-------------|
| `moss.store` | Create a new capsule |
| `moss.fetch` | Retrieve a capsule by ID or name |
| `moss.fetch_many` | Batch fetch multiple capsules |
| `moss.update` | Update an existing capsule |
| `moss.delete` | Soft-delete a capsule |
| `moss.latest` | Get most recent capsule in workspace |
| `moss.list` | List capsules in a workspace |
| `moss.inventory` | List all capsules across workspaces |
| `moss.export` | Export capsules to JSONL file |
| `moss.import` | Import capsules from JSONL file |
| `moss.purge` | Permanently delete soft-deleted capsules |

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

# Export to file
moss export --path=/tmp/backup.jsonl

# Import from file
moss import --path=/tmp/backup.jsonl --mode=replace

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

- **No arguments**: Starts MCP server (stdio transport)
- **Subcommand**: Runs CLI command (e.g., `moss store`, `moss fetch`)
- **--help / --version**: Shows help or version

---

## Configuration

### Config File

Location: `~/.moss/config.json`

```json
{
  "capsule_max_chars": 12000
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `capsule_max_chars` | 12000 | Maximum characters per capsule (~3k tokens) |

If the file doesn't exist, defaults are used.

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

Expected: JSON response listing 11 tools.

### 2. Inventory (Empty Store)

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"moss.inventory","arguments":{}}}' | ./moss
```

Expected: `{"items":[],"pagination":{"limit":100,"offset":0,"has_more":false,"total":0},"sort":"updated_at_desc"}`

### 3. Store and Fetch

```bash
# Store a capsule
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"moss.store","arguments":{"capsule_text":"## Objective\nTest\n## Current status\nTesting\n## Decisions\nNone\n## Next actions\nVerify\n## Key locations\n./test\n## Open questions\nNone","name":"test","workspace":"default"}}}' | ./moss

# Fetch it back
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"moss.fetch","arguments":{"workspace":"default","name":"test"}}}' | ./moss
```

### 4. Error Cases

**Missing sections (422):**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"moss.store","arguments":{"capsule_text":"too short"}}}' | ./moss
```

Expected: `isError: true` with `code: "CAPSULE_TOO_THIN"`

**Ambiguous addressing (400):**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"moss.fetch","arguments":{"id":"01ABC","workspace":"default","name":"test"}}}' | ./moss
```

Expected: `isError: true` with `code: "AMBIGUOUS_ADDRESSING"`

---

## Common Operations

### Store a Capsule

```
moss.store {
  "workspace": "myproject",
  "name": "auth",
  "capsule_text": "## Objective\n...\n## Current status\n...\n## Decisions\n...\n## Next actions\n...\n## Key locations\n...\n## Open questions\n..."
}
```

### Fetch by Name

```
moss.fetch { "workspace": "myproject", "name": "auth" }
```

### Fetch by ID

```
moss.fetch { "id": "01KFPRNV1JEK4F870H1K84XS6S" }
```

### List All Capsules

```
moss.inventory {}
```

### Export for Backup

```
moss.export { "path": "/tmp/moss-backup.jsonl" }
```

### Import from Backup

```
moss.import { "path": "/tmp/moss-backup.jsonl", "mode": "error" }
```

---

## Orchestration

Multi-agent workflows can use `run_id`, `phase`, and `role` to scope capsules.

### Store with Orchestration

```
moss.store {
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
moss.list {
  "workspace": "myproject",
  "run_id": "pr-review-abc123"
}
```

### Latest Design Capsule from Run

```
moss.latest {
  "workspace": "myproject",
  "run_id": "pr-review-abc123",
  "phase": "design",
  "include_text": true
}
```

### Cross-Workspace Run Query

```
moss.inventory {
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
