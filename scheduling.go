package aeries

import "context"

// SchedulingService exposes courses, sections, rosters, and course request endpoints.
type SchedulingService struct {
	client *Client
}

// ListStudentClassSchedule returns class schedule rows for one student or many students.
func (s *SchedulingService) ListStudentClassSchedule(ctx context.Context, req SchoolStudentLookupRequest) ([]JSONDocument, error) {
	return s.client.doDocuments(ctx, "scheduling.list_student_class_schedule", RequestOptions{
		PathParams: map[string]string{
			"SchoolCode": req.SchoolCode,
			"StudentID":  intString(req.StudentID),
		},
		DatabaseYear: req.DatabaseYear,
	})
}

// ListCourses returns course rows and can narrow to one course ID.
func (s *SchedulingService) ListCourses(ctx context.Context, req CourseLookupRequest) ([]JSONDocument, error) {
	return s.client.doDocuments(ctx, "scheduling.list_courses", RequestOptions{
		PathParams:   map[string]string{"CourseID": req.CourseID},
		DatabaseYear: req.DatabaseYear,
	})
}

// ListCourseDataChanges returns course IDs that changed since the caller's sync cursor.
func (s *SchedulingService) ListCourseDataChanges(ctx context.Context, req TimestampRequest) ([]JSONDocument, error) {
	return s.client.doDocuments(ctx, "scheduling.list_course_data_changes", RequestOptions{
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

// ListSections returns section rows and can narrow to one section number.
func (s *SchedulingService) ListSections(ctx context.Context, req SectionLookupRequest) ([]JSONDocument, error) {
	return s.client.doDocuments(ctx, "scheduling.list_sections", RequestOptions{
		PathParams: map[string]string{
			"SchoolCode":    req.SchoolCode,
			"SectionNumber": intString(req.SectionNumber),
		},
		DatabaseYear: req.DatabaseYear,
	})
}

// ListSectionDataChanges returns section numbers that changed since the caller's sync cursor.
func (s *SchedulingService) ListSectionDataChanges(ctx context.Context, req TimestampRequest) ([]JSONDocument, error) {
	return s.client.doDocuments(ctx, "scheduling.list_section_data_changes", RequestOptions{
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

// CreateSection creates one scheduling master schedule section.
func (s *SchedulingService) CreateSection(ctx context.Context, req SectionCreateRequest) (JSONDocument, error) {
	return s.client.doDocument(ctx, "scheduling.create_section", RequestOptions{
		PathParams:   map[string]string{"SchoolCode": req.SchoolCode},
		JSONBody:     req.Values,
		DatabaseYear: req.DatabaseYear,
	})
}

// UpdateSection updates one scheduling master schedule section.
func (s *SchedulingService) UpdateSection(ctx context.Context, req SectionMutationRequest) (JSONDocument, error) {
	return s.client.doDocument(ctx, "scheduling.update_section", RequestOptions{
		PathParams: map[string]string{
			"SchoolCode":    req.SchoolCode,
			"SectionNumber": intString(req.SectionNumber),
		},
		JSONBody:     req.Values,
		DatabaseYear: req.DatabaseYear,
	})
}

// DeleteSection removes one scheduling master schedule section.
func (s *SchedulingService) DeleteSection(ctx context.Context, req SectionLookupRequest) error {
	return s.client.doNoContent(ctx, "scheduling.delete_section", RequestOptions{
		PathParams: map[string]string{
			"SchoolCode":    req.SchoolCode,
			"SectionNumber": intString(req.SectionNumber),
		},
		DatabaseYear: req.DatabaseYear,
	})
}

// ListSectionRoster returns the students assigned to one section.
func (s *SchedulingService) ListSectionRoster(ctx context.Context, req SectionLookupRequest) ([]JSONDocument, error) {
	return s.client.doDocuments(ctx, "scheduling.list_section_roster", RequestOptions{
		PathParams: map[string]string{
			"SchoolCode":    req.SchoolCode,
			"SectionNumber": intString(req.SectionNumber),
		},
		DatabaseYear: req.DatabaseYear,
	})
}

// ListSectionRosterDataChanges returns section roster changes since the caller's sync cursor.
func (s *SchedulingService) ListSectionRosterDataChanges(ctx context.Context, req TimestampRequest) ([]JSONDocument, error) {
	return s.client.doDocuments(ctx, "scheduling.list_section_roster_data_changes", RequestOptions{
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

// ListCourseRequests returns course request rows for one school.
func (s *SchedulingService) ListCourseRequests(ctx context.Context, req SchoolLookupRequest) ([]JSONDocument, error) {
	return s.client.doDocuments(ctx, "scheduling.list_course_requests", RequestOptions{
		PathParams:   map[string]string{"SchoolCode": req.SchoolCode},
		DatabaseYear: req.DatabaseYear,
	})
}

// ListStudentCourseRequests returns course request rows for one student and optional sequence number.
func (s *SchedulingService) ListStudentCourseRequests(ctx context.Context, req CourseRequestLookupRequest) ([]JSONDocument, error) {
	return s.client.doDocuments(ctx, "scheduling.list_student_course_requests", RequestOptions{
		PathParams: map[string]string{
			"SchoolCode":     req.SchoolCode,
			"StudentID":      intString(req.StudentID),
			"SequenceNumber": intString(req.SequenceNumber),
		},
		DatabaseYear: req.DatabaseYear,
	})
}

// CreateStudentCourseRequest creates one student course request row.
func (s *SchedulingService) CreateStudentCourseRequest(ctx context.Context, req CourseRequestCreateRequest) (JSONDocument, error) {
	return s.client.doDocument(ctx, "scheduling.create_student_course_request", RequestOptions{
		PathParams: map[string]string{
			"SchoolCode": req.SchoolCode,
			"StudentID":  intString(req.StudentID),
		},
		JSONBody:     req.Values,
		DatabaseYear: req.DatabaseYear,
	})
}

// UpdateStudentCourseRequest updates one student course request row by sequence number.
func (s *SchedulingService) UpdateStudentCourseRequest(ctx context.Context, req CourseRequestMutationRequest) (JSONDocument, error) {
	return s.client.doDocument(ctx, "scheduling.update_student_course_request", RequestOptions{
		PathParams: map[string]string{
			"SchoolCode":     req.SchoolCode,
			"StudentID":      intString(req.StudentID),
			"SequenceNumber": intString(req.SequenceNumber),
		},
		JSONBody:     req.Values,
		DatabaseYear: req.DatabaseYear,
	})
}

// DeleteStudentCourseRequest deletes one student course request row by sequence number.
func (s *SchedulingService) DeleteStudentCourseRequest(ctx context.Context, req CourseRequestMutationRequest) error {
	return s.client.doNoContent(ctx, "scheduling.delete_student_course_request", RequestOptions{
		PathParams: map[string]string{
			"SchoolCode":     req.SchoolCode,
			"StudentID":      intString(req.StudentID),
			"SequenceNumber": intString(req.SequenceNumber),
		},
		DatabaseYear: req.DatabaseYear,
	})
}

// ListAlternateCourseRequests returns alternate course request rows for one student.
func (s *SchedulingService) ListAlternateCourseRequests(ctx context.Context, req CourseRequestLookupRequest) ([]JSONDocument, error) {
	return s.client.doDocuments(ctx, "scheduling.list_alternate_course_requests", RequestOptions{
		PathParams: map[string]string{
			"SchoolCode":     req.SchoolCode,
			"StudentID":      intString(req.StudentID),
			"SequenceNumber": intString(req.SequenceNumber),
		},
		DatabaseYear: req.DatabaseYear,
	})
}

// CreateAlternateCourseRequest creates one alternate course request row.
func (s *SchedulingService) CreateAlternateCourseRequest(ctx context.Context, req CourseRequestCreateRequest) (JSONDocument, error) {
	return s.client.doDocument(ctx, "scheduling.create_alternate_course_request", RequestOptions{
		PathParams: map[string]string{
			"SchoolCode": req.SchoolCode,
			"StudentID":  intString(req.StudentID),
		},
		JSONBody:     req.Values,
		DatabaseYear: req.DatabaseYear,
	})
}

// UpdateAlternateCourseRequest updates one alternate course request row by sequence number.
func (s *SchedulingService) UpdateAlternateCourseRequest(ctx context.Context, req CourseRequestMutationRequest) (JSONDocument, error) {
	return s.client.doDocument(ctx, "scheduling.update_alternate_course_request", RequestOptions{
		PathParams: map[string]string{
			"SchoolCode":     req.SchoolCode,
			"StudentID":      intString(req.StudentID),
			"SequenceNumber": intString(req.SequenceNumber),
		},
		JSONBody:     req.Values,
		DatabaseYear: req.DatabaseYear,
	})
}

// DeleteAlternateCourseRequest deletes one alternate course request row by sequence number.
func (s *SchedulingService) DeleteAlternateCourseRequest(ctx context.Context, req CourseRequestMutationRequest) error {
	return s.client.doNoContent(ctx, "scheduling.delete_alternate_course_request", RequestOptions{
		PathParams: map[string]string{
			"SchoolCode":     req.SchoolCode,
			"StudentID":      intString(req.StudentID),
			"SequenceNumber": intString(req.SequenceNumber),
		},
		DatabaseYear: req.DatabaseYear,
	})
}
