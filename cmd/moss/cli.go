package main

import (
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/errors"
	"github.com/hpungsan/moss/internal/mcp"
	"github.com/hpungsan/moss/internal/ops"
)

// newCLIApp creates the CLI application with all commands.
func newCLIApp(db *sql.DB, cfg *config.Config) *cli.App {
	app := &cli.App{
		Name:    "moss",
		Usage:   "Local context capsule store",
		Version: Version,
		Commands: []*cli.Command{
			storeCmd(db, cfg),
			fetchCmd(db, cfg),
			updateCmd(db, cfg),
			deleteCmd(db),
			listCmd(db),
			inventoryCmd(db),
			latestCmd(db),
			exportCmd(db, cfg),
			importCmd(db, cfg),
			purgeCmd(db),
			toolsCmd(cfg),
		},
	}
	// Disable default exit error handler to allow proper error return in tests
	app.ExitErrHandler = func(_ *cli.Context, _ error) {}
	return app
}

// storeCmd creates the store command.
func storeCmd(db *sql.DB, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "store",
		Usage: "Store a new capsule (reads capsule_text from stdin)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "workspace", Aliases: []string{"w"}, Value: "default", Usage: "Workspace name"},
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "Capsule name (optional)"},
			&cli.StringFlag{Name: "title", Aliases: []string{"t"}, Usage: "Capsule title (defaults to name)"},
			&cli.StringFlag{Name: "tags", Usage: "Comma-separated tags"},
			&cli.StringFlag{Name: "mode", Aliases: []string{"m"}, Value: "error", Usage: "Collision mode: error|replace"},
			&cli.BoolFlag{Name: "allow-thin", Usage: "Allow capsules without all required sections"},
		},
		Action: func(c *cli.Context) error {
			// Require stdin input
			if !stdinHasData() {
				return outputError(errors.NewInvalidRequest("capsule_text must be piped via stdin"))
			}

			capsuleText, err := readStdin(cfg.CapsuleMaxChars)
			if err != nil {
				return outputError(errors.NewInvalidRequest(err.Error()))
			}
			if capsuleText == "" {
				return outputError(errors.NewInvalidRequest("capsule_text is required"))
			}

			input := ops.StoreInput{
				Workspace:   c.String("workspace"),
				CapsuleText: capsuleText,
				Mode:        ops.StoreMode(c.String("mode")),
				AllowThin:   c.Bool("allow-thin"),
			}

			if name := c.String("name"); name != "" {
				input.Name = &name
			}
			if title := c.String("title"); title != "" {
				input.Title = &title
			}
			if tags := c.String("tags"); tags != "" {
				input.Tags = parseTags(tags)
			}

			output, err := ops.Store(c.Context, db, cfg, input)
			if err != nil {
				return outputError(err)
			}

			return outputJSON(output)
		},
	}
}

// fetchCmd creates the fetch command.
func fetchCmd(db *sql.DB, _ *config.Config) *cli.Command {
	return &cli.Command{
		Name:      "fetch",
		Usage:     "Fetch a capsule by ID or name",
		ArgsUsage: "[id]",
		Flags: append(addressingFlags(),
			&cli.BoolFlag{Name: "include-deleted", Usage: "Include soft-deleted capsules"},
			&cli.BoolFlag{Name: "no-text", Usage: "Exclude capsule_text from output"},
		),
		Action: func(c *cli.Context) error {
			addr, err := parseAddressing(c)
			if err != nil {
				return outputError(err)
			}

			input := ops.FetchInput{
				ID:             addr.ID,
				Workspace:      addr.Workspace,
				Name:           addr.Name,
				IncludeDeleted: c.Bool("include-deleted"),
			}

			if c.Bool("no-text") {
				includeText := false
				input.IncludeText = &includeText
			}

			output, err := ops.Fetch(c.Context, db, input)
			if err != nil {
				return outputError(err)
			}

			return outputJSON(output)
		},
	}
}

