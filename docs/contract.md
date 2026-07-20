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
- `internal/contract/source/service_wrappers.txt` maps each operation to its stable public request type and preserves its public method comment.
- `requests.go` defines the fields and Go types on those request contracts.

Run `make generate` after changing any of these sources. The generator writes readable, formatted Go files in the repository root and marks them with a generated-file header. Run `make generate-check` in review or CI-style validation to verify that the checked-in wrappers and contract artifacts have no drift. Keep custom transport, authentication, validation, and response decoding behavior in handwritten helpers instead of adding special cases to generated method bodies.

Most contract parameter names map directly to request fields. Two documented aliases are intentional: the Aeries query parameter `code` reads `ProgramLookupRequest.ProgramCode`, and attendance endpoints using `SchoolStringLookupRequest` map the path parameter `AcademicYear` to its general-purpose `Value` field. The generator validates every endpoint, request type, and parameter field and fails closed when metadata does not match. If a future endpoint introduces another naming exception or unusual body shape, add a narrow, documented mapping to `cmd/wrappergen` and cover it with a generator test rather than weakening validation or hiding the exception in generated code.

## Cross-SDK parity

The deterministic `cmd/contractparity` command compares this endpoint inventory with the Python SDK's committed `contract_snapshot.json`. It matches unchanged routes, then requires every differing operation pair and every language-only operation to appear explicitly in `internal/contract/parity_disputes.json`. The ledger does not accept patterns or wildcards, so a new route, verb, version, placeholder-casing change, or side-effect classification fails the comparison until a contributor investigates and documents it.

See [Go and Python Contract Parity](contract-parity.md) for the snapshot versions, confirmed additive operations, held upstream disputes, and retry implications.
