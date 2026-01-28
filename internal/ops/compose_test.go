package ops

import (
	"strings"
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

func TestCompose_Markdown_MultipleCapsules(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store two capsules
	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("cap1"),
		Title:       stringPtr("Capsule One"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store cap1 failed: %v", err)
	}

	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("cap2"),
		Title:       stringPtr("Capsule Two"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store cap2 failed: %v", err)
	}

	// Compose them
	output, err := Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "cap1"},
			{Workspace: "default", Name: "cap2"},
		},
		Format: "markdown",
	})
	if err != nil {
		t.Fatalf("Compose failed: %v", err)
	}

	if output.PartsCount != 2 {
		t.Errorf("PartsCount = %d, want 2", output.PartsCount)
	}
	if output.BundleChars == 0 {
		t.Error("BundleChars should not be 0")
	}
	if !strings.Contains(output.BundleText, "## Capsule One") {
		t.Error("BundleText should contain '## Capsule One'")
	}
	if !strings.Contains(output.BundleText, "## Capsule Two") {
		t.Error("BundleText should contain '## Capsule Two'")
	}
	if !strings.Contains(output.BundleText, "---") {
		t.Error("BundleText should contain '---' separator")
	}
	if output.Stored != nil {
		t.Error("Stored should be nil when store_as not provided")
	}
}

func TestCompose_JSON_Format(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store two capsules
	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("cap1"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store cap1 failed: %v", err)
	}

	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("cap2"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store cap2 failed: %v", err)
	}

	// Compose them in JSON format
	output, err := Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "cap1"},
			{Workspace: "default", Name: "cap2"},
		},
		Format: "json",
	})
	if err != nil {
		t.Fatalf("Compose failed: %v", err)
	}

	if output.PartsCount != 2 {
		t.Errorf("PartsCount = %d, want 2", output.PartsCount)
	}
	if !strings.Contains(output.BundleText, `"parts"`) {
		t.Error("JSON BundleText should contain 'parts' key")
	}
	if !strings.Contains(output.BundleText, `"id"`) {
		t.Error("JSON BundleText should contain 'id' field")
	}
	if !strings.Contains(output.BundleText, `"text"`) {
		t.Error("JSON BundleText should contain 'text' field")
	}
}

func TestCompose_ByID(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	stored, err := Store(database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Compose by ID
	output, err := Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{ID: stored.ID},
		},
	})
	if err != nil {
		t.Fatalf("Compose failed: %v", err)
	}

	if output.PartsCount != 1 {
		t.Errorf("PartsCount = %d, want 1", output.PartsCount)
	}
	// Title should fall back to ID for unnamed capsule without title
	if !strings.Contains(output.BundleText, "## "+stored.ID) {
		t.Errorf("BundleText should contain capsule ID as heading, got: %s", output.BundleText[:100])
	}
}

func TestCompose_MissingCapsule_AllOrNothing(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store only one capsule
	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("exists"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Try to compose with one existing and one missing
	_, err = Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "exists"},
			{Workspace: "default", Name: "missing"},
		},
	})
	if err == nil {
		t.Fatal("Compose should fail when any capsule is missing")
	}
	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestCompose_EmptyItems(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Try to compose with empty items
	_, err = Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{},
	})
	if err == nil {
		t.Fatal("Compose should fail with empty items")
	}
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("error = %v, want ErrInvalidRequest", err)
	}
}

func TestCompose_TooManyItems(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Create more refs than allowed
	refs := make([]ComposeRef, MaxFetchManyItems+1)
	for i := range refs {
		refs[i] = ComposeRef{ID: "some-id"}
	}

	_, err = Compose(database, cfg, ComposeInput{Items: refs})
	if err == nil {
		t.Fatal("Compose should fail with too many items")
	}
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("error = %v, want ErrInvalidRequest", err)
	}
}

func TestCompose_SizeLimitExceeded(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	// Use normal config for storing
	storeCfg := config.DefaultConfig()

	// Store a capsule
	_, err = Store(database, storeCfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("cap1"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store cap1 failed: %v", err)
	}

	// Use small config for compose - smaller than the capsule size
	composeCfg := &config.Config{CapsuleMaxChars: 100}

	// Try to compose - should exceed size limit
	_, err = Compose(database, composeCfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "cap1"},
		},
	})
	if err == nil {
		t.Fatal("Compose should fail when size limit exceeded")
	}
	if !errors.Is(err, errors.ErrComposeTooLarge) {
		t.Errorf("error = %v, want ErrComposeTooLarge", err)
	}
}

