package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/errors"
)

// Querier is an interface satisfied by both *sql.DB and *sql.Tx.
// This allows functions to work with either a database connection or a transaction.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Insert stores a new capsule in the database.
func Insert(ctx context.Context, q Querier, c *capsule.Capsule) error {
	// Convert tags to JSON
	var tagsJSON sql.NullString
	if len(c.Tags) > 0 {
		data, err := json.Marshal(c.Tags)
		if err != nil {
			return errors.NewInternal(err)
		}
		tagsJSON = sql.NullString{String: string(data), Valid: true}
	}

	// Convert nullable fields
	nameRaw := toNullString(c.NameRaw)
	nameNorm := toNullString(c.NameNorm)
	title := toNullString(c.Title)
	source := toNullString(c.Source)
	runID := toNullString(c.RunID)
	phase := toNullString(c.Phase)
	role := toNullString(c.Role)

	query := `
		INSERT INTO capsules (
			id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_text, capsule_chars, tokens_estimate,
			tags_json, source, run_id, phase, role,
			created_at, updated_at, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
	`

	_, err := q.ExecContext(ctx, query,
		c.ID, c.WorkspaceRaw, c.WorkspaceNorm, nameRaw, nameNorm,
		title, c.CapsuleText, c.CapsuleChars, c.TokensEstimate,
		tagsJSON, source, runID, phase, role,
		c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		if isNameUniquenessViolation(err) && c.NameRaw != nil {
			return errors.NewNameAlreadyExists(c.WorkspaceRaw, *c.NameRaw)
		}
		return errors.NewInternal(err)
	}

	return nil
}

func isNameUniquenessViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// SQLite typically formats this as:
	// "constraint failed: UNIQUE constraint failed: capsules.workspace_norm, capsules.name_norm (2067)"
	return strings.Contains(msg, "UNIQUE constraint failed") &&
		strings.Contains(msg, "capsules.workspace_norm") &&
		strings.Contains(msg, "capsules.name_norm")
}

// UpsertResult contains the result of an Upsert operation.
type UpsertResult struct {
	ID        string // The final capsule ID (existing on update, new on insert)
	WasUpdate bool   // True if an existing capsule was updated
}

// Upsert atomically inserts a new capsule or updates an existing one with the same name.
// This is used for mode:replace to avoid race conditions between concurrent callers.
//
// For named capsules: On conflict with (workspace_norm, name_norm), updates the existing row.
// For unnamed capsules (name is nil): Always inserts (no conflict possible).
//
// On update, preserves: id, workspace_raw/norm, name_raw/norm, created_at
// On update, changes: capsule_text, title, tags, source, run_id, phase, role, updated_at, metrics
func Upsert(ctx context.Context, q Querier, c *capsule.Capsule) (*UpsertResult, error) {
	// Convert tags to JSON
	var tagsJSON sql.NullString
	if len(c.Tags) > 0 {
		data, err := json.Marshal(c.Tags)
		if err != nil {
			return nil, errors.NewInternal(err)
		}
		tagsJSON = sql.NullString{String: string(data), Valid: true}
	}

	// Convert nullable fields
	nameRaw := toNullString(c.NameRaw)
	nameNorm := toNullString(c.NameNorm)
	title := toNullString(c.Title)
	source := toNullString(c.Source)
	runID := toNullString(c.RunID)
	phase := toNullString(c.Phase)
	role := toNullString(c.Role)

	// Use SQLite UPSERT syntax with partial index conflict target.
	// The conflict target matches our unique partial index:
	//   idx_capsules_workspace_name_norm ON (workspace_norm, name_norm)
	//   WHERE name_norm IS NOT NULL AND deleted_at IS NULL
	//
	// When name_norm IS NULL, the partial index doesn't apply, so no conflict occurs.
	// When there's a conflict, we update the mutable fields and preserve the original ID.
	//
	// RETURNING id gives us the final row's ID:
	// - On insert: the new ID we provided (c.ID)
	// - On conflict/update: the existing capsule's ID (preserved)
	// We determine WasUpdate by comparing the returned ID with the ID we tried to insert.
	query := `
		INSERT INTO capsules (
			id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_text, capsule_chars, tokens_estimate,
			tags_json, source, run_id, phase, role,
			created_at, updated_at, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
		ON CONFLICT(workspace_norm, name_norm) WHERE name_norm IS NOT NULL AND deleted_at IS NULL
		DO UPDATE SET
			title = excluded.title,
			capsule_text = excluded.capsule_text,
			capsule_chars = excluded.capsule_chars,
			tokens_estimate = excluded.tokens_estimate,
			tags_json = excluded.tags_json,
			source = excluded.source,
			run_id = excluded.run_id,
			phase = excluded.phase,
			role = excluded.role,
			updated_at = excluded.updated_at
		RETURNING id
	`

	var resultID string
	err := q.QueryRowContext(ctx, query,
		c.ID, c.WorkspaceRaw, c.WorkspaceNorm, nameRaw, nameNorm,
		title, c.CapsuleText, c.CapsuleChars, c.TokensEstimate,
		tagsJSON, source, runID, phase, role,
		c.CreatedAt, c.UpdatedAt,
	).Scan(&resultID)

	if err != nil {
		return nil, errors.NewInternal(err)
	}

	return &UpsertResult{
		ID:        resultID,
		WasUpdate: resultID != c.ID, // If IDs differ, we updated an existing row
	}, nil
}

