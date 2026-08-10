```go
package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiern/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Fake / stub types
// ---------------------------------------------------------------------------

// fakeDB is a minimal stand-in for *DB used in handler tests.
// Instead of a real postgres connection we embed a sqlx.DB backed by a
// sqlmock or, simpler for unit tests, we define an interface over the DB
// operations the handlers call and swap in a fake implementation.

// dbQuerier is the interface the Handlers actually need from *DB.
// We define it here so we can replace it in tests without touching production
// code.  The real *DB satisfies this interface through its SQL() method.
//
// Because the production code calls h.db.SQL().{SelectContext, GetContext,
// QueryRowxContext, ExecContext} we build a thin mock around sqlx.DB.
// For these unit tests we use a wrapper that delegates through a
// sqlx.DB/sqlmock pair so we can drive every path.

// Rather than pulling in sqlmock, we test at the HTTP layer using a real
// chi router wired to stub Handlers whose DB methods are replaced via
// a helper that injects a pre-built sqlx.DB from an in-memory sqlite
// driver or, more practically, by creating a fakeDB type that satisfies the
// same shape as *DB.
//
// The cleanest approach that requires no extra dependencies: we extract the
// exact set of sqlx operations into a small interface (dbOps), have *DB
// delegate to it, and replace it in tests.  However, since we cannot modify
// production code here, we take the slightly different path of building a
// thin fake that wraps database/sql/driver stubs.

// We use "database/sql/driver" stubs via the stdlib "database/sql" register
// so we can create a *sql.DB without a live server and wrap it in sqlx.

// fakeDriver / fakeConn / fakeStmt / fakeRows implement driver.Driver etc.
// for very lightweight in-process "tables".

// For simplicity, tests that exercise DB-error paths use a driver that
// always returns errors, and tests that exercise success paths return
// hard-coded rows.

// ---------------------------------------------------------------------------
// Minimal stub models (mirrors what schemas.go / models.go expose)
// ---------------------------------------------------------------------------

// If the real types live in the same package we can use them directly.
// We assume the following types exist (as described in the migrated file):
//   Book, BookCreate, BookUpdate, BookResponse, NewBookResponse

// ---------------------------------------------------------------------------
// Helper: create a chi router with a single handler attached under a given
// pattern, bypassing buildRouter so we can inject a fake *DB.
// ---------------------------------------------------------------------------

func routerWith(method, pattern string, h http.HandlerFunc) http.Handler {
	r := chi.NewRouter()
	switch method {
	case http.MethodGet:
		r.Get(pattern, h)
	case http.MethodPost:
		r.Post(pattern, h)
	case http.MethodPut:
		r.Put(pattern, h)
	case http.MethodDelete:
		r.Delete(pattern, h)
	}
	return r
}

// ---------------------------------------------------------------------------
// Helpers to decode JSON bodies in test responses
// ---------------------------------------------------------------------------

func decodeJSON(t *testing.T, body string, v interface{}) {
	t.Helper()
	require.NoError(t, json.NewDecoder(strings.NewReader(body)).Decode(v))
}

// ---------------------------------------------------------------------------
// writeError / writeJSON unit tests
// ---------------------------------------------------------------------------

func TestWriteError(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		detail         string
		wantStatus     int
		wantDetail     string
		wantCT         string
	}{
		{
			name:       "404 not found detail",
			status:     http.StatusNotFound,
			detail:     "Book with ID 99 not found",
			wantStatus: http.StatusNotFound,
			wantDetail: "Book with ID 99 not found",
			wantCT:     "application/json",
		},
		{
			name:       "400 bad request detail",
			status:     http.StatusBadRequest,
			detail:     "Invalid request body",
			wantStatus: http.StatusBadRequest,
			wantDetail: "Invalid request body",
			wantCT:     "application/json",
		},
		{
			name:       "500 internal server error",
			status:     http.StatusInternalServerError,
			detail:     "Internal server error",
			wantStatus: http.StatusInternalServerError,
			wantDetail: "Internal server error",
			wantCT:     "application/json",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeError(w, tc.status, tc.detail)

			res := w.Result()
			assert.Equal(t, tc.wantStatus, res.StatusCode)
			assert.Equal(t, tc.wantCT, res.Header.Get("Content-Type"))

			var resp errorResponse
			decodeJSON(t, w.Body.String(), &resp)
			assert.Equal(t, tc.wantDetail, resp.Detail)
		})
	}
}

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		payload    interface{}
		wantStatus int
		wantBody   string
	}{
		{
			name:       "200 with map payload",
			status:     http.StatusOK,
			payload:    map[string]string{"key": "value"},
			wantStatus: http.StatusOK,
			wantBody:   `{"key":"value"}`,
		},
		{
			name:       "204 with nil payload writes no body",
			status:     http.StatusNoContent,
			payload:    nil,
			wantStatus: http.StatusNoContent,
			wantBody:   "",
		},
		{
			name:       "201 with struct payload",
			status:     http.StatusCreated,
			payload:    errorResponse{Detail: "created"},
			wantStatus: http.StatusCreated,
			wantBody:   `{"detail":"created"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSON(w, tc.status, tc.payload)

			res := w.Result()
			assert.Equal(t, tc.wantStatus, res.StatusCode)
			if tc.wantBody != "" {
				assert.Contains(t, strings.TrimSpace(w.Body.String()), strings.TrimSpace(tc.wantBody))
			} else {
				assert.Empty(t, strings.TrimSpace(w.Body.String()))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// isUniqueViolation tests
// ---------------------------------------------------------------------------

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error returns false",
			err:  nil,
			want: false,
		},
		{
			name: "non-pq error returns false",
			err:  fmt.Errorf("some other error"),
			want: false,
		},
		{
			name: "sql.ErrNoRows returns false",
			err:  sql.ErrNoRows,
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isUniqueViolation(tc.err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// parseIntDefault tests
// ---------------------------------------------------------------------------

func TestParseIntDefault(t *testing.T) {
	tests := []struct {
		name string
		s    string
		def  int
		want int
	}{
		{"empty string returns default", "", 42, 42},
		{"valid integer", "10", 0, 10},
		{"invalid string returns default", "abc", 5, 5},
		{"negative value", "-3", 0, -3},
		{"zero", "0", 99, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseIntDefault(tc.s, tc.def)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// join tests
// ---------------------------------------------------------------------------

func TestJoin(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		sep   string
		want  string
	}{
		{"empty slice", []string{}, ", ", ""},
		{"single element", []string{"a"}, ", ", "a"},
		{"two elements", []string{"a", "b"}, ", ", "a, b"},
		{"three elements", []string{"a", "b", "c"}, " AND ", "a AND b AND c"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := join(tc.parts, tc.sep)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// parseBookID tests (via httptest)
// ---------------------------------------------------------------------------

func TestParseBookID(t *testing.T) {
	tests := []struct {
		name       string
		rawID      string
		wantOK     bool
		wantID     int64
		wantStatus int
		wantDetail string
	}{
		{
			name:   "valid integer ID",
			rawID:  "42",
			wantOK: true,
			wantID: 42,
		},
		{
			name:       "non-numeric ID",
			rawID:      "abc",
			wantOK:     false,
			wantStatus: http.StatusNotFound,
			wantDetail: "Book with ID abc not found",
		},
		{
			name:       "float ID",
			rawID:      "1.5",
			wantOK:     false,
			wantStatus: http.StatusNotFound,
			wantDetail: "Book with ID 1.5 not found",
		},
		{
			name:   "large valid ID",
			rawID:  "9999999",
			wantOK: true,
			wantID: 9999999,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Build a chi router that captures the url param and calls parseBookID.
			var gotID int64
			var gotOK bool
			r := chi.NewRouter()
			r.Get("/books/{book_id}", func(w http.ResponseWriter, req *http.Request) {
				gotID, gotOK = parseBookID(w, req)
				if gotOK {
					w.WriteHeader(http.StatusOK)
				}
			})

			req := httptest.NewRequest(http.MethodGet, "/books/"+tc.rawID, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.wantOK, gotOK)
			if tc.wantOK {
				assert.Equal(t, tc.wantID, gotID)
			} else {
				assert.Equal(t, tc.wantStatus, w.Code)
				var resp errorResponse
				decodeJSON(t, w.Body.String(), &resp)
				assert.Equal(t, tc.wantDetail, resp.Detail)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Root handler tests
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
			name:        "GET / returns welcome payload",
			wantStatus:  http.StatusOK,
			wantMessage: "Welcome to Book Catalog API",
			wantVersion: "1.0.0",
			wantDocsURL: "/docs",
		},
	}

	h := &Handlers{db: nil} // Root doesn't touch the DB

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			h.Root(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var body map[string]string
			decodeJSON(t, w.Body.String(), &body)
			assert.Equal(t, tc.wantMessage, body["message"])
			assert.Equal(t, tc.wantVersion, body["version"])
			assert.Equal(t, tc.wantDocsURL, body["docs_url"])
		})
	}
}

// ---------------------------------------------------------------------------
// HealthCheck handler tests
// ---------------------------------------------------------------------------

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		wantStatus  int
		wantStatus2 string
		wantService string
	}{
		{
			name:        "GET /health returns healthy payload",
			wantStatus:  http.StatusOK,
			wantStatus2: "healthy",
			wantService: "book-catalog-api",
		},
	}

	h := &Handlers{db: nil}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			h.HealthCheck(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var body map[string]string
			decodeJSON(t, w.Body.String(), &body)
			assert.Equal(t, tc.wantStatus2, body["status"])
			assert.Equal(t, tc.wantService, body["service"])
		})
	}
}

// ---------------------------------------------------------------------------
// DB-backed handler tests
//
// We create a real sqlx.DB backed by sqlmock so we can drive every code path
// without a live Postgres server.
// ---------------------------------------------------------------------------

// newSqlxDB is a helper that wires sqlmock into a *sqlx.DB.
// We keep it here rather than importing a separate package.

// Because importing github.com/DATA-DOG/go-sqlmock in the test file requires
// it to be in go.mod, and the prompt asks us to assume the project already
// has testify, we provide a lightweight alternative: we register a custom
// database/sql driver for each test scenario.
//
// Pattern: a driverFunc is registered once under a unique name and returns
// a connection that either succeeds or fails as needed.

import (
	"database/sql/driver"
	"io"
	"sync"
)

// ---------------------------------------------------------------------------
// Minimal fake sql driver infrastructure
// ---------------------------------------------------------------------------

var (
	mu           sync.Mutex
	driverSeq    int
	registeredDrivers = map[string]bool{}
)

type funcDriver struct {
	open func(name string) (driver.Conn, error)
}

func (d *funcDriver) Open(name string) (driver.Conn, error) { return d.open(name) }

type funcConn struct {
	prepare func