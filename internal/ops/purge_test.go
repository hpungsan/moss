package ops

import (
	"context"
	"testing"
	"time"

	"github.com/hpungsan/moss/internal/capsule"
	"github.com/hpungsan/moss/internal/db"
)

func newTestCapsuleForPurge(id, workspaceRaw, text string) *capsule.Capsule {
	now := time.Now().Unix()
	return &capsule.Capsule{
		ID:             id,
		WorkspaceRaw:   workspaceRaw,
		WorkspaceNorm:  capsule.Normalize(workspaceRaw),
		CapsuleText:    text,
		CapsuleChars:   capsule.CountChars(text),
		TokensEstimate: capsule.EstimateTokens(text),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func TestPurge_AllDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Create and soft-delete capsules
	c1 := newTestCapsuleForPurge("01PURGE1", "default", "Deleted 1")
	c2 := newTestCapsuleForPurge("01PURGE2", "default", "Deleted 2")
	c3 := newTestCapsuleForPurge("01PURGE3", "default", "Active")

	for _, c := range []*capsule.Capsule{c1, c2, c3} {
		if err := db.Insert(database, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	if err := db.SoftDelete(database, c1.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}
	if err := db.SoftDelete(database, c2.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Purge all deleted
	output, err := Purge(context.Background(), database, PurgeInput{})
	if err != nil {
		t.Fatalf("Purge failed: %v", err)
	}

	if output.Purged != 2 {
		t.Errorf("Purged = %d, want 2", output.Purged)
	}
	if output.Message == "" {
		t.Error("Message should not be empty")
	}

	// Verify active capsule still exists
	_, err = db.GetByID(database, c3.ID, false)
	if err != nil {
		t.Errorf("Active capsule should still exist: %v", err)
	}
}

func TestPurge_WorkspaceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Create and soft-delete capsules in different workspaces
	c1 := newTestCapsuleForPurge("01PURGE4", "target", "Deleted in target")
	c2 := newTestCapsuleForPurge("01PURGE5", "other", "Deleted in other")

	for _, c := range []*capsule.Capsule{c1, c2} {
		if err := db.Insert(database, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
		if err := db.SoftDelete(database, c.ID); err != nil {
			t.Fatalf("SoftDelete failed: %v", err)
		}
	}

	// Purge only target workspace
	ws := "target"
	output, err := Purge(context.Background(), database, PurgeInput{Workspace: &ws})
	if err != nil {
		t.Fatalf("Purge failed: %v", err)
	}

	if output.Purged != 1 {
		t.Errorf("Purged = %d, want 1", output.Purged)
	}

	// Verify other workspace capsule still exists
	_, err = db.GetByID(database, c2.ID, true)
	if err != nil {
		t.Errorf("Other workspace capsule should still exist: %v", err)
	}
}

func TestPurge_OlderThanDays(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	c1 := newTestCapsuleForPurge("01PURGE6", "default", "Recent")
	c2 := newTestCapsuleForPurge("01PURGE7", "default", "Old")

	for _, c := range []*capsule.Capsule{c1, c2} {
		if err := db.Insert(database, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	// Soft-delete c1 (recent)
	if err := db.SoftDelete(database, c1.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// For c2, manually set deleted_at to 15 days ago
	fifteenDaysAgo := time.Now().Unix() - (15 * 24 * 60 * 60)
	_, err = database.Exec("UPDATE capsules SET deleted_at = ? WHERE id = ?", fifteenDaysAgo, c2.ID)
	if err != nil {
		t.Fatalf("Failed to set old deleted_at: %v", err)
	}

	// Purge capsules deleted more than 7 days ago
	days := 7
	output, err := Purge(context.Background(), database, PurgeInput{OlderThanDays: &days})
	if err != nil {
		t.Fatalf("Purge failed: %v", err)
	}

	if output.Purged != 1 {
		t.Errorf("Purged = %d, want 1", output.Purged)
	}

	// Recent deleted capsule should still exist
	_, err = db.GetByID(database, c1.ID, true)
	if err != nil {
		t.Errorf("Recent deleted capsule should still exist: %v", err)
	}
}

func TestPurge_CombinedFilters(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Create capsules in different workspaces with different ages
	c1 := newTestCapsuleForPurge("01PURGE8", "ws1", "ws1 old")
	c2 := newTestCapsuleForPurge("01PURGE9", "ws1", "ws1 recent")
	c3 := newTestCapsuleForPurge("01PURGEA", "ws2", "ws2 old")

	for _, c := range []*capsule.Capsule{c1, c2, c3} {
		if err := db.Insert(database, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	// Soft-delete c2 (recent)
	if err := db.SoftDelete(database, c2.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Set old deleted_at for c1 and c3
	tenDaysAgo := time.Now().Unix() - (10 * 24 * 60 * 60)
	for _, id := range []string{c1.ID, c3.ID} {
		_, err = database.Exec("UPDATE capsules SET deleted_at = ? WHERE id = ?", tenDaysAgo, id)
		if err != nil {
			t.Fatalf("Failed to set old deleted_at: %v", err)
		}
	}

	// Purge only ws1, older than 7 days
	ws := "ws1"
	days := 7
	output, err := Purge(context.Background(), database, PurgeInput{Workspace: &ws, OlderThanDays: &days})
	if err != nil {
		t.Fatalf("Purge failed: %v", err)
	}

	if output.Purged != 1 {
		t.Errorf("Purged = %d, want 1 (only ws1 old)", output.Purged)
	}

	// ws1 recent should still exist
	_, err = db.GetByID(database, c2.ID, true)
	if err != nil {
		t.Errorf("ws1 recent should still exist: %v", err)
	}

	// ws2 old should still exist (different workspace)
	_, err = db.GetByID(database, c3.ID, true)
	if err != nil {
		t.Errorf("ws2 old should still exist: %v", err)
	}
}

func TestPurge_NoDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Insert only active capsule
	c := newTestCapsuleForPurge("01PURGEB", "default", "Active")
	if err := db.Insert(database, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	output, err := Purge(context.Background(), database, PurgeInput{})
	if err != nil {
		t.Fatalf("Purge failed: %v", err)
	}

	if output.Purged != 0 {
		t.Errorf("Purged = %d, want 0", output.Purged)
	}
	if output.Message != "No deleted capsules to purge" {
		t.Errorf("Message = %q, want 'No deleted capsules to purge'", output.Message)
	}
}

func TestPurge_DoesNotAffectActive(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Create multiple active capsules
	for i := range 5 {
		c := newTestCapsuleForPurge("01PURGEC"+string(rune('0'+i)), "default", "Active")
		if err := db.Insert(database, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	output, err := Purge(context.Background(), database, PurgeInput{})
	if err != nil {
		t.Fatalf("Purge failed: %v", err)
	}

	if output.Purged != 0 {
		t.Errorf("Purged = %d, want 0", output.Purged)
	}

	// Verify all active capsules still exist
	for i := range 5 {
		_, err := db.GetByID(database, "01PURGEC"+string(rune('0'+i)), false)
		if err != nil {
			t.Errorf("Active capsule %d should still exist: %v", i, err)
		}
	}
}

func TestPurge_NegativeOlderThanDays(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	negativeDays := -1
	_, err = Purge(context.Background(), database, PurgeInput{
		OlderThanDays: &negativeDays,
	})
	if err == nil {
		t.Fatal("Expected error for negative older_than_days, got nil")
	}
	// Error format is "INVALID_REQUEST: older_than_days cannot be negative"
	want := "INVALID_REQUEST: older_than_days cannot be negative"
	if err.Error() != want {
		t.Errorf("Error = %q, want %q", err.Error(), want)
	}
}