// GetByID retrieves a capsule by its ULID.
// If includeDeleted is false, soft-deleted capsules are excluded.
func GetByID(ctx context.Context, q Querier, id string, includeDeleted bool) (*capsule.Capsule, error) {
	query := `
		SELECT id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_text, capsule_chars, tokens_estimate,
			tags_json, source, run_id, phase, role,
			created_at, updated_at, deleted_at
		FROM capsules
		WHERE id = ?
	`
	if !includeDeleted {
		query += " AND deleted_at IS NULL"
	}

	row := q.QueryRowContext(ctx, query, id)
	c, err := scanCapsule(row)
	if err == sql.ErrNoRows {
		return nil, errors.NewNotFound(id)
	}
	if err != nil {
		return nil, errors.NewInternal(err)
	}

	return c, nil
}

// GetByName retrieves a capsule by normalized workspace and name.
// If includeDeleted is false, soft-deleted capsules are excluded.
func GetByName(ctx context.Context, q Querier, workspaceNorm, nameNorm string, includeDeleted bool) (*capsule.Capsule, error) {
	query := `
		SELECT id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_text, capsule_chars, tokens_estimate,
			tags_json, source, run_id, phase, role,
			created_at, updated_at, deleted_at
		FROM capsules
		WHERE workspace_norm = ? AND name_norm = ?
	`
	if !includeDeleted {
		query += " AND deleted_at IS NULL LIMIT 1"
	} else {
		// If both active and soft-deleted capsules exist for the same name, prefer the active one.
		// If no active capsule exists, return the most recently updated deleted capsule.
		query += " ORDER BY (deleted_at IS NULL) DESC, updated_at DESC LIMIT 1"
	}

	row := q.QueryRowContext(ctx, query, workspaceNorm, nameNorm)
	c, err := scanCapsule(row)
	if err == sql.ErrNoRows {
		return nil, errors.NewNotFound(workspaceNorm + "/" + nameNorm)
	}
	if err != nil {
		return nil, errors.NewInternal(err)
	}

	return c, nil
}

// CheckNameExists checks if an active capsule with the given name exists.
func CheckNameExists(ctx context.Context, q Querier, workspaceNorm, nameNorm string) (bool, error) {
	query := `
		SELECT 1 FROM capsules
		WHERE workspace_norm = ? AND name_norm = ? AND deleted_at IS NULL
		LIMIT 1
	`

	var exists int
	err := q.QueryRowContext(ctx, query, workspaceNorm, nameNorm).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, errors.NewInternal(err)
	}

	return true, nil
}

