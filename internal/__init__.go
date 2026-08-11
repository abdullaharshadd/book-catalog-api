// Package internal contains the Book Catalog API — a simple CRUD service
// for managing books.
//
// This file is the Go analog of the source project's app/__init__.py, which
// carried only package-level metadata (version, author, email). In Go there
// is no dunder-metadata convention, so these values are exposed as exported
// constants that other packages (e.g. a /version endpoint or build-info log
// line) can reference.
package internal

// Package metadata, migrated from app/__init__.py's __version__, __author__,
// and __email__ dunder attributes.
const (
	// Version is the current release version of the Book Catalog API.
	Version = "1.0.0"

	// Author is the primary author of the Book Catalog API.
	Author = "Abdullah Arshad"

	// Email is the contact email for the Book Catalog API author.
	Email = "abdullah.arshad.314@gmail.com"
)
