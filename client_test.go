package aeries

import (
	"context"
	"errors"
	"net/http"
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
}

// TestClientDoSanitizesStructuredAPIError verifies that provider detail cannot echo request secrets or grow without bound.
func TestClientDoSanitizesStructuredAPIError(t *testing.T) {
	const querySecret = "query-secret"
	const headerSecret = "header-secret"
	const userAgent = "private-user-agent"
	client, err := NewClient(Config{
		BaseURL:     "https://district.example.test",
		Certificate: testCertificate,
		UserAgent:   userAgent,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			message := testCertificate + "\n" + querySecret + "\t" + headerSecret + " " + userAgent + " " + r.URL.String() + strings.Repeat("x", maxProviderDetailBytes)
			return jsonResponse(http.StatusBadRequest, `{"Message":`+strconv.Quote(message)+`}`), nil
		})},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	err = client.Do(context.Background(), http.MethodGet, "/api/v5/systeminfo", RequestOptions{
		Query:   map[string]string{"filter": querySecret},
		Headers: map[string]string{"X-Request-Token": headerSecret},
	}, &JSONDocument{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if len(apiErr.Message) > maxProviderDetailBytes {
		t.Fatalf("provider detail length = %d, want at most %d", len(apiErr.Message), maxProviderDetailBytes)
	}
	for _, forbidden := range []string{testCertificate, querySecret, headerSecret, userAgent, "district.example.test", "\n", "\t"} {
		if strings.Contains(apiErr.Message, forbidden) || strings.Contains(apiErr.Error(), forbidden) {
			t.Fatalf("sanitized error retained forbidden value %q: %q", forbidden, apiErr.Error())
		}
	}
}

// TestSafeContractReadRetriesRetriableStatus verifies that manifest-backed reads may retry transient API failures.
func TestSafeContractReadRetriesRetriableStatus(t *testing.T) {
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

	_, err = client.System.GetInfo(context.Background(), SystemInfoRequest{})
	if err != nil {
		t.Fatalf("GetInfo returned error: %v", err)
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d, want 2", attempts.Load())
	}
}

// TestContractMutationsAndRawRequestsDoNotRetry verifies that unproven or state-changing calls receive one attempt.
func TestContractMutationsAndRawRequestsDoNotRetry(t *testing.T) {
	tests := []struct {
		name string
		call func(*Client) error
	}{
		{
			name: "post mutation",
			call: func(client *Client) error {
				_, err := client.Students.Create(context.Background(), StudentCreateRequest{SchoolCode: "994"})
				return err
			},
		},
		{
			name: "get side effect",
			call: func(client *Client) error {
				_, err := client.PreEnroll.TriggerInactive(context.Background(), PreEnrollInactiveRequest{StudentID: 1001, NextSchoolCode: 994})
				return err
			},
		},
		{
			name: "raw escape hatch",
			call: func(client *Client) error {
				return client.Do(context.Background(), http.MethodGet, "/api/v5/custom-trigger", RequestOptions{}, &JSONDocument{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var attempts atomic.Int32
			client, err := NewClient(Config{
				BaseURL:      "https://district.example.test",
				Certificate:  testCertificate,
				MaxRetries:   2,
				RetryBackoff: 1,
				HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					attempts.Add(1)
					return jsonResponse(http.StatusBadGateway, `{"Message":"temporary"}`), nil
				})},
			})
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			if err := tt.call(client); err == nil {
				t.Fatal("expected transient failure")
			}
			if attempts.Load() != 1 {
				t.Fatalf("attempts = %d, want 1", attempts.Load())
			}
		})
	}
}

// TestAddedParityOperations verifies the additive wrappers, paths, verbs, and request-specific DatabaseYear propagation.
func TestAddedParityOperations(t *testing.T) {
	tests := []struct {
		name       string
		wantMethod string
		wantPath   string
		call       func(*Client) error
	}{
		{name: "inactive pre-enrollment", wantMethod: http.MethodGet, wantPath: "/aeries/api/v5/PreEnrollInactiveStudent/1001/994", call: func(client *Client) error {
			_, err := client.PreEnroll.TriggerInactive(context.Background(), PreEnrollInactiveRequest{StudentID: 1001, NextSchoolCode: 994, DatabaseYear: "2025"})
			return err
		}},
		{name: "student grades", wantMethod: http.MethodGet, wantPath: "/aeries/api/v5/schools/994/Grades", call: func(client *Client) error {
			_, err := client.StudentGrades.ListGrades(context.Background(), SchoolLookupRequest{SchoolCode: "994", DatabaseYear: "2025"})
			return err
		}},
		{name: "staff create", wantMethod: http.MethodPost, wantPath: "/aeries/api/v5/staff", call: func(client *Client) error {
			_, err := client.Staff.Create(context.Background(), StaffCreateRequest{DatabaseYear: "2025"})
			return err
		}},
		{name: "staff update", wantMethod: http.MethodPut, wantPath: "/aeries/api/v5/staff/42", call: func(client *Client) error {
			_, err := client.Staff.Update(context.Background(), StaffUpdateRequest{StaffID: 42, DatabaseYear: "2025"})
			return err
		}},
		{name: "contact create", wantMethod: http.MethodPost, wantPath: "/aeries/api/v5/InsertContact/1001", call: func(client *Client) error {
			_, err := client.Students.CreateContact(context.Background(), StudentMutationRequest{StudentID: 1001, DatabaseYear: "2025"})
			return err
		}},
		{name: "discipline", wantMethod: http.MethodGet, wantPath: "/aeries/api/v5/schools/994/Discipline/1001", call: func(client *Client) error {
			_, err := client.Students.ListDiscipline(context.Background(), SchoolStudentLookupRequest{SchoolCode: "994", StudentID: 1001, DatabaseYear: "2025"})
			return err
		}},
		{name: "scheduling section", wantMethod: http.MethodGet, wantPath: "/aeries/api/v5/schools/994/scheduling/sections/7", call: func(client *Client) error {
			_, err := client.Scheduling.GetSchedulingSection(context.Background(), SectionLookupRequest{SchoolCode: "994", SectionNumber: 7, DatabaseYear: "2025"})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(Config{
				BaseURL:     "https://district.example.test",
				Certificate: testCertificate,
				HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					if r.Method != tt.wantMethod || r.URL.Path != tt.wantPath {
						t.Fatalf("request = %s %s, want %s %s", r.Method, r.URL.Path, tt.wantMethod, tt.wantPath)
					}
					if got := r.URL.Query().Get("DatabaseYear"); got != "2025" {
						t.Fatalf("DatabaseYear = %q, want 2025", got)
					}
					if tt.name == "staff create" || tt.name == "staff update" || tt.name == "contact create" || tt.name == "scheduling section" {
						return jsonResponse(http.StatusOK, `{}`), nil
					}
					return jsonResponse(http.StatusOK, `[]`), nil
				})},
			})
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			if err := tt.call(client); err != nil {
				t.Fatalf("call returned error: %v", err)
			}
		})
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