// UpdateByID updates mutable fields of an existing capsule.
// Sets updated_at to current timestamp.
// Does NOT change: id, workspace, name
func UpdateByID(ctx context.Context, db *sql.DB, c *capsule.Capsule) error {
	// Convert tags to JSON
	var tagsJSON sql.NullString
	if len(c.Tags) > 0 {
		data, err := json.Marshal(c.Tags)
		if err != nil {
			return errors.NewInternal(err)
		}
		tagsJSON = sql.NullString{String: string(data), Valid: true}
	}

	// Convert nullable fields
	title := toNullString(c.Title)
	source := toNullString(c.Source)
	runID := toNullString(c.RunID)
	phase := toNullString(c.Phase)
	role := toNullString(c.Role)

	now := time.Now().Unix()

	query := `
		UPDATE capsules
		SET capsule_text = ?, title = ?, tags_json = ?, source = ?,
			run_id = ?, phase = ?, role = ?,
			capsule_chars = ?, tokens_estimate = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := db.ExecContext(ctx, query,
		c.CapsuleText, title, tagsJSON, source,
		runID, phase, role,
		c.CapsuleChars, c.TokensEstimate, now,
		c.ID,
	)
	if err != nil {
		return errors.NewInternal(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternal(err)
	}
	if rowsAffected == 0 {
		return errors.NewNotFound(c.ID)
	}

	// Update the struct's UpdatedAt field
	c.UpdatedAt = now

	return nil
}

// SoftDelete marks a capsule as deleted by setting deleted_at.
func SoftDelete(ctx context.Context, db *sql.DB, id string) error {
	now := time.Now().Unix()

	query := `
		UPDATE capsules
		SET deleted_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := db.ExecContext(ctx, query, now, id)
	if err != nil {
		return errors.NewInternal(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternal(err)
	}
	if rowsAffected == 0 {
		return errors.NewNotFound(id)
	}

	return nil
}

// scanCapsule scans a single row into a Capsule struct.
func scanCapsule(row *sql.Row) (*capsule.Capsule, error) {
	var (
		c         capsule.Capsule
		nameRaw   sql.NullString
		nameNorm  sql.NullString
		title     sql.NullString
		tagsJSON  sql.NullString
		source    sql.NullString
		runID     sql.NullString
		phase     sql.NullString
		role      sql.NullString
		deletedAt sql.NullInt64
	)

	err := row.Scan(
		&c.ID, &c.WorkspaceRaw, &c.WorkspaceNorm, &nameRaw, &nameNorm,
		&title, &c.CapsuleText, &c.CapsuleChars, &c.TokensEstimate,
		&tagsJSON, &source, &runID, &phase, &role,
		&c.CreatedAt, &c.UpdatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert nullable fields
	c.NameRaw = fromNullString(nameRaw)
	c.NameNorm = fromNullString(nameNorm)
	c.Title = fromNullString(title)
	c.Source = fromNullString(source)
	c.RunID = fromNullString(runID)
	c.Phase = fromNullString(phase)
	c.Role = fromNullString(role)

	// Convert deleted_at
	if deletedAt.Valid {
		c.DeletedAt = &deletedAt.Int64
	}

	// Parse tags JSON
	if tagsJSON.Valid && tagsJSON.String != "" {
		if err := json.Unmarshal([]byte(tagsJSON.String), &c.Tags); err != nil {
			return nil, err
		}
	}

	return &c, nil
}

// toNullString converts a *string to sql.NullString.
func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// fromNullString converts a sql.NullString to *string.
func fromNullString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// escapeLikePattern escapes SQL LIKE wildcards (%, _) and the escape char (\).
func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// scanCapsuleSummary scans a single row into a CapsuleSummary struct.
// Expects columns: id, workspace_raw, workspace_norm, name_raw, name_norm,
// title, capsule_chars, tokens_estimate, tags_json, source, run_id, phase, role,
// created_at, updated_at, deleted_at
func scanCapsuleSummary(scanner interface{ Scan(...any) error }) (*capsule.CapsuleSummary, error) {
	var (
		s         capsule.CapsuleSummary
		nameRaw   sql.NullString
		nameNorm  sql.NullString
		title     sql.NullString
		tagsJSON  sql.NullString
		source    sql.NullString
		runID     sql.NullString
		phase     sql.NullString
		role      sql.NullString
		deletedAt sql.NullInt64
	)

	err := scanner.Scan(
		&s.ID, &s.Workspace, &s.WorkspaceNorm, &nameRaw, &nameNorm,
		&title, &s.CapsuleChars, &s.TokensEstimate,
		&tagsJSON, &source, &runID, &phase, &role,
		&s.CreatedAt, &s.UpdatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert nullable fields
	s.Name = fromNullString(nameRaw)
	s.NameNorm = fromNullString(nameNorm)
	s.Title = fromNullString(title)
	s.Source = fromNullString(source)
	s.RunID = fromNullString(runID)
	s.Phase = fromNullString(phase)
	s.Role = fromNullString(role)

	// Convert deleted_at
	if deletedAt.Valid {
		s.DeletedAt = &deletedAt.Int64
	}

	// Parse tags JSON
	if tagsJSON.Valid && tagsJSON.String != "" {
		if err := json.Unmarshal([]byte(tagsJSON.String), &s.Tags); err != nil {
			return nil, err
		}
	}

	return &s, nil
}

// ListFilters contains optional filters for list operations.
type ListFilters struct {
	RunID *string
	Phase *string
	Role  *string
}

// ListByWorkspace retrieves capsule summaries for a workspace with pagination.
// Returns summaries (no capsule_text) + total count.
// Ordered by updated_at DESC, id DESC (stable pagination).
func ListByWorkspace(ctx context.Context, db *sql.DB, workspaceNorm string, filters ListFilters, limit, offset int, includeDeleted bool) ([]capsule.CapsuleSummary, int, error) {
	// Build WHERE conditions
	conditions := []string{"workspace_norm = ?"}
	args := []any{workspaceNorm}

	if !includeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}
	if filters.RunID != nil {
		conditions = append(conditions, "run_id = ?")
		args = append(args, *filters.RunID)
	}
	if filters.Phase != nil {
		conditions = append(conditions, "phase = ?")
		args = append(args, *filters.Phase)
	}
	if filters.Role != nil {
		conditions = append(conditions, "role = ?")
		args = append(args, *filters.Role)
	}

	whereClause := " WHERE " + strings.Join(conditions, " AND ")

	// Build count query
	countQuery := "SELECT COUNT(*) FROM capsules" + whereClause
	var total int
	if err := db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.NewInternal(err)
	}

	// Build list query
	listQuery := `
		SELECT id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_chars, tokens_estimate, tags_json, source,
			run_id, phase, role, created_at, updated_at, deleted_at
		FROM capsules` + whereClause + " ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?"

	listArgs := append(args, limit, offset)
	rows, err := db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, errors.NewInternal(err)
	}
	defer rows.Close()

	var summaries []capsule.CapsuleSummary
	for rows.Next() {
		s, err := scanCapsuleSummary(rows)
		if err != nil {
			return nil, 0, errors.NewInternal(err)
		}
		summaries = append(summaries, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, errors.NewInternal(err)
	}

	return summaries, total, nil
}

// InventoryFilters contains optional filters for the ListAll operation.
type InventoryFilters struct {
	Workspace  *string // filter by workspace_norm
	Tag        *string // filter by tag using JSON1
	NamePrefix *string // filter by name_norm LIKE 'prefix%'
	RunID      *string // filter by run_id
	Phase      *string // filter by phase
	Role       *string // filter by role
}

// ListAll retrieves capsule summaries across all workspaces with optional filters.
// Returns summaries (no capsule_text) + total count.
// Ordered by updated_at DESC, id DESC (stable pagination).
func ListAll(ctx context.Context, db *sql.DB, filters InventoryFilters, limit, offset int, includeDeleted bool) ([]capsule.CapsuleSummary, int, error) {
	// Build WHERE clauses
	var conditions []string
	var args []any

	if !includeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}
	if filters.Workspace != nil {
		conditions = append(conditions, "workspace_norm = ?")
		args = append(args, *filters.Workspace)
	}
	if filters.Tag != nil {
		conditions = append(conditions, "EXISTS(SELECT 1 FROM json_each(tags_json) WHERE value = ?)")
		args = append(args, *filters.Tag)
	}
	if filters.NamePrefix != nil {
		conditions = append(conditions, "name_norm LIKE ? ESCAPE '\\'")
		args = append(args, escapeLikePattern(*filters.NamePrefix)+"%")
	}
	if filters.RunID != nil {
		conditions = append(conditions, "run_id = ?")
		args = append(args, *filters.RunID)
	}
	if filters.Phase != nil {
		conditions = append(conditions, "phase = ?")
		args = append(args, *filters.Phase)
	}
	if filters.Role != nil {
		conditions = append(conditions, "role = ?")
		args = append(args, *filters.Role)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Build count query
	countQuery := "SELECT COUNT(*) FROM capsules" + whereClause
	var total int
	if err := db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.NewInternal(err)
	}

	// Build list query
	listQuery := `
		SELECT id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_chars, tokens_estimate, tags_json, source,
			run_id, phase, role, created_at, updated_at, deleted_at
		FROM capsules` + whereClause + " ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?"

	listArgs := append(args, limit, offset)
	rows, err := db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, errors.NewInternal(err)
	}
	defer rows.Close()

	var summaries []capsule.CapsuleSummary
	for rows.Next() {
		s, err := scanCapsuleSummary(rows)
		if err != nil {
			return nil, 0, errors.NewInternal(err)
		}
		summaries = append(summaries, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, errors.NewInternal(err)
	}

	return summaries, total, nil
}

