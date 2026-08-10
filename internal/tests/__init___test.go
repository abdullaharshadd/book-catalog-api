```go
package tests_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sourceFile returns the absolute path to the __init__.go file under test.
func sourceFile(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller must succeed")
	// thisFile is the test file; __init__.go lives in the same directory
	dir := filepath.Dir(thisFile)
	return filepath.Join(dir, "__init__.go")
}

// parsedFile returns the parsed AST file and file-set for __init__.go.
func parsedFile(t *testing.T) (*ast.File, *token.FileSet) {
	t.Helper()
	path := sourceFile(t)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	require.NoError(t, err, "parsing __init__.go must not error")
	return f, fset
}

// ---- tests ---------------------------------------------------------------

// TestPackageExists validates that the __init__.go file is present so that Go
// recognises the tests directory as a package (mirrors the Python __init__.py
// package-marker invariant).
func TestPackageExists(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{
			name: "init file present in tests directory",
			path: sourceFile(t),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := os.Stat(tc.path)
			assert.NoError(t, err, "__init__.go must exist so the tests directory is a Go package")
		})
	}
}

// TestPackageClause validates that the file declares the expected package name.
func TestPackageClause(t *testing.T) {
	cases := []struct {
		name            string
		wantPackageName string
	}{
		{
			name:            "package name is 'tests'",
			wantPackageName: "tests",
		},
	}

	f, _ := parsedFile(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantPackageName, f.Name.Name,
				"package clause must declare package tests")
		})
	}
}

// TestNoExecutableCodeOrDefinitions validates that __init__.go contains no
// top-level declarations (vars, consts, types, funcs), mirroring the Python
// invariant that the module has no executable code or definitions.
func TestNoExecutableCodeOrDefinitions(t *testing.T) {
	cases := []struct {
		name          string
		wantDeclCount int
	}{
		{
			name:          "file has no top-level declarations",
			wantDeclCount: 0,
		},
	}

	f, _ := parsedFile(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Len(t, f.Decls, tc.wantDeclCount,
				"__init__.go must contain no top-level declarations")
		})
	}
}

// TestNoImports validates that importing the package produces no side effects —
// concretely there must be no import statements (which would imply init()
// side-effects or dependency coupling).
func TestNoImports(t *testing.T) {
	cases := []struct {
		name            string
		wantImportCount int
	}{
		{
			name:            "file has no import declarations",
			wantImportCount: 0,
		},
	}

	f, _ := parsedFile(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Len(t, f.Imports, tc.wantImportCount,
				"__init__.go must have no imports so importing the package has no side effects")
		})
	}
}

// TestPackageDocstringPresent validates that a package-level doc comment exists
// (mirrors the Python invariant that the module carries a module docstring).
func TestPackageDocstringPresent(t *testing.T) {
	cases := []struct {
		name string
	}{
		{name: "package-level doc comment must exist"},
	}

	f, _ := parsedFile(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotNil(t, f.Doc,
				"__init__.go must have a package-level doc comment acting as a module docstring: %s", tc.name)
		})
	}
}

// TestDocstringDescribesBookCatalogAPI validates that the package doc comment
// mentions the Book Catalog API, mirroring the Python docstring content
// invariant.
func TestDocstringDescribesBookCatalogAPI(t *testing.T) {
	cases := []struct {
		name           string
		requiredPhrase string
	}{
		{
			name:           "docstring mentions Book Catalog API",
			requiredPhrase: "Book Catalog API",
		},
		{
			name:           "docstring mentions test suite",
			requiredPhrase: "test",
		},
	}

	f, _ := parsedFile(t)
	require.NotNil(t, f.Doc, "package doc comment must exist before checking content")

	docText := f.Doc.Text()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t,
				strings.Contains(strings.ToLower(docText), strings.ToLower(tc.requiredPhrase)),
				"package doc must contain %q; got: %q", tc.requiredPhrase, docText)
		})
	}
}

// TestNoInitFunction validates that no init() function is defined, which would
// introduce side-effects on import.
func TestNoInitFunction(t *testing.T) {
	cases := []struct {
		name string
	}{
		{name: "no init() function defined"},
	}

	f, _ := parsedFile(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, decl := range f.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok {
					assert.NotEqual(t, "init", fn.Name.Name,
						"%s: found unexpected init() function which would cause side-effects on import", tc.name)
				}
			}
		})
	}
}

// TestFileContainsOnlyPackageAndComments validates that the raw source of
// __init__.go contains a package declaration and comments but nothing else of
// substance, providing a higher-level view complementary to the AST checks.
func TestFileContainsOnlyPackageAndComments(t *testing.T) {
	cases := []struct {
		name              string
		forbiddenKeywords []string
	}{
		{
			name:              "no variable declarations",
			forbiddenKeywords: []string{"\nvar "},
		},
		{
			name:              "no constant declarations",
			forbiddenKeywords: []string{"\nconst "},
		},
		{
			name:              "no type declarations",
			forbiddenKeywords: []string{"\ntype "},
		},
		{
			name:              "no function declarations",
			forbiddenKeywords: []string{"\nfunc "},
		},
		{
			name:              "no import declarations",
			forbiddenKeywords: []string{"\nimport "},
		},
	}

	raw, err := os.ReadFile(sourceFile(t))
	require.NoError(t, err, "must be able to read __init__.go")
	content := string(raw)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, kw := range tc.forbiddenKeywords {
				assert.False(t,
					strings.Contains(content, kw),
					"__init__.go must not contain %q — file should hold only package clause and doc comment", kw)
			}
		})
	}
}

// TestMigrationNotePresent validates that the migration note comment is present,
// documenting the Go-idiomatic rationale for this file's existence.
func TestMigrationNotePresent(t *testing.T) {
	cases := []struct {
		name           string
		requiredPhrase string
	}{
		{
			name:           "migration note references Python source",
			requiredPhrase: "MIGRATION_NOTE",
		},
	}

	raw, err := os.ReadFile(sourceFile(t))
	require.NoError(t, err)
	content := string(raw)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t,
				strings.Contains(content, tc.requiredPhrase),
				"__init__.go must contain a %q comment explaining the migration rationale", tc.requiredPhrase)
		})
	}
}
```