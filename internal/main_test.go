package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDB struct {
	mock.Mock
}

func (m *mockDB) ListBooks(ctx context.Context, skip, limit int) ([]*model.Book, error) {
	args := m.Called(ctx, skip, limit)
	return args.Get(0).([]*model.Book), args.Error(1)
}

func (m *mockDB) GetBookByID(ctx context.Context, id int) (*model.Book, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.Book), args.Error(1)
}

func (m *mockDB) CreateBook(ctx context.Context, book *model.Book) error {
	args := m.Called(ctx, book)
	return args.Error(0)
}

func (m *mockDB) UpdateBook(ctx context.Context, book *model.Book, update schemas.BookUpdate) error {
	args := m.Called(ctx, book, update)
	return args.Error(0)
}

func (m *mockDB) DeleteBook(ctx context.Context, book *model.Book) error {
	args := m.Called(ctx, book)
	return args.Error(0)
}

func TestRootHandler(t *testing.T) {
	testCases := []struct {
		name         string
		expectedCode int
		expectedBody string
	}{
		{"welcome message", http.StatusOK, `{"message":"Welcome to Book Catalog API","version":"1.0.0","docs_url":"/docs"}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(rootHandler)
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedCode, rr.Code)
			assert.JSONEq(t, tc.expectedBody, rr.Body.String())
		})
	}
}

func TestListBooksHandler(t *testing.T) {
	testCases := []struct {
		name               string
		query              string
		mockBehavior       func(m *mockDB)
		expectedCode       int
		expectedBody       string
		expectedSideEffect string
	}{
		{"within bounds", "?skip=0&limit=10", func(m *mockDB) {
			m.On("ListBooks", mock.Anything, 0, 10).Return([]*model.Book{{ID: 1, Title: "Test Book"}}, nil)
		}, http.StatusOK, `[{"id":1,"title":"Test Book"}]`, ""},
		{"exceeds max", "?skip=0&limit=1001", func(m *mockDB) {
			m.On("ListBooks", mock.Anything, 0, 1000).Return([]*model.Book{{ID: 1, Title: "Test Book"}}, nil)
		}, http.StatusOK, `[{"id":1,"title":"Test Book"}]`, ""},
		{"db error", "?skip=0&limit=10", func(m *mockDB) {
			m.On("ListBooks", mock.Anything, 0, 10).Return(nil, errors.New("db error"))
		}, http.StatusInternalServerError, "", "Error retrieving books"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := &mockDB{}
			tc.mockBehavior(db)

			r := buildRouter()
			req, err := http.NewRequest("GET", "/books/"+tc.query, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedCode, rr.Code)
			assert.JSONEq(t, tc.expectedBody, rr.Body.String())

			if tc.expectedSideEffect != "" {
				db.AssertCalled(t, "ListBooks", mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}
}

func TestGetBookHandler(t *testing.T) {
	testCases := []struct {
		name               string
		bookID             string
		mockBehavior       func(m *mockDB)
		expectedCode       int
		expectedBody       string
		expectedSideEffect string
	}{
		{"existing book", "1", func(m *mockDB) {
			m.On("GetBookByID", mock.Anything, 1).Return(&model.Book{ID: 1, Title: "Test Book"}, nil)
		}, http.StatusOK, `{"id":1,"title":"Test Book"}`, ""},
		{"not found", "2", func(m *mockDB) {
			m.On("GetBookByID", mock.Anything, 2).Return(nil, sql.ErrNoRows)
		}, http.StatusNotFound, "", ""},
		{"db error", "3", func(m *mockDB) {
			m.On("GetBookByID", mock.Anything, 3).Return(nil, errors.New("db error"))
		}, http.StatusInternalServerError, "", "Error retrieving book"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := &mockDB{}
			tc.mockBehavior(db)

			r := buildRouter()
			req, err := http.NewRequest("GET", "/books/"+tc.bookID, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedCode, rr.Code)
			assert.JSONEq(t, tc.expectedBody, rr.Body.String())

			if tc.expectedSideEffect != "" {
				db.AssertCalled(t, "GetBookByID", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestCreateBookHandler(t *testing.T) {
	testCases := []struct {
		name               string
		body               string
		mockBehavior       func(m *mockDB)
		expectedCode       int
		expectedBody       string
		expectedSideEffect string
	}{
		{"valid book", `{"title":"New Book","author":"Author Name","published_year":2023}`, func(m *mockDB) {
			m.On("CreateBook", mock.Anything, mock.Anything).Return(nil)
		}, http.StatusCreated, `{"id":0,"title":"New Book","author":"Author Name","published_year":2023}`, ""},
		{"duplicate book", `{"title":"Existing Book","author":"Existing Author","published_year":2022}`, func(m *mockDB) {
			m.On("CreateBook", mock.Anything, mock.Anything).Return(&model.UniqueConstraintViolationError{})
		}, http.StatusBadRequest, "", ""},
		{"invalid book", `{"title":"Invalid","author":"","published_year":2023}`, func(m *mockDB) {
			m.On("CreateBook", mock.Anything, mock.Anything).Return(errors.New("db error"))
		}, http.StatusBadRequest, "", ""},
		{"db error", `{"title":"Error Book","author":"Error Author","published_year":2023}`, func(m *mockDB) {
			m.On("CreateBook", mock.Anything, mock.Anything).Return(errors.New("db error"))
		}, http.StatusInternalServerError, "", "Error creating book"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := &mockDB{}
			tc.mockBehavior(db)

			r := buildRouter()
			req, err := http.NewRequest("POST", "/books/", bytes.NewBufferString(tc.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedCode, rr.Code)
			assert.JSONEq(t, tc.expectedBody, rr.Body.String())

			if tc.expectedSideEffect != "" {
				db.AssertCalled(t, "CreateBook", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestUpdateBookHandler(t *testing.T) {
	testCases := []struct {
		name               string
		bookID             string
		body               string
		mockBehavior       func(m *mockDB)
		expectedCode       int
		expectedBody       string
		expectedSideEffect string
	}{
		{"valid update", "1", `{"title":"Updated Book","author":"Updated Author","published_year":2024}`, func(m *mockDB) {
			m.On("GetBookByID", mock.Anything, 1).Return(&model.Book{ID: 1, Title: "Old Book"}, nil)
			m.On("UpdateBook", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		}, http.StatusOK, `{"id":1,"title":"Updated Book","author":"Updated Author","published_year":2024}`, ""},
		{"not found", "2", `{"title":"Not Found Book","author":"Not Found Author","published_year":2025}`, func(m *mockDB) {
			m.On("GetBookByID", mock.Anything, 2).Return(nil, sql.ErrNoRows)
		}, http.StatusNotFound, "", ""},
		{"duplicate book", "1", `{"title":"Existing Book","author":"Existing Author","published_year":2026}`, func(m *mockDB) {
			m.On("GetBookByID", mock.Anything, 1).Return(&model.Book{ID: 1, Title: "Old Book"}, nil)
			m.On("UpdateBook", mock.Anything, mock.Anything, mock.Anything).Return(&model.UniqueConstraintViolationError{})
		}, http.StatusBadRequest, "", ""},
		{"db error", "1", `{"title":"Error Book","author":"Error Author","published_year":2027}`, func(m *mockDB) {
			m.On("GetBookByID", mock.Anything, 1).Return(&model.Book{ID: 1, Title: "Old Book"}, nil)
			m.On("UpdateBook", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db error"))
		}, http.StatusInternalServerError, "", "Error updating book"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := &mockDB{}
			tc.mockBehavior(db)

			r := buildRouter()
			req, err := http.NewRequest("PUT", "/books/"+tc.bookID, bytes.NewBufferString(tc.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedCode, rr.Code)
			assert.JSONEq(t, tc.expectedBody, rr.Body.String())

			if tc.expectedSideEffect != "" {
				db.AssertCalled(t, "UpdateBook", mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}
}

func TestDeleteBookHandler(t *testing.T) {
	testCases := []struct {
		name               string
		bookID             string
		mockBehavior       func(m *mockDB)
		expectedCode       int
		expectedSideEffect string
	}{
		{"valid delete", "1", func(m *mockDB) {
			m.On("GetBookByID", mock.Anything, 1).Return(&model.Book{ID: 1, Title: "Book to Delete"}, nil)
			m.On("DeleteBook", mock.Anything, mock.Anything).Return(nil)
		}, http.StatusNoContent, ""},
		{"not found", "2", func(m *mockDB) {
			m.On("GetBookByID", mock.Anything, 2).Return(nil, sql.ErrNoRows)
		}, http.StatusNotFound, ""},
		{"db error", "1", func(m *mockDB) {
			m.On("GetBookByID", mock.Anything, 1).Return(&model.Book{ID: 1, Title: "Book to Delete"}, nil)
			m.On("DeleteBook", mock.Anything, mock.Anything).Return(errors.New("db error"))
		}, http.StatusInternalServerError, "Error deleting book"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := &mockDB{}
			tc.mockBehavior(db)

			r := buildRouter()
			req, err := http.NewRequest("DELETE", "/books/"+tc.bookID, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedCode, rr.Code)

			if tc.expectedSideEffect != "" {
				db.AssertCalled(t, "DeleteBook", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestHealthCheckHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthCheckHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `{"status":"healthy","service":"book-catalog-api"}`, rr.Body.String())
}