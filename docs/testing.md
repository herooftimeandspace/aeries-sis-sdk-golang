# Testing Guide

This page will describe the full test strategy for the SDK.

## What the tests will cover

- transport and auth
- contract loading and normalization
- endpoint and service behavior
- docs generation and drift checks
- integration smoke tests with explicit environment variables or a repo-local `.env` fallback

## Coverage goal

The default suite must stay at or above 95% statement coverage.

## Integration smoke tests

The live smoke tests stay read-only on purpose. They are meant to answer three quick setup questions:

- is the configured Aeries URL reachable
- is the certificate accepted
- can we read JSON from a documented endpoint

The tests are behind the Go build tag `integration` and first look for exported shell variables. If a variable is not exported, the test helper falls back to the ignored repo-local `.env` file.

Required settings:

- `AERIES_BASE_URL`
- `AERIES_CERT`

Optional settings:

- `AERIES_DATABASE_YEAR`
- `AERIES_USER_AGENT`

Run the live suite with either command:

- `go test -tags=integration ./...`
- `make integration-test`

When neither required credential is present, the integration tests skip cleanly. When one required credential is present without the other, the suite fails fast with a setup error so the local configuration problem is obvious.
