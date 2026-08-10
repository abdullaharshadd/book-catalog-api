// Package tests contains the test suite for the Book Catalog API.
//
// It provides unit tests for the models and schemas defined in the internal
// package, plus integration tests for the HTTP API endpoints.
//
// MIGRATION_NOTE: The Python source was a tests/__init__.py file whose only
// purpose was to mark the tests directory as a package and carry a module
// docstring. Go has no equivalent package-marker file requirement — a package
// is defined by the package clause in its .go files. This file therefore
// exists solely to hold the package-level documentation; the actual tests
// live in *_test.go files in this same package. Idiomatically, Go test files
// often live alongside the code under test (in package internal), but a
// dedicated tests package is retained here to mirror the source layout for
// black-box / integration tests that import internal via its public API.
package tests
