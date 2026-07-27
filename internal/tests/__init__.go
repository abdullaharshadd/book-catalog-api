// Package tests contains the test suite for the Book Catalog API.
//
// It holds unit tests for models and schemas, plus integration tests for the
// API endpoints.
//
// MIGRATION_NOTE: The Python source was an empty tests/__init__.py package
// marker whose only content was a module docstring. Go has no equivalent of
// __init__.py — a directory becomes a package simply by containing .go files
// that share a package clause, and there is no package-initialization file to
// migrate. This file exists solely to carry the package's godoc and to declare
// the package so the directory is a valid Go package even before the actual
// _test.go files are added. It contains no executable code, mirroring the
// no-op nature of the original.
//
// Note that in idiomatic Go, test files (e.g. model_test.go, schemas_test.go)
// normally live alongside the code they test in package internal (or
// internal_test), rather than in a separate tests/ directory. If the migrated
// test files follow that convention, this package may ultimately be
// unnecessary and can be removed during manual review.
package tests
