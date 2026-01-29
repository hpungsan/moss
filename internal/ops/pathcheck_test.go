package ops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/errors"
)

func TestValidatePath_TraversalRejected(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		name string
		path string
	}{
		{"parent traversal", "../backup.jsonl"},
		{"deep traversal", "../../etc/backup.jsonl"},
		{"mid-path traversal", "/tmp/../etc/backup.jsonl"},
		{"hidden in path", "/tmp/safe/../../../etc/shadow.jsonl"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePath(tc.path, PathCheckWrite, cfg)
			if err == nil {
				t.Error("expected error for path traversal, got nil")
			}
			if !errors.Is(err, errors.ErrInvalidRequest) {
				t.Errorf("expected ErrInvalidRequest, got: %v", err)
			}
		})
	}
}

func TestValidatePath_ExtensionRequired(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.AllowUnsafePaths = true // Allow any directory

	tests := []struct {
		name string
		path string
	}{
		{"no extension", "/tmp/backup"},
		{"wrong extension", "/tmp/backup.json"},
		{"txt extension", "/tmp/backup.txt"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePath(tc.path, PathCheckWrite, cfg)
			if err == nil {
				t.Error("expected error for wrong extension, got nil")
			}
			if !errors.Is(err, errors.ErrInvalidRequest) {
				t.Errorf("expected ErrInvalidRequest, got: %v", err)
			}
		})
	}
}

func TestValidatePath_DirectoryRestriction(t *testing.T) {
	cfg := config.DefaultConfig()
	// Default config: only ~/.moss/exports allowed

	// Path outside allowed directories should fail
	err := ValidatePath("/tmp/backup.jsonl", PathCheckWrite, cfg)
	if err == nil {
		t.Error("expected error for path outside allowed directories, got nil")
	}
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("expected ErrInvalidRequest, got: %v", err)
	}
}

