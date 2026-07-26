// Package internal contains the Book Catalog API — a simple CRUD service
// for managing books.
//
// This file corresponds to the Python package's __init__.py, which declared
// package-level metadata (version, author, email) as dunder variables. In Go
// there is no direct equivalent to Python's package __init__ module or dunder
// metadata; that information is conventionally expressed as exported package
// constants and/or embedded in build metadata via ldflags.
//
// MIGRATION_NOTE: The target file path was internal/__init__.go, but Go does
// not use __init__ files. A single doc.go-style file per package holds the
// package doc comment. The metadata below is exposed as exported constants so
// callers can reference it programmatically (e.g. for /version endpoints).
package internal

// Package metadata mirrored from the original Python package's dunder
// variables (__version__, __author__, __email__).
const (
	// Version is the semantic version of the Book Catalog API.
	Version = "1.0.0"

	// Author is the primary author of the Book Catalog API.
	Author = "Abdullah Arshad"

	// Email is the contact email for the Book Catalog API author.
	Email = "abdullah.arshad.314@gmail.com"
)
