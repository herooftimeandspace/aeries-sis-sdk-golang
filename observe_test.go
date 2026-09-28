package aeries

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// newObserverTestClient builds a client whose transport replies with the supplied responses in order.
func newObserverTestClient(t *testing.T, responses ...*http.Response) *Client {
	t.Helper()
	remaining := responses
	client, err := NewClient(Config{
		BaseURL:      "https://district.example.test",
		Certificate:  testCertificate,
		MaxRetries:   1,
		RetryBackoff: time.Nanosecond,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			if len(remaining) == 0 {
				t.Fatal("transport called more times than the test supplied responses")
			}
			next := remaining[0]
			remaining = remaining[1:]
			return next, nil
		})},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

// TestResponseObserverReportsSuccessStatus verifies that a typed service method reports the status it observed.
func TestResponseObserverReportsSuccessStatus(t *testing.T) {
	client := newObserverTestClient(t, jsonResponse(http.StatusPartialContent, `[{"StudentID":1}]`))
	var statuses []int
	ctx := WithResponseObserver(context.Background(), func(statusCode int) {
		statuses = append(statuses, statusCode)
	})
	if _, err := client.Students.ListPictures(ctx, SchoolStudentLookupRequest{SchoolCode: "1", StudentID: 1}); err != nil {
		t.Fatalf("ListPictures: %v", err)
	}
	if len(statuses) != 1 || statuses[0] != http.StatusPartialContent {
		t.Fatalf("observed statuses = %v, want [206]", statuses)
	}
}

// TestResponseObserverReportsEmptySuccessStatus verifies the no-body success path still reports its status.
func TestResponseObserverReportsEmptySuccessStatus(t *testing.T) {
	client := newObserverTestClient(t, noContentResponse())
	var statuses []int
	ctx := WithResponseObserver(context.Background(), func(statusCode int) {
		statuses = append(statuses, statusCode)
	})
	if err := client.Do(ctx, http.MethodGet, "/api/v5/systeminfo", RequestOptions{}, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if len(statuses) != 1 || statuses[0] != http.StatusNoContent {
		t.Fatalf("observed statuses = %v, want [204]", statuses)
	}
}

// TestResponseObserverReportsEveryRetriedAttempt verifies that each completed attempt of a safe read is reported.
func TestResponseObserverReportsEveryRetriedAttempt(t *testing.T) {
	client := newObserverTestClient(t,
		jsonResponse(http.StatusServiceUnavailable, `{"Message":"try later"}`),
		jsonResponse(http.StatusOK, `[]`),
	)
	var statuses []int
	ctx := WithResponseObserver(context.Background(), func(statusCode int) {
		statuses = append(statuses, statusCode)
	})
	if _, err := client.Students.ListPictures(ctx, SchoolStudentLookupRequest{SchoolCode: "1", StudentID: 1}); err != nil {
		t.Fatalf("ListPictures: %v", err)
	}
	if len(statuses) != 2 || statuses[0] != http.StatusServiceUnavailable || statuses[1] != http.StatusOK {
		t.Fatalf("observed statuses = %v, want [503 200]", statuses)
	}
}

// TestResponseObserverReportsErrorStatus verifies the hook also fires for the APIError path.
func TestResponseObserverReportsErrorStatus(t *testing.T) {
	client := newObserverTestClient(t, jsonResponse(http.StatusNotFound, `{"Message":"missing"}`))
	var statuses []int
	ctx := WithResponseObserver(context.Background(), func(statusCode int) {
		statuses = append(statuses, statusCode)
	})
	_, err := client.Students.ListPictures(ctx, SchoolStudentLookupRequest{SchoolCode: "1", StudentID: 1})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *APIError", err)
	}
	if len(statuses) != 1 || statuses[0] != http.StatusNotFound {
		t.Fatalf("observed statuses = %v, want [404]", statuses)
	}
}

// TestResponseObserverIsOptional verifies that requests without an observer, or with a cleared one, still work.
func TestResponseObserverIsOptional(t *testing.T) {
	client := newObserverTestClient(t, jsonResponse(http.StatusOK, `[]`))
	ctx := WithResponseObserver(WithResponseObserver(context.Background(), func(int) {
		t.Fatal("cleared observer should not be called")
	}), nil)
	if _, err := client.Students.ListPictures(ctx, SchoolStudentLookupRequest{SchoolCode: "1", StudentID: 1}); err != nil {
		t.Fatalf("ListPictures: %v", err)
	}
	if observe := responseObserverFrom(nil); observe != nil { //nolint:staticcheck // a nil context must not panic
		t.Fatal("a nil context should carry no observer")
	}
}

// TestResponseObserverReportsOversizedStatus verifies the hook fires before the size limit rejects a response.
func TestResponseObserverReportsOversizedStatus(t *testing.T) {
	client, err := NewClient(Config{
		BaseURL:          "https://district.example.test",
		Certificate:      testCertificate,
		MaxResponseBytes: 8,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, strings.Repeat("a", 64)), nil
		})},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	var statuses []int
	ctx := WithResponseObserver(context.Background(), func(statusCode int) {
		statuses = append(statuses, statusCode)
	})
	_, err = client.Students.ListPictures(ctx, SchoolStudentLookupRequest{SchoolCode: "1", StudentID: 1})
	var tooLarge *ResponseTooLargeError
	if !errors.As(err, &tooLarge) {
		t.Fatalf("error = %v, want *ResponseTooLargeError", err)
	}
	if len(statuses) != 1 || statuses[0] != http.StatusOK {
		t.Fatalf("observed statuses = %v, want [200]", statuses)
	}
}

// TestResponseDecodeErrorCarriesStatus verifies that a malformed success body keeps the status the transport saw.
func TestResponseDecodeErrorCarriesStatus(t *testing.T) {
	client := newObserverTestClient(t, jsonResponse(http.StatusOK, `not json`))
	var statuses []int
	ctx := WithResponseObserver(context.Background(), func(statusCode int) {
		statuses = append(statuses, statusCode)
	})
	_, err := client.Students.ListPictures(ctx, SchoolStudentLookupRequest{SchoolCode: "1", StudentID: 1})
	var decodeErr *ResponseDecodeError
	if !errors.As(err, &decodeErr) {
		t.Fatalf("error = %v, want *ResponseDecodeError", err)
	}
	if decodeErr.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", decodeErr.StatusCode)
	}
	if decodeErr.Method != http.MethodGet {
		t.Fatalf("Method = %q, want GET", decodeErr.Method)
	}
	if decodeErr.Path != "/api/v5/schools/{SchoolCode}/StudentPictures/{StudentID}" {
		t.Fatalf("Path = %q", decodeErr.Path)
	}
	if !strings.Contains(decodeErr.Error(), "status 200") {
		t.Fatalf("Error() = %q, want the status included", decodeErr.Error())
	}
	var nilDecodeError *ResponseDecodeError
	if nilDecodeError.Error() != "" {
		t.Fatal("nil error receiver should render an empty string")
	}
	if len(statuses) != 1 || statuses[0] != http.StatusOK {
		t.Fatalf("observed statuses = %v, want [200]", statuses)
	}
}
