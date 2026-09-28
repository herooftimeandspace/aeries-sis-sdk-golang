package aeries

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

type stubNetError struct {
	timeout bool
}

// Error returns a stable message for the fake network error used by coverage-oriented tests.
func (e stubNetError) Error() string {
	return "temporary network issue"
}

// Timeout reports whether the fake network error should behave like a timeout.
func (e stubNetError) Timeout() bool {
	return e.timeout
}

// Temporary reports that the fake network error is transient.
func (e stubNetError) Temporary() bool {
	return true
}

// TestErrorRenderers exercises the user-facing error strings for the SDK's typed error values.
func TestErrorRenderers(t *testing.T) {
	var nilConfigError *ConfigError
	var nilAPIError *APIError
	var nilValidationError *ValidationError
	if nilConfigError.Error() != "" || nilAPIError.Error() != "" || nilValidationError.Error() != "" {
		t.Fatal("nil error receivers should render empty strings")
	}
	if (&ConfigError{Field: "BaseURL", Message: "is required"}).Error() != "BaseURL: is required" {
		t.Fatal("ConfigError with field rendered unexpectedly")
	}
	if (&ConfigError{Message: "plain"}).Error() != "plain" {
		t.Fatal("ConfigError without field rendered unexpectedly")
	}
	if (&APIError{StatusCode: 400, Method: "GET", Path: "/api/v5/systeminfo", Message: "bad"}).Error() == "" {
		t.Fatal("APIError should render a message")
	}
	if (&ValidationError{Message: "nope"}).Error() != "nope" {
		t.Fatal("ValidationError rendered unexpectedly")
	}
	if !strings.Contains((&APIError{StatusCode: 400, Method: "GET", Path: "/x"}).Error(), "status 400") {
		t.Fatal("APIError without message should still include the status")
	}
}

// TestClientHelperBranches exercises small helper branches that are easy to miss in higher-level tests.
func TestClientHelperBranches(t *testing.T) {
	if _, err := encodeBody(make(chan int)); err == nil {
		t.Fatal("encodeBody should reject unsupported JSON values")
	}
	if !isRetriable(stubNetError{timeout: true}) {
		t.Fatal("expected net.Error to be retriable")
	}
	if isRetriable(context.Canceled) {
		t.Fatal("context cancellation should not be retriable")
	}
	if err := sleepWithContext(func() context.Context {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}(), time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("sleepWithContext error = %v", err)
	}
	query := map[string]string{}
	addQueryValue(query, "code", "")
	addIntQueryValue(query, "StartingRecord", 0)
	if len(query) != 0 {
		t.Fatalf("expected empty query, got %#v", query)
	}
	addQueryValue(query, "code", "P1")
	addIntQueryValue(query, "StartingRecord", 10)
	if query["code"] != "P1" || query["StartingRecord"] != "10" {
		t.Fatalf("unexpected query values %#v", query)
	}
	if cloneMap(nil) == nil {
		t.Fatal("cloneMap should always return a map")
	}
	if copied := cloneMap(map[string]string{"a": "b"}); copied["a"] != "b" {
		t.Fatalf("cloneMap copied = %#v", copied)
	}
	if !strings.Contains(stubNetError{timeout: true}.Error(), "network") {
		t.Fatal("stubNetError should provide a readable message")
	}
}

// TestClientBranchCoverage exercises low-level transport branches that are not specific to one endpoint family.
func TestClientBranchCoverage(t *testing.T) {
	client, err := NewClient(Config{
		BaseURL:      "https://district.example.test/aeries/api/v3",
		Certificate:  testCertificate,
		MaxRetries:   1,
		RetryBackoff: 1,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/aeries/api/v5/invalid-json":
				return jsonResponse(http.StatusOK, "{"), nil
			case "/aeries/api/v5/no-content":
				return noContentResponse(), nil
			case "/aeries/api/v5/bad-request":
				return jsonResponse(http.StatusBadRequest, `plain`), nil
			default:
				return jsonResponse(http.StatusOK, `[]`), nil
			}
		})},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := client.buildURL("api/v5/schools/{SchoolCode}", RequestOptions{PathParams: map[string]string{"SchoolCode": "994"}}); err != nil {
		t.Fatalf("buildURL should accept paths without a leading slash: %v", err)
	}
	if err := client.Do(context.Background(), "", "/api/v5/systeminfo", RequestOptions{}, nil); err == nil {
		t.Fatal("Do should reject an empty method")
	}
	if err := client.Do(context.Background(), http.MethodGet, "", RequestOptions{}, nil); err == nil {
		t.Fatal("Do should reject an empty path")
	}
	if err := client.doOperation(context.Background(), "missing.operation", RequestOptions{}, nil); err == nil {
		t.Fatal("doOperation should reject unknown operations")
	}
	if err := client.Do(context.Background(), http.MethodGet, "/api/v5/invalid-json", RequestOptions{}, &JSONDocument{}); err == nil {
		t.Fatal("expected invalid JSON response to fail")
	}
	if err := client.Do(context.Background(), http.MethodGet, "/api/v5/no-content", RequestOptions{}, nil); err != nil {
		t.Fatalf("expected no-content call to succeed: %v", err)
	}
	if err := client.Do(context.Background(), http.MethodGet, "/api/v5/bad-request", RequestOptions{}, &JSONDocument{}); err == nil {
		t.Fatal("expected bad request to produce an error")
	}
	if _, err := client.doList(context.Background(), "missing.operation", RequestOptions{}); err == nil {
		t.Fatal("doList should surface operation lookup errors")
	}
	if _, err := client.doDocument(context.Background(), "missing.operation", RequestOptions{}); err == nil {
		t.Fatal("doDocument should surface operation lookup errors")
	}
	if info, err := client.doSystemInfo(context.Background(), "missing.operation", RequestOptions{}); err == nil || info != nil {
		t.Fatalf("doSystemInfo error result = %#v, %v; want nil value and error", info, err)
	}
	if _, err := client.doDocuments(context.Background(), "missing.operation", RequestOptions{}); err == nil {
		t.Fatal("doDocuments should surface operation lookup errors")
	}
}

// TestConfigValidationCoversRemainingBranches hits validation and normalization cases that are easy to forget.
func TestConfigValidationCoversRemainingBranches(t *testing.T) {
	tests := []string{
		"https://district.example.test/aeries/api/v4",
		"https://district.example.test/admin/api/v5",
		"https://district.example.test/aeries/api",
		"https://district.example.test/custom",
	}
	for _, input := range tests {
		config := Config{BaseURL: input, Certificate: testCertificate}
		if _, err := config.normalizedBaseURL(); err != nil {
			t.Fatalf("normalizedBaseURL(%q) returned error: %v", input, err)
		}
	}
	if got := (Config{MaxRetries: -5}).normalizedMaxRetries(); got != -1 {
		t.Fatalf("normalizedMaxRetries negative branch = %d", got)
	}
	if err := (Config{BaseURL: "https://district.example.test", Certificate: testCertificate, Timeout: -1}).validate(); err == nil || !strings.Contains(err.Error(), "Timeout") {
		t.Fatalf("expected timeout validation error, got %v", err)
	}
	if _, err := NewClient(Config{BaseURL: "https://district.example.test", Certificate: "bad"}); err == nil {
		t.Fatal("NewClient should reject invalid config")
	}
	if err := (Config{BaseURL: "https://district.example.test", Certificate: testCertificate, RetryBackoff: -1}).validate(); err == nil || !strings.Contains(err.Error(), "RetryBackoff") {
		t.Fatalf("expected retry backoff validation error, got %v", err)
	}
}
