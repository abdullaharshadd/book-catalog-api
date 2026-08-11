```go
package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bookcatalog/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers (mirrors the helpers in test_api.go but usable from _test.go files)
// ---------------------------------------------------------------------------

func setupTestEnv(t *testing.T) (*httptest.Server, func()) {
	t.Helper()

	db := internal.NewTestDB(t)
	router := internal.NewTestRouter(t, db)
	srv := internal.NewTestServer(t, router)

	cleanup := func() {
		srv.Close()
		if err := db.Close(); err != nil {
			t.Errorf("closing test db: %v", err)
		}
	}
	return srv, cleanup
}

func doJSONReq(t *testing.T, srv *httptest.Server, method, path string, body any, out any) *http.Response {
	t.Helper()

	var reqBody *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err, "marshaling request body")
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, srv.URL+path, reqBody)
	require.NoError(t, err, "building request")
	req.Header.Set("Content-Type", "application/json")

	resp, err := srv.Client().Do(req)
	require.NoError(t, err, "performing request")

	if out != nil {
		defer resp.Body.Close()
		require.NoError(t, json.NewDecoder(resp.Body).Decode(out), "decoding response body")
	}
	return resp
}

func createBookHelper(t *testing.T, srv *httptest.Server, body map[string]any) map[string]any {
	t.Helper()
	var created map[string]any
	resp := doJSONReq(t, srv, http.MethodPost, "/books/", body, &created)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "createBook: unexpected status")
	return created
}

// ---------------------------------------------------------------------------
// TestReadRootTableDriven
// ---------------------------------------------------------------------------

func TestReadRootTableDriven(t *testing.T) {
	srv, cleanup := setupTestEnv(t)
	defer cleanup()

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
			name:           "GET request to root path returns welcome message",
			method:         http.MethodGet,
			path:           "/",
			wantStatus:     http.StatusOK,
			wantMessage:    "Welcome to Book Catalog API",
			wantVersion:    "1.0.0",
			wantDocsURLKey: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var data map[string]any
			resp := doJSONReq(t, srv, tt.method, tt.path, nil, &data)

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			assert.Equal(t, tt.wantMessage, data["message"], "message field")
			assert.Equal(t, tt.wantVersion, data["version"], "version field")
			if tt.wantDocsURLKey {
				_, ok := data["docs_url"]
				assert.True(t, ok, "expected docs_url key in response")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestHealthCheckTableDriven
// ---------------------------------------------------------------------------

func TestHealthCheckTableDriven(t *testing.T) {
	srv, cleanup := setupTestEnv(t)
	defer cleanup()

	tests := []struct {
		name        string
		method      string
		path        string
		wantStatus  int
		wantStatus2 string
		wantService string
	}{
		{
			name:        "GET request to health path returns healthy status",
			method:      http.MethodGet,
			path:        "/health",
			wantStatus:  http.StatusOK,
			wantStatus2: "healthy",
			wantService: "book-catalog-api",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var data map[string]any
			resp := doJSONReq(t, srv, tt.method, tt.path, nil, &data)

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			assert.Equal(t, tt.wantStatus2, data["status"], "status field")
			assert.Equal(t, tt.wantService, data["service"], "service field")
		})
	}
}

// ---------------------------------------------------------------------------
// TestCreateBookTableDriven
// ---------------------------------------------------------------------------

func TestCreateBookTableDriven(t *testing.T) {
	tests := []struct {
		name           string
		body           map[string]any
		wantStatus     int
		wantTitle      string
		wantAuthor     string
		wantYear       float64
		wantSummary    any // nil means expect null/absent
		checkSummaryNil bool
		checkID        bool
		checkDetail    string // non-empty means check detail contains this substring
	}{
		{
			name: "valid book with all fields including summary",
			body: map[string]any{
				"title":          "Test Book",
				"author":         "Test Author",
				"published_year": 2023,
				"summary":        "A test book summary",
			},
			wantStatus:  http.StatusCreated,
			wantTitle:   "Test Book",
			wantAuthor:  "Test Author",
			wantYear:    2023,
			wantSummary: "A test book summary",
			checkID:     true,
		},
		{
			name: "valid book without summary",
			body: map[string]any{
				"title":          "Book Without Summary",
				"author":         "Author",
				"published_year": 2023,
			},
			wantStatus:      http.StatusCreated,
			wantTitle:       "Book Without Summary",
			wantAuthor:      "Author",
			wantYear:        2023,
			checkSummaryNil: true,
			checkID:         true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			srv, cleanup := setupTestEnv(t)
			defer cleanup()

			var data map[string]any
			resp := doJSONReq(t, srv, http.MethodPost, "/books/", tt.body, &data)

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			assert.Equal(t, tt.wantTitle, data["title"])
			assert.Equal(t, tt.wantAuthor, data["author"])
			assert.Equal(t, tt.wantYear, data["published_year"])

			if tt.checkSummaryNil {
				got, ok := data["summary"]
				if ok {
					assert.Nil(t, got, "summary should be null")
				}
			} else if tt.wantSummary != nil {
				assert.Equal(t, tt.wantSummary, data["summary"])
			}

			if tt.checkID {
				_, ok := data["id"]
				assert.True(t, ok, "expected id key in response")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestCreateBookValidationErrorTableDriven
// ---------------------------------------------------------------------------

func TestCreateBookValidationErrorTableDriven(t *testing.T) {
	srv, cleanup := setupTestEnv(t)
	defer cleanup()

	tests := []struct {
		name string
		body map[string]any
	}{
		{
			name: "missing required fields – only title provided",
			body: map[string]any{"title": "Test Book"},
		},
		{
			name: "published_year too early – below minimum 1000",
			body: map[string]any{
				"title":          "Test Book",
				"author":         "Test Author",
				"published_year": 999,
			},
		},
		{
			name: "missing author field",
			body: map[string]any{
				"title":          "No Author Book",
				"published_year": 2020,
			},
		},
		{
			name: "missing published_year field",
			body: map[string]any{
				"title":  "No Year Book",
				"author": "Some Author",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var verr internal.ValidationError
			resp := doJSONReq(t, srv, http.MethodPost, "/books/", tt.body, &verr)

			assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
			require.NotEmpty(t, verr.Detail, "expected at least one field error in detail")

			for i, fe := range verr.Detail {
				assert.NotEmpty(t, fe.Loc, "detail[%d].loc is empty", i)
				assert.NotEmpty(t, fe.Msg, "detail[%d].msg is empty", i)
				assert.NotEmpty(t, fe.Type, "detail[%d].type is empty", i)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestCreateBookDuplicateTableDriven
// ---------------------------------------------------------------------------

func TestCreateBookDuplicateTableDriven(t *testing.T) {
	tests := []struct {
		name         string
		first        map[string]any
		second       map[string]any
		wantStatus   int
		detailSubstr string
	}{
		{
			name: "duplicate title and author returns 400",
			first: map[string]any{
				"title":          "Dup Book",
				"author":         "Dup Author",
				"published_year": 2022,
			},
			second: map[string]any{
				"title":          "Dup Book",
				"author":         "Dup Author",
				"published_year": 2022,
			},
			wantStatus:   http.StatusBadRequest,
			detailSubstr: "already exists",
		},
		{
			name: "same title different author is allowed",
			first: map[string]any{
				"title":          "Shared Title",
				"author":         "Author One",
				"published_year": 2021,
			},
			second: map[string]any{
				"title":          "Shared Title",
				"author":         "Author Two",
				"published_year": 2021,
			},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			srv, cleanup := setupTestEnv(t)
			defer cleanup()

			// Create first book — must succeed.
			first := createBookHelper(t, srv, tt.first)
			firstID, ok := first["id"].(float64)
			require.True(t, ok, "first book id missing")

			// Attempt second creation.
			var data map[string]any
			resp := doJSONReq(t, srv, http.MethodPost, "/books/", tt.second, &data)

			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if tt.detailSubstr != "" {
				detail, _ := data["detail"].(string)
				assert.True(t,
					strings.Contains(strings.ToLower(detail), tt.detailSubstr),
					"detail %q should contain %q", detail, tt.detailSubstr,
				)
			}

			// For the allowed-duplicate case, verify both books exist with distinct ids.
			if tt.wantStatus == http.StatusCreated {
				secondID, ok := data["id"].(float64)
				require.True(t, ok, "second book id missing")
				assert.NotEqual(t, firstID, secondID, "expected distinct ids")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetBooksTableDriven
// ---------------------------------------------------------------------------

func TestGetBooksTableDriven(t *testing.T) {
	tests := []struct {
		name        string
		seed        []map[string]any
		path        string
		wantStatus  int
		minCount    int
		exactCount  *int // pointer so 0 is distinguishable from "not checked"
	}{
		{
			name:       "empty database returns empty array",
			seed:       nil,
			path:       "/books/",
			wantStatus: http.StatusOK,
			exactCount: intPtr(0),
		},
		{
			name: "existing books are returned",
			seed: []map[string]any{
				{"title": "Book A", "author": "Author A", "published_year": 2021},
				{"title": "Book B", "author": "Author B", "published_year": 2022},
				{"title": "Book C", "author": "Author C", "published_year": 2023},
			},
			path:       "/books/",
			wantStatus: http.StatusOK,
			minCount:   3,
		},
		{
			name: "pagination skip=2&limit=2 returns 2 items",
			seed: []map[string]any{
				{"title": "P Book 1", "author": "P Author 1", "published_year": 2020},
				{"title": "P Book 2", "author": "P Author 2", "published_year": 2021},
				{"title": "P Book 3", "author": "P Author 3", "published_year": 2022},
				{"title": "P Book 4", "author": "P Author 4", "published_year": 2023},
				{"title": "P Book 5", "author": "P Author 5", "published_year": 2024},
			},
			path:       "/books/?skip=2&limit=2",
			wantStatus: http.StatusOK,
			exactCount: intPtr(2),
		},
		{
			name: "pagination skip=0&limit=3 returns 3 items",
			seed: []map[string]any{
				{"title": "L Book 1", "author": "L Author 1", "published_year": 2020},
				{"title": "L Book 2", "author": "L Author 2", "published_year": 2021},
				{"title": "L Book 3", "author": "L Author 3", "published_year": 2022},
				{"title": "L Book 4", "author": "L Author 4", "published_year": 2023},
			},
			path:       "/books/?skip=0&limit=3",
			wantStatus: http.StatusOK,
			exactCount: intPtr(3),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			srv, cleanup := setupTestEnv(t)
			defer cleanup()

			createdIDs := make(map[float64]bool)
			for _, bd := range tt.seed {
				created := createBookHelper(t, srv, bd)
				if id, ok := created["id"].(float64); ok {
					createdIDs[id] = true
				}
			}

			var books []map[string]any
			resp := doJSONReq(t, srv,