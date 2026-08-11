```go
package internal

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── helpers ────────────────────────────────────────────────────────────────

// newMockServer creates a BookServer backed by a sqlmock database.
func newMockServer(t *testing.T) (*BookServer, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	sqlxDB := sqlx.NewDb(db, "postgres")
	return NewBookServer(sqlxDB), mock
}

// newMockRouter creates a fully-wired chi router backed by a sqlmock database.
func newMockRouter(t *testing.T) (http.Handler, sqlmock.Sqlmock) {
	t.Helper()
	srv, mock := newMockServer(t)

	// Re-use buildRouter logic but inject our mock server.
	// We rebuild the router inline so we don't rely on the package-level serverDB.
	r := buildRouterFromServer(srv)
	return r, mock
}

// buildRouterFromServer is a test helper that wires a BookServer into a router
// without touching the package-level serverDB variable.
func buildRouterFromServer(srv *BookServer) http.Handler {
	// We import chi inside the package, so we can call buildRouter after
	// setting serverDB. But to keep tests isolated we wire manually.
	// Instead, we expose the routes through a small local helper.
	//
	// Because buildRouter() reads serverDB, we temporarily set serverDB for
	// the duration of this call.
	savedDB := serverDB
	serverDB = srv.db
	h := buildRouter()
	serverDB = savedDB
	return h
}

// decodeBody is a convenience function that decodes JSON from a response body.
func decodeBody(t *testing.T, body *bytes.Buffer, v interface{}) {
	t.Helper()
	err := json.NewDecoder(body).Decode(v)
	require.NoError(t, err)
}

// pqUniqueViolationErr returns a *pq.Error that represents a unique violation.
func pqUniqueViolationErr() *pq.Error {
	return &pq.Error{Code: "23505", Message: "duplicate key value violates unique constraint"}
}

// ─── unit: parsePaginationParam ─────────────────────────────────────────────

func TestParsePaginationParam(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		def     int
		wantVal int
		wantOK  bool
	}{
		{"empty uses default 0", "", 0, 0, true},
		{"empty uses default 100", "", 100, 100, true},
		{"valid integer", "42", 0, 42, true},
		{"zero string", "0", 0, 0, true},
		{"negative integer", "-5", 0, -5, true},
		{"non-integer returns false", "abc", 0, 0, false},
		{"float string returns false", "3.14", 0, 0, false},
		{"empty with space is non-integer", " ", 0, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, ok := parsePaginationParam(tc.raw, tc.def)
			assert.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				assert.Equal(t, tc.wantVal, val)
			}
		})
	}
}

// ─── unit: isUniqueViolation ─────────────────────────────────────────────────

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"generic error", errors.New("some error"), false},
		{"pq unique violation 23505", pqUniqueViolationErr(), true},
		{"pq other code 23514", &pq.Error{Code: "23514"}, false},
		{"wrapped pq unique violation", fmt.Errorf("wrap: %w", pqUniqueViolationErr()), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isUniqueViolation(tc.err))
		})
	}
}

// ─── GET / ──────────────────────────────────────────────────────────────────

func TestRoot(t *testing.T) {
	tests := []struct {
		name           string
		wantStatus     int
		wantMessage    string
		wantVersion    string
		wantDocsURL    string
	}{
		{
			name:        "GET / returns welcome JSON",
			wantStatus:  http.StatusOK,
			wantMessage: "Welcome to Book Catalog API",
			wantVersion: "1.0.0",
			wantDocsURL: "/docs",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler, mock := newMockRouter(t)
			_ = mock // no DB calls expected

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tc.wantStatus, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

			var body map[string]string
			decodeBody(t, rr.Body, &body)
			assert.Equal(t, tc.wantMessage, body["message"])
			assert.Equal(t, tc.wantVersion, body["version"])
			assert.Equal(t, tc.wantDocsURL, body["docs_url"])
		})
	}
}

// ─── GET /health ─────────────────────────────────────────────────────────────

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		wantStatus  int
		wantStatus2 string
		wantService string
	}{
		{
			name:        "GET /health returns healthy JSON",
			wantStatus:  http.StatusOK,
			wantStatus2: "healthy",
			wantService: "book-catalog-api",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler, mock := newMockRouter(t)
			_ = mock

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tc.wantStatus, rr.Code)

			var body map[string]string
			decodeBody(t, rr.Body, &body)
			assert.Equal(t, tc.wantStatus2, body["status"])
			assert.Equal(t, tc.wantService, body["service"])
		})
	}
}

// ─── GET /books/ ─────────────────────────────────────────────────────────────

func TestListBooks(t *testing.T) {
	bookCols := []string{"id", "title", "author", "published_year", "summary"}

	tests := []struct {
		name       string
		url        string
		setupMock  func(mock sqlmock.Sqlmock)
		wantStatus int
		wantLen    int
		wantDetail string
	}{
		{
			name: "default params returns books",
			url:  "/books/",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(bookCols).
					AddRow(1, "Book A", "Author A", 2020, nil).
					AddRow(2, "Book B", "Author B", 2021, strPtr("A summary"))
				mock.ExpectQuery(`SELECT id, title, author, published_year, summary FROM books`).
					WithArgs(0, 100).
					WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name: "no books returns empty list",
			url:  "/books/",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(bookCols)
				mock.ExpectQuery(`SELECT id, title, author, published_year, summary FROM books`).
					WithArgs(0, 100).
					WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name: "skip and limit applied",
			url:  "/books/?skip=10&limit=5",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(bookCols).
					AddRow(11, "Book K", "Author K", 2019, nil)
				mock.ExpectQuery(`SELECT id, title, author, published_year, summary FROM books`).
					WithArgs(10, 5).
					WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name: "limit greater than 1000 is capped",
			url:  "/books/?limit=9999",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(bookCols)
				mock.ExpectQuery(`SELECT id, title, author, published_year, summary FROM books`).
					WithArgs(0, 1000).
					WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name: "invalid skip returns 422",
			url:  "/books/?skip=abc",
			setupMock: func(mock sqlmock.Sqlmock) {
				// no DB call expected
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantDetail: "skip must be an integer",
		},
		{
			name: "invalid limit returns 422",
			url:  "/books/?limit=xyz",
			setupMock: func(mock sqlmock.Sqlmock) {
				// no DB call expected
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantDetail: "limit must be an integer",
		},
		{
			name: "database error returns 500",
			url:  "/books/",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, title, author, published_year, summary FROM books`).
					WithArgs(0, 100).
					WillReturnError(errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
			wantDetail: "Internal server error while retrieving books",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler, mock := newMockRouter(t)
			tc.setupMock(mock)

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tc.wantStatus, rr.Code)

			if tc.wantDetail != "" {
				var body errorBody
				decodeBody(t, rr.Body, &body)
				assert.Equal(t, tc.wantDetail, body.Detail)
			} else {
				var books []map[string]interface{}
				decodeBody(t, rr.Body, &books)
				assert.Len(t, books, tc.wantLen)
				// Confirm each book has required fields.
				for _, b := range books {
					assert.Contains(t, b, "id")
					assert.Contains(t, b, "title")
					assert.Contains(t, b, "author")
					assert.Contains(t, b, "published_year")
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ─── GET /books/{book_id} ─────────────────────────────────────────────────────

func TestGetBook(t *testing.T) {
	bookCols := []string{"id", "title", "author", "published_year", "summary"}

	tests := []struct {
		name       string
		url        string
		setupMock  func(mock sqlmock.Sqlmock)
		wantStatus int
		wantID     float64
		wantTitle  string
		wantDetail string
	}{
		{
			name: "existing book returns 200",
			url:  "/books/1",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(bookCols).
					AddRow(1, "Go Programming", "Alan Donovan", 2015, strPtr("Great book"))
				mock.ExpectQuery(`SELECT id, title, author, published_year, summary FROM books WHERE id`).
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantID:     1,
			wantTitle:  "Go Programming",
		},
		{
			name: "non-existent book returns 404",
			url:  "/books/999",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, title, author, published_year, summary FROM books WHERE id`).
					WithArgs(int64(999)).
					WillReturnError(sql.ErrNoRows)
			},
			wantStatus: http.StatusNotFound,
			wantDetail: "Book with ID 999 not found",
		},
		{
			name: "database error returns 500",
			url:  "/books/1",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, title, author, published_year, summary FROM books WHERE id`).
					WithArgs(int64(1)).
					WillReturnError(errors.New("connection error"))
			},
			wantStatus: http.StatusInternalServerError,
			wantDetail: "Internal server error while retrieving book",
		},
		{
			name:       "non-integer book_id returns 422",
			url:        "/books/abc",
			setupMock:  func(mock sqlmock.Sqlmock) {},
			wantStatus: http.StatusUnprocessableEntity,
			wantDetail: "book_id must be an integer",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler, mock := newMockRouter(t)
			tc.setupMock(mock)

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tc.wantStatus, rr.Code)