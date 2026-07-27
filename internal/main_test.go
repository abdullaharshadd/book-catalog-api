```go
package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock bookStore
// ---------------------------------------------------------------------------

type mockStore struct {
	listFn   func(ctx context.Context, skip, limit int) ([]Book, error)
	getFn    func(ctx context.Context, id int64) (Book, bool, error)
	createFn func(ctx context.Context, b *Book) error
	updateFn func(ctx context.Context, b *Book) error
	deleteFn func(ctx context.Context, id int64) (bool, error)
}

func (m *mockStore) List(ctx context.Context, skip, limit int) ([]Book, error) {
	if m.listFn != nil {
		return m.listFn(ctx, skip, limit)
	}
	return nil, errors.New("listFn not set")
}

func (m *mockStore) Get(ctx context.Context, id int64) (Book, bool, error) {
	if m.getFn != nil {
		return m.getFn(ctx, id)
	}
	return Book{}, false, errors.New("getFn not set")
}

func (m *mockStore) Create(ctx context.Context, b *Book) error {
	if m.createFn != nil {
		return m.createFn(ctx, b)
	}
	return errors.New("createFn not set")
}

func (m *mockStore) Update(ctx context.Context, b *Book) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, b)
	}
	return errors.New("updateFn not set")
}

func (m *mockStore) Delete(ctx context.Context, id int64) (bool, error) {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return false, errors.New("deleteFn not set")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newRouter wires a BookHandler backed by the given store into a chi router,
// matching the routes defined in buildRouter.
func newRouter(store bookStore) http.Handler {
	h := NewBookHandler(store)
	r := chi.NewRouter()
	r.Get("/", h.Root)
	r.Get("/health", h.HealthCheck)
	r.Route("/books", func(r chi.Router) {
		r.Get("/", h.ListBooks)
		r.Post("/", h.CreateBook)
		r.Get("/{book_id}", h.GetBook)
		r.Put("/{book_id}", h.UpdateBook)
		r.Delete("/{book_id}", h.DeleteBook)
	})
	return r
}

func decodeBody(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &m))
	return m
}

func decodeArray(t *testing.T, body []byte) []interface{} {
	t.Helper()
	var arr []interface{}
	require.NoError(t, json.Unmarshal(body, &arr))
	return arr
}

func jsonBody(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

// ---------------------------------------------------------------------------
// Root
// ---------------------------------------------------------------------------

func TestRoot(t *testing.T) {
	tests := []struct {
		name           string
		wantStatus     int
		wantMessage    string
		wantVersion    string
		wantDocsURL    string
	}{
		{
			name:        "returns welcome payload",
			wantStatus:  http.StatusOK,
			wantMessage: "Welcome to Book Catalog API",
			wantVersion: "1.0.0",
			wantDocsURL: "/docs",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := newRouter(&mockStore{})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.wantStatus, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

			body := decodeBody(t, rr.Body.Bytes())
			assert.Equal(t, tc.wantMessage, body["message"])
			assert.Equal(t, tc.wantVersion, body["version"])
			assert.Equal(t, tc.wantDocsURL, body["docs_url"])
		})
	}
}

// ---------------------------------------------------------------------------
// HealthCheck
// ---------------------------------------------------------------------------

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		wantStatus  int
		wantStatus2 string
		wantService string
	}{
		{
			name:        "returns healthy payload",
			wantStatus:  http.StatusOK,
			wantStatus2: "healthy",
			wantService: "book-catalog-api",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := newRouter(&mockStore{})
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.wantStatus, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

			body := decodeBody(t, rr.Body.Bytes())
			assert.Equal(t, tc.wantStatus2, body["status"])
			assert.Equal(t, tc.wantService, body["service"])
		})
	}
}

// ---------------------------------------------------------------------------
// ListBooks
// ---------------------------------------------------------------------------

func TestListBooks(t *testing.T) {
	sampleBooks := []Book{
		{ID: 1, Title: "Go Programming", Author: "John", PublishedYear: 2020},
		{ID: 2, Title: "Advanced Go", Author: "Jane", PublishedYear: 2021},
	}

	tests := []struct {
		name       string
		query      string
		listFn     func(ctx context.Context, skip, limit int) ([]Book, error)
		wantStatus int
		wantLen    *int
		wantDetail string
		checkArgs  func(t *testing.T, skip, limit int)
	}{
		{
			name:  "default pagination returns books",
			query: "",
			listFn: func(ctx context.Context, skip, limit int) ([]Book, error) {
				assert.Equal(t, 0, skip)
				assert.Equal(t, 100, limit)
				return sampleBooks, nil
			},
			wantStatus: http.StatusOK,
			wantLen:    intPtr(2),
		},
		{
			name:  "no books returns empty array",
			query: "",
			listFn: func(ctx context.Context, skip, limit int) ([]Book, error) {
				return []Book{}, nil
			},
			wantStatus: http.StatusOK,
			wantLen:    intPtr(0),
		},
		{
			name:  "limit capped at 1000",
			query: "?limit=5000",
			listFn: func(ctx context.Context, skip, limit int) ([]Book, error) {
				assert.Equal(t, 1000, limit)
				return []Book{}, nil
			},
			wantStatus: http.StatusOK,
			wantLen:    intPtr(0),
		},
		{
			name:  "skip and limit applied",
			query: "?skip=5&limit=10",
			listFn: func(ctx context.Context, skip, limit int) ([]Book, error) {
				assert.Equal(t, 5, skip)
				assert.Equal(t, 10, limit)
				return sampleBooks, nil
			},
			wantStatus: http.StatusOK,
			wantLen:    intPtr(2),
		},
		{
			name:  "negative skip treated as 0",
			query: "?skip=-5",
			listFn: func(ctx context.Context, skip, limit int) ([]Book, error) {
				assert.Equal(t, 0, skip)
				return []Book{}, nil
			},
			wantStatus: http.StatusOK,
			wantLen:    intPtr(0),
		},
		{
			name:  "negative limit treated as 0",
			query: "?limit=-1",
			listFn: func(ctx context.Context, skip, limit int) ([]Book, error) {
				assert.Equal(t, 0, limit)
				return []Book{}, nil
			},
			wantStatus: http.StatusOK,
			wantLen:    intPtr(0),
		},
		{
			name:  "database error returns 500",
			query: "",
			listFn: func(ctx context.Context, skip, limit int) ([]Book, error) {
				return nil, errors.New("db connection lost")
			},
			wantStatus: http.StatusInternalServerError,
			wantDetail: "Internal server error while retrieving books",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &mockStore{listFn: tc.listFn}
			router := newRouter(store)

			req := httptest.NewRequest(http.MethodGet, "/books/"+tc.query, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.wantStatus, rr.Code)

			if tc.wantLen != nil {
				arr := decodeArray(t, rr.Body.Bytes())
				assert.Len(t, arr, *tc.wantLen)
			}
			if tc.wantDetail != "" {
				body := decodeBody(t, rr.Body.Bytes())
				assert.Equal(t, tc.wantDetail, body["detail"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetBook
// ---------------------------------------------------------------------------

func TestGetBook(t *testing.T) {
	sampleBook := Book{ID: 42, Title: "The Go Book", Author: "Rob Pike", PublishedYear: 2019}

	tests := []struct {
		name       string
		bookID     string
		getFn      func(ctx context.Context, id int64) (Book, bool, error)
		wantStatus int
		wantDetail string
		wantID     *float64
	}{
		{
			name:   "book found returns 200",
			bookID: "42",
			getFn: func(ctx context.Context, id int64) (Book, bool, error) {
				assert.Equal(t, int64(42), id)
				return sampleBook, true, nil
			},
			wantStatus: http.StatusOK,
			wantID:     float64Ptr(42),
		},
		{
			name:   "book not found returns 404",
			bookID: "99",
			getFn: func(ctx context.Context, id int64) (Book, bool, error) {
				return Book{}, false, nil
			},
			wantStatus: http.StatusNotFound,
			wantDetail: "Book with ID 99 not found",
		},
		{
			name:       "non-integer book_id returns 404",
			bookID:     "abc",
			wantStatus: http.StatusNotFound,
			wantDetail: "Book with ID abc not found",
		},
		{
			name:   "database error returns 500",
			bookID: "1",
			getFn: func(ctx context.Context, id int64) (Book, bool, error) {
				return Book{}, false, errors.New("timeout")
			},
			wantStatus: http.StatusInternalServerError,
			wantDetail: "Internal server error while retrieving book",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &mockStore{getFn: tc.getFn}
			router := newRouter(store)

			req := httptest.NewRequest(http.MethodGet, "/books/"+tc.bookID, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.wantStatus, rr.Code)

			body := decodeBody(t, rr.Body.Bytes())
			if tc.wantDetail != "" {
				assert.Equal(t, tc.wantDetail, body["detail"])
			}
			if tc.wantID != nil {
				assert.Equal(t, *tc.wantID, body["id"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CreateBook
// ---------------------------------------------------------------------------

func TestCreateBook(t *testing.T) {
	tests := []struct {
		name       string
		payload    interface{}
		createFn   func(ctx context.Context, b *Book) error
		wantStatus int
		wantDetail string
		checkBody  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "valid book created returns 201",
			payload: map[string]interface{}{
				"title":          "Clean Code",
				"author":         "Robert Martin",
				"published_year": 2008,
			},
			createFn: func(ctx context.Context, b *Book) error {
				b.ID = 10
				return nil
			},
			wantStatus: http.StatusCreated,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, float64(10), body["id"])
				assert.Equal(t, "Clean Code", body["title"])
				assert.Equal(t, "Robert Martin", body["author"])
			},
		},
		{
			name: "duplicate book returns 400",
			payload: map[string]interface{}{
				"title":          "Duplicate",
				"author":         "Author",
				"published_year": 2000,
			},
			createFn: func(ctx context.Context, b *Book) error {
				return errDuplicate
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "Book with this title and author already exists",
		},
		{
			name:       "missing required fields returns 422",
			payload:    map[string]interface{}{"title": "No Author"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid JSON body returns 422",
			payload:    "not-json{{{{",
			wantStatus: http.StatusUnprocessableEntity,
			wantDetail: "invalid request body",
		},
		{
			name: "unexpected database error returns 500",
			payload: map[string]interface{}{
				"title":          "Some Book",
				"author":         "Some Author",
				"published_year": 2010,
			},
			createFn: func(ctx context.Context, b *Book) error {
				return errors.New("unexpected db error")
			},
			wantStatus: http.StatusInternalServerError,
			wantDetail: "Internal server error while creating book",
		},
		{
			name: "book with summary created returns 201",
			payload: map[string]interface{}{
				"title":          "Go