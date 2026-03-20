package aeries

// RequestOptions describes one low-level API request.
//
// Most callers will use the typed service methods instead of constructing this
// directly, but the escape hatch is useful when the Aeries platform adds a new
// endpoint before the SDK ships a dedicated wrapper.
type RequestOptions struct {
	PathParams   map[string]string
	Query        map[string]string
	JSONBody     any
	Headers      map[string]string
	DatabaseYear string
}
