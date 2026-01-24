package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hpungsan/moss/internal/db"
)

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

	fmt.Println("Moss initialized successfully")
}
