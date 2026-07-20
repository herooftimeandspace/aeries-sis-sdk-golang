# API Contract

The contract for this SDK will come from the Aeries support articles, not from guesswork.

## Contract sources

- Official Aeries documentation articles
- Normalized manifest generated from the articles
- Vendored source snapshots under `internal/contract/source`

## Contract expectations

- JSON-only public behavior
- typed request and response models
- documented endpoint coverage
- drift checks against the manifest

## Generated service wrappers

The public service methods are generated from three checked-in sources so repetitive request plumbing does not have to be maintained by hand:

- `internal/contract/source/endpoints.json` defines operation names, path and query parameters, response shapes, and contract summaries.
- `internal/contract/source/service_wrappers.txt` maps each operation to its stable public request type, preserves its public method comment, and explicitly names `body=Values` or `body=Items` for POST and PUT operations whose request type carries that payload. GET and DELETE operations must remain body-free, and a body-bearing POST or PUT fails generation when its body metadata is missing.
- `requests.go` defines the fields and Go types on those request contracts.

Run `make generate` after changing any of these sources. The generator writes readable, formatted Go files in the repository root, marks them with a generated-file header, and removes a previously generated root wrapper when its service no longer exists in the contract inventory. It refuses empty inventories, unknown service names, duplicate output filenames, and any attempt to overwrite a root Go file that lacks its exact ownership header. Run `make generate-check` in review or CI-style validation to verify content and fail when a stale generated wrapper remains. Keep custom transport, authentication, validation, and response decoding behavior in handwritten helpers instead of adding special cases to generated method bodies.

The generated `internal/contract/golden/public_surface.json` records each method's request type and resolved Go return type in addition to its service, method name, and response shape. Contract drift checks therefore flag accidental signature changes even when wrapper source is regenerated from edited metadata.

Most contract parameter names map directly to request fields. Two documented aliases are intentional: the Aeries query parameter `code` reads `ProgramLookupRequest.ProgramCode`, and attendance endpoints using `SchoolStringLookupRequest` map the path parameter `AcademicYear` to its general-purpose `Value` field. The generator validates every endpoint, request type, and parameter field and fails closed when metadata does not match. If a future endpoint introduces another naming exception or unusual body shape, add a narrow, documented mapping to `cmd/wrappergen` and cover it with a generator test rather than weakening validation or hiding the exception in generated code.
