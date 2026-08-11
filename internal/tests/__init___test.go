```go
package tests_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sourceFile returns the absolute path to the target source file relative to
// this test file's location.
func sourceFilePath() string {
	_, currentFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(currentFile)
	return filepath.Join(dir, "__init__.go")
}

// parsedFile returns the parsed AST and file-set for the target source file.
func parsedFile(t *testing.T) (*ast.File, *token.FileSet) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, sourceFilePath(), nil, parser.ParseComments)
	require.NoError(t, err, "source file must be parseable")
	return f, fset
}

// TestPackageName validates that the file declares the expected package name.
func TestPackageName(t *testing.T) {
	cases := []struct {
		name     string
		wantPkg  string
	}{
		{
			name:    "package is named tests",
			wantPkg: "tests",
		},
	}

	f, _ := parsedFile(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantPkg, f.Name.Name,
				"package declaration must be %q", tc.wantPkg)
		})
	}
}

// TestPackageDocComment validates that the package-level doc comment exists and
// contains the expected description of the Book Catalog API test suite.
func TestPackageDocComment(t *testing.T) {
	cases := []struct {
		name        string
		mustContain string
	}{
		{
			name:        "doc mentions Book Catalog API",
			mustContain: "Book Catalog API",
		},
		{
			name:        "doc mentions test suite",
			mustContain: "test suite",
		},
		{
			name:        "doc mentions unit tests",
			mustContain: "unit tests",
		},
		{
			name:        "doc mentions integration tests",
			mustContain: "integration tests",
		},
		{
			name:        "doc mentions HTTP API endpoints",
			mustContain: "HTTP API endpoints",
		},
		{
			name:        "migration note explains no __init__.py concept",
			mustContain: "__init__.py",
		},
		{
			name:        "migration note mentions docstring preservation",
			mustContain: "docstring",
		},
	}

	f, _ := parsedFile(t)

	// Collect all package-level doc text.
	var docText strings.Builder
	if f.Doc != nil {
		for _, c := range f.Doc.List {
			docText.WriteString(c.Text)
			docText.WriteRune('\n')
		}
	}
	fullDoc := docText.String()

	require.NotEmpty(t, fullDoc, "package must have a doc comment")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Contains(t, fullDoc, tc.mustContain,
				"package doc comment must contain %q", tc.mustContain)
		})
	}
}

// TestNoPublicDeclarations validates that the file exposes no public
// functions, types, variables, or constants — matching the original empty
// __init__.py behaviour.
func TestNoPublicDeclarations(t *testing.T) {
	cases := []struct {
		name string
		kind string
	}{
		{name: "no exported functions", kind: "func"},
		{name: "no exported types", kind: "type"},
		{name: "no exported variables", kind: "var"},
		{name: "no exported constants", kind: "const"},
	}

	f, _ := parsedFile(t)

	// Gather all exported top-level names by category.
	exported := map[string][]string{
		"func":  {},
		"type":  {},
		"var":   {},
		"const": {},
	}

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.IsExported() {
				exported["func"] = append(exported["func"], d.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.IsExported() {
						exported["type"] = append(exported["type"], s.Name.Name)
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name.IsExported() {
							switch d.Tok.String() {
							case "var":
								exported["var"] = append(exported["var"], name.Name)
							case "const":
								exported["const"] = append(exported["const"], name.Name)
							}
						}
					}
				}
			}
		}
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Empty(t, exported[tc.kind],
				"file must not export any %s declarations, found: %v",
				tc.kind, exported[tc.kind])
		})
	}
}

// TestImportSucceedsWithoutSideEffects validates (at the Go level) that
// importing the tests package does not register any init() functions with
// observable side effects. We check this by asserting there are no init
// function declarations in the source file.
func TestImportSucceedsWithoutSideEffects(t *testing.T) {
	cases := []struct {
		name string
	}{
		{name: "no init() function declared"},
	}

	f, _ := parsedFile(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, decl := range f.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok {
					assert.NotEqual(t, "init", fn.Name.Name,
						"package must not declare an init() function to avoid side effects")
				}
			}
		})
	}
}

// TestNoRuntimeCode validates that there are no function bodies or variable
// initialisers — the file should contain documentation only.
func TestNoRuntimeCode(t *testing.T) {
	cases := []struct {
		name string
	}{
		{name: "file contains no executable declarations"},
	}

	f, _ := parsedFile(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Empty(t, f.Decls,
				"%s: __init__.go must have no declarations (it is documentation-only)",
				tc.name)
		})
	}
}

// TestDirectoryRecognizedAsPackage validates that the package declaration
// itself makes the directory a valid Go package — analogous to the Python
// invariant that __init__.py marks a directory as a package.
func TestDirectoryRecognizedAsPackage(t *testing.T) {
	cases := []struct {
		name    string
		wantPkg string
	}{
		{
			name:    "directory is a valid Go package",
			wantPkg: "tests",
		},
	}

	f, _ := parsedFile(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotNil(t, f.Name, "file must have a package clause")
			assert.Equal(t, tc.wantPkg, f.Name.Name,
				"package clause must be %q so the directory is a recognized Go package",
				tc.wantPkg)
		})
	}
}

// TestFileExists validates that the source file is actually present on disk.
func TestFileExists(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{
			name: "__init__.go exists on disk",
			path: sourceFilePath(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, tc.path, nil, parser.ParseComments)
			assert.NoError(t, err, "file at %q must exist and be parseable", tc.path)
			assert.NotNil(t, f, "parsed file must not be nil")
		})
	}
}
```