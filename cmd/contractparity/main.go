package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

type goSnapshot struct {
	Endpoints []goOperation `json:"endpoints"`
}

type goOperation struct {
	ID           string `json:"id"`
	HTTPMethod   string `json:"http_method"`
	PathTemplate string `json:"path_template"`
	Mutation     bool   `json:"mutation"`
}

type pythonSnapshot struct {
	Operations []pythonOperation `json:"operations"`
}

type pythonOperation struct {
	ID           string `json:"operation_id"`
	HTTPMethod   string `json:"method"`
	PathTemplate string `json:"path_template"`
	SideEffect   bool   `json:"side_effect"`
}

type disputeLedger struct {
	Pairs      []disputePair `json:"pairs"`
	GoOnly     []string      `json:"go_only"`
	PythonOnly []string      `json:"python_only"`
}

type disputePair struct {
	GoID        string   `json:"go_id"`
	PythonID    string   `json:"python_id"`
	Differences []string `json:"differences"`
}

type comparisonReport struct {
	GoCount       int
	PythonCount   int
	Equivalent    int
	DisputedPairs int
	GoOnly        int
	PythonOnly    int
}

var placeholderPattern = regexp.MustCompile(`\{([^{}]+)\}`)
var apiVersionPattern = regexp.MustCompile(`(?i)^/api/v[0-9]+`)

// main reports parity failures without a stack trace so the command is suitable for local gates and CI logs.
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run loads both committed snapshots and the checked-in dispute ledger before printing a stable summary.
func run(args []string) error {
	flags := flag.NewFlagSet("contractparity", flag.ContinueOnError)
	goPath := flags.String("go", "internal/contract/source/endpoints.json", "path to the committed Go endpoint snapshot")
	pythonPath := flags.String("python", os.Getenv("PYTHON_CONTRACT_SNAPSHOT"), "path to the committed Python contract snapshot")
	ledgerPath := flags.String("ledger", "internal/contract/parity_disputes.json", "path to the checked-in dispute ledger")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*pythonPath) == "" {
		return fmt.Errorf("python snapshot is required: pass -python or set PYTHON_CONTRACT_SNAPSHOT")
	}
	var goValue goSnapshot
	if err := loadJSON(*goPath, &goValue); err != nil {
		return fmt.Errorf("load Go snapshot: %w", err)
	}
	var pythonValue pythonSnapshot
	if err := loadJSON(*pythonPath, &pythonValue); err != nil {
		return fmt.Errorf("load Python snapshot: %w", err)
	}
	var ledger disputeLedger
	if err := loadJSON(*ledgerPath, &ledger); err != nil {
		return fmt.Errorf("load dispute ledger: %w", err)
	}
	report, err := compare(goValue.Endpoints, pythonValue.Operations, ledger)
	if err != nil {
		return err
	}
	fmt.Printf("contract parity verified: Go=%d Python=%d equivalent=%d disputed=%d Go-only=%d Python-only=%d\n", report.GoCount, report.PythonCount, report.Equivalent, report.DisputedPairs, report.GoOnly, report.PythonOnly)
	return nil
}

// loadJSON decodes one snapshot or ledger while preserving the original file as the reviewable source of truth.
func loadJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return err
	}
	return nil
}

// compare accounts for every operation exactly once and rejects undocumented or stale ledger entries.
func compare(goOperations []goOperation, pythonOperations []pythonOperation, ledger disputeLedger) (comparisonReport, error) {
	report := comparisonReport{GoCount: len(goOperations), PythonCount: len(pythonOperations)}
	goByID, err := indexGo(goOperations)
	if err != nil {
		return report, err
	}
	pythonByID, err := indexPython(pythonOperations)
	if err != nil {
		return report, err
	}
	usedGo := map[string]bool{}
	usedPython := map[string]bool{}
	for _, pair := range ledger.Pairs {
		goOperation, ok := goByID[pair.GoID]
		if !ok {
			return report, fmt.Errorf("dispute ledger references missing Go operation %q", pair.GoID)
		}
		pythonOperation, ok := pythonByID[pair.PythonID]
		if !ok {
			return report, fmt.Errorf("dispute ledger references missing Python operation %q", pair.PythonID)
		}
		if usedGo[pair.GoID] || usedPython[pair.PythonID] {
			return report, fmt.Errorf("dispute ledger reuses operation pair %q / %q", pair.GoID, pair.PythonID)
		}
		actual := operationDifferences(goOperation, pythonOperation)
		if !sameStrings(actual, pair.Differences) {
			return report, fmt.Errorf("dispute %s / %s differences = %v, ledger = %v", pair.GoID, pair.PythonID, actual, sortedCopy(pair.Differences))
		}
		usedGo[pair.GoID], usedPython[pair.PythonID] = true, true
		report.DisputedPairs++
	}

	for _, goOperation := range goOperations {
		if usedGo[goOperation.ID] {
			continue
		}
		matches := []pythonOperation{}
		for _, pythonOperation := range pythonOperations {
			if usedPython[pythonOperation.ID] {
				continue
			}
			if normalizedRoute(goOperation.HTTPMethod, goOperation.PathTemplate) == normalizedRoute(pythonOperation.HTTPMethod, pythonOperation.PathTemplate) {
				matches = append(matches, pythonOperation)
			}
		}
		if len(matches) != 1 {
			continue
		}
		if differences := operationDifferences(goOperation, matches[0]); len(differences) != 0 {
			continue
		}
		usedGo[goOperation.ID], usedPython[matches[0].ID] = true, true
		report.Equivalent++
	}

	goOnlyErr := accountOnly("Go", ledger.GoOnly, goByID, usedGo)
	pythonOnlyErr := accountOnly("Python", ledger.PythonOnly, pythonByID, usedPython)
	if goOnlyErr != nil || pythonOnlyErr != nil {
		return report, fmt.Errorf("parity accounting failed: Go: %v; Python: %v", goOnlyErr, pythonOnlyErr)
	}
	report.GoOnly, report.PythonOnly = len(ledger.GoOnly), len(ledger.PythonOnly)
	return report, nil
}

