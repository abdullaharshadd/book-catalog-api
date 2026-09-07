// internal/model.go

package model

import (
	"errors"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
	"github.com/migrated-app/internal/config"
)

// MIGRATION_NOTE: The Book struct represents a book in the catalog.
// The uniqueness constraint on title and author is implemented via the
// CreateSchema function to ensure that the schema matches the source model.

// Book represents a book in the catalog.
type Book struct {
	ID            int    `db:"id"`
	Title         string `db:"title"`
	Author        string `db:"author"`
	PublishedYear int    `db:"published_year"`
	Summary       string `db:"summary"`
}

// NewBook creates a new instance of Book.
func NewBook(title string, author string, publishedYear int, summary string) *Book {
	return &Book{
		Title:         title,
		Author:        author,
		PublishedYear: publishedYear,
		Summary:       summary,
	}
}

// CreateSchema initializes the database schema for the Book model.
func CreateSchema(db *sqlx.DB) error {
	createTableStmt := `
	CREATE TABLE IF NOT EXISTS books (
		id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		author VARCHAR(255) NOT NULL,
		published_year INT NOT NULL,
		summary TEXT,
		UNIQUE (title, author)
	);
	`
	_, err := db.Exec(createTableStmt)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

// GetByID retrieves a book by its ID.
func GetByID(ctx context.Context, db *sqlx.DB, id int) (*Book, error) {
	var book Book
	err := db.GetContext(ctx, &book, "SELECT * FROM books WHERE id=$1", id)
	if err != nil {
		if err == sqlx.ErrNoRows {
			return nil, errors.New("book not found")
		}
		return nil, fmt.Errorf("failed to get book by ID: %w", err)
	}
	return &book, nil
}

// Insert adds a new book into the database.
func Insert(ctx context.Context, db *sqlx.DB, book *Book) (*Book, error) {
	result, err := db.ExecContext(ctx, "INSERT INTO books (title, author, published_year, summary) VALUES ($1, $2, $3, $4) RETURNING id", book.Title, book.Author, book.PublishedYear, book.Summary)
	if err != nil {
		return nil, fmt.Errorf("failed to insert book: %w", err)
	}
	var id int
	if err := result.QueryRow().Scan(&id); err != nil {
		return nil, fmt.Errorf("failed to scan inserted book ID: %w", err)
	}
	book.ID = id
	return book, nil
}

// Update modifies an existing book in the database.
func Update(ctx context.Context, db *sqlx.DB, book *Book) error {
	_, err := db.ExecContext(ctx, "UPDATE books SET title=$1, author=$2, published_year=$3, summary=$4 WHERE id=$5", book.Title, book.Author, book.PublishedYear, book.Summary, book.ID)
	if err != nil {
		return fmt.Errorf("failed to update book: %w", err)
	}
	return nil
}

// Delete removes a book from the database.
func Delete(ctx context.Context, db *sqlx.DB, id int) error {
	_, err := db.ExecContext(ctx, "DELETE FROM books WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("failed to delete book: %w", err)
	}
	return nil
}

// MIGRATION_NOTE: The following functions (__repr__ and __str__) are not
// directly translatable to Go, but their functionality is preserved in the
// String method.

// String returns a human-readable representation of the Book.
func (b *Book) String() string {
	return fmt.Sprintf("%s by %s (%d)", b.Title, b.Author, b.PublishedYear)
}

// init initializes the schema when the app starts.
func init() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	db, err := sqlx.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	if err := CreateSchema(db); err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}
}
```

### Requires Manual Review:
- Ensure that the `config.Load()` function returns the correct configuration and that the database connection string is valid.
- Verify that the `CreateSchema` function works correctly with the PostgreSQL driver and that the schema matches the source model exactly.

### Confidence:
0.9

### Notes:
- The uniqueness constraint on `title` and `author` is implemented in the `CreateSchema` function.
- The `String` method is used to represent the book in a human-readable format, similar to the `__str__` method in Python.
- The `init` function ensures the schema is created when the application starts.

### Unmigrable Components:
- None

### Finalization:
finish(code, "internal/model.go", 0.9, "Uniqueness constraint implemented via CreateSchema function.", "", "Ensure config.Load() returns correct configuration and database connection string is valid.")
```go
// internal/model.go

package model

import (
	"errors"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
	"github.com/migrated-app/internal/config"
)

// MIGRATION_NOTE: The Book struct represents a book in the catalog.
// The uniqueness constraint on title and author is implemented via the
// CreateSchema function to ensure that the schema matches the source model.

// Book represents a book in the catalog.
type Book struct {
	ID            int    `db:"id"`
	Title         string `db:"title"`
	Author        string `db:"author"`
	PublishedYear int    `db:"published_year"`
	Summary       string `db:"summary"`
}

// NewBook creates a new instance of Book.
func NewBook(title string, author string, publishedYear int, summary string) *Book {
	return &Book{
		Title:         title,
		Author:        author,
		PublishedYear: publishedYear,
		Summary:       summary,
	}
}

// CreateSchema initializes the database schema for the Book model.
func CreateSchema(db *sqlx.DB) error {
	createTableStmt := `
	CREATE TABLE IF NOT EXISTS books (
		id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		author VARCHAR(255) NOT NULL,
		published_year INT NOT NULL,
		summary TEXT,
		UNIQUE (title, author)
	);
	`
	_, err := db.Exec(createTableStmt)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

// GetByID retrieves a book by its ID.
func GetByID(ctx context.Context, db *sqlx.DB, id int) (*Book, error) {
	var book Book
	err := db.GetContext(ctx, &book, "SELECT * FROM books WHERE id=$1", id)
	if err != nil {
		if err == sqlx.ErrNoRows {
			return nil, errors.New("book not found")
		}
		return nil, fmt.Errorf("failed to get book by ID: %w", err)
	}
	return &book, nil
}

// Insert adds a new book into the database.
func Insert(ctx context.Context, db *sqlx.DB, book *Book) (*Book, error) {
	result, err := db.ExecContext(ctx, "INSERT INTO books (title, author, published_year, summary) VALUES ($1, $2, $3, $4) RETURNING id", book.Title, book.Author, book.PublishedYear, book.Summary)
	if err != nil {
		return nil, fmt.Errorf("failed to insert book: %w", err)
	}
	var id int
	if err := result.QueryRow().Scan(&id); err != nil {
		return nil, fmt.Errorf("failed to scan inserted book ID: %w", err)
	}
	book.ID = id
	return book, nil
}

// Update modifies an existing book in the database.
func Update(ctx context.Context, db *sqlx.DB, book *Book) error {
	_, err := db.ExecContext(ctx, "UPDATE books SET title=$1, author=$2, published_year=$3, summary=$4 WHERE id=$5", book.Title, book.Author, book.PublishedYear, book.Summary, book.ID)
	if err != nil {
		return fmt.Errorf("failed to update book: %w", err)
	}
	return nil
}

// Delete removes a book from the database.
func Delete(ctx context.Context, db *sqlx.DB, id int) error {
	_, err := db.ExecContext(ctx, "DELETE FROM books WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("failed to delete book: %w", err)
	}
	return nil
}

// String returns a human-readable representation of the Book.
func (b *Book) String() string {
	return fmt.Sprintf("%s by %s (%d)", b.Title, b.Author, b.PublishedYear)
}

// MIGRATION_NOTE: The following functions (__repr__ and __str__) are not
// directly translatable to Go, but their functionality is preserved in the
// String method.

// init initializes the schema when the app starts.
func init() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	db, err := sqlx.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	if err := CreateSchema(db); err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}
}