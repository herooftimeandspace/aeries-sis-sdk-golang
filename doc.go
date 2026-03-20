// Package aeries provides a beginner-friendly Go SDK for the Aeries SIS API.
//
// The package is intentionally organized around plain-language service groups so a
// junior engineer can start from the business concept they care about, such as
// students, attendance, or gradebooks, and then drill into individual methods.
// Each service method wraps one documented Aeries endpoint and routes through the
// same transport, retry, and error handling code so behavior stays consistent.
package aeries
