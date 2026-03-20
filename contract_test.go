package aeries

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	publiccontract "github.com/herooftimeandspace/aeries-sis-sdk-golang/contract"
)

// TestContractLoadIncludesSourcesAndMultipleVersions verifies that the vendored manifest looks like the expected Aeries snapshot.
func TestContractLoadIncludesSourcesAndMultipleVersions(t *testing.T) {
	manifest, err := publiccontract.Load()
	if err != nil {
		t.Fatalf("contract.Load: %v", err)
	}
	if manifest.SchemaVersion == "" {
		t.Fatal("expected schema version")
	}
	if len(manifest.Sources) < 5 {
		t.Fatalf("expected multiple sources, got %d", len(manifest.Sources))
	}
	if len(manifest.Endpoints) < 40 {
		t.Fatalf("expected many endpoints, got %d", len(manifest.Endpoints))
	}

	versions := map[string]bool{}
	for _, endpoint := range manifest.Endpoints {
		versions[endpoint.APIVersion] = true
		if len(endpoint.SourceIDs) == 0 {
			t.Fatalf("endpoint %s missing source ids", endpoint.ID)
		}
	}
	for _, want := range []string{"v2", "v3", "v5"} {
		if !versions[want] {
			t.Fatalf("expected manifest to include %s endpoints", want)
		}
	}
}

// TestContractSyncIsUpToDate verifies that the checked-in generated artifacts still match the source snapshots.
func TestContractSyncIsUpToDate(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/contractsync", "-check")
	cmd.Dir = projectRoot(t)
	cmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "gocache"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("contractsync -check failed: %v\n%s", err, string(output))
	}
}

// TestPublicSurfaceJSONIsReadable verifies that the generated service inventory remains valid JSON.
func TestPublicSurfaceJSONIsReadable(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(projectRoot(t), "internal", "contract", "golden", "public_surface.json"))
	if err != nil {
		t.Fatalf("read public surface: %v", err)
	}
	var surface []map[string]any
	if err := json.Unmarshal(data, &surface); err != nil {
		t.Fatalf("unmarshal public surface: %v", err)
	}
	if len(surface) == 0 {
		t.Fatal("expected public surface entries")
	}
}
