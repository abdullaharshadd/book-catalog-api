```go
package tests_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bookcatalog/internal"
	"bookcatalog/internal/tests"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type testClient struct {
	t      *testing.T
	server *httptest.Server
}

func newTestClient(t *testing.T) *testClient {
	t.Helper()

	db, err := internal.NewTestDB()
	require.NoError(t, err, "NewTestDB")

	srv, err := internal.NewTestServer(db)
	require.NoError(t, err, "NewTestServer")

	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	return &testClient{t: t, server: ts}
}

func (c *testClient) url(path string) string {
	return c.server.URL + path
}

func (c *testClient) get(path string) *http.Response {
	c.t.Helper()
	resp, err := http.Get(c.url(path))
	require.NoError(c.t, err, "GET %s", path)
	return resp
}

func (c *testClient) post(path string, body any) *http.Response {
	c.t.Helper()
	buf, err := json.Marshal(body)
	require.NoError(c.t, err, "marshal POST body")
	resp, err := http.Post(c.url(path), "application/json", bytes.NewReader(buf))
	require.NoError(c.t, err, "POST %s", path)
	return resp
}

func (c *testClient) put(path string, body any) *http.Response {
	c.t.Helper()
	buf, err := json.Marshal(body)
	require.NoError(c.t, err, "marshal PUT body")
	req, err := http.NewRequest(http.MethodPut, c.url(path), bytes.NewReader(buf))
	require.NoError(c.t, err, "build PUT %s", path)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(c.t, err, "PUT %s", path)
	return resp
}

func (c *testClient) del(path string) *http.Response {
	c.t.Helper()
	req, err := http.NewRequest(http.MethodDelete, c.url(path), nil)
	require.NoError(c.t, err, "build DELETE %s", path)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(c.t, err, "DELETE %s", path)
	return resp
}

func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "read body")
	return data
}

func decodeObject(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	data := readBody(t, resp)
	var out map[string]any
	err := json.Unmarshal(data, &out)
	require.NoError(t, err, "unmarshal object: %s", string(data))
	return out
}

func decodeList(t *testing.T, resp *http.Response) []map[string]any {
	t.Helper()
	data := readBody(t, resp)
	var out []map[string]any
	err := json.Unmarshal(data, &out)
	require.NoError(t, err, "unmarshal list: %s", string(data))
	return out
}

// idStr converts an id field (float64 or string) to a string key usable in maps.
func idStr(v any) string {
	switch x := v.(type) {
	case float64:
		return fmt.Sprintf("%.0f", x)
	case string:
		return x
	default:
		return fmt.Sprint(v)
	}
}

// numEq compares a JSON-decoded number (float64) to an int.
func numEq(v any, want int) bool {
	switch x := v.(type) {
	case float64:
		return int(x) == want
	case json.Number:
		i, err := x.Int64()
		return err == nil && int(i) == want
	}
	return false
}

// ---------------------------------------------------------------------------
// RunAPITests – the public entry-point mirroring test_api.go
// ---------------------------------------------------------------------------

func TestRunAPITests(t *testing.T) {
	tests.RunAPITests(t)
}

// ---------------------------------------------------------------------------
// Table-driven tests covering every behavioural spec
// ---------------------------------------------------------------------------

// --- Root endpoint ---

func TestReadRoot(t *testing.T) {
	cases := []struct {
		name           string
		wantStatus     int
		wantMessage    string
		wantVersion    string
		wantDocsURLKey bool
	}{
		{
			name:           "GET / returns welcome payload",
			wantStatus:     http.StatusOK,
			wantMessage:    "Welcome to Book Catalog API",
			wantVersion:    "1.0.0",
			wantDocsURLKey: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			c := newTestClient(t)

			resp := c.get("/")
			assert.Equal(t, tc.wantStatus, resp.StatusCode)

			data := decodeObject(t, resp)
			assert.Equal(t, tc.wantMessage, data["message"])
			assert.Equal(t, tc.wantVersion, data["version"])
			if tc.wantDocsURLKey {
				assert.Contains(t, data, "docs_url", "response must contain docs_url key")
			}
		})
	}
}

// --- Health endpoint ---

func TestHealthCheck(t *testing.T) {
	cases := []struct {
		name          string
		wantStatus    int
		wantStatus2   string
		wantService   string
	}{
		{
			name:        "GET /health returns healthy",
			wantStatus:  http.StatusOK,
			wantStatus2: "healthy",
			wantService: "book-catalog-api",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			c := newTestClient(t)

			resp := c.get("/health")
			assert.Equal(t, tc.wantStatus, resp.StatusCode)

			data := decodeObject(t, resp)
			assert.Equal(t, tc.wantStatus2, data["status"])
			assert.Equal(t, tc.wantService, data["service"])
		})
	}
}

// --- Create book ---

func TestCreateBook(t *testing.T) {
	cases := []struct {
		name        string
		body        map[string]any
		wantStatus  int
		checkFields func(t *testing.T, data map[string]any, body map[string]any)
	}{
		{
			name: "all fields present",
			body: map[string]any{
				"title":          "Test Book",
				"author":         "Test Author",
				"published_year": 2023,
				"summary":        "A test book summary",
			},
			wantStatus: http.StatusCreated,
			checkFields: func(t *testing.T, data, body map[string]any) {
				assert.Equal(t, body["title"], data["title"])
				assert.Equal(t, body["author"], data["author"])
				assert.True(t, numEq(data["published_year"], 2023))
				assert.Equal(t, body["summary"], data["summary"])
				assert.Contains(t, data, "id", "response must contain id")
			},
		},
		{
			name: "without summary – summary is null",
			body: map[string]any{
				"title":          "Book Without Summary",
				"author":         "Author",
				"published_year": 2023,
			},
			wantStatus: http.StatusCreated,
			checkFields: func(t *testing.T, data, body map[string]any) {
				assert.Equal(t, body["title"], data["title"])
				assert.Equal(t, body["author"], data["author"])
				assert.True(t, numEq(data["published_year"], 2023))
				assert.Nil(t, data["summary"])
				assert.Contains(t, data, "id")
			},
		},
		{
			name:       "missing required fields returns 422",
			body:       map[string]any{"title": "Test Book"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "published_year below minimum returns 422",
			body: map[string]any{
				"title":          "Test Book",
				"author":         "Test Author",
				"published_year": 999,
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			c := newTestClient(t)

			resp := c.post("/books/", tc.body)
			assert.Equal(t, tc.wantStatus, resp.StatusCode)

			if tc.checkFields != nil {
				data := decodeObject(t, resp)
				tc.checkFields(t, data, tc.body)
			} else {
				resp.Body.Close()
			}
		})
	}
}

// --- Duplicate / same-title-different-author ---

func TestCreateDuplicateBook(t *testing.T) {
	t.Run("same title and author returns 400 with already exists", func(t *testing.T) {
		c := newTestClient(t)

		book := map[string]any{
			"title":          "Duplicate Title",
			"author":         "Same Author",
			"published_year": 2020,
		}

		resp1 := c.post("/books/", book)
		assert.Equal(t, http.StatusCreated, resp1.StatusCode)
		resp1.Body.Close()

		resp2 := c.post("/books/", book)
		assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)

		data := decodeObject(t, resp2)
		detail := strings.ToLower(fmt.Sprint(data["detail"]))
		assert.Contains(t, detail, "already exists")
	})
}

func TestBooksSameTitleDifferentAuthors(t *testing.T) {
	t.Run("same title different authors both return 201 with distinct ids", func(t *testing.T) {
		c := newTestClient(t)

		b1 := map[string]any{"title": "Shared Title", "author": "Author A", "published_year": 2021}
		b2 := map[string]any{"title": "Shared Title", "author": "Author B", "published_year": 2022}

		r1 := c.post("/books/", b1)
		assert.Equal(t, http.StatusCreated, r1.StatusCode)
		d1 := decodeObject(t, r1)

		r2 := c.post("/books/", b2)
		assert.Equal(t, http.StatusCreated, r2.StatusCode)
		d2 := decodeObject(t, r2)

		assert.NotEqual(t, idStr(d1["id"]), idStr(d2["id"]), "ids must differ")
	})
}

// --- Get books (list) ---

func TestGetBooksEmpty(t *testing.T) {
	t.Run("empty database returns empty array", func(t *testing.T) {
		c := newTestClient(t)

		resp := c.get("/books/")
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		books := decodeList(t, resp)
		assert.Empty(t, books)
	})
}

func TestGetBooksWithData(t *testing.T) {
	t.Run("three created books all appear in list", func(t *testing.T) {
		c := newTestClient(t)

		booksData := []map[string]any{
			{"title": "Book 1", "author": "Author 1", "published_year": 2021},
			{"title": "Book 2", "author": "Author 2", "published_year": 2022},
			{"title": "Book 3", "author": "Author 3", "published_year": 2023},
		}

		createdIDs := map[string]bool{}
		for _, bd := range booksData {
			resp := c.post("/books/", bd)
			require.Equal(t, http.StatusCreated, resp.StatusCode)
			d := decodeObject(t, resp)
			createdIDs[idStr(d["id"])] = true
		}

		resp := c.get("/books/")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		books := decodeList(t, resp)

		assert.Len(t, books, 3)

		retrievedIDs := map[string]bool{}
		for _, b := range books {
			retrievedIDs[idStr(b["id"])] = true
		}
		assert.Equal(t, createdIDs, retrievedIDs)
	})
}

func TestGetBooksWithPagination(t *testing.T) {
	cases := []struct {
		name        string
		query       string
		seedCount   int
		wantMinAll  int
		wantExactPg int
	}{
		{
			name:        "skip=2&limit=2 returns exactly 2 books",
			query:       "/books/?skip=2&limit=2",
			seedCount:   5,
			wantMinAll:  5,
			wantExactPg: 2,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			c := newTestClient(t)

			for i := 0; i < tc.seedCount; i++ {
				bd := map[string]any{
					"title":          fmt.Sprintf("Pagination Book %d", i+1),
					"author":         fmt.Sprintf("Pagination Author %d", i+1),
					"published_year": 2020 + i,
				}
				resp := c.post("/books/", bd)
				require.Equal(t, http.StatusCreated, resp.StatusCode)
				resp.Body.Close()
			}

			allResp := c.get("/books/")
			assert.Equal(t, http.StatusOK, allResp.StatusCode)
			all := decodeList(t, allResp)
			assert.GreaterOrEqual(t, len(all), tc.wantMinAll)

			pgResp := c.get(tc.query)
			assert.Equal(t, http.StatusOK, pgResp.StatusCode)
			pg := decodeList(t, pgResp)
			assert.Len(t, pg, tc.wantExactPg)
		})
	}
}

// --- Get book by ID ---

func TestGetBookByID(t *testing.T) {
	cases := []struct {
		name       string
		id         string
		wantStatus int
		wantDetail string // substring in detail field for error cases
	}{
		{
			name:       "existing book returns 200",
			id:         "", // filled dynamically
			wantStatus: http.StatusOK,