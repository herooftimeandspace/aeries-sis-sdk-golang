package aeries

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDocGoFilesExist verifies that every handwritten package has a package-level orientation file.
func TestDocGoFilesExist(t *testing.T) {
	root := projectRoot(t)
	for _, relative := range []string{"doc.go", "contract/doc.go", "internal/contract/doc.go", "cmd/contractsync/doc.go"} {
		if _, err := os.Stat(filepath.Join(root, relative)); err != nil {
			t.Fatalf("missing %s: %v", relative, err)
		}
	}
}

// TestEveryFunctionHasDocComment verifies the plan requirement that every handwritten function explains its intent.
func TestEveryFunctionHasDocComment(t *testing.T) {
	root := projectRoot(t)
	fileSet := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if base == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, parseErr := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
		if parseErr != nil {
			return parseErr
		}
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if funcDecl.Doc == nil || strings.TrimSpace(funcDecl.Doc.Text()) == "" {
				t.Fatalf("function %s in %s is missing a doc comment", funcDecl.Name.Name, path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo: %v", err)
	}
}

// TestDocsIndexMirrorsReadme verifies the README-to-docs home page transformation required by the Pages site.
func TestDocsIndexMirrorsReadme(t *testing.T) {
	root := projectRoot(t)
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	index, err := os.ReadFile(filepath.Join(root, "docs", "index.md"))
	if err != nil {
		t.Fatalf("read docs/index.md: %v", err)
	}
	want := strings.ReplaceAll(string(readme), "](docs/", "](")
	if string(index) != want {
		t.Fatal("docs/index.md is not the README mirror")
	}
}

// TestDocsReferencePagesExist verifies that the docs site has the expected API reference pages checked in.
func TestDocsReferencePagesExist(t *testing.T) {
	root := projectRoot(t)
	for _, relative := range []string{
		"docs/reference/index.md",
		"docs/reference/aeries.md",
		"docs/reference/contract.md",
		"docs/reference/internal.md",
		".github/workflows/pages.yml",
		"mkdocs.yml",
		".env.example",
	} {
		if _, err := os.Stat(filepath.Join(root, relative)); err != nil {
			t.Fatalf("missing %s: %v", relative, err)
		}
	}
}

// TestReadmeContainsLLMDisclaimer preserves the README disclaimer expected by the wider workspace conventions.
func TestReadmeContainsLLMDisclaimer(t *testing.T) {
	readme, err := os.ReadFile(filepath.Join(projectRoot(t), "README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	text := string(readme)
	if !strings.Contains(text, "LLM Usage Disclaimer") || !strings.Contains(text, "LLM-driven project") {
		t.Fatal("README.md must keep the LLM usage disclaimer")
	}
}
