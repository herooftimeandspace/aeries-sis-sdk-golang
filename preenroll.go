package aeries

import "context"

// PreEnrollService exposes command-style pre-enrollment endpoints.
type PreEnrollService struct {
	client *Client
}

// Trigger asks Aeries to run the documented student pre-enrollment action for one student.
func (s *PreEnrollService) Trigger(ctx context.Context, req PreEnrollRequest) (JSONDocument, error) {
	return s.client.doDocument(ctx, "preenroll.trigger", RequestOptions{
		PathParams:   map[string]string{"StudentID": intString(req.StudentID)},
		JSONBody:     req.Values,
		DatabaseYear: req.DatabaseYear,
	})
}
