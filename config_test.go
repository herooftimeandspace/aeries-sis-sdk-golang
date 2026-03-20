package aeries

import (
	"strings"
	"testing"
	"time"
)

// TestConfigNormalizedBaseURLVariants verifies that common district URL forms all normalize to the API root.
func TestConfigNormalizedBaseURLVariants(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantURL string
	}{
		{name: "root host", input: "https://demo.aeries.net", wantURL: "https://demo.aeries.net/aeries"},
		{name: "aeries path", input: "https://demo.aeries.net/aeries", wantURL: "https://demo.aeries.net/aeries"},
		{name: "api v5 path", input: "https://demo.aeries.net/aeries/api/v5", wantURL: "https://demo.aeries.net/aeries"},
		{name: "trailing slash", input: "https://demo.aeries.net/aeries/", wantURL: "https://demo.aeries.net/aeries"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{BaseURL: tt.input, Certificate: testCertificate}
			got, err := config.normalizedBaseURL()
			if err != nil {
				t.Fatalf("normalizedBaseURL returned error: %v", err)
			}
			if got != tt.wantURL {
				t.Fatalf("normalizedBaseURL = %q, want %q", got, tt.wantURL)
			}
		})
	}
}

// TestConfigValidateRejectsBadValues verifies that the client refuses unsafe configuration.
func TestConfigValidateRejectsBadValues(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		wantSubstr string
	}{
		{
			name:       "missing base url",
			config:     Config{Certificate: testCertificate},
			wantSubstr: "BaseURL",
		},
		{
			name:       "http base url",
			config:     Config{BaseURL: "http://demo.aeries.net", Certificate: testCertificate},
			wantSubstr: "https",
		},
		{
			name:       "bad certificate",
			config:     Config{BaseURL: "https://demo.aeries.net", Certificate: "short"},
			wantSubstr: "Certificate",
		},
		{
			name:       "negative retries",
			config:     Config{BaseURL: "https://demo.aeries.net", Certificate: testCertificate, MaxRetries: -1},
			wantSubstr: "MaxRetries",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if err == nil || !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Fatalf("validate error = %v, want substring %q", err, tt.wantSubstr)
			}
		})
	}
}

// TestConfigNormalizedDefaults verifies that default client behavior is stable and reviewable.
func TestConfigNormalizedDefaults(t *testing.T) {
	config := Config{BaseURL: "https://demo.aeries.net", Certificate: testCertificate}
	if config.normalizedHTTPClient().Timeout != defaultTimeout {
		t.Fatalf("normalizedHTTPClient timeout = %v, want %v", config.normalizedHTTPClient().Timeout, defaultTimeout)
	}
	if config.normalizedUserAgent() != defaultUserAgent {
		t.Fatalf("normalizedUserAgent = %q, want %q", config.normalizedUserAgent(), defaultUserAgent)
	}
	if config.normalizedMaxRetries() != defaultMaxRetries {
		t.Fatalf("normalizedMaxRetries = %d, want %d", config.normalizedMaxRetries(), defaultMaxRetries)
	}
	if config.normalizedRetryBackoff() != defaultRetryBackoff {
		t.Fatalf("normalizedRetryBackoff = %v, want %v", config.normalizedRetryBackoff(), defaultRetryBackoff)
	}

	custom := Config{
		BaseURL:      "https://demo.aeries.net",
		Certificate:  testCertificate,
		UserAgent:    "custom-agent",
		MaxRetries:   4,
		RetryBackoff: 2 * time.Second,
	}
	if custom.normalizedUserAgent() != "custom-agent" {
		t.Fatalf("normalizedUserAgent did not preserve caller value")
	}
	if custom.normalizedMaxRetries() != 4 {
		t.Fatalf("normalizedMaxRetries did not preserve caller value")
	}
	if custom.normalizedRetryBackoff() != 2*time.Second {
		t.Fatalf("normalizedRetryBackoff did not preserve caller value")
	}
}