// indexGo rejects duplicate operation IDs because duplicates make parity results order-dependent.
func indexGo(operations []goOperation) (map[string]goOperation, error) {
	indexed := make(map[string]goOperation, len(operations))
	for _, operation := range operations {
		if _, exists := indexed[operation.ID]; exists || operation.ID == "" {
			return nil, fmt.Errorf("Go snapshot contains duplicate or empty operation id %q", operation.ID)
		}
		indexed[operation.ID] = operation
	}
	return indexed, nil
}

// indexPython rejects duplicate operation IDs for the same deterministic comparison guarantee as the Go index.
func indexPython(operations []pythonOperation) (map[string]pythonOperation, error) {
	indexed := make(map[string]pythonOperation, len(operations))
	for _, operation := range operations {
		if _, exists := indexed[operation.ID]; exists || operation.ID == "" {
			return nil, fmt.Errorf("Python snapshot contains duplicate or empty operation id %q", operation.ID)
		}
		indexed[operation.ID] = operation
	}
	return indexed, nil
}

// accountOnly validates that the ledger's language-only list exactly equals the operations still unmatched.
func accountOnly[T any](language string, expected []string, indexed map[string]T, used map[string]bool) error {
	want := map[string]bool{}
	for _, id := range expected {
		if want[id] {
			return fmt.Errorf("%s-only ledger contains duplicate operation %q", language, id)
		}
		if _, ok := indexed[id]; !ok {
			return fmt.Errorf("%s-only ledger references missing operation %q", language, id)
		}
		want[id] = true
	}
	actual := []string{}
	for id := range indexed {
		if !used[id] {
			actual = append(actual, id)
		}
	}
	sort.Strings(actual)
	if !sameStrings(actual, expected) {
		return fmt.Errorf("unaccounted %s operations = %v, ledger = %v", language, actual, sortedCopy(expected))
	}
	return nil
}

// normalizedRoute ignores placeholder spelling only while retaining the HTTP verb, API version, and literal path casing.
func normalizedRoute(method string, path string) string {
	path = placeholderPattern.ReplaceAllString(path, "{}")
	return strings.ToUpper(strings.TrimSpace(method)) + " " + path
}

// operationDifferences returns stable category names suitable for the checked-in dispute ledger.
func operationDifferences(goOperation goOperation, pythonOperation pythonOperation) []string {
	differences := []string{}
	if !strings.EqualFold(goOperation.HTTPMethod, pythonOperation.HTTPMethod) {
		differences = append(differences, "verb")
	}
	goVersion, pythonVersion := apiVersion(goOperation.PathTemplate), apiVersion(pythonOperation.PathTemplate)
	if goVersion != pythonVersion {
		differences = append(differences, "api_version")
	}
	goWithoutVersion := apiVersionPattern.ReplaceAllString(goOperation.PathTemplate, "/api/v{}")
	pythonWithoutVersion := apiVersionPattern.ReplaceAllString(pythonOperation.PathTemplate, "/api/v{}")
	if placeholderPattern.ReplaceAllString(goWithoutVersion, "{}") != placeholderPattern.ReplaceAllString(pythonWithoutVersion, "{}") {
		differences = append(differences, "path")
	}
	if !sameStrings(placeholders(goOperation.PathTemplate), placeholders(pythonOperation.PathTemplate)) {
		differences = append(differences, "placeholder_casing")
	}
	goSegments := placeholderPattern.ReplaceAllString(goWithoutVersion, "{}")
	pythonSegments := placeholderPattern.ReplaceAllString(pythonWithoutVersion, "{}")
	if goSegments != pythonSegments && strings.EqualFold(goSegments, pythonSegments) {
		differences = append(differences, "segment_casing")
	}
	if goOperation.Mutation != pythonOperation.SideEffect {
		differences = append(differences, "side_effect")
	}
	sort.Strings(differences)
	return differences
}

// apiVersion extracts the version token from a documented API path.
func apiVersion(path string) string {
	return strings.ToLower(apiVersionPattern.FindString(path))
}

// placeholders returns path parameter names in order so casing drift remains visible.
func placeholders(path string) []string {
	matches := placeholderPattern.FindAllStringSubmatch(path, -1)
	values := make([]string, 0, len(matches))
	for _, match := range matches {
		values = append(values, match[1])
	}
	return values
}

// sameStrings compares two string sets deterministically without requiring ledger authors to maintain sort order.
func sameStrings(left []string, right []string) bool {
	left, right = sortedCopy(left), sortedCopy(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// sortedCopy avoids mutating decoded snapshot or ledger slices while producing stable diagnostics.
func sortedCopy(values []string) []string {
	copyOfValues := append([]string(nil), values...)
	sort.Strings(copyOfValues)
	return copyOfValues
}
