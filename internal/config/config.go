package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
	configPath := filepath.Join(baseDir, "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// File doesn't exist, return defaults
			return DefaultConfig(), nil
		}
		return nil, err
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
