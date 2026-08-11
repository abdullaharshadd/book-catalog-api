package internal

import "fmt"

// Book represents a book in the catalog.
//
// This is the Go analog of the SQLAlchemy Book model in app/models.py. The
// source model declared:
//   - id:             primary key, auto-incrementing integer
//   - title:          String(255), NOT NULL, indexed
//   - author:         String(255), NOT NULL, indexed
//   - published_year: Integer, NOT NULL, indexed
//   - summary:        Text, nullable
// plus a table-level UniqueConstraint on (title, author).
//
// The actual CREATE TABLE / index / unique-constraint DDL that mirrors these
// fields lives in internal/database.go (schema creation at boot); this file
// defines only the in-memory shape and its formatting helpers.
//
// MIGRATION_NOTE: published_year is NOT NULL in the source. A JSON body that
// omits it (or sends null) leaves PublishedYear as its zero value (0) here; the
// repository/handler layer is responsible for rejecting missing years, and the
// DB's NOT NULL constraint is the final backstop — a null reaching the INSERT
// surfaces as a PostgreSQL not-null violation which the error-classification
// layer maps to HTTP 400.
type Book struct {
	// ID is the auto-incrementing primary key.
	ID int64 `json:"id" db:"id"`

	// Title is the book title (required).
	Title string `json:"title" db:"title"`

	// Author is the book author (required).
	Author string `json:"author" db:"author"`

	// PublishedYear is the year the book was published (required, NOT NULL).
	PublishedYear int `json:"published_year" db:"published_year"`

	// Summary is an optional description of the book. It maps to a nullable
	// TEXT column; the empty string represents SQL NULL / absent summary.
	Summary string `json:"summary,omitempty" db:"summary"`
}

// String returns a human-readable representation of the book, mirroring the
// source model's __str__ dunder: "<title> by <author> (<year>)".
func (b Book) String() string {
	return fmt.Sprintf("%s by %s (%d)", b.Title, b.Author, b.PublishedYear)
}

// GoString returns a debug representation of the book, mirroring the source
// model's __repr__ dunder. It is invoked by the %#v fmt verb.
func (b Book) GoString() string {
	return fmt.Sprintf("<Book(id=%d, title='%s', author='%s', year=%d)>",
		b.ID, b.Title, b.Author, b.PublishedYear)
}
