package db

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/errors"
)

// Insert stores a new capsule in the database.
func Insert(db *sql.DB, c *capsule.Capsule) error {
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

	query := `
		INSERT INTO capsules (
			id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_text, capsule_chars, tokens_estimate,
			tags_json, source, created_at, updated_at, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
	`

	_, err := db.Exec(query,
		c.ID, c.WorkspaceRaw, c.WorkspaceNorm, nameRaw, nameNorm,
		title, c.CapsuleText, c.CapsuleChars, c.TokensEstimate,
		tagsJSON, source, c.CreatedAt, c.UpdatedAt,
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

// GetByID retrieves a capsule by its ULID.
// If includeDeleted is false, soft-deleted capsules are excluded.
func GetByID(db *sql.DB, id string, includeDeleted bool) (*capsule.Capsule, error) {
	query := `
		SELECT id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_text, capsule_chars, tokens_estimate,
			tags_json, source, created_at, updated_at, deleted_at
		FROM capsules
		WHERE id = ?
	`
	if !includeDeleted {
		query += " AND deleted_at IS NULL"
	}

	row := db.QueryRow(query, id)
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
func GetByName(db *sql.DB, workspaceNorm, nameNorm string, includeDeleted bool) (*capsule.Capsule, error) {
	query := `
		SELECT id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_text, capsule_chars, tokens_estimate,
			tags_json, source, created_at, updated_at, deleted_at
		FROM capsules
		WHERE workspace_norm = ? AND name_norm = ?
	`
	if !includeDeleted {
		query += " AND deleted_at IS NULL"
	} else {
		// If both active and soft-deleted capsules exist for the same name, prefer the active one.
		// If no active capsule exists, return the most recently updated deleted capsule.
		query += " ORDER BY (deleted_at IS NULL) DESC, updated_at DESC LIMIT 1"
	}

	row := db.QueryRow(query, workspaceNorm, nameNorm)
	c, err := scanCapsule(row)
	if err == sql.ErrNoRows {
		return nil, errors.NewNotFound(nameNorm)
	}
	if err != nil {
		return nil, errors.NewInternal(err)
	}

	return c, nil
}

// CheckNameExists checks if an active capsule with the given name exists.
func CheckNameExists(db *sql.DB, workspaceNorm, nameNorm string) (bool, error) {
	query := `
		SELECT 1 FROM capsules
		WHERE workspace_norm = ? AND name_norm = ? AND deleted_at IS NULL
		LIMIT 1
	`

	var exists int
	err := db.QueryRow(query, workspaceNorm, nameNorm).Scan(&exists)
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
func UpdateByID(db *sql.DB, c *capsule.Capsule) error {
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

	now := time.Now().Unix()

	query := `
		UPDATE capsules
		SET capsule_text = ?, title = ?, tags_json = ?, source = ?,
			capsule_chars = ?, tokens_estimate = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := db.Exec(query,
		c.CapsuleText, title, tagsJSON, source,
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
func SoftDelete(db *sql.DB, id string) error {
	now := time.Now().Unix()

	query := `
		UPDATE capsules
		SET deleted_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := db.Exec(query, now, id)
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
		deletedAt sql.NullInt64
	)

	err := row.Scan(
		&c.ID, &c.WorkspaceRaw, &c.WorkspaceNorm, &nameRaw, &nameNorm,
		&title, &c.CapsuleText, &c.CapsuleChars, &c.TokensEstimate,
		&tagsJSON, &source, &c.CreatedAt, &c.UpdatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert nullable fields
	c.NameRaw = fromNullString(nameRaw)
	c.NameNorm = fromNullString(nameNorm)
	c.Title = fromNullString(title)
	c.Source = fromNullString(source)

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
// title, capsule_chars, tokens_estimate, tags_json, source, created_at, updated_at, deleted_at
func scanCapsuleSummary(scanner interface{ Scan(...any) error }) (*capsule.CapsuleSummary, error) {
	var (
		s         capsule.CapsuleSummary
		nameRaw   sql.NullString
		nameNorm  sql.NullString
		title     sql.NullString
		tagsJSON  sql.NullString
		source    sql.NullString
		deletedAt sql.NullInt64
	)

	err := scanner.Scan(
		&s.ID, &s.Workspace, &s.WorkspaceNorm, &nameRaw, &nameNorm,
		&title, &s.CapsuleChars, &s.TokensEstimate,
		&tagsJSON, &source, &s.CreatedAt, &s.UpdatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert nullable fields
	s.Name = fromNullString(nameRaw)
	s.NameNorm = fromNullString(nameNorm)
	s.Title = fromNullString(title)
	s.Source = fromNullString(source)

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

// ListByWorkspace retrieves capsule summaries for a workspace with pagination.
// Returns summaries (no capsule_text) + total count.
// Ordered by updated_at DESC, id DESC (stable pagination).
func ListByWorkspace(db *sql.DB, workspaceNorm string, limit, offset int, includeDeleted bool) ([]capsule.CapsuleSummary, int, error) {
	// Build count query
	countQuery := `SELECT COUNT(*) FROM capsules WHERE workspace_norm = ?`
	if !includeDeleted {
		countQuery += " AND deleted_at IS NULL"
	}

	var total int
	if err := db.QueryRow(countQuery, workspaceNorm).Scan(&total); err != nil {
		return nil, 0, errors.NewInternal(err)
	}

	// Build list query
	listQuery := `
		SELECT id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_chars, tokens_estimate, tags_json, source,
			created_at, updated_at, deleted_at
		FROM capsules
		WHERE workspace_norm = ?
	`
	if !includeDeleted {
		listQuery += " AND deleted_at IS NULL"
	}
	listQuery += " ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?"

	rows, err := db.Query(listQuery, workspaceNorm, limit, offset)
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
}

// ListAll retrieves capsule summaries across all workspaces with optional filters.
// Returns summaries (no capsule_text) + total count.
// Ordered by updated_at DESC, id DESC (stable pagination).
func ListAll(db *sql.DB, filters InventoryFilters, limit, offset int, includeDeleted bool) ([]capsule.CapsuleSummary, int, error) {
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

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Build count query
	countQuery := "SELECT COUNT(*) FROM capsules" + whereClause
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.NewInternal(err)
	}

	// Build list query
	listQuery := `
		SELECT id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_chars, tokens_estimate, tags_json, source,
			created_at, updated_at, deleted_at
		FROM capsules` + whereClause + " ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?"

	listArgs := append(args, limit, offset)
	rows, err := db.Query(listQuery, listArgs...)
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

// GetLatestSummary retrieves the most recent capsule summary in a workspace.
// Returns summary (no capsule_text).
// Returns nil, nil if workspace is empty (not an error).
func GetLatestSummary(db *sql.DB, workspaceNorm string, includeDeleted bool) (*capsule.CapsuleSummary, error) {
	query := `
		SELECT id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_chars, tokens_estimate, tags_json, source,
			created_at, updated_at, deleted_at
		FROM capsules
		WHERE workspace_norm = ?
	`
	if !includeDeleted {
		query += " AND deleted_at IS NULL"
	}
	query += " ORDER BY updated_at DESC, id DESC LIMIT 1"

	row := db.QueryRow(query, workspaceNorm)
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
func GetLatestFull(db *sql.DB, workspaceNorm string, includeDeleted bool) (*capsule.Capsule, error) {
	query := `
		SELECT id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_text, capsule_chars, tokens_estimate,
			tags_json, source, created_at, updated_at, deleted_at
		FROM capsules
		WHERE workspace_norm = ?
	`
	if !includeDeleted {
		query += " AND deleted_at IS NULL"
	}
	query += " ORDER BY updated_at DESC, id DESC LIMIT 1"

	row := db.QueryRow(query, workspaceNorm)
	c, err := scanCapsule(row)
	if err == sql.ErrNoRows {
		return nil, nil // Empty workspace is not an error
	}
	if err != nil {
		return nil, errors.NewInternal(err)
	}

	return c, nil
}