// LatestFilters contains optional filters for latest queries.
type LatestFilters struct {
	RunID *string
	Phase *string
	Role  *string
}

// GetLatestSummary retrieves the most recent capsule summary in a workspace.
// Returns summary (no capsule_text).
// Returns nil, nil if workspace is empty (not an error).
func GetLatestSummary(ctx context.Context, db *sql.DB, workspaceNorm string, filters LatestFilters, includeDeleted bool) (*capsule.CapsuleSummary, error) {
	conditions := []string{"workspace_norm = ?"}
	args := []any{workspaceNorm}

	if !includeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}
	if filters.RunID != nil {
		conditions = append(conditions, "run_id = ?")
		args = append(args, *filters.RunID)
	}
	if filters.Phase != nil {
		conditions = append(conditions, "phase = ?")
		args = append(args, *filters.Phase)
	}
	if filters.Role != nil {
		conditions = append(conditions, "role = ?")
		args = append(args, *filters.Role)
	}

	query := `
		SELECT id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_chars, tokens_estimate, tags_json, source,
			run_id, phase, role, created_at, updated_at, deleted_at
		FROM capsules
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY updated_at DESC, id DESC LIMIT 1`

	row := db.QueryRowContext(ctx, query, args...)
	s, err := scanCapsuleSummary(row)
	if err == sql.ErrNoRows {
		return nil, nil // Empty workspace is not an error
	}
	if err != nil {
		return nil, errors.NewInternal(err)
	}

	return s, nil
}

