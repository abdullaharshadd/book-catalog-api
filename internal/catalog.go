// Package catalog is the root package for the Book Catalog API,
// a simple CRUD service for managing books.
//
// The original source file (app/__init__.py) contained only Python package
// metadata (version, author, email) and a package docstring. It held no
// business logic, routes, ORM queries, or framework glue.
//
// In idiomatic Go, package-level metadata such as version and authorship does
// not live in source files. Instead:
//
//   - The module path and dependencies belong in go.mod.
//   - The version is typically injected at build time via -ldflags
//     (e.g. go build -ldflags "-X internal.Version=1.0.0") or derived from a
//     git tag, rather than hardcoded.
//   - Authorship/attribution belongs in LICENSE, AUTHORS, or the repository
//     metadata, not in exported source constants.
//
// MIGRATION_NOTE: The values below are preserved for traceability. Prefer
// injecting Version at build time and moving Author/Email to an AUTHORS file
// or the module's LICENSE header. There are no HTTP routes to register in this
// file; the source contained none.
package catalog

// Build/version metadata for the Book Catalog API.
//
// Version is declared as a var (not a const) so it can be overridden at build
// time with linker flags:
//
//	go build -ldflags "-X 'github.com/your/module/internal.Version=1.0.0'" ./...
var (
	// Version is the semantic version of the Book Catalog API.
	Version = "1.0.0"

	// Author is the primary author of the Book Catalog API.
	Author = "Abdullah Arshad"

	// Email is the contact email for the author.
	Email = "abdullah.arshad.314@gmail.com"
)
