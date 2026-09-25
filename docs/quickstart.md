# Quickstart

This page is the first-stop guide for a developer who is new to Aeries and new to this SDK.

## What you should learn here

- How to create a client
- How the certificate header works
- How the base URL is normalized for `/aeries`, `/admin`, and explicit `/api/...` inputs
- How to make a safe first request

## Create a client

The following configuration uses the default 30-second timeout, up to two transient retries for contract-classified safe reads, and a 32 MiB response limit. Mutations and command-style side effects receive one attempt. Load the base URL and certificate from a secret-aware application configuration source; do not hard-code a real certificate in source code.

```go
client, err := aeries.NewClient(aeries.Config{
	BaseURL:     districtBaseURL,
	Certificate: districtCertificate,
})
if err != nil {
	return err
}
```

Applications that intentionally retrieve larger batches of student pictures can raise `MaxResponseBytes`. Keep the value as small as the application's expected workload permits, paginate picture requests where practical, and handle `*aeries.ResponseTooLargeError` as a typed, non-retriable safety failure. See [Configuration](configuration.md#response-size-safety) for the complete limit and error contract.