// GetLatestFull retrieves the most recent full capsule (including text) in a workspace.
// Returns nil, nil if workspace is empty (not an error).
func GetLatestFull(ctx context.Context, db *sql.DB, workspaceNorm string, filters LatestFilters, includeDeleted bool) (*capsule.Capsule, error) {
	conditions := []string{"workspace_norm = ?"}
	args := []any{workspaceNorm}

	if !includeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}
	if filters.RunID != nil {
		conditions = append(conditions, "run_id = ?")
		args = append(args, *filters.RunID)
	}
	if filters.Phase != nil {
		conditions = append(conditions, "phase = ?")
		args = append(args, *filters.Phase)
	}
	if filters.Role != nil {
		conditions = append(conditions, "role = ?")
		args = append(args, *filters.Role)
	}

	query := `
		SELECT id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_text, capsule_chars, tokens_estimate,
			tags_json, source, run_id, phase, role,
			created_at, updated_at, deleted_at
		FROM capsules
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY updated_at DESC, id DESC LIMIT 1`

	row := db.QueryRowContext(ctx, query, args...)
	c, err := scanCapsule(row)
	if err == sql.ErrNoRows {
		return nil, nil // Empty workspace is not an error
	}
	if err != nil {
		return nil, errors.NewInternal(err)
	}

	return c, nil
}

