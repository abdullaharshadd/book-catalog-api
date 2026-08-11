// Package tests contains the test suite for the Book Catalog API.
//
// It holds unit tests for the model and schema layers (internal/model.go and
// internal/schemas.go) plus integration tests for the HTTP API endpoints
// registered in internal/main.go.
//
// MIGRATION_NOTE: The Python source (tests/__init__.py) was an empty package
// marker whose only content was a module docstring. Go has no __init__.py
// concept: a directory becomes a package simply by having .go files that
// declare `package <name>`, and there is no separate initialisation file to
// migrate. The docstring has been preserved as this package-level doc comment.
//
// MIGRATION_NOTE: The Go convention is to place tests in the same package as
// the code under test (internal_test.go files with `package internal` or
// `package internal_test`) rather than in a dedicated tests/ subdirectory.
// The individual test files (formerly tests/test_*.py) should be migrated as
// *_test.go files. This file exists only to carry the suite-level
// documentation; it declares no runtime code because the original had none.
package tests
