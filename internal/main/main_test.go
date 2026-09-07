// internal/main_test.go

package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/rs/zerolog/log"
	"migrated-app/internal/database"
	"migrated-app/internal/model"
	"migrated-app/internal/schemas"
)

type mockDatabase struct {
	mock.Mock
}

func (m *mockDatabase) GetBooks(ctx context.Context, skip int, limit int) ([]model.Book, error) {
	args := m.Called(ctx, skip, limit)
	return args.Get(0).([]model.Book), args.Error(1)
}

func (m *mockDatabase) GetByID(ctx context.Context, id int) (*model.Book, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.Book), args.Error(1)
}

func (m *mockDatabase) Insert(ctx context.Context, book *model.Book) error {
	args := m.Called(ctx, book)
	return args.Error(0)
}

func (m *mockDatabase) Update(ctx context.Context, book *model.Book) error {
	args := m.Called(ctx, book)
	return args.Error(0)
}

func (m *mockDatabase) Delete(ctx context.Context, book *model.Book) error {
	args := m.Called(ctx, book)
	return args.Error(0)
}

func TestBuildRouter(t *testing.T) {
	type args struct {
		path string
		method string
		query string
		body []byte
	}
	tests := []struct {
		name string
		args args
		wantStatusCode int
		wantBody string
	}{
		{
			name: "Root Endpoint",
			args: args{
				path: "/",
				method: "GET",
			},
			wantStatusCode: http.StatusOK,
			wantBody: `{"message":"Welcome to Book Catalog API","version":"1.0.0","docs_url":"/docs"}`,
		},
		{
			name: "Health Check",
			args: args{
				path: "/health",
				method: "GET",
			},
			wantStatusCode: http.StatusOK,
			wantBody: `{"status":"healthy","service":"book-catalog-api"}`,
		},
		{
			name: "List Books",
			args: args{
				path: "/books/",
				method: "GET",
				query: "skip=0&limit=10",
			},
			wantStatusCode: http.StatusOK,
			wantBody: `[]`,
		},
		{
			name: "Get Book - Exists",
			args: args{
				path: "/books/1",
				method: "GET",
			},
			wantStatusCode: http.StatusOK,
			wantBody: `{"id":1,"title":"Sample Book","author":"John Doe","publishedYear":2021,"summary":"A sample book."}`,
		},
		{
			name: "Get Book - Not Found",
			args: args{
				path: "/books/999",
				method: "GET",
			},
			wantStatusCode: http.StatusNotFound,
			wantBody: "",
		},
		{
			name: "Create Book - Success",
			args: args{
				path: "/books/",
				method: "POST",
				body: []byte(`{"title":"New Book","author":"Jane Doe","publishedYear":2022}`),
			},
			wantStatusCode: http.StatusCreated,
			wantBody: `{"id":1,"title":"New Book","author":"Jane Doe","publishedYear":2022,"summary":""}`,
		},
		{
			name: "Create Book - Conflict",
			args: args{
				path: "/books/",
				method: "POST",
				body: []byte(`{"title":"Existing Book","author":"John Doe","publishedYear":2021}`),
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody: "",
		},
		{
			name: "Update Book - Success",
			args: args{
				path: "/books/1",
				method: "PUT",
				body: []byte(`{"title":{"Valid":true,"String":"Updated Book"}}`),
			},
			wantStatusCode: http.StatusOK,
			wantBody: `{"id":1,"title":"Updated Book","author":"John Doe","publishedYear":2021,"summary":"A sample book."}`,
		},
		{
			name: "Delete Book - Success",
			args: args{
				path: "/books/1",
				method: "DELETE",
			},
			wantStatusCode: http.StatusNoContent,
			wantBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := new(mockDatabase)
			ctx := context.Background()
			database.ToContext(ctx, db)

			r := buildRouter()
			req := httptest.NewRequest(tt.args.method, fmt.Sprintf("%s?%s", tt.args.path, tt.args.query), bytes.NewReader(tt.args.body))
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			switch tt.args.method {
			case "GET":
				if tt.args.path == "/books/" {
					db.On("GetBooks", ctx, 0, 10).Return([]model.Book{}, nil)
				} else if tt.args.path == "/books/1" {
					db.On("GetByID", ctx, 1).Return(&model.Book{ID: 1, Title: "Sample Book", Author: "John Doe", PublishedYear: 2021, Summary: sql.NullString{String: "A sample book.", Valid: true}}, nil)
				} else if tt.args.path == "/books/999" {
					db.On("GetByID", ctx, 999).Return(nil, sql.ErrNoRows)
				}
			case "POST":
				var bookCreate schemas.BookCreate
				require.NoError(t, json.Unmarshal(tt.args.body, &bookCreate))
				book := model.Book{Title: bookCreate.Title, Author: bookCreate.Author, PublishedYear: bookCreate.PublishedYear.Int32}
				if tt.name == "Create Book - Success" {
					db.On("Insert", ctx, &book).Return(nil)
				} else if tt.name == "Create Book - Conflict" {
					db.On("Insert", ctx, &book).Return(sql.ErrNoRows)
				}
			case "PUT":
				var bookUpdate schemas.BookUpdate
				require.NoError(t, json.Unmarshal(tt.args.body, &bookUpdate))
				book := &model.Book{ID: 1, Title: "Sample Book", Author: "John Doe", PublishedYear: 2021, Summary: sql.NullString{String: "A sample book.", Valid: true}}
				db.On("GetByID", ctx, 1).Return(book, nil)
				db.On("Update", ctx, book).Return(nil)
			case "DELETE":
				book := &model.Book{ID: 1, Title: "Sample Book", Author: "John Doe", PublishedYear: 2021, Summary: sql.NullString{String: "A sample book.", Valid: true}}
				db.On("GetByID", ctx, 1).Return(book, nil)
				db.On("Delete", ctx, book).Return(nil)
			}

			r.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatusCode, w.Code)
			assert.JSONEq(t, tt.wantBody, w.Body.String())

			db.AssertExpectations(t)
		})
	}
}