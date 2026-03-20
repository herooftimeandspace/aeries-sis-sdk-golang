package aeries

import "context"

// AlertsService exposes the documented alert send endpoint.
type AlertsService struct {
	client *Client
}

// Send triggers one alert or action alert by alert name.
func (s *AlertsService) Send(ctx context.Context, req AlertSendRequest) (JSONDocument, error) {
	return s.client.doDocument(ctx, "alerts.send", RequestOptions{
		PathParams:   map[string]string{"AlertName": req.AlertName},
		JSONBody:     req.Values,
		DatabaseYear: req.DatabaseYear,
	})
}
