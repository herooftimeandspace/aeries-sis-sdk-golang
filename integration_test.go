//go:build integration

package aeries

import (
	"context"
	"sort"
	"testing"
)

// TestIntegrationSystemInfoSmoke performs the safest documented smoke call when integration credentials are present.
func TestIntegrationSystemInfoSmoke(t *testing.T) {
	client := newIntegrationClient(t)
	info, err := client.System.GetInfo(context.Background(), SystemInfoRequest{})
	if err != nil {
		t.Fatalf("system info smoke test failed: %s", describeIntegrationFailure(err))
	}
	if len(info) == 0 {
		t.Fatal("System.GetInfo returned an empty JSON object")
	}
	assertDocumentHasAnyKey(t, "System.GetInfo", info, "AeriesVersion", "DatabaseYear", "Version", "DistrictName")
}

// TestIntegrationSchoolsListSmoke verifies that a second documented read-only endpoint can return school data.
func TestIntegrationSchoolsListSmoke(t *testing.T) {
	client := newIntegrationClient(t)

	// Start with the safest installation-wide call so a later failure is easier to interpret.
	if _, err := client.System.GetInfo(context.Background(), SystemInfoRequest{}); err != nil {
		t.Fatalf("System.GetInfo must succeed before the schools smoke test can classify downstream failures: %s", describeIntegrationFailure(err))
	}

	schools, err := client.Schools.List(context.Background(), SchoolLookupRequest{})
	if err != nil {
		t.Fatalf("schools list smoke test failed after System.GetInfo succeeded, so the base URL and certificate are likely valid: %s", describeIntegrationFailure(err))
	}
	if len(schools) == 0 {
		t.Log("Schools.List reached the endpoint successfully but returned zero rows for this tenant.")
		return
	}
	assertDocumentHasAnyKey(t, "Schools.List first row", schools[0], "SchoolCode", "SchoolName", "Name", "Code")
}

// newIntegrationClient builds the live client from shell variables or the repo-local .env file.
func newIntegrationClient(t *testing.T) *Client {
	t.Helper()
	client, err := NewClient(requireIntegrationConfig(t))
	if err != nil {
		t.Fatalf("NewClient: %s", describeIntegrationFailure(err))
	}
	return client
}

// assertDocumentHasAnyKey proves that a live JSON object contains at least one stable top-level field.
func assertDocumentHasAnyKey(t *testing.T, label string, document JSONDocument, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if _, ok := document[key]; ok {
			return
		}
	}
	t.Fatalf("%s did not include any of the expected keys %v; got keys %v", label, keys, sortedDocumentKeys(document))
}

// sortedDocumentKeys returns the current top-level JSON keys in a stable order for failure messages.
func sortedDocumentKeys(document JSONDocument) []string {
	keys := make([]string, 0, len(document))
	for key := range document {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
