# Go and Python Contract Parity

The Go and Python SDKs describe the same Aeries API, but their committed snapshots were created from different upstream documentation captures. This page records every known difference before SDK behavior changes. It gives a new contributor enough context to distinguish a deliberate compatibility decision from accidental contract drift.

## Snapshots compared

- Go snapshot: `internal/contract/source/endpoints.json`, schema version `2026-02-09.aeries-public-api`.
- Python snapshot: `src/aeries_sis_sdk/generated/contract_snapshot.json` at Python commit `7dc1ab3`, generated from the upstream capture dated `2026-03-20`.
- Comparison command: `make parity-check PYTHON_CONTRACT_SNAPSHOT=/absolute/path/to/contract_snapshot.json`.

The comparison reads only committed JSON snapshots. It does not fetch the Aeries website, inspect generated language wrappers, or modify the Python repository. `internal/contract/parity_disputes.json` is the machine-readable companion to this page. The command fails if either snapshot gains an operation or a difference that the ledger does not explain.

## Confirmed additive Go operations

The newer Python capture contains seven upstream-documented operations that the older Go inventory omitted. They are additive in Go and do not replace an existing method:

| Go method | Upstream verb and path | Why it is additive |
| --- | --- | --- |
| `PreEnroll.TriggerInactive` | `GET /api/v5/PreEnrollInactiveStudent/{StudentID}/{NextSchoolCode}` | A distinct command for an inactive student. The upstream article calls `NextSchoolCode` a school number and uses numeric `994` in its examples, so the Go request uses `int`, consistent with other numeric command identifiers. The endpoint returns an array containing all records for the student. Although the Python capture did not classify it as a side effect, pre-enrollment changes state, so Go marks it as a mutation and never retries it automatically. |
| `StudentGrades.ListGrades` | `GET /api/v5/schools/{SchoolCode}/Grades` | Reads the current student-grade collection; it does not replace the existing bulk update operation. |
| `Staff.Create` | `POST /api/v5/staff` | Creates a staff record. |
| `Staff.Update` | `PUT /api/v5/staff/{StaffID}` | Updates a staff record and coexists with the `GET` lookup on the same path. The upstream notes say a successful `200` update or `201` upsert returns the updated or created Staff object, so Go returns `JSONDocument` rather than treating success as empty. |
| `Students.CreateContact` | `POST /api/v5/InsertContact/{StudentID}` | Creates a contact and complements the existing update and delete methods. |
| `Students.ListDiscipline` | `GET /api/v5/schools/{SchoolCode}/Discipline/{StudentID}` | Reads the less-severe discipline history documented separately from assertive discipline. |
| `Scheduling.GetSchedulingSection` | `GET /api/v5/schools/{SchoolCode}/scheduling/sections/{SectionNumber}` | Reads one scheduling-master section. It is distinct from the general section endpoint and the existing create, update, and delete wrappers. |

## Upstream disputes held for confirmation

The following differences are recorded but do not change existing Go behavior in this issue. Changing them without upstream confirmation could silently break users whose Aeries installation follows the older contract.

| Area | Go snapshot | Python snapshot | Difference and current decision |
| --- | --- | --- | --- |
| Gradebook reads | `/api/v3/...` | `/api/v5/...` | API version differs for all gradebook operations. Preserve the existing v3 Go methods pending upstream confirmation. |
| Assignment create | `POST /api/v3/gradebooks/{GradebookNumber}/InsertAssignment` | `POST /api/v5/gradebooks/{GradebookNumber}/Assignments` | Version, final path segment, and segment casing differ. Preserve Go. |
| Assignment update | `POST /api/v3/.../assignments/{AssignmentNumber}` | `PUT /api/v5/.../assignments/{AssignmentNumber}` | Version and verb differ. Preserve Go. |
| Assignment update by ID | `POST /api/v3/gradebooks/Assignments/{UniqueID}` | `PUT /api/v5/gradebooks/Assignments/{UniqueID}` | Version and verb differ; `Assignments` casing agrees. Preserve Go. |
| Score updates | `POST .../UpdateScores` | `POST .../Scores` | Version and final path-segment casing/name differ for both assignment-number and unique-ID forms. Preserve Go. |
| Pre-enrollment command | `POST /api/v5/PreEnrollStudent/{StudentID}` | `GET /api/v5/commands/preenrollstudent/{StudentID}` | Verb, path prefix, and segment casing differ. Both snapshots classify the operation as a side effect. Preserve the existing Go method and its non-retry behavior. |
| Course change feed | `GET /api/v2/CourseDataChanges/{Year}/{Month}/{Day}/{Hour}/{Minute}` | `GET /api/v5/CourseDataChanges/{year}/{month}/{day}/{hour}/{minute}` | API version and placeholder casing differ. Preserve Go. |
| Other change feeds | Go uses exported-field casing such as `{Year}` and `{AcademicYear}` | Python uses source spelling such as `{year}` | Attendance-history-by-year, student-data, section-data, and section-roster change feeds differ only in placeholder spelling. Expanded URLs are equivalent, but the ledger retains each pair so generator field mapping drift is visible. |
| Valid marks | `/api/v5/schools/{SchoolCode}/Transcript/validmarks` | `/api/v5/schools/{SchoolCode}/validmarks` | The Python capture omits the `Transcript` segment. Preserve Go. |
| Test-score update | `POST /api/v5/testing/UpdateScores`, side effect | `GET /api/v5/testing/UpdateScores`, classified as a read | Verb and side-effect classification differ. Preserve the Go mutation because score updates must never be retried automatically. |

Placeholder names are compared case-sensitively in the report even though casing does not change an expanded URL. This catches generator drift such as `{Year}` becoming `{year}` and keeps request-field mappings reviewable.

## Go-only operations

The Go inventory includes 22 documented operations absent from the Python snapshot. They remain supported; parity is additive and does not mean reducing one SDK to the smaller snapshot. The exact stable IDs are:

- `alerts.send`
- `scheduling.create_alternate_course_request`
- `scheduling.create_student_course_request`
- `scheduling.delete_alternate_course_request`
- `scheduling.delete_section`
- `scheduling.delete_student_course_request`
- `scheduling.list_alternate_course_requests`
- `scheduling.list_course_requests`
- `scheduling.list_student_class_schedule`
- `scheduling.list_student_course_requests`
- `scheduling.update_alternate_course_request`
- `scheduling.update_section`
- `scheduling.update_student_course_request`
- `staff.get_by_hrid`
- `staff.list_data_changes`
- `staff.list_teacher_bridge`
- `staff.list_teachers`
- `student_grades.update_grades`
- `students.delete_contact`
- `students.list_available_tests`
- `students.update_contact`
- `students.update_school_supplemental`

## Retry contract

Only a manifest-backed operation explicitly classified as a non-mutation read is eligible for automatic retry after a transient network or server failure. Mutation verbs and side-effecting commands receive one attempt, including the `GET`-based inactive-student pre-enrollment command. Raw `Client.Do` calls also receive one attempt because the SDK cannot prove an arbitrary method/path pair is safe to repeat. Response-size failures remain non-retryable and retain the bounded, redacted diagnostics described in the configuration guide.
