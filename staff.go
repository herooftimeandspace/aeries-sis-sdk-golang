package aeries

import "context"

// StaffService exposes staff, teacher, and staff change-feed endpoints.
type StaffService struct {
	client *Client
}

// List returns staff rows and can narrow to one staff ID.
func (s *StaffService) List(ctx context.Context, req StaffLookupRequest) ([]JSONDocument, error) {
	return s.client.doDocuments(ctx, "staff.list", RequestOptions{
		PathParams:   map[string]string{"StaffID": intString(req.StaffID)},
		Query:        cloneMap(req.Filters),
		DatabaseYear: req.DatabaseYear,
	})
}

// GetByHRID returns one staff row by HR ID.
func (s *StaffService) GetByHRID(ctx context.Context, req StaffHRIDLookupRequest) (JSONDocument, error) {
	return s.client.doDocument(ctx, "staff.get_by_hrid", RequestOptions{
		PathParams:   map[string]string{"HRID": req.HRID},
		Query:        cloneMap(req.Filters),
		DatabaseYear: req.DatabaseYear,
	})
}

// ListClasses returns the classes or sections assigned to one staff member.
func (s *StaffService) ListClasses(ctx context.Context, req StaffClassesRequest) ([]JSONDocument, error) {
	return s.client.doDocuments(ctx, "staff.list_classes", RequestOptions{
		PathParams:   map[string]string{"StaffID": intString(req.StaffID)},
		DatabaseYear: req.DatabaseYear,
	})
}

// ListDataChanges returns staff IDs that changed since the caller's sync cursor.
func (s *StaffService) ListDataChanges(ctx context.Context, req TimestampRequest) ([]JSONDocument, error) {
	return s.client.doDocuments(ctx, "staff.list_data_changes", RequestOptions{
		PathParams: map[string]string{
			"Year":   intString(req.Year),
			"Month":  intString(req.Month),
			"Day":    intString(req.Day),
			"Hour":   intString(req.Hour),
			"Minute": intString(req.Minute),
		},
		DatabaseYear: req.DatabaseYear,
	})
}

// ListTeachers returns teacher rows for one school and optional teacher number.
func (s *StaffService) ListTeachers(ctx context.Context, req TeacherLookupRequest) ([]JSONDocument, error) {
	return s.client.doDocuments(ctx, "staff.list_teachers", RequestOptions{
		PathParams: map[string]string{
			"SchoolCode":    req.SchoolCode,
			"TeacherNumber": intString(req.TeacherNumber),
		},
		DatabaseYear: req.DatabaseYear,
	})
}

// ListTeacherBridge returns teacher rows associated with one staff member.
func (s *StaffService) ListTeacherBridge(ctx context.Context, req StaffClassesRequest) ([]JSONDocument, error) {
	return s.client.doDocuments(ctx, "staff.list_teacher_bridge", RequestOptions{
		PathParams:   map[string]string{"StaffID": intString(req.StaffID)},
		DatabaseYear: req.DatabaseYear,
	})
}
