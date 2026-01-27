package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

// testSetup creates a temporary database and config for testing.
func testSetup(t *testing.T) (*sql.DB, *config.Config, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	cfg := config.DefaultConfig()

	cleanup := func() {
		database.Close()
	}

	return database, cfg, cleanup
}

// makeRequest creates a CallToolRequest with the given arguments.
func makeRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

// validCapsuleText returns a capsule text with all required sections.
func validCapsuleText() string {
	return `## Objective
Test the MCP integration

## Current status
Writing tests

## Decisions
Use table-driven tests

## Next actions
Run the tests

## Key locations
internal/mcp/mcp_test.go

## Open questions
None for now`
}

// TestHandleStore tests the store handler.
func TestHandleStore(t *testing.T) {
	database, cfg, cleanup := testSetup(t)
	defer cleanup()

	h := NewHandlers(database, cfg)
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		wantError bool
		errorCode string
	}{
		{
			name: "store valid capsule",
			args: map[string]any{
				"capsule_text": validCapsuleText(),
				"workspace":    "test",
				"name":         "test-capsule",
			},
			wantError: false,
		},
		{
			name: "store without capsule_text",
			args: map[string]any{
				"workspace": "test",
				"name":      "empty",
			},
			wantError: true,
			errorCode: "INVALID_REQUEST",
		},
		{
			name: "store thin capsule without allow_thin",
			args: map[string]any{
				"capsule_text": "too short",
				"workspace":    "test",
				"name":         "thin",
			},
			wantError: true,
			errorCode: "CAPSULE_TOO_THIN",
		},
		{
			name: "store thin capsule with allow_thin",
			args: map[string]any{
				"capsule_text": "thin but allowed",
				"workspace":    "test",
				"name":         "thin-allowed",
				"allow_thin":   true,
			},
			wantError: false,
		},
		{
			name: "store duplicate name with mode:error",
			args: map[string]any{
				"capsule_text": validCapsuleText(),
				"workspace":    "test",
				"name":         "test-capsule", // already exists from first test
				"mode":         "error",
			},
			wantError: true,
			errorCode: "NAME_ALREADY_EXISTS",
		},
		{
			name: "store duplicate name with mode:replace",
			args: map[string]any{
				"capsule_text": validCapsuleText(),
				"workspace":    "test",
				"name":         "test-capsule",
				"mode":         "replace",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest(tt.args)
			result, err := h.HandleStore(ctx, req)

			if err != nil {
				t.Fatalf("handler returned error: %v", err)
			}

			if tt.wantError {
				if !result.IsError {
					t.Errorf("expected error result, got success")
				}
				if tt.errorCode != "" {
					assertErrorCode(t, result, tt.errorCode)
				}
			} else {
				if result.IsError {
					t.Errorf("expected success, got error: %v", extractErrorMessage(result))
				}
			}
		})
	}
}

// TestHandleFetch tests the fetch handler.
func TestHandleFetch(t *testing.T) {
	database, cfg, cleanup := testSetup(t)
	defer cleanup()

	h := NewHandlers(database, cfg)
	ctx := context.Background()

	// Store a capsule first
	storeReq := makeRequest(map[string]any{
		"capsule_text": validCapsuleText(),
		"workspace":    "test",
		"name":         "fetch-test",
	})
	storeResult, _ := h.HandleStore(ctx, storeReq)
	if storeResult.IsError {
		t.Fatalf("setup store failed: %v", extractErrorMessage(storeResult))
	}

	// Extract the ID from store result
	var storeOutput map[string]any
	if err := json.Unmarshal([]byte(storeResult.Content[0].(mcp.TextContent).Text), &storeOutput); err != nil {
		t.Fatalf("failed to unmarshal store result: %v", err)
	}
	capsuleID := storeOutput["id"].(string)

	tests := []struct {
		name      string
		args      map[string]any
		wantError bool
		errorCode string
	}{
		{
			name: "fetch by name",
			args: map[string]any{
				"workspace": "test",
				"name":      "fetch-test",
			},
			wantError: false,
		},
		{
			name: "fetch by id",
			args: map[string]any{
				"id": capsuleID,
			},
			wantError: false,
		},
		{
			name: "fetch non-existent",
			args: map[string]any{
				"workspace": "test",
				"name":      "does-not-exist",
			},
			wantError: true,
			errorCode: "NOT_FOUND",
		},
		{
			name: "fetch with ambiguous addressing",
			args: map[string]any{
				"id":        capsuleID,
				"workspace": "test",
				"name":      "fetch-test",
			},
			wantError: true,
			errorCode: "AMBIGUOUS_ADDRESSING",
		},
		{
			name:      "fetch with no addressing",
			args:      map[string]any{},
			wantError: true,
			errorCode: "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest(tt.args)
			result, err := h.HandleFetch(ctx, req)

			if err != nil {
				t.Fatalf("handler returned error: %v", err)
			}

			if tt.wantError {
				if !result.IsError {
					t.Errorf("expected error result, got success")
				}
				if tt.errorCode != "" {
					assertErrorCode(t, result, tt.errorCode)
				}
			} else {
				if result.IsError {
					t.Errorf("expected success, got error: %v", extractErrorMessage(result))
				}
			}
		})
	}
}

