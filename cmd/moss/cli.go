package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/errors"
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
			exportCmd(db),
			importCmd(db),
			purgeCmd(db),
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

			capsuleText, err := readStdin()
			if err != nil {
				return outputError(errors.NewInternal(err))
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

			output, err := ops.Store(db, cfg, input)
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
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "workspace", Aliases: []string{"w"}, Value: "default", Usage: "Workspace name"},
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "Capsule name"},
			&cli.BoolFlag{Name: "include-deleted", Usage: "Include soft-deleted capsules"},
			&cli.BoolFlag{Name: "no-text", Usage: "Exclude capsule_text from output"},
		},
		Action: func(c *cli.Context) error {
			input := ops.FetchInput{
				IncludeDeleted: c.Bool("include-deleted"),
			}

			// Check for positional ID argument
			if c.NArg() > 0 {
				input.ID = c.Args().First()
			} else {
				input.Workspace = c.String("workspace")
				input.Name = c.String("name")
			}

			if c.Bool("no-text") {
				includeText := false
				input.IncludeText = &includeText
			}

			output, err := ops.Fetch(db, input)
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
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "workspace", Aliases: []string{"w"}, Value: "default", Usage: "Workspace name"},
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "Capsule name"},
			&cli.StringFlag{Name: "title", Aliases: []string{"t"}, Usage: "New title"},
			&cli.StringFlag{Name: "tags", Usage: "New comma-separated tags"},
			&cli.BoolFlag{Name: "allow-thin", Usage: "Allow capsules without all required sections"},
		},
		Action: func(c *cli.Context) error {
			input := ops.UpdateInput{
				AllowThin: c.Bool("allow-thin"),
			}

			// Check for positional ID argument
			if c.NArg() > 0 {
				input.ID = c.Args().First()
			} else {
				input.Workspace = c.String("workspace")
				input.Name = c.String("name")
			}

			// Read capsule_text from stdin if piped
			if stdinHasData() {
				text, err := readStdin()
				if err != nil {
					return outputError(errors.NewInternal(err))
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

			output, err := ops.Update(db, cfg, input)
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
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "workspace", Aliases: []string{"w"}, Value: "default", Usage: "Workspace name"},
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "Capsule name"},
		},
		Action: func(c *cli.Context) error {
			input := ops.DeleteInput{}

			// Check for positional ID argument
			if c.NArg() > 0 {
				input.ID = c.Args().First()
			} else {
				input.Workspace = c.String("workspace")
				input.Name = c.String("name")
			}

			output, err := ops.Delete(db, input)
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
			input := ops.ListInput{
				Workspace:      c.String("workspace"),
				Limit:          c.Int("limit"),
				Offset:         c.Int("offset"),
				IncludeDeleted: c.Bool("include-deleted"),
			}

			output, err := ops.List(db, input)
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
			input := ops.InventoryInput{
				Limit:          c.Int("limit"),
				Offset:         c.Int("offset"),
				IncludeDeleted: c.Bool("include-deleted"),
			}

			if workspace := c.String("workspace"); workspace != "" {
				input.Workspace = &workspace
			}
			if tag := c.String("tag"); tag != "" {
				input.Tag = &tag
			}
			if prefix := c.String("name-prefix"); prefix != "" {
				input.NamePrefix = &prefix
			}

			output, err := ops.Inventory(db, input)
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

			output, err := ops.Latest(db, input)
			if err != nil {
				return outputError(err)
			}

			return outputJSON(output)
		},
	}
}

// exportCmd creates the export command.
func exportCmd(db *sql.DB) *cli.Command {
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
			}

			if workspace := c.String("workspace"); workspace != "" {
				input.Workspace = &workspace
			}

			output, err := ops.Export(db, input)
			if err != nil {
				return outputError(err)
			}

			return outputJSON(output)
		},
	}
}

// importCmd creates the import command.
func importCmd(db *sql.DB) *cli.Command {
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

			output, err := ops.Import(db, input)
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
			input := ops.PurgeInput{}

			if workspace := c.String("workspace"); workspace != "" {
				input.Workspace = &workspace
			}
			if olderThan := c.String("older-than"); olderThan != "" {
				days, err := parseDuration(olderThan)
				if err != nil {
					return outputError(errors.NewInvalidRequest(err.Error()))
				}
				input.OlderThanDays = &days
			}

			output, err := ops.Purge(db, input)
			if err != nil {
				return outputError(err)
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
	if mossErr, ok := err.(*errors.MossError); ok {
		return cli.Exit(fmt.Sprintf("[%s] %s", mossErr.Code, mossErr.Message), 1)
	}
	return cli.Exit(err.Error(), 1)
}

// stdinHasData returns true if stdin has piped data (not a terminal).
func stdinHasData() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// readStdin reads all content from stdin.
func readStdin() (string, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
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
