// Package catalog defines the domain data model for the Book Catalog API.
//
// This file was migrated from app/models.py, which declared a SQLAlchemy
// declarative ORM model (Book) mapped to the "books" table. Idiomatic Go does
// not use a declarative-ORM base class; instead a plain struct describes the
// row shape and struct tags document column mappings. Persistence is handled
// by database/sql (see internal/database.py.go), not by the model itself.
//
// The SQLAlchemy metadata maps onto Go as follows:
//
//   - declarative_base()                 -> no equivalent; Book is a plain struct.
//   - Column(...) declarations           -> exported struct fields + db tags.
//   - index=True on columns              -> DDL indexes (see CreateTableStmts).
//   - UniqueConstraint('title','author') -> a UNIQUE table constraint in DDL.
//   - __repr__ / __str__                 -> Go's fmt.Stringer + a Repr method.
package catalog

import "fmt"

// Book represents a single book in the catalog.
//
// Fields:
//   - ID:            primary key, auto-incrementing integer.
//   - Title:         book title (required).
//   - Author:        book author (required).
//   - PublishedYear: year the book was published (required).
//   - Summary:       optional summary/description of the book.
//
// The combination of Title and Author is unique across the catalog; this is
// enforced at the database level by the unique_title_author constraint (see
// CreateTableStmts).
type Book struct {
	ID            int64  `json:"id" db:"id"`
	Title         string `json:"title" db:"title"`
	Author        string `json:"author" db:"author"`
	PublishedYear int    `json:"published_year" db:"published_year"`
	// Summary is optional. A nil pointer represents SQL NULL, distinguishing
	// "no summary" from an empty string.
	Summary *string `json:"summary,omitempty" db:"summary"`
}

// TableName returns the database table name for the Book model.
func (b Book) TableName() string {
	return "books"
}

// String implements fmt.Stringer, mirroring the Python __str__ method.
// It renders the book in the form "Title by Author (Year)".
func (b Book) String() string {
	return fmt.Sprintf("%s by %s (%d)", b.Title, b.Author, b.PublishedYear)
}

// Repr returns a debug-oriented representation of the book, mirroring the
// Python __repr__ method. Go has no __repr__ concept, so this is exposed as an
// explicit method for logging and diagnostics.
func (b Book) Repr() string {
	return fmt.Sprintf("<Book(id=%d, title='%s', author='%s', year=%d)>",
		b.ID, b.Title, b.Author, b.PublishedYear)
}

// CreateTableStmts returns the DDL statements required to create the books
// table together with the indexes and unique constraint that the original
// SQLAlchemy model declared.
//
// MIGRATION_NOTE: SQLAlchemy generated this schema implicitly from the model
// metadata (Column index=True, UniqueConstraint, autoincrement). In Go there
// is no ORM to emit DDL, so the schema is expressed explicitly here. These
// statements target SQLite (matching the original aiosqlite setup); review and
// adjust the types/AUTOINCREMENT syntax if the deployment target changes.
// Callers typically execute these during startup (e.g. from InitDB in
// internal/database.py.go).
func CreateTableStmts() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS books (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			title          VARCHAR(255) NOT NULL,
			author         VARCHAR(255) NOT NULL,
			published_year INTEGER NOT NULL,
			summary        TEXT,
			CONSTRAINT unique_title_author UNIQUE (title, author)
		)`,
		`CREATE INDEX IF NOT EXISTS ix_books_title ON books (title)`,
		`CREATE INDEX IF NOT EXISTS ix_books_author ON books (author)`,
		`CREATE INDEX IF NOT EXISTS ix_books_published_year ON books (published_year)`,
	}
}
