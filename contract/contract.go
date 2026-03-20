package contract

import internalcontract "github.com/herooftimeandspace/aeries-sis-sdk-golang/internal/contract"

// Manifest describes the vendored documentation snapshot used by the SDK.
type Manifest = internalcontract.Manifest

// Source describes one official Aeries article captured in the vendored contract snapshot.
type Source = internalcontract.Source

// RequestDefaults describes the shared request rules documented by Aeries.
type RequestDefaults = internalcontract.RequestDefaults

// Endpoint describes one documented API endpoint exposed through the SDK.
type Endpoint = internalcontract.Endpoint

// Load returns the vendored manifest that powers the SDK's endpoint routing and tests.
func Load() (*Manifest, error) {
	return internalcontract.Load()
}
