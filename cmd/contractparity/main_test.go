package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCompareAcceptsEquivalentDisputedAndLanguageOnlyOperations covers every accounting path in one small fixture.
func TestCompareAcceptsEquivalentDisputedAndLanguageOnlyOperations(t *testing.T) {
	goOperations := []goOperation{
		{ID: "same", HTTPMethod: "GET", PathTemplate: "/api/v5/items/{ID}"},
		{ID: "dispute", HTTPMethod: "POST", PathTemplate: "/api/v3/Commands/{Year}", Mutation: true},
		{ID: "go-only", HTTPMethod: "GET", PathTemplate: "/api/v5/go"},
	}
	pythonOperations := []pythonOperation{
		{ID: "py-same", HTTPMethod: "GET", PathTemplate: "/api/v5/items/{ID}"},
		{ID: "py-dispute", HTTPMethod: "GET", PathTemplate: "/api/v5/commands/{year}"},
		{ID: "py-only", HTTPMethod: "GET", PathTemplate: "/api/v5/python"},
	}
	ledger := disputeLedger{
		Pairs:  []disputePair{{GoID: "dispute", PythonID: "py-dispute", Differences: []string{"verb", "api_version", "path", "segment_casing", "placeholder_casing", "side_effect"}}},
		GoOnly: []string{"go-only"}, PythonOnly: []string{"py-only"},
	}
	report, err := compare(goOperations, pythonOperations, ledger)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if report.Equivalent != 1 || report.DisputedPairs != 1 || report.GoOnly != 1 || report.PythonOnly != 1 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

// TestCompareRejectsDrift exercises stale, duplicate, and unexplained ledger states.
func TestCompareRejectsDrift(t *testing.T) {
	baseGo := []goOperation{{ID: "go", HTTPMethod: "GET", PathTemplate: "/api/v5/items"}}
	basePython := []pythonOperation{{ID: "python", HTTPMethod: "GET", PathTemplate: "/api/v5/items"}}
	tests := []struct {
		name   string
		goOps  []goOperation
		pyOps  []pythonOperation
		ledger disputeLedger
	}{
		{name: "unaccounted", goOps: append(baseGo, goOperation{ID: "extra"}), pyOps: basePython},
		{name: "missing pair", goOps: baseGo, pyOps: basePython, ledger: disputeLedger{Pairs: []disputePair{{GoID: "missing", PythonID: "python"}}}},
		{name: "stale differences", goOps: baseGo, pyOps: basePython, ledger: disputeLedger{Pairs: []disputePair{{GoID: "go", PythonID: "python", Differences: []string{"verb"}}}}},
		{name: "duplicate go id", goOps: append(baseGo, baseGo[0]), pyOps: basePython},
		{name: "duplicate python id", goOps: baseGo, pyOps: append(basePython, basePython[0])},
		{name: "empty go id", goOps: []goOperation{{}}, pyOps: basePython},
		{name: "empty python id", goOps: baseGo, pyOps: []pythonOperation{{}}},
		{name: "missing python pair", goOps: baseGo, pyOps: basePython, ledger: disputeLedger{Pairs: []disputePair{{GoID: "go", PythonID: "missing"}}}},
		{name: "reused pair", goOps: baseGo, pyOps: basePython, ledger: disputeLedger{Pairs: []disputePair{{GoID: "go", PythonID: "python"}, {GoID: "go", PythonID: "python"}}}},
		{name: "missing only operation", goOps: baseGo, pyOps: basePython, ledger: disputeLedger{GoOnly: []string{"missing"}}},
		{
			name:  "duplicate only id",
			goOps: append(baseGo, goOperation{ID: "extra"}),
			pyOps: basePython,
			ledger: disputeLedger{
				GoOnly: []string{"extra", "extra"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := compare(tt.goOps, tt.pyOps, tt.ledger); err == nil {
				t.Fatal("expected comparison failure")
			}
		})
	}
}

// TestRunReadsFilesAndRequiresPythonSnapshot verifies the command boundary and its human-readable setup error.
func TestRunReadsFilesAndRequiresPythonSnapshot(t *testing.T) {
	t.Setenv("PYTHON_CONTRACT_SNAPSHOT", "")
	if err := run(nil); err == nil || !strings.Contains(err.Error(), "python snapshot is required") {
		t.Fatalf("run without Python snapshot = %v", err)
	}
	directory := t.TempDir()
	goPath := filepath.Join(directory, "go.json")
	pythonPath := filepath.Join(directory, "python.json")
	ledgerPath := filepath.Join(directory, "ledger.json")
	for path, content := range map[string]string{
		goPath:     `{"endpoints":[{"id":"go","http_method":"GET","path_template":"/api/v5/items","mutation":false}]}`,
		pythonPath: `{"operations":[{"operation_id":"python","method":"GET","path_template":"/api/v5/items","side_effect":false}]}`,
		ledgerPath: `{}`,
	} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}
	if err := run([]string{"-go", goPath, "-python", pythonPath, "-ledger", ledgerPath}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if err := run([]string{"-go", filepath.Join(directory, "missing"), "-python", pythonPath, "-ledger", ledgerPath}); err == nil {
		t.Fatal("expected missing Go snapshot to fail")
	}
	if err := run([]string{"-go", goPath, "-python", pythonPath, "-ledger", filepath.Join(directory, "missing")}); err == nil {
		t.Fatal("expected missing ledger to fail")
	}
	if err := run([]string{"-unknown"}); err == nil {
		t.Fatal("expected unknown flag to fail")
	}
	if err := os.WriteFile(goPath, []byte("{"), 0o600); err != nil {
		t.Fatalf("write invalid fixture: %v", err)
	}
	if err := run([]string{"-go", goPath, "-python", pythonPath, "-ledger", ledgerPath}); err == nil {
		t.Fatal("expected invalid Go snapshot to fail")
	}
}
