package errors

import (
	"fmt"
	"testing"
)

func TestMossError_Error(t *testing.T) {
	err := &MossError{
		Code:    ErrNotFound,
		Status:  404,
		Message: "capsule not found",
	}

	expected := "NOT_FOUND: capsule not found"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestNewAmbiguousAddressing(t *testing.T) {
	err := NewAmbiguousAddressing()

	if err.Code != ErrAmbiguousAddressing {
		t.Errorf("Code = %q, want %q", err.Code, ErrAmbiguousAddressing)
	}
	if err.Status != 400 {
		t.Errorf("Status = %d, want 400", err.Status)
	}
}

func TestNewInvalidRequest(t *testing.T) {
	err := NewInvalidRequest("capsule_text is required")

	if err.Code != ErrInvalidRequest {
		t.Errorf("Code = %q, want %q", err.Code, ErrInvalidRequest)
	}
	if err.Status != 400 {
		t.Errorf("Status = %d, want 400", err.Status)
	}
	if err.Message != "capsule_text is required" {
		t.Errorf("Message = %q, want %q", err.Message, "capsule_text is required")
	}
}

func TestNewNotFound(t *testing.T) {
	err := NewNotFound("auth")

	if err.Code != ErrNotFound {
		t.Errorf("Code = %q, want %q", err.Code, ErrNotFound)
	}
	if err.Status != 404 {
		t.Errorf("Status = %d, want 404", err.Status)
	}
	if err.Details["identifier"] != "auth" {
		t.Errorf("Details[identifier] = %v, want %q", err.Details["identifier"], "auth")
	}
}

func TestNewNameAlreadyExists(t *testing.T) {
	err := NewNameAlreadyExists("default", "auth")

	if err.Code != ErrNameAlreadyExists {
		t.Errorf("Code = %q, want %q", err.Code, ErrNameAlreadyExists)
	}
	if err.Status != 409 {
		t.Errorf("Status = %d, want 409", err.Status)
	}
	if err.Details["workspace"] != "default" {
		t.Errorf("Details[workspace] = %v, want %q", err.Details["workspace"], "default")
	}
	if err.Details["name"] != "auth" {
		t.Errorf("Details[name] = %v, want %q", err.Details["name"], "auth")
	}
}

func TestNewConflict(t *testing.T) {
	err := NewConflict("concurrent modification detected")

	if err.Code != ErrConflict {
		t.Errorf("Code = %q, want %q", err.Code, ErrConflict)
	}
	if err.Status != 409 {
		t.Errorf("Status = %d, want 409", err.Status)
	}
}

func TestNewCapsuleTooLarge(t *testing.T) {
	err := NewCapsuleTooLarge(12000, 15000)

	if err.Code != ErrCapsuleTooLarge {
		t.Errorf("Code = %q, want %q", err.Code, ErrCapsuleTooLarge)
	}
	if err.Status != 413 {
		t.Errorf("Status = %d, want 413", err.Status)
	}
	if err.Details["max_chars"] != 12000 {
		t.Errorf("Details[max_chars] = %v, want 12000", err.Details["max_chars"])
	}
	if err.Details["actual_chars"] != 15000 {
		t.Errorf("Details[actual_chars] = %v, want 15000", err.Details["actual_chars"])
	}
}

func TestNewFileTooLarge(t *testing.T) {
	err := NewFileTooLarge(10*1024*1024, 15*1024*1024)

	// Must use ErrFileTooLarge (not ErrCapsuleTooLarge) for file size errors
	if err.Code != ErrFileTooLarge {
		t.Errorf("Code = %q, want %q", err.Code, ErrFileTooLarge)
	}
	if err.Status != 413 {
		t.Errorf("Status = %d, want 413", err.Status)
	}
	if err.Details["max_bytes"] != int64(10*1024*1024) {
		t.Errorf("Details[max_bytes] = %v, want %v", err.Details["max_bytes"], int64(10*1024*1024))
	}
	if err.Details["actual_bytes"] != int64(15*1024*1024) {
		t.Errorf("Details[actual_bytes] = %v, want %v", err.Details["actual_bytes"], int64(15*1024*1024))
	}
}

func TestNewComposeTooLarge(t *testing.T) {
	err := NewComposeTooLarge(12000, 15000)

	if err.Code != ErrComposeTooLarge {
		t.Errorf("Code = %q, want %q", err.Code, ErrComposeTooLarge)
	}
	if err.Status != 413 {
		t.Errorf("Status = %d, want 413", err.Status)
	}
	if err.Details["max_chars"] != 12000 {
		t.Errorf("Details[max_chars] = %v, want 12000", err.Details["max_chars"])
	}
	if err.Details["actual_chars"] != 15000 {
		t.Errorf("Details[actual_chars] = %v, want 15000", err.Details["actual_chars"])
	}
}

func TestNewCapsuleTooThin(t *testing.T) {
	missing := []string{"Objective", "Next actions"}
	err := NewCapsuleTooThin(missing)

	if err.Code != ErrCapsuleTooThin {
		t.Errorf("Code = %q, want %q", err.Code, ErrCapsuleTooThin)
	}
	if err.Status != 422 {
		t.Errorf("Status = %d, want 422", err.Status)
	}
	if sections, ok := err.Details["missing_sections"].([]string); !ok || len(sections) != 2 {
		t.Errorf("Details[missing_sections] = %v, want %v", err.Details["missing_sections"], missing)
	}
}

func TestNewInternal(t *testing.T) {
	t.Run("with error", func(t *testing.T) {
		originalErr := fmt.Errorf("database connection failed")
		err := NewInternal(originalErr)

		if err.Code != ErrInternal {
			t.Errorf("Code = %q, want %q", err.Code, ErrInternal)
		}
		if err.Status != 500 {
			t.Errorf("Status = %d, want 500", err.Status)
		}
		// Message should be generic (not leak internal details)
		if err.Message != "an internal error occurred" {
			t.Errorf("Message = %q, want %q", err.Message, "an internal error occurred")
		}
		// Original error should be stored in Details for logging
		if err.Details["internal_error"] != "database connection failed" {
			t.Errorf("Details[internal_error] = %q, want %q", err.Details["internal_error"], "database connection failed")
		}
	})

	t.Run("with nil", func(t *testing.T) {
		err := NewInternal(nil)

		if err.Message != "an internal error occurred" {
			t.Errorf("Message = %q, want %q", err.Message, "an internal error occurred")
		}
		// Details should be empty but not nil
		if err.Details == nil {
			t.Error("Details should not be nil")
		}
	})
}

func TestIs(t *testing.T) {
	t.Run("matching code", func(t *testing.T) {
		err := NewNotFound("test")
		if !Is(err, ErrNotFound) {
			t.Error("Is() = false, want true")
		}
	})

	t.Run("non-matching code", func(t *testing.T) {
		err := NewNotFound("test")
		if Is(err, ErrConflict) {
			t.Error("Is() = true, want false")
		}
	})

	t.Run("non-MossError", func(t *testing.T) {
		err := fmt.Errorf("plain error")
		if Is(err, ErrNotFound) {
			t.Error("Is() = true, want false for non-MossError")
		}
	})

	t.Run("wrapped MossError", func(t *testing.T) {
		inner := NewNotFound("test")
		wrapped := fmt.Errorf("items[0]: %w", inner)
		if !Is(wrapped, ErrNotFound) {
			t.Error("Is() = false, want true for wrapped MossError")
		}
		if Is(wrapped, ErrConflict) {
			t.Error("Is() = true, want false for wrong code on wrapped MossError")
		}
	})
}
