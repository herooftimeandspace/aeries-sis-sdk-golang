package contract

import (
	"sync"
	"testing"
)

// TestLoadMemoizesManifest verifies that the embedded manifest can be loaded repeatedly without error.
func TestLoadMemoizesManifest(t *testing.T) {
	first, err := Load()
	if err != nil {
		t.Fatalf("Load first call: %v", err)
	}
	second, err := Load()
	if err != nil {
		t.Fatalf("Load second call: %v", err)
	}
	if first != second {
		t.Fatal("expected Load to memoize the manifest pointer")
	}
}

// TestLoadReportsJSONErrors verifies that invalid embedded JSON is surfaced cleanly.
func TestLoadReportsJSONErrors(t *testing.T) {
	originalJSON := manifestJSON
	originalManifest := cachedManifest
	originalErr := cachedManifestErr
	// A sync.Once cannot be copied, so the cleanup restores the cached values behind a fresh
	// Once instead of saving and reassigning the original one.
	t.Cleanup(func() {
		loadOnce = sync.Once{}
		manifestJSON = originalJSON
		cachedManifest = originalManifest
		cachedManifestErr = originalErr
		loadOnce.Do(func() {})
	})

	manifestJSON = []byte("{")
	cachedManifest = nil
	cachedManifestErr = nil
	loadOnce = sync.Once{}

	if _, err := Load(); err == nil {
		t.Fatal("expected invalid JSON to fail")
	}
}