// TestHandleFetchMany tests the fetch_many handler.
func TestHandleFetchMany(t *testing.T) {
	database, cfg, cleanup := testSetup(t)
	defer cleanup()

	h := NewHandlers(database, cfg)
	ctx := context.Background()

	// Store some capsules
	for i, name := range []string{"one", "two", "three"} {
		req := makeRequest(map[string]any{
			"capsule_text": validCapsuleText(),
			"workspace":    "test",
			"name":         name,
			"allow_thin":   i > 0, // vary the setup
		})
		if _, err := h.HandleStore(ctx, req); err != nil {
			t.Fatalf("setup store failed: %v", err)
		}
	}

	tests := []struct {
		name          string
		args          map[string]any
		wantItems     int
		wantErrors    int
		wantToolError bool
	}{
		{
			name: "fetch multiple existing",
			args: map[string]any{
				"items": []any{
					map[string]any{"workspace": "test", "name": "one"},
					map[string]any{"workspace": "test", "name": "two"},
				},
			},
			wantItems:  2,
			wantErrors: 0,
		},
		{
			name: "fetch with some missing",
			args: map[string]any{
				"items": []any{
					map[string]any{"workspace": "test", "name": "one"},
					map[string]any{"workspace": "test", "name": "missing"},
				},
			},
			wantItems:  1,
			wantErrors: 1,
		},
		{
			name: "fetch empty list",
			args: map[string]any{
				"items": []any{},
			},
			wantItems:  0,
			wantErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest(tt.args)
			result, err := h.HandleFetchMany(ctx, req)

			if err != nil {
				t.Fatalf("handler returned error: %v", err)
			}

			if tt.wantToolError {
				if !result.IsError {
					t.Errorf("expected tool error")
				}
				return
			}

			if result.IsError {
				t.Errorf("expected success, got error: %v", extractErrorMessage(result))
				return
			}

			// Parse response
			var output map[string]any
			if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &output); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			items, _ := output["items"].([]any)
			errors, _ := output["errors"].([]any)

			if len(items) != tt.wantItems {
				t.Errorf("got %d items, want %d", len(items), tt.wantItems)
			}
			if len(errors) != tt.wantErrors {
				t.Errorf("got %d errors, want %d", len(errors), tt.wantErrors)
			}
		})
	}
}

