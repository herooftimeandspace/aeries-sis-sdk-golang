package aeries

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

// TestClientDoSendsHeadersBodyAndDefaultDatabaseYear verifies the low-level request contract.
func TestClientDoSendsHeadersBodyAndDefaultDatabaseYear(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		wantPath string
	}{
		{name: "bare host defaults to aeries", baseURL: "https://district.example.test", wantPath: "/aeries/api/v5/systeminfo"},
		{name: "admin portal root is preserved", baseURL: "https://district.example.test/admin/", wantPath: "/admin/api/v5/systeminfo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sawRequest atomic.Bool
			client, err := NewClient(Config{
				BaseURL:             tt.baseURL,
				Certificate:         testCertificate,
				DefaultDatabaseYear: "2024",
				HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					sawRequest.Store(true)
					if got := r.Header.Get("AERIES-CERT"); got != testCertificate {
						t.Fatalf("AERIES-CERT = %q, want %q", got, testCertificate)
					}
					if got := r.Header.Get("Accept"); got != "application/json" {
						t.Fatalf("Accept = %q", got)
					}
					if got := r.URL.Query().Get("DatabaseYear"); got != "2024" {
						t.Fatalf("DatabaseYear = %q, want 2024", got)
					}
					if r.URL.Path != tt.wantPath {
						t.Fatalf("path = %q, want %q", r.URL.Path, tt.wantPath)
					}
					return jsonResponse(http.StatusOK, `{"status":"ok"}`), nil
				})},
			})
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}

			var response JSONDocument
			if err := client.Do(context.Background(), http.MethodGet, "/api/v5/systeminfo", RequestOptions{}, &response); err != nil {
				t.Fatalf("Do returned error: %v", err)
			}
			if !sawRequest.Load() {
				t.Fatal("expected the server to receive one request")
			}
			if response["status"] != "ok" {
				t.Fatalf("response status = %#v", response["status"])
			}
		})
	}
}

// TestClientDoParsesStructuredAPIError verifies that the documented Message payload becomes a typed API error.
func TestClientDoParsesStructuredAPIError(t *testing.T) {
	client, err := NewClient(Config{
		BaseURL:     "https://district.example.test",
		Certificate: testCertificate,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusUnauthorized, `{"Message":"bad certificate"}`), nil
		})},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	err = client.Do(context.Background(), http.MethodGet, "/api/v5/systeminfo", RequestOptions{}, &JSONDocument{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Message != "bad certificate" {
		t.Fatalf("apiErr.Message = %q", apiErr.Message)
	}
	if apiErr.Body != "" {
		t.Fatalf("apiErr.Body = %q, want empty compatibility field", apiErr.Body)
	}
}

// TestClientDoSanitizesStructuredAPIError verifies that provider detail cannot echo request secrets or grow without bound.
func TestClientDoSanitizesStructuredAPIError(t *testing.T) {
	const querySecret = "query secret"
	const encodedQuerySecret = "abc/def"
	const headerSecret = "header-secret"
	const bearerSecret = "abc123"
	const cookieSecret = "cookie456"
	const userAgent = "private-user-agent"
	const studentID = "student 12345"
	client, err := NewClient(Config{
		BaseURL:     "https://district.example.test",
		Certificate: testCertificate,
		UserAgent:   userAgent,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			message := testCertificate + "\n" + url.QueryEscape(querySecret) + " " + url.PathEscape(querySecret) + " abc%2fdef\t" + headerSecret + " " + bearerSecret + " " + cookieSecret + " " + userAgent + " student\t12345 query    secret " + strings.Repeat("x", maxProviderDetailBytes)
			return jsonResponse(http.StatusBadRequest, `{"Message":`+strconv.Quote(message)+`}`), nil
		})},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	err = client.Do(context.Background(), http.MethodGet, "/api/v5/students/{StudentID}", RequestOptions{
		PathParams: map[string]string{"StudentID": studentID},
		Query:      map[string]string{"filter": querySecret, "encoded": encodedQuerySecret},
		Headers:    map[string]string{"X-Request-Token": headerSecret, "Authorization": "Bearer " + bearerSecret, "Cookie": "session=" + cookieSecret},
	}, &JSONDocument{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if len(apiErr.Message) > maxProviderDetailBytes {
		t.Fatalf("provider detail length = %d, want at most %d", len(apiErr.Message), maxProviderDetailBytes)
	}
	for _, forbidden := range []string{testCertificate, querySecret, url.QueryEscape(querySecret), url.PathEscape(querySecret), "abc%2fdef", headerSecret, bearerSecret, cookieSecret, userAgent, studentID, "\n", "\t", "\x1b"} {
		if strings.Contains(apiErr.Message, forbidden) || strings.Contains(apiErr.Error(), forbidden) {
			t.Fatalf("sanitized error retained forbidden value %q: %q", forbidden, apiErr.Error())
		}
	}
}

// TestClientDoDropsProviderDetailForRequestBodies verifies errors cannot retain echoed mutation payloads.
func TestClientDoDropsProviderDetailForRequestBodies(t *testing.T) {
	const studentPII = "student-private-value"
	client, err := NewClient(Config{
		BaseURL:     "https://district.example.test",
		Certificate: testCertificate,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusBadRequest, `{"Message":"request body was {\\"Name\\":\\"`+studentPII+`\\"}"}`), nil
		})},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	err = client.Do(context.Background(), http.MethodPost, "/api/v5/UpdateStudent", RequestOptions{
		JSONBody: JSONDocument{"Name": studentPII},
	}, &JSONDocument{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Message != "" || apiErr.Body != "" || strings.Contains(apiErr.Error(), studentPII) {
		t.Fatalf("body-bearing error retained provider detail: %#v", apiErr)
	}
}

// TestClientDoRetriesRetriableStatus verifies that transient API failures are retried.
func TestClientDoRetriesRetriableStatus(t *testing.T) {
	var attempts atomic.Int32
	client, err := NewClient(Config{
		BaseURL:      "https://district.example.test",
		Certificate:  testCertificate,
		MaxRetries:   1,
		RetryBackoff: 1,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if attempts.Add(1) == 1 {
				return jsonResponse(http.StatusBadGateway, `{"Message":"temporary"}`), nil
			}
			return jsonResponse(http.StatusOK, `{"retried":true}`), nil
		})},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	var response JSONDocument
	if err := client.Do(context.Background(), http.MethodGet, "/api/v5/systeminfo", RequestOptions{}, &response); err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d, want 2", attempts.Load())
	}
	if response["retried"] != true {
		t.Fatalf("response retried = %#v", response["retried"])
	}
}

// TestClientBuildURLRejectsMissingPathParam verifies that placeholder mistakes fail fast before any HTTP request is sent.
func TestClientBuildURLRejectsMissingPathParam(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "https://demo.aeries.net", Certificate: testCertificate})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.buildURL("/api/v5/schools/{SchoolCode}", RequestOptions{})
	if err == nil || !strings.Contains(err.Error(), "missing path parameters") {
		t.Fatalf("buildURL error = %v", err)
	}
}
