package aeries

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadIntegrationEnvironmentUsesDotEnvFallback verifies that local integration runs can read credentials from the ignored repo .env file.
func TestLoadIntegrationEnvironmentUsesDotEnvFallback(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, ".env"), strings.Join([]string{
		"# Local Aeries credentials",
		"AERIES_BASE_URL=\"https://district.example/aeries/api\"",
		"AERIES_CERT='1234567890ABCDEF1234567890ABCDEF'",
		"AERIES_DATABASE_YEAR=2025",
		"AERIES_USER_AGENT=integration-suite",
		"",
	}, "\n"))

	environment, err := loadIntegrationEnvironment(repoRoot, emptyLookup)
	if err != nil {
		t.Fatalf("loadIntegrationEnvironment returned error: %v", err)
	}
	if environment.BaseURL != "https://district.example/aeries/api" {
		t.Fatalf("BaseURL = %q, want %q", environment.BaseURL, "https://district.example/aeries/api")
	}
	if environment.Certificate != testCertificate {
		t.Fatalf("Certificate = %q, want %q", environment.Certificate, testCertificate)
	}
	if environment.DatabaseYear != "2025" {
		t.Fatalf("DatabaseYear = %q, want 2025", environment.DatabaseYear)
	}
	if environment.UserAgent != "integration-suite" {
		t.Fatalf("UserAgent = %q, want integration-suite", environment.UserAgent)
	}
}

// TestLoadIntegrationEnvironmentPrefersProcessEnv verifies that exported shell variables override stale values in the repo .env file.
func TestLoadIntegrationEnvironmentPrefersProcessEnv(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, ".env"), strings.Join([]string{
		"AERIES_BASE_URL=https://from-dotenv.example/aeries/api",
		"AERIES_CERT=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"AERIES_DATABASE_YEAR=2024",
		"AERIES_USER_AGENT=dotenv-agent",
		"",
	}, "\n"))

	lookup := mapLookup(map[string]string{
		"AERIES_BASE_URL":      "https://from-shell.example/aeries/api",
		"AERIES_CERT":          testCertificate,
		"AERIES_DATABASE_YEAR": "2026",
		"AERIES_USER_AGENT":    "shell-agent",
	})

	environment, err := loadIntegrationEnvironment(repoRoot, lookup)
	if err != nil {
		t.Fatalf("loadIntegrationEnvironment returned error: %v", err)
	}
	if environment.BaseURL != "https://from-shell.example/aeries/api" {
		t.Fatalf("BaseURL = %q, want shell value", environment.BaseURL)
	}
	if environment.Certificate != testCertificate {
		t.Fatalf("Certificate = %q, want shell value", environment.Certificate)
	}
	if environment.DatabaseYear != "2026" {
		t.Fatalf("DatabaseYear = %q, want shell value", environment.DatabaseYear)
	}
	if environment.UserAgent != "shell-agent" {
		t.Fatalf("UserAgent = %q, want shell value", environment.UserAgent)
	}
}

// TestLoadIntegrationEnvironmentRejectsPartialConfiguration verifies that the helper fails fast when only one required credential value is present.
func TestLoadIntegrationEnvironmentRejectsPartialConfiguration(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, ".env"), "AERIES_BASE_URL=https://district.example/aeries/api\n")

	_, err := loadIntegrationEnvironment(repoRoot, emptyLookup)
	if err == nil {
		t.Fatal("loadIntegrationEnvironment succeeded, want partial configuration error")
	}
	if !strings.Contains(err.Error(), "AERIES_BASE_URL") || !strings.Contains(err.Error(), "AERIES_CERT") {
		t.Fatalf("partial configuration error = %v, want both required variable names", err)
	}
}

// TestDescribeIntegrationFailureClassifiesCommonCases verifies that live-test failures point developers toward the likely setup problem.
func TestDescribeIntegrationFailureClassifiesCommonCases(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantSubstr string
	}{
		{
			name:       "config error",
			err:        &ConfigError{Field: "BaseURL", Message: "must use https"},
			wantSubstr: "configuration was rejected",
		},
		{
			name:       "permission error",
			err:        &APIError{StatusCode: http.StatusForbidden, Method: "GET", Path: "/api/v5/system/info"},
			wantSubstr: "certificate is invalid or does not have permission",
		},
		{
			name:       "transport url error",
			err:        &url.Error{Op: "Get", URL: "https://district.example/aeries/api/v5/system/info", Err: errors.New("dial tcp lookup failed")},
			wantSubstr: "configured Aeries URL",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := describeIntegrationFailure(tt.err)
			if !strings.Contains(got, tt.wantSubstr) {
				t.Fatalf("describeIntegrationFailure(%T) = %q, want substring %q", tt.err, got, tt.wantSubstr)
			}
		})
	}
}

// writeTestFile creates one small fixture file for env-loading tests.
func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// emptyLookup simulates a process environment that does not define any integration variables.
func emptyLookup(string) string {
	return ""
}

// mapLookup turns a simple map into the lookup function used by the env-loading helper.
func mapLookup(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}
