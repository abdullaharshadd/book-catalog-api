package tests

// This file replaces the pytest suite tests/test_models.py, which validated
// the SQLAlchemy Book model: field creation, the optional summary field, the
// String / GoString representations, and the unique (title, author) composite
// constraint. It bootstrapped an in-memory SQLite schema via
// Base.metadata.create_all for per-test isolation.
//
// MIGRATION_NOTE: In Go, tests live in *_test.go files run by `go test`. The
// pytest class grouping (TestBookModel) and the function-scoped db_session
// fixture map onto per-test helper calls. The actual runnable tests live in
// the sibling file test_models_test.go so `go test` picks them up; this file
// carries only shared documentation for the model test suite.
//
// MIGRATION_NOTE: The source used an in-memory SQLite engine
// (sqlite:///:memory:). The target dialect for this project is PostgreSQL, so
// the model tests obtain an isolated database via the NewTestDB helper defined
// in internal/conftest.go rather than an in-process SQLite engine. Schema
// creation that the source performed with Base.metadata.create_all is handled
// there against the real Book table.
//
// MIGRATION_NOTE: The Book model, its String / GoString methods, and the
// persistence layer already live in the internal package (internal/model.go).
// There is nothing new to declare here beyond this documentation; adding a
// duplicate Book type or fixture constructor would collide with the already
// migrated symbols. The behavioural assertions are expressed as table-driven
// tests in test_models_test.go.
