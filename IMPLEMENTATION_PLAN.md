# Aeries SIS Go SDK Build Plan (`aeries-sis-sdk-golang`)

## Summary
- The first mutating step is creating `./aeries-sis-sdk-golang/IMPLEMENTATION_PLAN.md` with this exact content; any later scope or interface change must update that file before code changes.
- Build a greenfield Go module `github.com/herooftimeandspace/aeries-sis-sdk-golang` with root package `aeries`, using only `./aeries-sis-sdk-golang`.
- Target the current official Aeries docs snapshot centered on February 9, 2026: use v5 for core APIs and the still-current documented v3 endpoints where Aeries defines them that way.
- Keep the SDK JSON-only for v1, with a dual contract surface: typed Go request/response models plus a generated machine-readable manifest vendored from the official Aeries docs.
- Treat documentation as a first-class deliverable: combine the README, architecture guides, endpoint reference, and generated code reference into one GitHub Pages site written for a junior engineer who may be new to both Go and this codebase.

## Public Interfaces and Implementation Changes
- Public entry points:
  - `aeries.Config`
  - `aeries.NewClient(Config) (*Client, error)`
  - `(*Client).Do(ctx, method, path string, opts RequestOptions, out any) error`
  - service accessors on `Client`: `System`, `Schools`, `CodeSets`, `PreEnroll`, `Students`, `StudentGrades`, `Attendance`, `Staff`, `Scheduling`, `Gradebook`, `Alerts`, `Programs`
  - public `contract` package with `Load() (*Manifest, error)` plus read-only `Manifest` and `Endpoint` types
- `Config` includes `BaseURL`, `Certificate`, `HTTPClient`, `UserAgent`, `Timeout`, retry/backoff settings, and optional default `DatabaseYear`. Base URL normalization must resolve requests against the `/aeries/api/...` root.
- Service methods map 1:1 to the currently documented endpoint families and use consistent verbs: `List...`, `Get...`, `Create...`, `Update...`, `Delete...`, and `Trigger...`. Each method accepts typed request structs and returns typed models or slices matching the documented JSON shape.
- Include all currently documented read and write operations in scope, including school updates, contacts update/delete, pre-enroll triggers, test-score updates, grade updates, scheduling section create/update/delete, gradebook mutations, and alerts.
- Keep pagination explicit in typed request structs using Aeries’ documented fields such as `StartingRecord`, `EndingRecord`, `StartingStudent`, and `EndingStudent`; do not invent a generic cursor abstraction for v1.
- Use a hybrid implementation strategy:
  - hand-author transport, config, auth, errors, package docs, and public model types
  - generate the normalized contract manifest and repetitive operation inventory from the docs
  - do not blindly generate public Go structs from HTML examples alone
- Add `internal/contract/source/` for vendored raw article snapshots and fetch metadata, `internal/contract/manifest.json` for normalized endpoint metadata, `internal/contract/golden/public_surface.json` for drift detection, and `cmd/contractsync` to refresh sources and regenerate the manifest.
- Add package-level orientation docs with `doc.go` in the root package and every internal package. Every handwritten function and method, including non-trivial test helpers, must have a human-readable leading comment that explains intent, why the code exists, inputs/outputs, and any side effects in language suitable for a junior engineer unfamiliar with Go.
- Add inline intent comments around non-obvious control flow, parsing rules, retry behavior, and Aeries-specific business rules. Comments must explain purpose, not restate syntax. Generated artifacts are exempt only if the generator emits a clear file header pointing readers to the canonical handwritten docs.

## Test Plan
- Follow strict TDD order: config/auth transport first, then contract loader/parser, then each service family, then docs/examples and CI gates.
- Enforce `>=95%` statement coverage on the always-run suite with `go test ./... -covermode=atomic -coverprofile=coverage.out`. The required suite includes unit tests, contract-manifest tests, hermetic integration tests with `httptest`, and documentation contract checks.
- Add contract drift tests that fail when:
  - any documented endpoint is missing from the manifest
  - any manifest endpoint lacks a typed service method
  - any public service method lacks a manifest entry
  - fixture coverage is missing for a documented request or response shape
- Add AST-based documentation tests that fail when any handwritten package lacks `doc.go`, any handwritten function or method lacks a leading intent comment, or generated API reference pages are out of sync with the source comments.
- Add docs build tests that fail when the README is not mirrored into the site home page, when package reference pages are missing, or when the GitHub Pages build cannot render successfully.
- Add live smoke tests behind `-tags=integration` and env vars `AERIES_BASE_URL` and `AERIES_CERT`, plus optional known IDs for safe read-only endpoints. Missing env vars must cause clean skips with clear messages.
- Live write-operation tests require explicit opt-in via a separate mutation flag and sandbox identifiers; they are never part of the default suite.
- Cover documented failure modes and edge cases: `{"Message": ...}` error parsing, `AERIES-CERT` header behavior, base URL normalization, `DatabaseYear` propagation, v3 versus v5 path selection, null and optional fields, and large-result endpoints that require pagination or year filters.

## Docs and Delivery
- Ship `README.md`, `.env.example`, `mkdocs.yml`, `docs/`, and package docs that explain certificate-based auth, base URL setup, JSON-only behavior, `DatabaseYear`, testing strategy, contract sync, and endpoint-family usage with beginner-friendly examples.
- Build a single GitHub Pages site using [MkDocs](https://www.mkdocs.org/) with [Material for MkDocs](https://squidfunk.github.io/mkdocs-material/). The site home page must be generated from the repository README so the README and published docs stay aligned.
- Generate API reference Markdown from Go doc comments using [gomarkdoc](https://github.com/princjef/gomarkdoc), including internal package reference needed for onboarding engineers who are new to the codebase.
- Publish the built static site through a GitHub Actions Pages workflow on `main`. The Pages site must include:
  - README-driven landing page
  - quickstart and configuration guides
  - Aeries contract and endpoint-family guides
  - testing and coverage guide
  - generated API reference for public and internal packages
- Keep docs generation deterministic and checked in or checkable in CI; CI must run the docs generation step, verify no drift, build the Pages site, and enforce the 95% coverage gate.

## Assumptions and Defaults
- Go baseline is `1.26.1`, matching the installed toolchain and sibling Go SDK conventions in this workspace.
- Scope is the full currently documented Aeries surface from the official docs and quick reference, not legacy aliases; where the current docs still use v3 endpoints, the SDK supports those documented current paths.
- Contract sources are the official Aeries articles: [Full Documentation](https://support.aeries.com/support/solutions/articles/14000077926-aeries-api-full-documentation), [Building a Request](https://support.aeries.com/support/solutions/articles/14000113681-aeries-api-building-a-request), [Quick Reference](https://support.aeries.com/support/solutions/articles/14000070569-api-quick-reference-guide), and the linked endpoint-family pages for School, Pre-Enroll, Student, Student Grades, Attendance/Enrollment, Staff, Scheduling, Gradebook, and Alerts.
- JSON is the only first-class public serialization format for v1; XML is out of scope.
- Documentation tone assumes the reader may be entirely unfamiliar with Go, Aeries terminology, and this repository, so comments and guides must define jargon on first use and prefer plain English over shorthand.
