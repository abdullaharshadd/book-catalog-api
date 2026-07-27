package internal

import "fmt"

// Book represents a book in the catalog.
//
// MIGRATION_NOTE: The Python source used SQLAlchemy's declarative ORM. Go has
// no built-in ORM; this is a plain domain struct. Persistence concerns (table
// name, column indexes, unique constraint) are expressed in the schema DDL
// (see InitDB / migrations) rather than as struct metadata. The original
// `UniqueConstraint('title', 'author', name='unique_title_author')` and the
// indexes on title, author and published_year must be created in the schema,
// e.g.:
//
//	CREATE TABLE books (
//	    id             SERIAL PRIMARY KEY,
//	    title          VARCHAR(255) NOT NULL,
//	    author         VARCHAR(255) NOT NULL,
//	    published_year INTEGER      NOT NULL,
//	    summary        TEXT,
//	    CONSTRAINT unique_title_author UNIQUE (title, author)
//	);
//	CREATE INDEX idx_books_title ON books (title);
//	CREATE INDEX idx_books_author ON books (author);
//	CREATE INDEX idx_books_published_year ON books (published_year);
type Book struct {
	// ID is the auto-incrementing primary key (Postgres SERIAL / IDENTITY).
	ID int64 `db:"id" json:"id"`
	// Title is the book title (required).
	Title string `db:"title" json:"title"`
	// Author is the book author (required).
	Author string `db:"author" json:"author"`
	// PublishedYear is the year the book was published (required).
	PublishedYear int `db:"published_year" json:"published_year"`
	// Summary is an optional summary/description of the book. It is a pointer
	// so a NULL database value maps to nil (and to null in JSON) rather than
	// causing a scan error or silently becoming an empty string.
	Summary *string `db:"summary" json:"summary"`
}

// String returns a human-readable representation of the book, mirroring the
// Python model's __str__ method.
func (b Book) String() string {
	return fmt.Sprintf("%s by %s (%d)", b.Title, b.Author, b.PublishedYear)
}

// GoString returns a debug-oriented representation of the book, mirroring the
// Python model's __repr__ method. It is used by the %#v fmt verb.
func (b Book) GoString() string {
	return fmt.Sprintf("<Book(id=%d, title='%s', author='%s', year=%d)>",
		b.ID, b.Title, b.Author, b.PublishedYear)
}
