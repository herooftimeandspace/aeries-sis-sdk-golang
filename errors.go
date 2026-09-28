package aeries

import "fmt"

// ConfigError explains why a client configuration value is invalid.
type ConfigError struct {
	Field   string
	Message string
}

// Error returns a stable human-readable validation message for configuration failures.
func (e *ConfigError) Error() string {
	if e == nil {
		return ""
	}
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// APIError captures a non-success HTTP response from the Aeries API.
type APIError struct {
	StatusCode int
	Method     string
	Path       string
	// Message contains at most a small, sanitized provider detail. It never contains the complete response body.
	Message string
	// Body is retained for source compatibility and is always empty so raw provider payloads cannot escape through errors.
	// Deprecated: use Message for the optional bounded, sanitized provider detail.
	Body string
}

// Error returns a message that includes the HTTP status and any API-provided detail.
func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return fmt.Sprintf("%s %s failed with status %d: %s", e.Method, e.Path, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%s %s failed with status %d", e.Method, e.Path, e.StatusCode)
}

// ResponseTooLargeError reports that a response exceeded the configured safe read limit.
// It deliberately retains no response bytes, full URL, query values, or request headers.
type ResponseTooLargeError struct {
	StatusCode int
	Method     string
	Path       string
	Limit      int64
	Retryable  bool
}

// Error returns bounded diagnostic metadata without including any response content.
func (e *ResponseTooLargeError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s %s failed with status %d: response exceeded %d byte limit", e.Method, e.Path, e.StatusCode, e.Limit)
}

// ValidationError explains why a typed request or helper argument cannot be used safely.
type ValidationError struct {
	Message string
}

// Error returns the validation detail in plain English.
func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// ResponseDecodeError reports a success response whose body was not valid JSON.
//
// The transport already observed the HTTP status when the decode failed, so the
// error keeps it rather than dropping it. Like the other transport errors, it
// retains no response bytes, no full URL, no query values, and no headers.
type ResponseDecodeError struct {
	StatusCode int
	Method     string
	Path       string
	// Message describes the decoding failure. It contains the JSON decoder's own
	// complaint and never the response body.
	Message string
}

// Error returns the decode failure together with the status the response carried.
func (e *ResponseDecodeError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("response for %s %s with status %d was not valid JSON: %s", e.Method, e.Path, e.StatusCode, e.Message)
}
