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

var createBookTests = []struct {
	name          string
	title         string
	author        string
	publishedYear int
	summary       sql.NullString
	wantErr       bool
}{
	{"with_summary", "Test Book", "Test Author", 2023, sql.NullString{String: "A test book summary", Valid: true}, false},
	{"without_summary", "Test Book No Summary", "Test Author", 2023, sql.NullString{}, false},
}

func TestCreateBook(t *testing.T) {
	for _, tt := range createBookTests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := database.OpenDatabase("postgres://user:password@localhost/dbname?sslmode=disable")
			require.NoError(t, err)
			defer db.Close()

			err = model.CreateSchema(db)
			require.NoError(t, err)

			book := model.NewBook(tt.title, tt.author, tt.publishedYear, tt.summary)
			err = model.Insert(db, book)
			if tt.wantErr {
				require.Error(t, err, "expected error")
			} else {
				require.NoError(t, err, "unexpected error")
			}

			assert.NotZero(t, book.ID)
			assert.Equal(t, tt.title, book.Title)
			assert.Equal(t, tt.author, book.Author)
			assert.Equal(t, tt.publishedYear, book.PublishedYear)
			assert.Equal(t, tt.summary, book.Summary)
		})
	}
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

var booksWithSameTitleDifferentAuthorsTests = []struct {
	name     string
	title    string
	author   string
	year     int
	summary  sql.NullString
}{
	{"same_title_diff_authors_1", "Common Title", "Author One", 2023, sql.NullString{}},
	{"same_title_diff_authors_2", "Common Title", "Author Two", 2023, sql.NullString{}},
}

func TestBooksWithSameTitleDifferentAuthors(t *testing.T) {
	db, err := database.OpenDatabase("postgres://user:password@localhost/dbname?sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	err = model.CreateSchema(db)
	require.NoError(t, err)

	var books []*model.Book
	for _, tt := range booksWithSameTitleDifferentAuthorsTests {
		t.Run(tt.name, func(t *testing.T) {
			book := model.NewBook(tt.title, tt.author, tt.year, tt.summary)
			err = model.Insert(db, book)
			require.NoError(t, err)
			books = append(books, book)
		})
	}

	for i, b := range books {
		assert.NotZero(t, b.ID)
		if i > 0 {
			assert.NotEqual(t, books[i-1].ID, b.ID)
		}
	}

}

var booksWithSameAuthorDifferentTitlesTests = []struct {
	name     string
	title    string
	author   string
	year     int
	summary  sql.NullString
}{
	{"same_author_diff_titles_1", "First Book", "Prolific Author", 2023, sql.NullString{}},
	{"same_author_diff_titles_2", "Second Book", "Prolific Author", 2024, sql.NullString{}},
}

func TestBooksWithSameAuthorDifferentTitles(t *testing.T) {
	db, err := database.OpenDatabase("postgres://user:password@localhost/dbname?sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	err = model.CreateSchema(db)
	require.NoError(t, err)

	var books []*model.Book
	for _, tt := range booksWithSameAuthorDifferentTitlesTests {
		t.Run(tt.name, func(t *testing.T) {
			book := model.NewBook(tt.title, tt.author, tt.year, tt.summary)
			err = model.Insert(db, book)
			require.NoError(t, err)
			books = append(books, book)
		})
	}

	for i, b := range books {
		assert.NotZero(t, b.ID)
		if i > 0 {
			assert.NotEqual(t, books[i-1].ID, b.ID)
		}
	}
}