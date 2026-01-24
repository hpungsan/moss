package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DefaultWhenMissing(t *testing.T) {
	tmpDir := t.TempDir()

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.CapsuleMaxChars != DefaultConfig().CapsuleMaxChars {
		t.Fatalf("CapsuleMaxChars = %d, want %d", cfg.CapsuleMaxChars, DefaultConfig().CapsuleMaxChars)
	}
}

func TestLoad_OverridesFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	if err := os.WriteFile(configPath, []byte(`{"capsule_max_chars": 500}`), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.CapsuleMaxChars != 500 {
		t.Fatalf("CapsuleMaxChars = %d, want %d", cfg.CapsuleMaxChars, 500)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	if err := os.WriteFile(configPath, []byte(`{not json}`), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := Load(tmpDir); err == nil {
		t.Fatalf("Load() expected error, got nil")
	}
}
