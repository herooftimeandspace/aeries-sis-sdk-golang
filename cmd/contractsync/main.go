package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type sourceFile struct {
	Sources []sourceSnapshot `json:"sources"`
}

type sourceSnapshot struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	URL        string   `json:"url"`
	ModifiedOn string   `json:"modified_on"`
	Notes      []string `json:"notes"`
}

type requestDefaultsFile struct {
	RequestDefaults requestDefaultsSnapshot `json:"request_defaults"`
}

type requestDefaultsSnapshot struct {
	AcceptHeader         string   `json:"accept_header"`
	CertificateHeader    string   `json:"certificate_header"`
	DatabaseYearQueryKey string   `json:"database_year_query_key"`
	SupportedFormats     []string `json:"supported_formats"`
	Notes                []string `json:"notes"`
}

type endpointsFile struct {
	Endpoints []endpointSnapshot `json:"endpoints"`
}

type endpointSnapshot struct {
	ID                   string   `json:"id"`
	Service              string   `json:"service"`
	MethodName           string   `json:"method_name"`
	Summary              string   `json:"summary"`
	HTTPMethod           string   `json:"http_method"`
	PathTemplate         string   `json:"path_template"`
	APIVersion           string   `json:"api_version"`
	ResponseShape        string   `json:"response_shape"`
	PathParameters       []string `json:"path_parameters"`
	QueryParameters      []string `json:"query_parameters"`
	SupportsDatabaseYear bool     `json:"supports_database_year"`
	Mutation             bool     `json:"mutation"`
	SourceIDs            []string `json:"source_ids"`
}

type manifest struct {
	GeneratedAt     string                  `json:"generated_at"`
	SchemaVersion   string                  `json:"schema_version"`
	RequestDefaults requestDefaultsSnapshot `json:"request_defaults"`
	Sources         []sourceSnapshot        `json:"sources"`
	Endpoints       []endpointSnapshot      `json:"endpoints"`
}

type publicSurfaceEntry struct {
	ID            string `json:"id"`
	Service       string `json:"service"`
	MethodName    string `json:"method_name"`
	RequestType   string `json:"request_type"`
	ReturnType    string `json:"return_type"`
	ResponseShape string `json:"response_shape"`
}

var exitWithError = exitf

// main delegates to run so the business logic stays directly testable.
func main() {
	if err := run(os.Args[1:]); err != nil {
		exitWithError("%v", err)
	}
}

// run loads the snapshot inputs, rewrites the generated artifacts, and optionally checks for drift.
func run(args []string) error {
	flagSet := flag.NewFlagSet("contractsync", flag.ContinueOnError)
	checkOnly := flagSet.Bool("check", false, "verify generated artifacts without rewriting them")
	if err := flagSet.Parse(args); err != nil {
		return err
	}
	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine working directory: %w", err)
	}
	sourceDir := filepath.Join(root, "internal", "contract", "source")
	manifestPath := filepath.Join(root, "internal", "contract", "manifest.json")
	publicSurfacePath := filepath.Join(root, "internal", "contract", "golden", "public_surface.json")

	manifestValue, publicSurfaceValue, err := generateArtifacts(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to generate contract artifacts: %w", err)
	}
	if *checkOnly {
		if err := checkFile(manifestPath, manifestValue); err != nil {
			return err
		}
		if err := checkFile(publicSurfacePath, publicSurfaceValue); err != nil {
			return err
		}
		return nil
	}
	return writeArtifacts(manifestPath, publicSurfacePath, manifestValue, publicSurfaceValue)
}

// writeArtifacts creates the target directories and writes both generated JSON files.
func writeArtifacts(manifestPath string, publicSurfacePath string, manifestValue []byte, publicSurfaceValue []byte) error {
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(publicSurfacePath), 0o755); err != nil {
		return fmt.Errorf("failed to create golden directory: %w", err)
	}
	if err := os.WriteFile(manifestPath, manifestValue, 0o644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}
	if err := os.WriteFile(publicSurfacePath, publicSurfaceValue, 0o644); err != nil {
		return fmt.Errorf("failed to write public surface: %w", err)
	}
	return nil
}

