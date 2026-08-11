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

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newTestDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return sqlx.NewDb(db, "sqlmock"), mock
}

func mustJSON(t *testing.T, body string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(body), &m))
	return m
}

func mustJSONArray(t *testing.T, body string) []interface{} {
	t.Helper()
	var a []interface{}
	require.NoError(t, json.Unmarshal([]byte(body), &a))
	return a
}

// bookColumns are the columns returned by SELECT / RETURNING queries.
var bookColumns = []string{"id", "title", "author", "published_year", "summary", "created_at", "updated_at"}

func bookRow(id int64, title, author string, year int, summary string) *sqlmock.Rows {
	now := time.Now()
	nullSummary := sql.NullString{String: summary, Valid: summary != ""}
	_ = nullSummary
	rows := sqlmock.NewRows(bookColumns).AddRow(id, title, author, year, sql.NullString{String: summary, Valid: summary != ""}, now, now)
	return rows
}

// ---------------------------------------------------------------------------
// writeJSON / writeError (unit)
// ---------------------------------------------------------------------------

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       interface{}
		wantStatus int
		wantBody   string
	}{
		{
			name:       "200 with map",
			status:     http.StatusOK,
			body:       map[string]string{"key": "value"},
			wantStatus: http.StatusOK,
			wantBody:   `{"key":"value"}`,
		},
		{
			name:       "201 no body",
			status:     http.StatusCreated,
			body:       nil,
			wantStatus: http.StatusCreated,
			wantBody:   "",
		},
		{
			name:       "500 error string",
			status:     http.StatusInternalServerError,
			body:       "oops",
			wantStatus: http.StatusInternalServerError,
			wantBody:   `"oops"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSON(w, tc.status, tc.body)
			assert.Equal(t, tc.wantStatus, w.Code)
			if tc.wantBody == "" {
				assert.Empty(t, strings.TrimSpace(w.Body.String()))
			} else {
				assert.Contains(t, w.Body.String(), tc.wantBody)
			}
			if tc.body != nil {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		detail     interface{}
		wantStatus int
		wantDetail interface{}
	}{
		{
			name:       "404 string detail",
			status:     http.StatusNotFound,
			detail:     "Book not found",
			wantStatus: http.StatusNotFound,
			wantDetail: "Book not found",
		},
		{
			name:       "400 object detail",
			status:     http.StatusBadRequest,
			detail:     map[string]string{"field": "required"},
			wantStatus: http.StatusBadRequest,
			wantDetail: map[string]interface{}{"field": "required"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeError(w, tc.status, tc.detail)
			assert.Equal(t, tc.wantStatus, w.Code)

			var resp errorBody
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			switch expected := tc.wantDetail.(type) {
			case string:
				assert.Equal(t, expected, resp.Detail)
			default:
				detailBytes, _ := json.Marshal(resp.Detail)
				expectedBytes, _ := json.Marshal(expected)
				assert.JSONEq(t, string(expectedBytes), string(detailBytes))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseIntDefault (unit)
// ---------------------------------------------------------------------------

func TestParseIntDefault(t *testing.T) {
	tests := []struct {
		s    string
		def  int
		want int
	}{
		{"", 0, 0},
		{"", 42, 42},
		{"10", 0, 10},
		{"abc", 5, 5},
		{"-3", 0, -3},
		{"1000", 0, 1000},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("s=%q,def=%d", tc.s, tc.def), func(t *testing.T) {
			got := parseIntDefault(tc.s, tc.def)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// bookFromCreate / applyBookUpdate (unit)
// ---------------------------------------------------------------------------

func TestBookFromCreate(t *testing.T) {
	summary := "A great book"
	tests := []struct {
		name        string
		in          BookCreate
		wantSummary sql.NullString
	}{
		{
			name:        "with summary",
			in:          BookCreate{Title: "T", Author: "A", PublishedYear: 2020, Summary: &summary},
			wantSummary: sql.NullString{String: summary, Valid: true},
		},
		{
			name:        "without summary",
			in:          BookCreate{Title: "T", Author: "A", PublishedYear: 2020, Summary: nil},
			wantSummary: sql.NullString{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := bookFromCreate(&tc.in)
			assert.Equal(t, tc.in.Title, got.Title)
			assert.Equal(t, tc.in.Author, got.Author)
			assert.Equal(t, tc.in.PublishedYear, got.PublishedYear)
			assert.Equal(t, tc.wantSummary, got.Summary)
		})
	}
}

func TestApplyBookUpdate(t *testing.T) {
	newTitle := "New Title"
	newAuthor := "New Author"
	newYear := 2023
	newSummary := "New summary"

	tests := []struct {
		name       string
		initial    Book
		update     BookUpdate
		wantTitle  string
		wantAuthor string
		wantYear   int
	}{
		{
			name:       "update title only",
			initial:    Book{Title: "Old", Author: "Author", PublishedYear: 2000},
			update:     BookUpdate{Title: &newTitle},
			wantTitle:  "New Title",
			wantAuthor: "Author",
			wantYear:   2000,
		},
		{
			name:       "update all fields",
			initial:    Book{Title: "Old", Author: "OldAuthor", PublishedYear: 2000},
			update:     BookUpdate{Title: &newTitle, Author: &newAuthor, PublishedYear: &newYear, Summary: &newSummary},
			wantTitle:  "New Title",
			wantAuthor: "New Author",
			wantYear:   2023,
		},
		{
			name:       "no fields updated",
			initial:    Book{Title: "Old", Author: "Author", PublishedYear: 2000},
			update:     BookUpdate{},
			wantTitle:  "Old",
			wantAuthor: "Author",
			wantYear:   2000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := tc.initial
			applyBookUpdate(&b, &tc.update)
			assert.Equal(t, tc.wantTitle, b.Title)
			assert.Equal(t, tc.wantAuthor, b.Author)
			assert.Equal(t, tc.wantYear, b.PublishedYear)
		})
	}
}

// ---------------------------------------------------------------------------
// Root handler
// ---------------------------------------------------------------------------

func TestRoot(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		wantStatus int
		wantFields map[string]string
	}{
		{
			name:       "GET / returns 200 with API info",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantFields: map[string]string{
				"message":  "Welcome to Book Catalog API",
				"version":  "1.0.0",
				"docs_url": "/docs",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, _ := newTestDB(t)
			h := NewBookHandler(db)

			req := httptest.NewRequest(tc.method, "/", nil)
			w := httptest.NewRecorder()
			h.Root(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			body := mustJSON(t, w.Body.String())
			for k, v := range tc.wantFields {
				assert.Equal(t, v, body[k], "field %q", k)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// HealthCheck handler
// ---------------------------------------------------------------------------

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		wantStatus int
		wantStatus_ string
		wantService string
	}{
		{
			name:        "GET /health returns healthy",
			wantStatus:  http.StatusOK,
			wantStatus_: "healthy",
			wantService: "book-catalog-api",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, _ := newTestDB(t)
			h := NewBookHandler(db)

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()
			h.HealthCheck(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
			body := mustJSON(t, w.Body.String())
			assert.Equal(t, tc.wantStatus_, body["status"])
			assert.Equal(t, tc.wantService, body["service"])
		})
	}
}

// ---------------------------------------------------------------------------
// ListBooks handler
// ---------------------------------------------------------------------------

func TestListBooks(t *testing.T) {
	listQuery := `SELECT id, title, author, published_year, summary, created_at, updated_at FROM books ORDER BY id OFFSET \$1 LIMIT \$2`

	now := time.Now()

	tests := []struct {
		name       string
		query      string
		mockSetup  func(mock sqlmock.Sqlmock)
		wantStatus int
		wantLen    int
		checkBody  func(t *testing.T, body string)
	}{
		{
			name:  "default pagination returns books",
			query: "/books/",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(bookColumns).
					AddRow(1, "Book One", "Author A", 2020, sql.NullString{String: "Summary", Valid: true}, now, now).
					AddRow(2, "Book Two", "Author B", 2021, sql.NullString{}, now, now)
				mock.ExpectQuery(listQuery).WithArgs(0, 100).WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name:  "no books returns empty array",
			query: "/books/",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(bookColumns)
				mock.ExpectQuery(listQuery).WithArgs(0, 100).WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
			checkBody: func(t *testing.T, body string) {
				arr := mustJSONArray(t, body)
				assert.Empty(t, arr)
			},
		},
		{
			name:  "skip and limit provided",
			query: "/books/?skip=5&limit=10",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(bookColumns).
					AddRow(6, "Book Six", "Author F", 2019, sql.NullString{}, now, now)
				mock.ExpectQuery(listQuery).WithArgs(5, 10).WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:  "limit capped at 1000",
			query: "/books/?limit=9999",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(bookColumns)
				mock.ExpectQuery(listQuery).WithArgs(0, 1000).WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:  "negative skip clamped to 0",
			query: "/books/?skip=-10",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(bookColumns)
				mock.ExpectQuery(listQuery).WithArgs(0, 100).WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:  "negative limit clamped to 0",
			query: "/books/?limit=-5",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(bookColumns)
				mock.ExpectQuery(listQuery).WithArgs(0, 0).WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:  "db error returns 500",
			query: "/books/",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery