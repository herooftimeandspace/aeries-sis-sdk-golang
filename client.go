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
	"strings"
	"time"

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
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return decodeAPIError(response.StatusCode, method, path, payload)
	}
	if len(bytes.TrimSpace(payload)) == 0 || out == nil {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return &ValidationError{Message: fmt.Sprintf("response for %s %s was not valid JSON: %v", method, path, err)}
	}
	return nil
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
func decodeAPIError(statusCode int, method string, path string, payload []byte) error {
	apiErr := &APIError{
		StatusCode: statusCode,
		Method:     method,
		Path:       path,
		Body:       string(bytes.TrimSpace(payload)),
	}
	var structured struct {
		Message string `json:"Message"`
	}
	if err := json.Unmarshal(payload, &structured); err == nil && structured.Message != "" {
		apiErr.Message = structured.Message
	}
	return apiErr
}

// isRetriable returns true for short-lived transport and gateway failures that are worth retrying.
func isRetriable(err error) bool {
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