func TestCompose_WithStoreAs(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store two capsules
	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("cap1"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store cap1 failed: %v", err)
	}

	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("cap2"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store cap2 failed: %v", err)
	}

	// Compose and store
	output, err := Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "cap1"},
			{Workspace: "default", Name: "cap2"},
		},
		StoreAs: &ComposeStoreAs{
			Workspace: "composed",
			Name:      "bundle",
			Mode:      StoreModeError,
		},
	})
	if err != nil {
		t.Fatalf("Compose failed: %v", err)
	}

	if output.Stored == nil {
		t.Fatal("Stored should not be nil when store_as provided")
	}
	if output.Stored.ID == "" {
		t.Error("Stored.ID should not be empty")
	}
	if output.Stored.FetchKey.MossCapsule != "bundle" {
		t.Errorf("FetchKey.MossCapsule = %q, want 'bundle'", output.Stored.FetchKey.MossCapsule)
	}
	if output.Stored.FetchKey.MossWorkspace != "composed" {
		t.Errorf("FetchKey.MossWorkspace = %q, want 'composed'", output.Stored.FetchKey.MossWorkspace)
	}

	// Verify the stored capsule exists
	fetched, err := Fetch(database, FetchInput{
		Workspace: "composed",
		Name:      "bundle",
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if fetched.CapsuleText != output.BundleText {
		t.Error("Fetched capsule text should match bundle text")
	}
}

func TestCompose_StoreAs_ModeReplace(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("cap1"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store cap1 failed: %v", err)
	}

	// First compose and store
	_, err = Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "cap1"},
		},
		StoreAs: &ComposeStoreAs{
			Workspace: "composed",
			Name:      "bundle",
		},
	})
	if err != nil {
		t.Fatalf("First Compose failed: %v", err)
	}

	// Second compose with replace mode - should succeed
	output, err := Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "cap1"},
		},
		StoreAs: &ComposeStoreAs{
			Workspace: "composed",
			Name:      "bundle",
			Mode:      StoreModeReplace,
		},
	})
	if err != nil {
		t.Fatalf("Second Compose with replace failed: %v", err)
	}
	if output.Stored == nil {
		t.Error("Stored should not be nil")
	}
}

func TestCompose_StoreAs_NameCollision_ModeError(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("cap1"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store cap1 failed: %v", err)
	}

	// First compose and store
	_, err = Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "cap1"},
		},
		StoreAs: &ComposeStoreAs{
			Workspace: "composed",
			Name:      "bundle",
		},
	})
	if err != nil {
		t.Fatalf("First Compose failed: %v", err)
	}

	// Second compose with error mode (default) - should fail
	_, err = Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "cap1"},
		},
		StoreAs: &ComposeStoreAs{
			Workspace: "composed",
			Name:      "bundle",
			Mode:      StoreModeError,
		},
	})
	if err == nil {
		t.Fatal("Second Compose with error mode should fail on name collision")
	}
	if !errors.Is(err, errors.ErrNameAlreadyExists) {
		t.Errorf("error = %v, want ErrNameAlreadyExists", err)
	}
}

func TestCompose_DisplayNamePriority(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Case 1: Title takes priority
	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("cap-with-title"),
		Title:       stringPtr("My Custom Title"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Case 2: Name when no title
	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("cap-name-only"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Case 3: ID when no name or title
	idOnly, err := Store(database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Compose all three
	output, err := Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "cap-with-title"},
			{Workspace: "default", Name: "cap-name-only"},
			{ID: idOnly.ID},
		},
	})
	if err != nil {
		t.Fatalf("Compose failed: %v", err)
	}

	// Verify display names in output
	if !strings.Contains(output.BundleText, "## My Custom Title") {
		t.Error("Should use title when available")
	}
	if !strings.Contains(output.BundleText, "## cap-name-only") {
		t.Error("Should use name when no title")
	}
	if !strings.Contains(output.BundleText, "## "+idOnly.ID) {
		t.Error("Should use ID when no name or title")
	}
}

func TestCompose_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("cap1"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Try to compose with invalid format
	_, err = Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "cap1"},
		},
		Format: "xml",
	})
	if err == nil {
		t.Fatal("Compose should fail with invalid format")
	}
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("error = %v, want ErrInvalidRequest", err)
	}
}

