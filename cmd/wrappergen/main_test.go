package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFixture copies one repository input into a temporary checkout used for command-level testing.
func writeFixture(t *testing.T, source string, destination string) {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read fixture %s: %v", source, err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(destination, data, 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", destination, err)
	}
}

// repositoryRoot returns the checkout root while keeping test file lookups independent of the caller's directory.
func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}

// loadRepositoryInputs exercises the same checked-in inputs used by the command and returns their decoded forms.
func loadRepositoryInputs(t *testing.T) ([]endpoint, []requestSpec, map[string]map[string]fieldInfo) {
	t.Helper()
	root := repositoryRoot(t)
	endpoints, err := loadEndpoints(filepath.Join(root, "internal", "contract", "source", "endpoints.json"))
	if err != nil {
		t.Fatalf("load endpoints: %v", err)
	}
	specs, err := loadSpecs(filepath.Join(root, "internal", "contract", "source", "service_wrappers.txt"))
	if err != nil {
		t.Fatalf("load specs: %v", err)
	}
	fields, err := loadRequestFields(filepath.Join(root, "requests.go"))
	if err != nil {
		t.Fatalf("load request fields: %v", err)
	}
	return endpoints, specs, fields
}

// TestGenerateMatchesCheckedInWrappers verifies deterministic output for every service in the real contract inventory.
func TestGenerateMatchesCheckedInWrappers(t *testing.T) {
	endpoints, specs, fields := loadRepositoryInputs(t)
	outputs, err := generate(endpoints, specs, fields)
	if err != nil {
		t.Fatalf("generate wrappers: %v", err)
	}
	if len(outputs) != len(serviceComments) {
		t.Fatalf("generated %d services, want %d", len(outputs), len(serviceComments))
	}
	root := repositoryRoot(t)
	for name, expected := range outputs {
		actual, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(actual) != string(expected) {
			t.Fatalf("%s differs from generated content", name)
		}
	}
}

