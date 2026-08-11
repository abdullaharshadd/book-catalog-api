// Package internal contains the core application logic for the Book Catalog API,
// a simple CRUD service for managing books.
//
// This file replaces the Python package initializer (app/__init__.py), which
// carried package-level metadata via the dunder convention (__version__,
// __author__, __email__). In Go there is no runtime package-init metadata
// convention, so that information is exposed here as exported constants that
// can be referenced by handlers (e.g. a /version endpoint) or build tooling.
package internal

const (
	// Version is the semantic version of the Book Catalog API.
	Version = "1.0.0"

	// Author is the primary author of the Book Catalog API.
	Author = "Abdullah Arshad"

	// Email is the contact email for the Book Catalog API author.
	Email = "abdullah.arshad.314@gmail.com"
)
