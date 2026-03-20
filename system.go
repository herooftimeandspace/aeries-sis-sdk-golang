package aeries

import "context"

// SystemService exposes installation-wide Aeries endpoints.
type SystemService struct {
	client *Client
}

// GetInfo returns the installation metadata that Aeries documents for the current district.
func (s *SystemService) GetInfo(ctx context.Context, req SystemInfoRequest) (SystemInfo, error) {
	return s.client.doDocument(ctx, "system.get_info", RequestOptions{DatabaseYear: req.DatabaseYear})
}
