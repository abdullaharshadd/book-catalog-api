// internal/conftest_test.go

package conftest

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"migrated-app/internal/database"
	"migrated-app/internal/model"
	"migrated-app/internal/schemas"
)

var (
	testDBURL = "postgres://user:password@localhost/testdb?sslmode=disable"
)

type MockDB struct {
	mock.Mock
}

func (m *MockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return m.Called(ctx, query, args).Get(0), m.Called(ctx, query, args).Error(1)
}

func (m *MockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return m.Called(ctx, query, args).Get(0).(*sql.Row)
}

func (m *MockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return m.Called(ctx, query, args).Get(0).(*sql.Rows), m.Called(ctx, query, args).Error(1)
}

func TestMain(t *testing.T) {
	tests := []struct {
		name        string
		setup       func()
		expectedErr bool
	}{
		{
			name: "successful setup and teardown",
			setup: func() {
				db, err := database.OpenDatabase(testDBURL)
				require.NoError(t, err)
				defer db.Close()

				err = model.CreateSchema(db)
				require.NoError(t, err)

				err = dropAllTables(db)
				require.NoError(t, err)
			},
			expectedErr: false,
		},
		{
			name: "failed drop all tables",
			setup: func() {
				db, err := database.OpenDatabase(testDBURL)
				require.NoError(t, err)
				defer db.Close()

				err = model.CreateSchema(db)
				require.NoError(t, err)

				err = dropAllTables(db)
				require.Error(t, err)
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
		})
	}
}

