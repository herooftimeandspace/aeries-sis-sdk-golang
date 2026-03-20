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
	Message    string
	Body       string
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
