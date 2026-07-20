# Configuration

This page explains the runtime settings accepted by `aeries.NewClient`. Configuration is provided as an `aeries.Config` value so applications can keep credentials in their own secret store and construct the SDK without relying on package-global state.

## Settings

- `BaseURL` is the HTTPS district portal root. It is required.
- `Certificate` is the required 32-character alphanumeric Aeries API certificate. Treat it as a secret and never log it.
- `HTTPClient` optionally supplies an application-managed `http.Client`. When it is omitted, the SDK creates one using `Timeout`.
- `UserAgent` optionally replaces the default SDK user agent.
- `Timeout` controls the generated HTTP client's timeout. Zero uses 30 seconds. Negative durations are invalid.
- `DefaultDatabaseYear` supplies `DatabaseYear` when an individual request does not set one. A request-specific value takes precedence.
- `MaxRetries` controls retries for transient transport and API failures. Zero uses two retries, and negative values are invalid.
- `RetryBackoff` controls the base delay between retries. Zero uses 300 milliseconds, and negative durations are invalid.
- `MaxResponseBytes` limits the bytes read from every API response before status handling or JSON decoding. Zero uses the conservative 32 MiB default, negative values and values above 1 TiB are invalid, and callers that intentionally accept larger photo batches must opt in explicitly.

## Response-size safety

The response limit applies to successful and unsuccessful responses through the shared transport. The SDK reads at most the configured limit plus one byte, which means a valid JSON payload exactly equal to the limit succeeds while any larger payload fails deterministically.

An oversized response returns `*aeries.ResponseTooLargeError`. The typed error contains only the HTTP method, contract path template, status code, configured limit, and retry classification. It does not retain response bytes, the full request URL, query values, request headers, or the Aeries certificate. A response that is already known to be oversized is not retried, including when its status code would ordinarily qualify for a safe transient retry.

`MaxResponseBytes` intentionally extends `Config` before the SDK's first tagged release so every client has one explicit response-safety policy. Applications should use keyed `Config` literals, which remain source-compatible when new optional settings are added; positional literals couple callers to the exact field count and are not a supported compatibility boundary.

Non-success responses within the configured limit return `*aeries.APIError`. The SDK does not retain the raw provider body. The deprecated `APIError.Body` field remains available for source compatibility but is always empty. For body-free requests using the documented `{"Message":"..."}` shape, `APIError.Message` contains only a control-character-free, whitespace-normalized provider detail capped at 512 bytes after known credential, custom-header, encoded and decoded query, and expanded path values are redacted. Provider detail is omitted entirely when the request had a JSON body because that body can contain student or staff data that cannot be safely identified field by field.

Applications should use `errors.As` rather than matching error text:

```go
var tooLarge *aeries.ResponseTooLargeError
if errors.As(err, &tooLarge) {
	log.Printf("Aeries response exceeded %d bytes for %s %s", tooLarge.Limit, tooLarge.Method, tooLarge.Path)
}
```

Do not add the original request URL, headers, query parameters, or provider response body to that diagnostic. Those values can contain credentials, student identifiers, or base64 photo data.

## Environment variables

The repository root includes `.env.example` for local setup guidance.

## Base URL guidance

The SDK accepts the district portal root and then adds the documented API path during requests.

- a bare host like `https://district.example.org` defaults to `https://district.example.org/aeries`
- an explicit portal root like `https://district.example.org/aeries` is preserved
- an admin portal root like `https://district.example.org/admin` is preserved
- an explicit API root like `https://district.example.org/admin/api/v5` is normalized back to `https://district.example.org/admin`
