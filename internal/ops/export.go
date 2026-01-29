package ops

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

	// Write to temp file first, then atomic rename to preserve existing file on failure
	randBytes := make([]byte, 8)
	if _, err := rand.Read(randBytes); err != nil {
		return nil, errors.NewInternal(fmt.Errorf("failed to generate temp file name: %w", err))
	}
	tempPath := exportPath + "." + hex.EncodeToString(randBytes) + ".tmp"
	file, err := openFileNoFollow(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil, errors.NewInternal(fmt.Errorf("failed to create export file: %w", err))
	}

	// Clean up temp file on failure (original file is preserved)
	success := false
	defer func() {
		if file != nil {
			file.Close()
		}
		if !success {
			os.Remove(tempPath)
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

	// Close before atomic replace (required on Windows; fine elsewhere).
	if err := file.Close(); err != nil {
		return nil, errors.NewInternal(fmt.Errorf("failed to close export file: %w", err))
	}
	file = nil

	// Check if destination is a symlink (os.Rename would follow it)
	if info, err := os.Lstat(exportPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.NewInternal(fmt.Errorf("export path is a symlink"))
	}

	// Finalize export by renaming temp file into place.
	//
	// Note: On Windows, os.Rename fails if the destination exists. We intentionally
	// fail safely (preserving the existing file) instead of doing a non-atomic
	// delete+rename that could lose the original if rename fails.
	if err := os.Rename(tempPath, exportPath); err != nil {
		if runtime.GOOS == "windows" {
			if _, statErr := os.Stat(exportPath); statErr == nil {
				return nil, errors.NewInvalidRequest("export destination already exists; overwriting is not supported on Windows yet (choose a new path or delete the existing file)")
			}
		}
		return nil, errors.NewInternal(fmt.Errorf("failed to finalize export: %w", err))
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
