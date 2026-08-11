```go
package tests

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

// sourceFilePath returns the absolute path to the __init__.go file under test.
func sourceFilePath(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller must succeed")
	// The test file and the target file live in the same directory.
	return filepath.Join(filepath.Dir(currentFile), "__init__.go")
}

// parsedFile parses the target source file and returns the AST file node.
func parsedFile(t *testing.T) *ast.File {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, sourceFilePath(t), nil, parser.ParseComments)
	require.NoError(t, err, "target file must parse without errors")
	return f
}

// TestPackageFileExists verifies the target file is present on disk.
func TestPackageFileExists(t *testing.T) {
	tests := []struct {
		name     string
		wantFile string
	}{
		{
			name:     "init file exists in tests directory",
			wantFile: "__init__.go",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(filepath.Dir(sourceFilePath(t)), tc.wantFile)
			_, err := os.Stat(path)
			assert.NoError(t, err, "file %q must exist", tc.wantFile)
		})
	}
}

// TestPackageDeclaration validates that the file declares the correct package name,
// confirming the tests directory is recognized as a Go package.
func TestPackageDeclaration(t *testing.T) {
	tests := []struct {
		name        string
		wantPkg     string
		description string
	}{
		{
			name:        "package name is tests",
			wantPkg:     "tests",
			description: "the tests directory must be recognized as a Go package named 'tests'",
		},
	}

	f := parsedFile(t)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantPkg, f.Name.Name, tc.description)
		})
	}
}

// TestNoExecutableCode verifies the file contains no functions, types, variables,
// or constants — only the package declaration and doc comment.
func TestNoExecutableCode(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "no declarations in file",
			description: "the file must contain no executable code, functions, classes, or endpoints",
		},
	}

	f := parsedFile(t)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Empty(t, f.Decls, tc.description)
		})
	}
}

// TestNoImports verifies the file has no import statements, which would imply
// side effects from package initialisation.
func TestNoImports(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "no import declarations",
			description: "importing the package must produce no side effects beyond standard package initialisation",
		},
	}

	f := parsedFile(t)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Empty(t, f.Imports, tc.description)
		})
	}
}

// TestNoInitFunctions checks that no init() functions are defined, ensuring
// importing the package produces no side effects.
func TestNoInitFunctions(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "no init function defined",
			description: "the package must define no init() function to avoid side effects on import",
		},
	}

	f := parsedFile(t)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, decl := range f.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok {
					assert.NotEqual(t, "init", fn.Name.Name, tc.description)
				}
			}
		})
	}
}

// TestPackageDocComment verifies the package-level doc comment is present and
// documents the intended contents of the test suite.
func TestPackageDocComment(t *testing.T) {
	tests := []struct {
		name             string
		mustContain      []string
		description      string
	}{
		{
			name: "doc comment mentions unit tests for models",
			mustContain: []string{
				"unit tests",
				"model",
			},
			description: "doc comment must reference unit tests for the domain model",
		},
		{
			name: "doc comment mentions unit tests for schemas",
			mustContain: []string{
				"schemas",
			},
			description: "doc comment must reference unit tests for request/response schemas",
		},
		{
			name: "doc comment mentions integration tests for API endpoints",
			mustContain: []string{
				"integration tests",
				"endpoint",
			},
			description: "doc comment must reference integration tests for HTTP API endpoints",
		},
		{
			name: "doc comment mentions Book Catalog API",
			mustContain: []string{
				"Book Catalog",
			},
			description: "doc comment must identify the subject as the Book Catalog API",
		},
	}

	f := parsedFile(t)

	// Collect the full text of the package doc comment.
	var docText strings.Builder
	if f.Doc != nil {
		for _, c := range f.Doc.List {
			// Strip comment markers.
			line := strings.TrimPrefix(c.Text, "//")
			line = strings.TrimPrefix(line, "/*")
			line = strings.TrimSuffix(line, "*/")
			docText.WriteString(line)
			docText.WriteRune('\n')
		}
	}
	doc := strings.ToLower(docText.String())

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEmpty(t, doc, "package doc comment must not be empty")
			for _, needle := range tc.mustContain {
				assert.Contains(t, doc, strings.ToLower(needle),
					"%s — expected doc to contain %q", tc.description, needle)
			}
		})
	}
}

// TestPackageDocCommentNotEmpty is a baseline guard ensuring the doc comment
// exists at all, mirroring the invariant that the original docstring carried
// over documentation from the Python source.
func TestPackageDocCommentNotEmpty(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "doc comment is present",
			description: "the package-level docstring must be carried over from the Python source",
		},
	}

	f := parsedFile(t)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.NotNil(t, f.Doc, tc.description)
			assert.Greater(t, len(f.Doc.List), 0, tc.description)
		})
	}
}

// TestDirectoryIsRecognizedAsPackage verifies the directory satisfies Go's
// package recognition rules: every .go file in the directory must share the
// same package name.
func TestDirectoryIsRecognizedAsPackage(t *testing.T) {
	tests := []struct {
		name        string
		wantPkg     string
		description string
	}{
		{
			name:        "all go files share package name tests",
			wantPkg:     "tests",
			description: "the tests directory must be recognised as a single Go package",
		},
	}

	dir := filepath.Dir(sourceFilePath(t))

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := os.ReadDir(dir)
			require.NoError(t, err, "must be able to read tests directory")

			fset := token.NewFileSet()
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				if !strings.HasSuffix(entry.Name(), ".go") {
					continue
				}
				path := filepath.Join(dir, entry.Name())
				af, err := parser.ParseFile(fset, path, nil, 0)
				if err != nil {
					// Skip files that don't parse (e.g. build-tag-only files).
					continue
				}
				pkgName := af.Name.Name
				// Test files may use package "tests" or "tests_test"; both are valid.
				if strings.HasSuffix(pkgName, "_test") {
					pkgName = strings.TrimSuffix(pkgName, "_test")
				}
				assert.Equal(t, tc.wantPkg, pkgName,
					"file %q must belong to package %q: %s", entry.Name(), tc.wantPkg, tc.description)
			}
		})
	}
}

// TestNoGlobalVariables ensures no package-level variables are declared,
// which could introduce state and side effects on import.
func TestNoGlobalVariables(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "no var declarations at package level",
			description: "package must not declare global variables that could cause import side effects",
		},
	}

	f := parsedFile(t)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, decl := range f.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				assert.NotEqual(t, token.VAR, gen.Tok,
					tc.description)
			}
		})
	}
}

// TestNoConstants ensures no package-level constants are declared.
func TestNoConstants(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "no const declarations at package level",
			description: "package must not declare constants (the file is documentation-only)",
		},
	}

	f := parsedFile(t)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, decl := range f.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				assert.NotEqual(t, token.CONST, gen.Tok, tc.description)
			}
		})
	}
}

// TestNoTypeDeclarations ensures no type declarations are present.
func TestNoTypeDeclarations(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "no type declarations at package level",
			description: "package must not declare types (the file is documentation-only)",
		},
	}

	f := parsedFile(t)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, decl := range f.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				assert.NotEqual(t, token.TYPE, gen.Tok, tc.description)
			}
		})
	}
}

// TestFileIsValidGo ensures the file is syntactically valid Go source.
func TestFileIsValidGo(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "file parses without errors",
			description: "the migrated file must be valid Go source code",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			_, err := parser.ParseFile(fset, sourceFilePath(t), nil, parser.ParseComments)
			assert.NoError(t, err, tc.description)
		})
	}
}
```