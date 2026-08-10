package internal

import (
	"context"
	"database/sql"
	"fmt"
)

// Book represents a book in the catalog.
//
// MIGRATION_NOTE: The Python source used a SQLAlchemy declarative model
// (declarative_base) with column/index/constraint metadata. In this Go
// migration the persistence mapping is expressed as a plain struct with
// `db` tags for sqlx scanning. The DDL (indexes, the unique(title, author)
// constraint, and the SERIAL primary key) belongs in the schema migration —
// see InitSchema in internal/database.go — not in this model file, so the
// table-level metadata from __table_args__ is not represented here.
type Book struct {
	// ID is the auto-incrementing primary key.
	//
	// MIGRATION_NOTE: The source used autoincrement Integer. Per the target
	// PostgreSQL dialect this maps to a SERIAL / GENERATED ALWAYS AS IDENTITY
	// column populated via INSERT ... RETURNING id (there is no LastInsertId
	// on lib/pq).
	ID int64 `db:"id" json:"id"`

	// Title is the book title (required, indexed, part of the unique constraint).
	Title string `db:"title" json:"title"`

	// Author is the book author (required, indexed, part of the unique constraint).
	Author string `db:"author" json:"author"`

	// PublishedYear is the year the book was published (required, indexed).
	PublishedYear int `db:"published_year" json:"published_year"`

	// Summary is an optional description of the book.
	//
	// MIGRATION_NOTE: The source column was nullable (Text, nullable=True).
	// We model it as sql.NullString so that a missing summary maps to SQL NULL
	// rather than an empty string. See coerceEmptyToNil for the write path.
	Summary sql.NullString `db:"summary" json:"summary,omitempty"`
}

// String returns a human-readable representation of the Book, mirroring the
// Python __str__ implementation.
func (b Book) String() string {
	return fmt.Sprintf("%s by %s (%d)", b.Title, b.Author, b.PublishedYear)
}

// GoString returns a debug representation of the Book, mirroring the Python
// __repr__ implementation. It is used by fmt's %#v verb.
func (b Book) GoString() string {
	return fmt.Sprintf("<Book(id=%d, title='%s', author='%s', year=%d)>",
		b.ID, b.Title, b.Author, b.PublishedYear)
}

// coerceEmptyToNil converts an empty summary string into a NULL-valued
// sql.NullString, preserving the source model's nullable semantics for the
// summary column: a missing/empty summary must be stored as SQL NULL, not as
// an empty string.
//
// MIGRATION_NOTE: This encodes the "source of truth for DB field nullability"
// decision from the migration debate. Repository/DAO write paths should pass
// user-supplied summaries through this helper before persisting.
func coerceEmptyToNil(summary string) sql.NullString {
	if summary == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: summary, Valid: true}
}

// SummaryOrEmpty returns the summary as a plain string, treating SQL NULL as
// the empty string. This is a convenience for read paths that prefer a
// (string) value over branching on Valid.
func (b Book) SummaryOrEmpty() string {
	if !b.Summary.Valid {
		return ""
	}
	return b.Summary.String
}

// ensureModelContext exists only to document that any I/O involving Book
// (persistence, lookup) must accept a context.Context as its first parameter,
// per the repository pattern used elsewhere in this package. It performs no
// work and is not exported.
//
// MIGRATION_NOTE: The SQLAlchemy model carried no I/O behavior itself; query
// logic lived in session usage at call sites. In Go that logic belongs in a
// repository type (not this model file). This placeholder is retained solely
// to signal the context requirement to the eventual repository author.
var _ = func(ctx context.Context) {}