// updateCmd creates the update command.
func updateCmd(db *sql.DB, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "Update an existing capsule (optionally reads capsule_text from stdin)",
		ArgsUsage: "[id]",
		Flags: append(addressingFlags(),
			&cli.StringFlag{Name: "title", Aliases: []string{"t"}, Usage: "New title"},
			&cli.StringFlag{Name: "tags", Usage: "New comma-separated tags"},
			&cli.BoolFlag{Name: "allow-thin", Usage: "Allow capsules without all required sections"},
		),
		Action: func(c *cli.Context) error {
			addr, err := parseAddressing(c)
			if err != nil {
				return outputError(err)
			}

			input := ops.UpdateInput{
				ID:        addr.ID,
				Workspace: addr.Workspace,
				Name:      addr.Name,
				AllowThin: c.Bool("allow-thin"),
			}

			// Read capsule_text from stdin if piped
			if stdinHasData() {
				text, err := readStdin(cfg.CapsuleMaxChars)
				if err != nil {
					return outputError(errors.NewInvalidRequest(err.Error()))
				}
				if text != "" {
					input.CapsuleText = &text
				}
			}

			if title := c.String("title"); title != "" {
				input.Title = &title
			}
			if c.IsSet("tags") {
				tags := parseTags(c.String("tags"))
				input.Tags = &tags
			}

			output, err := ops.Update(c.Context, db, cfg, input)
			if err != nil {
				return outputError(err)
			}

			return outputJSON(output)
		},
	}
}

// deleteCmd creates the delete command.
func deleteCmd(db *sql.DB) *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Soft-delete a capsule",
		ArgsUsage: "[id]",
		Flags:     addressingFlags(),
		Action: func(c *cli.Context) error {
			addr, err := parseAddressing(c)
			if err != nil {
				return outputError(err)
			}

			input := ops.DeleteInput{
				ID:        addr.ID,
				Workspace: addr.Workspace,
				Name:      addr.Name,
			}

			output, err := ops.Delete(c.Context, db, input)
			if err != nil {
				return outputError(err)
			}

			return outputJSON(output)
		},
	}
}

// listCmd creates the list command.
func listCmd(db *sql.DB) *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List capsules in a workspace",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "workspace", Aliases: []string{"w"}, Value: "default", Usage: "Workspace name"},
			&cli.IntFlag{Name: "limit", Aliases: []string{"l"}, Value: 20, Usage: "Maximum items to return"},
			&cli.IntFlag{Name: "offset", Aliases: []string{"o"}, Value: 0, Usage: "Items to skip"},
			&cli.BoolFlag{Name: "include-deleted", Usage: "Include soft-deleted capsules"},
		},
		Action: func(c *cli.Context) error {
			if err := validatePagination(c); err != nil {
				return outputError(err)
			}

			input := ops.ListInput{
				Workspace:      c.String("workspace"),
				Limit:          c.Int("limit"),
				Offset:         c.Int("offset"),
				IncludeDeleted: c.Bool("include-deleted"),
			}

			output, err := ops.List(c.Context, db, input)
			if err != nil {
				return outputError(err)
			}

			return outputJSON(output)
		},
	}
}

// inventoryCmd creates the inventory command.
func inventoryCmd(db *sql.DB) *cli.Command {
	return &cli.Command{
		Name:  "inventory",
		Usage: "List all capsules across workspaces with optional filters",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "workspace", Aliases: []string{"w"}, Usage: "Filter by workspace"},
			&cli.StringFlag{Name: "tag", Usage: "Filter by tag"},
			&cli.StringFlag{Name: "name-prefix", Usage: "Filter by name prefix"},
			&cli.IntFlag{Name: "limit", Aliases: []string{"l"}, Value: 100, Usage: "Maximum items to return"},
			&cli.IntFlag{Name: "offset", Aliases: []string{"o"}, Value: 0, Usage: "Items to skip"},
			&cli.BoolFlag{Name: "include-deleted", Usage: "Include soft-deleted capsules"},
		},
		Action: func(c *cli.Context) error {
			if err := validatePagination(c); err != nil {
				return outputError(err)
			}

			input := ops.InventoryInput{
				Limit:          c.Int("limit"),
				Offset:         c.Int("offset"),
				IncludeDeleted: c.Bool("include-deleted"),
				Workspace:      optionalString(c, "workspace"),
				Tag:            optionalString(c, "tag"),
				NamePrefix:     optionalString(c, "name-prefix"),
			}

			output, err := ops.Inventory(c.Context, db, input)
			if err != nil {
				return outputError(err)
			}

			return outputJSON(output)
		},
	}
}

