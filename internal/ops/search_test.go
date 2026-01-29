package ops

import (
	"context"
	"strings"
	"testing"

	"github.com/hpungsan/moss/internal/config"
	"github.com/hpungsan/moss/internal/db"
	"github.com/hpungsan/moss/internal/errors"
)

func TestSearch_BasicMatch(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule with specific content
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("jwt-auth"),
		CapsuleText: validCapsuleText, // Contains "JWT" and "authentication"
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Search for "JWT"
	output, err := Search(context.Background(), database, SearchInput{
		Query: "JWT",
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(output.Items))
	}
	if output.Sort != "relevance" {
		t.Errorf("Sort = %q, want 'relevance'", output.Sort)
	}
	if output.Items[0].Snippet == "" {
		t.Error("Snippet should not be empty")
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	_, err = Search(context.Background(), database, SearchInput{
		Query: "",
	})
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("Search should return ErrInvalidRequest for empty query, got: %v", err)
	}
}

func TestSearch_WhitespaceOnlyQuery(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	_, err = Search(context.Background(), database, SearchInput{
		Query: "   \t\n  ",
	})
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("Search should return ErrInvalidRequest for whitespace query, got: %v", err)
	}
}

func TestSearch_WorkspaceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store in different workspaces
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "alpha",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "beta",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Search with workspace filter
	workspace := "alpha"
	output, err := Search(context.Background(), database, SearchInput{
		Query:     "authentication",
		Workspace: &workspace,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(output.Items))
	}
	if output.Items[0].WorkspaceNorm != "alpha" {
		t.Errorf("WorkspaceNorm = %q, want 'alpha'", output.Items[0].WorkspaceNorm)
	}
}

func TestSearch_TagFilter(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store with tag
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
		Tags:        []string{"important"},
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Store without tag
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Search with tag filter
	tag := "important"
	output, err := Search(context.Background(), database, SearchInput{
		Query: "authentication",
		Tag:   &tag,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(output.Items))
	}
}

func TestSearch_PhaseRoleFilters(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store with phase/role
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
		Phase:       stringPtr("review"),
		Role:        stringPtr("security"),
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Store with different phase
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
		Phase:       stringPtr("implement"),
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Search with phase filter
	phase := "review"
	output, err := Search(context.Background(), database, SearchInput{
		Query: "authentication",
		Phase: &phase,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(output.Items))
	}
	if output.Items[0].Phase == nil || *output.Items[0].Phase != "review" {
		t.Errorf("Phase = %v, want 'review'", output.Items[0].Phase)
	}
}

func TestSearch_Pagination(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store 5 capsules
	for i := 0; i < 5; i++ {
		_, err := Store(context.Background(), database, cfg, StoreInput{
			Workspace:   "default",
			CapsuleText: validCapsuleText,
		})
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}
	}

	// First page
	page1, err := Search(context.Background(), database, SearchInput{
		Query:  "authentication",
		Limit:  2,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("Search page 1 failed: %v", err)
	}

	if len(page1.Items) != 2 {
		t.Errorf("page1 len = %d, want 2", len(page1.Items))
	}
	if page1.Pagination.Total != 5 {
		t.Errorf("Total = %d, want 5", page1.Pagination.Total)
	}
	if !page1.Pagination.HasMore {
		t.Error("HasMore = false, want true")
	}

	// Last page
	page3, err := Search(context.Background(), database, SearchInput{
		Query:  "authentication",
		Limit:  2,
		Offset: 4,
	})
	if err != nil {
		t.Fatalf("Search page 3 failed: %v", err)
	}

	if len(page3.Items) != 1 {
		t.Errorf("page3 len = %d, want 1", len(page3.Items))
	}
	if page3.Pagination.HasMore {
		t.Error("HasMore = true, want false")
	}
}

