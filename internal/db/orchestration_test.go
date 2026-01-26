package db

import (
	"testing"

	"github.com/hpungsan/moss/internal/capsule"
)

// =============================================================================
// Orchestration Fields Tests (run_id, phase, role)
// =============================================================================

func TestInsert_WithOrchestrationFields(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	runID := "pr-review-abc123"
	phase := "design"
	role := "design-intent"

	c := &capsule.Capsule{
		ID:             "01ORCH001",
		WorkspaceRaw:   "default",
		WorkspaceNorm:  "default",
		CapsuleText:    "Test content",
		CapsuleChars:   12,
		TokensEstimate: 3,
		RunID:          &runID,
		Phase:          &phase,
		Role:           &role,
		CreatedAt:      1000,
		UpdatedAt:      1000,
	}

	if err := Insert(db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Verify fields were stored
	got, err := GetByID(db, c.ID, false)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.RunID == nil || *got.RunID != runID {
		t.Errorf("RunID = %v, want %q", got.RunID, runID)
	}
	if got.Phase == nil || *got.Phase != phase {
		t.Errorf("Phase = %v, want %q", got.Phase, phase)
	}
	if got.Role == nil || *got.Role != role {
		t.Errorf("Role = %v, want %q", got.Role, role)
	}
}

func TestListByWorkspace_FilterByRunID(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	// Insert capsules with different run_ids
	runID1 := "run-001"
	runID2 := "run-002"

	c1 := newTestCapsule("01ORCH101", "default", "Content 1")
	c1.RunID = &runID1
	c2 := newTestCapsule("01ORCH102", "default", "Content 2")
	c2.RunID = &runID1
	c3 := newTestCapsule("01ORCH103", "default", "Content 3")
	c3.RunID = &runID2

	for _, c := range []*capsule.Capsule{c1, c2, c3} {
		if err := Insert(db, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	// Filter by run_id
	summaries, total, err := ListByWorkspace(db, "default", ListFilters{RunID: &runID1}, 10, 0, false)
	if err != nil {
		t.Fatalf("ListByWorkspace failed: %v", err)
	}

	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(summaries) != 2 {
		t.Errorf("len(summaries) = %d, want 2", len(summaries))
	}
}

func TestListByWorkspace_FilterByPhaseAndRole(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	runID := "run-001"
	designPhase := "design"
	implPhase := "implement"
	designerRole := "designer"
	reviewerRole := "reviewer"

	c1 := newTestCapsule("01ORCH201", "default", "Design intent")
	c1.RunID = &runID
	c1.Phase = &designPhase
	c1.Role = &designerRole

	c2 := newTestCapsule("01ORCH202", "default", "Implementation")
	c2.RunID = &runID
	c2.Phase = &implPhase
	c2.Role = &designerRole

	c3 := newTestCapsule("01ORCH203", "default", "Review feedback")
	c3.RunID = &runID
	c3.Phase = &designPhase
	c3.Role = &reviewerRole

	for _, c := range []*capsule.Capsule{c1, c2, c3} {
		if err := Insert(db, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	// Filter by phase only
	_, total, _ := ListByWorkspace(db, "default", ListFilters{Phase: &designPhase}, 10, 0, false)
	if total != 2 {
		t.Errorf("filter by phase: total = %d, want 2", total)
	}

	// Filter by phase AND role
	summaries, total, _ := ListByWorkspace(db, "default", ListFilters{Phase: &designPhase, Role: &designerRole}, 10, 0, false)
	if total != 1 {
		t.Errorf("filter by phase+role: total = %d, want 1", total)
	}
	if len(summaries) > 0 && summaries[0].ID != "01ORCH201" {
		t.Errorf("expected 01ORCH201, got %s", summaries[0].ID)
	}
}

func TestGetLatestSummary_FilterByRunID(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	runID1 := "run-001"
	runID2 := "run-002"

	c1 := newTestCapsule("01ORCH301", "default", "Old run1")
	c1.RunID = &runID1
	c1.UpdatedAt = 1000

	c2 := newTestCapsule("01ORCH302", "default", "New run2")
	c2.RunID = &runID2
	c2.UpdatedAt = 2000

	c3 := newTestCapsule("01ORCH303", "default", "Newer run1")
	c3.RunID = &runID1
	c3.UpdatedAt = 1500

	for _, c := range []*capsule.Capsule{c1, c2, c3} {
		if err := Insert(db, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	// Latest without filter - should get most recent overall
	summary, _ := GetLatestSummary(db, "default", LatestFilters{}, false)
	if summary.ID != "01ORCH302" {
		t.Errorf("latest overall = %q, want 01ORCH302", summary.ID)
	}

	// Latest for run-001 only
	summary, _ = GetLatestSummary(db, "default", LatestFilters{RunID: &runID1}, false)
	if summary.ID != "01ORCH303" {
		t.Errorf("latest for run-001 = %q, want 01ORCH303", summary.ID)
	}
}

func TestInventory_FilterByOrchestrationFields(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	runID := "run-001"
	phase := "design"

	c1 := newTestCapsule("01ORCH401", "ws1", "Content")
	c1.RunID = &runID
	c1.Phase = &phase

	c2 := newTestCapsule("01ORCH402", "ws2", "Content")
	c2.RunID = &runID

	c3 := newTestCapsule("01ORCH403", "ws1", "Content")

	for _, c := range []*capsule.Capsule{c1, c2, c3} {
		if err := Insert(db, c); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	// Filter by run_id across all workspaces
	_, total, _ := ListAll(db, InventoryFilters{RunID: &runID}, 10, 0, false)
	if total != 2 {
		t.Errorf("filter by run_id: total = %d, want 2", total)
	}

	// Filter by run_id + phase
	summaries, total, _ := ListAll(db, InventoryFilters{RunID: &runID, Phase: &phase}, 10, 0, false)
	if total != 1 {
		t.Errorf("filter by run_id+phase: total = %d, want 1", total)
	}
	if len(summaries) > 0 && summaries[0].ID != "01ORCH401" {
		t.Errorf("expected 01ORCH401, got %s", summaries[0].ID)
	}
}

func TestUpdateByID_OrchestrationFields(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	c := newTestCapsule("01ORCH501", "default", "Content")
	if err := Insert(db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Update with orchestration fields
	runID := "new-run"
	phase := "implement"
	role := "coder"
	c.RunID = &runID
	c.Phase = &phase
	c.Role = &role

	if err := UpdateByID(db, c); err != nil {
		t.Fatalf("UpdateByID failed: %v", err)
	}

	// Verify update
	got, _ := GetByID(db, c.ID, false)
	if got.RunID == nil || *got.RunID != runID {
		t.Errorf("RunID = %v, want %q", got.RunID, runID)
	}
	if got.Phase == nil || *got.Phase != phase {
		t.Errorf("Phase = %v, want %q", got.Phase, phase)
	}
	if got.Role == nil || *got.Role != role {
		t.Errorf("Role = %v, want %q", got.Role, role)
	}
}

func TestCapsuleSummary_IncludesOrchestrationFields(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	runID := "test-run"
	phase := "test-phase"
	role := "test-role"

	c := newTestCapsule("01ORCH601", "default", "Content")
	c.RunID = &runID
	c.Phase = &phase
	c.Role = &role

	if err := Insert(db, c); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	summaries, _, _ := ListByWorkspace(db, "default", ListFilters{}, 10, 0, false)
	if len(summaries) == 0 {
		t.Fatal("expected at least one summary")
	}

	s := summaries[0]
	if s.RunID == nil || *s.RunID != runID {
		t.Errorf("summary.RunID = %v, want %q", s.RunID, runID)
	}
	if s.Phase == nil || *s.Phase != phase {
		t.Errorf("summary.Phase = %v, want %q", s.Phase, phase)
	}
	if s.Role == nil || *s.Role != role {
		t.Errorf("summary.Role = %v, want %q", s.Role, role)
	}
}
