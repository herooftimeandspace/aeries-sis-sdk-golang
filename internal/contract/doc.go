// Package contract stores the vendored Aeries documentation manifest and source snapshots.
//
// The public contract package re-exports the stable types from this internal
// package so runtime code can embed the JSON manifest while tests and tools can
// still inspect the raw snapshot data in one place.
package contract
