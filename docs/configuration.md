# Configuration

This page explains the runtime settings the SDK will support.

## Planned settings

- `BaseURL`
- `Certificate`
- `HTTPClient`
- `UserAgent`
- `Timeout`
- `DatabaseYear`
- retry and backoff controls

## Environment variables

The repository includes [`.env.example`](../.env.example) for local setup guidance.

## Base URL guidance

The SDK accepts the district portal root and then adds the documented API path during requests.

- a bare host like `https://district.example.org` defaults to `https://district.example.org/aeries`
- an explicit portal root like `https://district.example.org/aeries` is preserved
- an admin portal root like `https://district.example.org/admin` is preserved
- an explicit API root like `https://district.example.org/admin/api/v5` is normalized back to `https://district.example.org/admin`
