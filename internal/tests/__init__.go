// Package tests contains the test suite for the Book Catalog API.
//
// It holds unit tests for the models and schemas (see internal/model.go and
// internal/schemas.go) plus integration tests for the HTTP API endpoints
// exposed by BookServer (see internal/main.go).
//
// MIGRATION_NOTE: The source file tests/__init__.py was a Python package
// initializer whose only content was a module-level docstring marking the
// directory as a package. Go has no equivalent "package marker" file: a
// directory becomes a package simply by containing .go files with a matching
// `package` clause. This file therefore carries only the package declaration
// and the doc comment; the actual test functions live in *_test.go files
// within this same package (e.g. model_test.go, schemas_test.go,
// main_test.go). No business logic existed in the source to preserve.
package tests
