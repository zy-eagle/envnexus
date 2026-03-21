package errors

import (
	"fmt"
)

// ErrorCode defines the standard error codes for EnvNexus
type ErrorCode string

const (
	// Common Errors
	ErrInternalServer ErrorCode = "internal_server_error"
	ErrBadRequest     ErrorCode = "bad_request"
	ErrUnauthorized   ErrorCode = "unauthorized"
	ErrForbidden      ErrorCode = "forbidden"
	ErrNotFound       ErrorCode = "not_found"
	ErrConflict       ErrorCode = "conflict"

	// Agent Core Errors (from proposal 12.11.11)
	ErrBootstrap  ErrorCode = "bootstrap_error"
	ErrConfig     ErrorCode = "config_error"
	ErrDevice     ErrorCode = "device_error"
	ErrSession    ErrorCode = "session_error"
	ErrPolicy     ErrorCode = "policy_error"
	ErrApproval   ErrorCode = "approval_error"
	ErrTool       ErrorCode = "tool_error"
	ErrGovernance ErrorCode = "governance_error"
	ErrTransport  ErrorCode = "transport_error"
)

// EnvNexusError is the standard error structure
type EnvNexusError struct {
	Code      ErrorCode              `json:"code"`
	Message   string                 `json:"message"`
	Retryable bool                   `json:"retryable"`
	Temporary bool                   `json:"temporary"`
	Cause     error                  `json:"-"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func (e *EnvNexusError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// New creates a new EnvNexusError
func New(code ErrorCode, message string) *EnvNexusError {
	return &EnvNexusError{
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error with an EnvNexusError
func Wrap(code ErrorCode, message string, cause error) *EnvNexusError {
	return &EnvNexusError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// WithRetryable sets the retryable flag
func (e *EnvNexusError) WithRetryable(retryable bool) *EnvNexusError {
	e.Retryable = retryable
	return e
}

// WithTemporary sets the temporary flag
func (e *EnvNexusError) WithTemporary(temporary bool) *EnvNexusError {
	e.Temporary = temporary
	return e
}

// WithMetadata adds metadata to the error
func (e *EnvNexusError) WithMetadata(key string, value interface{}) *EnvNexusError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}
