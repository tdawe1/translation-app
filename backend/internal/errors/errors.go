// Package errors provides typed error values for the API
package errors

// ErrorCode represents a unique error code
type ErrorCode string

const (
	// Request validation errors
	ErrInvalidRequest  ErrorCode = "INVALID_REQUEST"
	ErrWeakPassword    ErrorCode = "WEAK_PASSWORD"
	ErrInvalidUserID   ErrorCode = "INVALID_USER_ID"

	// Authentication errors
	ErrNotAuthenticated ErrorCode = "NOT_AUTHENTICATED"
	ErrInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	ErrInactiveUser      ErrorCode = "INACTIVE_USER"
	ErrTokenError        ErrorCode = "TOKEN_ERROR"

	// User errors
	ErrUserExists    ErrorCode = "USER_EXISTS"
	ErrUserNotFound  ErrorCode = "USER_NOT_FOUND"
	ErrCreateError   ErrorCode = "CREATE_ERROR"
	ErrUpdateError   ErrorCode = "UPDATE_ERROR"
	ErrDeleteError   ErrorCode = "DELETE_ERROR"
	ErrConfigError   ErrorCode = "CONFIG_ERROR"
	ErrStateError    ErrorCode = "STATE_ERROR"

	// Database errors
	ErrDatabase    ErrorCode = "DATABASE_ERROR"
	ErrCommitError ErrorCode = "COMMIT_ERROR"
	ErrPasswordError ErrorCode = "PASSWORD_ERROR"

	// Internal errors
	ErrInternal ErrorCode = "INTERNAL_ERROR"
)

// APIError represents a structured error response
type APIError struct {
	Code    ErrorCode            `json:"code"`
	Message string               `json:"error"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return e.Message
}

// New creates a new APIError
func New(code ErrorCode, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

// WithDetails adds details to the error
func (e *APIError) WithDetails(details map[string]interface{}) *APIError {
	e.Details = details
	return e
}