// generateArtifacts converts the curated snapshot JSON files into the normalized manifest outputs.
func generateArtifacts(sourceDir string) ([]byte, []byte, error) {
	var sources sourceFile
	if err := loadJSON(filepath.Join(sourceDir, "sources.json"), &sources); err != nil {
		return nil, nil, err
	}
	var defaults requestDefaultsFile
	if err := loadJSON(filepath.Join(sourceDir, "request_defaults.json"), &defaults); err != nil {
		return nil, nil, err
	}
	var endpoints endpointsFile
	if err := loadJSON(filepath.Join(sourceDir, "endpoints.json"), &endpoints); err != nil {
		return nil, nil, err
	}
	requestTypes, err := loadWrapperRequestTypes(filepath.Join(sourceDir, "service_wrappers.txt"))
	if err != nil {
		return nil, nil, err
	}

	sort.SliceStable(sources.Sources, func(i, j int) bool {
		return sources.Sources[i].ID < sources.Sources[j].ID
	})
	sort.SliceStable(endpoints.Endpoints, func(i, j int) bool {
		if endpoints.Endpoints[i].Service == endpoints.Endpoints[j].Service {
			return endpoints.Endpoints[i].MethodName < endpoints.Endpoints[j].MethodName
		}
		return endpoints.Endpoints[i].Service < endpoints.Endpoints[j].Service
	})

	manifestValue := manifest{
		GeneratedAt:     "2026-03-20",
		SchemaVersion:   "2026-02-09.aeries-public-api",
		RequestDefaults: defaults.RequestDefaults,
		Sources:         sources.Sources,
		Endpoints:       endpoints.Endpoints,
	}
	manifestJSON, err := marshalJSON(manifestValue)
	if err != nil {
		return nil, nil, err
	}

	surface := make([]publicSurfaceEntry, 0, len(endpoints.Endpoints))
	for _, endpoint := range endpoints.Endpoints {
		requestType, ok := requestTypes[endpoint.ID]
		if !ok {
			return nil, nil, fmt.Errorf("endpoint %s has no wrapper request type", endpoint.ID)
		}
		returnType, err := publicReturnType(endpoint)
		if err != nil {
			return nil, nil, err
		}
		surface = append(surface, publicSurfaceEntry{
			ID:            endpoint.ID,
			Service:       endpoint.Service,
			MethodName:    endpoint.MethodName,
			RequestType:   requestType,
			ReturnType:    returnType,
			ResponseShape: endpoint.ResponseShape,
		})
	}
	publicSurfaceJSON, err := marshalJSON(surface)
	if err != nil {
		return nil, nil, err
	}
	return manifestJSON, publicSurfaceJSON, nil
}

// loadWrapperRequestTypes reads the compact wrapper mapping used to generate public method signatures.
func loadWrapperRequestTypes(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open wrapper metadata: %w", err)
	}
	defer file.Close()
	requests := map[string]string{}
	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		columns := strings.SplitN(text, "|", 2)
		parts := strings.Fields(columns[0])
		if len(parts) < 2 {
			return nil, fmt.Errorf("wrapper metadata line %d is missing an operation id or request type", line)
		}
		if _, exists := requests[parts[0]]; exists {
			return nil, fmt.Errorf("duplicate wrapper metadata for %s", parts[0])
		}
		requests[parts[0]] = parts[1]
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan wrapper metadata: %w", err)
	}
	return requests, nil
}

// publicReturnType records the resolved Go result contract independently of wrapper generation.
func publicReturnType(endpoint endpointSnapshot) (string, error) {
	if endpoint.ID == "system.get_info" {
		return "(SystemInfo, error)", nil
	}
	switch endpoint.ResponseShape {
	case "object":
		return "(JSONDocument, error)", nil
	case "list":
		return "([]JSONDocument, error)", nil
	case "empty":
		return "error", nil
	default:
		return "", fmt.Errorf("endpoint %s has unsupported response shape %q", endpoint.ID, endpoint.ResponseShape)
	}
}

// loadJSON decodes one JSON file into the caller's struct.
func loadJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("failed to decode %s: %w", path, err)
	}
	return nil
}

// marshalJSON writes human-readable JSON so the generated files stay easy to review in git.
func marshalJSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	return data, nil
}

// checkFile compares the generated output against the checked-in file content.
func checkFile(path string, expected []byte) error {
	actual, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read generated file %s: %w", path, err)
	}
	if !bytes.Equal(actual, expected) {
		return fmt.Errorf("generated artifact drift detected for %s", path)
	}
	return nil
}

// exitf prints a user-friendly message and exits with a non-zero code.
func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
