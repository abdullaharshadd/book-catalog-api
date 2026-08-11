// Package tests contains the test suite for the Book Catalog API.
//
// It holds unit tests for the domain model (internal/model.go) and request /
// response schemas (internal/schemas.go), plus integration tests for the HTTP
// API endpoints registered in internal/main.go.
//
// MIGRATION_NOTE: The source file tests/__init__.py is a Python package marker
// (__init__.py) whose only real content is a module docstring. Go has no direct
// equivalent — packages are defined implicitly by the `package` clause on each
// file in a directory, and there is no per-directory initialiser file required.
//
// This file therefore exists purely to declare the test package and carry over
// the documentation from the original docstring. The actual tests live in
// sibling *_test.go files within this package (e.g. model_test.go,
// schemas_test.go, main_test.go), following Go's table-driven testing
// convention. Those files import the migrated code from the parent module's
// internal package.
package tests
