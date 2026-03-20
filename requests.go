package aeries

// SystemInfoRequest selects the school year context for the system information endpoint.
type SystemInfoRequest struct {
	DatabaseYear string
}

// SchoolLookupRequest identifies a school-scoped read request.
type SchoolLookupRequest struct {
	SchoolCode   string
	DatabaseYear string
}

// SchoolDateRequest identifies a school and date for date-specific endpoints such as bell schedules.
type SchoolDateRequest struct {
	SchoolCode   string
	Date         string
	DatabaseYear string
}

// AbsenceCodeLookupRequest identifies one school and an optional absence code.
type AbsenceCodeLookupRequest struct {
	SchoolCode   string
	AbsenceCode  string
	DatabaseYear string
}

// CodeSetLookupRequest identifies the table and field used for a code set lookup.
type CodeSetLookupRequest struct {
	Table        string
	Field        string
	DatabaseYear string
}

// PreEnrollRequest identifies a pre-enrollment command request.
type PreEnrollRequest struct {
	StudentID    int
	Values       JSONDocument
	DatabaseYear string
}

// StudentLookupRequest identifies a student or student collection using district student IDs.
type StudentLookupRequest struct {
	SchoolCode     string
	StudentID      int
	StartingRecord int
	EndingRecord   int
	DatabaseYear   string
}

// StudentGradeLookupRequest identifies a school and grade level.
type StudentGradeLookupRequest struct {
	SchoolCode     string
	GradeLevel     string
	StartingRecord int
	EndingRecord   int
	DatabaseYear   string
}

// StudentNumberLookupRequest identifies a student by school-scoped student number.
type StudentNumberLookupRequest struct {
	SchoolCode    string
	StudentNumber string
	DatabaseYear  string
}

// StudentDataChangesRequest identifies a change feed window for one student data area.
type StudentDataChangesRequest struct {
	DataArea     string
	Year         int
	Month        int
	Day          int
	Hour         int
	Minute       int
	DatabaseYear string
}

// StudentCreateRequest inserts a brand-new district student record.
type StudentCreateRequest struct {
	SchoolCode   string
	Values       JSONDocument
	DatabaseYear string
}

// SchoolPayloadRequest carries one school-scoped mutation payload.
type SchoolPayloadRequest struct {
	SchoolCode   string
	Values       any
	DatabaseYear string
}

// StudentMutationRequest updates one student-scoped record.
type StudentMutationRequest struct {
	StudentID    int
	Values       JSONDocument
	DatabaseYear string
}

// StudentSequenceMutationRequest updates or deletes one student child record addressed by sequence number.
type StudentSequenceMutationRequest struct {
	StudentID      int
	SequenceNumber int
	Values         JSONDocument
	DatabaseYear   string
}

// ProgramLookupRequest identifies a student program lookup and optional program code filter.
type ProgramLookupRequest struct {
	SchoolCode   string
	StudentID    int
	ProgramCode  string
	DatabaseYear string
}

// TestScoresUpdateRequest inserts or updates test scores in bulk.
type TestScoresUpdateRequest struct {
	Items        []JSONDocument
	DatabaseYear string
}

// SchoolStudentLookupRequest identifies a school and an optional student ID.
type SchoolStudentLookupRequest struct {
	SchoolCode     string
	StudentID      int
	StartingRecord int
	EndingRecord   int
	DatabaseYear   string
}

// SchoolStudentNumberMutationRequest identifies a school and student number for school-scoped update endpoints.
type SchoolStudentNumberMutationRequest struct {
	SchoolCode    string
	StudentNumber string
	Values        JSONDocument
	DatabaseYear  string
}

// SchoolStringLookupRequest identifies a school and one extra string selector such as academic year or grade level.
type SchoolStringLookupRequest struct {
	SchoolCode     string
	Value          string
	StartingRecord int
	EndingRecord   int
	DatabaseYear   string
}

// AttendanceWindowRequest identifies a school attendance read plus optional date window.
type AttendanceWindowRequest struct {
	SchoolCode   string
	StudentID    int
	StartDate    string
	EndDate      string
	DatabaseYear string
}

// StudentAcademicYearRequest identifies one district student and one academic year.
type StudentAcademicYearRequest struct {
	StudentID    int
	AcademicYear string
	DatabaseYear string
}