func TestCompose_JSON_WithStoreAs_Rejected(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("cap1"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Try to compose with JSON format and store_as - should be rejected
	_, err = Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "cap1"},
		},
		Format: "json",
		StoreAs: &ComposeStoreAs{
			Workspace: "composed",
			Name:      "bundle",
		},
	})
	if err == nil {
		t.Fatal("Compose should reject format:json with store_as")
	}
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("error = %v, want ErrInvalidRequest", err)
	}
	if !strings.Contains(err.Error(), "json") {
		t.Errorf("error message should mention json, got: %v", err)
	}
}

func TestCompose_DefaultFormat(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("cap1"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Compose without specifying format - should default to markdown
	output, err := Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "cap1"},
		},
	})
	if err != nil {
		t.Fatalf("Compose failed: %v", err)
	}

	// Markdown format uses ## headings
	if !strings.Contains(output.BundleText, "## ") {
		t.Error("Default format should be markdown with ## headings")
	}
}

func TestCompose_StoreAs_RequiresName(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("cap1"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Try to compose with store_as but no name
	_, err = Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "cap1"},
		},
		StoreAs: &ComposeStoreAs{
			Workspace: "composed",
			// Name is missing
		},
	})
	if err == nil {
		t.Fatal("Compose should fail when store_as.name is missing")
	}
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("error = %v, want ErrInvalidRequest", err)
	}
}

func TestCompose_StoreAs_LintFailure(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a thin capsule (allow_thin)
	thinCapsuleText := "Just some text without sections"
	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("thin-cap"),
		CapsuleText: thinCapsuleText,
		AllowThin:   true,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Try to compose and store - should fail lint
	_, err = Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "thin-cap"},
		},
		StoreAs: &ComposeStoreAs{
			Workspace: "composed",
			Name:      "bundle",
		},
	})
	if err == nil {
		t.Fatal("Compose with store_as should fail lint when content is thin")
	}
	if !errors.Is(err, errors.ErrCapsuleTooThin) {
		t.Errorf("error = %v, want ErrCapsuleTooThin", err)
	}
}

func TestCompose_DuplicateReferences(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	_, err = Store(database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("cap1"),
		Title:       stringPtr("Capsule One"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Compose with the same capsule referenced twice
	output, err := Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "cap1"},
			{Workspace: "default", Name: "cap1"},
		},
	})
	if err != nil {
		t.Fatalf("Compose failed: %v", err)
	}

	// Should produce 2 parts (duplicates allowed)
	if output.PartsCount != 2 {
		t.Errorf("PartsCount = %d, want 2", output.PartsCount)
	}

	// Both sections should be present
	count := strings.Count(output.BundleText, "## Capsule One")
	if count != 2 {
		t.Errorf("Expected 2 occurrences of '## Capsule One', got %d", count)
	}
}

func TestCompose_ReadTransaction(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store 3 capsules
	names := []string{"snap-a", "snap-b", "snap-c"}
	for _, name := range names {
		_, err := Store(database, cfg, StoreInput{
			Workspace:   "default",
			Name:        stringPtr(name),
			Title:       stringPtr("Title " + name),
			CapsuleText: validCapsuleText,
		})
		if err != nil {
			t.Fatalf("Store %s failed: %v", name, err)
		}
	}

	// Compose all three — verifies the transactional read path works
	output, err := Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{Workspace: "default", Name: "snap-a"},
			{Workspace: "default", Name: "snap-b"},
			{Workspace: "default", Name: "snap-c"},
		},
	})
	if err != nil {
		t.Fatalf("Compose failed: %v", err)
	}

	if output.PartsCount != 3 {
		t.Errorf("PartsCount = %d, want 3", output.PartsCount)
	}

	// All three titles should appear in the bundle
	for _, name := range names {
		expected := "## Title " + name
		if !strings.Contains(output.BundleText, expected) {
			t.Errorf("BundleText should contain %q", expected)
		}
	}
}

func TestCompose_AmbiguousAddressing(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Try to compose with ambiguous ref (both ID and name)
	_, err = Compose(database, cfg, ComposeInput{
		Items: []ComposeRef{
			{ID: "some-id", Name: "some-name"},
		},
	})
	if err == nil {
		t.Fatal("Compose should fail with ambiguous addressing")
	}
	// The error should mention the ambiguous addressing
	if !strings.Contains(err.Error(), "AMBIGUOUS_ADDRESSING") {
		t.Errorf("error = %v, should mention AMBIGUOUS_ADDRESSING", err)
	}
}
