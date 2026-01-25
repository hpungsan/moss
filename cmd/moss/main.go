package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/mcp"
)

// Version is set via -ldflags at build time.
var Version = "dev"

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not determine home directory: %v\n", err)
		os.Exit(1)
	}

	baseDir := filepath.Join(homeDir, ".moss")

	database, err := db.Init(baseDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	cfg, err := config.Load(baseDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	// If subcommand provided, defer to CLI (Phase 6)
	// Otherwise, run MCP server
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		// CLI mode - placeholder for Phase 6
		fmt.Fprintf(os.Stderr, "CLI commands not yet implemented\n")
		os.Exit(1)
	}

	// MCP server mode (default)
	if err := mcp.Run(database, cfg, Version); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
