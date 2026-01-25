package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/ops"
)

// setupTestDB creates a temporary database for testing.
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	cleanup := func() {
		database.Close()
	}
	return database, cleanup
}

// testConfig returns a default config for testing.
func testConfig() *config.Config {
	return &config.Config{
		CapsuleMaxChars: 50000,
	}
}

// validCapsuleText returns a valid capsule with all required sections.
func validCapsuleText() string {
	return `## Objective
Test objective
## Current status
Test status
## Decisions
Test decisions
## Next actions
Test actions
## Key locations
Test locations
## Open questions
None`
}

// TestParseTags tests the parseTags helper function.
func TestParseTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single tag",
			input:    "foo",
			expected: []string{"foo"},
		},
		{
			name:     "multiple tags",
			input:    "foo,bar,baz",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "tags with spaces",
			input:    " foo , bar , baz ",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "empty tags filtered",
			input:    "foo,,bar,",
			expected: []string{"foo", "bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTags(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d tags, got %d", len(tt.expected), len(result))
				return
			}
			for i, tag := range result {
				if tag != tt.expected[i] {
					t.Errorf("expected tag[%d]=%q, got %q", i, tt.expected[i], tag)
				}
			}
		})
	}
}

// TestParseDuration tests the parseDuration helper function.
func TestParseDuration(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int
		expectError bool
	}{
		{
			name:     "valid days",
			input:    "7d",
			expected: 7,
		},
		{
			name:     "zero days",
			input:    "0d",
			expected: 0,
		},
		{
			name:     "large number",
			input:    "365d",
			expected: 365,
		},
		{
			name:        "negative days",
			input:       "-7d",
			expectError: true,
		},
		{
			name:        "no suffix",
			input:       "7",
			expectError: true,
		},
		{
			name:        "wrong suffix",
			input:       "7h",
			expectError: true,
		},
		{
			name:        "invalid number",
			input:       "abcd",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDuration(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestCLIStore tests the store command.
func TestCLIStore(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	cfg := testConfig()

	app := newCLIApp(database, cfg)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create a pipe for stdin
	oldStdin := os.Stdin
	stdinR, stdinW, _ := os.Pipe()
	os.Stdin = stdinR

	// Write capsule text to stdin
	go func() {
		_, _ = stdinW.WriteString(validCapsuleText())
		stdinW.Close()
	}()

	// Run store command
	err := app.Run([]string{"moss", "store", "--name=test-capsule", "--tags=foo,bar"})

	// Restore stdin
	os.Stdin = oldStdin

	// Read stdout
	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("store command failed: %v", err)
	}

	// Parse output
	var output ops.StoreOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse output: %v\nOutput: %s", err, buf.String())
	}

	if output.ID == "" {
		t.Error("expected non-empty ID")
	}
	if output.TaskLink.MossCapsule != "test-capsule" {
		t.Errorf("expected task_link.moss_capsule=test-capsule, got %s", output.TaskLink.MossCapsule)
	}
}

// TestCLIFetch tests the fetch command.
func TestCLIFetch(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	cfg := testConfig()

	// Store a capsule first
	name := "fetch-test"
	storeOutput, err := ops.Store(database, cfg, ops.StoreInput{
		Workspace:   "default",
		Name:        &name,
		CapsuleText: validCapsuleText(),
	})
	if err != nil {
		t.Fatalf("failed to store test capsule: %v", err)
	}

	app := newCLIApp(database, cfg)

	t.Run("fetch by name", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := app.Run([]string{"moss", "fetch", "--name=fetch-test"})

		w.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("fetch command failed: %v", err)
		}

		var output ops.FetchOutput
		if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
			t.Fatalf("failed to parse output: %v", err)
		}

		if output.ID != storeOutput.ID {
			t.Errorf("expected ID=%s, got %s", storeOutput.ID, output.ID)
		}
	})

	t.Run("fetch by id", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := app.Run([]string{"moss", "fetch", storeOutput.ID})

		w.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("fetch command failed: %v", err)
		}

		var output ops.FetchOutput
		if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
			t.Fatalf("failed to parse output: %v", err)
		}

		if output.ID != storeOutput.ID {
			t.Errorf("expected ID=%s, got %s", storeOutput.ID, output.ID)
		}
	})
}

// TestCLIList tests the list command.
func TestCLIList(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	cfg := testConfig()

	// Store some capsules
	for i := range 3 {
		name := "list-test-" + string(rune('a'+i))
		_, err := ops.Store(database, cfg, ops.StoreInput{
			Workspace:   "default",
			Name:        &name,
			CapsuleText: validCapsuleText(),
		})
		if err != nil {
			t.Fatalf("failed to store test capsule: %v", err)
		}
	}

	app := newCLIApp(database, cfg)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := app.Run([]string{"moss", "list"})

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	var output ops.ListOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if len(output.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(output.Items))
	}
	if output.Pagination.Total != 3 {
		t.Errorf("expected total=3, got %d", output.Pagination.Total)
	}
}