// TestHandleUpdate tests the update handler.
func TestHandleUpdate(t *testing.T) {
	database, cfg, cleanup := testSetup(t)
	defer cleanup()

	h := NewHandlers(database, cfg)
	ctx := context.Background()

	// Store a capsule first
	storeReq := makeRequest(map[string]any{
		"capsule_text": validCapsuleText(),
		"workspace":    "test",
		"name":         "update-test",
	})
	if _, err := h.HandleStore(ctx, storeReq); err != nil {
		t.Fatalf("setup store failed: %v", err)
	}

	tests := []struct {
		name      string
		args      map[string]any
		wantError bool
		errorCode string
	}{
		{
			name: "update title",
			args: map[string]any{
				"workspace": "test",
				"name":      "update-test",
				"title":     "New Title",
			},
			wantError: false,
		},
		{
			name: "update capsule_text",
			args: map[string]any{
				"workspace":    "test",
				"name":         "update-test",
				"capsule_text": validCapsuleText() + "\n\nUpdated!",
			},
			wantError: false,
		},
		{
			name: "update non-existent",
			args: map[string]any{
				"workspace": "test",
				"name":      "missing",
				"title":     "New Title",
			},
			wantError: true,
			errorCode: "NOT_FOUND",
		},
		{
			name: "update with no editable fields",
			args: map[string]any{
				"workspace": "test",
				"name":      "update-test",
			},
			wantError: true,
			errorCode: "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest(tt.args)
			result, err := h.HandleUpdate(ctx, req)

			if err != nil {
				t.Fatalf("handler returned error: %v", err)
			}

			if tt.wantError {
				if !result.IsError {
					t.Errorf("expected error result, got success")
				}
				if tt.errorCode != "" {
					assertErrorCode(t, result, tt.errorCode)
				}
			} else {
				if result.IsError {
					t.Errorf("expected success, got error: %v", extractErrorMessage(result))
				}
			}
		})
	}
}

// TestHandleDelete tests the delete handler.
func TestHandleDelete(t *testing.T) {
	database, cfg, cleanup := testSetup(t)
	defer cleanup()

	h := NewHandlers(database, cfg)
	ctx := context.Background()

	// Store a capsule first
	storeReq := makeRequest(map[string]any{
		"capsule_text": validCapsuleText(),
		"workspace":    "test",
		"name":         "delete-test",
	})
	if _, err := h.HandleStore(ctx, storeReq); err != nil {
		t.Fatalf("setup store failed: %v", err)
	}

	tests := []struct {
		name      string
		args      map[string]any
		wantError bool
		errorCode string
	}{
		{
			name: "delete existing",
			args: map[string]any{
				"workspace": "test",
				"name":      "delete-test",
			},
			wantError: false,
		},
		{
			name: "delete already deleted",
			args: map[string]any{
				"workspace": "test",
				"name":      "delete-test",
			},
			wantError: true,
			errorCode: "NOT_FOUND",
		},
		{
			name: "delete non-existent",
			args: map[string]any{
				"workspace": "test",
				"name":      "never-existed",
			},
			wantError: true,
			errorCode: "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest(tt.args)
			result, err := h.HandleDelete(ctx, req)

			if err != nil {
				t.Fatalf("handler returned error: %v", err)
			}

			if tt.wantError {
				if !result.IsError {
					t.Errorf("expected error result, got success")
				}
				if tt.errorCode != "" {
					assertErrorCode(t, result, tt.errorCode)
				}
			} else {
				if result.IsError {
					t.Errorf("expected success, got error: %v", extractErrorMessage(result))
				}
			}
		})
	}
}