// SchoolStudentAcademicYearRequest identifies one school, one student, and one academic year.
type SchoolStudentAcademicYearRequest struct {
	SchoolCode   string
	StudentID    int
	AcademicYear string
	DatabaseYear string
}

// TimestampRequest identifies a change-feed time window.
type TimestampRequest struct {
	Year         int
	Month        int
	Day          int
	Hour         int
	Minute       int
	DatabaseYear string
}

// StaffLookupRequest identifies one staff lookup by district staff ID.
type StaffLookupRequest struct {
	StaffID      int
	Filters      map[string]string
	DatabaseYear string
}

// StaffHRIDLookupRequest identifies one staff lookup by HR ID.
type StaffHRIDLookupRequest struct {
	HRID         string
	Filters      map[string]string
	DatabaseYear string
}

// StaffClassesRequest identifies one staff member for class and teacher bridge reads.
type StaffClassesRequest struct {
	StaffID      int
	DatabaseYear string
}

// TeacherLookupRequest identifies one school and an optional teacher number.
type TeacherLookupRequest struct {
	SchoolCode    string
	TeacherNumber int
	DatabaseYear  string
}

// CourseLookupRequest identifies one optional course ID lookup.
type CourseLookupRequest struct {
	CourseID     string
	DatabaseYear string
}

// SectionLookupRequest identifies one school and an optional section number.
type SectionLookupRequest struct {
	SchoolCode    string
	SectionNumber int
	DatabaseYear  string
}

// SectionMutationRequest identifies one section mutation and carries the request body.
type SectionMutationRequest struct {
	SchoolCode    string
	SectionNumber int
	Values        JSONDocument
	DatabaseYear  string
}

// SectionCreateRequest identifies one section create mutation and carries the request body.
type SectionCreateRequest struct {
	SchoolCode   string
	Values       JSONDocument
	DatabaseYear string
}

// CourseRequestLookupRequest identifies one school and one student's course request collection.
type CourseRequestLookupRequest struct {
	SchoolCode     string
	StudentID      int
	SequenceNumber int
	DatabaseYear   string
}

// CourseRequestMutationRequest identifies one course request mutation and carries the request body.
type CourseRequestMutationRequest struct {
	SchoolCode     string
	StudentID      int
	SequenceNumber int
	Values         JSONDocument
	DatabaseYear   string
}

// CourseRequestCreateRequest identifies one course request create mutation.
type CourseRequestCreateRequest struct {
	SchoolCode   string
	StudentID    int
	Values       JSONDocument
	DatabaseYear string
}

// GradebookByStaffRequest identifies one staff member for gradebook lookups.
type GradebookByStaffRequest struct {
	StaffID      int
	DatabaseYear string
}

// GradebookBySectionRequest identifies one section for gradebook lookups.
type GradebookBySectionRequest struct {
	SchoolCode    string
	SectionNumber int
	DatabaseYear  string
}

// GradebookLookupRequest identifies one gradebook number.
type GradebookLookupRequest struct {
	GradebookNumber int
	DatabaseYear    string
}

// AssignmentLookupRequest identifies one assignment by gradebook number and optional assignment number.
type AssignmentLookupRequest struct {
	GradebookNumber  int
	AssignmentNumber int
	UniqueID         string
	DatabaseYear     string
}

// AssignmentMutationRequest identifies one assignment mutation and carries the request body.
type AssignmentMutationRequest struct {
	GradebookNumber  int
	AssignmentNumber int
	UniqueID         string
	Values           JSONDocument
	DatabaseYear     string
}

// GradebookStudentsRequest identifies one gradebook term and an optional student filter.
type GradebookStudentsRequest struct {
	GradebookNumber int
	GradebookTerm   string
	StudentID       int
	DatabaseYear    string
}

// AssignmentScoresLookupRequest identifies one assignment score collection.
type AssignmentScoresLookupRequest struct {
	GradebookNumber  int
	AssignmentNumber int
	UniqueID         string
	StudentID        int
	DatabaseYear     string
}

// AssignmentScoresUpdateRequest identifies one assignment score mutation collection.
type AssignmentScoresUpdateRequest struct {
	GradebookNumber  int
	AssignmentNumber int
	UniqueID         string
	Items            []JSONDocument
	DatabaseYear     string
}

// AlertSendRequest identifies one alert name and carries the alert payload.
type AlertSendRequest struct {
	AlertName    string
	Values       JSONDocument
	DatabaseYear string
}
