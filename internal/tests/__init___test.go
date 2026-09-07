// Package tests contains tests for the tests package itself.
// Since the migrated file is documentation-only and declares no executable
// logic, these tests validate the package-level invariants described in the
// behavioral specs.
package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sourceFile is the path to the file under test, relative to the test file.
const sourceFile = "__init__.go"

// TestPackageFileExists validates that the __init__.go file exists in the
// tests directory (analogous to __init__.py existing in a Python package).
func TestPackageFileExists(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "init file exists",
			filename: sourceFile,
			wantErr:  false,
		},
		{
			name:     "non-existent file does not exist",
			filename: "nonexistent_file.go",
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := os.Stat(tc.filename)
			if tc.wantErr {
				assert.True(t, os.IsNotExist(err), "expected file to not exist: %s", tc.filename)
			} else {
				assert.NoError(t, err, "expected file to exist: %s", tc.filename)
			}
		})
	}
}

// TestPackageDeclaration validates that the file declares the correct package name.
func TestPackageDeclaration(t *testing.T) {
	tests := []struct {
		name            string
		filename        string
		expectedPackage string
	}{
		{
			name:            "package is named tests",
			filename:        sourceFile,
			expectedPackage: "tests",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, tc.filename, nil, parser.PackageClauseOnly)
			require.NoError(t, err, "failed to parse file: %s", tc.filename)
			assert.Equal(t, tc.expectedPackage, node.Name.Name,
				"package name should be %q", tc.expectedPackage)
		})
	}
}

// TestFileContainsNoExecutableLogic validates that the __init__.go file
// declares no functions, types, variables, or constants — it is documentation-only.
func TestFileContainsNoExecutableLogic(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "init file has no declarations",
			filename: sourceFile,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, tc.filename, nil, parser.AllErrors)
			require.NoError(t, err, "failed to parse file: %s", tc.filename)

			assert.Empty(t, node.Decls,
				"file should contain no declarations (functions, types, vars, consts)")
			assert.Empty(t, node.Imports,
				"file should contain no import statements")
		})
	}
}

// TestFileContainsDocstring validates that the file contains a meaningful
// package-level doc comment (the Go equivalent of a Python module docstring).
func TestFileContainsDocstring(t *testing.T) {
	tests := []struct {
		name            string
		filename        string
		expectedSubstrs []string
	}{
		{
			name:     "package doc comment contains migration note",
			filename: sourceFile,
			expectedSubstrs: []string{
				"Package tests",
				"Book Catalog API",
				"MIGRATION_NOTE",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, tc.filename, nil, parser.ParseComments)
			require.NoError(t, err, "failed to parse file: %s", tc.filename)

			require.NotNil(t, node.Doc,
				"file should have a package-level doc comment")

			docText := node.Doc.Text()
			for _, substr := range tc.expectedSubstrs {
				assert.Contains(t, docText, substr,
					"doc comment should contain %q", substr)
			}
		})
	}
}

// TestNoSideEffectsOnImport validates that importing the package produces
// no side effects. Since the file has no init() functions, vars with
// initializers, or any declarations, this is verified by checking the AST.
func TestNoSideEffectsOnImport(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "no init functions defined",
			filename: sourceFile,
		},
		{
			name:     "no global variable declarations with side effects",
			filename: sourceFile,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, tc.filename, nil, parser.AllErrors)
			require.NoError(t, err, "failed to parse file: %s", tc.filename)

			// Check for init() functions specifically.
			hasInitFunc := false
			hasVarDecl := false
			for _, decl := range node.Decls {
				switch d := decl.(type) {
				case *ast.FuncDecl:
					if d.Name.Name == "init" {
						hasInitFunc = true
					}
				case *ast.GenDecl:
					if d.Tok == token.VAR {
						hasVarDecl = true
					}
				}
			}

			assert.False(t, hasInitFunc,
				"file should not define an init() function (no side effects on import)")
			assert.False(t, hasVarDecl,
				"file should not declare package-level variables (no side effects on import)")
		})
	}
}

// TestFileReadability validates that the file is readable and well-formed Go source.
func TestFileReadability(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "file is valid Go source",
			filename: sourceFile,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content, err := os.ReadFile(tc.filename)
			require.NoError(t, err, "should be able to read file: %s", tc.filename)
			assert.NotEmpty(t, content, "file should not be empty")

			fset := token.NewFileSet()
			_, err = parser.ParseFile(fset, tc.filename, content, parser.AllErrors)
			assert.NoError(t, err, "file should parse without errors")
		})
	}
}

// TestPackageNameConsistency validates that the package name in the file
// matches the directory name convention used in Go projects.
func TestPackageNameConsistency(t *testing.T) {
	tests := []struct {
		name            string
		filename        string
		expectedPkgName string
	}{
		{
			name:            "package name matches directory convention",
			filename:        sourceFile,
			expectedPkgName: "tests",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Verify the directory name.
			absPath, err := filepath.Abs(tc.filename)
			require.NoError(t, err)

			dirName := filepath.Base(filepath.Dir(absPath))
			assert.Equal(t, tc.expectedPkgName, dirName,
				"directory name should match expected package name")

			// Verify the declared package name.
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, tc.filename, nil, parser.PackageClauseOnly)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedPkgName, node.Name.Name,
				"declared package name should be %q", tc.expectedPkgName)
		})
	}
}

// TestDocCommentMentionsPythonMigration validates that the migration note
// references the original Python __init__.py, preserving traceability.
func TestDocCommentMentionsPythonMigration(t *testing.T) {
	tests := []struct {
		name             string
		filename         string
		migrationKeyword string
	}{
		{
			name:             "doc comment references original Python file",
			filename:         sourceFile,
			migrationKeyword: "__init__.py",
		},
		{
			name:             "doc comment references Python package concept",
			filename:         sourceFile,
			migrationKeyword: "Python",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content, err := os.ReadFile(tc.filename)
			require.NoError(t, err)

			assert.True(t,
				strings.Contains(string(content), tc.migrationKeyword),
				"file content should mention %q for migration traceability", tc.migrationKeyword)
		})
	}
}

// TestNoFunctionsExported validates there are no exported (or unexported)
// functions, since the file is documentation-only.
func TestNoFunctionsExported(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "no exported functions",
			filename: sourceFile,
		},
		{
			name:     "no unexported functions",
			filename: sourceFile,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, tc.filename, nil, parser.AllErrors)
			require.NoError(t, err)

			funcCount := 0
			for _, decl := range node.Decls {
				if _, ok := decl.(*ast.FuncDecl); ok {
					funcCount++
				}
			}

			assert.Equal(t, 0, funcCount,
				"file should define zero functions, got %d", funcCount)
		})
	}
}

// TestNoTypeDeclarations validates there are no type declarations in the file.
func TestNoTypeDeclarations(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "no struct types declared",
			filename: sourceFile,
		},
		{
			name:     "no interface types declared",
			filename: sourceFile,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, tc.filename, nil, parser.AllErrors)
			require.NoError(t, err)

			typeCount := 0
			for _, decl := range node.Decls {
				if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
					typeCount++
				}
			}

			assert.Equal(t, 0, typeCount,
				"file should define zero types, got %d", typeCount)
		})
	}
}

// TestNoConstDeclarations validates there are no constant declarations.
func TestNoConstDeclarations(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "no constants declared",
			filename: sourceFile,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, tc.filename, nil, parser.AllErrors)
			require.NoError(t, err)

			constCount := 0
			for _, decl := range node.Decls {
				if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.CONST {
					constCount++
				}
			}

			assert.Equal(t, 0, constCount,
				"file should define zero constants, got %d", constCount)
		})
	}
}