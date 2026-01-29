package ops

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

// ExportInput contains parameters for the Export operation.
type ExportInput struct {
	Path           string  // optional, default: ~/.moss/exports/<workspace>-<timestamp>.jsonl
	Workspace      *string // optional filter by workspace
	IncludeDeleted bool
}

// ExportOutput contains the result of the Export operation.
type ExportOutput struct {
	Path       string `json:"path"`
	Count      int    `json:"count"`
	ExportedAt int64  `json:"exported_at"`
}

// ExportHeader represents the header line in a JSONL export file.
type ExportHeader struct {
	MossExport    bool   `json:"_moss_export"`
	SchemaVersion string `json:"schema_version"`
	ExportedAt    int64  `json:"exported_at"`
}

// Export exports capsules to a JSONL file.
func Export(ctx context.Context, database *sql.DB, cfg *config.Config, input ExportInput) (*ExportOutput, error) {
	now := time.Now()
	exportedAt := now.Unix()

	// Determine export path
	exportPath := input.Path
	if exportPath == "" {
		var err error
		exportPath, err = defaultExportPath(input.Workspace, now)
		if err != nil {
			return nil, err
		}
	}

	// Validate ALL paths (both user-provided and default) for security
	// This catches workspace injection attacks in default paths
	if err := ValidatePath(exportPath, PathCheckWrite, cfg); err != nil {
		return nil, err
	}

	// Ensure parent directory exists
	dir := filepath.Dir(exportPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, errors.NewInternal(fmt.Errorf("failed to create export directory: %w", err))
	}

	// Create export file with O_NOFOLLOW to prevent TOCTOU symlink attacks
	file, err := openFileNoFollow(exportPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil, errors.NewInternal(fmt.Errorf("failed to create export file: %w", err))
	}

	// Clean up partial file on failure
	success := false
	defer func() {
		file.Close()
		if !success {
			os.Remove(exportPath)
		}
	}()

	// Write header line
	header := ExportHeader{
		MossExport:    true,
		SchemaVersion: "1.0",
		ExportedAt:    exportedAt,
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return nil, errors.NewInternal(err)
	}
	if _, err := file.Write(headerJSON); err != nil {
		return nil, errors.NewInternal(err)
	}
	if _, err := file.Write([]byte("\n")); err != nil {
		return nil, errors.NewInternal(err)
	}

	// Stream capsules and write to file
	rows, err := db.StreamForExport(ctx, database, input.Workspace, input.IncludeDeleted)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, errors.NewCancelled("export")
		default:
		}

		c, err := db.ScanCapsuleFromRows(rows)
		if err != nil {
			return nil, errors.NewInternal(err)
		}

		record := capsule.CapsuleToExportRecord(c)
		recordJSON, err := json.Marshal(record)
		if err != nil {
			return nil, errors.NewInternal(err)
		}

		if _, err := file.Write(recordJSON); err != nil {
			return nil, errors.NewInternal(err)
		}
		if _, err := file.Write([]byte("\n")); err != nil {
			return nil, errors.NewInternal(err)
		}

		count++
	}

	if err := rows.Err(); err != nil {
		return nil, errors.NewInternal(err)
	}

	// Ensure file is written
	if err := file.Sync(); err != nil {
		return nil, errors.NewInternal(err)
	}

	success = true
	return &ExportOutput{
		Path:       exportPath,
		Count:      count,
		ExportedAt: exportedAt,
	}, nil
}

// defaultExportPath generates the default export path.
// Format: ~/.moss/exports/<workspace>-<timestamp>.jsonl or all-<timestamp>.jsonl
func defaultExportPath(workspace *string, now time.Time) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.NewInternal(fmt.Errorf("failed to get home directory: %w", err))
	}

	timestamp := now.Format("2006-01-02T150405")
	name := "all"
	if workspace != nil && *workspace != "" {
		// Normalize first (lowercase, collapse whitespace), then sanitize for filename
		// to prevent path traversal/injection via malicious workspace names
		name = SanitizeForFilename(capsule.Normalize(*workspace))
	}

	filename := fmt.Sprintf("%s-%s.jsonl", name, timestamp)
	return filepath.Join(homeDir, ".moss", "exports", filename), nil
}
