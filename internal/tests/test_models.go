package tests

import (
	"context"
	"testing"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"internal"
	"internal/database"
	"internal/model"
	"internal/schemas"
	"internal/conftest"
	"database/sql"
	"fmt"
)

func testCreateBook(t *testing.T, db *sqlx.DB) {
	// Create a basic book
	bookCreate := schemas.BookCreate{
		Title:         "Test Book",
		Author:       "Test Author",
		PublishedYear: 2023,
		Summary:      "A test book summary",
	}
	book, err := model.CreateBook(context.Background(), db, &bookCreate)
	require.NoError(t, err)
	assert.NotNil(t, book)
	assert.Equal(t, "Test Book", book.Title)
	assert.Equal(t, "Test Author", book.Author)
	assert.Equal(t, 2023, book.PublishedYear)
	assert.Equal(t, "A test book summary", book.Summary)
}

func testCreateBookWithoutSummary(t *testing.T, db *sqlx.DB) {
	bookCreate := schemas.BookCreate{
		Title:         "Test Book No Summary",
		Author:       "Test Author",
		PublishedYear: 2023,
	}
	book, err := model.CreateBook(context.Background(), db, &bookCreate)
	require.NoError(t, err)
	assert.NotNil(t, book)
	assert.Equal(t, "Test Book No Summary", book.Title)
	assert.Equal(t, "Test Author", book.Author)
	assert.Equal(t, 2023, book.PublishedYear)
	assert.Nil(t, book.Summary)
}

func testBookRepr(t *testing.T, db *sqlx.DB) {
	bookCreate := schemas.BookCreate{
		Title:         "Repr Test",
		Author:       "Repr Author",
		PublishedYear: 2023,
		Summary:      "Test summary",
	}
	book, err := model.CreateBook(context.Background(), db, &bookCreate)
	require.NoError(t, err)
	assert.NotNil(t, book)
	expectedRepr := fmt.Sprintf("<Book(id=%d, title='Repr Test', author='Repr Author', year=2023)>", book.ID)
	assert.Equal(t, expectedRepr, book.String())
}

func testBookStr(t *testing.T, db *sqlx.DB) {
	bookCreate := schemas.BookCreate{
		Title:         "String Test",
		Author:       "String Author",
		PublishedYear: 2023,
	}
	book, err := model.CreateBook(context.Background(), db, &bookCreate)
	require.NoError(t, err)
	assert.NotNil(t, book)
	expectedStr := "String Test by String Author (2023)"
	assert.Equal(t, expectedStr, book.String())
}

func testUniqueConstraintViolation(t *testing.T, db *sqlx.DB) {
	bookCreate1 := schemas.BookCreate{
		Title:         "Duplicate Test",
		Author:       "Duplicate Author",
		PublishedYear: 2023,
	}
	book1, err := model.CreateBook(context.Background(), db, &bookCreate1)
	require.NoError(t, err)
	assert.NotNil(t, book1)
	
	bookCreate2 := schemas.BookCreate{
		Title:         "Duplicate Test",
		Author:       "Duplicate Author",
		PublishedYear: 2024,
	}
	_, err = model.CreateBook(context.Background(), db, &bookCreate2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate key value violates unique constraint")
}

func testBooksWithSameTitleDifferentAuthors(t *testing.T, db *sqlx.DB) {
	bookCreate1 := schemas.BookCreate{
		Title:         "Common Title",
		Author:       "Author One",
		PublishedYear: 2023,
	}
	book1, err := model.CreateBook(context.Background(), db, &bookCreate1)
	require.NoError(t, err)
	assert.NotNil(t, book1)
	
	bookCreate2 := schemas.BookCreate{
		Title:         "Common Title",
		Author:       "Author Two",
		PublishedYear: 2023,
	}
	book2, err := model.CreateBook(context.Background(), db, &bookCreate2)
	require.NoError(t, err)
	assert.NotNil(t, book2)
	
	assert.NotEqual(t, book1.ID, book2.ID)
}

func testBooksWithSameAuthorDifferentTitles(t *testing.T, db *sqlx.DB) {
	bookCreate1 := schemas.BookCreate{
		Title:         "First Book",
		Author:       "Prolific Author",
		PublishedYear: 2023,
	}
	book1, err := model.CreateBook(context.Background(), db, &bookCreate1)
	require.NoError(t, err)
	assert.NotNil(t, book1)
	
	bookCreate2 := schemas.BookCreate{
		Title:         "Second Book",
		Author:       "Prolific Author",
		PublishedYear: 2024,
	}
	book2, err := model.CreateBook(context.Background(), db, &bookCreate2)
	require.NoError(t, err)
	assert.NotNil(t, book2)
	
	assert.NotEqual(t, book1.ID, book2.ID)
}

func testModels(t *testing.T, db *sqlx.DB) {
	conftest.DropAllTables(context.Background(), db)
	InitDB(context.Background(), db)
	
	testCreateBook(t, db)
	testCreateBookWithoutSummary(t, db)
	testBookRepr(t, db)
	testBookStr(t, db)
	testUniqueConstraintViolation(t, db)
	testBooksWithSameTitleDifferentAuthors(t, db)
	testBooksWithSameAuthorDifferentTitles(t, db)
}

// TestSuite represents the test suite for the Book Catalog API.
// It contains unit tests for models and schemas, plus integration tests for API endpoints.
func TestSuite(t *testing.T) {
	api := internal.NewBookCatalogAPI()
	db, err := database.NewDatabase()
	require.NoError(t, err)
	db = conftest.GetSyncDB(t, db)
	defer db.Close()
	
	testModels(t, db)
}
