package internal

import (
	"database/sql"
	"fmt"
	"time"
)

// Book represents a book in the catalog.
type Book struct {
	// ID is the primary key, auto-incrementing (SERIAL/IDENTITY in PostgreSQL).
	ID int64 `json:"id" db:"id"`

	// Title is the book title (required).
	Title string `json:"title" db:"title"`

	// Author is the book author (required).
	Author string `json:"author" db:"author"`

	// Description is an optional description of the book.
	Description sql.NullString `json:"description" db:"description"`

	// Published indicates whether the book has been published.
	Published bool `json:"published" db:"published"`

	// CreatedAt is the timestamp when the record was created.
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt is the timestamp when the record was last updated.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// String returns a human-readable representation of the book.
func (b Book) String() string {
	return fmt.Sprintf("%s by %s", b.Title, b.Author)
}

// GoString returns a debug representation of the book.
func (b Book) GoString() string {
	return fmt.Sprintf("<Book(id=%d, title='%s', author='%s')>",
		b.ID, b.Title, b.Author)
}