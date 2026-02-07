package web

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/errors"
	"github.com/hpungsan/moss/internal/ops"
)

// Handlers contains HTTP route handlers for the web UI.
type Handlers struct {
	db       *sql.DB
	cfg      *config.Config
	renderer *Renderer
}

// HandleList handles GET /capsules — list capsules in a workspace.
func (h *Handlers) HandleList(w http.ResponseWriter, r *http.Request) {
	workspace := r.URL.Query().Get("workspace")
	if workspace == "" {
		workspace = "default"
	}

	input := ops.ListInput{
		Workspace:      workspace,
		RunID:          ptrString(r.URL.Query().Get("run_id")),
		Phase:          ptrString(r.URL.Query().Get("phase")),
		Role:           ptrString(r.URL.Query().Get("role")),
		Limit:          parseIntParam(r, "limit", 20),
		Offset:         parseIntParam(r, "offset", 0),
		IncludeDeleted: parseBoolParam(r, "include_deleted"),
	}

	result, err := ops.List(r.Context(), h.db, input)
	if err != nil {
		h.renderer.renderError(w, r, err)
		return
	}

	h.renderer.renderPage(w, r, "list", ListPageData{
		PageData: PageData{
			Title:   "Capsules",
			Version: h.renderer.version,
			Nav:     "capsules",
		},
		Items:      result.Items,
		Pagination: result.Pagination,
		Workspace:  workspace,
		RunID:      r.URL.Query().Get("run_id"),
		Phase:      r.URL.Query().Get("phase"),
		Role:       r.URL.Query().Get("role"),
		Deleted:    input.IncludeDeleted,
	})
}

// HandleSearch handles GET /capsules/search — full-text search.
func (h *Handlers) HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	workspace := r.URL.Query().Get("workspace")
	tag := r.URL.Query().Get("tag")
	runID := r.URL.Query().Get("run_id")
	phase := r.URL.Query().Get("phase")
	role := r.URL.Query().Get("role")

	data := SearchPageData{
		PageData: PageData{
			Title:   "Search",
			Version: h.renderer.version,
			Nav:     "search",
		},
		Query:     query,
		Workspace: workspace,
		Tag:       tag,
		RunID:     runID,
		Phase:     phase,
		Role:      role,
		Deleted:   parseBoolParam(r, "include_deleted"),
		HasQuery:  query != "",
	}

	if query == "" {
		// If htmx targets #results (user cleared the search box), return just the results fragment
		if r.Header.Get("HX-Target") == "results" {
			h.renderer.renderBlock(w, http.StatusOK, "search", "search-results", data)
			return
		}
		h.renderer.renderPage(w, r, "search", data)
		return
	}

	input := ops.SearchInput{
		Query:          query,
		Workspace:      ptrString(workspace),
		Tag:            ptrString(tag),
		RunID:          ptrString(runID),
		Phase:          ptrString(phase),
		Role:           ptrString(role),
		Limit:          parseIntParam(r, "limit", 20),
		Offset:         parseIntParam(r, "offset", 0),
		IncludeDeleted: data.Deleted,
	}

	result, err := ops.Search(r.Context(), h.db, input)
	if err != nil {
		h.renderer.renderError(w, r, err)
		return
	}

	data.Items = result.Items
	data.Pagination = result.Pagination

	// If htmx targets #results, render only the results fragment
	if r.Header.Get("HX-Target") == "results" {
		h.renderer.renderBlock(w, http.StatusOK, "search", "search-results", data)
		return
	}

	h.renderer.renderPage(w, r, "search", data)
}

// HandleInventory handles GET /capsules/inventory — cross-workspace listing.
func (h *Handlers) HandleInventory(w http.ResponseWriter, r *http.Request) {
	workspace := r.URL.Query().Get("workspace")
	tag := r.URL.Query().Get("tag")
	namePrefix := r.URL.Query().Get("name_prefix")
	runID := r.URL.Query().Get("run_id")
	phase := r.URL.Query().Get("phase")
	role := r.URL.Query().Get("role")

	input := ops.InventoryInput{
		Workspace:      ptrString(workspace),
		Tag:            ptrString(tag),
		NamePrefix:     ptrString(namePrefix),
		RunID:          ptrString(runID),
		Phase:          ptrString(phase),
		Role:           ptrString(role),
		Limit:          parseIntParam(r, "limit", 100),
		Offset:         parseIntParam(r, "offset", 0),
		IncludeDeleted: parseBoolParam(r, "include_deleted"),
	}

	result, err := ops.Inventory(r.Context(), h.db, input)
	if err != nil {
		h.renderer.renderError(w, r, err)
		return
	}

	h.renderer.renderPage(w, r, "inventory", InventoryPageData{
		PageData: PageData{
			Title:   "Inventory",
			Version: h.renderer.version,
			Nav:     "inventory",
		},
		Items:      result.Items,
		Pagination: result.Pagination,
		Workspace:  workspace,
		Tag:        tag,
		NamePrefix: namePrefix,
		RunID:      runID,
		Phase:      phase,
		Role:       role,
		Deleted:    input.IncludeDeleted,
	})
}

