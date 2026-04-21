package aeries

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCertificate = "1234567890ABCDEF1234567890ABCDEF"

// integrationEnvironment collects the live-test settings that can come from the shell or the repo .env file.
type integrationEnvironment struct {
	BaseURL      string
	Certificate  string
	DatabaseYear string
	UserAgent    string
}

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

// requireIntegrationConfig returns a ready-to-use client configuration or skips when live credentials are absent.
func requireIntegrationConfig(t *testing.T) Config {
	t.Helper()
	environment, err := loadIntegrationEnvironment(projectRoot(t), os.Getenv)
	if err != nil {
		t.Fatalf("integration test setup failed: %v", err)
	}
	if environment.BaseURL == "" && environment.Certificate == "" {
		t.Skip("set AERIES_BASE_URL and AERIES_CERT in the shell or the repo .env file to run integration tests")
	}
	return Config{
		BaseURL:             environment.BaseURL,
		Certificate:         environment.Certificate,
		DefaultDatabaseYear: environment.DatabaseYear,
		UserAgent:           environment.UserAgent,
	}
}

// loadIntegrationEnvironment reads live-test settings from the shell first and then falls back to the repo .env file.
func loadIntegrationEnvironment(repoRoot string, lookup func(string) string) (integrationEnvironment, error) {
	dotEnvValues, err := readDotEnvFile(filepath.Join(repoRoot, ".env"))
	if err != nil {
		return integrationEnvironment{}, err
	}
	return mergeIntegrationEnvironment(dotEnvValues, lookup)
}

// mergeIntegrationEnvironment applies the precedence rules and rejects partial credential setup.
func mergeIntegrationEnvironment(dotEnvValues map[string]string, lookup func(string) string) (integrationEnvironment, error) {
	environment := integrationEnvironment{
		BaseURL:      integrationValue("AERIES_BASE_URL", lookup, dotEnvValues),
		Certificate:  integrationValue("AERIES_CERT", lookup, dotEnvValues),
		DatabaseYear: integrationValue("AERIES_DATABASE_YEAR", lookup, dotEnvValues),
		UserAgent:    integrationValue("AERIES_USER_AGENT", lookup, dotEnvValues),
	}

	// The integration suite needs both credential values together. Having only one usually means local setup is incomplete.
	switch {
	case environment.BaseURL == "" && environment.Certificate == "":
		return environment, nil
	case environment.BaseURL == "":
		return integrationEnvironment{}, fmt.Errorf("both AERIES_BASE_URL and AERIES_CERT must be set together; AERIES_CERT is present but AERIES_BASE_URL is missing")
	case environment.Certificate == "":
		return integrationEnvironment{}, fmt.Errorf("both AERIES_BASE_URL and AERIES_CERT must be set together; AERIES_BASE_URL is present but AERIES_CERT is missing")
	default:
		return environment, nil
	}
}

// integrationValue returns the process environment value when present and otherwise falls back to the parsed .env file.
func integrationValue(key string, lookup func(string) string, dotEnvValues map[string]string) string {
	if lookup != nil {
		if value := strings.TrimSpace(lookup(key)); value != "" {
			return value
		}
	}
	return strings.TrimSpace(dotEnvValues[key])
}

// readDotEnvFile loads one repo-local .env file and tolerates the file being absent on clean checkouts.
func readDotEnvFile(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err == nil {
		return parseDotEnv(string(content))
	}
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	return nil, err
}

// parseDotEnv reads a tiny KEY=value file format that is sufficient for local integration credentials.
func parseDotEnv(content string) (map[string]string, error) {
	values := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(content))
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf(".env line %d must use KEY=value format", lineNumber)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf(".env line %d is missing a key name", lineNumber)
		}
		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) || (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

// describeIntegrationFailure translates common live-test failures into setup guidance without exposing secrets.
func describeIntegrationFailure(err error) string {
	if err == nil {
		return ""
	}

	var configErr *ConfigError
	if errors.As(err, &configErr) {
		return fmt.Sprintf("client configuration was rejected before any request was sent: %v", configErr)
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return fmt.Sprintf("the Aeries server returned status %d, which usually means the certificate is invalid or does not have permission for this endpoint: %v", apiErr.StatusCode, apiErr)
		default:
			return fmt.Sprintf("the Aeries server returned status %d: %v", apiErr.StatusCode, apiErr)
		}
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Sprintf("the request could not be completed against the configured Aeries URL: %v", urlErr)
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return fmt.Sprintf("the request could not reach the Aeries server cleanly: %v", netErr)
	}

	return err.Error()
}
