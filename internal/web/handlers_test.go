package web

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/ops"
)

const validCapsuleText = `## Objective
Build a user authentication system.

## Current status
Database schema is complete.

## Decisions
Using JWT for tokens.

## Next actions
Implement login endpoint.

## Key locations
cmd/auth/main.go

## Open questions
Should we support OAuth?
`

func stringPtr(s string) *string { return &s }

func setupTest(t *testing.T) *Handlers {
	t.Helper()
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	cfg := config.DefaultConfig()

	templateSub, err := fs.Sub(templateFS, "templates")
	if err != nil {
		t.Fatalf("template sub-FS: %v", err)
	}
	renderer := NewRenderer(templateSub, "test")

	return &Handlers{
		db:       database,
		cfg:      cfg,
		renderer: renderer,
	}
}

// seedCapsule stores a capsule and returns its ID.
func seedCapsule(t *testing.T, h *Handlers, name, workspace string) string {
	t.Helper()
	input := ops.StoreInput{
		Workspace:   workspace,
		Name:        stringPtr(name),
		CapsuleText: validCapsuleText,
		Tags:        []string{"test"},
	}
	out, err := ops.Store(context.Background(), h.db, h.cfg, input)
	if err != nil {
		t.Fatalf("seed capsule %q: %v", name, err)
	}
	return out.ID
}

// --- HandleList ---

func TestHandleList_Default(t *testing.T) {
	h := setupTest(t)
	seedCapsule(t, h, "alpha", "default")

	req := httptest.NewRequest("GET", "/capsules", nil)
	rec := httptest.NewRecorder()
	h.HandleList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "alpha") {
		t.Error("expected capsule name 'alpha' in response")
	}
	if !strings.Contains(body, "Capsules") {
		t.Error("expected page title 'Capsules' in response")
	}
}

func TestHandleList_WithWorkspaceFilter(t *testing.T) {
	h := setupTest(t)
	seedCapsule(t, h, "in-ws", "myws")
	seedCapsule(t, h, "other", "default")

	req := httptest.NewRequest("GET", "/capsules?workspace=myws", nil)
	rec := httptest.NewRecorder()
	h.HandleList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "in-ws") {
		t.Error("expected capsule 'in-ws' in filtered results")
	}
	if strings.Contains(body, ">other<") {
		t.Error("did not expect capsule 'other' in filtered results")
	}
}

func TestHandleList_Empty(t *testing.T) {
	h := setupTest(t)

	req := httptest.NewRequest("GET", "/capsules", nil)
	rec := httptest.NewRecorder()
	h.HandleList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "No capsules found") {
		t.Error("expected empty state message")
	}
}

func TestHandleList_HtmxReturnsContentOnly(t *testing.T) {
	h := setupTest(t)
	seedCapsule(t, h, "htmx-test", "default")

	req := httptest.NewRequest("GET", "/capsules", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.HandleList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	// Htmx response should not contain the full layout shell
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("htmx response should not contain full layout")
	}
	if !strings.Contains(body, "htmx-test") {
		t.Error("htmx response should contain capsule data")
	}
}

func TestHandleList_InvalidLimitFallsBack(t *testing.T) {
	h := setupTest(t)

	req := httptest.NewRequest("GET", "/capsules?limit=notanumber&offset=bad", nil)
	rec := httptest.NewRecorder()
	h.HandleList(rec, req)

	// Should not error — falls back to defaults
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

// --- HandleSearch ---

func TestHandleSearch_EmptyQuery(t *testing.T) {
	h := setupTest(t)

	req := httptest.NewRequest("GET", "/capsules/search", nil)
	rec := httptest.NewRecorder()
	h.HandleSearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Enter a search query") {
		t.Error("expected empty search prompt")
	}
}

func TestHandleSearch_WithQuery(t *testing.T) {
	h := setupTest(t)
	seedCapsule(t, h, "auth-capsule", "default")

	req := httptest.NewRequest("GET", "/capsules/search?q=authentication", nil)
	rec := httptest.NewRecorder()
	h.HandleSearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "auth-capsule") {
		t.Error("expected search result with capsule name")
	}
}

func TestHandleSearch_NoResults(t *testing.T) {
	h := setupTest(t)

	req := httptest.NewRequest("GET", "/capsules/search?q=zzzznonexistent", nil)
	rec := httptest.NewRecorder()
	h.HandleSearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "No results") {
		t.Error("expected 'No results' message")
	}
}

