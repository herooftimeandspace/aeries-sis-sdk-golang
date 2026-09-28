package aeries

import "context"

// ResponseObserver receives the HTTP status code of one completed API attempt.
//
// The hook deliberately carries the status integer and nothing else. Response
// headers, response bytes, and the request URL are withheld so an audit or
// diagnostic sink cannot become a new escape path for provider payloads,
// student identifiers, or the Aeries certificate.
type ResponseObserver func(statusCode int)

// responseObserverKey is the unexported context key used to carry the observer.
type responseObserverKey struct{}

// WithResponseObserver returns a context that reports the HTTP status of every
// completed attempt made with it to the supplied function.
//
// The SDK's typed service methods build their own RequestOptions, so the
// observer travels on the context instead, in the same spirit as
// httptrace.WithClientTrace. That keeps the hook available to every service
// method and to raw Client.Do calls without changing a single method signature.
//
// The function is called once per attempt that produced an HTTP response, which
// means a retried safe read reports each attempt in order and the final call is
// the one the returned value or error came from. Attempts that never reached a
// response, such as a transport or context failure, report nothing. The call
// happens on the goroutine performing the request and before the response body
// is read, so an observer should record the status and return promptly.
//
// A nil observer clears any observer already attached to the context.
func WithResponseObserver(ctx context.Context, observe ResponseObserver) context.Context {
	return context.WithValue(ctx, responseObserverKey{}, observe)
}

// responseObserverFrom returns the observer attached to the context, if any.
func responseObserverFrom(ctx context.Context) ResponseObserver {
	if ctx == nil {
		return nil
	}
	observe, _ := ctx.Value(responseObserverKey{}).(ResponseObserver)
	return observe
}

// observeStatus reports one completed attempt's status when the caller asked for it.
func observeStatus(ctx context.Context, statusCode int) {
	if observe := responseObserverFrom(ctx); observe != nil {
		observe(statusCode)
	}
}
