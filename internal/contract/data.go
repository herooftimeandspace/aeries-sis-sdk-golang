package contract

import (
	_ "embed"
	"encoding/json"
	"sync"
)

// Manifest is the normalized, machine-readable contract built from the official Aeries docs.
type Manifest struct {
	GeneratedAt     string          `json:"generated_at"`
	SchemaVersion   string          `json:"schema_version"`
	RequestDefaults RequestDefaults `json:"request_defaults"`
	Sources         []Source        `json:"sources"`
	Endpoints       []Endpoint      `json:"endpoints"`
}

// RequestDefaults captures the global rules that apply to every Aeries API request.
type RequestDefaults struct {
	AcceptHeader         string   `json:"accept_header"`
	CertificateHeader    string   `json:"certificate_header"`
	DatabaseYearQueryKey string   `json:"database_year_query_key"`
	SupportedFormats     []string `json:"supported_formats"`
	Notes                []string `json:"notes"`
}

// Source captures citation metadata and summary notes for one official article.
type Source struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	URL        string   `json:"url"`
	ModifiedOn string   `json:"modified_on"`
	Notes      []string `json:"notes"`
}

// Endpoint captures one normalized endpoint from the vendored documentation snapshot.
type Endpoint struct {
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

//go:embed manifest.json
var manifestJSON []byte

var (
	loadOnce          sync.Once
	cachedManifest    *Manifest
	cachedManifestErr error
)

// Load decodes and memoizes the vendored manifest so callers do not repeat the same JSON work.
func Load() (*Manifest, error) {
	loadOnce.Do(func() {
		var manifest Manifest
		if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
			cachedManifestErr = err
			return
		}
		cachedManifest = &manifest
	})
	return cachedManifest, cachedManifestErr
}
