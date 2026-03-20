//go:build integration

package aeries

import (
	"context"
	"os"
	"testing"
)

// TestIntegrationSystemInfoSmoke performs the safest documented smoke call when integration credentials are present.
func TestIntegrationSystemInfoSmoke(t *testing.T) {
	baseURL := os.Getenv("AERIES_BASE_URL")
	certificate := os.Getenv("AERIES_CERT")
	if baseURL == "" || certificate == "" {
		t.Skip("set AERIES_BASE_URL and AERIES_CERT to run integration tests")
	}
	client, err := NewClient(Config{BaseURL: baseURL, Certificate: certificate})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := client.System.GetInfo(context.Background(), SystemInfoRequest{}); err != nil {
		t.Fatalf("system info smoke test failed: %v", err)
	}
}
