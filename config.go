package aeries

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	defaultUserAgent    = "aeries-sis-sdk-golang/0.1.0"
	defaultTimeout      = 30 * time.Second
	defaultMaxRetries   = 2
	defaultRetryBackoff = 300 * time.Millisecond
)

var certificatePattern = regexp.MustCompile(`^[A-Za-z0-9]{32}$`)

// Config describes how the SDK should connect to one district's Aeries instance.
type Config struct {
	BaseURL             string
	Certificate         string
	HTTPClient          *http.Client
	UserAgent           string
	Timeout             time.Duration
	MaxRetries          int
	RetryBackoff        time.Duration
	DefaultDatabaseYear string
}

// normalizedBaseURL returns the canonical district root used to build API requests.
func (c Config) normalizedBaseURL() (string, error) {
	if strings.TrimSpace(c.BaseURL) == "" {
		return "", &ConfigError{Field: "BaseURL", Message: "is required"}
	}
	parsed, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", &ConfigError{Field: "BaseURL", Message: "must be a valid URL"}
	}
	if parsed.Scheme != "https" {
		return "", &ConfigError{Field: "BaseURL", Message: "must use https"}
	}
	if parsed.Host == "" {
		return "", &ConfigError{Field: "BaseURL", Message: "must include a host"}
	}
	trimmed := strings.TrimSuffix(parsed.Path, "/")
	switch {
	case trimmed == "":
		trimmed = "/aeries"
	case strings.HasSuffix(trimmed, "/aeries/api/v5"):
		trimmed = strings.TrimSuffix(trimmed, "/api/v5")
	case strings.HasSuffix(trimmed, "/aeries/api/v4"):
		trimmed = strings.TrimSuffix(trimmed, "/api/v4")
	case strings.HasSuffix(trimmed, "/aeries/api/v3"):
		trimmed = strings.TrimSuffix(trimmed, "/api/v3")
	case strings.HasSuffix(trimmed, "/aeries/api"):
		trimmed = strings.TrimSuffix(trimmed, "/api")
	case !strings.HasSuffix(trimmed, "/aeries"):
		trimmed = strings.TrimSuffix(trimmed, "/") + "/aeries"
	}
	parsed.Path = trimmed
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimSuffix(parsed.String(), "/"), nil
}

// normalizedHTTPClient fills in a safe default HTTP client when the caller does not provide one.
func (c Config) normalizedHTTPClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &http.Client{Timeout: timeout}
}

// normalizedUserAgent fills in the SDK's default user agent when the caller does not provide one.
func (c Config) normalizedUserAgent() string {
	if strings.TrimSpace(c.UserAgent) == "" {
		return defaultUserAgent
	}
	return c.UserAgent
}

// normalizedMaxRetries fills in the SDK's default retry count when the caller does not provide one.
func (c Config) normalizedMaxRetries() int {
	if c.MaxRetries < 0 {
		return -1
	}
	if c.MaxRetries == 0 {
		return defaultMaxRetries
	}
	return c.MaxRetries
}

// normalizedRetryBackoff fills in the SDK's default retry pause when the caller does not provide one.
func (c Config) normalizedRetryBackoff() time.Duration {
	if c.RetryBackoff <= 0 {
		return defaultRetryBackoff
	}
	return c.RetryBackoff
}

// validate checks that the caller provided the minimum information needed to build requests safely.
func (c Config) validate() error {
	if _, err := c.normalizedBaseURL(); err != nil {
		return err
	}
	if !certificatePattern.MatchString(c.Certificate) {
		return &ConfigError{Field: "Certificate", Message: "must be a 32 character alphanumeric Aeries certificate"}
	}
	if c.MaxRetries < 0 {
		return &ConfigError{Field: "MaxRetries", Message: "cannot be negative"}
	}
	if c.Timeout < 0 {
		return &ConfigError{Field: "Timeout", Message: "cannot be negative"}
	}
	if c.RetryBackoff < 0 {
		return &ConfigError{Field: "RetryBackoff", Message: "cannot be negative"}
	}
	return nil
}
