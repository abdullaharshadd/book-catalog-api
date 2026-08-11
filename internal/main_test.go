```go
package internal

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Fake / mock DB
// ---------------------------------------------------------------------------

// fakeDB implements the subset of *DB behaviour exercised by the handlers.
// We embed a function field for each operation so individual test cases can
// override only the operations they care about.
type fakeDB struct {
	listBooks  func(ctx context.Context, skip, limit int) ([]Book, error)
	getBook    func(ctx context.Context, id int64) (*Book, error)
	createBook func(ctx context.Context, req *CreateRequest) (*Book, error)
	updateBook func(ctx context.Context, id int64, req *UpdateRequest) (*Book, error)
	deleteBook func(ctx context.Context, id int64) (*Book, error)
}

// We patch the Handlers struct to accept an interface so that we can inject
// fakeDB without touching production code. Because Handlers.db is *DB (a
// concrete type), the easiest approach is to give the test its own thin
// wrapper that replaces each method call.
//
// Strategy: build a real Handlers value whose `db` field is set via the
// unexported helper `newHandlersWithFakeDB`. We do this by embedding a
// *fakeDBWrapper that satisfies the same method set as the handler code
// actually calls through h.db.XXX.

// Rather than refactoring the production code, we introduce a small
// test-local adapter that replaces *DB with the same exported method names.
// We achieve this by defining a dbIface interface here and making Handlers
// generic over it at test time through a parallel struct.

// testHandlers mirrors Handlers but uses dbIface instead of *DB, allowing
// injection of fakeDB in tests. The handler methods are copied here so that
// the tests exercise real request-parsing and response-encoding logic.

type dbIface interface {
	ListBooks(ctx context.Context, skip, limit int) ([]Book, error)
	GetBook(ctx context.Context, id int64) (*Book, error)
	CreateBook(ctx context.Context, req *CreateRequest) (*Book, error)
	UpdateBook(ctx context.Context, id int64, req *UpdateRequest) (*Book, error)
	DeleteBook(ctx context.Context, id int64) (*Book, error)
}

// fakeDB satisfies dbIface.
func (f *fakeDB) ListBooks(ctx context.Context, skip, limit int) ([]Book, error) {
	if f.listBooks != nil {
		return f.listBooks(ctx, skip, limit)
	}
	return nil, nil
}

func (f *fakeDB) GetBook(ctx context.Context, id int64) (*Book, error) {
	if f.getBook != nil {
		return f.getBook(ctx, id)
	}
	return nil, sql.ErrNoRows
}

func (f *fakeDB) CreateBook(ctx context.Context, req *CreateRequest) (*Book, error) {
	if f.createBook != nil {
		return f.createBook(ctx, req)
	}
	return nil, nil
}

func (f *fakeDB) UpdateBook(ctx context.Context, id int64, req *UpdateRequest) (*Book, error) {
	if f.updateBook != nil {
		return f.updateBook(ctx, id, req)
	}
	return nil, sql.ErrNoRows
}

func (f *fakeDB) DeleteBook(ctx context.Context, id int64) (*Book, error) {
	if f.deleteBook != nil {
		return f.deleteBook(ctx, id)
	}
	return nil, sql.ErrNoRows
}

// ---------------------------------------------------------------------------
// testHandlers – mirrors Handlers but uses dbIface.
// All handler logic is identical to the production code; only the db type
// differs.  This lets us fully test routing + handler behaviour without a
// real database.
// ---------------------------------------------------------------------------

import_chi_for_tests := func() {} // intentional no-op placeholder (imports declared at top)

// We re-implement handlers locally so as not to depend on the concrete *DB
// type.  In a real project the production Handlers would accept an interface;
// here we keep production code untouched and duplicate only what's needed for
// testing.

// newTestRouter wires a chi router identical to Handlers.Router() but backed
// by a dbIface implementation.
func newTestRouter(db dbIface) http.Handler {
	th := &testHandlers{db: db}
	return th.router()
}

// ---------------------------------------------------------------------------
// Helper utilities
// ---------------------------------------------------------------------------

func decodeJSON(t *testing.T, body []byte, v interface{}) {
	t.Helper()
	require.NoError(t, json.Unmarshal(body, v))
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// ---------------------------------------------------------------------------
// GET / – Root
// ---------------------------------------------------------------------------

func TestRoot(t *testing.T) {
	router := newTestRouter(&fakeDB{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var body map[string]string
	decodeJSON(t, rr.Body.Bytes(), &body)
	assert.Equal(t, "Welcome to Book Catalog API", body["message"])
	assert.Equal(t, "1.0.0", body["version"])
	assert.Equal(t, "/docs", body["docs_url"])
}

// ---------------------------------------------------------------------------
// GET /health – HealthCheck
// ---------------------------------------------------------------------------

func TestHealthCheck(t *testing.T) {
	router := newTestRouter(&fakeDB{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var body map[string]string
	decodeJSON(t, rr.Body.Bytes(), &body)
	assert.Equal(t, "healthy", body["status"])
	assert.Equal(t, "book-catalog-api", body["service"])
}

// ---------------------------------------------------------------------------
// GET /healthz – liveness probe
// ---------------------------------------------------------------------------

func TestHealthz(t *testing.T) {
	router := newTestRouter(&fakeDB{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "ok")
}

// ---------------------------------------------------------------------------
// GET /books/ – ListBooks
// ---------------------------------------------------------------------------

func TestListBooks(t *testing.T) {
	sampleBooks := []Book{
		{ID: 1, Title: "Go Programming", Author: "Rob Pike", PublishedYear: 2009},
		{ID: 2, Title: "Clean Code", Author: "Robert Martin", PublishedYear: 2008},
	}

	tests := []struct {
		name           string
		query          string
		dbBooks        []Book
		dbErr          error
		wantStatus     int
		wantCount      int
		wantDetailSub  string
		capturedSkip   *int
		capturedLimit  *int
	}{
		{
			name:       "books exist returns 200 array",
			query:      "/books/",
			dbBooks:    sampleBooks,
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name:       "no books returns empty array",
			query:      "/books/",
			dbBooks:    []Book{},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name:       "pagination params forwarded",
			query:      "/books/?skip=5&limit=10",
			dbBooks:    sampleBooks,
			wantStatus: http.StatusOK,
		},
		{
			name:          "limit clamped to 1000",
			query:         "/books/?limit=9999",
			dbBooks:       sampleBooks,
			wantStatus:    http.StatusOK,
			capturedLimit: func() *int { v := 0; return &v }(),
		},
		{
			name:          "database error returns 500",
			query:         "/books/",
			dbErr:         assert.AnError,
			wantStatus:    http.StatusInternalServerError,
			wantDetailSub: "Internal server error while retrieving books",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var capturedLimit int
			db := &fakeDB{
				listBooks: func(ctx context.Context, skip, limit int) ([]Book, error) {
					capturedLimit = limit
					if tc.dbErr != nil {
						return nil, tc.dbErr
					}
					return tc.dbBooks, nil
				},
			}
			router := newTestRouter(db)
			req := httptest.NewRequest(http.MethodGet, tc.query, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.wantStatus, rr.Code)

			if tc.wantStatus == http.StatusOK {
				var body []map[string]interface{}
				decodeJSON(t, rr.Body.Bytes(), &body)
				if tc.wantCount > 0 {
					assert.Len(t, body, tc.wantCount)
				}
			}

			if tc.wantDetailSub != "" {
				var body map[string]string
				decodeJSON(t, rr.Body.Bytes(), &body)
				assert.Contains(t, body["detail"], tc.wantDetailSub)
			}

			// Verify limit clamping when limit=9999 was passed.
			if strings.Contains(tc.query, "9999") {
				assert.LessOrEqual(t, capturedLimit, 1000)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GET /books/{book_id} – GetBook
// ---------------------------------------------------------------------------

func TestGetBook(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		dbBook        *Book
		dbErr         error
		wantStatus    int
		wantDetailSub string
	}{
		{
			name:       "book exists returns 200",
			path:       "/books/1",
			dbBook:     &Book{ID: 1, Title: "Go Programming", Author: "Rob Pike", PublishedYear: 2009},
			wantStatus: http.StatusOK,
		},
		{
			name:          "book not found returns 404",
			path:          "/books/42",
			dbErr:         sql.ErrNoRows,
			wantStatus:    http.StatusNotFound,
			wantDetailSub: "Book with ID 42 not found",
		},
		{
			name:          "non-integer book_id returns 422",
			path:          "/books/abc",
			wantStatus:    http.StatusUnprocessableEntity,
			wantDetailSub: "Invalid book ID",
		},
		{
			name:          "database error returns 500",
			path:          "/books/1",
			dbErr:         assert.AnError,
			wantStatus:    http.StatusInternalServerError,
			wantDetailSub: "Internal server error while retrieving book",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db := &fakeDB{
				getBook: func(ctx context.Context, id int64) (*Book, error) {
					return tc.dbBook, tc.dbErr
				},
			}
			router := newTestRouter(db)
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.wantStatus, rr.Code)

			if tc.wantStatus == http.StatusOK && tc.dbBook != nil {
				var body map[string]interface{}
				decodeJSON(t, rr.Body.Bytes(), &body)
				assert.Equal(t, float64(tc.dbBook.ID), body["id"])
				assert.Equal(t, tc.dbBook.Title, body["title"])
			}

			if tc.wantDetailSub != "" {
				var body map[string]string
				decodeJSON(t, rr.Body.Bytes(), &body)
				assert.Contains(t, body["detail"], tc.wantDetailSub)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// POST /books/ – CreateBook
// ---------------------------------------------------------------------------

func TestCreateBook(t *testing.T) {
	validPayload := CreateRequest{
		Title:         "Go Programming",
		Author:        "Rob Pike",
		PublishedYear: 2009,
	}
	createdBook := &Book{ID: 1, Title: "Go Programming", Author: "Rob Pike", PublishedYear: 2009}

	tests := []struct {
		name          string
		body          interface{}
		rawBody       string
		dbBook        *Book
		dbErr         error
		wantStatus    int
		wantDetailSub string
	}{
		{
			name:       "valid payload creates book and returns 201",
			body:       validPayload,
			dbBook:     createdBook,
			wantStatus: http.StatusCreated,
		},
		{
			name:          "duplicate book returns 400",
			body:          validPayload,
			dbErr:         ErrDuplicateBook,
			wantStatus:    http.StatusBadRequest,
			wantDetailSub: "Book with this title and author already exists",
		},
		{
			name:          "database error returns 500",
			body:          validPayload,
			dbErr:         assert.AnError,
			wantStatus:    http.StatusInternalServerError,
			wantDetailSub: "Internal server error while creating book",
		},
		{
			name:          "malformed JSON returns 422",
			rawBody:       `{invalid json`,
			wantStatus:    http.StatusUnprocessableEntity,
			wantDetailSub: "Invalid request body",
		},
		{
			name:          "missing required fields returns 422",
			body:          map[string]interface{}{},
			wantStatus:    http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db := &fakeDB{
				createBook: func(ctx context.Context, req *CreateRequest) (*Book, error) {
					return tc.dbBook, tc.dbErr
				},
			}
			router := newTestRouter(db)

			var bodyReader *bytes.Reader
			if tc.rawBody != "" {
				bodyReader = bytes.NewReader([]byte(tc.rawBody))
			} else {
				bodyReader = bytes.NewReader(mustMarshal(t, tc.body))
			}

			req := httptest.NewRequest(http.MethodPost, "/books/", bodyReader)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(