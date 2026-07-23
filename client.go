package aeries

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	publiccontract "github.com/herooftimeandspace/aeries-sis-sdk-golang/contract"
)

// Client is the shared entry point for every typed Aeries service.
type Client struct {
	baseURL             string
	certificate         string
	httpClient          *http.Client
	userAgent           string
	maxRetries          int
	retryBackoff        time.Duration
	maxResponseBytes    int64
	defaultDatabaseYear string
	manifest            *publiccontract.Manifest
	endpoints           map[string]publiccontract.Endpoint

	System        *SystemService
	Schools       *SchoolsService
	CodeSets      *CodeSetsService
	PreEnroll     *PreEnrollService
	Students      *StudentsService
	StudentGrades *StudentGradesService
	Attendance    *AttendanceService
	Staff         *StaffService
	Scheduling    *SchedulingService
	Gradebook     *GradebookService
	Alerts        *AlertsService
	Programs      *ProgramsService
}

// NewClient validates the configuration, loads the vendored contract, and wires up each service group.
func NewClient(config Config) (*Client, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	baseURL, err := config.normalizedBaseURL()
	if err != nil {
		return nil, err
	}
	manifest, err := publiccontract.Load()
	if err != nil {
		return nil, err
	}
	client := &Client{
		baseURL:             baseURL,
		certificate:         config.Certificate,
		httpClient:          config.normalizedHTTPClient(),
		userAgent:           config.normalizedUserAgent(),
		maxRetries:          config.normalizedMaxRetries(),
		retryBackoff:        config.normalizedRetryBackoff(),
		maxResponseBytes:    config.normalizedMaxResponseBytes(),
		defaultDatabaseYear: config.DefaultDatabaseYear,
		manifest:            manifest,
		endpoints:           make(map[string]publiccontract.Endpoint, len(manifest.Endpoints)),
	}
	for _, endpoint := range manifest.Endpoints {
		client.endpoints[endpoint.ID] = endpoint
	}
	client.System = &SystemService{client: client}
	client.Schools = &SchoolsService{client: client}
	client.CodeSets = &CodeSetsService{client: client}
	client.PreEnroll = &PreEnrollService{client: client}
	client.Students = &StudentsService{client: client}
	client.StudentGrades = &StudentGradesService{client: client}
	client.Attendance = &AttendanceService{client: client}
	client.Staff = &StaffService{client: client}
	client.Scheduling = &SchedulingService{client: client}
	client.Gradebook = &GradebookService{client: client}
	client.Alerts = &AlertsService{client: client}
	client.Programs = &ProgramsService{client: client}
	return client, nil
}

// Do performs one raw API request using the shared transport and error handling rules.
func (c *Client) Do(ctx context.Context, method string, path string, opts RequestOptions, out any) error {
	if strings.TrimSpace(path) == "" {
		return &ValidationError{Message: "path is required"}
	}
	requestURL, err := c.buildURL(path, opts)
	if err != nil {
		return err
	}
	bodyReader, err := encodeBody(opts.JSONBody)
	if err != nil {
		return err
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		return &ValidationError{Message: "method is required"}
	}
	return c.doHTTPRequest(ctx, method, path, requestURL, bodyReader, opts.Headers, out)
}

// doOperation resolves one endpoint from the vendored manifest and then sends the request.
func (c *Client) doOperation(ctx context.Context, operationID string, opts RequestOptions, out any) error {
	endpoint, ok := c.endpoints[operationID]
	if !ok {
		return &ValidationError{Message: fmt.Sprintf("unknown contract operation %q", operationID)}
	}
	return c.Do(ctx, endpoint.HTTPMethod, endpoint.PathTemplate, opts, out)
}

// buildURL expands path parameters, applies the base URL, and appends any supported query parameters.
func (c *Client) buildURL(pathTemplate string, opts RequestOptions) (string, error) {
	expandedPath := pathTemplate
	for key, value := range opts.PathParams {
		expandedPath = strings.ReplaceAll(expandedPath, "{"+key+"}", url.PathEscape(value))
	}
	if strings.Contains(expandedPath, "{") {
		return "", &ValidationError{Message: fmt.Sprintf("missing path parameters for %q", expandedPath)}
	}
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(expandedPath, "/") {
		expandedPath = "/" + expandedPath
	}
	base.Path = strings.TrimSuffix(base.Path, "/") + expandedPath
	query := base.Query()
	for key, value := range opts.Query {
		query.Set(key, value)
	}
	databaseYear := strings.TrimSpace(opts.DatabaseYear)
	if databaseYear == "" {
		databaseYear = strings.TrimSpace(c.defaultDatabaseYear)
	}
	if databaseYear != "" {
		query.Set("DatabaseYear", databaseYear)
	}
	base.RawQuery = query.Encode()
	return base.String(), nil
}