// TestHandleLatest tests the latest handler with contract assertions.
func TestHandleLatest(t *testing.T) {
	database, cfg, cleanup := testSetup(t)
	defer cleanup()

	h := NewHandlers(database, cfg)
	ctx := context.Background()

	// Test empty workspace returns null item
	t.Run("empty workspace returns null", func(t *testing.T) {
		req := makeRequest(map[string]any{"workspace": "empty"})
		result, err := h.HandleLatest(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		output := parseOutput(t, result)
		if output["item"] != nil {
			t.Errorf("expected null item for empty workspace")
		}
	})

	// Store a capsule, then delete it
	storeReq := makeRequest(map[string]any{
		"capsule_text": validCapsuleText(),
		"workspace":    "test",
		"name":         "latest-test",
	})
	if _, err := h.HandleStore(ctx, storeReq); err != nil {
		t.Fatalf("setup store failed: %v", err)
	}

	// Test include_text:false (default) omits capsule_text
	t.Run("include_text:false omits capsule_text", func(t *testing.T) {
		req := makeRequest(map[string]any{
			"workspace":    "test",
			"include_text": false,
		})
		result, err := h.HandleLatest(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		output := parseOutput(t, result)
		item := output["item"].(map[string]any)
		if item["capsule_text"] != nil && item["capsule_text"] != "" {
			t.Error("include_text:false should omit capsule_text")
		}
	})

	// Test include_text:true includes capsule_text
	t.Run("include_text:true includes capsule_text", func(t *testing.T) {
		req := makeRequest(map[string]any{
			"workspace":    "test",
			"include_text": true,
		})
		result, err := h.HandleLatest(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		output := parseOutput(t, result)
		item := output["item"].(map[string]any)
		if item["capsule_text"] == nil || item["capsule_text"] == "" {
			t.Error("include_text:true should include capsule_text")
		}
	})

	// Delete the capsule for include_deleted test
	deleteReq := makeRequest(map[string]any{"workspace": "test", "name": "latest-test"})
	if _, err := h.HandleDelete(ctx, deleteReq); err != nil {
		t.Fatalf("setup delete failed: %v", err)
	}

	// Test include_deleted:false returns null (deleted capsule hidden)
	t.Run("include_deleted:false hides deleted", func(t *testing.T) {
		req := makeRequest(map[string]any{
			"workspace":       "test",
			"include_deleted": false,
		})
		result, err := h.HandleLatest(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		output := parseOutput(t, result)
		if output["item"] != nil {
			t.Error("include_deleted:false should return null for deleted-only workspace")
		}
	})

	// Test include_deleted:true returns deleted capsule
	t.Run("include_deleted:true shows deleted", func(t *testing.T) {
		req := makeRequest(map[string]any{
			"workspace":       "test",
			"include_deleted": true,
		})
		result, err := h.HandleLatest(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		output := parseOutput(t, result)
		if output["item"] == nil {
			t.Error("include_deleted:true should return deleted capsule")
		}
	})
}

// TestHandleList tests the list handler with contract assertions.
func TestHandleList(t *testing.T) {
	database, cfg, cleanup := testSetup(t)
	defer cleanup()

	h := NewHandlers(database, cfg)
	ctx := context.Background()

	// Store 3 capsules, delete 1
	for _, name := range []string{"list-1", "list-2", "list-3"} {
		req := makeRequest(map[string]any{
			"capsule_text": validCapsuleText(),
			"workspace":    "test",
			"name":         name,
		})
		if _, err := h.HandleStore(ctx, req); err != nil {
			t.Fatalf("setup store failed: %v", err)
		}
	}
	deleteReq := makeRequest(map[string]any{"workspace": "test", "name": "list-3"})
	if _, err := h.HandleDelete(ctx, deleteReq); err != nil {
		t.Fatalf("setup delete failed: %v", err)
	}

	// Test pagination metadata
	t.Run("pagination metadata correct", func(t *testing.T) {
		req := makeRequest(map[string]any{
			"workspace": "test",
			"limit":     1,
			"offset":    0,
		})
		result, err := h.HandleList(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		output := parseOutput(t, result)
		pagination := output["pagination"].(map[string]any)

		if int(pagination["limit"].(float64)) != 1 {
			t.Errorf("pagination.limit = %v, want 1", pagination["limit"])
		}
		if int(pagination["offset"].(float64)) != 0 {
			t.Errorf("pagination.offset = %v, want 0", pagination["offset"])
		}
		if pagination["has_more"] != true {
			t.Errorf("pagination.has_more = %v, want true", pagination["has_more"])
		}
		if int(pagination["total"].(float64)) != 2 {
			t.Errorf("pagination.total = %v, want 2 (active only)", pagination["total"])
		}
	})

	// Test include_deleted:false (default) excludes deleted
	t.Run("include_deleted:false excludes deleted", func(t *testing.T) {
		req := makeRequest(map[string]any{
			"workspace":       "test",
			"include_deleted": false,
		})
		result, err := h.HandleList(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		output := parseOutput(t, result)
		items := output["items"].([]any)
		if len(items) != 2 {
			t.Errorf("got %d items, want 2 (deleted excluded)", len(items))
		}
	})

	// Test include_deleted:true includes deleted
	t.Run("include_deleted:true includes deleted", func(t *testing.T) {
		req := makeRequest(map[string]any{
			"workspace":       "test",
			"include_deleted": true,
		})
		result, err := h.HandleList(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		output := parseOutput(t, result)
		items := output["items"].([]any)
		if len(items) != 3 {
			t.Errorf("got %d items, want 3 (deleted included)", len(items))
		}
	})

	// Test list never returns capsule_text (bloat rule)
	t.Run("list never returns capsule_text", func(t *testing.T) {
		req := makeRequest(map[string]any{"workspace": "test"})
		result, err := h.HandleList(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		output := parseOutput(t, result)
		items := output["items"].([]any)
		for i, item := range items {
			m := item.(map[string]any)
			if m["capsule_text"] != nil {
				t.Errorf("item[%d] has capsule_text, list should never include it", i)
			}
		}
	})
}

// TestHandleInventory tests the inventory handler with contract assertions.
func TestHandleInventory(t *testing.T) {
	database, cfg, cleanup := testSetup(t)
	defer cleanup()

	h := NewHandlers(database, cfg)
	ctx := context.Background()

	// Store 4 capsules across 2 workspaces, delete 1
	for _, ws := range []string{"ws1", "ws2"} {
		for _, name := range []string{"a", "b"} {
			req := makeRequest(map[string]any{
				"capsule_text": validCapsuleText(),
				"workspace":    ws,
				"name":         name,
			})
			if _, err := h.HandleStore(ctx, req); err != nil {
				t.Fatalf("setup store failed: %v", err)
			}
		}
	}
	deleteReq := makeRequest(map[string]any{"workspace": "ws1", "name": "b"})
	if _, err := h.HandleDelete(ctx, deleteReq); err != nil {
		t.Fatalf("setup delete failed: %v", err)
	}

	// Test pagination metadata
	t.Run("pagination metadata correct", func(t *testing.T) {
		req := makeRequest(map[string]any{
			"limit":  2,
			"offset": 1,
		})
		result, err := h.HandleInventory(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		output := parseOutput(t, result)
		pagination := output["pagination"].(map[string]any)

		if int(pagination["limit"].(float64)) != 2 {
			t.Errorf("pagination.limit = %v, want 2", pagination["limit"])
		}
		if int(pagination["offset"].(float64)) != 1 {
			t.Errorf("pagination.offset = %v, want 1", pagination["offset"])
		}
		if int(pagination["total"].(float64)) != 3 {
			t.Errorf("pagination.total = %v, want 3 (active only)", pagination["total"])
		}
	})

	// Test include_deleted:false (default) excludes deleted
	t.Run("include_deleted:false excludes deleted", func(t *testing.T) {
		req := makeRequest(map[string]any{"include_deleted": false})
		result, err := h.HandleInventory(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		output := parseOutput(t, result)
		items := output["items"].([]any)
		if len(items) != 3 {
			t.Errorf("got %d items, want 3 (deleted excluded)", len(items))
		}
	})

	// Test include_deleted:true includes deleted
	t.Run("include_deleted:true includes deleted", func(t *testing.T) {
		req := makeRequest(map[string]any{"include_deleted": true})
		result, err := h.HandleInventory(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		output := parseOutput(t, result)
		items := output["items"].([]any)
		if len(items) != 4 {
			t.Errorf("got %d items, want 4 (deleted included)", len(items))
		}
	})

	// Test inventory never returns capsule_text (bloat rule)
	t.Run("inventory never returns capsule_text", func(t *testing.T) {
		req := makeRequest(map[string]any{})
		result, err := h.HandleInventory(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		output := parseOutput(t, result)
		items := output["items"].([]any)
		for i, item := range items {
			m := item.(map[string]any)
			if m["capsule_text"] != nil {
				t.Errorf("item[%d] has capsule_text, inventory should never include it", i)
			}
		}
	})

	// Test workspace filter
	t.Run("workspace filter", func(t *testing.T) {
		req := makeRequest(map[string]any{"workspace": "ws2"})
		result, err := h.HandleInventory(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		output := parseOutput(t, result)
		items := output["items"].([]any)
		if len(items) != 2 {
			t.Errorf("got %d items, want 2 (ws2 only)", len(items))
		}
	})
}

// TestHandleExportImport tests the export and import handlers.
func TestHandleExportImport(t *testing.T) {
	database, cfg, cleanup := testSetup(t)
	defer cleanup()

	h := NewHandlers(database, cfg)
	ctx := context.Background()

	// Store a capsule
	storeReq := makeRequest(map[string]any{
		"capsule_text": validCapsuleText(),
		"workspace":    "test",
		"name":         "export-test",
	})
	if _, err := h.HandleStore(ctx, storeReq); err != nil {
		t.Fatalf("setup store failed: %v", err)
	}

	// Export
	exportPath := filepath.Join(t.TempDir(), "export.jsonl")
	exportReq := makeRequest(map[string]any{
		"path": exportPath,
	})
	exportResult, err := h.HandleExport(ctx, exportReq)
	if err != nil {
		t.Fatalf("export handler returned error: %v", err)
	}
	if exportResult.IsError {
		t.Fatalf("export failed: %v", extractErrorMessage(exportResult))
	}

	// Verify export file exists
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Fatal("export file not created")
	}

	// Create new database for import test
	database2, cfg2, cleanup2 := testSetup(t)
	defer cleanup2()
	h2 := NewHandlers(database2, cfg2)

	// Import
	importReq := makeRequest(map[string]any{
		"path": exportPath,
		"mode": "error",
	})
	importResult, err := h2.HandleImport(ctx, importReq)
	if err != nil {
		t.Fatalf("import handler returned error: %v", err)
	}
	if importResult.IsError {
		t.Fatalf("import failed: %v", extractErrorMessage(importResult))
	}

	// Verify imported capsule exists
	fetchReq := makeRequest(map[string]any{
		"workspace": "test",
		"name":      "export-test",
	})
	fetchResult, _ := h2.HandleFetch(ctx, fetchReq)
	if fetchResult.IsError {
		t.Error("imported capsule not found")
	}
}

// TestHandlePurge tests the purge handler.
func TestHandlePurge(t *testing.T) {
	database, cfg, cleanup := testSetup(t)
	defer cleanup()

	h := NewHandlers(database, cfg)
	ctx := context.Background()

	// Store and delete a capsule
	storeReq := makeRequest(map[string]any{
		"capsule_text": validCapsuleText(),
		"workspace":    "test",
		"name":         "purge-test",
	})
	if _, err := h.HandleStore(ctx, storeReq); err != nil {
		t.Fatalf("setup store failed: %v", err)
	}

	deleteReq := makeRequest(map[string]any{
		"workspace": "test",
		"name":      "purge-test",
	})
	if _, err := h.HandleDelete(ctx, deleteReq); err != nil {
		t.Fatalf("setup delete failed: %v", err)
	}

	// Purge
	purgeReq := makeRequest(map[string]any{})
	purgeResult, err := h.HandlePurge(ctx, purgeReq)
	if err != nil {
		t.Fatalf("purge handler returned error: %v", err)
	}
	if purgeResult.IsError {
		t.Fatalf("purge failed: %v", extractErrorMessage(purgeResult))
	}

	// Verify capsule is gone even with include_deleted
	fetchReq := makeRequest(map[string]any{
		"workspace":       "test",
		"name":            "purge-test",
		"include_deleted": true,
	})
	fetchResult, _ := h.HandleFetch(ctx, fetchReq)
	if !fetchResult.IsError {
		t.Error("purged capsule should not be found")
	}
}

func TestServerRegistration(t *testing.T) {
	database, cfg, cleanup := testSetup(t)
	defer cleanup()

	s := NewServer(database, cfg, "test")
	tools := s.ListTools()
	if tools == nil {
		t.Fatal("expected tools to be registered, got nil")
	}

	expectedTools := []string{
		"store",
		"fetch",
		"fetch_many",
		"update",
		"delete",
		"latest",
		"list",
		"inventory",
		"export",
		"import",
		"purge",
		"compose",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("registered tool count = %d, want %d", len(tools), len(expectedTools))
	}

	for _, name := range expectedTools {
		if _, ok := tools[name]; !ok {
			t.Errorf("missing registered tool: %s", name)
		}
	}
}

func TestErrorResult_InternalDoesNotExposeDetails(t *testing.T) {
	r := errorResult(errors.NewInternal(fmt.Errorf("sql error: open /tmp/secret.db: permission denied")))
	if !r.IsError {
		t.Fatal("expected IsError=true")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(r.Content[0].(mcp.TextContent).Text), &payload); err != nil {
		t.Fatalf("failed to unmarshal error payload: %v", err)
	}
	errObj := payload["error"].(map[string]any)

	if errObj["code"] != string(errors.ErrInternal) {
		t.Fatalf("code=%v, want %v", errObj["code"], errors.ErrInternal)
	}
	if _, ok := errObj["details"]; ok {
		t.Fatal("expected INTERNAL errors to omit details")
	}
}

func TestErrorResult_WrappedErrorPreservesContext(t *testing.T) {
	// Simulate wrapped error like compose.go does: fmt.Errorf("items[%d]: %w", i, err)
	originalErr := errors.NewAmbiguousAddressing()
	wrappedErr := fmt.Errorf("items[2]: %w", originalErr)

	r := errorResult(wrappedErr)
	if !r.IsError {
		t.Fatal("expected IsError=true")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(r.Content[0].(mcp.TextContent).Text), &payload); err != nil {
		t.Fatalf("failed to unmarshal error payload: %v", err)
	}
	errObj := payload["error"].(map[string]any)

	// Should extract the correct code from wrapped error
	if errObj["code"] != string(errors.ErrAmbiguousAddressing) {
		t.Errorf("code=%v, want %v", errObj["code"], errors.ErrAmbiguousAddressing)
	}

	// Message should include the wrapper context "items[2]:"
	msg := errObj["message"].(string)
	if !strings.Contains(msg, "items[2]") {
		t.Errorf("message should contain wrapper context 'items[2]', got: %s", msg)
	}
}

func TestErrorResult_NonInternalIncludesDetails(t *testing.T) {
	r := errorResult(errors.NewNotFound("abc"))
	if !r.IsError {
		t.Fatal("expected IsError=true")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(r.Content[0].(mcp.TextContent).Text), &payload); err != nil {
		t.Fatalf("failed to unmarshal error payload: %v", err)
	}
	errObj := payload["error"].(map[string]any)

	if errObj["code"] != string(errors.ErrNotFound) {
		t.Fatalf("code=%v, want %v", errObj["code"], errors.ErrNotFound)
	}
	if _, ok := errObj["details"]; !ok {
		t.Fatal("expected non-INTERNAL errors to include details when present")
	}
}

// Helper functions

// parseOutput extracts and unmarshals the JSON output from an MCP result.
func parseOutput(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()
	if result.IsError {
		t.Fatalf("expected success, got error: %v", extractErrorMessage(result))
	}
	var output map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &output); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	return output
}

func assertErrorCode(t *testing.T, result *mcp.CallToolResult, expectedCode string) {
	t.Helper()

	if len(result.Content) == 0 {
		t.Errorf("no content in error result")
		return
	}

	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Errorf("content is not TextContent")
		return
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(text.Text), &payload); err != nil {
		t.Errorf("failed to unmarshal error payload: %v", err)
		return
	}

	errorObj, ok := payload["error"].(map[string]any)
	if !ok {
		t.Errorf("no error object in payload")
		return
	}

	code, ok := errorObj["code"].(string)
	if !ok {
		t.Errorf("no code in error object")
		return
	}

	if code != expectedCode {
		t.Errorf("got error code %q, want %q", code, expectedCode)
	}
}

func extractErrorMessage(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return "<no content>"
	}

	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		return "<not text content>"
	}

	return text.Text
}
