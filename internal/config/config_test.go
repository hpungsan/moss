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

func TestLoad_DisabledTools(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	if err := os.WriteFile(configPath, []byte(`{"disabled_tools": ["capsule_purge", "capsule_bulk_delete"]}`), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.DisabledTools) != 2 {
		t.Fatalf("DisabledTools length = %d, want 2", len(cfg.DisabledTools))
	}
	if cfg.DisabledTools[0] != "capsule_purge" {
		t.Errorf("DisabledTools[0] = %q, want %q", cfg.DisabledTools[0], "capsule_purge")
	}
	if cfg.DisabledTools[1] != "capsule_bulk_delete" {
		t.Errorf("DisabledTools[1] = %q, want %q", cfg.DisabledTools[1], "capsule_bulk_delete")
	}
}

func TestLoad_DisabledToolsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	if err := os.WriteFile(configPath, []byte(`{}`), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.DisabledTools) != 0 {
		t.Fatalf("DisabledTools = %v, want nil or empty", cfg.DisabledTools)
	}
}

func TestLoadWithRepo_BothPresent(t *testing.T) {
	globalDir := t.TempDir()
	repoRoot := t.TempDir()

	// Global config
	globalConfig := `{"capsule_max_chars": 8000, "disabled_tools": ["capsule_purge"]}`
	if err := os.WriteFile(filepath.Join(globalDir, "config.json"), []byte(globalConfig), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Repo config at repoRoot/.moss/config.json
	mossDir := filepath.Join(repoRoot, ".moss")
	if err := os.MkdirAll(mossDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	repoConfig := `{"capsule_max_chars": 5000, "disabled_tools": ["capsule_bulk_delete"]}`
	if err := os.WriteFile(filepath.Join(mossDir, "config.json"), []byte(repoConfig), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := LoadWithRepo(globalDir, repoRoot)
	if err != nil {
		t.Fatalf("LoadWithRepo() error = %v", err)
	}

	// Repo overrides scalar
	if cfg.CapsuleMaxChars != 5000 {
		t.Errorf("CapsuleMaxChars = %d, want 5000 (repo override)", cfg.CapsuleMaxChars)
	}

	// Arrays merged
	if len(cfg.DisabledTools) != 2 {
		t.Errorf("DisabledTools length = %d, want 2", len(cfg.DisabledTools))
	}
}

func TestLoadWithRepo_OnlyGlobal(t *testing.T) {
	globalDir := t.TempDir()
	repoDir := t.TempDir() // No config file

	globalConfig := `{"capsule_max_chars": 8000, "disabled_tools": ["capsule_purge"]}`
	if err := os.WriteFile(filepath.Join(globalDir, "config.json"), []byte(globalConfig), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := LoadWithRepo(globalDir, repoDir)
	if err != nil {
		t.Fatalf("LoadWithRepo() error = %v", err)
	}

	if cfg.CapsuleMaxChars != 8000 {
		t.Errorf("CapsuleMaxChars = %d, want 8000", cfg.CapsuleMaxChars)
	}
	if len(cfg.DisabledTools) != 1 || cfg.DisabledTools[0] != "capsule_purge" {
		t.Errorf("DisabledTools = %v, want [capsule_purge]", cfg.DisabledTools)
	}
}

func TestLoadWithRepo_OnlyRepo(t *testing.T) {
	globalDir := t.TempDir() // No config file
	repoRoot := t.TempDir()

	// Repo config at repoRoot/.moss/config.json
	mossDir := filepath.Join(repoRoot, ".moss")
	if err := os.MkdirAll(mossDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	repoConfig := `{"disabled_tools": ["capsule_bulk_delete", "capsule_bulk_update"]}`
	if err := os.WriteFile(filepath.Join(mossDir, "config.json"), []byte(repoConfig), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := LoadWithRepo(globalDir, repoRoot)
	if err != nil {
		t.Fatalf("LoadWithRepo() error = %v", err)
	}

	// Default value preserved
	if cfg.CapsuleMaxChars != 12000 {
		t.Errorf("CapsuleMaxChars = %d, want 12000 (default)", cfg.CapsuleMaxChars)
	}
	if len(cfg.DisabledTools) != 2 {
		t.Errorf("DisabledTools length = %d, want 2", len(cfg.DisabledTools))
	}
}

func TestLoadWithRepo_NeitherPresent(t *testing.T) {
	globalDir := t.TempDir()
	repoDir := t.TempDir()

	cfg, err := LoadWithRepo(globalDir, repoDir)
	if err != nil {
		t.Fatalf("LoadWithRepo() error = %v", err)
	}

	// All defaults
	if cfg.CapsuleMaxChars != 12000 {
		t.Errorf("CapsuleMaxChars = %d, want 12000", cfg.CapsuleMaxChars)
	}
	if len(cfg.DisabledTools) != 0 {
		t.Errorf("DisabledTools = %v, want empty", cfg.DisabledTools)
	}
}

func TestMerge_ScalarOverride(t *testing.T) {
	base := &Config{CapsuleMaxChars: 10000, DBMaxOpenConns: 5}
	overlay := &Config{CapsuleMaxChars: 5000} // DBMaxOpenConns is 0 (zero value)

	result := Merge(base, overlay)

	if result.CapsuleMaxChars != 5000 {
		t.Errorf("CapsuleMaxChars = %d, want 5000 (overlay)", result.CapsuleMaxChars)
	}
	if result.DBMaxOpenConns != 5 {
		t.Errorf("DBMaxOpenConns = %d, want 5 (base, overlay is zero)", result.DBMaxOpenConns)
	}
}

func TestMerge_BooleanOr(t *testing.T) {
	base := &Config{AllowUnsafePaths: true}
	overlay := &Config{AllowUnsafePaths: false}

	result := Merge(base, overlay)

	if !result.AllowUnsafePaths {
		t.Error("AllowUnsafePaths should be true (base OR overlay)")
	}
}

func TestMerge_ArrayMergeDedup(t *testing.T) {
	base := &Config{DisabledTools: []string{"capsule_purge", "capsule_bulk_delete"}}
	overlay := &Config{DisabledTools: []string{"capsule_bulk_delete", "capsule_bulk_update"}}

	result := Merge(base, overlay)

	if len(result.DisabledTools) != 3 {
		t.Errorf("DisabledTools length = %d, want 3 (merged, deduped)", len(result.DisabledTools))
	}

	// Check all three are present
	has := make(map[string]bool)
	for _, s := range result.DisabledTools {
		has[s] = true
	}
	for _, want := range []string{"capsule_purge", "capsule_bulk_delete", "capsule_bulk_update"} {
		if !has[want] {
			t.Errorf("DisabledTools missing %q", want)
		}
	}
}

func TestFindRepoConfig_InCurrentDir(t *testing.T) {
	tmpDir := t.TempDir()
	mossDir := filepath.Join(tmpDir, ".moss")
	if err := os.MkdirAll(mossDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	configPath := filepath.Join(mossDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	found := FindRepoConfig(tmpDir)
	if found != configPath {
		t.Errorf("FindRepoConfig() = %q, want %q", found, configPath)
	}
}

func TestFindRepoConfig_InParentDir(t *testing.T) {
	// Create: tmpDir/.moss/config.json
	//         tmpDir/subdir/deeper/
	tmpDir := t.TempDir()
	mossDir := filepath.Join(tmpDir, ".moss")
	if err := os.MkdirAll(mossDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	configPath := filepath.Join(mossDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	subdir := filepath.Join(tmpDir, "subdir", "deeper")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Start from subdir, should find config in parent
	found := FindRepoConfig(subdir)
	if found != configPath {
		t.Errorf("FindRepoConfig() = %q, want %q", found, configPath)
	}
}

func TestFindRepoConfig_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	// No .moss directory

	found := FindRepoConfig(tmpDir)
	if found != "" {
		t.Errorf("FindRepoConfig() = %q, want empty string", found)
	}
}

func TestLoadWithRepo_WalksUpward(t *testing.T) {
	// Create: tmpDir/.moss/config.json with disabled_tools
	//         tmpDir/subdir/
	tmpDir := t.TempDir()
	globalDir := t.TempDir() // Separate global dir

	mossDir := filepath.Join(tmpDir, ".moss")
	if err := os.MkdirAll(mossDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	repoConfig := `{"disabled_tools": ["capsule_purge"]}`
	if err := os.WriteFile(filepath.Join(mossDir, "config.json"), []byte(repoConfig), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	subdir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Load from subdir, should find repo config in parent
	cfg, err := LoadWithRepo(globalDir, subdir)
	if err != nil {
		t.Fatalf("LoadWithRepo() error = %v", err)
	}

	if len(cfg.DisabledTools) != 1 || cfg.DisabledTools[0] != "capsule_purge" {
		t.Errorf("DisabledTools = %v, want [capsule_purge]", cfg.DisabledTools)
	}
}