// TestRunWritesAndChecksTemporaryCheckout exercises both command modes without mutating the developer's checkout.
func TestRunWritesAndChecksTemporaryCheckout(t *testing.T) {
	repository := repositoryRoot(t)
	temporary := t.TempDir()
	for _, name := range []string{
		"internal/contract/source/endpoints.json",
		"internal/contract/source/service_wrappers.txt",
		"requests.go",
	} {
		writeFixture(t, filepath.Join(repository, name), filepath.Join(temporary, name))
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(temporary); err != nil {
		t.Fatalf("enter temporary checkout: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
	if err := run(nil); err != nil {
		t.Fatalf("write wrappers: %v", err)
	}
	if err := run([]string{"-check"}); err != nil {
		t.Fatalf("check wrappers: %v", err)
	}
	stalePath := filepath.Join(temporary, "stale.go")
	if err := os.WriteFile(stalePath, []byte(generatedWrapperHeader+"\npackage aeries\n"), 0o644); err != nil {
		t.Fatalf("write stale wrapper: %v", err)
	}
	if err := run([]string{"-check"}); err == nil || !strings.Contains(err.Error(), "stale generated wrapper") {
		t.Fatalf("stale wrapper check error = %v", err)
	}
	if err := run(nil); err != nil {
		t.Fatalf("remove stale wrapper: %v", err)
	}
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("stale wrapper still exists: %v", err)
	}
	if err := os.WriteFile(filepath.Join(temporary, "alerts.go"), []byte("drift\n"), 0o644); err != nil {
		t.Fatalf("introduce drift: %v", err)
	}
	blockedStalePath := filepath.Join(temporary, "blocked-stale.go")
	if err := os.WriteFile(blockedStalePath, []byte(generatedWrapperHeader+"\npackage aeries\n"), 0o644); err != nil {
		t.Fatalf("write blocked stale wrapper: %v", err)
	}
	if err := run([]string{"-check"}); err == nil || !strings.Contains(err.Error(), "non-generated file") {
		t.Fatalf("non-generated target error = %v", err)
	}
	if err := run(nil); err == nil || !strings.Contains(err.Error(), "non-generated file") {
		t.Fatalf("write preflight error = %v", err)
	}
	if _, err := os.Stat(blockedStalePath); err != nil {
		t.Fatalf("write preflight removed stale file before failing: %v", err)
	}
	if err := run([]string{"-unknown"}); err == nil {
		t.Fatal("unknown flag unexpectedly succeeded")
	}
}

// TestRunReportsSourceAndOutputFailures verifies command errors retain the failing generation stage.
func TestRunReportsSourceAndOutputFailures(t *testing.T) {
	repository := repositoryRoot(t)
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	t.Run("missing endpoints", func(t *testing.T) {
		temporary := t.TempDir()
		if err := os.Chdir(temporary); err != nil {
			t.Fatalf("enter temporary checkout: %v", err)
		}
		if err := run(nil); err == nil || !strings.Contains(err.Error(), "endpoint metadata") {
			t.Fatalf("run error = %v", err)
		}
	})

	t.Run("missing wrapper metadata", func(t *testing.T) {
		temporary := t.TempDir()
		writeFixture(t, filepath.Join(repository, "internal/contract/source/endpoints.json"), filepath.Join(temporary, "internal/contract/source/endpoints.json"))
		if err := os.Chdir(temporary); err != nil {
			t.Fatalf("enter temporary checkout: %v", err)
		}
		if err := run(nil); err == nil || !strings.Contains(err.Error(), "wrapper metadata") {
			t.Fatalf("run error = %v", err)
		}
	})

	t.Run("missing request source", func(t *testing.T) {
		temporary := t.TempDir()
		for _, name := range []string{"internal/contract/source/endpoints.json", "internal/contract/source/service_wrappers.txt"} {
			writeFixture(t, filepath.Join(repository, name), filepath.Join(temporary, name))
		}
		if err := os.Chdir(temporary); err != nil {
			t.Fatalf("enter temporary checkout: %v", err)
		}
		if err := run(nil); err == nil || !strings.Contains(err.Error(), "public request types") {
			t.Fatalf("run error = %v", err)
		}
	})

	t.Run("missing generated output", func(t *testing.T) {
		temporary := t.TempDir()
		for _, name := range []string{"internal/contract/source/endpoints.json", "internal/contract/source/service_wrappers.txt", "requests.go"} {
			writeFixture(t, filepath.Join(repository, name), filepath.Join(temporary, name))
		}
		if err := os.Chdir(temporary); err != nil {
			t.Fatalf("enter temporary checkout: %v", err)
		}
		if err := run([]string{"-check"}); err == nil || !strings.Contains(err.Error(), "read generated wrapper") {
			t.Fatalf("run error = %v", err)
		}
	})

	t.Run("invalid generation metadata", func(t *testing.T) {
		temporary := t.TempDir()
		for _, name := range []string{"internal/contract/source/endpoints.json", "requests.go"} {
			writeFixture(t, filepath.Join(repository, name), filepath.Join(temporary, name))
		}
		metadataPath := filepath.Join(temporary, "internal/contract/source/service_wrappers.txt")
		if err := os.MkdirAll(filepath.Dir(metadataPath), 0o755); err != nil {
			t.Fatalf("create metadata directory: %v", err)
		}
		if err := os.WriteFile(metadataPath, []byte("missing.operation SystemInfoRequest | missing operation\n"), 0o644); err != nil {
			t.Fatalf("write invalid generation metadata: %v", err)
		}
		if err := os.Chdir(temporary); err != nil {
			t.Fatalf("enter temporary checkout: %v", err)
		}
		if err := run(nil); err == nil || !strings.Contains(err.Error(), "has no wrapper request mapping") {
			t.Fatalf("run error = %v", err)
		}
	})

	t.Run("unreadable output target", func(t *testing.T) {
		temporary := t.TempDir()
		for _, name := range []string{"internal/contract/source/endpoints.json", "internal/contract/source/service_wrappers.txt", "requests.go"} {
			writeFixture(t, filepath.Join(repository, name), filepath.Join(temporary, name))
		}
		if err := os.Mkdir(filepath.Join(temporary, "alerts.go"), 0o755); err != nil {
			t.Fatalf("create unreadable output target: %v", err)
		}
		if err := os.Chdir(temporary); err != nil {
			t.Fatalf("enter temporary checkout: %v", err)
		}
		if err := run(nil); err == nil || !strings.Contains(err.Error(), "inspect generated wrapper") {
			t.Fatalf("run error = %v", err)
		}
	})
}

// TestGenerateRejectsMetadataDrift covers missing, extra, and invalid source-of-truth mappings.
func TestGenerateRejectsMetadataDrift(t *testing.T) {
	endpoints, specs, fields := loadRepositoryInputs(t)
	cases := []struct {
		name      string
		endpoints []endpoint
		specs     []requestSpec
		want      string
	}{
		{name: "missing mapping", endpoints: endpoints, specs: specs[1:], want: "has no wrapper request mapping"},
		{name: "extra mapping", endpoints: endpoints[:1], specs: specs, want: "entries but contract has"},
		{name: "unknown request", endpoints: endpoints[:1], specs: []requestSpec{{ID: endpoints[0].ID, Request: "MissingRequest"}}, want: "unknown request type"},
		{name: "empty inventories", endpoints: nil, specs: nil, want: "must not be empty"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := generate(test.endpoints, test.specs, fields)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("generate error = %v, want text %q", err, test.want)
			}
		})
	}
}