// latestCmd creates the latest command.
func latestCmd(db *sql.DB) *cli.Command {
	return &cli.Command{
		Name:  "latest",
		Usage: "Get the most recently updated capsule in a workspace",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "workspace", Aliases: []string{"w"}, Value: "default", Usage: "Workspace name"},
			&cli.BoolFlag{Name: "include-text", Usage: "Include capsule_text in output"},
			&cli.BoolFlag{Name: "include-deleted", Usage: "Include soft-deleted capsules"},
		},
		Action: func(c *cli.Context) error {
			input := ops.LatestInput{
				Workspace:      c.String("workspace"),
				IncludeDeleted: c.Bool("include-deleted"),
			}

			if c.Bool("include-text") {
				includeText := true
				input.IncludeText = &includeText
			}

			output, err := ops.Latest(c.Context, db, input)
			if err != nil {
				return outputError(err)
			}

			return outputJSON(output)
		},
	}
}

// exportCmd creates the export command.
func exportCmd(db *sql.DB, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "export",
		Usage: "Export capsules to a JSONL file",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "path", Aliases: []string{"p"}, Usage: "Export file path (default: ~/.moss/exports/<workspace>-<timestamp>.jsonl)"},
			&cli.StringFlag{Name: "workspace", Aliases: []string{"w"}, Usage: "Filter by workspace"},
			&cli.BoolFlag{Name: "include-deleted", Usage: "Include soft-deleted capsules"},
		},
		Action: func(c *cli.Context) error {
			input := ops.ExportInput{
				Path:           c.String("path"),
				IncludeDeleted: c.Bool("include-deleted"),
				Workspace:      optionalString(c, "workspace"),
			}

			output, err := ops.Export(c.Context, db, cfg, input)
			if err != nil {
				return outputError(err)
			}

			return outputJSON(output)
		},
	}
}

// importCmd creates the import command.
func importCmd(db *sql.DB, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "import",
		Usage: "Import capsules from a JSONL file",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "path", Aliases: []string{"p"}, Required: true, Usage: "Import file path"},
			&cli.StringFlag{Name: "mode", Aliases: []string{"m"}, Value: "error", Usage: "Collision mode: error|replace|rename"},
		},
		Action: func(c *cli.Context) error {
			input := ops.ImportInput{
				Path: c.String("path"),
				Mode: ops.ImportMode(c.String("mode")),
			}

			output, err := ops.Import(c.Context, db, cfg, input)
			if err != nil {
				return outputError(err)
			}

			return outputJSON(output)
		},
	}
}

// purgeCmd creates the purge command.
func purgeCmd(db *sql.DB) *cli.Command {
	return &cli.Command{
		Name:  "purge",
		Usage: "Permanently delete soft-deleted capsules",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "workspace", Aliases: []string{"w"}, Usage: "Filter by workspace"},
			&cli.StringFlag{Name: "older-than", Usage: "Only purge if deleted more than N days ago (e.g., 7d)"},
		},
		Action: func(c *cli.Context) error {
			input := ops.PurgeInput{
				Workspace: optionalString(c, "workspace"),
			}

			if olderThan := c.String("older-than"); olderThan != "" {
				days, err := parseDuration(olderThan)
				if err != nil {
					return outputError(errors.NewInvalidRequest(err.Error()))
				}
				input.OlderThanDays = &days
			}

			output, err := ops.Purge(c.Context, db, input)
			if err != nil {
				return outputError(err)
			}

			return outputJSON(output)
		},
	}
}