func TestSearch_LimitBounds(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule to search
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Test default limit
	output, err := Search(context.Background(), database, SearchInput{
		Query: "authentication",
		Limit: 0,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if output.Pagination.Limit != DefaultSearchLimit {
		t.Errorf("Limit = %d, want %d", output.Pagination.Limit, DefaultSearchLimit)
	}

	// Test max limit
	output, err = Search(context.Background(), database, SearchInput{
		Query: "authentication",
		Limit: 1000,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if output.Pagination.Limit != MaxSearchLimit {
		t.Errorf("Limit = %d, want %d", output.Pagination.Limit, MaxSearchLimit)
	}
}

func TestSearch_IncludeDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store and delete
	stored, err := Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	if err := db.SoftDelete(context.Background(), database, stored.ID); err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Store active
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Without include_deleted
	output, err := Search(context.Background(), database, SearchInput{
		Query:          "authentication",
		IncludeDeleted: false,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if output.Pagination.Total != 1 {
		t.Errorf("Total = %d, want 1", output.Pagination.Total)
	}

	// With include_deleted
	output, err = Search(context.Background(), database, SearchInput{
		Query:          "authentication",
		IncludeDeleted: true,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if output.Pagination.Total != 2 {
		t.Errorf("Total = %d, want 2", output.Pagination.Total)
	}
}

func TestSearch_PhraseQuery(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store capsules with different content
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText, // Contains "user authentication system"
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Phrase search
	output, err := Search(context.Background(), database, SearchInput{
		Query: "\"user authentication\"",
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should find the capsule
	if len(output.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(output.Items))
	}
}

func TestSearch_PrefixQuery(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText, // Contains "authentication"
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Prefix search
	output, err := Search(context.Background(), database, SearchInput{
		Query: "auth*",
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(output.Items))
	}
}

func TestSearch_ORQuery(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store with "JWT"
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText, // Contains JWT
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Store with different content (using allow_thin to customize content)
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: "## Objective\nBuild OAuth login.\n## Current status\nResearching OAuth providers.",
		AllowThin:   true,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// OR query
	output, err := Search(context.Background(), database, SearchInput{
		Query: "JWT OR OAuth",
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(output.Items) != 2 {
		t.Errorf("len(Items) = %d, want 2", len(output.Items))
	}
}

func TestSearch_TitleSearch(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store with explicit title
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("session-mgmt"),
		Title:       stringPtr("Redis Session Management"),
		CapsuleText: validCapsuleText, // Does not contain "Redis"
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Search for title content
	output, err := Search(context.Background(), database, SearchInput{
		Query: "Redis",
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(output.Items))
	}
}

func TestSearch_NoResults(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Search for non-existent term
	output, err := Search(context.Background(), database, SearchInput{
		Query: "nonexistentterm12345",
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should return empty array, not error
	if len(output.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(output.Items))
	}
	if output.Pagination.Total != 0 {
		t.Errorf("Total = %d, want 0", output.Pagination.Total)
	}
	if output.Items == nil {
		t.Error("Items should be empty array, not nil")
	}
}

func TestSearch_FetchKeyIncluded(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store named capsule
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "myworkspace",
		Name:        stringPtr("myauth"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	output, err := Search(context.Background(), database, SearchInput{
		Query: "authentication",
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(output.Items))
	}

	// Verify fetch_key
	item := output.Items[0]
	if item.FetchKey.MossCapsule != "myauth" {
		t.Errorf("FetchKey.MossCapsule = %q, want 'myauth'", item.FetchKey.MossCapsule)
	}
	if item.FetchKey.MossWorkspace != "myworkspace" {
		t.Errorf("FetchKey.MossWorkspace = %q, want 'myworkspace'", item.FetchKey.MossWorkspace)
	}
}

func TestSearch_SnippetTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store with long content
	longText := "## Objective\n" + strings.Repeat("authentication ", 100) + "\n## Current status\nDone."
	_, err = Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		CapsuleText: longText,
		AllowThin:   true,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	output, err := Search(context.Background(), database, SearchInput{
		Query: "authentication",
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(output.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(output.Items))
	}

	// Snippet should be truncated to ~300 chars
	snippet := output.Items[0].Snippet
	if len(snippet) > MaxSnippetChars+10 { // Allow for ellipsis and word boundary
		t.Errorf("Snippet length = %d, want <= %d", len(snippet), MaxSnippetChars+10)
	}
}

func TestSearch_TriggerSync(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule
	stored, err := Store(context.Background(), database, cfg, StoreInput{
		Workspace:   "default",
		Name:        stringPtr("trigger-test"),
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Search should find it
	output, err := Search(context.Background(), database, SearchInput{
		Query: "JWT",
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(output.Items) != 1 {
		t.Errorf("After store: len(Items) = %d, want 1", len(output.Items))
	}

	// Update the capsule with different content
	_, err = Update(context.Background(), database, cfg, UpdateInput{
		ID:          stored.ID,
		CapsuleText: stringPtr("## Objective\nBuild Redis cache.\n## Current status\nPlanning."),
		AllowThin:   true,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Search for old term should not find it
	output, err = Search(context.Background(), database, SearchInput{
		Query: "JWT",
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(output.Items) != 0 {
		t.Errorf("After update (old term): len(Items) = %d, want 0", len(output.Items))
	}

	// Search for new term should find it
	output, err = Search(context.Background(), database, SearchInput{
		Query: "Redis",
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(output.Items) != 1 {
		t.Errorf("After update (new term): len(Items) = %d, want 1", len(output.Items))
	}

	// Delete the capsule
	_, err = Delete(context.Background(), database, DeleteInput{
		ID: stored.ID,
	})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Search should not find it (without include_deleted)
	output, err = Search(context.Background(), database, SearchInput{
		Query:          "Redis",
		IncludeDeleted: false,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(output.Items) != 0 {
		t.Errorf("After delete: len(Items) = %d, want 0", len(output.Items))
	}
}

func TestSearch_FTS5SyntaxErrors(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Init(tmpDir)
	if err != nil {
		t.Fatalf("db.Init failed: %v", err)
	}
	defer database.Close()

	cfg := config.DefaultConfig()

	// Store a capsule first
	_, err = Store(context.Background(), database, cfg, StoreInput{
		CapsuleText: validCapsuleText,
	})
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// All these invalid queries should return INVALID_REQUEST, not INTERNAL
	invalidQueries := []string{
		`"unclosed quote`, // unterminated string
		`(unclosed paren`, // syntax error
		`AND`,             // standalone operator
		`OR`,              // standalone operator
		`NOT`,             // standalone operator
		`* `,              // wildcard only
		`test AND`,        // trailing operator
	}

	for _, query := range invalidQueries {
		_, err := Search(context.Background(), database, SearchInput{
			Query: query,
		})
		if err == nil {
			t.Errorf("Query %q: expected error, got nil", query)
			continue
		}
		if !errors.Is(err, errors.ErrInvalidRequest) {
			t.Errorf("Query %q: expected INVALID_REQUEST, got %v", query, err)
		}
	}
}

func TestTruncateSnippet(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxChars int
		wantMax  int
		wantEnd  string
	}{
		{
			name:     "short string unchanged",
			input:    "hello world",
			maxChars: 300,
			wantMax:  11,
		},
		{
			name:     "truncates at word boundary",
			input:    "hello world this is a test",
			maxChars: 15,
			wantMax:  15,
			wantEnd:  "...",
		},
		{
			name:     "exact length unchanged",
			input:    "hello",
			maxChars: 5,
			wantMax:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateSnippet(tt.input, tt.maxChars)
			if len(result) > tt.wantMax+3 { // +3 for ellipsis
				t.Errorf("truncateSnippet(%q, %d) length = %d, want <= %d",
					tt.input, tt.maxChars, len(result), tt.wantMax+3)
			}
			if tt.wantEnd != "" && !strings.HasSuffix(result, tt.wantEnd) {
				t.Errorf("truncateSnippet(%q, %d) = %q, want suffix %q",
					tt.input, tt.maxChars, result, tt.wantEnd)
			}
		})
	}
}
