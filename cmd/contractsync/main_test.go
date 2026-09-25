package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// projectRoot returns the repository root from the contractsync package test directory.
func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(dir, "..", ".."))
}

// copySourceFixtures copies the checked-in snapshot files into a temporary fake repository.
func copySourceFixtures(t *testing.T, destination string) {
	t.Helper()
	root := projectRoot(t)
	for _, name := range []string{"sources.json", "request_defaults.json", "endpoints.json", "service_wrappers.txt"} {
		sourcePath := filepath.Join(root, "internal", "contract", "source", name)
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatalf("read %s: %v", sourcePath, err)
		}
		targetPath := filepath.Join(destination, "internal", "contract", "source", name)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", targetPath, err)
		}
		if err := os.WriteFile(targetPath, data, 0o644); err != nil {
			t.Fatalf("write %s: %v", targetPath, err)
		}
	}
}

// TestGenerateArtifacts verifies that the source snapshots can be transformed into stable JSON outputs.
func TestGenerateArtifacts(t *testing.T) {
	sourceDir := filepath.Join(projectRoot(t), "internal", "contract", "source")
	manifestJSON, publicSurfaceJSON, err := generateArtifacts(sourceDir)
	if err != nil {
		t.Fatalf("generateArtifacts: %v", err)
	}
	if !bytes.Contains(manifestJSON, []byte(`"schema_version"`)) {
		t.Fatal("manifest output is missing schema_version")
	}
	if !bytes.Contains(publicSurfaceJSON, []byte(`"method_name"`)) {
		t.Fatal("public surface output is missing method_name")
	}
	if !bytes.Contains(publicSurfaceJSON, []byte(`"request_type"`)) || !bytes.Contains(publicSurfaceJSON, []byte(`"return_type"`)) {
		t.Fatal("public surface output is missing resolved method signature fields")
	}
}

// TestWrapperSurfaceHelpersRejectInvalidMetadata verifies signature goldens fail closed on malformed mappings and shapes.
func TestWrapperSurfaceHelpersRejectInvalidMetadata(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "service_wrappers.txt")
	if _, err := loadWrapperRequestTypes(filepath.Join(directory, "missing.txt")); err == nil {
		t.Fatal("missing wrapper metadata unexpectedly succeeded")
	}
	if err := os.WriteFile(path, []byte("malformed\n"), 0o644); err != nil {
		t.Fatalf("write malformed wrapper metadata: %v", err)
	}
	if _, err := loadWrapperRequestTypes(path); err == nil {
		t.Fatal("malformed wrapper metadata unexpectedly succeeded")
	}
	if err := os.WriteFile(path, []byte("one Request | comment\none OtherRequest | duplicate\n"), 0o644); err != nil {
		t.Fatalf("write wrapper metadata: %v", err)
	}
	if _, err := loadWrapperRequestTypes(path); err == nil {
		t.Fatal("duplicate wrapper metadata unexpectedly succeeded")
	}
	if _, err := publicReturnType(endpointSnapshot{ID: "one", ResponseShape: "binary"}); err == nil {
		t.Fatal("unsupported public return shape unexpectedly succeeded")
	}
	if _, err := publicReturnType(endpointSnapshot{ID: "system.get_info", ResponseShape: "list"}); err == nil {
		t.Fatal("system public return type accepted a non-object response shape")
	}
	for _, metadata := range []string{
		"one Request\n",
		"one Request | \n",
		"one Request payload | comment\n",
		"one Request body=DatabaseYear | comment\n",
		"one Request body=Values extra | comment\n",
	} {
		if err := os.WriteFile(path, []byte(metadata), 0o644); err != nil {
			t.Fatalf("write strict wrapper metadata fixture: %v", err)
		}
		if _, err := loadWrapperRequestTypes(path); err == nil {
			t.Fatalf("invalid wrapper metadata unexpectedly succeeded: %q", metadata)
		}
	}
}

// TestGenerateArtifactsFailsForMissingInputs verifies that input validation fails early when the source snapshot is incomplete.
func TestGenerateArtifactsFailsForMissingInputs(t *testing.T) {
	if _, _, err := generateArtifacts(t.TempDir()); err == nil {
		t.Fatal("expected missing source files to fail")
	}
}