// toolsCmd creates the tools command.
func toolsCmd(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "tools",
		Usage: "List available MCP tools",
		Action: func(c *cli.Context) error {
			// Get all tool names and sort them
			allNames := mcp.AllToolNames()
			sort.Strings(allNames)

			// Build disabled set from primitives + individual tools
			// Track reason for disabling: "primitive" or "tool"
			disabledByPrimitive := make(map[string]bool)
			for _, tool := range mcp.ExpandPrimitivesToTools(cfg.DisabledPrimitives) {
				disabledByPrimitive[tool] = true
			}

			disabledByTool := make(map[string]bool)
			for _, name := range cfg.DisabledTools {
				disabledByTool[name] = true
			}

			// Build tool list with status
			type toolStatus struct {
				Name      string `json:"name"`
				Primitive string `json:"primitive"`
				Enabled   bool   `json:"enabled"`
				Reason    string `json:"reason,omitempty"`
			}

			tools := make([]toolStatus, 0, len(allNames))
			enabledCount := 0
			for _, name := range allNames {
				prim := mcp.GetPrimitiveForTool(name)
				ts := toolStatus{
					Name:      name,
					Primitive: prim,
					Enabled:   true,
				}

				// Check if disabled (primitive takes precedence in reason)
				if disabledByPrimitive[name] {
					ts.Enabled = false
					ts.Reason = "primitive"
				} else if disabledByTool[name] {
					ts.Enabled = false
					ts.Reason = "tool"
				}

				tools = append(tools, ts)
				if ts.Enabled {
					enabledCount++
				}
			}

			output := struct {
				Tools    []toolStatus `json:"tools"`
				Total    int          `json:"total"`
				Enabled  int          `json:"enabled"`
				Disabled int          `json:"disabled"`
			}{
				Tools:    tools,
				Total:    len(tools),
				Enabled:  enabledCount,
				Disabled: len(tools) - enabledCount,
			}

			return outputJSON(output)
		},
	}
}

// Helper functions

// outputJSON marshals result to stdout as JSON.
func outputJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// outputError formats error for CLI.
func outputError(err error) error {
	var mossErr *errors.MossError
	if stderrors.As(err, &mossErr) {
		return cli.Exit(fmt.Sprintf("[%s] %s", mossErr.Code, mossErr.Message), 1)
	}
	return cli.Exit(err.Error(), 1)
}

// addressingFlags returns common flags for commands that use ID or name addressing.
func addressingFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "workspace", Aliases: []string{"w"}, Value: "default", Usage: "Workspace name"},
		&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "Capsule name"},
	}
}

// addressing holds parsed addressing info (ID or workspace+name).
type addressing struct {
	ID        string
	Workspace string
	Name      string
}

// parseAddressing extracts addressing from CLI context.
// Returns error if both positional ID and --name flag are provided (ambiguous).
func parseAddressing(c *cli.Context) (addressing, error) {
	if c.NArg() > 0 {
		// If the user explicitly sets name/workspace flags while also providing an ID,
		// treat it as ambiguous (mirrors MCP's mutual exclusivity).
		if c.IsSet("name") || c.IsSet("workspace") {
			return addressing{}, errors.NewAmbiguousAddressing()
		}
		return addressing{ID: c.Args().First()}, nil
	}
	return addressing{
		Workspace: c.String("workspace"),
		Name:      c.String("name"),
	}, nil
}

// stdinHasData returns true if stdin has piped data (not a terminal).
func stdinHasData() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// validatePagination checks that limit and offset are non-negative.
func validatePagination(c *cli.Context) error {
	if c.Int("limit") < 0 {
		return errors.NewInvalidRequest("limit must be non-negative")
	}
	if c.Int("offset") < 0 {
		return errors.NewInvalidRequest("offset must be non-negative")
	}
	return nil
}

// optionalString returns a pointer to the flag value if set and non-empty, nil otherwise.
func optionalString(c *cli.Context, flag string) *string {
	if v := c.String(flag); v != "" {
		return &v
	}
	return nil
}

// readStdin reads content from stdin with a size limit.
// Returns error if stdin exceeds maxBytes.
func readStdin(maxBytes int) (string, error) {
	// Read up to maxBytes + 1 to detect overflow
	limited := io.LimitReader(os.Stdin, int64(maxBytes+1))
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	if len(data) > maxBytes {
		return "", fmt.Errorf("input exceeds maximum size of %d bytes", maxBytes)
	}
	return strings.TrimSpace(string(data)), nil
}

// parseTags splits a comma-separated string into a slice of tags.
func parseTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// parseDuration parses "7d" format to days.
func parseDuration(s string) (int, error) {
	if numStr, ok := strings.CutSuffix(s, "d"); ok {
		days, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, fmt.Errorf("invalid duration: %s", s)
		}
		if days < 0 {
			return 0, fmt.Errorf("duration must be non-negative")
		}
		return days, nil
	}
	return 0, fmt.Errorf("duration must end with 'd' (days), e.g., 7d")
}
