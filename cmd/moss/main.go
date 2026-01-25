package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/mcp"
)

// Version is set via -ldflags at build time.
var Version = "dev"

// cliCommands contains known CLI subcommands.
var cliCommands = map[string]bool{
	"store": true, "fetch": true, "update": true, "delete": true,
	"list": true, "inventory": true, "latest": true,
	"export": true, "import": true, "purge": true,
	"help": true, "version": true,
}

// isCLIMode determines if we should run CLI vs MCP server.
func isCLIMode() bool {
	if len(os.Args) < 2 {
		return false // No args → MCP server
	}
	arg := os.Args[1]
	// Known subcommand → CLI
	if cliCommands[arg] {
		return true
	}
	// --help or --version → CLI
	if arg == "--help" || arg == "-h" || arg == "--version" || arg == "-v" {
		return true
	}
	return false // Default → MCP server
}

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

	// CLI mode: known subcommand or --help/--version
	if isCLIMode() {
		app := newCLIApp(database, cfg)
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// MCP server mode (default)
	if err := mcp.Run(database, cfg, Version); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