// TestRenderMethodRejectsInvalidShapes verifies generator errors identify bad contract response and parameter metadata.
func TestRenderMethodRejectsInvalidShapes(t *testing.T) {
	base := endpoint{ID: "sample", MethodName: "Sample", Summary: "Sample operation.", ResponseShape: "object"}
	cases := []struct {
		name      string
		operation endpoint
		fields    map[string]fieldInfo
		want      string
	}{
		{name: "response", operation: endpoint{ID: "sample", ResponseShape: "binary"}, fields: map[string]fieldInfo{}, want: "unsupported response shape"},
		{name: "path", operation: func() endpoint { value := base; value.PathParameters = []string{"Missing"}; return value }(), fields: map[string]fieldInfo{}, want: "path parameter"},
		{name: "query", operation: func() endpoint { value := base; value.QueryParameters = []string{"Missing"}; return value }(), fields: map[string]fieldInfo{}, want: "query parameter"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			spec := requestSpec{Request: "SampleRequest", Comment: "runs the sample operation."}
			_, err := renderMethod("SampleService", test.operation, spec, test.fields)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("renderMethod error = %v, want text %q", err, test.want)
			}
		})
	}
}

// TestSmallMetadataHelpers covers stable aliases, response bindings, query filtering, and filenames.
func TestSmallMetadataHelpers(t *testing.T) {
	if got := requestField("code", "ProgramLookupRequest"); got != "ProgramCode" {
		t.Fatalf("code field = %q", got)
	}
	if got := requestField("AcademicYear", "SchoolStringLookupRequest"); got != "Value" {
		t.Fatalf("academic year field = %q", got)
	}
	if got := requestField("SchoolCode", "SchoolLookupRequest"); got != "SchoolCode" {
		t.Fatalf("ordinary field = %q", got)
	}
	if got := withoutDatabaseYear([]string{"StartingRecord", "DatabaseYear"}); len(got) != 1 || got[0] != "StartingRecord" {
		t.Fatalf("filtered query parameters = %#v", got)
	}
	if got, err := serviceFilename("StudentGrades"); err != nil || got != "student_grades.go" {
		t.Fatalf("student grades filename = %q, %v", got, err)
	}
	if got, err := serviceFilename("Schools"); err != nil || got != "schools.go" {
		t.Fatalf("schools filename = %q, %v", got, err)
	}
	if _, err := serviceFilename("../outside"); err == nil || !strings.Contains(err.Error(), "unknown wrapper service") {
		t.Fatalf("unsafe service error = %v", err)
	}
	for _, shape := range []string{"object", "list", "empty"} {
		if _, _, err := responseBinding(endpoint{ID: "sample", ResponseShape: shape}); err != nil {
			t.Fatalf("responseBinding(%q): %v", shape, err)
		}
	}
	if result, helper, err := responseBinding(endpoint{ID: "system.get_info"}); err != nil || result != "(SystemInfo, error)" || helper != "doSystemInfo" {
		t.Fatalf("system binding = %q, %q, %v", result, helper, err)
	}
}

