package db

import (
	"database/sql"

	"github.com/hpungsan/moss/internal/capsule"
)

// Insert stores a new capsule in the database.
// Stub for Phase 2 implementation.
func Insert(db *sql.DB, c *capsule.Capsule) error {
	// TODO: Implement in Phase 2
	return nil
}

// GetByID retrieves a capsule by its ULID.
// Stub for Phase 2 implementation.
func GetByID(db *sql.DB, id string) (*capsule.Capsule, error) {
	// TODO: Implement in Phase 2
	return nil, nil
}

// GetByName retrieves a capsule by workspace and name.
// Stub for Phase 2 implementation.
func GetByName(db *sql.DB, workspace, name string) (*capsule.Capsule, error) {
	// TODO: Implement in Phase 2
	return nil, nil
}

// Update modifies an existing capsule.
// Stub for Phase 2 implementation.
func Update(db *sql.DB, c *capsule.Capsule) error {
	// TODO: Implement in Phase 2
	return nil
}

// SoftDelete marks a capsule as deleted without removing it.
// Stub for Phase 2 implementation.
func SoftDelete(db *sql.DB, id string) error {
	// TODO: Implement in Phase 2
	return nil
}
