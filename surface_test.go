package aeries

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	publiccontract "github.com/herooftimeandspace/aeries-sis-sdk-golang/contract"
)

type surfaceEntry struct {
	ID            string `json:"id"`
	Service       string `json:"service"`
	MethodName    string `json:"method_name"`
	RequestType   string `json:"request_type"`
	ReturnType    string `json:"return_type"`
	ResponseShape string `json:"response_shape"`
}

// loadSurfaceEntries loads the generated public method inventory used by the drift tests.
func loadSurfaceEntries(t *testing.T) []surfaceEntry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(projectRoot(t), "internal", "contract", "golden", "public_surface.json"))
	if err != nil {
		t.Fatalf("read public surface: %v", err)
	}
	var entries []surfaceEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("unmarshal public surface: %v", err)
	}
	return entries
}

// TestPublicSurfaceMatchesManifest verifies that every manifest endpoint has one public service binding and vice versa.
func TestPublicSurfaceMatchesManifest(t *testing.T) {
	manifest, err := publiccontract.Load()
	if err != nil {
		t.Fatalf("contract.Load: %v", err)
	}
	surface := loadSurfaceEntries(t)
	if len(surface) != len(manifest.Endpoints) {
		t.Fatalf("surface count = %d, manifest count = %d", len(surface), len(manifest.Endpoints))
	}
	endpointIDs := map[string]bool{}
	for _, endpoint := range manifest.Endpoints {
		endpointIDs[endpoint.ID] = true
	}
	for _, entry := range surface {
		if !endpointIDs[entry.ID] {
			t.Fatalf("surface entry %s missing from manifest", entry.ID)
		}
	}
}

// TestPublicSurfaceMethodsExist verifies that every generated binding points at a real exported service method.
func TestPublicSurfaceMethodsExist(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "https://demo.aeries.net", Certificate: testCertificate})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	clientValue := reflect.ValueOf(client).Elem()
	for _, entry := range loadSurfaceEntries(t) {
		service := clientValue.FieldByName(entry.Service)
		if !service.IsValid() || service.IsNil() {
			t.Fatalf("client missing service field %s", entry.Service)
		}
		method := service.MethodByName(entry.MethodName)
		if !method.IsValid() {
			t.Fatalf("%s missing method %s", entry.Service, entry.MethodName)
		}
		if got := method.Type().In(1).Name(); got != entry.RequestType {
			t.Fatalf("%s.%s request type = %s, want %s", entry.Service, entry.MethodName, got, entry.RequestType)
		}
		if got := reflectedReturnType(method.Type()); got != entry.ReturnType {
			t.Fatalf("%s.%s return type = %s, want %s", entry.Service, entry.MethodName, got, entry.ReturnType)
		}
	}
}

// reflectedReturnType formats the generated methods' supported result shapes like the public-surface golden.
func reflectedReturnType(method reflect.Type) string {
	if method.NumOut() == 1 {
		return method.Out(0).Name()
	}
	result := method.Out(0)
	name := result.Name()
	if result.Kind() == reflect.Slice {
		name = "[]" + result.Elem().Name()
	}
	return "(" + name + ", " + method.Out(1).Name() + ")"
}

// TestPublicSurfaceMethodsInvokeThroughTransport exercises every documented wrapper against a local test server.
func TestPublicSurfaceMethodsInvokeThroughTransport(t *testing.T) {
	var current surfaceEntry
	client, err := NewClient(Config{
		BaseURL:             "https://district.example.test",
		Certificate:         testCertificate,
		DefaultDatabaseYear: "2024",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if got := r.Header.Get("AERIES-CERT"); got != testCertificate {
				t.Fatalf("AERIES-CERT = %q", got)
			}
			if got := r.URL.Query().Get("DatabaseYear"); got != "2024" {
				t.Fatalf("DatabaseYear = %q, want 2024", got)
			}
			if !strings.HasPrefix(r.URL.Path, "/aeries/api/") {
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
			switch current.ResponseShape {
			case "object":
				return jsonResponse(http.StatusOK, `{}`), nil
			case "list":
				return jsonResponse(http.StatusOK, `[]`), nil
			case "empty":
				return noContentResponse(), nil
			default:
				t.Fatalf("unsupported response shape %q", current.ResponseShape)
				return nil, nil
			}
		})},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	clientValue := reflect.ValueOf(client).Elem()

	for _, entry := range loadSurfaceEntries(t) {
		current = entry
		service := clientValue.FieldByName(entry.Service)
		method := service.MethodByName(entry.MethodName)
		requestValue := reflect.Zero(method.Type().In(1))
		results := method.Call([]reflect.Value{reflect.ValueOf(context.Background()), requestValue})
		last := results[len(results)-1]
		if !last.IsNil() {
			t.Fatalf("%s.%s returned error: %v", entry.Service, entry.MethodName, last.Interface())
		}
	}
}