// =============================================================================
// Export/Import/Purge Functions
// =============================================================================

// StreamForExport returns a row iterator for exporting capsules.
// The caller is responsible for closing the returned rows.
// Capsules are ordered by created_at ASC for stable export order.
func StreamForExport(ctx context.Context, db *sql.DB, workspace *string, includeDeleted bool) (*sql.Rows, error) {
	var conditions []string
	var args []any

	if !includeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}
	if workspace != nil {
		conditions = append(conditions, "workspace_norm = ?")
		args = append(args, capsule.Normalize(*workspace))
	}

	query := `
		SELECT id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_text, capsule_chars, tokens_estimate,
			tags_json, source, run_id, phase, role,
			created_at, updated_at, deleted_at
		FROM capsules
	`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at ASC, id ASC"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.NewInternal(err)
	}

	return rows, nil
}

// ScanCapsuleFromRows scans a single capsule from sql.Rows.
// This is used for streaming export.
func ScanCapsuleFromRows(rows *sql.Rows) (*capsule.Capsule, error) {
	var (
		c         capsule.Capsule
		nameRaw   sql.NullString
		nameNorm  sql.NullString
		title     sql.NullString
		tagsJSON  sql.NullString
		source    sql.NullString
		runID     sql.NullString
		phase     sql.NullString
		role      sql.NullString
		deletedAt sql.NullInt64
	)

	err := rows.Scan(
		&c.ID, &c.WorkspaceRaw, &c.WorkspaceNorm, &nameRaw, &nameNorm,
		&title, &c.CapsuleText, &c.CapsuleChars, &c.TokensEstimate,
		&tagsJSON, &source, &runID, &phase, &role,
		&c.CreatedAt, &c.UpdatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert nullable fields
	c.NameRaw = fromNullString(nameRaw)
	c.NameNorm = fromNullString(nameNorm)
	c.Title = fromNullString(title)
	c.Source = fromNullString(source)
	c.RunID = fromNullString(runID)
	c.Phase = fromNullString(phase)
	c.Role = fromNullString(role)

	// Convert deleted_at
	if deletedAt.Valid {
		c.DeletedAt = &deletedAt.Int64
	}

	// Parse tags JSON
	if tagsJSON.Valid && tagsJSON.String != "" {
		if err := json.Unmarshal([]byte(tagsJSON.String), &c.Tags); err != nil {
			return nil, err
		}
	}

	return &c, nil
}

// UpdateFull updates all fields of an existing capsule by ID.
// Unlike UpdateByID, this can update workspace and name, and respects provided timestamps.
// Used during import to restore exact capsule state.
func UpdateFull(ctx context.Context, q Querier, c *capsule.Capsule) error {
	// Convert tags to JSON
	var tagsJSON sql.NullString
	if len(c.Tags) > 0 {
		data, err := json.Marshal(c.Tags)
		if err != nil {
			return errors.NewInternal(err)
		}
		tagsJSON = sql.NullString{String: string(data), Valid: true}
	}

	// Convert nullable fields
	nameRaw := toNullString(c.NameRaw)
	nameNorm := toNullString(c.NameNorm)
	title := toNullString(c.Title)
	source := toNullString(c.Source)
	runID := toNullString(c.RunID)
	phase := toNullString(c.Phase)
	role := toNullString(c.Role)
	var deletedAt sql.NullInt64
	if c.DeletedAt != nil {
		deletedAt = sql.NullInt64{Int64: *c.DeletedAt, Valid: true}
	}

	query := `
		UPDATE capsules
		SET workspace_raw = ?, workspace_norm = ?, name_raw = ?, name_norm = ?,
			title = ?, capsule_text = ?, capsule_chars = ?, tokens_estimate = ?,
			tags_json = ?, source = ?, run_id = ?, phase = ?, role = ?,
			created_at = ?, updated_at = ?, deleted_at = ?
		WHERE id = ?
	`

	result, err := q.ExecContext(ctx, query,
		c.WorkspaceRaw, c.WorkspaceNorm, nameRaw, nameNorm,
		title, c.CapsuleText, c.CapsuleChars, c.TokensEstimate,
		tagsJSON, source, runID, phase, role,
		c.CreatedAt, c.UpdatedAt, deletedAt,
		c.ID,
	)
	if err != nil {
		if isNameUniquenessViolation(err) && c.NameRaw != nil {
			return errors.NewNameAlreadyExists(c.WorkspaceRaw, *c.NameRaw)
		}
		return errors.NewInternal(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternal(err)
	}
	if rowsAffected == 0 {
		return errors.NewNotFound(c.ID)
	}

	return nil
}

// FindUniqueName finds the next available unique name by appending -N suffix.
// Used during import with mode:rename to avoid name collisions.
// Returns the original baseName if it doesn't exist, otherwise tries baseName-1, baseName-2, etc.
func FindUniqueName(ctx context.Context, q Querier, workspaceNorm, baseName string) (string, error) {
	// First check if baseName itself is available
	exists, err := CheckNameExists(ctx, q, workspaceNorm, baseName)
	if err != nil {
		return "", err
	}
	if !exists {
		return baseName, nil
	}

	// Try suffixed versions
	for i := 1; i <= 1000; i++ {
		select {
		case <-ctx.Done():
			return "", errors.NewCancelled("find unique name")
		default:
		}
		candidate := baseName + "-" + itoa(i)
		exists, err := CheckNameExists(ctx, q, workspaceNorm, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}

	return "", errors.NewConflict("could not find unique name after 1000 attempts")
}

// itoa converts an integer to a string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// PurgeDeleted permanently deletes soft-deleted capsules.
// Optional filters:
//   - workspace: only purge capsules in this workspace
//   - olderThanDays: only purge capsules deleted more than N days ago
//
// Returns the number of capsules purged.
func PurgeDeleted(ctx context.Context, db *sql.DB, workspace *string, olderThanDays *int) (int, error) {
	var conditions []string
	var args []any

	// Always require deleted_at IS NOT NULL
	conditions = append(conditions, "deleted_at IS NOT NULL")

	if workspace != nil {
		conditions = append(conditions, "workspace_norm = ?")
		args = append(args, capsule.Normalize(*workspace))
	}

	if olderThanDays != nil {
		if *olderThanDays < 0 {
			return 0, errors.NewInvalidRequest("older_than_days cannot be negative")
		}
		cutoff := time.Now().Unix() - int64(*olderThanDays)*24*60*60
		conditions = append(conditions, "deleted_at < ?")
		args = append(args, cutoff)
	}

	query := "DELETE FROM capsules WHERE " + strings.Join(conditions, " AND ")

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.NewInternal(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.NewInternal(err)
	}

	return int(rowsAffected), nil
}

// GetByIDIncludeDeleted retrieves a capsule by ID, optionally including deleted ones.
// This is an alias for GetByID for clarity in import logic.
func GetByIDIncludeDeleted(ctx context.Context, q Querier, id string) (*capsule.Capsule, error) {
	return GetByID(ctx, q, id, true)
}

// BulkSoftDelete sets deleted_at on all active capsules matching the given filters.
// Only targets active capsules (deleted_at IS NULL is hardcoded).
// Filter validation is the caller's responsibility.
func BulkSoftDelete(ctx context.Context, db *sql.DB, filters InventoryFilters) (int, error) {
	now := time.Now().Unix()

	conditions := []string{"deleted_at IS NULL"}
	var args []any

	if filters.Workspace != nil {
		conditions = append(conditions, "workspace_norm = ?")
		args = append(args, *filters.Workspace)
	}
	if filters.Tag != nil {
		conditions = append(conditions, "EXISTS(SELECT 1 FROM json_each(tags_json) WHERE value = ?)")
		args = append(args, *filters.Tag)
	}
	if filters.NamePrefix != nil {
		conditions = append(conditions, "name_norm LIKE ? ESCAPE '\\'")
		args = append(args, escapeLikePattern(*filters.NamePrefix)+"%")
	}
	if filters.RunID != nil {
		conditions = append(conditions, "run_id = ?")
		args = append(args, *filters.RunID)
	}
	if filters.Phase != nil {
		conditions = append(conditions, "phase = ?")
		args = append(args, *filters.Phase)
	}
	if filters.Role != nil {
		conditions = append(conditions, "role = ?")
		args = append(args, *filters.Role)
	}

	query := "UPDATE capsules SET deleted_at = ? WHERE " + strings.Join(conditions, " AND ")
	// Prepend deleted_at value to args
	args = append([]any{now}, args...)

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.NewInternal(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.NewInternal(err)
	}

	return int(rowsAffected), nil
}

// BulkUpdateFields contains the fields to update in a bulk update operation.
type BulkUpdateFields struct {
	Phase *string
	Role  *string
	Tags  *[]string
}

// BulkUpdate updates metadata on all active capsules matching the given filters.
// Only targets active capsules (deleted_at IS NULL is hardcoded).
// Filter validation is the caller's responsibility.
// Empty string values in fields mean "clear the field" (set to NULL).
func BulkUpdate(ctx context.Context, db *sql.DB, filters InventoryFilters, fields BulkUpdateFields) (int, error) {
	now := time.Now().Unix()

	// Build SET clause from non-nil fields
	var setClauses []string
	var setArgs []any

	if fields.Phase != nil {
		if *fields.Phase == "" {
			setClauses = append(setClauses, "phase = NULL")
		} else {
			setClauses = append(setClauses, "phase = ?")
			setArgs = append(setArgs, *fields.Phase)
		}
	}
	if fields.Role != nil {
		if *fields.Role == "" {
			setClauses = append(setClauses, "role = NULL")
		} else {
			setClauses = append(setClauses, "role = ?")
			setArgs = append(setArgs, *fields.Role)
		}
	}
	if fields.Tags != nil {
		if len(*fields.Tags) == 0 {
			setClauses = append(setClauses, "tags_json = NULL")
		} else {
			data, err := json.Marshal(*fields.Tags)
			if err != nil {
				return 0, errors.NewInternal(err)
			}
			setClauses = append(setClauses, "tags_json = ?")
			setArgs = append(setArgs, string(data))
		}
	}

	// Always include updated_at
	setClauses = append(setClauses, "updated_at = ?")
	setArgs = append(setArgs, now)

	// Build WHERE clause from filters
	conditions := []string{"deleted_at IS NULL"}
	var filterArgs []any

	if filters.Workspace != nil {
		conditions = append(conditions, "workspace_norm = ?")
		filterArgs = append(filterArgs, *filters.Workspace)
	}
	if filters.Tag != nil {
		conditions = append(conditions, "EXISTS(SELECT 1 FROM json_each(tags_json) WHERE value = ?)")
		filterArgs = append(filterArgs, *filters.Tag)
	}
	if filters.NamePrefix != nil {
		conditions = append(conditions, "name_norm LIKE ? ESCAPE '\\'")
		filterArgs = append(filterArgs, escapeLikePattern(*filters.NamePrefix)+"%")
	}
	if filters.RunID != nil {
		conditions = append(conditions, "run_id = ?")
		filterArgs = append(filterArgs, *filters.RunID)
	}
	if filters.Phase != nil {
		conditions = append(conditions, "phase = ?")
		filterArgs = append(filterArgs, *filters.Phase)
	}
	if filters.Role != nil {
		conditions = append(conditions, "role = ?")
		filterArgs = append(filterArgs, *filters.Role)
	}

	query := "UPDATE capsules SET " + strings.Join(setClauses, ", ") + " WHERE " + strings.Join(conditions, " AND ")
	args := append(setArgs, filterArgs...)

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.NewInternal(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.NewInternal(err)
	}

	return int(rowsAffected), nil
}
