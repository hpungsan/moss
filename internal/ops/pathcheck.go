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
// 3. Symlink safety (realpath must match input for writes, or be in allowed dirs)
// 4. Directory restrictions (must be in ~/.moss/exports or allowed_paths, unless allow_unsafe_paths)
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

	// If unsafe paths allowed, skip directory and symlink checks
	if cfg != nil && cfg.AllowUnsafePaths {
		// For read mode, still verify file exists (prevents confusing internal errors).
		if mode == PathCheckRead {
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				return errors.NewFileNotFound(path)
			}
		}
		return nil
	}

	// Get allowed directories
	allowedDirsAbs, allowedDirsResolved, err := getAllowedDirs(cfg)
	if err != nil {
		return err
	}

	// Check basic directory restriction first (based on absolute cleaned path).
	if !isInAllowedDirs(absPath, allowedDirsAbs) {
		return errors.NewInvalidRequest(
			fmt.Sprintf("path outside allowed directories; allowed: %v (set allow_unsafe_paths:true in config to override)",
				allowedDirsAbs))
	}

	if mode == PathCheckRead {
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return errors.NewFileNotFound(path)
		}
	}

	// Resolve symlinks (including symlinked directories) and verify the resolved path
	// stays within allowed directories. For write mode we resolve as much as possible
	// (best-effort) to catch existing symlink components even if the file doesn't exist yet.
	resolvedPath, err := resolvePathBestEffort(absPath)
	if err != nil {
		return err
	}
	if !isInAllowedDirs(resolvedPath, allowedDirsResolved) {
		return errors.NewInvalidRequest(
			fmt.Sprintf("path resolves outside allowed directories (resolved: %s; allowed: %v)", resolvedPath, allowedDirsAbs))
	}

	// For write mode, reject if the destination already exists as a symlink.
	// (Prevents clobbering arbitrary targets via symlink file.)
	if mode == PathCheckWrite {
		if info, err := os.Lstat(absPath); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return errors.NewInvalidRequest("path must not be a symlink")
			}
		}
	}

	return nil
}

// getAllowedDirs returns absolute+clean allowed dirs and their best-effort symlink-resolved equivalents.
func getAllowedDirs(cfg *config.Config) (allowedAbs []string, allowedResolved []string, _ error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, errors.NewInternal(fmt.Errorf("failed to get home directory: %w", err))
	}

	// Default: ~/.moss/exports
	defaultDir := filepath.Join(homeDir, ".moss", "exports")
	dirs := []string{defaultDir}

	// Add configured allowed paths (only absolute paths)
	if cfg != nil {
		for _, p := range cfg.AllowedPaths {
			if filepath.IsAbs(p) {
				// Clean and add
				dirs = append(dirs, filepath.Clean(p))
			}
		}
	}

	absDirs := make([]string, 0, len(dirs))
	resolvedDirs := make([]string, 0, len(dirs))
	for _, d := range dirs {
		abs, err := filepath.Abs(filepath.Clean(d))
		if err != nil {
			return nil, nil, errors.NewInvalidRequest(fmt.Sprintf("invalid allowed path: %v", err))
		}
		absDirs = append(absDirs, abs)

		resolved, err := resolvePathBestEffort(abs)
		if err != nil {
			return nil, nil, err
		}
		resolvedDirs = append(resolvedDirs, resolved)
	}

	return absDirs, resolvedDirs, nil
}

// isInAllowedDirs checks if a path is within one of the allowed directories.
func isInAllowedDirs(absPath string, allowedDirs []string) bool {
	for _, dir := range allowedDirs {
		if isSubPath(absPath, dir) {
			return true
		}
	}
	return false
}

// isSubPath checks if path is equal to or under baseDir.
func isSubPath(path, baseDir string) bool {
	// Ensure both are clean absolute paths
	path = filepath.Clean(path)
	baseDir = filepath.Clean(baseDir)

	// Check if path starts with baseDir
	if path == baseDir {
		return true
	}

	// Add separator to avoid /foo/bar matching /foo/barbaz
	if strings.HasPrefix(path, baseDir+string(filepath.Separator)) {
		return true
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

// resolvePathBestEffort resolves symlinks in an absolute path as much as possible.
// If the full path doesn't exist (common for writes), it resolves the deepest existing
// ancestor and reattaches the remaining suffix.
func resolvePathBestEffort(absPath string) (string, error) {
	absPath = filepath.Clean(absPath)

	existing := absPath
	for {
		if _, err := os.Lstat(existing); err == nil {
			break
		} else if os.IsNotExist(err) {
			parent := filepath.Dir(existing)
			if parent == existing {
				break
			}
			existing = parent
			continue
		} else {
			return "", errors.NewInternal(fmt.Errorf("failed to stat path: %w", err))
		}
	}

	realExisting, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return "", errors.NewInvalidRequest(fmt.Sprintf("cannot resolve symlink in path: %v", err))
	}

	rel, err := filepath.Rel(existing, absPath)
	if err != nil {
		return "", errors.NewInternal(fmt.Errorf("failed to compute relative path: %w", err))
	}
	if rel == "." {
		return filepath.Clean(realExisting), nil
	}

	return filepath.Clean(filepath.Join(realExisting, rel)), nil
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