func TestCreateBook(t *testing.T) {
	mockDB := new(MockDB)
	mockDB.On("ExecContext", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	mockDB.On("QueryRowContext", mock.Anything, mock.Anything, mock.Anything).Return(sqlmock.NewRows([]string{"id", "title", "author", "year_published", "summary"}).AddRow(1, "Test Book", "Test Author", 2023, sql.NullString{String: "", Valid: false}))
	mockDB.On("QueryContext", mock.Anything, mock.Anything, mock.Anything).Return(sqlmock.NewRows([]string{"id", "title", "author", "year_published", "summary"}).AddRow(1, "Test Book", "Test Author", 2023, sql.NullString{String: "", Valid: false}), nil)

	book := model.NewBook("Test Book", "Test Author", 2023, sql.NullString{String: "", Valid: false})
	err := book.Insert(mockDB)
	require.NoError(t, err)

	fetchedBook, err := model.GetByID(mockDB, book.ID)
	require.NoError(t, err)
	assert.Equal(t, book.Title, fetchedBook.Title)
	assert.Equal(t, book.Author, fetchedBook.Author)
	assert.Equal(t, book.YearPublished, fetchedBook.YearPublished)
	assert.Equal(t, book.Summary, fetchedBook.Summary)

	mockDB.AssertExpectations(t)
}

func TestCreateBookWithoutSummary(t *testing.T) {
	mockDB := new(MockDB)
	mockDB.On("ExecContext", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	mockDB.On("QueryRowContext", mock.Anything, mock.Anything, mock.Anything).Return(sqlmock.NewRows([]string{"id", "title", "author", "year_published", "summary"}).AddRow(1, "Test Book Without Summary", "Test Author", 2023, sql.NullString{Valid: false}))
	mockDB.On("QueryContext", mock.Anything, mock.Anything, mock.Anything).Return(sqlmock.NewRows([]string{"id", "title", "author", "year_published", "summary"}).AddRow(1, "Test Book Without Summary", "Test Author", 2023, sql.NullString{Valid: false}), nil)

	book := model.NewBook("Test Book Without Summary", "Test Author", 2023, sql.NullString{Valid: false})
	err := book.Insert(mockDB)
	require.NoError(t, err)

	fetchedBook, err := model.GetByID(mockDB, book.ID)
	require.NoError(t, err)
	assert.Equal(t, book.Title, fetchedBook.Title)
	assert.Equal(t, book.Author, fetchedBook.Author)
	assert.Equal(t, book.YearPublished, fetchedBook.YearPublished)
	assert.False(t, fetchedBook.Summary.Valid)

	mockDB.AssertExpectations(t)
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
	mockDB := new(MockDB)
	mockDB.On("ExecContext", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("unique constraint violation"))

	book := model.NewBook("Duplicate Title", "Test Author", 2023, sql.NullString{String: "This is a summary", Valid: true})
	err := book.Insert(mockDB)
	require.Error(t, err)

	mockDB.AssertExpectations(t)
}

func TestBooksWithSameTitleDifferentAuthors(t *testing.T) {
	mockDB := new(MockDB)
	mockDB.On("ExecContext", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	mockDB.On("QueryContext", mock.Anything, mock.Anything, mock.Anything).Return(sqlmock.NewRows([]string{"id", "title", "author", "year_published", "summary"}).AddRow(1, "Same Title", "Author One", 2023, sql.NullString{String: "Summary One", Valid: true}).AddRow(2, "Same Title", "Author Two", 2023, sql.NullString{String: "Summary Two", Valid: true}), nil)

	book1 := model.NewBook("Same Title", "Author One", 2023, sql.NullString{String: "Summary One", Valid: true})
	err := book1.Insert(mockDB)
	require.NoError(t, err)

	book2 := model.NewBook("Same Title", "Author Two", 2023, sql.NullString{String: "Summary Two", Valid: true})
	err = book2.Insert(mockDB)
	require.NoError(t, err)

	books, err := model.ListBooks(mockDB)
	require.NoError(t, err)
	assert.Len(t, books, 2)
	assert.Equal(t, "Same Title", books[0].Title)
	assert.Equal(t, "Author One", books[0].Author)
	assert.Equal(t, "Same Title", books[1].Title)
	assert.Equal(t, "Author Two", books[1].Author)

	mockDB.AssertExpectations(t)
}

func TestBooksWithSameAuthorDifferentTitles(t *testing.T) {
	mockDB := new(MockDB)
	mockDB.On("ExecContext", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	mockDB.On("QueryContext", mock.Anything, mock.Anything, mock.Anything).Return(sqlmock.NewRows([]string{"id", "title", "author", "year_published", "summary"}).AddRow(1, "Title One", "Same Author", 2023, sql.NullString{String: "Summary One", Valid: true}).AddRow(2, "Title Two", "Same Author", 2023, sql.NullString{String: "Summary Two", Valid: true}), nil)

	book1 := model.NewBook("Title One", "Same Author", 2023, sql.NullString{String: "Summary One", Valid: true})
	err := book1.Insert(mockDB)
	require.NoError(t, err)

	book2 := model.NewBook("Title Two", "Same Author", 2023, sql.NullString{String: "Summary Two", Valid: true})
	err = book2.Insert(mockDB)
	require.NoError(t, err)

	books, err := model.ListBooks(mockDB)
	require.NoError(t, err)
	assert.Len(t, books, 2)
	assert.Equal(t, "Title One", books[0].Title)
	assert.Equal(t, "Title Two", books[1].Title)
	assert.Equal(t, "Same Author", books[0].Author)
	assert.Equal(t, "Same Author", books[1].Author)

	mockDB.AssertExpectations(t)
}

func TestHTTPHandlerWithMockDB(t *testing.T) {
	mockDB := new(MockDB)
	mockDB.On("QueryContext", mock.Anything, mock.Anything, mock.Anything).Return(sqlmock.NewRows([]string{"id", "title", "author", "year_published", "summary"}).AddRow(1, "Test Book", "Test Author", 2023, sql.NullString{String: "Summary", Valid: true}), nil)

	req := httptest.NewRequest(http.MethodGet, "/books", nil)
	w := httptest.NewRecorder()

	// Assuming ListBooksHandler is the handler function that uses the database
	ListBooksHandler(mockDB)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Test Book")

	mockDB.AssertExpectations(t)
}