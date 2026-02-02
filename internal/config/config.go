package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Config holds application configuration.
type Config struct {
	// CapsuleMaxChars is the maximum character count for capsule text
	CapsuleMaxChars int `json:"capsule_max_chars"`

	// AllowedPaths is an allowlist of directories for import/export operations.
	// Paths outside ~/.moss/exports require either being in this list or AllowUnsafePaths=true.
	// Paths should be absolute (relative paths are ignored).
	AllowedPaths []string `json:"allowed_paths,omitempty"`

	// AllowUnsafePaths disables directory restrictions for import/export.
	// When true, any directory is allowed (but symlink and extension checks still apply).
	// Use with caution: enables file read/write outside ~/.moss/exports.
	AllowUnsafePaths bool `json:"allow_unsafe_paths,omitempty"`

	// DBMaxOpenConns limits the maximum number of open database connections.
	// If set to 1, all database access is serialized (reduces "database is locked" errors).
	// 0 means use sql.DB default (unlimited). Only set if you experience contention.
	DBMaxOpenConns int `json:"db_max_open_conns,omitempty"`

	// DBMaxIdleConns limits the maximum number of idle database connections.
	// 0 means use sql.DB default. Typically set equal to DBMaxOpenConns.
	DBMaxIdleConns int `json:"db_max_idle_conns,omitempty"`

	// DisabledTools is a list of MCP tool names to exclude from registration.
	// All 15 tools are enabled by default. Unknown tool names are logged as warnings.
	DisabledTools []string `json:"disabled_tools,omitempty"`

	// DisabledTypes is a list of type names to disable entirely.
	// All tools belonging to disabled types are excluded from registration.
	// Known types: "capsule". Unknown type names are logged as warnings.
	DisabledTypes []string `json:"disabled_types,omitempty"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		CapsuleMaxChars: 12000,
	}
}

// Load loads configuration from baseDir/config.json.
// Returns default config if the file doesn't exist.
// The baseDir parameter allows tests to use t.TempDir() instead of ~/.moss.
func Load(baseDir string) (*Config, error) {
	return loadFile(filepath.Join(baseDir, "config.json"))
}

// LoadWithRepo loads configuration from both global (~/.moss) and repo (.moss) directories.
// Repo config is found by walking upward from startDir to find the nearest .moss/config.json.
// Repo config takes precedence for scalar values; arrays are merged (deduplicated).
// Either or both configs may be missing.
func LoadWithRepo(globalDir, startDir string) (*Config, error) {
	global, err := loadFileRaw(filepath.Join(globalDir, "config.json"))
	if err != nil {
		return nil, err
	}

	// Walk upward from startDir to find repo config
	repoConfigPath := FindRepoConfig(startDir)
	repo, err := loadFileRaw(repoConfigPath)
	if err != nil {
		return nil, err
	}

	// Apply defaults, then global, then repo
	return Merge(Merge(DefaultConfig(), global), repo), nil
}

// FindRepoConfig walks upward from startDir to find the nearest .moss/config.json.
// Returns the path if found, or empty string if not found.
func FindRepoConfig(startDir string) string {
	dir := startDir
	for {
		configPath := filepath.Join(dir, ".moss", "config.json")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, not found
			return ""
		}
		dir = parent
	}
}

// loadFileRaw loads configuration from a specific file path.
// Returns zero-valued config if the file doesn't exist (not defaults).
func loadFileRaw(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// File doesn't exist, return zero config
			return &Config{}, nil
		}
		return nil, err
	}

	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadFile loads configuration from a specific file path.
// Returns default config if the file doesn't exist.
func loadFile(configPath string) (*Config, error) {
	cfg, err := loadFileRaw(configPath)
	if err != nil {
		return nil, err
	}
	return Merge(DefaultConfig(), cfg), nil
}

// Merge combines base and overlay configs.
// Overlay values take precedence for scalars; arrays are merged and deduplicated.
func Merge(base, overlay *Config) *Config {
	result := &Config{}

	// Scalars: overlay wins if non-zero, else base
	result.CapsuleMaxChars = overlay.CapsuleMaxChars
	if result.CapsuleMaxChars == 0 {
		result.CapsuleMaxChars = base.CapsuleMaxChars
	}

	result.DBMaxOpenConns = overlay.DBMaxOpenConns
	if result.DBMaxOpenConns == 0 {
		result.DBMaxOpenConns = base.DBMaxOpenConns
	}

	result.DBMaxIdleConns = overlay.DBMaxIdleConns
	if result.DBMaxIdleConns == 0 {
		result.DBMaxIdleConns = base.DBMaxIdleConns
	}

	// Booleans: overlay wins if true, else base
	result.AllowUnsafePaths = base.AllowUnsafePaths || overlay.AllowUnsafePaths

	// Arrays: merge and deduplicate
	result.AllowedPaths = mergeStringSlice(base.AllowedPaths, overlay.AllowedPaths)
	result.DisabledTools = mergeStringSlice(base.DisabledTools, overlay.DisabledTools)
	result.DisabledTypes = mergeStringSlice(base.DisabledTypes, overlay.DisabledTypes)

	return result
}

// mergeStringSlice combines two slices, trims whitespace, and removes duplicates.
func mergeStringSlice(a, b []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(a)+len(b))

	for _, s := range a {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}
