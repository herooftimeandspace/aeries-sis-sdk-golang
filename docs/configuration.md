# Configuration

This page explains the runtime settings accepted by `aeries.NewClient`. Configuration is provided as an `aeries.Config` value so applications can keep credentials in their own secret store and construct the SDK without relying on package-global state.

## Settings

- `BaseURL` is the HTTPS district portal root. It is required.
- `PortalRoot` optionally overrides the portal root inferred from `BaseURL`. A nil pointer keeps the inferred behaviour. A non-nil pointer is used verbatim, so a pointer to `""` (or `"/"`) means the portal root is the domain root and suppresses the `/aeries` default entirely.
- `Certificate` is the required 32-character alphanumeric Aeries API certificate. Treat it as a secret and never log it.
- `HTTPClient` optionally supplies an application-managed `http.Client`. When it is omitted, the SDK creates one using `Timeout`.
- `UserAgent` optionally replaces the default SDK user agent.
- `Timeout` controls the generated HTTP client's timeout. Zero uses 30 seconds. Negative durations are invalid.
- `DefaultDatabaseYear` supplies `DatabaseYear` when an individual request does not set one. A request-specific value takes precedence.
- `MaxRetries` controls retries for transient transport and API failures on manifest-backed operations classified as safe reads. Zero uses two retries, and negative values are invalid. Mutations, side-effect commands, and raw `Client.Do` calls are never retried automatically.
- `RetryBackoff` controls the base delay between retries. Zero uses 300 milliseconds, and negative durations are invalid.
- `MaxResponseBytes` limits the bytes read from every API response before status handling or JSON decoding. Zero uses the conservative 32 MiB default, negative values and values above 1 TiB are invalid, and callers that intentionally accept larger photo batches must opt in explicitly.

## Response-size safety

The response limit applies to successful and unsuccessful responses through the shared transport. The SDK reads at most the configured limit plus one byte, which means a valid JSON payload exactly equal to the limit succeeds while any larger payload fails deterministically.

An oversized response returns `*aeries.ResponseTooLargeError`. For typed service methods, the error contains only the HTTP method, contract path template, status code, configured limit, and retry classification. Raw `Client.Do` calls omit the path because a concrete caller-supplied path can contain private identifiers. The error never retains response bytes, the full request URL, query values, request headers, or the Aeries certificate. A response that is already known to be oversized is not retried, including when its status code would ordinarily qualify for a safe transient retry.

`MaxResponseBytes` intentionally extends `Config` before the SDK's first tagged release so every client has one explicit response-safety policy. Applications should use keyed `Config` literals, which remain source-compatible when new optional settings are added; positional literals couple callers to the exact field count and are not a supported compatibility boundary.

Non-success responses within the configured limit return `*aeries.APIError`. The SDK does not retain the raw provider body. The deprecated `APIError.Body` field remains available for source compatibility but is always empty. For body-free requests using the documented `{"Message":"..."}` shape, `APIError.Message` contains only a control-character-free, whitespace-normalized provider detail capped at 512 bytes after known credential, custom-header, encoded and decoded query, and expanded path values are redacted. Provider detail is omitted entirely when the request had a JSON body because that body can contain student or staff data that cannot be safely identified field by field.

A `2xx` response whose body is not valid JSON returns `*aeries.ResponseDecodeError`. It carries the HTTP status the transport observed along with the HTTP method and the contract path template, so a malformed success does not have to be recorded as an unknown status. Like the other transport errors it retains no response bytes; its `Message` holds only the JSON decoder's own complaint.

Applications should use `errors.As` rather than matching error text:

```go
var tooLarge *aeries.ResponseTooLargeError
if errors.As(err, &tooLarge) {
	log.Printf("Aeries response exceeded %d bytes for %s %s", tooLarge.Limit, tooLarge.Method, tooLarge.Path)
}
```

Do not add the original request URL, headers, query parameters, or provider response body to that diagnostic. Those values can contain credentials, student identifiers, or base64 photo data.

## Observing the response status

The typed error paths carry the HTTP status, but a successful call returns only the decoded value. Applications that record provider call metadata, such as a human-gated sync run writing an audit row per request, can attach a status observer to the context instead of fabricating a status:

```go
var status int
ctx = aeries.WithResponseObserver(ctx, func(statusCode int) {
	status = statusCode
})
pictures, err := client.Students.ListPictures(ctx, req)
// status now holds what the server actually returned, including 200 versus 206.
```

The observer travels on the context rather than on `RequestOptions` because the typed service methods build their own `RequestOptions`. That keeps the hook available to every service method and to raw `Client.Do` calls without changing any method signature, in the same spirit as `httptrace.WithClientTrace`.

The hook is deliberately limited to the status integer. Response headers, response bytes, and the request URL are withheld so an audit sink cannot become a new escape path for provider payloads, student identifiers, or the Aeries certificate.

It is called once per attempt that produced an HTTP response, on the goroutine performing the request and before the response body is read. A retried safe read therefore reports each attempt in order, and the last reported status is the one the returned value or error came from. Attempts that never reached a response, such as a transport or context failure, report nothing. An observer should record the status and return promptly. Passing `nil` clears an observer already attached to the context.


## Environment variables

The repository includes [`.env.example`](https://github.com/herooftimeandspace/aeries-sis-sdk-golang/blob/main/.env.example) for local setup guidance.

## Base URL guidance

The SDK accepts the district portal root and then adds the documented API path during requests.

- a bare host like `https://district.example.org` defaults to `https://district.example.org/aeries`
- an explicit portal root like `https://district.example.org/aeries` is preserved
- an admin portal root like `https://district.example.org/admin` is preserved
- an explicit API root like `https://district.example.org/admin/api/v5` is normalized back to `https://district.example.org/admin`
- a path that normalizes to the domain root, like `https://district.example.org/api` or `https://district.example.org/api/v5`, is preserved as the domain root, so requests are sent to `https://district.example.org/api/v5/...`

The `/aeries` default applies only when `BaseURL` carries no path at all. A district whose Aeries API is served from the domain root can supply `https://district.example.org/api`, or set `PortalRoot` to a pointer to the empty string to guarantee no portal root is injected regardless of the `BaseURL` shape:

```go
portalRoot := ""
client, err := aeries.NewClient(aeries.Config{
	BaseURL:     "https://district.example.org",
	PortalRoot:  &portalRoot,
	Certificate: os.Getenv("AERIES_CERT"),
})
```