// TestCLIDelete tests the delete command.
func TestCLIDelete(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	cfg := testConfig()

	// Store a capsule first
	name := "delete-test"
	storeOutput, err := ops.Store(database, cfg, ops.StoreInput{
		Workspace:   "default",
		Name:        &name,
		CapsuleText: validCapsuleText(),
	})
	if err != nil {
		t.Fatalf("failed to store test capsule: %v", err)
	}

	app := newCLIApp(database, cfg)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = app.Run([]string{"moss", "delete", "--name=delete-test"})

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("delete command failed: %v", err)
	}

	var output ops.DeleteOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if !output.Deleted {
		t.Error("expected deleted=true")
	}
	if output.ID != storeOutput.ID {
		t.Errorf("expected ID=%s, got %s", storeOutput.ID, output.ID)
	}
}

// TestCLILatest tests the latest command.
func TestCLILatest(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	cfg := testConfig()

	// Store a capsule
	name := "latest-test"
	_, err := ops.Store(database, cfg, ops.StoreInput{
		Workspace:   "default",
		Name:        &name,
		CapsuleText: validCapsuleText(),
	})
	if err != nil {
		t.Fatalf("failed to store test capsule: %v", err)
	}

	app := newCLIApp(database, cfg)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = app.Run([]string{"moss", "latest"})

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("latest command failed: %v", err)
	}

	var output ops.LatestOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if output.Item == nil {
		t.Fatal("expected non-nil item")
	}
	if output.Item.Name == nil || *output.Item.Name != "latest-test" {
		t.Error("expected name=latest-test")
	}
}

// TestCLIExportImport tests the export and import commands.
func TestCLIExportImport(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	cfg := testConfig()

	// Store some capsules
	for i := range 2 {
		name := "export-test-" + string(rune('a'+i))
		_, err := ops.Store(database, cfg, ops.StoreInput{
			Workspace:   "default",
			Name:        &name,
			CapsuleText: validCapsuleText(),
		})
		if err != nil {
			t.Fatalf("failed to store test capsule: %v", err)
		}
	}

	app := newCLIApp(database, cfg)
	exportPath := filepath.Join(t.TempDir(), "export.jsonl")

	// Test export
	t.Run("export", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := app.Run([]string{"moss", "export", "--path=" + exportPath})

		w.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("export command failed: %v", err)
		}

		var output ops.ExportOutput
		if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
			t.Fatalf("failed to parse output: %v", err)
		}

		if output.Count != 2 {
			t.Errorf("expected count=2, got %d", output.Count)
		}
		if output.Path != exportPath {
			t.Errorf("expected path=%s, got %s", exportPath, output.Path)
		}
	})

	// Create new database for import test
	database2, cleanup2 := setupTestDB(t)
	defer cleanup2()
	app2 := newCLIApp(database2, cfg)

	// Test import
	t.Run("import", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := app2.Run([]string{"moss", "import", "--path=" + exportPath})

		w.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("import command failed: %v", err)
		}

		var output ops.ImportOutput
		if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
			t.Fatalf("failed to parse output: %v", err)
		}

		if output.Imported != 2 {
			t.Errorf("expected imported=2, got %d", output.Imported)
		}
	})
}

// TestCLIPurge tests the purge command.
func TestCLIPurge(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	cfg := testConfig()

	// Store and delete a capsule
	name := "purge-test"
	storeOutput, err := ops.Store(database, cfg, ops.StoreInput{
		Workspace:   "default",
		Name:        &name,
		CapsuleText: validCapsuleText(),
	})
	if err != nil {
		t.Fatalf("failed to store test capsule: %v", err)
	}

	_, err = ops.Delete(database, ops.DeleteInput{ID: storeOutput.ID})
	if err != nil {
		t.Fatalf("failed to delete test capsule: %v", err)
	}

	app := newCLIApp(database, cfg)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Purge without --older-than to purge all deleted capsules
	err = app.Run([]string{"moss", "purge"})

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("purge command failed: %v", err)
	}

	var output ops.PurgeOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if output.Purged != 1 {
		t.Errorf("expected purged=1, got %d", output.Purged)
	}
}

// TestCLIInventory tests the inventory command.
func TestCLIInventory(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	cfg := testConfig()

	// Store capsules in different workspaces
	workspaces := []string{"ws1", "ws2"}
	for _, ws := range workspaces {
		name := "inv-" + ws
		_, err := ops.Store(database, cfg, ops.StoreInput{
			Workspace:   ws,
			Name:        &name,
			CapsuleText: validCapsuleText(),
		})
		if err != nil {
			t.Fatalf("failed to store test capsule: %v", err)
		}
	}

	app := newCLIApp(database, cfg)

	t.Run("all workspaces", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := app.Run([]string{"moss", "inventory"})

		w.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("inventory command failed: %v", err)
		}

		var output ops.InventoryOutput
		if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
			t.Fatalf("failed to parse output: %v", err)
		}

		if len(output.Items) != 2 {
			t.Errorf("expected 2 items, got %d", len(output.Items))
		}
	})

	t.Run("filter by workspace", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := app.Run([]string{"moss", "inventory", "--workspace=ws1"})

		w.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("inventory command failed: %v", err)
		}

		var output ops.InventoryOutput
		if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
			t.Fatalf("failed to parse output: %v", err)
		}

		if len(output.Items) != 1 {
			t.Errorf("expected 1 item, got %d", len(output.Items))
		}
	})
}