func TestHandleSearch_HtmxTargetResults_ReturnsFragment(t *testing.T) {
	h := setupTest(t)
	seedCapsule(t, h, "frag-test", "default")

	req := httptest.NewRequest("GET", "/capsules/search?q=authentication", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "results")
	rec := httptest.NewRecorder()
	h.HandleSearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	// Should not contain the search form (only the results fragment)
	if strings.Contains(body, "<h1>Search</h1>") {
		t.Error("results fragment should not contain search page header")
	}
	if strings.Contains(body, "search-form") {
		t.Error("results fragment should not contain the search form")
	}
	if !strings.Contains(body, "frag-test") {
		t.Error("results fragment should contain search result")
	}
}

func TestHandleSearch_HtmxTargetResults_EmptyQuery(t *testing.T) {
	h := setupTest(t)

	req := httptest.NewRequest("GET", "/capsules/search", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "results")
	rec := httptest.NewRecorder()
	h.HandleSearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "search-form") {
		t.Error("results fragment should not contain search form")
	}
	if !strings.Contains(body, "Enter a search query") {
		t.Error("expected empty search prompt in results fragment")
	}
}

func TestHandleSearch_HtmxReturnsContentOnly(t *testing.T) {
	h := setupTest(t)

	req := httptest.NewRequest("GET", "/capsules/search?q=test", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.HandleSearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "<!DOCTYPE html>") {
		t.Error("htmx response should not contain full layout")
	}
}

// --- HandleInventory ---

func TestHandleInventory_CrossWorkspace(t *testing.T) {
	h := setupTest(t)
	seedCapsule(t, h, "cap-a", "workspace-one")
	seedCapsule(t, h, "cap-b", "workspace-two")

	req := httptest.NewRequest("GET", "/capsules/inventory", nil)
	rec := httptest.NewRecorder()
	h.HandleInventory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "cap-a") {
		t.Error("expected capsule from workspace-one")
	}
	if !strings.Contains(body, "cap-b") {
		t.Error("expected capsule from workspace-two")
	}
	if !strings.Contains(body, "workspace-one") {
		t.Error("expected workspace-one badge")
	}
}

func TestHandleInventory_WorkspaceFilter(t *testing.T) {
	h := setupTest(t)
	seedCapsule(t, h, "inv-target", "target-ws")
	seedCapsule(t, h, "inv-other", "other-ws")

	req := httptest.NewRequest("GET", "/capsules/inventory?workspace=target-ws", nil)
	rec := httptest.NewRecorder()
	h.HandleInventory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "inv-target") {
		t.Error("expected filtered capsule")
	}
}

func TestHandleInventory_Empty(t *testing.T) {
	h := setupTest(t)

	req := httptest.NewRequest("GET", "/capsules/inventory", nil)
	rec := httptest.NewRecorder()
	h.HandleInventory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "No capsules found") {
		t.Error("expected empty state message")
	}
}

// --- HandleDetail ---

func TestHandleDetail_Found(t *testing.T) {
	h := setupTest(t)
	id := seedCapsule(t, h, "detail-cap", "default")

	req := httptest.NewRequest("GET", "/capsules/"+id, nil)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	h.HandleDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "detail-cap") {
		t.Error("expected capsule name in detail page")
	}
	// Check rendered markdown is present
	if !strings.Contains(body, "Objective") {
		t.Error("expected rendered markdown content")
	}
	// Check metadata sidebar
	if !strings.Contains(body, "Metadata") {
		t.Error("expected metadata section")
	}
	// Check raw text toggle
	if !strings.Contains(body, "Raw capsule text") {
		t.Error("expected raw text toggle")
	}
}

