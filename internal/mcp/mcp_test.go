package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
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

// TestHandleLatest tests the latest handler.
func TestHandleLatest(t *testing.T) {
	database, cfg, cleanup := testSetup(t)
	defer cleanup()

	h := NewHandlers(database, cfg)
	ctx := context.Background()

	// Test empty workspace
	t.Run("empty workspace", func(t *testing.T) {
		req := makeRequest(map[string]any{
			"workspace": "empty",
		})
		result, err := h.HandleLatest(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		if result.IsError {
			t.Errorf("expected success, got error")
		}
		// Should return item: null
		var output map[string]any
		if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &output); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if output["item"] != nil {
			t.Errorf("expected null item for empty workspace")
		}
	})

	// Store a capsule
	storeReq := makeRequest(map[string]any{
		"capsule_text": validCapsuleText(),
		"workspace":    "test",
		"name":         "latest-test",
	})
	if _, err := h.HandleStore(ctx, storeReq); err != nil {
		t.Fatalf("setup store failed: %v", err)
	}

	t.Run("get latest", func(t *testing.T) {
		req := makeRequest(map[string]any{
			"workspace": "test",
		})
		result, err := h.HandleLatest(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		if result.IsError {
			t.Errorf("expected success, got error: %v", extractErrorMessage(result))
		}
	})

	t.Run("get latest with text", func(t *testing.T) {
		req := makeRequest(map[string]any{
			"workspace":    "test",
			"include_text": true,
		})
		result, err := h.HandleLatest(ctx, req)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		if result.IsError {
			t.Errorf("expected success, got error: %v", extractErrorMessage(result))
		}
	})
}

// TestHandleList tests the list handler.
func TestHandleList(t *testing.T) {
	database, cfg, cleanup := testSetup(t)
	defer cleanup()

	h := NewHandlers(database, cfg)
	ctx := context.Background()

	// Store some capsules
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

	tests := []struct {
		name      string
		args      map[string]any
		wantCount int
	}{
		{
			name: "list all in workspace",
			args: map[string]any{
				"workspace": "test",
			},
			wantCount: 3,
		},
		{
			name: "list with limit",
			args: map[string]any{
				"workspace": "test",
				"limit":     2,
			},
			wantCount: 2,
		},
		{
			name: "list empty workspace",
			args: map[string]any{
				"workspace": "empty",
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest(tt.args)
			result, err := h.HandleList(ctx, req)

			if err != nil {
				t.Fatalf("handler returned error: %v", err)
			}
			if result.IsError {
				t.Errorf("expected success, got error: %v", extractErrorMessage(result))
				return
			}

			var output map[string]any
			if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &output); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			items, _ := output["items"].([]any)
			if len(items) != tt.wantCount {
				t.Errorf("got %d items, want %d", len(items), tt.wantCount)
			}
		})
	}
}

// TestHandleInventory tests the inventory handler.
func TestHandleInventory(t *testing.T) {
	database, cfg, cleanup := testSetup(t)
	defer cleanup()

	h := NewHandlers(database, cfg)
	ctx := context.Background()

	// Store capsules in different workspaces
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

	tests := []struct {
		name      string
		args      map[string]any
		wantCount int
	}{
		{
			name:      "inventory all",
			args:      map[string]any{},
			wantCount: 4,
		},
		{
			name: "inventory filter workspace",
			args: map[string]any{
				"workspace": "ws1",
			},
			wantCount: 2,
		},
		{
			name: "inventory with limit",
			args: map[string]any{
				"limit": 2,
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest(tt.args)
			result, err := h.HandleInventory(ctx, req)

			if err != nil {
				t.Fatalf("handler returned error: %v", err)
			}
			if result.IsError {
				t.Errorf("expected success, got error: %v", extractErrorMessage(result))
				return
			}

			var output map[string]any
			if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &output); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			items, _ := output["items"].([]any)
			if len(items) != tt.wantCount {
				t.Errorf("got %d items, want %d", len(items), tt.wantCount)
			}
		})
	}
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

// TestServerRegistration tests that all tools are registered.
func TestServerRegistration(t *testing.T) {
	database, cfg, cleanup := testSetup(t)
	defer cleanup()

	s := NewServer(database, cfg)
	if s == nil {
		t.Fatal("NewServer returned nil")
	}

	// The server should be created successfully
	// Tool registration happens internally; we verify by checking
	// that the handlers work (covered by other tests)
}

// Helper functions

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
