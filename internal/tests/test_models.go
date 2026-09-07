// internal/tests/test_models.go

package tests

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"migrated-app/internal/database"
	"migrated-app/internal/model"
	"migrated-app/internal/schemas"
)

// MIGRATION_NOTE: In Go, there is no direct equivalent to Python's fixtures.
// We initialize the database connection and schema in each test function.
// This is intentional and aligns with Go's testing framework.

func TestCreateBook(t *testing.T) {
	db, err := database.OpenDatabase("postgres://user:password@localhost/dbname?sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	err = model.CreateSchema(db)
	require.NoError(t, err)

	book := model.NewBook("Test Book", "Test Author", 2023, sql.NullString{String: "A test book summary", Valid: true})
	err = model.Insert(db, book)
	require.NoError(t, err)

	assert.NotZero(t, book.ID)
	assert.Equal(t, "Test Book", book.Title)
	assert.Equal(t, "Test Author", book.Author)
	assert.Equal(t, 2023, book.PublishedYear)
	assert.True(t, book.Summary.Valid)
	assert.Equal(t, "A test book summary", book.Summary.String)
}

func TestCreateBookWithoutSummary(t *testing.T) {
	db, err := database.OpenDatabase("postgres://user:password@localhost/dbname?sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	err = model.CreateSchema(db)
	require.NoError(t, err)

	book := model.NewBook("Test Book No Summary", "Test Author", 2023, sql.NullString{})
	err = model.Insert(db, book)
	require.NoError(t, err)

	assert.NotZero(t, book.ID)
	assert.Equal(t, "Test Book No Summary", book.Title)
	assert.Equal(t, "Test Author", book.Author)
	assert.Equal(t, 2023, book.PublishedYear)
	assert.False(t, book.Summary.Valid)
}

func TestBookRepr(t *testing.T) {
	db, err := database.OpenDatabase("postgres://user:password@localhost/dbname?sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	err = model.CreateSchema(db)
	require.NoError(t, err)

	book := model.NewBook("Repr Test", "Repr Author", 2023, sql.NullString{String: "Test summary", Valid: true})
	err = model.Insert(db, book)
	require.NoError(t, err)

	expectedRepr := `<Book(id=` + book.ID.String() + `, title='Repr Test', author='Repr Author', year=2023)>`
	assert.Equal(t, expectedRepr, book.Repr())
}

func TestBookStr(t *testing.T) {
	db, err := database.OpenDatabase("postgres://user:password@localhost/dbname?sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	err = model.CreateSchema(db)
	require.NoError(t, err)

	book := model.NewBook("String Test", "String Author", 2023, sql.NullString{})
	err = model.Insert(db, book)
	require.NoError(t, err)

	expectedStr := "String Test by String Author (2023)"
	assert.Equal(t, expectedStr, book.String())
}

func TestUniqueConstraintViolation(t *testing.T) {
	db, err := database.OpenDatabase("postgres://user:password@localhost/dbname?sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	err = model.CreateSchema(db)
	require.NoError(t, err)

	book1 := model.NewBook("Duplicate Test", "Duplicate Author", 2023, sql.NullString{})
	err = model.Insert(db, book1)
	require.NoError(t, err)

	book2 := model.NewBook("Duplicate Test", "Duplicate Author", 2024, sql.NullString{})
	err = model.Insert(db, book2)
	require.Error(t, err, "expected unique constraint violation")
}

func TestBooksWithSameTitleDifferentAuthors(t *testing.T) {
	db, err := database.OpenDatabase("postgres://user:password@localhost/dbname?sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	err = model.CreateSchema(db)
	require.NoError(t, err)

	book1 := model.NewBook("Common Title", "Author One", 2023, sql.NullString{})
	book2 := model.NewBook("Common Title", "Author Two", 2023, sql.NullString{})

	err = model.Insert(db, book1)
	require.NoError(t, err)

	err = model.Insert(db, book2)
	require.NoError(t, err)

	assert.NotZero(t, book1.ID)
	assert.NotZero(t, book2.ID)
	assert.NotEqual(t, book1.ID, book2.ID)
}

func TestBooksWithSameAuthorDifferentTitles(t *testing.T) {
	db, err := database.OpenDatabase("postgres://user:password@localhost/dbname?sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	err = model.CreateSchema(db)
	require.NoError(t, err)

	book1 := model.NewBook("First Book", "Prolific Author", 2023, sql.NullString{})
	book2 := model.NewBook("Second Book", "Prolific Author", 2024, sql.NullString{})

	err = model.Insert(db, book1)
	require.NoError(t, err)

	err = model.Insert(db, book2)
	require.NoError(t, err)

	assert.NotZero(t, book1.ID)
	assert.NotZero(t, book2.ID)
	assert.NotEqual(t, book1.ID, book2.ID)
}