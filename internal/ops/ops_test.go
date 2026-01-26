package ops

import (
	"testing"

	"github.com/hpungsan/moss/internal/errors"
)

func TestValidateAddress_ByID(t *testing.T) {
	addr, err := ValidateAddress("01ABC123", "", "")
	if err != nil {
		t.Fatalf("ValidateAddress failed: %v", err)
	}

	if !addr.ByID {
		t.Error("ByID = false, want true")
	}
	if addr.ID != "01ABC123" {
		t.Errorf("ID = %q, want %q", addr.ID, "01ABC123")
	}
}

func TestValidateAddress_Ambiguous_IDWithWorkspace(t *testing.T) {
	// Strict: id + workspace is ambiguous (must specify exactly one addressing mode)
	_, err := ValidateAddress("01ABC123", "myworkspace", "")
	if !errors.Is(err, errors.ErrAmbiguousAddressing) {
		t.Errorf("ValidateAddress should return ErrAmbiguousAddressing, got: %v", err)
	}
}

func TestValidateAddress_Ambiguous_IDWithBoth(t *testing.T) {
	// Strict: id + workspace + name is ambiguous
	_, err := ValidateAddress("01ABC123", "myworkspace", "myname")
	if !errors.Is(err, errors.ErrAmbiguousAddressing) {
		t.Errorf("ValidateAddress should return ErrAmbiguousAddressing, got: %v", err)
	}
}

func TestValidateAddress_ByName(t *testing.T) {
	addr, err := ValidateAddress("", "MyWorkspace", "Auth System")
	if err != nil {
		t.Fatalf("ValidateAddress failed: %v", err)
	}

	if addr.ByID {
		t.Error("ByID = true, want false")
	}
	if addr.Workspace != "myworkspace" {
		t.Errorf("Workspace = %q, want %q (normalized)", addr.Workspace, "myworkspace")
	}
	if addr.Name != "auth system" {
		t.Errorf("Name = %q, want %q (normalized)", addr.Name, "auth system")
	}
}

func TestValidateAddress_ByName_DefaultWorkspace(t *testing.T) {
	addr, err := ValidateAddress("", "", "my-capsule")
	if err != nil {
		t.Fatalf("ValidateAddress failed: %v", err)
	}

	if addr.Workspace != "default" {
		t.Errorf("Workspace = %q, want %q (default)", addr.Workspace, "default")
	}
}

func TestValidateAddress_ByName_WhitespaceWorkspaceDefaults(t *testing.T) {
	addr, err := ValidateAddress("", "   \t\n  ", "my-capsule")
	if err != nil {
		t.Fatalf("ValidateAddress failed: %v", err)
	}

	if addr.Workspace != "default" {
		t.Errorf("Workspace = %q, want %q (default)", addr.Workspace, "default")
	}
}

func TestValidateAddress_ByName_WhitespaceNameInvalid(t *testing.T) {
	_, err := ValidateAddress("", "default", "   \t\n  ")
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("ValidateAddress should return ErrInvalidRequest, got: %v", err)
	}
}

func TestValidateAddress_Ambiguous(t *testing.T) {
	_, err := ValidateAddress("01ABC123", "", "my-capsule")
	if !errors.Is(err, errors.ErrAmbiguousAddressing) {
		t.Errorf("ValidateAddress should return ErrAmbiguousAddressing, got: %v", err)
	}
}

func TestValidateAddress_Invalid_Neither(t *testing.T) {
	_, err := ValidateAddress("", "", "")
	if !errors.Is(err, errors.ErrInvalidRequest) {
		t.Errorf("ValidateAddress should return ErrInvalidRequest, got: %v", err)
	}
}

func TestBuildFetchKey_Named(t *testing.T) {
	link := BuildFetchKey("myworkspace", "auth", "01ABC123")

	if link.MossCapsule != "auth" {
		t.Errorf("MossCapsule = %q, want %q", link.MossCapsule, "auth")
	}
	if link.MossWorkspace != "myworkspace" {
		t.Errorf("MossWorkspace = %q, want %q", link.MossWorkspace, "myworkspace")
	}
	if link.MossID != "" {
		t.Errorf("MossID = %q, want empty (named capsule)", link.MossID)
	}
}

func TestBuildFetchKey_Unnamed(t *testing.T) {
	link := BuildFetchKey("default", "", "01ABC123")

	if link.MossID != "01ABC123" {
		t.Errorf("MossID = %q, want %q", link.MossID, "01ABC123")
	}
	if link.MossCapsule != "" {
		t.Errorf("MossCapsule = %q, want empty (unnamed capsule)", link.MossCapsule)
	}
	if link.MossWorkspace != "" {
		t.Errorf("MossWorkspace = %q, want empty (unnamed capsule)", link.MossWorkspace)
	}
}
