```go
package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type MockDB struct {
	mock.Mock
}

func (m *MockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	args := m.Called(query, args)
	return args.Get(0).(*sqlx.Rows), args.Error(1)
}

func (m *MockDB) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	args := m.Called(query, args)
	return args.Error(0)
}

func (m *MockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	args := m.Called(query, args)
	return args.Get(0).(*sqlx.Row)
}

func TestRoot(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)

	root(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	expected := map[string]string{
		"message": "Welcome to Book Catalog API",
		"version": "1.0.0",
		"docs_url": "/docs",
	}
	var actual map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &actual)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestListBooks(t *testing.T) {
	type args struct {
		skip  int
		limit int
	}
	type result struct {
		books []BookResponse
		err   error
	}
	tests := []struct {
		name    string
		args    args
		result  result
		wantErr bool
	}{
		{
			name: "valid request",
			args: args{skip: 0, limit: 10},
			result: result{
				books: []BookResponse{{ID: 1, Title: "Book1", Author: "Author1", PublishedYear: 2021, Summary: "Summary1"}},
				err:   nil,
			},
			wantErr: false,
		},
		{
			name: "database error",
			args: args{skip: 0, limit: 10},
			result: result{
				books: nil,
				err:   fmt.Errorf("db error"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDB)
			mockRows := sqlx.NewStructRows([]BookResponse{tt.result.books...})
			mockDB.On("QueryContext", mock.Anything, mock.Anything, tt.args.skip, tt.args.limit).Return(mockRows, tt.result.err)
			database.SetSyncDB(mockDB)

			w := httptest.NewRecorder()
			r, _ := http.NewRequest("GET", "/books?skip=0&limit=10", nil)

			listBooks(w, r)

			if tt.wantErr {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
			} else {
				assert.Equal(t, http.StatusOK, w.Code)
				var actual []BookResponse
				err := json.Unmarshal(w.Body.Bytes(), &actual)
				require.NoError(t, err)
				assert.Equal(t, tt.result.books, actual)
			}
		})
	}
}

func TestGetBook(t *testing.T) {
	type args struct {
		bookID string
	}
	type result struct {
		book BookResponse
		err  error
	}
	tests := []struct {
		name    string
		args    args
		result  result
		wantErr bool
	}{
		{
			name: "book found",
			args: args{bookID: "1"},
			result: result{
				book: BookResponse{ID: 1, Title: "Book1", Author: "Author1", PublishedYear: 2021, Summary: "Summary1"},
				err:  nil,
			},
			wantErr: false,
		},
		{
			name: "book not found",
			args: args{bookID: "2"},
			result: result{
				book: BookResponse{},
				err:  sql.ErrNoRows,
			},
			wantErr: true,
		},
		{
			name: "database error",
			args: args{bookID: "3"},
			result: result{
				book: BookResponse{},
				err:  fmt.Errorf("db error"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDB)
			mockDB.On("GetContext", mock.Anything, mock.Anything, mock.Anything, tt.args.bookID).Return(tt.result.err)
			database.SetSyncDB(mockDB)

			w := httptest.NewRecorder()
			r, _ := http.NewRequest("GET", fmt.Sprintf("/books/%s", tt.args.bookID), nil)

			getBook(w, r)

			if tt.wantErr {
				if tt.result.err == sql.ErrNoRows {
					assert.Equal(t, http.StatusNotFound, w.Code)
				} else {
					assert.Equal(t, http.StatusInternalServerError, w.Code)
				}
			} else {
				assert.Equal(t, http.StatusOK, w.Code)
				var actual BookResponse
				err := json.Unmarshal(w.Body.Bytes(), &actual)
				require.NoError(t, err)
				assert.Equal(t, tt.result.book, actual)
			}
		})
	}
}

func TestCreateBook(t *testing.T) {
	type args struct {
		bookCreate BookCreate
	}
	type result struct {
		newID int64
		err   error
	}
	tests := []struct {
		name    string
		args    args
		result  result
		wantErr bool
	}{
		{
			name: "book created",
			args: args{bookCreate: BookCreate{Title: "Book1", Author: "Author1", PublishedYear: 2021, Summary: "Summary1"}},
			result: result{
				newID: 1,
				err:   nil,
			},
			wantErr: false,
		},
		{
			name: "unique violation",
			args: args{bookCreate: BookCreate{Title: "Book1", Author: "Author1", PublishedYear: 2021, Summary: "Summary1"}},
			result: result{
				newID: 0,
				err:   &pq.Error{Code: pq.ErrorCode{Name: "unique_violation"}},
			},
			wantErr: true,
		},
		{
			name: "database error",
			args: args{bookCreate: BookCreate{Title: "Book1", Author: "Author1", PublishedYear: 2021, Summary: "Summary1"}},
			result: result{
				newID: 0,
				err:   fmt.Errorf("db error"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDB)
			mockDB.On("QueryRowContext", mock.Anything, mock.Anything, tt.args.bookCreate.Title, tt.args.bookCreate.Author, tt.args.bookCreate.PublishedYear, tt.args.bookCreate.Summary).Return(&sqlx.Row{Err: tt.result.err, ScanFunc: func(dest ...interface{}) error {
				if dest[0] != nil {
					*dest[0].(*int64) = tt.result.newID
				}
				return nil
			}})
			database.SetSyncDB(mockDB)

			body, _ := json.Marshal(tt.args.bookCreate)
			w := httptest.NewRecorder()
			r, _ := http.NewRequest("POST", "/books", bytes.NewBuffer(body))

			createBook(w, r)

			if tt.wantErr {
				if pqErr, ok := tt.result.err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
					assert.Equal(t, http.StatusBadRequest, w.Code)
				} else {
					assert.Equal(t, http.StatusInternalServerError, w.Code)
				}
			} else {
				assert.Equal(t, http.StatusCreated, w.Code)
				var actual BookResponse
				err := json.Unmarshal(w.Body.Bytes(), &actual)
				require.NoError(t, err)
				assert.Equal(t, tt.result.newID, actual.ID)
			}
		})
	}
}

func TestUpdateBook(t *testing.T) {
	type args struct {
		bookID     string
		bookUpdate BookUpdate
	}
	type result struct {
		updatedID int64
		err       error
	}
	tests := []struct {
		name    string
		args    args
		result  result
		wantErr bool
	}{
		{
			name: "book updated",
			args: args{bookID: "1", bookUpdate: BookUpdate{Title: "Book1", Author: "Author1", PublishedYear: 2021, Summary: "Summary1"}},
			result: result{
				updatedID: 1,
				err:       nil,
			},
			wantErr: false,
		},
		{
			name: "unique violation",
			args: args{bookID: "2", bookUpdate: BookUpdate{Title: "Book1", Author: "Author1", PublishedYear: 2021, Summary: "Summary1"}},
			result: result{
				updatedID: 0,
				err:       &pq.Error{Code: pq.ErrorCode{Name: "unique_violation"}},
			},
			wantErr: true,
		},
		{
			name: "book not found",
			args: args{bookID: "3", bookUpdate: BookUpdate{Title: "Book1", Author: "Author1", PublishedYear: 2021, Summary: "Summary1"}},
			result: result{
				updatedID: 0,
				err:       sql.ErrNoRows,
			},
			wantErr: true,
		},
		{
			name: "database error",
			args: args{bookID: "4", bookUpdate: BookUpdate{Title: "Book1", Author: "Author1", PublishedYear: 2021, Summary: "Summary1"}},
			result: result{
				updatedID: 0,
				err:       fmt.Errorf("db error"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDB)
			mockDB.On("QueryRowContext", mock.Anything, mock.Anything, tt.args.bookUpdate.Title, tt.args.bookUpdate.Author, tt.args.bookUpdate.PublishedYear, tt.args.bookUpdate.Summary, tt.args.bookID).Return(&sqlx.Row{Err: tt.result.err, ScanFunc: func(dest ...interface{}) error {
				if dest[0] != nil {
					*dest[0].(*int64) = tt.result.updatedID
				}
				return nil
			}})
			database.SetSyncDB(mockDB)

			body, _ := json.Marshal(tt.args.bookUpdate)
			w := httptest.NewRecorder()
			r, _ := http.NewRequest("PUT", fmt.Sprintf("/books/%s", tt.args.bookID), bytes.NewBuffer(body))

			updateBook(w, r)

			if tt.wantErr {
				if tt.result.err == sql.ErrNoRows {
					assert.Equal(t, http.StatusNotFound, w.Code)
				} else if pqErr, ok := tt.result.err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
					assert.Equal(t, http.StatusBadRequest, w.Code)
				} else {
					assert.Equal(t, http.StatusInternalServerError, w.Code)
				}
			} else {
				assert.Equal(t, http.StatusOK, w.Code)
				var actual BookResponse
				err := json.Unmarshal(w.Body.Bytes(), &actual)
				require.NoError(t, err)
				assert.Equal(t, tt.result.updatedID, actual.ID)
			}
		})
	}
}

func TestDeleteBook(t *testing.T) {
	type args struct {
		bookID string
	}
	type result struct {
		deletedID int64
		err       error
	}
	tests := []struct {
		name    string
		args    args
		result  result
		wantErr bool
	}{
		{
			name: "book deleted",
			args: args{bookID: "1"},
			result: result{
				deletedID: 1,
				err:       nil,
			},
			wantErr: false,
		},
		{
			name: "book not found",
			args: args{bookID: "2"},
			result: result{
				deletedID: 0,
				err:       sql.ErrNoRows,
			},
			wantErr: true,
		},
		{
			name: "database error",
			args: args{bookID: "3"},
			result: result{
				deletedID: 0,
				err:       fmt.Errorf("db error"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDB)
			mockDB.On("QueryRowContext", mock.Anything, mock.Anything, tt.args.bookID).Return(&sqlx.Row{Err: tt.result.err, ScanFunc: func(dest ...interface{}) error {
				if dest[0] != nil {
					*dest[0].(*int64) = tt.result.deletedID
				}
				return nil
			}})
			database.SetSyncDB(mockDB)

			w := httptest.NewRecorder()
			r, _ := http.NewRequest("DELETE", fmt.Sprintf("/books/%s", tt.args.bookID), nil)

			deleteBook(w, r)

			if tt.wantErr {
				if tt.result.err == sql.ErrNoRows {
					assert.Equal(t, http.StatusNotFound, w.Code)
				} else {
					assert.Equal(t, http.StatusInternalServerError, w.Code)
				}
			} else {
				assert.Equal(t, http.StatusNoContent, w.Code)
			}
		})
	}
}

func TestHealthCheck(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/health", nil)

	healthCheck(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	expected := map[string]string{
		"status":  "healthy",
		"service": "book-catalog-api",
	}
	var actual map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &actual)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}
```