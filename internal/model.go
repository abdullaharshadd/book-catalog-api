package internal

import "fmt"

// Book represents a book in the catalog.
//
// This replaces the SQLAlchemy declarative model from app/models.py. The
// underlying "books" table is created by internal/database.go at startup;
// this file defines the Go-side domain type and its field mapping.
//
// Field declaration order (Title, Author, PublishedYear, Summary) mirrors the
// source model so validation error ordering stays consistent with the original
// Pydantic/SQLAlchemy behaviour.
//
// The composite uniqueness of (title, author) — the source's
// UniqueConstraint('title', 'author', name='unique_title_author') — is enforced
// at the database level via a UNIQUE constraint in the schema DDL, not in this
// struct.
type Book struct {
	// ID is the primary key, auto-incrementing (SERIAL/IDENTITY in PostgreSQL).
	ID int64 `json:"id" db:"id"`

	// Title is the book title (required).
	Title string `json:"title" db:"title"`

	// Author is the book author (required).
	Author string `json:"author" db:"author"`

	// PublishedYear is the year the book was published (required).
	PublishedYear int `json:"published_year" db:"published_year"`

	// Summary is an optional description of the book. It maps to a nullable
	// TEXT column; an empty string represents the absence of a summary.
	Summary string `json:"summary" db:"summary"`
}

// String returns a human-readable representation of the book, replacing the
// source model's __str__ dunder method.
func (b Book) String() string {
	return fmt.Sprintf("%s by %s (%d)", b.Title, b.Author, b.PublishedYear)
}

// GoString returns a debug representation of the book, replacing the source
// model's __repr__ dunder method. It is used by the %#v verb in fmt.
func (b Book) GoString() string {
	return fmt.Sprintf("<Book(id=%d, title='%s', author='%s', year=%d)>",
		b.ID, b.Title, b.Author, b.PublishedYear)
}