// HandleDetail handles GET /capsules/{id} — view a single capsule.
func (h *Handlers) HandleDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.renderer.renderError(w, r, errors.NewInvalidRequest("capsule ID is required"))
		return
	}

	includeText := true
	input := ops.FetchInput{
		ID:             id,
		IncludeDeleted: parseBoolParam(r, "include_deleted"),
		IncludeText:    &includeText,
	}

	capsule, err := ops.Fetch(r.Context(), h.db, input)
	if err != nil {
		h.renderer.renderError(w, r, err)
		return
	}

	rendered := renderMarkdown(capsule.CapsuleText)

	h.renderer.renderPage(w, r, "detail", DetailPageData{
		PageData: PageData{
			Title:   displayName(capsule.Name, capsule.ID),
			Version: h.renderer.version,
			Nav:     "capsules",
		},
		Capsule:      capsule,
		RenderedHTML: rendered,
		DisplayName:  displayName(capsule.Name, capsule.ID),
	})
}

// HandleDelete handles DELETE /capsules/{id} — soft-delete a capsule.
func (h *Handlers) HandleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.renderer.renderError(w, r, errors.NewInvalidRequest("capsule ID is required"))
		return
	}

	result, err := ops.Delete(r.Context(), h.db, ops.DeleteInput{ID: id})
	if err != nil {
		h.renderer.renderError(w, r, err)
		return
	}

	// HTMX request: redirect via HX-Redirect header
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/capsules")
		w.WriteHeader(http.StatusOK)
		return
	}

	// JSON request
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		renderJSON(w, http.StatusOK, map[string]any{
			"deleted": result.Deleted,
			"id":      result.ID,
		})
		return
	}

	// Default: redirect
	http.Redirect(w, r, "/capsules", http.StatusFound)
}

// HandlePurge handles POST /capsules/purge — permanently delete soft-deleted capsules.
func (h *Handlers) HandlePurge(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderer.renderError(w, r, errors.NewInvalidRequest("invalid form data"))
		return
	}

	if r.FormValue("confirm") != "true" {
		h.renderer.renderError(w, r, errors.NewInvalidRequest("confirm parameter must be \"true\""))
		return
	}

	input := ops.PurgeInput{
		Workspace: ptrString(r.FormValue("workspace")),
	}

	if days := r.FormValue("older_than_days"); days != "" {
		d, err := strconv.Atoi(days)
		if err != nil {
			h.renderer.renderError(w, r, errors.NewInvalidRequest("older_than_days must be an integer"))
			return
		}
		input.OlderThanDays = &d
	}

	result, err := ops.Purge(r.Context(), h.db, input)
	if err != nil {
		h.renderer.renderError(w, r, err)
		return
	}

	// HTMX request: return HTML fragment
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<div class="purge-result">` + template.HTMLEscapeString(result.Message) + `</div>`))
		return
	}

	// JSON request
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		renderJSON(w, http.StatusOK, map[string]any{
			"purged":  result.Purged,
			"message": result.Message,
		})
		return
	}

	// Default: redirect
	http.Redirect(w, r, "/capsules?include_deleted=true", http.StatusFound)
}

// parseIntParam parses an integer query parameter with a default value.
func parseIntParam(r *http.Request, name string, defaultVal int) int {
	s := r.URL.Query().Get(name)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// parseBoolParam parses a boolean query parameter.
func parseBoolParam(r *http.Request, name string) bool {
	s := r.URL.Query().Get(name)
	return s == "true" || s == "1"
}

// ptrString returns a pointer to s if non-empty, nil otherwise.
func ptrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// displayName returns the capsule name if present, or a truncated ID.
func displayName(name *string, id string) string {
	if name != nil && *name != "" {
		return *name
	}
	if len(id) > 10 {
		return id[:10] + "..."
	}
	return id
}
