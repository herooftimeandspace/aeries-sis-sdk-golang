package aeries

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	defaultUserAgent              = "aeries-sis-sdk-golang/0.1.1"
	defaultTimeout                = 30 * time.Second
	defaultMaxRetries             = 2
	defaultRetryBackoff           = 300 * time.Millisecond
	defaultMaxResponseBytes int64 = 32 << 20
	maxResponseBytesLimit   int64 = 1 << 40
)

var certificatePattern = regexp.MustCompile(`^[A-Za-z0-9]{32}$`)

// Config describes how the SDK should connect to one district's Aeries instance.
type Config struct {
	BaseURL      string
	Certificate  string
	HTTPClient   *http.Client
	UserAgent    string
	Timeout      time.Duration
	MaxRetries   int
	RetryBackoff time.Duration
	// MaxResponseBytes limits every response body before error handling or JSON decoding. Zero uses the 32 MiB default.
	MaxResponseBytes    int64
	DefaultDatabaseYear string
	// PortalRoot overrides the portal root inferred from BaseURL's path. A nil value keeps the
	// inferred behaviour. A non-nil value is used verbatim, so a pointer to the empty string means
	// "the portal root is the domain root" and suppresses the historical /aeries default entirely.
	PortalRoot *string
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
	trimmed, err := c.portalRoot(parsed.Path)
	if err != nil {
		return "", err
	}
	parsed.Path = trimmed
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimSuffix(parsed.String(), "/"), nil
}

// portalRoot resolves the portal root to use for the supplied BaseURL path, honouring an explicit
// Config.PortalRoot override and applying the historical /aeries default only when the caller
// supplied no path at all.
func (c Config) portalRoot(rawPath string) (string, error) {
	if c.PortalRoot != nil {
		override := strings.TrimSpace(*c.PortalRoot)
		if override == "" || override == "/" {
			return "", nil
		}
		if !strings.HasPrefix(override, "/") {
			return "", &ConfigError{Field: "PortalRoot", Message: "must begin with /"}
		}
		if parsed, err := url.Parse(override); err != nil || parsed.Scheme != "" || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return "", &ConfigError{Field: "PortalRoot", Message: "must be a URL path, not an absolute URL"}
		}
		return strings.TrimSuffix(override, "/"), nil
	}
	// Only a caller who supplied no path at all gets the historical /aeries default. A supplied path
	// that normalizes to the root is an explicit statement that the portal root is the domain root.
	if suppliedPath := strings.TrimSpace(rawPath); suppliedPath == "" || suppliedPath == "/" {
		return "/aeries", nil
	}
	return normalizePortalRoot(rawPath), nil
}

// normalizePortalRoot preserves an explicit portal root and strips any trailing API suffix from it.
func normalizePortalRoot(rawPath string) string {
	trimmed := strings.TrimSuffix(strings.TrimSpace(rawPath), "/")
	for _, suffix := range []string{"/api/v5", "/api/v4", "/api/v3", "/api"} {
		if strings.HasSuffix(trimmed, suffix) {
			return strings.TrimSuffix(trimmed, suffix)
		}
	}
	return trimmed
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

// normalizedMaxResponseBytes fills in the conservative response-body limit when the caller does not provide one.
func (c Config) normalizedMaxResponseBytes() int64 {
	if c.MaxResponseBytes == 0 {
		return defaultMaxResponseBytes
	}
	return c.MaxResponseBytes
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
	if c.MaxResponseBytes < 0 {
		return &ConfigError{Field: "MaxResponseBytes", Message: "cannot be negative"}
	}
	if c.MaxResponseBytes > maxResponseBytesLimit {
		return &ConfigError{Field: "MaxResponseBytes", Message: "cannot exceed 1 TiB"}
	}
	return nil
}
