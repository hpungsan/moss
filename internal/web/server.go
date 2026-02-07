package web

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hpungsan/moss/internal/config"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

// NewServer creates and configures the HTTP server for the Moss web UI.
func NewServer(db *sql.DB, cfg *config.Config, version, bind string, port int) *http.Server {
	// Create sub-FS for templates (strip "templates/" prefix)
	templateSub, err := fs.Sub(templateFS, "templates")
	if err != nil {
		log.Fatalf("failed to create template sub-FS: %v", err)
	}

	// Create sub-FS for static files (strip "static/" prefix)
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("failed to create static sub-FS: %v", err)
	}

	renderer := NewRenderer(templateSub, version)

	h := &Handlers{
		db:       db,
		cfg:      cfg,
		renderer: renderer,
	}

	mux := http.NewServeMux()

	// Routes using Go 1.22+ pattern syntax
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/capsules", http.StatusFound)
	})
	mux.HandleFunc("GET /capsules", h.HandleList)
	mux.HandleFunc("GET /capsules/search", h.HandleSearch)
	mux.HandleFunc("GET /capsules/inventory", h.HandleInventory)
	mux.HandleFunc("GET /capsules/{id}", h.HandleDetail)
	mux.HandleFunc("DELETE /capsules/{id}", h.HandleDelete)
	mux.HandleFunc("POST /capsules/purge", h.HandlePurge)

	// Static file server
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticSub)))

	// Wrap with security headers
	handler := securityHeaders(mux)

	return &http.Server{
		Addr:    fmt.Sprintf("%s:%d", bind, port),
		Handler: handler,
	}
}

// securityHeaders adds security-related HTTP headers to all responses.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

// Run starts the HTTP server and handles graceful shutdown on SIGINT/SIGTERM.
func Run(srv *http.Server) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	log.Printf("Moss UI running at http://%s", srv.Addr)

	if strings.Contains(srv.Addr, "0.0.0.0") || strings.Contains(srv.Addr, "::") {
		log.Printf("WARNING: Server is binding to all interfaces and may be accessible from the network")
	}

	select {
	case err := <-errCh:
		return err
	case <-sigCh:
		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}