// TestCLIUpdate tests the update command.
func TestCLIUpdate(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	cfg := testConfig()

	// Store a capsule first
	name := "update-test"
	storeOutput, err := ops.Store(database, cfg, ops.StoreInput{
		Workspace:   "default",
		Name:        &name,
		CapsuleText: validCapsuleText(),
	})
	if err != nil {
		t.Fatalf("failed to store test capsule: %v", err)
	}

	app := newCLIApp(database, cfg)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = app.Run([]string{"moss", "update", "--name=update-test", "--title=New Title"})

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("update command failed: %v", err)
	}

	var output ops.UpdateOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if output.ID != storeOutput.ID {
		t.Errorf("expected ID=%s, got %s", storeOutput.ID, output.ID)
	}

	// Verify the update
	fetchOutput, err := ops.Fetch(database, ops.FetchInput{ID: storeOutput.ID})
	if err != nil {
		t.Fatalf("failed to fetch updated capsule: %v", err)
	}
	if fetchOutput.Title == nil || *fetchOutput.Title != "New Title" {
		t.Errorf("expected title=New Title, got %v", fetchOutput.Title)
	}
}

// TestCLIErrorHandling tests error handling in CLI commands.
func TestCLIErrorHandling(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	cfg := testConfig()

	app := newCLIApp(database, cfg)

	t.Run("fetch not found returns error", func(t *testing.T) {
		// cli.Exit writes to stderr, so just verify the error is returned
		err := app.Run([]string{"moss", "fetch", "--name=nonexistent"})
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("delete not found returns error", func(t *testing.T) {
		err := app.Run([]string{"moss", "delete", "--name=nonexistent"})
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("invalid duration format returns error", func(t *testing.T) {
		err := app.Run([]string{"moss", "purge", "--older-than=invalid"})
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

// TestIsCLIMode tests the isCLIMode function.
func TestIsCLIMode(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "no args",
			args:     []string{"moss"},
			expected: false,
		},
		{
			name:     "store command",
			args:     []string{"moss", "store"},
			expected: true,
		},
		{
			name:     "fetch command",
			args:     []string{"moss", "fetch"},
			expected: true,
		},
		{
			name:     "help flag",
			args:     []string{"moss", "--help"},
			expected: true,
		},
		{
			name:     "version flag",
			args:     []string{"moss", "--version"},
			expected: true,
		},
		{
			name:     "short help flag",
			args:     []string{"moss", "-h"},
			expected: true,
		},
		{
			name:     "short version flag",
			args:     []string{"moss", "-v"},
			expected: true,
		},
		{
			name:     "unknown arg defaults to MCP",
			args:     []string{"moss", "--unknown"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore os.Args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = tt.args
			result := isCLIMode()

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestIsHelpOrVersion tests the isHelpOrVersion function.
func TestIsHelpOrVersion(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "no args",
			args:     []string{"moss"},
			expected: false,
		},
		{
			name:     "help flag",
			args:     []string{"moss", "--help"},
			expected: true,
		},
		{
			name:     "short help flag",
			args:     []string{"moss", "-h"},
			expected: true,
		},
		{
			name:     "version flag",
			args:     []string{"moss", "--version"},
			expected: true,
		},
		{
			name:     "short version flag",
			args:     []string{"moss", "-v"},
			expected: true,
		},
		{
			name:     "help subcommand",
			args:     []string{"moss", "help"},
			expected: true,
		},
		{
			name:     "store command is not help",
			args:     []string{"moss", "store"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = tt.args
			result := isHelpOrVersion()

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestReadStdinWithLimit tests the readStdin function respects size limits.
func TestReadStdinWithLimit(t *testing.T) {
	t.Run("within limit", func(t *testing.T) {
		// Create a pipe with content under limit
		content := "small content"
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}

		// Write and close in goroutine
		go func() {
			_, _ = w.WriteString(content)
			w.Close()
		}()

		// Temporarily replace stdin
		oldStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = oldStdin }()

		result, err := readStdin(1000)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != content {
			t.Errorf("expected %q, got %q", content, result)
		}
	})

	t.Run("exceeds limit", func(t *testing.T) {
		// Create content that exceeds the limit
		content := strings.Repeat("x", 100)
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}

		go func() {
			_, _ = w.WriteString(content)
			w.Close()
		}()

		oldStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = oldStdin }()

		// Limit is 50 bytes, content is 100
		_, err = readStdin(50)
		if err == nil {
			t.Error("expected error for content exceeding limit, got nil")
		}
	})
}