// TestGenerateArtifactsFailsForIntermediateInputs verifies the error branches after sources.json has loaded successfully.
func TestGenerateArtifactsFailsForIntermediateInputs(t *testing.T) {
	tempRoot := t.TempDir()
	sourceDir := filepath.Join(tempRoot, "internal", "contract", "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "sources.json"), []byte(`{"sources":[]}`), 0o644); err != nil {
		t.Fatalf("write sources: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "request_defaults.json"), []byte(`{`), 0o644); err != nil {
		t.Fatalf("write defaults: %v", err)
	}
	if _, _, err := generateArtifacts(sourceDir); err == nil {
		t.Fatal("expected invalid request_defaults.json to fail")
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "request_defaults.json"), []byte(`{"request_defaults":{"accept_header":"application/json","certificate_header":"AERIES-CERT","database_year_query_key":"DatabaseYear","supported_formats":["json"],"notes":[]}}`), 0o644); err != nil {
		t.Fatalf("rewrite defaults: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "endpoints.json"), []byte(`{`), 0o644); err != nil {
		t.Fatalf("write endpoints: %v", err)
	}
	if _, _, err := generateArtifacts(sourceDir); err == nil {
		t.Fatal("expected invalid endpoints.json to fail")
	}
}

// TestGenerateArtifactsRejectsSignatureDrift verifies every endpoint must resolve to a complete Go method signature.
func TestGenerateArtifactsRejectsSignatureDrift(t *testing.T) {
	t.Run("missing request type", func(t *testing.T) {
		root := t.TempDir()
		copySourceFixtures(t, root)
		sourceDir := filepath.Join(root, "internal", "contract", "source")
		path := filepath.Join(sourceDir, "service_wrappers.txt")
		metadata, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read wrapper metadata: %v", err)
		}
		rowStart := bytes.Index(metadata, []byte("system.get_info "))
		if rowStart < 0 {
			t.Fatal("wrapper metadata has no system.get_info row")
		}
		rowEnd := bytes.IndexByte(metadata[rowStart:], '\n')
		if rowEnd < 0 {
			rowEnd = len(metadata) - rowStart
		} else {
			rowEnd++
		}
		metadata = append(metadata[:rowStart], metadata[rowStart+rowEnd:]...)
		if err := os.WriteFile(path, metadata, 0o644); err != nil {
			t.Fatalf("remove wrapper mapping: %v", err)
		}
		if _, _, err := generateArtifacts(sourceDir); err == nil {
			t.Fatal("missing request type unexpectedly succeeded")
		}
	})

	t.Run("unsupported return type", func(t *testing.T) {
		root := t.TempDir()
		copySourceFixtures(t, root)
		sourceDir := filepath.Join(root, "internal", "contract", "source")
		path := filepath.Join(sourceDir, "endpoints.json")
		metadata, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read endpoint metadata: %v", err)
		}
		metadata = bytes.ReplaceAll(metadata, []byte(`"response_shape": "object"`), []byte(`"response_shape": "binary"`))
		if err := os.WriteFile(path, metadata, 0o644); err != nil {
			t.Fatalf("write unsupported response shape: %v", err)
		}
		if _, _, err := generateArtifacts(sourceDir); err == nil {
			t.Fatal("unsupported return type unexpectedly succeeded")
		}
	})

	t.Run("stale wrapper mapping", func(t *testing.T) {
		root := t.TempDir()
		copySourceFixtures(t, root)
		sourceDir := filepath.Join(root, "internal", "contract", "source")
		path := filepath.Join(sourceDir, "service_wrappers.txt")
		file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			t.Fatalf("open wrapper metadata: %v", err)
		}
		if _, err := file.WriteString("stale.operation SystemInfoRequest | stale mapping\n"); err != nil {
			_ = file.Close()
			t.Fatalf("append stale mapping: %v", err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("close wrapper metadata: %v", err)
		}
		if _, _, err := generateArtifacts(sourceDir); err == nil {
			t.Fatal("stale wrapper mapping unexpectedly succeeded")
		}
	})
}

