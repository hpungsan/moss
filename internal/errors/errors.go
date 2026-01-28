package errors

import (
	stderrors "errors"
	"fmt"
)

// ErrorCode represents a Moss error code.
type ErrorCode string

const (
	ErrAmbiguousAddressing ErrorCode = "AMBIGUOUS_ADDRESSING" // 400
	ErrInvalidRequest      ErrorCode = "INVALID_REQUEST"      // 400
	ErrNotFound            ErrorCode = "NOT_FOUND"            // 404
	ErrNameAlreadyExists   ErrorCode = "NAME_ALREADY_EXISTS"  // 409
	ErrConflict            ErrorCode = "CONFLICT"             // 409 (for future optimistic concurrency)
	ErrCapsuleTooLarge     ErrorCode = "CAPSULE_TOO_LARGE"    // 413
	ErrFileTooLarge        ErrorCode = "FILE_TOO_LARGE"       // 413
	ErrComposeTooLarge     ErrorCode = "COMPOSE_TOO_LARGE"    // 413
	ErrCapsuleTooThin      ErrorCode = "CAPSULE_TOO_THIN"     // 422
	ErrCancelled           ErrorCode = "CANCELLED"            // 499
	ErrInternal            ErrorCode = "INTERNAL"             // 500
)

// MossError represents a structured error with code, status, and details.
type MossError struct {
	Code    ErrorCode
	Status  int
	Message string
	Details map[string]any
}

// Error implements the error interface.
func (e *MossError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewAmbiguousAddressing creates a 400 error for when both ID and name are provided.
func NewAmbiguousAddressing() *MossError {
	return &MossError{
		Code:    ErrAmbiguousAddressing,
		Status:  400,
		Message: "cannot specify both id and name; use one addressing mode",
	}
}

// NewInvalidRequest creates a 400 error for invalid request parameters.
func NewInvalidRequest(msg string) *MossError {
	return &MossError{
		Code:    ErrInvalidRequest,
		Status:  400,
		Message: msg,
	}
}

// NewNotFound creates a 404 error for when a capsule cannot be found.
func NewNotFound(identifier string) *MossError {
	return &MossError{
		Code:    ErrNotFound,
		Status:  404,
		Message: fmt.Sprintf("capsule not found: %s", identifier),
		Details: map[string]any{"identifier": identifier},
	}
}

// NewFileNotFound creates a 404 error for when a local file path cannot be found.
func NewFileNotFound(path string) *MossError {
	return &MossError{
		Code:    ErrNotFound,
		Status:  404,
		Message: fmt.Sprintf("file not found: %s", path),
		Details: map[string]any{"path": path},
	}
}

// NewNameAlreadyExists creates a 409 error for name collisions.
func NewNameAlreadyExists(workspace, name string) *MossError {
	return &MossError{
		Code:    ErrNameAlreadyExists,
		Status:  409,
		Message: fmt.Sprintf("capsule with name %q already exists in workspace %q", name, workspace),
		Details: map[string]any{"workspace": workspace, "name": name},
	}
}

// NewConflict creates a 409 error for general conflicts.
func NewConflict(msg string) *MossError {
	return &MossError{
		Code:    ErrConflict,
		Status:  409,
		Message: msg,
	}
}

// NewCapsuleTooLarge creates a 413 error when capsule exceeds size limit.
func NewCapsuleTooLarge(max, actual int) *MossError {
	return &MossError{
		Code:    ErrCapsuleTooLarge,
		Status:  413,
		Message: fmt.Sprintf("capsule exceeds maximum size: %d chars (max %d)", actual, max),
		Details: map[string]any{"max_chars": max, "actual_chars": actual},
	}
}

// NewFileTooLarge creates a 413 error when a file exceeds size limit.
func NewFileTooLarge(maxBytes, actualBytes int64) *MossError {
	return &MossError{
		Code:    ErrFileTooLarge,
		Status:  413,
		Message: fmt.Sprintf("file exceeds maximum size: %d bytes (max %d)", actualBytes, maxBytes),
		Details: map[string]any{"max_bytes": maxBytes, "actual_bytes": actualBytes},
	}
}

// NewComposeTooLarge creates a 413 error when composed bundle exceeds size limit.
func NewComposeTooLarge(max, actual int) *MossError {
	return &MossError{
		Code:    ErrComposeTooLarge,
		Status:  413,
		Message: fmt.Sprintf("composed bundle exceeds maximum size: %d chars (max %d)", actual, max),
		Details: map[string]any{"max_chars": max, "actual_chars": actual},
	}
}

// NewCapsuleTooThin creates a 422 error when capsule is missing required sections.
func NewCapsuleTooThin(missing []string) *MossError {
	return &MossError{
		Code:    ErrCapsuleTooThin,
		Status:  422,
		Message: fmt.Sprintf("capsule missing required sections: %v", missing),
		Details: map[string]any{"missing_sections": missing},
	}
}

// NewCancelled creates a 499 error for context cancellation.
func NewCancelled(operation string) *MossError {
	return &MossError{
		Code:    ErrCancelled,
		Status:  499,
		Message: fmt.Sprintf("%s cancelled", operation),
	}
}

// NewInternal creates a 500 error for unexpected internal errors.
// The underlying error is stored in Details for logging but not exposed in Message.
func NewInternal(err error) *MossError {
	details := map[string]any{}
	if err != nil {
		details["internal_error"] = err.Error()
	}
	return &MossError{
		Code:    ErrInternal,
		Status:  500,
		Message: "an internal error occurred",
		Details: details,
	}
}

// Is checks if an error (or any wrapped error in its chain) is a MossError with the given code.
func Is(err error, code ErrorCode) bool {
	var mErr *MossError
	if stderrors.As(err, &mErr) {
		return mErr.Code == code
	}
	return false
}
