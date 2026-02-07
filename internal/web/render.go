package web

import (
	"bytes"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/yuin/goldmark"

	"github.com/hpungsan/moss/internal/errors"
	"github.com/hpungsan/moss/internal/ops"
)

// PageData contains common fields used across all page templates.
type PageData struct {
	Title   string
	Version string
	Nav     string // active nav item: "capsules", "inventory", "search"
}

// ListPageData is the template data for the capsule list page.
type ListPageData struct {
	PageData
	Items      []ops.SummaryItem
	Pagination ops.Pagination
	Workspace  string
	RunID      string
	Phase      string
	Role       string
	Deleted    bool
}

// DetailPageData is the template data for the capsule detail page.
type DetailPageData struct {
	PageData
	Capsule      *ops.FetchOutput
	RenderedHTML template.HTML
	DisplayName  string
}

// SearchPageData is the template data for the search page.
type SearchPageData struct {
	PageData
	Query      string
	Items      []ops.SearchResultItem
	Pagination ops.Pagination
	Workspace  string
	Tag        string
	RunID      string
	Phase      string
	Role       string
	Deleted    bool
	HasQuery   bool
}

// InventoryPageData is the template data for the inventory page.
type InventoryPageData struct {
	PageData
	Items      []ops.SummaryItem
	Pagination ops.Pagination
	Workspace  string
	Tag        string
	NamePrefix string
	RunID      string
	Phase      string
	Role       string
	Deleted    bool
}

// ErrorPageData is the template data for the error page.
type ErrorPageData struct {
	PageData
	StatusCode int
	Message    string
}

// PurgeResultData is the template data for purge results.
type PurgeResultData struct {
	PageData
	Purged  int
	Message string
}

// Renderer manages template parsing and rendering.
type Renderer struct {
	templates map[string]*template.Template
	version   string
}

// NewRenderer creates a Renderer by parsing templates from the given FS.
func NewRenderer(templateFS fs.FS, version string) *Renderer {
	funcMap := template.FuncMap{
		"add":         func(a, b int) int { return a + b },
		"sub":         func(a, b int) int { return a - b },
		"formatTime":  formatTime,
		"formatChars": formatChars,
		"safeHTML":    func(s string) template.HTML { return template.HTML(s) },
		"deref":       deref,
		"hasValue":    hasValue,
	}

	// Parse layout as the base template
	layoutTmpl := template.Must(template.New("layout").Funcs(funcMap).ParseFS(templateFS, "layout.html"))

	pages := map[string]string{
		"list":      "list.html",
		"detail":    "detail.html",
		"search":    "search.html",
		"inventory": "inventory.html",
		"error":     "error.html",
	}

	templates := make(map[string]*template.Template, len(pages))
	for name, file := range pages {
		t := template.Must(layoutTmpl.Clone())
		template.Must(t.ParseFS(templateFS, file))
		templates[name] = t
	}

	return &Renderer{
		templates: templates,
		version:   version,
	}
}

// renderPage renders a named page template with the given data and HTTP 200 status.
func (r *Renderer) renderPage(w http.ResponseWriter, req *http.Request, name string, data any) {
	r.renderPageStatus(w, req, http.StatusOK, name, data)
}

// renderPageStatus renders a named page template with the given data and HTTP status code.
// For HTMX requests, only the "content" block is rendered to avoid duplicating the layout.
func (r *Renderer) renderPageStatus(w http.ResponseWriter, req *http.Request, status int, name string, data any) {
	t, ok := r.templates[name]
	if !ok {
		log.Printf("template %q not found", name)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	block := "layout"
	if req != nil && req.Header.Get("HX-Request") == "true" {
		block = "content"
	}

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, block, data); err != nil {
		log.Printf("template execution error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// renderBlock renders a specific named block from a page template.
// Used for htmx partial swaps that target a sub-section of the page.
func (r *Renderer) renderBlock(w http.ResponseWriter, status int, page, block string, data any) {
	t, ok := r.templates[page]
	if !ok {
		log.Printf("template %q not found", page)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, block, data); err != nil {
		log.Printf("template block %q execution error: %v", block, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// renderError renders an error response with content negotiation.
func (r *Renderer) renderError(w http.ResponseWriter, req *http.Request, err error) {
	var mErr *errors.MossError
	if !stderrors.As(err, &mErr) {
		mErr = errors.NewInternal(err)
	}

	status := mErr.Status
	message := mErr.Message

	// HTMX request: return HTML fragment
	if req.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		fmt.Fprintf(w, `<div class="error-message">%s</div>`, template.HTMLEscapeString(message))
		return
	}

	// JSON request
	if strings.Contains(req.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    string(mErr.Code),
				"message": message,
				"status":  status,
			},
		})
		return
	}

	// Full error page
	r.renderPageStatus(w, req, status, "error", ErrorPageData{
		PageData: PageData{
			Title:   fmt.Sprintf("Error %d", status),
			Version: r.version,
		},
		StatusCode: status,
		Message:    message,
	})
}

// renderJSON writes a JSON response.
func renderJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// renderMarkdown converts markdown text to HTML using goldmark.
func renderMarkdown(md string) template.HTML {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(md))
	}
	return template.HTML(buf.String())
}

// formatTime formats a Unix timestamp as "2006-01-02 15:04" UTC.
func formatTime(unix int64) string {
	return time.Unix(unix, 0).UTC().Format("2006-01-02 15:04")
}

// formatChars formats an integer with comma thousands separators.
func formatChars(n int) string {
	if n < 0 {
		return "-" + formatChars(-n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var result strings.Builder
	remainder := len(s) % 3
	if remainder > 0 {
		result.WriteString(s[:remainder])
	}
	for i := remainder; i < len(s); i += 3 {
		if result.Len() > 0 {
			result.WriteByte(',')
		}
		result.WriteString(s[i : i+3])
	}
	return result.String()
}

// deref dereferences a pointer, returning the zero value if nil.
// Supports *string and *int64 (the pointer types used in templates).
func deref(v any) any {
	if v == nil {
		return ""
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return reflect.Zero(rv.Type().Elem()).Interface()
		}
		return rv.Elem().Interface()
	}
	return v
}

// hasValue checks if a pointer value is non-nil.
func hasValue(v any) bool {
	if v == nil {
		return false
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		return !rv.IsNil()
	}
	return true
}
