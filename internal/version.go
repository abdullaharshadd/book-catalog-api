// Package internal contains the core application code for the Book Catalog API,
// a simple CRUD service for managing books.
//
// MIGRATION_NOTE: The original app/__init__.py defined Python package-level
// dunder metadata (__version__, __author__, __email__). Go has no direct
// equivalent of package dunder attributes; the idiomatic replacement is a set
// of exported package-level constants that callers can reference (e.g. for a
// version endpoint or startup log line).
//
// MIGRATION_NOTE: The Python module lived at app/__init__.py and the requested
// target path is internal/__init__.go, but Go filenames may not begin with an
// underscore-only base name that the toolchain treats specially, and Go has no
// per-directory init file convention like Python's __init__.py. This file is
// therefore a regular Go source file declaring the `internal` package metadata.
// Consider renaming it to internal/version.go for clarity.
package internal

// Application metadata for the Book Catalog API. These mirror the package
// dunder metadata (__version__, __author__, __email__) from the original
// Python package initializer.
const (
	// Version is the current release version of the application.
	Version = "1.0.0"

	// Author is the primary author of the application.
	Author = "Abdullah Arshad"

	// Email is the contact email address for the author.
	Email = "abdullah.arshad.314@gmail.com"
)
