package ops

import (
	"bufio"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

// ImportMode controls collision behavior during import.
type ImportMode string

const (
	ImportModeError   ImportMode = "error"   // fail on collision (atomic)
	ImportModeReplace ImportMode = "replace" // overwrite on collision
	ImportModeRename  ImportMode = "rename"  // auto-suffix name on collision
)

// ImportInput contains parameters for the Import operation.
type ImportInput struct {
	Path string     // required
	Mode ImportMode // default: error
}

// ImportOutput contains the result of the Import operation.
type ImportOutput struct {
	Imported int           `json:"imported"`
	Skipped  int           `json:"skipped"`
	Errors   []ImportError `json:"errors"`
}

// ImportError represents an error that occurred during import.
type ImportError struct {
	Line    int    `json:"line"`
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Import imports capsules from a JSONL export file.
func Import(database *sql.DB, input ImportInput) (*ImportOutput, error) {
	// Validate input
	if input.Path == "" {
		return nil, errors.NewInvalidRequest("path is required")
	}
	if input.Mode == "" {
		input.Mode = ImportModeError
	}
	if input.Mode != ImportModeError && input.Mode != ImportModeReplace && input.Mode != ImportModeRename {
		return nil, errors.NewInvalidRequest("mode must be one of: error, replace, rename")
	}

	// Check file exists
	if _, err := os.Stat(input.Path); os.IsNotExist(err) {
		return nil, errors.NewNotFound(input.Path)
	}

	// Open file
	file, err := os.Open(input.Path)
	if err != nil {
		return nil, errors.NewInternal(fmt.Errorf("failed to open import file: %w", err))
	}
	defer file.Close()

	// Parse all records first
	records, parseErrors := parseExportFile(file)

	// For mode:error, fail on any parse errors
	if input.Mode == ImportModeError && len(parseErrors) > 0 {
		return &ImportOutput{
			Imported: 0,
			Skipped:  0,
			Errors:   parseErrors,
		}, nil
	}

	// Process records based on mode
	switch input.Mode {
	case ImportModeError:
		return importModeError(database, records)
	case ImportModeReplace:
		return importModeReplace(database, records, parseErrors)
	case ImportModeRename:
		return importModeRename(database, records, parseErrors)
	default:
		return nil, errors.NewInvalidRequest("invalid mode")
	}
}

// parseExportFile parses a JSONL export file into records.
func parseExportFile(file *os.File) ([]capsule.ExportRecord, []ImportError) {
	var records []capsule.ExportRecord
	var parseErrors []ImportError

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		var record capsule.ExportRecord
		if err := json.Unmarshal(line, &record); err != nil {
			parseErrors = append(parseErrors, ImportError{
				Line:    lineNum,
				Code:    "PARSE_ERROR",
				Message: fmt.Sprintf("invalid JSON: %v", err),
			})
			continue
		}

		// Skip header line
		if record.MossExport {
			continue
		}

		// Skip lines with no ID (invalid)
		if record.ID == "" {
			parseErrors = append(parseErrors, ImportError{
				Line:    lineNum,
				Code:    "INVALID_RECORD",
				Message: "missing id field",
			})
			continue
		}

		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		parseErrors = append(parseErrors, ImportError{
			Line:    lineNum,
			Code:    "READ_ERROR",
			Message: fmt.Sprintf("failed to read file: %v", err),
		})
	}

	return records, parseErrors
}

// importModeError imports all records atomically, rolling back on any collision.
func importModeError(database *sql.DB, records []capsule.ExportRecord) (*ImportOutput, error) {
	tx, err := database.Begin()
	if err != nil {
		return nil, errors.NewInternal(err)
	}
	defer tx.Rollback() //nolint:errcheck

	imported := 0
	var importErrors []ImportError

	for _, record := range records {
		// Check for ID collision
		existing, err := db.GetByID(database, record.ID, true)
		if err != nil && !errors.Is(err, errors.ErrNotFound) {
			return nil, err
		}
		if existing != nil {
			importErrors = append(importErrors, ImportError{
				ID:      record.ID,
				Code:    "ID_COLLISION",
				Message: fmt.Sprintf("capsule with id %q already exists", record.ID),
			})
			// Abort on first error for mode:error
			return &ImportOutput{
				Imported: 0,
				Skipped:  0,
				Errors:   importErrors,
			}, nil
		}

		// Check for name collision (if named)
		c := record.ToCapsule()
		if c.NameNorm != nil {
			exists, err := db.CheckNameExists(database, c.WorkspaceNorm, *c.NameNorm)
			if err != nil {
				return nil, err
			}
			if exists {
				name := ""
				if record.NameRaw != nil {
					name = *record.NameRaw
				}
				importErrors = append(importErrors, ImportError{
					ID:      record.ID,
					Name:    name,
					Code:    "NAME_COLLISION",
					Message: fmt.Sprintf("capsule with name %q already exists in workspace %q", name, record.WorkspaceRaw),
				})
				// Abort on first error for mode:error
				return &ImportOutput{
					Imported: 0,
					Skipped:  0,
					Errors:   importErrors,
				}, nil
			}
		}

		// Insert capsule
		if err := insertWithTx(tx, c); err != nil {
			return nil, err
		}
		imported++
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.NewInternal(err)
	}

	return &ImportOutput{
		Imported: imported,
		Skipped:  0,
		Errors:   importErrors,
	}, nil
}

// importModeReplace imports records, updating existing on collision.
func importModeReplace(database *sql.DB, records []capsule.ExportRecord, parseErrors []ImportError) (*ImportOutput, error) {
	imported := 0
	skipped := 0
	var importErrors []ImportError

	// Include parse errors
	importErrors = append(importErrors, parseErrors...)
	skipped += len(parseErrors)

	for _, record := range records {
		c := record.ToCapsule()

		// Check for ID collision
		existingByID, err := db.GetByID(database, record.ID, true)
		if err != nil && !errors.Is(err, errors.ErrNotFound) {
			return nil, err
		}

		// Check for name collision (if named)
		var existingByName *capsule.Capsule
		if c.NameNorm != nil {
			existingByName, err = db.GetByName(database, c.WorkspaceNorm, *c.NameNorm, true)
			if err != nil && !errors.Is(err, errors.ErrNotFound) {
				return nil, err
			}
		}

		// Handle ambiguous case: ID matches row A AND name matches different row B
		if existingByID != nil && existingByName != nil && existingByID.ID != existingByName.ID {
			name := ""
			if record.NameRaw != nil {
				name = *record.NameRaw
			}
			importErrors = append(importErrors, ImportError{
				ID:      record.ID,
				Name:    name,
				Code:    "AMBIGUOUS_COLLISION",
				Message: fmt.Sprintf("id %q matches existing capsule but name %q matches different capsule", record.ID, name),
			})
			skipped++
			continue
		}

		// Decide action based on collisions
		if existingByID != nil {
			// ID collision: update by ID
			if err := db.UpdateFull(database, c); err != nil {
				return nil, err
			}
			imported++
		} else if existingByName != nil {
			// Name collision (different ID): update by existing ID, keep new data
			c.ID = existingByName.ID
			if err := db.UpdateFull(database, c); err != nil {
				return nil, err
			}
			imported++
		} else {
			// No collision: insert new
			if err := db.Insert(database, c); err != nil {
				return nil, err
			}
			imported++
		}
	}

	return &ImportOutput{
		Imported: imported,
		Skipped:  skipped,
		Errors:   importErrors,
	}, nil
}

// importModeRename imports records, auto-renaming on collision.
func importModeRename(database *sql.DB, records []capsule.ExportRecord, parseErrors []ImportError) (*ImportOutput, error) {
	imported := 0
	skipped := 0
	var importErrors []ImportError

	// Include parse errors
	importErrors = append(importErrors, parseErrors...)
	skipped += len(parseErrors)

	for _, record := range records {
		c := record.ToCapsule()

		// Check for ID collision
		existingByID, err := db.GetByID(database, record.ID, true)
		if err != nil && !errors.Is(err, errors.ErrNotFound) {
			return nil, err
		}

		// If ID collision, generate new ULID
		if existingByID != nil {
			c.ID = generateNewULID()
		}

		// Check for name collision (if named)
		if c.NameNorm != nil {
			exists, err := db.CheckNameExists(database, c.WorkspaceNorm, *c.NameNorm)
			if err != nil {
				return nil, err
			}
			if exists {
				// Find unique name
				baseName := *c.NameNorm
				newName, err := db.FindUniqueName(database, c.WorkspaceNorm, baseName)
				if err != nil {
					name := ""
					if record.NameRaw != nil {
						name = *record.NameRaw
					}
					importErrors = append(importErrors, ImportError{
						ID:      record.ID,
						Name:    name,
						Code:    "RENAME_FAILED",
						Message: fmt.Sprintf("failed to find unique name: %v", err),
					})
					skipped++
					continue
				}
				// Apply new name to both raw and norm
				c.NameRaw = &newName
				c.NameNorm = &newName
			}
		}

		// Insert capsule
		if err := db.Insert(database, c); err != nil {
			name := ""
			if c.NameRaw != nil {
				name = *c.NameRaw
			}
			importErrors = append(importErrors, ImportError{
				ID:      c.ID,
				Name:    name,
				Code:    "INSERT_FAILED",
				Message: fmt.Sprintf("failed to insert: %v", err),
			})
			skipped++
			continue
		}
		imported++
	}

	return &ImportOutput{
		Imported: imported,
		Skipped:  skipped,
		Errors:   importErrors,
	}, nil
}

// insertWithTx inserts a capsule within a transaction.
func insertWithTx(tx *sql.Tx, c *capsule.Capsule) error {
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
	var deletedAt sql.NullInt64
	if c.DeletedAt != nil {
		deletedAt = sql.NullInt64{Int64: *c.DeletedAt, Valid: true}
	}

	query := `
		INSERT INTO capsules (
			id, workspace_raw, workspace_norm, name_raw, name_norm,
			title, capsule_text, capsule_chars, tokens_estimate,
			tags_json, source, created_at, updated_at, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(query,
		c.ID, c.WorkspaceRaw, c.WorkspaceNorm, nameRaw, nameNorm,
		title, c.CapsuleText, c.CapsuleChars, c.TokensEstimate,
		tagsJSON, source, c.CreatedAt, c.UpdatedAt, deletedAt,
	)
	if err != nil {
		return errors.NewInternal(err)
	}

	return nil
}

// toNullString converts *string to sql.NullString.
func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// generateNewULID generates a new ULID.
func generateNewULID() string {
	entropy := ulid.Monotonic(rand.Reader, 0)
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}
