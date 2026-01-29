package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/errors"
)

// PathCheckMode indicates whether the path check is for reading or writing.
type PathCheckMode int

const (
	PathCheckRead  PathCheckMode = iota // for import (read file)
	PathCheckWrite                      // for export (write file)
)

// ValidatePath performs comprehensive path validation for import/export operations.
// It checks:
// 1. Path traversal (.. sequences)
// 2. Extension (.jsonl required)
// 3. Directory restrictions (file must be DIRECTLY in ~/.moss/exports or allowed_paths - no subdirectories)
// 4. Symlink safety (parent dir must not be a symlink, file must not be a symlink for writes)
//
// The "no subdirectories" rule eliminates TOCTOU race conditions where an attacker could
// swap an intermediate directory component with a symlink between validation and open.
// Combined with O_NOFOLLOW on the final component, this provides complete symlink protection.
func ValidatePath(path string, mode PathCheckMode, cfg *config.Config) error {
	if path == "" {
		return errors.NewInvalidRequest("path is required")
	}

	// Reject paths containing ".." (traversal attempt)
	if containsTraversal(path) {
		return errors.NewInvalidRequest("path must not contain directory traversal (..)")
	}

	// Require .jsonl extension
	cleaned := filepath.Clean(path)
	if filepath.Ext(cleaned) != ".jsonl" {
		return errors.NewInvalidRequest("path must have .jsonl extension")
	}

	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return errors.NewInvalidRequest(fmt.Sprintf("invalid path: %v", err))
	}

	// If unsafe paths allowed, skip directory checks (but NOT symlink checks).
	// Symlink restrictions always apply because O_NOFOLLOW is used at open time.
	if cfg != nil && cfg.AllowUnsafePaths {
		// For read mode, still verify file exists (prevents confusing internal errors).
		if mode == PathCheckRead {
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				return errors.NewFileNotFound(path)
			}
		}
		// Reject symlink files even in unsafe mode (O_NOFOLLOW would reject at runtime anyway).
		if info, err := os.Lstat(absPath); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return errors.NewInvalidRequest("path must not be a symlink")
			}
		}
		return nil
	}

	// Get allowed directories (resolved to catch symlinked allowed_paths entries)
	allowedDirs, err := getAllowedDirs(cfg)
	if err != nil {
		return err
	}

	// File must be DIRECTLY in an allowed directory (no subdirectories allowed).
	// This eliminates TOCTOU races on intermediate directory components.
	parentDir := filepath.Dir(absPath)
	if !isDirectlyInAllowedDir(parentDir, allowedDirs) {
		return errors.NewInvalidRequest(
			fmt.Sprintf("file must be directly in an allowed directory (no subdirectories); allowed: %v",
				allowedDirs))
	}

	// Verify the parent directory is not a symlink (defense-in-depth).
	if info, err := os.Lstat(parentDir); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.NewInvalidRequest("parent directory must not be a symlink")
		}
	}

	if mode == PathCheckRead {
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return errors.NewFileNotFound(path)
		}
	}

	// Reject symlink files for both read and write modes.
	// O_NOFOLLOW at open time would catch this too, but rejecting early gives a clearer error.
	// Note: AllowUnsafePaths bypasses directory restrictions but NOT symlink restrictions.
	if info, err := os.Lstat(absPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.NewInvalidRequest("path must not be a symlink")
		}
	}

	return nil
}

// getAllowedDirs returns the list of allowed directories (absolute, cleaned).
// If an allowed directory exists, it is resolved to catch symlinked allowed_paths entries.
func getAllowedDirs(cfg *config.Config) ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.NewInternal(fmt.Errorf("failed to get home directory: %w", err))
	}

	// Default: ~/.moss/exports
	defaultDir := filepath.Join(homeDir, ".moss", "exports")
	dirs := []string{defaultDir}

	// Add configured allowed paths (only absolute paths)
	if cfg != nil {
		for _, p := range cfg.AllowedPaths {
			if filepath.IsAbs(p) {
				dirs = append(dirs, filepath.Clean(p))
			}
		}
	}

	result := make([]string, 0, len(dirs))
	for _, d := range dirs {
		abs, err := filepath.Abs(filepath.Clean(d))
		if err != nil {
			return nil, errors.NewInvalidRequest(fmt.Sprintf("invalid allowed path: %v", err))
		}

		// If the directory exists, resolve symlinks to get the real path.
		// This ensures that if allowed_paths contains a symlink, we match against the real target.
		if info, err := os.Lstat(abs); err == nil && info.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(abs)
			if err != nil {
				return nil, errors.NewInvalidRequest(fmt.Sprintf("cannot resolve symlink in allowed path: %v", err))
			}
			abs = resolved
		}
		result = append(result, abs)
	}

	return result, nil
}

// isDirectlyInAllowedDir checks if parentDir exactly matches one of the allowed directories.
// This is stricter than "is under" - the file must be directly in the allowed dir, not in a subdirectory.
func isDirectlyInAllowedDir(parentDir string, allowedDirs []string) bool {
	parentDir = filepath.Clean(parentDir)
	for _, dir := range allowedDirs {
		if parentDir == filepath.Clean(dir) {
			return true
		}
	}
	return false
}

// DefaultExportsDir returns the default exports directory (~/.moss/exports).
func DefaultExportsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.NewInternal(fmt.Errorf("failed to get home directory: %w", err))
	}
	return filepath.Join(homeDir, ".moss", "exports"), nil
}

// containsTraversal checks if path contains ".." directory traversal.
func containsTraversal(path string) bool {
	// Check each path component
	for _, part := range strings.Split(path, string(filepath.Separator)) {
		if part == ".." {
			return true
		}
	}
	// Also check for forward slashes on all platforms (e.g., user input)
	if filepath.Separator != '/' {
		for _, part := range strings.Split(path, "/") {
			if part == ".." {
				return true
			}
		}
	}
	return false
}

// SanitizeForFilename sanitizes a string for safe use in a filename.
// Removes/replaces characters that could be used for path traversal or injection.
func SanitizeForFilename(s string) string {
	// Replace path separators with dashes
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")

	// Replace ".." sequences (could be embedded)
	s = strings.ReplaceAll(s, "..", "-")

	// Remove null bytes and other control characters
	var result strings.Builder
	for _, r := range s {
		if r >= 32 && r != 127 { // printable ASCII and unicode
			result.WriteRune(r)
		}
	}
	s = result.String()

	// Collapse multiple dashes
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	// Trim leading/trailing dashes
	s = strings.Trim(s, "-")

	// If empty after sanitization, use a safe default
	if s == "" {
		s = "unnamed"
	}

	return s
}
