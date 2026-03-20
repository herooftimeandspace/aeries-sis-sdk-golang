package aeries

import "context"

// CodeSetsService exposes code set lookups used by many other Aeries objects.
type CodeSetsService struct {
	client *Client
}

// List returns the code values documented for one table and field pair.
func (s *CodeSetsService) List(ctx context.Context, req CodeSetLookupRequest) ([]JSONDocument, error) {
	return s.client.doDocuments(ctx, "codesets.list", RequestOptions{
		PathParams: map[string]string{
			"Table": req.Table,
			"Field": req.Field,
		},
		DatabaseYear: req.DatabaseYear,
	})
}
