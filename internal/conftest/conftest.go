// internal/conftest.go

package conftest

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"migrated-app/internal/database"
	"migrated-app/internal/model"
	"migrated-app/internal/schemas"
)

// MIGRATION_NOTE: In Go, there is no direct equivalent to Python's fixtures.
// We initialize the database connection and schema in each test function.
// This is intentional and aligns with Go's testing framework.

func TestMain(m *testing.M) {
	// Setup database
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost/testdb?sslmode=disable"
	}

	db, err := database.OpenDatabase(dbURL)
	if err != nil {
		log.Fatalf("Unable to open database: %v", err)
	}
	defer db.Close()

	err = model.CreateSchema(db)
	if err != nil {
		log.Fatalf("Unable to create schema: %v", err)
	}

	// Run tests
	code := m.Run()

	// Cleanup database
	err = dropAllTables(db)
	if err != nil {
		log.Fatalf("Unable to drop all tables: %v", err)
	}

	os.Exit(code)
}

func dropAllTables(db *sql.DB) error {
	_, err := db.ExecContext(context.Background(), `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`)
	return errors.Wrap(err, "failed to drop all tables")
}

func TestCreateBook(t *testing.T) {
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost/testdb?sslmode=disable"
	}

	db, err := database.OpenDatabase(dbURL)
	require.NoError(t, err)
	defer db.Close()

	err = model.CreateSchema(db)
	require.NoError(t, err)

	book := model.NewBook("Test Book", "Test Author", 2023, sql.NullString{String: "", Valid: false})
	err = book.Insert(db)
	require.NoError(t, err)

	fetchedBook, err := model.GetByID(db, book.ID)
	require.NoError(t, err)
	assert.Equal(t, book.Title, fetchedBook.Title)
	assert.Equal(t, book.Author, fetchedBook.Author)
	assert.Equal(t, book.YearPublished, fetchedBook.YearPublished)
	assert.Equal(t, book.Summary, fetchedBook.Summary)
}

func TestCreateBookWithoutSummary(t *testing.T) {
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost/testdb?sslmode=disable"
	}

	db, err := database.OpenDatabase(dbURL)
	require.NoError(t, err)
	defer db.Close()

	err = model.CreateSchema(db)
	require.NoError(t, err)

	book := model.NewBook("Test Book Without Summary", "Test Author", 2023, sql.NullString{Valid: false})
	err = book.Insert(db)
	require.NoError(t, err)

	fetchedBook, err := model.GetByID(db, book.ID)
	require.NoError(t, err)
	assert.Equal(t, book.Title, fetchedBook.Title)
	assert.Equal(t, book.Author, fetchedBook.Author)
	assert.Equal(t, book.YearPublished, fetchedBook.YearPublished)
	assert.False(t, fetchedBook.Summary.Valid)
}

func TestBookRepr(t *testing.T) {
	book := model.NewBook("Test Book", "Test Author", 2023, sql.NullString{String: "This is a summary", Valid: true})
	assert.Equal(t, "Book(title=Test Book, author=Test Author, year_published=2023, summary=This is a summary)", book.String())
}

func TestBookStr(t *testing.T) {
	book := model.NewBook("Test Book", "Test Author", 2023, sql.NullString{String: "This is a summary", Valid: true})
	assert.Equal(t, "Test Book by Test Author (2023)", book.String())
}

func TestUniqueConstraintViolation(t *testing.T) {
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost/testdb?sslmode=disable"
	}

	db, err := database.OpenDatabase(dbURL)
	require.NoError(t, err)
	defer db.Close()

	err = model.CreateSchema(db)
	require.NoError(t, err)

	book := model.NewBook("Duplicate Title", "Test Author", 2023, sql.NullString{String: "This is a summary", Valid: true})
	err = book.Insert(db)
	require.NoError(t, err)

	duplicateBook := model.NewBook("Duplicate Title", "Test Author", 2023, sql.NullString{String: "Another summary", Valid: true})
	err = duplicateBook.Insert(db)
	require.Error(t, err)
}

func TestBooksWithSameTitleDifferentAuthors(t *testing.T) {
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost/testdb?sslmode=disable"
	}

	db, err := database.OpenDatabase(dbURL)
	require.NoError(t, err)
	defer db.Close()

	err = model.CreateSchema(db)
	require.NoError(t, err)

	book1 := model.NewBook("Same Title", "Author One", 2023, sql.NullString{String: "Summary One", Valid: true})
	err = book1.Insert(db)
	require.NoError(t, err)

	book2 := model.NewBook("Same Title", "Author Two", 2023, sql.NullString{String: "Summary Two", Valid: true})
	err = book2.Insert(db)
	require.NoError(t, err)

	books, err := model.ListBooks(db)
	require.NoError(t, err)
	assert.Len(t, books, 2)
	assert.Equal(t, "Same Title", books[0].Title)
	assert.Equal(t, "Author One", books[0].Author)
	assert.Equal(t, "Same Title", books[1].Title)
	assert.Equal(t, "Author Two", books[1].Author)
}

func TestBooksWithSameAuthorDifferentTitles(t *testing.T) {
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost/testdb?sslmode=disable"
	}

	db, err := database.OpenDatabase(dbURL)
	require.NoError(t, err)
	defer db.Close()

	err = model.CreateSchema(db)
	require.NoError(t, err)

	book1 := model.NewBook("Title One", "Same Author", 2023, sql.NullString{String: "Summary One", Valid: true})
	err = book1.Insert(db)
	require.NoError(t, err)

	book2 := model.NewBook("Title Two", "Same Author", 2023, sql.NullString{String: "Summary Two", Valid: true})
	err = book2.Insert(db)
	require.NoError(t, err)

	books, err := model.ListBooks(db)
	require.NoError(t, err)
	assert.Len(t, books, 2)
	assert.Equal(t, "Title One", books[0].Title)
	assert.Equal(t, "Title Two", books[1].Title)
	assert.Equal(t, "Same Author", books[0].Author)
	assert.Equal(t, "Same Author", books[1].Author)
}