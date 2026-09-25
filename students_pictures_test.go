package aeries

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

// picturePayload builds a valid Aeries-shaped photo response with an exact byte length.
func picturePayload(totalBytes int) string {
	prefix := `[{"StudentID":12345,"Photo":"`
	suffix := `"}]`
	if totalBytes < len(prefix)+len(suffix) {
		panic("picture payload length is too small")
	}
	return prefix + strings.Repeat("A", totalBytes-len(prefix)-len(suffix)) + suffix
}

// newPictureTestClient builds a client whose only response is supplied by the test case.
func newPictureTestClient(t *testing.T, limit int64, retries int, transport roundTripFunc) *Client {
	t.Helper()
	client, err := NewClient(Config{
		BaseURL:          "https://district.example.test/admin",
		Certificate:      testCertificate,
		HTTPClient:       &http.Client{Transport: transport},
		MaxRetries:       retries,
		RetryBackoff:     1,
		MaxResponseBytes: limit,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

// TestStudentsListPicturesAcceptsRealisticPhotoPayload verifies that a normal base64 photo response remains usable below the default limit.
func TestStudentsListPicturesAcceptsRealisticPhotoPayload(t *testing.T) {
	payload := `[{"StudentID":12345,"Photo":"` + strings.Repeat("QUJD", 64<<10) + `"}]`
	client := newPictureTestClient(t, defaultMaxResponseBytes, 1, func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/admin/api/v5/schools/994/StudentPictures/12345" {
			t.Fatalf("request path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("DatabaseYear") != "2024" || r.URL.Query().Get("StartingRecord") != "1" || r.URL.Query().Get("EndingRecord") != "10" {
			t.Fatalf("request query = %q", r.URL.RawQuery)
		}
		return jsonResponse(http.StatusOK, payload), nil
	})

	rows, err := client.Students.ListPictures(context.Background(), SchoolStudentLookupRequest{
		SchoolCode: "994", StudentID: 12345, StartingRecord: 1, EndingRecord: 10, DatabaseYear: "2024",
	})
	if err != nil {
		t.Fatalf("ListPictures: %v", err)
	}
	if len(rows) != 1 || rows[0]["StudentID"] != float64(12345) || len(rows[0]["Photo"].(string)) == 0 {
		t.Fatalf("unexpected picture response %#v", rows)
	}
}

// TestStudentsListPicturesAcceptsPayloadExactlyAtLimit verifies the limit-plus-one boundary is not off by one.
func TestStudentsListPicturesAcceptsPayloadExactlyAtLimit(t *testing.T) {
	const limit = int64(2048)
	payload := picturePayload(int(limit))
	client := newPictureTestClient(t, limit, 1, func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, payload), nil
	})

	if _, err := client.Students.ListPictures(context.Background(), SchoolStudentLookupRequest{SchoolCode: "994", StudentID: 12345}); err != nil {
		t.Fatalf("ListPictures returned an error for an exactly-at-limit response: %v", err)
	}
}

// TestStudentsListPicturesRejectsOversizedSuccess verifies a 2xx response fails with sanitized typed metadata.
func TestStudentsListPicturesRejectsOversizedSuccess(t *testing.T) {
	const limit = int64(1024)
	var attempts atomic.Int32
	client := newPictureTestClient(t, limit, 2, func(r *http.Request) (*http.Response, error) {
		attempts.Add(1)
		return jsonResponse(http.StatusOK, picturePayload(int(limit)+1)), nil
	})

	_, err := client.Students.ListPictures(context.Background(), SchoolStudentLookupRequest{SchoolCode: "994", StudentID: 12345})
	assertOversizedPictureError(t, err, http.StatusOK, limit)
	if attempts.Load() != 1 {
		t.Fatalf("oversized success attempts = %d, want 1", attempts.Load())
	}
}

// TestStudentsListPicturesRejectsOversizedErrorWithoutRetry verifies even a normally retriable status is not downloaded repeatedly once its size is known.
func TestStudentsListPicturesRejectsOversizedErrorWithoutRetry(t *testing.T) {
	const limit = int64(1024)
	const secretPayload = "raw-photo-or-provider-secret"
	var attempts atomic.Int32
	client := newPictureTestClient(t, limit, 2, func(r *http.Request) (*http.Response, error) {
		attempts.Add(1)
		body := fmt.Sprintf(`{"Message":"%s%s"}`, secretPayload, strings.Repeat("x", int(limit)))
		return jsonResponse(http.StatusServiceUnavailable, body), nil
	})

	_, err := client.Students.ListPictures(context.Background(), SchoolStudentLookupRequest{SchoolCode: "994", StudentID: 12345})
	assertOversizedPictureError(t, err, http.StatusServiceUnavailable, limit)
	if attempts.Load() != 1 {
		t.Fatalf("oversized error attempts = %d, want 1", attempts.Load())
	}
	if strings.Contains(err.Error(), secretPayload) {
		t.Fatalf("oversized error exposed response content: %q", err)
	}
}

// assertOversizedPictureError checks the public, sanitized metadata retained for an oversized picture response.
func assertOversizedPictureError(t *testing.T, err error, status int, limit int64) {
	t.Helper()
	var tooLarge *ResponseTooLargeError
	if !errors.As(err, &tooLarge) {
		t.Fatalf("expected ResponseTooLargeError, got %T: %v", err, err)
	}
	if tooLarge.Method != http.MethodGet || tooLarge.Path != "/api/v5/schools/{SchoolCode}/StudentPictures/{StudentID}" || tooLarge.StatusCode != status || tooLarge.Limit != limit || tooLarge.Retryable {
		t.Fatalf("unexpected oversized metadata: %#v", tooLarge)
	}
}
