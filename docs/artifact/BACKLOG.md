# Artifact Backlog

---

## Query Enhancements

### `include_data: false`

Option on `artifact_list` for metadata-only listing. Edge case where data payloads are large and caller only needs IDs/metadata for routing.

### `artifact_search`

FTS5 full-text search on rendered `text` field for discovery across runs. Useful for debugging failed runs or finding prior art.

---

## Rendering

### Moss-side Rendering

Register renderers per `kind`, auto-generate `text` from `data` if omitted. Convenience feature; currently caller renders.

```typescript
moss.registerRenderer("explorer-finding", (data) => `...`);

// Then store without text — Moss renders
await moss.artifact_store({
  kind: "explorer-finding",
  data: output,
  // text omitted — Moss calls registered renderer
});
```

---

## Performance

### JSON Field Indexing

Generated columns for common `data` field queries. Currently `artifact_list` returns `data` and caller filters client-side. Fine for per-run queries (10-20 artifacts).

If v2 cross-run queries become slow (e.g., "all runs where `data.status = 'BLOCKED'`"), add targeted indexes:

```sql
ALTER TABLE artifacts ADD COLUMN status_idx TEXT
  GENERATED ALWAYS AS (json_extract(data_json, '$.status'));
CREATE INDEX idx_artifacts_status ON artifacts(kind, status_idx)
  WHERE kind = 'run-record';
```

Per-field, not generic. Add indexes for specific high-frequency query patterns as needed.

### StepRecord Events Persistence

Events stored inside RunRecord artifact. Each event append requires refetch→add→store. Fine for v1 (20-30 events per run). If overhead becomes noticeable, consider separate step-events artifact per step.

---

## Limits

### 50K JSON Limit

Current limit is 50K JSON + 12K text. Rough math: 50 steps × 500 chars = 25K, well under limit. If Extended Trace Fields (FINN backlog) push sizes higher, add `--minimal-trace` flag or increase limit.
