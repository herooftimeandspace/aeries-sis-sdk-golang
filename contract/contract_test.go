package contract

import "testing"

// TestLoadReturnsManifest verifies that the public contract package re-exports the vendored manifest successfully.
func TestLoadReturnsManifest(t *testing.T) {
	manifest, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if manifest.SchemaVersion == "" || len(manifest.Endpoints) == 0 {
		t.Fatalf("unexpected manifest %#v", manifest)
	}
}