func TestHandleDetail_NotFound(t *testing.T) {
	h := setupTest(t)

	req := httptest.NewRequest("GET", "/capsules/NONEXISTENT", nil)
	req.SetPathValue("id", "NONEXISTENT")
	rec := httptest.NewRecorder()
	h.HandleDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandleDetail_EmptyID(t *testing.T) {
	h := setupTest(t)

	req := httptest.NewRequest("GET", "/capsules/", nil)
	req.SetPathValue("id", "")
	rec := httptest.NewRecorder()
	h.HandleDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// --- HandleDelete ---

func TestHandleDelete_HtmxRequest(t *testing.T) {
	h := setupTest(t)
	id := seedCapsule(t, h, "del-htmx", "default")

	req := httptest.NewRequest("DELETE", "/capsules/"+id, nil)
	req.SetPathValue("id", id)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.HandleDelete(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("HX-Redirect"); got != "/capsules" {
		t.Errorf("HX-Redirect = %q, want /capsules", got)
	}
}

func TestHandleDelete_JSONRequest(t *testing.T) {
	h := setupTest(t)
	id := seedCapsule(t, h, "del-json", "default")

	req := httptest.NewRequest("DELETE", "/capsules/"+id, nil)
	req.SetPathValue("id", id)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	h.HandleDelete(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if resp["deleted"] != true {
		t.Errorf("deleted = %v, want true", resp["deleted"])
	}
	if resp["id"] != id {
		t.Errorf("id = %v, want %s", resp["id"], id)
	}
}

func TestHandleDelete_JSONRequest_MixedAccept(t *testing.T) {
	h := setupTest(t)
	id := seedCapsule(t, h, "del-mixed", "default")

	req := httptest.NewRequest("DELETE", "/capsules/"+id, nil)
	req.SetPathValue("id", id)
	req.Header.Set("Accept", "text/html, application/json")
	rec := httptest.NewRecorder()
	h.HandleDelete(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestHandleList_DeletedCapsuleLinks(t *testing.T) {
	h := setupTest(t)
	id := seedCapsule(t, h, "del-link", "default")
	// Soft-delete the capsule
	_, err := ops.Delete(context.Background(), h.db, ops.DeleteInput{ID: id})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	req := httptest.NewRequest("GET", "/capsules?include_deleted=true", nil)
	rec := httptest.NewRecorder()
	h.HandleList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	// The link to the deleted capsule should include include_deleted=true
	expected := "/capsules/" + id + "?include_deleted=true"
	if !strings.Contains(body, expected) {
		t.Errorf("expected link %q in response body", expected)
	}
}

func TestHandleDelete_DefaultRedirect(t *testing.T) {
	h := setupTest(t)
	id := seedCapsule(t, h, "del-redirect", "default")

	req := httptest.NewRequest("DELETE", "/capsules/"+id, nil)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	h.HandleDelete(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/capsules" {
		t.Errorf("Location = %q, want /capsules", loc)
	}
}

func TestHandleDelete_NotFound(t *testing.T) {
	h := setupTest(t)

	req := httptest.NewRequest("DELETE", "/capsules/NONEXISTENT", nil)
	req.SetPathValue("id", "NONEXISTENT")
	rec := httptest.NewRecorder()
	h.HandleDelete(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandleDelete_NotFound_JSON(t *testing.T) {
	h := setupTest(t)

	req := httptest.NewRequest("DELETE", "/capsules/NONEXISTENT", nil)
	req.SetPathValue("id", "NONEXISTENT")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	h.HandleDelete(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error object in JSON response")
	}
	if errObj["status"] != float64(404) {
		t.Errorf("error.status = %v, want 404", errObj["status"])
	}
}

func TestHandleDelete_EmptyID(t *testing.T) {
	h := setupTest(t)

	req := httptest.NewRequest("DELETE", "/capsules/", nil)
	req.SetPathValue("id", "")
	rec := httptest.NewRecorder()
	h.HandleDelete(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// --- HandlePurge ---

func TestHandlePurge_MissingConfirm(t *testing.T) {
	h := setupTest(t)

	form := url.Values{}
	req := httptest.NewRequest("POST", "/capsules/purge", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.HandlePurge(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandlePurge_ConfirmFalse(t *testing.T) {
	h := setupTest(t)

	form := url.Values{"confirm": {"false"}}
	req := httptest.NewRequest("POST", "/capsules/purge", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.HandlePurge(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandlePurge_DefaultRedirect(t *testing.T) {
	h := setupTest(t)

	form := url.Values{"confirm": {"true"}}
	req := httptest.NewRequest("POST", "/capsules/purge", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.HandlePurge(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/capsules?include_deleted=true" {
		t.Errorf("Location = %q, want /capsules?include_deleted=true", loc)
	}
}

func TestHandlePurge_JSONResponse(t *testing.T) {
	h := setupTest(t)
	// Seed and delete a capsule so purge has something to work on
	id := seedCapsule(t, h, "purge-target", "default")
	_, err := ops.Delete(context.Background(), h.db, ops.DeleteInput{ID: id})
	if err != nil {
		t.Fatalf("delete for purge setup: %v", err)
	}

	form := url.Values{"confirm": {"true"}}
	req := httptest.NewRequest("POST", "/capsules/purge", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	h.HandlePurge(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if resp["purged"] != float64(1) {
		t.Errorf("purged = %v, want 1", resp["purged"])
	}
}

func TestHandlePurge_HtmxResponse(t *testing.T) {
	h := setupTest(t)

	form := url.Values{"confirm": {"true"}}
	req := httptest.NewRequest("POST", "/capsules/purge", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.HandlePurge(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "purge-result") {
		t.Error("expected purge-result div in htmx response")
	}
}

func TestHandlePurge_InvalidOlderThanDays(t *testing.T) {
	h := setupTest(t)

	form := url.Values{"confirm": {"true"}, "older_than_days": {"notanumber"}}
	req := httptest.NewRequest("POST", "/capsules/purge", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.HandlePurge(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// --- Error rendering ---

func TestErrorRendering_HtmxFragment(t *testing.T) {
	h := setupTest(t)

	req := httptest.NewRequest("GET", "/capsules/NONEXISTENT", nil)
	req.SetPathValue("id", "NONEXISTENT")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.HandleDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "error-message") {
		t.Error("expected error-message div in htmx error response")
	}
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("htmx error should not contain full layout")
	}
}

func TestErrorRendering_JSONError(t *testing.T) {
	h := setupTest(t)

	req := httptest.NewRequest("GET", "/capsules/NONEXISTENT", nil)
	req.SetPathValue("id", "NONEXISTENT")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	h.HandleDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error object in JSON response")
	}
	if errObj["status"] != float64(404) {
		t.Errorf("error.status = %v, want 404", errObj["status"])
	}
}

func TestErrorRendering_FullErrorPage(t *testing.T) {
	h := setupTest(t)

	req := httptest.NewRequest("GET", "/capsules/NONEXISTENT", nil)
	req.SetPathValue("id", "NONEXISTENT")
	rec := httptest.NewRecorder()
	h.HandleDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("full error page should contain layout")
	}
	if !strings.Contains(body, "404") {
		t.Error("error page should show status code")
	}
}

// --- Helper functions ---

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		query    string
		name     string
		def      int
		expected int
	}{
		{"", "limit", 20, 20},
		{"limit=50", "limit", 20, 50},
		{"limit=bad", "limit", 20, 20},
		{"offset=10", "offset", 0, 10},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/?"+tt.query, nil)
		got := parseIntParam(req, tt.name, tt.def)
		if got != tt.expected {
			t.Errorf("parseIntParam(%q, %q, %d) = %d, want %d", tt.query, tt.name, tt.def, got, tt.expected)
		}
	}
}

func TestParseBoolParam(t *testing.T) {
	tests := []struct {
		query    string
		name     string
		expected bool
	}{
		{"", "include_deleted", false},
		{"include_deleted=true", "include_deleted", true},
		{"include_deleted=1", "include_deleted", true},
		{"include_deleted=false", "include_deleted", false},
		{"include_deleted=yes", "include_deleted", false},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/?"+tt.query, nil)
		got := parseBoolParam(req, tt.name)
		if got != tt.expected {
			t.Errorf("parseBoolParam(%q, %q) = %v, want %v", tt.query, tt.name, got, tt.expected)
		}
	}
}

func TestDisplayName(t *testing.T) {
	tests := []struct {
		name     *string
		id       string
		expected string
	}{
		{stringPtr("myname"), "01ABCDEFGHIJK", "myname"},
		{nil, "01ABCDEFGHIJK", "01ABCDEFGH..."},
		{nil, "SHORT", "SHORT"},
		{stringPtr(""), "01ABCDEFGHIJK", "01ABCDEFGH..."},
	}
	for _, tt := range tests {
		got := displayName(tt.name, tt.id)
		if got != tt.expected {
			t.Errorf("displayName(%v, %q) = %q, want %q", tt.name, tt.id, got, tt.expected)
		}
	}
}

func TestPtrString(t *testing.T) {
	if got := ptrString(""); got != nil {
		t.Error("ptrString(\"\") should return nil")
	}
	if got := ptrString("hello"); got == nil || *got != "hello" {
		t.Error("ptrString(\"hello\") should return pointer to \"hello\"")
	}
}