// TestCheckFile verifies that generated drift detection passes and fails in the expected cases.
func TestCheckFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.json")
	if err := os.WriteFile(path, []byte("{\n}\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := checkFile(path, []byte("{\n}\n")); err != nil {
		t.Fatalf("checkFile should succeed: %v", err)
	}
	if err := checkFile(path, []byte("{}\n")); err == nil {
		t.Fatal("checkFile should detect drift")
	}
	if err := checkFile(filepath.Join(t.TempDir(), "missing.json"), []byte("{}\n")); err == nil {
		t.Fatal("checkFile should fail for missing files")
	}
}

// TestRunWritesArtifacts verifies the success path of the CLI entry point without mutating the real repository.
func TestRunWritesArtifacts(t *testing.T) {
	tempRoot := t.TempDir()
	copySourceFixtures(t, tempRoot)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	if err := os.Chdir(tempRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := run([]string{"-validate"}); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tempRoot, "internal", "contract", "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("validate mode wrote manifest: %v", err)
	}
	if err := run(nil); err != nil {
		t.Fatalf("run: %v", err)
	}

	for _, relative := range []string{
		"internal/contract/manifest.json",
		"internal/contract/golden/public_surface.json",
	} {
		if _, err := os.Stat(filepath.Join(tempRoot, relative)); err != nil {
			t.Fatalf("expected %s to be generated: %v", relative, err)
		}
	}
}

// TestMainWritesArtifacts verifies that the top-level main function delegates to run on the success path.
func TestMainWritesArtifacts(t *testing.T) {
	tempRoot := t.TempDir()
	copySourceFixtures(t, tempRoot)

	originalArgs := os.Args
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		os.Args = originalArgs
		_ = os.Chdir(originalWD)
	})

	if err := os.Chdir(tempRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	os.Args = []string{"contractsync"}
	main()

	if _, err := os.Stat(filepath.Join(tempRoot, "internal", "contract", "manifest.json")); err != nil {
		t.Fatalf("expected main to write manifest: %v", err)
	}
}