// TestLoadSpecsRejectsMalformedMetadata verifies line-level diagnostics for maintainers editing the compact mapping.
func TestLoadSpecsRejectsMalformedMetadata(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "specs.txt")
	for _, test := range []struct {
		name string
		text string
		want string
	}{
		{name: "malformed", text: "one two three\n", want: "must contain"},
		{name: "missing comment", text: "one Request\n", want: "public method comment"},
		{name: "invalid body", text: "one Request payload | comment\n", want: "invalid body field"},
		{name: "empty body", text: "one Request body= | comment\n", want: "invalid body field"},
		{name: "too many fields", text: "one Request body=Values extra | comment\n", want: "optional body field"},
		{name: "duplicate", text: "one Request | comment\none Request | comment\n", want: "duplicate"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(test.text), 0o644); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			_, err := loadSpecs(path)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("loadSpecs error = %v, want text %q", err, test.want)
			}
		})
	}
}

// TestLoadersReportInvalidInputs covers actionable diagnostics for missing and malformed source files.
func TestLoadersReportInvalidInputs(t *testing.T) {
	directory := t.TempDir()
	missing := filepath.Join(directory, "missing")
	if _, err := loadEndpoints(missing); err == nil || !strings.Contains(err.Error(), "read endpoint metadata") {
		t.Fatalf("missing endpoints error = %v", err)
	}
	if _, err := loadSpecs(missing); err == nil || !strings.Contains(err.Error(), "open wrapper metadata") {
		t.Fatalf("missing specs error = %v", err)
	}
	if _, err := loadRequestFields(missing); err == nil || !strings.Contains(err.Error(), "parse public request types") {
		t.Fatalf("missing requests error = %v", err)
	}
	invalidJSON := filepath.Join(directory, "invalid.json")
	if err := os.WriteFile(invalidJSON, []byte("not json"), 0o644); err != nil {
		t.Fatalf("write invalid JSON: %v", err)
	}
	if _, err := loadEndpoints(invalidJSON); err == nil || !strings.Contains(err.Error(), "decode endpoint metadata") {
		t.Fatalf("invalid endpoints error = %v", err)
	}
	invalidGo := filepath.Join(directory, "invalid.go")
	if err := os.WriteFile(invalidGo, []byte("package"), 0o644); err != nil {
		t.Fatalf("write invalid Go: %v", err)
	}
	if _, err := loadRequestFields(invalidGo); err == nil {
		t.Fatal("invalid request source unexpectedly parsed")
	}

	// A valid JSON object without endpoints proves empty inventories decode cleanly.
	emptyJSON := filepath.Join(directory, "empty.json")
	data, err := json.Marshal(endpointFile{})
	if err != nil {
		t.Fatalf("marshal empty endpoint file: %v", err)
	}
	if err := os.WriteFile(emptyJSON, data, 0o644); err != nil {
		t.Fatalf("write empty endpoint file: %v", err)
	}
	if endpoints, err := loadEndpoints(emptyJSON); err != nil || len(endpoints) != 0 {
		t.Fatalf("empty endpoints = %#v, %v", endpoints, err)
	}
	variedGo := filepath.Join(directory, "varied.go")
	if err := os.WriteFile(variedGo, []byte("package sample\nfunc helper() {}\ntype Alias string\ntype Request struct { Embedded\n Count int\n}\n"), 0o644); err != nil {
		t.Fatalf("write varied Go: %v", err)
	}
	if requests, err := loadRequestFields(variedGo); err != nil || !requests["Request"]["Count"].Integer {
		t.Fatalf("varied request fields = %#v, %v", requests, err)
	}
}

// TestRenderServiceReportsFormattingFailure exercises the final generated-Go validation boundary.
func TestRenderServiceReportsFormattingFailure(t *testing.T) {
	operation := endpoint{ID: "sample", Service: "Sample", MethodName: "not-valid", Summary: "Invalid method for testing.", ResponseShape: "object"}
	_, err := renderService("Sample", []endpoint{operation}, map[string]requestSpec{"sample": {ID: "sample", Request: "Request"}}, map[string]map[string]fieldInfo{"Request": {}})
	if err == nil || !strings.Contains(err.Error(), "format generated") {
		t.Fatalf("renderService error = %v", err)
	}
}

