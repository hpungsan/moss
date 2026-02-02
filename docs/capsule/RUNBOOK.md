# Capsule Runbook

Operational guide for the Capsule primitive: MCP tools, common operations, orchestration patterns, and troubleshooting.

**Prereq:** Install and configure Moss first via [SETUP.md](../SETUP.md).

---

## Available Tools

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

Expected: `isError: true` with `code: "INVALID_REQUEST"` and message `"at least one filter is required"`.

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

Expected: `isError: true` with `code: "INVALID_REQUEST"` and one of:
- `"at least one update field is required"` (when no update fields are provided)
- `"at least one filter is required"` (when no filters are provided)

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

## Verification Tests

### Store and Fetch

```bash
# Store a capsule
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"capsule_store","arguments":{"capsule_text":"## Objective\nTest\n## Current status\nTesting\n## Decisions\nNone\n## Next actions\nVerify\n## Key locations\n./test\n## Open questions\nNone","name":"test","workspace":"default"}}}' | moss

# Fetch it back
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"capsule_fetch","arguments":{"workspace":"default","name":"test"}}}' | moss
```

### Error Cases

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

## Troubleshooting

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