func TestValidatePath_AllowUnsafePaths(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.AllowUnsafePaths = true

	// Create a test file for read mode
	testFile := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(testFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Should allow paths outside ~/.moss/exports when AllowUnsafePaths=true
	err := ValidatePath(testFile, PathCheckRead, cfg)
	if err != nil {
		t.Errorf("expected success with AllowUnsafePaths=true, got: %v", err)
	}

	// Write mode should also work
	writePath := filepath.Join(tmpDir, "output.jsonl")
	err = ValidatePath(writePath, PathCheckWrite, cfg)
	if err != nil {
		t.Errorf("expected success for write with AllowUnsafePaths=true, got: %v", err)
	}
}

func TestValidatePath_AllowedPaths(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.AllowedPaths = []string{tmpDir}

	// Create a test file for read mode
	testFile := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(testFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Should allow paths in AllowedPaths
	err := ValidatePath(testFile, PathCheckRead, cfg)
	if err != nil {
		t.Errorf("expected success for path in AllowedPaths, got: %v", err)
	}

	// Path outside AllowedPaths (and not in ~/.moss/exports) should fail
	otherDir := t.TempDir()
	otherFile := filepath.Join(otherDir, "other.jsonl")
	if err := os.WriteFile(otherFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err = ValidatePath(otherFile, PathCheckRead, cfg)
	if err == nil {
		t.Error("expected error for path outside AllowedPaths, got nil")
	}
}

func TestValidatePath_FileNotFound_ReadMode(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.AllowUnsafePaths = true

	nonExistent := filepath.Join(tmpDir, "nonexistent.jsonl")
	err := ValidatePath(nonExistent, PathCheckRead, cfg)
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestValidatePath_SymlinkRejected(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.AllowedPaths = []string{tmpDir}

	// Create a target file in a different directory (outside allowed paths)
	otherDir := t.TempDir()
	targetFile := filepath.Join(otherDir, "secret.jsonl")
	if err := os.WriteFile(targetFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to create target file: %v", err)
	}

	// Create a symlink in the allowed directory pointing outside
	symlink := filepath.Join(tmpDir, "link.jsonl")
	if err := os.Symlink(targetFile, symlink); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	// Reading via symlink should fail because it resolves outside allowed dirs
	err := ValidatePath(symlink, PathCheckRead, cfg)
	if err == nil {
		t.Error("expected error for symlink resolving outside allowed dirs, got nil")
	}
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("expected ErrInvalidRequest, got: %v", err)
	}
}

func TestValidatePath_SymlinkRejected_EvenWithUnsafePaths(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.AllowUnsafePaths = true

	// Create a target file
	targetFile := filepath.Join(tmpDir, "target.jsonl")
	if err := os.WriteFile(targetFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to create target file: %v", err)
	}

	// Create a symlink
	symlink := filepath.Join(tmpDir, "link.jsonl")
	if err := os.Symlink(targetFile, symlink); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	// Even with AllowUnsafePaths=true, symlinks should be rejected.
	// AllowUnsafePaths bypasses directory restrictions, NOT symlink restrictions.
	// O_NOFOLLOW is always used at open time, so validation should match.
	err := ValidatePath(symlink, PathCheckRead, cfg)
	if err == nil {
		t.Error("expected error for symlink even with AllowUnsafePaths=true, got nil")
	}
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("expected ErrInvalidRequest, got: %v", err)
	}
}

func TestValidatePath_NestedPathRejected_Read(t *testing.T) {
	allowedDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.AllowedPaths = []string{allowedDir}

	// Create a subdirectory (nested paths are not allowed)
	subDir := filepath.Join(allowedDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	targetFile := filepath.Join(subDir, "test.jsonl")
	if err := os.WriteFile(targetFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to create target file: %v", err)
	}

	// Nested paths are rejected to prevent TOCTOU attacks on directory components.
	err := ValidatePath(targetFile, PathCheckRead, cfg)
	if err == nil {
		t.Error("expected error for nested path, got nil")
	}
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("expected ErrInvalidRequest, got: %v", err)
	}
}

func TestValidatePath_NestedPathRejected_Write(t *testing.T) {
	allowedDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.AllowedPaths = []string{allowedDir}

	// Create a subdirectory (nested paths are not allowed)
	subDir := filepath.Join(allowedDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	nestedPath := filepath.Join(subDir, "out.jsonl")
	err := ValidatePath(nestedPath, PathCheckWrite, cfg)
	if err == nil {
		t.Error("expected error for nested path, got nil")
	}
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("expected ErrInvalidRequest, got: %v", err)
	}
}

func TestValidatePath_SymlinkFileRejected_Write(t *testing.T) {
	allowedDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.AllowedPaths = []string{allowedDir}

	otherDir := t.TempDir()
	targetFile := filepath.Join(otherDir, "secret.jsonl")
	if err := os.WriteFile(targetFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to create target file: %v", err)
	}

	symlink := filepath.Join(allowedDir, "out.jsonl")
	if err := os.Symlink(targetFile, symlink); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	err := ValidatePath(symlink, PathCheckWrite, cfg)
	if err == nil {
		t.Error("expected error for symlink file write, got nil")
	}
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("expected ErrInvalidRequest, got: %v", err)
	}
}

func TestContainsTraversal(t *testing.T) {
	tests := []struct {
		path     string
		contains bool
	}{
		{"/home/user/file.txt", false},
		{"../file.txt", true},
		{"/home/../etc/passwd", true},
		{"./file.txt", false},
		{"/home/user/.hidden/file.txt", false},
		{"file..name.txt", false}, // .. not as path component
		{"/tmp/a/b/../c.jsonl", true},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			result := containsTraversal(tc.path)
			if result != tc.contains {
				t.Errorf("containsTraversal(%q) = %v, want %v", tc.path, result, tc.contains)
			}
		})
	}
}

func TestSanitizeForFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple name", "myproject", "myproject"},
		{"with spaces", "my project", "my project"},
		{"forward slash", "path/to/file", "path-to-file"},
		{"backslash", "path\\to\\file", "path-to-file"},
		{"double dots", "foo..bar", "foo-bar"},
		{"traversal attempt", "../../../etc/passwd", "etc-passwd"},
		{"absolute path", "/tmp/evil", "tmp-evil"},
		{"mixed attack", "../foo/bar\\..\\baz", "foo-bar-baz"},
		{"null bytes", "foo\x00bar", "foobar"},
		{"control chars", "foo\x01\x02bar", "foobar"},
		{"empty after sanitize", "../../..", "unnamed"},
		{"only slashes", "///", "unnamed"},
		{"unicode preserved", "capsule-\u4e2d\u6587", "capsule-\u4e2d\u6587"},
		{"multiple dashes collapse", "a---b", "a-b"},
		{"leading dashes trimmed", "---foo", "foo"},
		{"trailing dashes trimmed", "foo---", "foo"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeForFilename(tc.input)
			if result != tc.expected {
				t.Errorf("SanitizeForFilename(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}
