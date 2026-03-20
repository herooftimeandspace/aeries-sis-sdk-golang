package aeries

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCertificate = "1234567890ABCDEF1234567890ABCDEF"

// projectRoot returns the repository root so tests can find docs and generated artifacts.
func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(dir)
}

// roundTripFunc lets tests stub the HTTP transport without opening real network sockets.
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip adapts the helper function to the http.RoundTripper interface.
func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

// jsonResponse builds a JSON HTTP response for one in-memory transport call.
func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// noContentResponse builds an empty success response for mutation endpoints that return no body.
func noContentResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusNoContent,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
	}
}