// TestMainReportsErrors verifies the main wrapper's error branch without terminating the test process.
func TestMainReportsErrors(t *testing.T) {
	originalArgs := os.Args
	originalExit := exitWithError
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempRoot := t.TempDir()
	t.Cleanup(func() {
		os.Args = originalArgs
		exitWithError = originalExit
		_ = os.Chdir(originalWD)
	})

	if err := os.Chdir(tempRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	os.Args = []string{"contractsync"}
	called := false
	exitWithError = func(format string, args ...any) {
		called = true
	}
	main()
	if !called {
		t.Fatal("expected main to report an error when fixtures are missing")
	}
}

// TestRunCheckMode verifies the CLI's non-mutating drift-check path.
func TestRunCheckMode(t *testing.T) {
	tempRoot := t.TempDir()
	copySourceFixtures(t, tempRoot)

	manifestJSON, publicSurfaceJSON, err := generateArtifacts(filepath.Join(tempRoot, "internal", "contract", "source"))
	if err != nil {
		t.Fatalf("generateArtifacts: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tempRoot, "internal", "contract", "golden"), 0o755); err != nil {
		t.Fatalf("mkdir golden: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempRoot, "internal", "contract", "manifest.json"), manifestJSON, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempRoot, "internal", "contract", "golden", "public_surface.json"), publicSurfaceJSON, 0o644); err != nil {
		t.Fatalf("write public surface: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	if err := os.Chdir(tempRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := run([]string{"-check"}); err != nil {
		t.Fatalf("run -check: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempRoot, "internal", "contract", "golden", "public_surface.json"), []byte("[]\n"), 0o644); err != nil {
		t.Fatalf("write public surface drift: %v", err)
	}
	if err := run([]string{"-check"}); err == nil {
		t.Fatal("check mode missed public surface drift")
	}
}

// TestRunCheckModeReportsDrift verifies that drift check mode returns a normal error when files are missing.
func TestRunCheckModeReportsDrift(t *testing.T) {
	tempRoot := t.TempDir()
	copySourceFixtures(t, tempRoot)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	if err := os.Chdir(tempRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := run([]string{"-check"}); err == nil {
		t.Fatal("expected check mode to report drift when generated files are absent")
	}
}

// TestLoadJSONAndMarshalJSONErrors verifies helper error paths that normally only appear during bad input.
func TestLoadJSONAndMarshalJSONErrors(t *testing.T) {
	invalidPath := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("write invalid json: %v", err)
	}
	if err := loadJSON(invalidPath, &sourceFile{}); err == nil {
		t.Fatal("expected loadJSON to fail on invalid JSON")
	}
	if _, err := marshalJSON(make(chan int)); err == nil {
		t.Fatal("expected marshalJSON to fail on unsupported values")
	}
}

// TestRunReportsBadArgs verifies that unknown CLI flags are surfaced as normal errors.
func TestRunReportsBadArgs(t *testing.T) {
	if err := run([]string{"-definitely-invalid"}); err == nil {
		t.Fatal("expected invalid flags to fail")
	}
}

// TestRunReportsMissingInputs verifies that the CLI reports missing snapshot files in a plain error form.
func TestRunReportsMissingInputs(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := run(nil); err == nil {
		t.Fatal("expected run to fail when source files are missing")
	}
}

// TestRunReportsWriteFailures verifies that run returns write errors instead of hiding them.
func TestRunReportsWriteFailures(t *testing.T) {
	tempRoot := t.TempDir()
	copySourceFixtures(t, tempRoot)

	if err := os.MkdirAll(filepath.Join(tempRoot, "internal", "contract"), 0o755); err != nil {
		t.Fatalf("mkdir contract dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempRoot, "internal", "contract", "golden"), []byte("blocker"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	if err := os.Chdir(tempRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := run(nil); err == nil {
		t.Fatal("expected run to fail when writeArtifacts cannot create output directories")
	}
}

// TestWriteArtifactsReportsFilesystemErrors verifies the wrapped error paths when directories cannot be created.
func TestWriteArtifactsReportsFilesystemErrors(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	err := writeArtifacts(filepath.Join(filePath, "manifest.json"), filepath.Join(filePath, "surface.json"), []byte("{}\n"), []byte("[]\n"))
	if err == nil {
		t.Fatal("expected writeArtifacts to fail")
	}

	secondBlocker := filepath.Join(t.TempDir(), "second-blocker")
	if err := os.WriteFile(secondBlocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write second blocker: %v", err)
	}
	err = writeArtifacts(filepath.Join(t.TempDir(), "manifest.json"), filepath.Join(secondBlocker, "surface.json"), []byte("{}\n"), []byte("[]\n"))
	if err == nil {
		t.Fatal("expected writeArtifacts to fail for the public surface path")
	}

	manifestDir := filepath.Join(t.TempDir(), "manifest-dir")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	err = writeArtifacts(manifestDir, filepath.Join(t.TempDir(), "surface.json"), []byte("{}\n"), []byte("[]\n"))
	if err == nil {
		t.Fatal("expected writeArtifacts to fail when manifest path is a directory")
	}

	surfaceDir := filepath.Join(t.TempDir(), "surface-dir")
	if err := os.MkdirAll(surfaceDir, 0o755); err != nil {
		t.Fatalf("mkdir surface dir: %v", err)
	}
	err = writeArtifacts(filepath.Join(t.TempDir(), "manifest.json"), surfaceDir, []byte("{}\n"), []byte("[]\n"))
	if err == nil {
		t.Fatal("expected writeArtifacts to fail when public surface path is a directory")
	}
}

// TestExitfSubprocess verifies the helper that terminates the process with a message.
func TestExitfSubprocess(t *testing.T) {
	if os.Getenv("AERIES_CONTRACTSYNC_EXITF") == "1" {
		exitf("boom")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run", "TestExitfSubprocess")
	cmd.Env = append(os.Environ(), "AERIES_CONTRACTSYNC_EXITF=1")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected subprocess to fail")
	}
	if !bytes.Contains(output, []byte("boom")) {
		t.Fatalf("expected exit message in output, got %s", string(output))
	}
}
