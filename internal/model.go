package internal

import (
	"database/sql"
	"fmt"
)

// MIGRATION_NOTE: The Python source used SQLAlchemy's declarative_base to define
// the Book ORM model, with column metadata (types, indexes, unique constraint)
// embedded in the class. Go has no ORM-style declarative model layer in the
// standard library; the idiomatic equivalent is a plain struct that maps to
// table columns, with the schema (types, indexes, unique constraint) owned by
// the migration/DDL layer rather than the struct. The struct below carries
// `db` tags for sqlx scanning and `json` tags for the HTTP response layer.
//
// MIGRATION_NOTE: The SQLAlchemy UniqueConstraint('title', 'author',
// name='unique_title_author') and the column-level indexes are schema concerns.
// They are not expressible on a Go struct; they belong in the CREATE TABLE DDL
// (see the migration/schema for the target PostgreSQL database). The constant
// UniqueTitleAuthorConstraint is exported so the repository layer can detect
// violations of this specific constraint via IsUniqueViolation.
//
// MIGRATION_NOTE: The `summary` column is nullable in the source
// (nullable=True). It is modeled here as sql.NullString so a NULL in the
// database round-trips correctly through database/sql scanning rather than
// being silently coerced to an empty string.

// BooksTable is the name of the database table backing the Book model.
// In the Python source this was Book.__tablename__ = "books".
const BooksTable = "books"

// UniqueTitleAuthorConstraint is the name of the PostgreSQL unique constraint
// covering (title, author). It mirrors the SQLAlchemy
// UniqueConstraint(name='unique_title_author') and is used by the repository
// layer together with IsUniqueViolation to distinguish this specific
// duplicate-book conflict from other integrity errors.
const UniqueTitleAuthorConstraint = "unique_title_author"

// Book represents a single book in the catalog. It maps to the "books" table.
//
// Fields:
//   - ID: primary key, auto-incrementing (GENERATED ALWAYS AS IDENTITY in
//     PostgreSQL). Zero value means "not yet persisted".
//   - Title: book title (required).
//   - Author: book author (required).
//   - PublishedYear: year the book was published (required).
//   - Summary: optional summary/description; NULL when absent.
type Book struct {
	ID            int64          `db:"id" json:"id"`
	Title         string         `db:"title" json:"title"`
	Author        string         `db:"author" json:"author"`
	PublishedYear int            `db:"published_year" json:"published_year"`
	Summary       sql.NullString `db:"summary" json:"-"`
}

// String returns a human-readable representation of the book, mirroring the
// Python __str__ implementation ("{title} by {author} ({year})").
func (b Book) String() string {
	return fmt.Sprintf("%s by %s (%d)", b.Title, b.Author, b.PublishedYear)
}

// GoString returns a debug representation of the book, mirroring the Python
// __repr__ implementation. It satisfies the fmt.GoStringer interface so %#v
// produces this form.
func (b Book) GoString() string {
	return fmt.Sprintf("<Book(id=%d, title=%q, author=%q, year=%d)>",
		b.ID, b.Title, b.Author, b.PublishedYear)
}
