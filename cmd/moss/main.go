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
	"help": true,
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

// isHelpOrVersion returns true if the user is requesting help or version info.
func isHelpOrVersion() bool {
	if len(os.Args) < 2 {
		return false
	}
	arg := os.Args[1]
	return arg == "--help" || arg == "-h" || arg == "--version" || arg == "-v" || arg == "help"
}

// isTerminal returns true if stdin is a terminal (not piped).
func isTerminal() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// printBanner displays a friendly banner when run interactively without args.
func printBanner() {
	fmt.Println(`
   __  __  ___  ___ ___
  |  \/  |/ _ \/ __/ __|
  | |\/| | (_) \__ \__ \
  |_|  |_|\___/|___/___/

  Local context capsule store

  Usage: moss <command> [options]
         moss --help

  MCP server mode requires piped input.`)
}

func main() {
	// No args + interactive terminal → show banner and exit
	if len(os.Args) < 2 && isTerminal() {
		printBanner()
		return
	}

	// Handle --help/--version before DB init (no DB needed)
	if isHelpOrVersion() {
		app := newCLIApp(nil, nil)
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

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

	// CLI mode: known subcommand
	if isCLIMode() {
		app := newCLIApp(database, cfg)
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Unknown argument + terminal → show error (don't start MCP server)
	if len(os.Args) >= 2 && isTerminal() {
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n", os.Args[1])
		fmt.Fprintf(os.Stderr, "Run 'moss --help' for usage.\n")
		os.Exit(1)
	}

	// MCP server mode (default)
	if err := mcp.Run(database, cfg, Version); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
