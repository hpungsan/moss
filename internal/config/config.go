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
