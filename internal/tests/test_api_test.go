```go
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Shared test helpers used across all table-driven tests
// ---------------------------------------------------------------------------

// newRouter builds a chi router wired to the provided *internal.DB.
// It is package-private so it can be called from any test in the package.
func newRouter(db *internal.DB) http.Handler {
	r := chi.NewRouter()
	mountAPIRoutes(r, db)
	return r
}

// doRequest is a generic helper that fires an HTTP request against the
// provided handler (via httptest.NewRecorder) and returns the recorder.
func doRequest(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	var err error
	if body != nil {
		b, merr := json.Marshal(body)
		require.NoError(t, merr, "marshal request body")
		req, err = http.NewRequest(method, path, bytes.NewReader(b))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, path, nil)
		require.NoError(t, err)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// decodeBodyMap decodes the recorder body into a map[string]any.
func decodeBodyMap(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&m), "decode map body")
	return m
}

// decodeBodyBook decodes the recorder body into a bookResponse.
func decodeBodyBook(t *testing.T, rr *httptest.ResponseRecorder) bookResponse {
	t.Helper()
	var b bookResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&b), "decode book body")
	return b
}

// decodeBodyBooks decodes the recorder body into []bookResponse.
func decodeBodyBooks(t *testing.T, rr *httptest.ResponseRecorder) []bookResponse {
	t.Helper()
	var bs []bookResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&bs), "decode books body")
	return bs
}

// mustCreate is a shortcut that posts a bookPayload and asserts 201.
func mustCreate(t *testing.T, h http.Handler, p bookPayload) bookResponse {
	t.Helper()
	rr := doRequest(t, h, http.MethodPost, "/books/", p)
	require.Equal(t, http.StatusCreated, rr.Code, "mustCreate: unexpected status")
	return decodeBodyBook(t, rr)
}

// newIntegrationEnv is the same as newTestEnv but returns the handler
// separately so table-driven tests can fire requests directly without needing
// the full *httptest.Server round-trip (faster, no TCP).
func newIntegrationEnv(t *testing.T) (*testEnv, http.Handler) {
	t.Helper()
	env := newTestEnv(t) // skips if TEST_DATABASE_URL not set
	// rebuild the handler from env.server's underlying DB-wired router –
	// we can't get the db back from env.server, so we construct a new
	// handler via the same code path used in newTestEnv.
	dsn := osGetenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	db, err := internal.NewDB(ctx, dsn)
	require.NoError(t, err)
	require.NoError(t, internal.InitDB(ctx, db))
	require.NoError(t, truncateBooks(ctx, db))
	t.Cleanup(func() {
		_ = truncateBooks(context.Background(), db)
		_ = db.Close()
	})
	h := newRouter(db)
	return env, h
}

// ---------------------------------------------------------------------------
// Root endpoint
// ---------------------------------------------------------------------------

func TestReadRoot_TableDriven(t *testing.T) {
	_, h := newIntegrationEnv(t)

	tests := []struct {
		name           string
		method         string
		path           string
		wantStatus     int
		wantMessage    string
		wantVersion    string
		wantDocsURLKey bool
	}{
		{
			name:           "GET / returns welcome message and version",
			method:         http.MethodGet,
			path:           "/",
			wantStatus:     http.StatusOK,
			wantMessage:    "Welcome to Book Catalog API",
			wantVersion:    "1.0.0",
			wantDocsURLKey: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rr := doRequest(t, h, tc.method, tc.path, nil)
			assert.Equal(t, tc.wantStatus, rr.Code)

			data := decodeBodyMap(t, rr)
			assert.Equal(t, tc.wantMessage, data["message"], "message field")
			assert.Equal(t, tc.wantVersion, data["version"], "version field")
			if tc.wantDocsURLKey {
				assert.Contains(t, data, "docs_url", "docs_url field should be present")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Health-check endpoint
// ---------------------------------------------------------------------------

func TestHealthCheck_TableDriven(t *testing.T) {
	_, h := newIntegrationEnv(t)

	tests := []struct {
		name        string
		path        string
		wantStatus  int
		wantStatus2 string
		wantService string
	}{
		{
			name:        "GET /health returns healthy status",
			path:        "/health",
			wantStatus:  http.StatusOK,
			wantStatus2: "healthy",
			wantService: "book-catalog-api",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rr := doRequest(t, h, http.MethodGet, tc.path, nil)
			assert.Equal(t, tc.wantStatus, rr.Code)
			data := decodeBodyMap(t, rr)
			assert.Equal(t, tc.wantStatus2, data["status"], "status field")
			assert.Equal(t, tc.wantService, data["service"], "service field")
		})
	}
}

// ---------------------------------------------------------------------------
// Create book – POST /books/
// ---------------------------------------------------------------------------

func TestCreateBook_TableDriven(t *testing.T) {
	_, h := newIntegrationEnv(t)

	summary := "A test book summary"

	tests := []struct {
		name           string
		payload        any
		wantStatus     int
		wantTitle      string
		wantAuthor     string
		wantYear       int
		wantSummary    *string // nil means we expect null/absent
		wantNonZeroID  bool
		wantDetailHas  string // substring in error detail (if non-empty)
	}{
		{
			name: "all fields provided",
			payload: bookPayload{
				Title:         "Test Book",
				Author:        "Test Author",
				PublishedYear: intPtr(2023),
				Summary:       &summary,
			},
			wantStatus:    http.StatusCreated,
			wantTitle:     "Test Book",
			wantAuthor:    "Test Author",
			wantYear:      2023,
			wantSummary:   &summary,
			wantNonZeroID: true,
		},
		{
			name: "summary omitted – should be null",
			payload: bookPayload{
				Title:         "Book Without Summary",
				Author:        "Author",
				PublishedYear: intPtr(2023),
			},
			wantStatus:    http.StatusCreated,
			wantTitle:     "Book Without Summary",
			wantAuthor:    "Author",
			wantYear:      2023,
			wantSummary:   nil,
			wantNonZeroID: true,
		},
		{
			name:       "missing required fields – author and published_year absent",
			payload:    map[string]any{"title": "Orphan Title"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "published_year below minimum (999)",
			payload: map[string]any{
				"title":          "Too Old",
				"author":         "Scribe",
				"published_year": 999,
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rr := doRequest(t, h, http.MethodPost, "/books/", tc.payload)
			assert.Equal(t, tc.wantStatus, rr.Code)

			if tc.wantStatus == http.StatusCreated {
				book := decodeBodyBook(t, rr)
				assert.Equal(t, tc.wantTitle, book.Title)
				assert.Equal(t, tc.wantAuthor, book.Author)
				assert.Equal(t, tc.wantYear, book.PublishedYear)
				if tc.wantSummary == nil {
					assert.Nil(t, book.Summary, "expected nil summary")
				} else {
					require.NotNil(t, book.Summary, "expected non-nil summary")
					assert.Equal(t, *tc.wantSummary, *book.Summary)
				}
				if tc.wantNonZeroID {
					assert.NotZero(t, book.ID, "id should be assigned by server")
				}
			}
		})
	}
}

// TestCreateBook_Duplicate verifies the uniqueness constraint (title+author).
func TestCreateBook_Duplicate(t *testing.T) {
	_, h := newIntegrationEnv(t)

	base := bookPayload{
		Title:         "Unique Title",
		Author:        "Unique Author",
		PublishedYear: intPtr(2020),
	}

	tests := []struct {
		name       string
		first      bookPayload
		second     bookPayload
		wantFirst  int
		wantSecond int
		wantDetail string
	}{
		{
			name:       "same title and author → 400 duplicate",
			first:      base,
			second:     base,
			wantFirst:  http.StatusCreated,
			wantSecond: http.StatusBadRequest,
			wantDetail: "already exists",
		},
		{
			name: "same title different author → both 201",
			first: bookPayload{
				Title:         "Shared Title",
				Author:        "Author A",
				PublishedYear: intPtr(2021),
			},
			second: bookPayload{
				Title:         "Shared Title",
				Author:        "Author B",
				PublishedYear: intPtr(2021),
			},
			wantFirst:  http.StatusCreated,
			wantSecond: http.StatusCreated,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rr1 := doRequest(t, h, http.MethodPost, "/books/", tc.first)
			assert.Equal(t, tc.wantFirst, rr1.Code, "first request status")

			rr2 := doRequest(t, h, http.MethodPost, "/books/", tc.second)
			assert.Equal(t, tc.wantSecond, rr2.Code, "second request status")

			if tc.wantDetail != "" {
				m := decodeBodyMap(t, rr2)
				detail, _ := m["detail"].(string)
				assert.Contains(t, detail, tc.wantDetail, "error detail")
			}

			if tc.wantFirst == http.StatusCreated && tc.wantSecond == http.StatusCreated {
				b1 := decodeBodyBook(t, rr1)
				b2 := decodeBodyBook(t, rr2)
				assert.NotEqual(t, b1.ID, b2.ID, "distinct IDs expected")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// List books – GET /books/
// ---------------------------------------------------------------------------

func TestGetBooks_TableDriven(t *testing.T) {
	_, h := newIntegrationEnv(t)

	// Seed some books for the non-empty and pagination cases.
	seeds := []bookPayload{
		{Title: "List Book 1", Author: "LA1", PublishedYear: intPtr(2020)},
		{Title: "List Book 2", Author: "LA2", PublishedYear: intPtr(2021)},
		{Title: "List Book 3", Author: "LA3", PublishedYear: intPtr(2022)},
		{Title: "List Book 4", Author: "LA4", PublishedYear: intPtr(2023)},
		{Title: "List Book 5", Author: "LA5", PublishedYear: intPtr(2024)},
	}
	seededIDs := make([]int64, 0, len(seeds))
	for _, s := range seeds {
		b := mustCreate(t, h, s)
		seededIDs = append(seededIDs, b.ID)
	}

	tests := []struct {
		name          string
		path          string
		wantStatus    int
		wantMinLen    int
		wantExactLen  int // 0 means "don't check exact"
		wantNonNil    bool
		wantAllSeeded bool // all seeded IDs present in response
	}{
		{
			name:       "empty path with seeded data returns all seeded books",
			path:       "/books/",
			wantStatus: http.StatusOK,
			wantMinLen: len(seeds),
			wantNonNil: true,
		},
		{
			name:         "skip=2&limit=2 returns exactly 2 items",
			path:         "/books/?skip=2&limit=2",
			wantStatus:   http.StatusOK,
			wantExactLen: 2,
			wantNonNil:   true,
		},
		{
			name:         "limit=0 returns no items (or honours limit=0)",
			path:         "/books/?limit=0",
			wantStatus:   http.StatusOK,
			wantExactLen: 0,
			wantNonNil:   true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rr := doRequest(t, h, http.MethodGet, tc.path, nil)
			assert.Equal(t, tc.wantStatus, rr.Code)

			books := decodeBodyBooks(t, rr)
			if tc.wantNonNil {
				assert.NotNil(t, books, "response must be a JSON array, not null")
			}
			if tc.wantMinLen > 0 {
				assert.GreaterOrEqual(t, len(books), tc.wantMinLen)
			}
			if tc.want