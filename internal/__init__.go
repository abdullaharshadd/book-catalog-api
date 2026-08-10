// Package internal contains the core implementation of the Book Catalog API,
// a simple CRUD service for managing books.
//
// MIGRATION_NOTE: The original Python package docstring referenced FastAPI,
// SQLAlchemy, and Pydantic. In this Go migration the equivalent stack is
// net/http with go-chi routing, sqlx over lib/pq (PostgreSQL), and plain
// structs for request/response payloads. The package-level dunder metadata
// (__version__, __author__, __email__) is expressed here as exported
// constants so it can be referenced elsewhere (e.g. in a /version endpoint
// or build info).
package internal

const (
	// Version is the semantic version of the Book Catalog API.
	Version = "1.0.0"

	// Author is the primary author of the Book Catalog API.
	Author = "Abdullah Arshad"

	// Email is the contact email for the Book Catalog API author.
	Email = "abdullah.arshad.314@gmail.com"
)
