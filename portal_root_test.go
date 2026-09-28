package aeries

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPortalRootRequestPaths pins the request path each supported BaseURL shape produces, so a
// district whose portal root is the domain root can no longer be silently rewritten to /aeries.
func TestPortalRootRequestPaths(t *testing.T) {
	tests := []struct {
		name       string
		pathSuffix string
		portalRoot *string
		wantPath   string
	}{
		{name: "bare host", pathSuffix: "", wantPath: "/aeries/api/v5/schools/994/StudentPictures/10010656"},
		{name: "bare host trailing slash", pathSuffix: "/", wantPath: "/aeries/api/v5/schools/994/StudentPictures/10010656"},
		{name: "domain root api", pathSuffix: "/api", wantPath: "/api/v5/schools/994/StudentPictures/10010656"},
		{name: "aeries portal", pathSuffix: "/aeries", wantPath: "/aeries/api/v5/schools/994/StudentPictures/10010656"},
		{name: "aeries portal api v5", pathSuffix: "/aeries/api/v5", wantPath: "/aeries/api/v5/schools/994/StudentPictures/10010656"},
		{name: "portal root override", pathSuffix: "/aeries", portalRoot: stringPointer(""), wantPath: "/api/v5/schools/994/StudentPictures/10010656"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[]`))
			}))
			defer server.Close()

			client, err := NewClient(Config{
				BaseURL:     server.URL + tt.pathSuffix,
				Certificate: testCertificate,
				HTTPClient:  server.Client(),
				PortalRoot:  tt.portalRoot,
			})
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			if _, err := client.Students.ListPictures(context.Background(), SchoolStudentLookupRequest{
				SchoolCode: "994", StudentID: 10010656, DatabaseYear: "2024",
			}); err != nil {
				t.Fatalf("ListPictures: %v", err)
			}
			if gotPath != tt.wantPath {
				t.Fatalf("request path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}

// stringPointer returns a pointer to the supplied value so tests can build an explicit PortalRoot.
func stringPointer(value string) *string {
	return &value
}
