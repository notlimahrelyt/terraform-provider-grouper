package client

import (
	"errors"
	"fmt"
)

// Sentinel errors let Terraform resource code distinguish failure classes
// (for example, treating not-found as "remove from state" rather than an error)
// without inspecting HTTP status codes directly. Wrap these with %w to preserve
// context; callers use errors.Is to match.
var (
	// ErrNotFound indicates the requested Grouper object does not exist.
	ErrNotFound = errors.New("grouper object not found")
	// ErrUnauthorized indicates authentication failed (HTTP 401).
	ErrUnauthorized = errors.New("grouper authentication failed")
	// ErrForbidden indicates the caller lacks privileges (HTTP 403).
	ErrForbidden = errors.New("grouper authorization failed")
	// ErrConflict indicates the object already exists or a state conflict (HTTP 409).
	ErrConflict = errors.New("grouper object conflict")
	// ErrUnexpectedResponse indicates an HTTP status or body the client could not interpret.
	ErrUnexpectedResponse = errors.New("unexpected grouper response")
)

// APIError carries sanitized details about a failed Grouper call. It never
// includes credentials, authorization headers, cookies, or full response
// bodies that may contain sensitive subject data.
type APIError struct {
	// Operation is a short description of what was attempted, e.g. "get stem".
	Operation string
	// Method is the HTTP method used.
	Method string
	// Path is the sanitized request path (no query secrets).
	Path string
	// StatusCode is the HTTP status code returned, if any.
	StatusCode int
	// GrouperResultCode is the Grouper application-level result code, if decoded.
	GrouperResultCode string
	// Message is a short, sanitized human-readable summary.
	Message string
	// Err is the wrapped underlying/sentinel error.
	Err error
}

func (e *APIError) Error() string {
	switch {
	case e.GrouperResultCode != "" && e.StatusCode != 0:
		return fmt.Sprintf("%s: HTTP %d, Grouper result %q: %s", e.Operation, e.StatusCode, e.GrouperResultCode, e.Message)
	case e.StatusCode != 0:
		return fmt.Sprintf("%s: HTTP %d: %s", e.Operation, e.StatusCode, e.Message)
	default:
		return fmt.Sprintf("%s: %s", e.Operation, e.Message)
	}
}

// Unwrap exposes the wrapped sentinel so errors.Is/As work through APIError.
func (e *APIError) Unwrap() error { return e.Err }

// IsNotFound reports whether err represents a Grouper not-found condition.
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

// IsUnauthorized reports whether err represents a Grouper authentication failure.
func IsUnauthorized(err error) bool { return errors.Is(err, ErrUnauthorized) }

// IsForbidden reports whether err represents a Grouper authorization failure.
func IsForbidden(err error) bool { return errors.Is(err, ErrForbidden) }

// IsConflict reports whether err represents a Grouper object conflict.
func IsConflict(err error) bool { return errors.Is(err, ErrConflict) }

// classifyStatus maps an HTTP status code to the appropriate sentinel error.
// It returns nil for 2xx responses. Grouper can also report failures inside a
// 200 body via result metadata; that is handled separately by the caller once
// the response envelope shape is confirmed during API discovery.
func classifyStatus(status int) error {
	switch {
	case status >= 200 && status < 300:
		return nil
	case status == 401:
		return ErrUnauthorized
	case status == 403:
		return ErrForbidden
	case status == 404:
		return ErrNotFound
	case status == 409:
		return ErrConflict
	default:
		return ErrUnexpectedResponse
	}
}
