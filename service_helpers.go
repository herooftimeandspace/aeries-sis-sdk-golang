package aeries

import (
	"context"
	"strconv"
)

// doList calls one manifest-backed operation and decodes the result into a JSON list.
func (c *Client) doList(ctx context.Context, operationID string, opts RequestOptions) (JSONList, error) {
	var out JSONList
	if err := c.doOperation(ctx, operationID, opts, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// doDocument calls one manifest-backed operation and decodes the result into a JSON object.
func (c *Client) doDocument(ctx context.Context, operationID string, opts RequestOptions) (JSONDocument, error) {
	var out JSONDocument
	if err := c.doOperation(ctx, operationID, opts, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// doDocuments calls one manifest-backed operation and decodes the result into a slice of JSON objects.
func (c *Client) doDocuments(ctx context.Context, operationID string, opts RequestOptions) ([]JSONDocument, error) {
	list, err := c.doList(ctx, operationID, opts)
	if err != nil {
		return nil, err
	}
	return []JSONDocument(list), nil
}

// doNoContent calls one manifest-backed operation where Aeries documents an empty success response.
func (c *Client) doNoContent(ctx context.Context, operationID string, opts RequestOptions) error {
	return c.doOperation(ctx, operationID, opts, nil)
}

// intString converts an integer to a decimal string for a path parameter or query string.
func intString(value int) string {
	return strconv.Itoa(value)
}

// addQueryValue records a query parameter only when the caller supplied a meaningful value.
func addQueryValue(target map[string]string, key string, value string) {
	if value == "" {
		return
	}
	target[key] = value
}

// addIntQueryValue records an integer query parameter only when the caller supplied a non-zero value.
func addIntQueryValue(target map[string]string, key string, value int) {
	if value == 0 {
		return
	}
	target[key] = intString(value)
}

// cloneMap copies string keys and values so callers cannot mutate shared request state by accident.
func cloneMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	target := make(map[string]string, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}