// TestRenderMethodUsesOnlyExplicitBodyMetadata verifies request structs do not accidentally add DELETE payloads.
func TestRenderMethodUsesOnlyExplicitBodyMetadata(t *testing.T) {
	operation := endpoint{ID: "sample.delete", MethodName: "Delete", HTTPMethod: "DELETE", ResponseShape: "empty", Mutation: true}
	fields := map[string]fieldInfo{"Values": {}}
	withoutBody, err := renderMethod("SampleService", operation, requestSpec{Request: "SampleRequest", Comment: "deletes the sample."}, fields)
	if err != nil {
		t.Fatalf("render body-free method: %v", err)
	}
	if strings.Contains(withoutBody, "JSONBody") {
		t.Fatalf("body-free operation generated a payload:\n%s", withoutBody)
	}
	withBodyOperation := operation
	withBodyOperation.ID = "sample.update"
	withBodyOperation.MethodName = "Update"
	withBodyOperation.HTTPMethod = "PUT"
	withBody, err := renderMethod("SampleService", withBodyOperation, requestSpec{Request: "SampleRequest", Body: "Values", Comment: "updates the sample."}, fields)
	if err != nil {
		t.Fatalf("render explicit body method: %v", err)
	}
	if !strings.Contains(withBody, "JSONBody: req.Values") {
		t.Fatalf("explicit body operation omitted its payload:\n%s", withBody)
	}
	_, err = renderMethod("SampleService", withBodyOperation, requestSpec{Request: "SampleRequest", Comment: "updates the sample."}, fields)
	if err == nil || !strings.Contains(err.Error(), "must declare body=Values") {
		t.Fatalf("missing body metadata error = %v", err)
	}
	_, err = renderMethod("SampleService", operation, requestSpec{Request: "SampleRequest", Body: "Values", Comment: "deletes the sample."}, fields)
	if err == nil || !strings.Contains(err.Error(), "invalid body field") {
		t.Fatalf("delete body metadata error = %v", err)
	}
	_, err = renderMethod("SampleService", withBodyOperation, requestSpec{Request: "SampleRequest", Body: "DatabaseYear", Comment: "updates the sample."}, map[string]fieldInfo{"DatabaseYear": {}})
	if err == nil || !strings.Contains(err.Error(), "invalid body field") {
		t.Fatalf("non-payload body metadata error = %v", err)
	}
}

// TestGenerateRejectsFilenameCollisions verifies two services cannot silently replace one generated output.
func TestGenerateRejectsFilenameCollisions(t *testing.T) {
	serviceComments["schools"] = "collision fixture"
	t.Cleanup(func() { delete(serviceComments, "schools") })
	endpoints := []endpoint{
		{ID: "one", Service: "Schools", MethodName: "One", HTTPMethod: "GET", ResponseShape: "object"},
		{ID: "two", Service: "schools", MethodName: "Two", HTTPMethod: "GET", ResponseShape: "object"},
	}
	specs := []requestSpec{
		{ID: "one", Request: "Request", Comment: "gets one."},
		{ID: "two", Request: "Request", Comment: "gets two."},
	}
	_, err := generate(endpoints, specs, map[string]map[string]fieldInfo{"Request": {}})
	if err == nil || !strings.Contains(err.Error(), "duplicate generated filename") {
		t.Fatalf("collision error = %v", err)
	}
}

// TestFindGeneratedWrappersReportsIOFailures verifies filesystem diagnostics identify discovery and read failures.
func TestFindGeneratedWrappersReportsIOFailures(t *testing.T) {
	t.Run("root is not a directory", func(t *testing.T) {
		rootFile := filepath.Join(t.TempDir(), "root-file")
		if err := os.WriteFile(rootFile, []byte("not a directory"), 0o644); err != nil {
			t.Fatalf("write root fixture: %v", err)
		}
		if _, err := findGeneratedWrappers(rootFile); err == nil || !strings.Contains(err.Error(), "list generated wrappers") {
			t.Fatalf("root discovery error = %v", err)
		}
	})

	t.Run("generated candidate cannot be read", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Symlink(filepath.Join(root, "missing-target"), filepath.Join(root, "broken.go")); err != nil {
			t.Fatalf("create broken generated candidate: %v", err)
		}
		if _, err := findGeneratedWrappers(root); err == nil || !strings.Contains(err.Error(), "read possible generated wrapper") {
			t.Fatalf("candidate read error = %v", err)
		}
	})
}
