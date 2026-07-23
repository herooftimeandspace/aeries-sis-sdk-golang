package aeries

// JSONDocument represents a JSON object returned by the Aeries API.
//
// The official Aeries documentation is field-rich and endpoint-specific, so the
// SDK pairs these flexible documents with a vendored contract manifest that
// explains the expected shapes for each operation.
type JSONDocument map[string]any

// JSONList represents a list of JSON objects returned by the Aeries API.
type JSONList []JSONDocument

// SystemInfo is the distinct documented JSON object returned by the system information endpoint.
type SystemInfo map[string]any

// School is the documented JSON object returned by school-related endpoints.
type School = JSONDocument

// SchoolTerm is the documented JSON object returned by school term endpoints.
type SchoolTerm = JSONDocument

// SchoolCalendarEntry is the documented JSON object returned by school calendar endpoints.
type SchoolCalendarEntry = JSONDocument

// BellScheduleEntry is the documented JSON object returned by bell schedule endpoints.
type BellScheduleEntry = JSONDocument

// AbsenceCode is the documented JSON object returned by absence code endpoints.
type AbsenceCode = JSONDocument

// CodeSetValue is the documented JSON object returned by code set endpoints.
type CodeSetValue = JSONDocument

// Student is the documented JSON object returned by student-related endpoints.
type Student = JSONDocument

// Contact is the documented JSON object returned by student contact endpoints.
type Contact = JSONDocument

// ProgramRecord is the documented JSON object returned by program-related endpoints.
type ProgramRecord = JSONDocument

// TestRecord is the documented JSON object returned by testing endpoints.
type TestRecord = JSONDocument

// DisciplineRecord is the documented JSON object returned by assertive discipline endpoints.
type DisciplineRecord = JSONDocument

// SupplementalRecord is the documented JSON object returned by supplemental data endpoints.
type SupplementalRecord = JSONDocument

// StudentFee is the documented JSON object returned by fee endpoints.
type StudentFee = JSONDocument

// StudentPicture is the documented JSON object returned by picture endpoints.
type StudentPicture = JSONDocument

// StudentGroup is the documented JSON object returned by student group endpoints.
type StudentGroup = JSONDocument

// EnrollmentRecord is the documented JSON object returned by enrollment endpoints.
type EnrollmentRecord = JSONDocument

// AttendanceRecord is the documented JSON object returned by attendance endpoints.
type AttendanceRecord = JSONDocument

// StudentGradeRecord is the documented JSON object returned by student grade endpoints.
type StudentGradeRecord = JSONDocument

// GraduationStatus is the documented JSON object returned by graduation endpoints.
type GraduationStatus = JSONDocument

// TranscriptRecord is the documented JSON object returned by transcript endpoints.
type TranscriptRecord = JSONDocument

// StaffRecord is the documented JSON object returned by staff endpoints.
type StaffRecord = JSONDocument

// TeacherRecord is the documented JSON object returned by teacher endpoints.
type TeacherRecord = JSONDocument

// StaffAssignmentRecord is the documented JSON object returned by staff assignment endpoints.
type StaffAssignmentRecord = JSONDocument

// ClassScheduleEntry is the documented JSON object returned by schedule endpoints.
type ClassScheduleEntry = JSONDocument

// CourseRecord is the documented JSON object returned by course endpoints.
type CourseRecord = JSONDocument

// SectionRecord is the documented JSON object returned by section endpoints.
type SectionRecord = JSONDocument

// CourseRequestRecord is the documented JSON object returned by course request endpoints.
type CourseRequestRecord = JSONDocument

// AlternateCourseRequestRecord is the documented JSON object returned by alternate course request endpoints.
type AlternateCourseRequestRecord = JSONDocument

// GradebookRecord is the documented JSON object returned by gradebook endpoints.
type GradebookRecord = JSONDocument

// AssignmentRecord is the documented JSON object returned by gradebook assignment endpoints.
type AssignmentRecord = JSONDocument

// GradebookStudentRecord is the documented JSON object returned by gradebook student endpoints.
type GradebookStudentRecord = JSONDocument

// AssignmentScoreRecord is the documented JSON object returned by assignment score endpoints.
type AssignmentScoreRecord = JSONDocument

// AlertResult is the documented JSON object returned by alert endpoints.
type AlertResult = JSONDocument
