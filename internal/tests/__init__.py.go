// Package tests contains the test suite for the Book Catalog API.
//
// This file was migrated from tests/__init__.py, a Python package marker
// that contained only a module-level docstring. Go has no direct equivalent
// of __init__.py: packages are defined implicitly by the package clause and
// the directory that contains their files. There is therefore no executable
// code to migrate here.
//
// The original docstring described the suite as follows:
//
//	"Test suite for Book Catalog API. Contains unit tests for models and
//	 schemas, plus integration tests for API endpoints."
//
// In idiomatic Go these tests live alongside (or near) the code under test
// as *_test.go files, using the standard testing package and table-driven
// test functions. Unit tests for the migrated models and schemas belong in
// internal/models_test.go and internal/schemas_test.go, while integration
// tests for the HTTP endpoints belong next to the server implementation.
//
// MIGRATION_NOTE: tests/__init__.py carried no logic. This file exists only
// to preserve the package's documentation. It may be deleted once the actual
// *_test.go files are written; nothing imports it.
package tests
