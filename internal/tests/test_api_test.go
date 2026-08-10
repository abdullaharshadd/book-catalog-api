```go
package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	app "example.com/bookcatalog/internal"
)

// newRecorder returns a fresh ResponseRecorder for capturing HTTP responses.
func newRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

// setupHandler creates a test handler backed by an isolated in-memory / test DB.
// It delegates to app.NewTestClient (defined in internal/conftest.go) which
// returns an http.Handler wired to a freshly-initialised test database.
func setupHandler(t *testing.T) http.Handler {
	t.Helper()
	db, err := app.NewTestDB(t)
	require.NoError(t, err, "NewTestDB should succeed")
	h := app.NewHandlers(db)
	return newRouter(h)
}

// ─── /  (root) ───────────────────────────────────────────────────────────────

func TestReadRoot(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		wantStatus     int
		wantKeys       []string
		wantFieldValues map[string]string
	}{
		{
			name:       "GET request to root path returns 200 with welcome body",
			method:     http.MethodGet,
			path:       "/",
			wantStatus: http.StatusOK,
			wantKeys:   []string{"message", "version", "docs_url"},
			wantFieldValues: map[string]string{
				"message": "Welcome to Book Catalog API",
				"version": "1.0.0",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := setupHandler(t)
			resp, err := doJSON(handler, tc.method, tc.path, nil)
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, resp.Status)

			body, err := resp.DecodeMap()
			require.NoError(t, err)

			for _, key := range tc.wantKeys {
				assert.Contains(t, body, key, "response should contain key %q", key)
			}
			for field, want := range tc.wantFieldValues {
				assert.Equal(t, want, body[field], "field %q should equal %q", field, want)
			}
		})
	}
}

// ─── /health ─────────────────────────────────────────────────────────────────

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		path            string
		wantStatus      int
		wantFieldValues map[string]string
	}{
		{
			name:       "GET /health returns 200 with status=healthy",
			method:     http.MethodGet,
			path:       "/health",
			wantStatus: http.StatusOK,
			wantFieldValues: map[string]string{
				"status":  "healthy",
				"service": "book-catalog-api",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := setupHandler(t)
			resp, err := doJSON(handler, tc.method, tc.path, nil)
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, resp.Status)

			body, err := resp.DecodeMap()
			require.NoError(t, err)

			for field, want := range tc.wantFieldValues {
				assert.Equal(t, want, body[field], "field %q should equal %q", field, want)
			}
		})
	}
}

// ─── POST /books/ ────────────────────────────────────────────────────────────

func TestCreateBook(t *testing.T) {
	tests := []struct {
		name           string
		payload        any
		wantStatus     int
		checkBody      func(t *testing.T, body map[string]any)
		checkDetail    func(t *testing.T, raw []byte)
	}{
		{
			name: "valid book with all fields returns 201 with echoed fields and id",
			payload: map[string]any{
				"title":          "The Go Programming Language",
				"author":         "Alan A. A. Donovan",
				"published_year": 2015,
				"summary":        "Comprehensive guide to Go.",
			},
			wantStatus: http.StatusCreated,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "The Go Programming Language", body["title"])
				assert.Equal(t, "Alan A. A. Donovan", body["author"])
				assert.EqualValues(t, 2015, toInt(body["published_year"]))
				assert.Equal(t, "Comprehensive guide to Go.", body["summary"])
				assert.NotNil(t, body["id"])
			},
		},
		{
			name: "valid book without summary returns 201 with null summary",
			payload: map[string]any{
				"title":          "Concurrency in Go",
				"author":         "Katherine Cox-Buday",
				"published_year": 2017,
			},
			wantStatus: http.StatusCreated,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "Concurrency in Go", body["title"])
				assert.Equal(t, "Katherine Cox-Buday", body["author"])
				assert.EqualValues(t, 2017, toInt(body["published_year"]))
				assert.Nil(t, body["summary"])
				assert.NotNil(t, body["id"])
			},
		},
		{
			name: "missing required fields returns 422",
			payload: map[string]any{
				"title": "Incomplete Book",
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid published_year too early returns 422",
			payload: map[string]any{
				"title":          "Ancient Book",
				"author":         "Old Author",
				"published_year": 999,
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "duplicate title and author returns 400 with already exists detail",
			payload: map[string]any{
				"title":          "Duplicate Book",
				"author":         "Same Author",
				"published_year": 2020,
			},
			wantStatus: http.StatusBadRequest,
			checkDetail: func(t *testing.T, raw []byte) {
				assert.True(t, strings.Contains(strings.ToLower(string(raw)), "already exists"),
					"detail should mention 'already exists', got: %s", raw)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := setupHandler(t)

			// For the duplicate test, pre-create the book first.
			if tc.name == "duplicate title and author returns 400 with already exists detail" {
				pre, err := doJSON(handler, http.MethodPost, "/books/", tc.payload)
				require.NoError(t, err)
				require.Equal(t, http.StatusCreated, pre.Status, "pre-create must succeed")
			}

			resp, err := doJSON(handler, http.MethodPost, "/books/", tc.payload)
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, resp.Status)

			if tc.checkBody != nil {
				body, decErr := resp.DecodeMap()
				require.NoError(t, decErr)
				tc.checkBody(t, body)
			}
			if tc.checkDetail != nil {
				tc.checkDetail(t, resp.Body)
			}
		})
	}
}

func TestCreateBook_SameTitleDifferentAuthors(t *testing.T) {
	handler := setupHandler(t)

	book1 := map[string]any{
		"title":          "Shared Title",
		"author":         "Author One",
		"published_year": 2010,
	}
	book2 := map[string]any{
		"title":          "Shared Title",
		"author":         "Author Two",
		"published_year": 2011,
	}

	resp1, err := doJSON(handler, http.MethodPost, "/books/", book1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp1.Status, "first book should be created")

	resp2, err := doJSON(handler, http.MethodPost, "/books/", book2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp2.Status, "second book with same title but different author should be created")

	b1, err := resp1.DecodeMap()
	require.NoError(t, err)
	b2, err := resp2.DecodeMap()
	require.NoError(t, err)

	id1 := toInt(b1["id"])
	id2 := toInt(b2["id"])
	assert.NotEqual(t, id1, id2, "books should have distinct ids")
}

// ─── GET /books/ ─────────────────────────────────────────────────────────────

func TestGetBooks(t *testing.T) {
	tests := []struct {
		name       string
		seed       []map[string]any
		query      string
		wantStatus int
		checkSlice func(t *testing.T, books []map[string]any)
	}{
		{
			name:       "empty database returns 200 with empty array",
			seed:       nil,
			query:      "",
			wantStatus: http.StatusOK,
			checkSlice: func(t *testing.T, books []map[string]any) {
				assert.Len(t, books, 0)
			},
		},
		{
			name: "books exist returns 200 with all books having ids",
			seed: []map[string]any{
				{"title": "Book A", "author": "Author A", "published_year": 2000},
				{"title": "Book B", "author": "Author B", "published_year": 2001},
			},
			query:      "",
			wantStatus: http.StatusOK,
			checkSlice: func(t *testing.T, books []map[string]any) {
				assert.Len(t, books, 2)
				for _, b := range books {
					assert.NotNil(t, b["id"])
				}
			},
		},
		{
			name: "pagination skip=1 limit=1 returns single middle book",
			seed: []map[string]any{
				{"title": "Paginate A", "author": "PA", "published_year": 2000},
				{"title": "Paginate B", "author": "PB", "published_year": 2001},
				{"title": "Paginate C", "author": "PC", "published_year": 2002},
			},
			query:      "?skip=1&limit=1",
			wantStatus: http.StatusOK,
			checkSlice: func(t *testing.T, books []map[string]any) {
				assert.Len(t, books, 1)
			},
		},
		{
			name: "pagination limit=2 returns at most 2 books",
			seed: []map[string]any{
				{"title": "Limit A", "author": "LA", "published_year": 2000},
				{"title": "Limit B", "author": "LB", "published_year": 2001},
				{"title": "Limit C", "author": "LC", "published_year": 2002},
			},
			query:      "?limit=2",
			wantStatus: http.StatusOK,
			checkSlice: func(t *testing.T, books []map[string]any) {
				assert.LessOrEqual(t, len(books), 2)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := setupHandler(t)

			for _, book := range tc.seed {
				r, err := doJSON(handler, http.MethodPost, "/books/", book)
				require.NoError(t, err)
				require.Equal(t, http.StatusCreated, r.Status, "seed book creation must succeed")
			}

			resp, err := doJSON(handler, http.MethodGet, "/books/"+tc.query, nil)
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, resp.Status)

			books, decErr := resp.DecodeSlice()
			require.NoError(t, decErr)

			if tc.checkSlice != nil {
				tc.checkSlice(t, books)
			}
		})
	}
}

// ─── GET /books/{book_id} ────────────────────────────────────────────────────

func TestGetBookByID(t *testing.T) {
	tests := []struct {
		name       string
		bookID     func(created map[string]any) string
		wantStatus int
		checkBody  func(t *testing.T, created map[string]any, resp map[string]any)
		checkRaw   func(t *testing.T, raw []byte)
	}{
		{
			name: "existing book returns 200 with full representation",
			bookID: func(created map[string]any) string {
				return fmt.Sprintf("%.0f", created["id"].(float64))
			},
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, created map[string]any, resp map[string]any) {
				assert.Equal(t, created["id"], resp["id"])
				assert.Equal(t, created["title"], resp["title"])
				assert.Equal(t, created["author"], resp["author"])
			},
		},
		{
			name: "non-existent id returns 404 with not found detail",
			bookID: func(_ map[string]any) string {
				return "999999"
			},
			wantStatus: http.StatusNotFound,
			checkRaw: func(t *testing.T, raw []byte) {
				assert.True(t, strings.Contains(strings.ToLower(string(raw)), "not found"),
					"detail should mention 'not found', got: %s", raw)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := setupHandler(t)

			// Seed one book.
			seed := map[string]any{
				"title":          "Test Book",
				"author":         "Test Author",
				"published_year": 2020,
				"summary":        "A test book.",
			}
			createResp, err := doJSON(handler, http.MethodPost, "/books/", seed)
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, createResp.Status)

			created, err := createResp.DecodeMap()
			require.NoError(t, err)

			bookID := tc.bookID(created)
			resp, err := doJSON(handler, http.MethodGet, "/books/"+bookID, nil)
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, resp.Status)

			if tc.checkBody != nil {
				body, decErr := resp.DecodeMap()
				require.NoError(t, decErr)