// doHTTPRequest sends the HTTP request and retries short-lived transport failures when configured.
func (c *Client) doHTTPRequest(ctx context.Context, method string, path string, requestURL string, body []byte, headers map[string]string, out any) error {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			if err := sleepWithContext(ctx, time.Duration(attempt)*c.retryBackoff); err != nil {
				return err
			}
		}
		err := c.sendOnce(ctx, method, path, requestURL, body, headers, out)
		if err == nil {
			return nil
		}
		lastErr = err
		if !isRetriable(err) || attempt == c.maxRetries {
			return err
		}
	}
	return lastErr
}

// sendOnce performs a single HTTP attempt and decodes the JSON response when one is present.
func (c *Client) sendOnce(ctx context.Context, method string, path string, requestURL string, body []byte, headers map[string]string, out any) error {
	request, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("AERIES-CERT", c.certificate)
	request.Header.Set("User-Agent", c.userAgent)
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	payload, oversized, err := readBoundedResponse(response.Body, c.maxResponseBytes)
	if err != nil {
		return err
	}
	if oversized {
		return &ResponseTooLargeError{
			StatusCode: response.StatusCode,
			Method:     method,
			Path:       path,
			Limit:      c.maxResponseBytes,
			Retryable:  false,
		}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return decodeAPIError(response.StatusCode, method, path, payload, sensitiveRequestValues(c.certificate, requestURL, path, request.Header), len(body) == 0)
	}
	if len(bytes.TrimSpace(payload)) == 0 || out == nil {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return &ValidationError{Message: fmt.Sprintf("response for %s %s was not valid JSON: %v", method, path, err)}
	}
	return nil
}

// readBoundedResponse reads at most limit plus one bytes so a payload exactly at the configured limit remains valid.
func readBoundedResponse(reader io.Reader, limit int64) ([]byte, bool, error) {
	payload, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(payload)) > limit {
		return nil, true, nil
	}
	return payload, false, nil
}

// encodeBody converts any request body into JSON before the request is sent.
func encodeBody(body any) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, &ValidationError{Message: fmt.Sprintf("request body could not be encoded as JSON: %v", err)}
	}
	return encoded, nil
}

// decodeAPIError prefers the documented Aeries {"Message": "..."} error format when it is present.
func decodeAPIError(statusCode int, method string, path string, payload []byte, sensitiveValues []string, retainProviderDetail bool) error {
	apiErr := &APIError{
		StatusCode: statusCode,
		Method:     method,
		Path:       path,
	}
	var structured struct {
		Message string `json:"Message"`
	}
	// Requests with bodies can contain student or staff data that cannot be
	// exhaustively identified by field name. Dropping provider detail for those
	// calls is safer than retaining an echo of a partial JSON payload.
	if retainProviderDetail {
		if err := json.Unmarshal(payload, &structured); err == nil && structured.Message != "" {
			apiErr.Message = sanitizeProviderDetail(structured.Message, sensitiveValues)
		}
	}
	return apiErr
}

const maxProviderDetailBytes = 512

// sensitiveRequestValues returns values that a provider might echo but that diagnostics must never retain.
func sensitiveRequestValues(certificate string, requestURL string, contractPath string, headers http.Header) []string {
	values := []string{certificate, requestURL}
	for name, headerValues := range headers {
		for _, value := range headerValues {
			values = append(values, value)
			values = append(values, credentialHeaderComponents(name, value)...)
		}
	}
	if parsed, err := url.Parse(requestURL); err == nil {
		values = append(values, parsed.RequestURI(), parsed.Path)
		for _, queryValues := range parsed.Query() {
			for _, value := range queryValues {
				queryEscaped := url.QueryEscape(value)
				pathEscaped := url.PathEscape(value)
				values = append(values, value, queryEscaped, lowercasePercentEscapes(queryEscaped), pathEscaped, lowercasePercentEscapes(pathEscaped))
			}
		}
		values = append(values, expandedPathParameterValues(contractPath, parsed)...)
	}
	return values
}

// credentialHeaderComponents extracts tokens that providers may echo without their surrounding header scheme or key.
func credentialHeaderComponents(name string, value string) []string {
	switch strings.ToLower(name) {
	case "authorization", "proxy-authorization":
		parts := strings.Fields(value)
		if len(parts) > 1 {
			return []string{strings.Join(parts[1:], " ")}
		}
	case "cookie":
		var components []string
		for _, cookie := range strings.Split(value, ";") {
			if _, token, ok := strings.Cut(cookie, "="); ok && strings.TrimSpace(token) != "" {
				components = append(components, strings.TrimSpace(token))
			}
		}
		return components
	}
	return nil
}

// lowercasePercentEscapes preserves literal character case while accepting lowercase hexadecimal escape digits.
func lowercasePercentEscapes(value string) string {
	encoded := []byte(value)
	for index := 0; index+2 < len(encoded); index++ {
		if encoded[index] == '%' {
			encoded[index+1] = byte(unicode.ToLower(rune(encoded[index+1])))
			encoded[index+2] = byte(unicode.ToLower(rune(encoded[index+2])))
			index += 2
		}
	}
	return string(encoded)
}

// expandedPathParameterValues extracts only expanded placeholder segments, avoiding broad redaction of fixed API path words.
func expandedPathParameterValues(contractPath string, requestURL *url.URL) []string {
	templateSegments := strings.Split(strings.Trim(contractPath, "/"), "/")
	escapedSegments := strings.Split(strings.Trim(requestURL.EscapedPath(), "/"), "/")
	if len(templateSegments) == 0 || len(escapedSegments) < len(templateSegments) {
		return nil
	}
	escapedSegments = escapedSegments[len(escapedSegments)-len(templateSegments):]
	values := []string{}
	for index, templateSegment := range templateSegments {
		if !strings.HasPrefix(templateSegment, "{") || !strings.HasSuffix(templateSegment, "}") {
			continue
		}
		escaped := escapedSegments[index]
		values = append(values, escaped)
		for attempt := 0; attempt < 3; attempt++ {
			decoded, err := url.PathUnescape(escaped)
			if err != nil || decoded == escaped {
				break
			}
			values = append(values, decoded)
			escaped = decoded
		}
	}
	return values
}

// normalizeProviderDetail removes controls and collapses whitespace so formatting cannot conceal a sensitive value.
func normalizeProviderDetail(detail string) string {
	detail = strings.Map(func(character rune) rune {
		if unicode.IsControl(character) {
			if unicode.IsSpace(character) {
				return ' '
			}
			return -1
		}
		return character
	}, detail)
	return strings.Join(strings.Fields(detail), " ")
}

// sanitizeProviderDetail normalizes provider text before redacting request values and enforcing a small byte bound.
func sanitizeProviderDetail(detail string, sensitiveValues []string) string {
	detail = normalizeProviderDetail(detail)
	// Replace longer values first so an overlapping short identifier cannot
	// leave a revealing suffix behind when it appears inside a longer value.
	sensitiveValues = append([]string(nil), sensitiveValues...)
	for index, value := range sensitiveValues {
		sensitiveValues[index] = normalizeProviderDetail(value)
	}
	sort.SliceStable(sensitiveValues, func(left int, right int) bool {
		return len(sensitiveValues[left]) > len(sensitiveValues[right])
	})
	for _, value := range sensitiveValues {
		if value != "" {
			detail = strings.ReplaceAll(detail, value, "[redacted]")
		}
	}
	if len(detail) <= maxProviderDetailBytes {
		return detail
	}
	truncated := []byte(detail[:maxProviderDetailBytes])
	for len(truncated) > 0 && !utf8.Valid(truncated) {
		truncated = truncated[:len(truncated)-1]
	}
	return string(truncated)
}

// isRetriable returns true for short-lived transport and gateway failures that are worth retrying.
func isRetriable(err error) bool {
	var tooLargeErr *ResponseTooLargeError
	if errors.As(err, &tooLargeErr) {
		return tooLargeErr.Retryable
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		default:
			return apiErr.StatusCode >= 500
		}
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

// sleepWithContext waits between retries while still respecting cancellation and deadlines.
func sleepWithContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
