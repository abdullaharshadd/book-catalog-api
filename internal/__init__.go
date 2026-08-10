// Package internal contains the Book Catalog API application code.
//
// A simple CRUD service for managing books, migrated from a Python
// (FastAPI/SQLAlchemy/Pydantic) codebase to idiomatic Go using chi for
// routing, sqlx for database access, and PostgreSQL as the backing store.
//
// This file corresponds to the original app/__init__.py, which only defined
// package-level metadata (version, author, email) via Python dunder
// variables. In Go, these are expressed as exported package constants.
package internal

const (
	// Version is the semantic version of the Book Catalog API.
	Version = "1.0.0"

	// Author is the name of the application author.
	Author = "Abdullah Arshad"

	// Email is the contact email of the application author.
	Email = "abdullah.arshad.314@gmail.com"
)
