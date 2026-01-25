package ops

import (
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
	"github.com/stretchr/testify/require"
)

// TestFullWorkflow exercises the complete capsule lifecycle:
// store → fetch → update → list → delete → purge → fetch (not found)
func TestFullWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	require.NoError(t, err)
	defer database.Close()

	cfg := config.DefaultConfig()

	ws := "workflow-test"
	name := "lifecycle"

	// 1. Store
	storeOut, err := Store(database, cfg, StoreInput{
		Workspace:   ws,
		Name:        stringPtr(name),
		CapsuleText: validCapsuleText,
	})
	require.NoError(t, err)
	require.NotEmpty(t, storeOut.ID)
	id := storeOut.ID

	// 2. Fetch by name
	fetchOut, err := Fetch(database, FetchInput{Workspace: ws, Name: name})
	require.NoError(t, err)
	require.Equal(t, id, fetchOut.ID)
	require.Contains(t, fetchOut.CapsuleText, "## Objective")

	// 3. Update title
	newTitle := "Updated Lifecycle Test"
	updateOut, err := Update(database, cfg, UpdateInput{ID: id, Title: &newTitle})
	require.NoError(t, err)
	require.Equal(t, id, updateOut.ID)

	// Verify title was updated
	fetchOut, err = Fetch(database, FetchInput{ID: id})
	require.NoError(t, err)
	require.NotNil(t, fetchOut.Title)
	require.Equal(t, newTitle, *fetchOut.Title)

	// 4. List - verify capsule appears
	listOut, err := List(database, ListInput{Workspace: ws})
	require.NoError(t, err)
	require.Len(t, listOut.Items, 1)
	require.Equal(t, id, listOut.Items[0].ID)

	// 5. Delete (soft)
	deleteOut, err := Delete(database, DeleteInput{ID: id})
	require.NoError(t, err)
	require.Equal(t, id, deleteOut.ID)

	// 6. List - verify excluded from default listing
	listOut, err = List(database, ListInput{Workspace: ws})
	require.NoError(t, err)
	require.Len(t, listOut.Items, 0)

	// Verify still accessible with include_deleted
	listOut, err = List(database, ListInput{Workspace: ws, IncludeDeleted: true})
	require.NoError(t, err)
	require.Len(t, listOut.Items, 1)

	// 7. Purge
	purgeOut, err := Purge(database, PurgeInput{Workspace: &ws})
	require.NoError(t, err)
	require.Equal(t, 1, purgeOut.Purged)

	// 8. Fetch - verify 404 (even with include_deleted, purged = gone)
	_, err = Fetch(database, FetchInput{ID: id, IncludeDeleted: true})
	require.Error(t, err)
	var mossErr *errors.MossError
	require.ErrorAs(t, err, &mossErr)
	require.Equal(t, errors.ErrNotFound, mossErr.Code)
}
