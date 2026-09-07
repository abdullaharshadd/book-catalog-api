Let me check the module path. The doc_test.go imports "github.com/yourusername/bookcatalog/internal" but go.mod says "migrated-app". Let me verify.

The instructions are explicit: module path is exactly "migrated-app". I'll use that. Let me write the model file with schema creation.

Let me verify the repr/str formats from tests.

The source clearly shows:
- `__repr__`: `<Book(id=1, title='...', author='...', year=...)>`
- `__str__`: `... by ... (...)`

Let me write the code.

```go
package internal
```

Let me compile-check.

run_compile_check:

```go
package internal

import (
	"context"
	"database/sql"
	"fmt"
)

// Version is the application version.
const Version = "1.0.0"

// Author is the application author.
const Author = "yourusername"

// Book represents a book in the catalog, mapped to the "books" table.
type Book struct {
	ID            int64
	Title         string
	Author        string
	PublishedYear int
	Summary       sql.NullString
}

// NewBook constructs a Book with the required fields.
func NewBook(title, author string, publishedYear int, summary string) *Book {
	b := &Book{
		Title:         title,
		Author:        author,
		PublishedYear: publishedYear,
	}
	if summary != "" {
		b.Summary = sql.NullString{String: summary, Valid: true}
	}
	return b
}

// String returns the human-readable representation of the book.
func (b *Book) String() string {
	return fmt.Sprintf("%s by %s (%d)", b.Title, b.Author, b.PublishedYear)
}

// Repr returns the debug representation of the book.
func (b *Book) Repr() string {
	return fmt.Sprintf("<Book(id=%d, title='%s', author='%s', year=%d)>",
		b.ID, b.Title, b.Author, b.PublishedYear)
}

const createBooksTableDDL = `
CREATE TABLE IF NOT EXISTS books (
	id             SERIAL PRIMARY KEY,
	title          VARCHAR(255) NOT NULL,
	author         VARCHAR(255) NOT NULL,
	published_year INTEGER NOT NULL,
	summary        TEXT,
	CONSTRAINT unique_title_author UNIQUE (title, author)
);
CREATE INDEX IF NOT EXISTS ix_books_title ON books (title);
CREATE INDEX IF NOT EXISTS ix_books_author ON books (author);
CREATE INDEX IF NOT EXISTS ix_books_published_year ON books (published_year);
`

// Migrate creates the books table and its indexes if they do not exist.
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, createBooksTableDDL); err != nil {
		return fmt.Errorf("creating books schema: %w", err)
	}
	return nil
}
```

Wait — doc_test.go already declares Version and Author usage in package internal_test, referencing internal.Version and internal.Author. But I don't know if Version/Author are already declared in another migrated file (doc.go). If they are, I'd collide. The doc_test references them, so there must be a doc.go defining them. Let me not redeclare Version/Author.

Let me remove those constants to avoid collision.


{"name": "search_files", "arguments": {"scope": "target", "query": "const Version"}}