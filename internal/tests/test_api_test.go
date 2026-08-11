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
)

// ---------------------------------------------------------------------------
// Minimal stub for mustNewTestServer so the file compiles standalone.
// Replace with the real implementation from internal/conftest.go.
// ---------------------------------------------------------------------------

// bookStore is the in-memory backing store used by the stub server.
type bookStore struct {
	books  []map[string]any
	nextID int
}

func newBookStore() *bookStore {
	return &bookStore{nextID: 1}
}

func (s *bookStore) add(book map[string]any) map[string]any {
	book["id"] = s.nextID
	s.nextID++
	s.books = append(s.books, book)
	return book
}

func (s *bookStore) find(id int) (map[string]any, bool) {
	for _, b := range s.books {
		if int(b["id"].(int)) == id {
			return b, true
		}
	}
	return nil, false
}

func (s *bookStore) findByTitleAuthor(title, author string) bool {
	for _, b := range s.books {
		if b["title"] == title && b["author"] == author {
			return true
		}
	}
	return false
}

func (s *bookStore) update(id int, patch map[string]any) (map[string]any, bool) {
	for i, b := range s.books {
		if int(b["id"].(int)) == id {
			for k, v := range patch {
				b[k] = v
			}
			s.books[i] = b
			return b, true
		}
	}
	return nil, false
}

func (s *bookStore) delete(id int) bool {
	for i, b := range s.books {
		if int(b["id"].(int)) == id {
			s.books = append(s.books[:i], s.books[i+1:]...)
			return true
		}
	}
	return false
}

func (s *bookStore) list(skip, limit int) []map[string]any {
	all := s.books
	if skip >= len(all) {
		return []map[string]any{}
	}
	all = all[skip:]
	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}
	return all
}

// mustNewTestServer builds a stub HTTP server that exercises the same contract
// as the real server. Replace body with internal.NewTestServer when available.
func mustNewTestServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	store := newBookStore()
	mux := http.NewServeMux()

	// GET /
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"message":  "Welcome to Book Catalog API",
			"version":  "1.0.0",
			"docs_url": "/docs",
		})
	})

	// GET /health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "healthy",
			"service": "book-catalog-api",
		})
	})

	// /books/  and  /books/{id}
	mux.HandleFunc("/books/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// Strip trailing slash for ID routes
		idPart := strings.TrimPrefix(path, "/books/")
		idPart = strings.TrimSuffix(idPart, "/")

		if idPart == "" {
			// Collection routes
			switch r.Method {
			case http.MethodGet:
				skip := queryInt(r, "skip", 0)
				limit := queryInt(r, "limit", 0)
				books := store.list(skip, limit)
				writeJSON(w, http.StatusOK, books)

			case http.MethodPost:
				var body map[string]any
				if err := decodeBody(r, &body); err != nil {
					writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
						"detail": "invalid body",
					})
					return
				}
				// Validate required fields
				title, hasTitle := body["title"].(string)
				author, hasAuthor := body["author"].(string)
				yearRaw, hasYear := body["published_year"]

				if !hasTitle || !hasAuthor || !hasYear {
					writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
						"detail": "title, author, and published_year are required",
					})
					return
				}

				year, ok := toInt(yearRaw)
				if !ok || year < 1000 {
					writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
						"detail": "published_year is invalid or below minimum",
					})
					return
				}

				if store.findByTitleAuthor(title, author) {
					writeJSON(w, http.StatusBadRequest, map[string]any{
						"detail": fmt.Sprintf("book with title '%s' and author '%s' already exists", title, author),
					})
					return
				}

				summary, _ := body["summary"]
				book := map[string]any{
					"title":          title,
					"author":         author,
					"published_year": year,
					"summary":        summary,
				}
				created := store.add(book)
				writeJSON(w, http.StatusCreated, created)

			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		// ID routes
		var id int
		if _, err := fmt.Sscanf(idPart, "%d", &id); err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"detail": "invalid id",
			})
			return
		}

		switch r.Method {
		case http.MethodGet:
			book, found := store.find(id)
			if !found {
				writeJSON(w, http.StatusNotFound, map[string]any{
					"detail": fmt.Sprintf("book with id %d not found", id),
				})
				return
			}
			writeJSON(w, http.StatusOK, book)

		case http.MethodPut:
			var body map[string]any
			if err := decodeBody(r, &body); err != nil {
				writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
					"detail": "invalid body",
				})
				return
			}

			if yearRaw, hasYear := body["published_year"]; hasYear {
				year, ok := toInt(yearRaw)
				if !ok || year < 1000 {
					writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
						"detail": "published_year is invalid or below minimum",
					})
					return
				}
				body["published_year"] = year
			}

			updated, found := store.update(id, body)
			if !found {
				writeJSON(w, http.StatusNotFound, map[string]any{
					"detail": fmt.Sprintf("book with id %d not found", id),
				})
				return
			}
			writeJSON(w, http.StatusOK, updated)

		case http.MethodDelete:
			if !store.delete(id) {
				writeJSON(w, http.StatusNotFound, map[string]any{
					"detail": fmt.Sprintf("book with id %d not found", id),
				})
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	srv := httptest.NewServer(mux)
	cleanup := func() { srv.Close() }
	return srv, cleanup
}

// ---------------------------------------------------------------------------
// Small helpers used by the stub server
// ---------------------------------------------------------------------------

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func decodeBody(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func toInt(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	case json.Number:
		n, err := x.Int64()
		return int(n), err == nil
	}
	return 0, false
}

func queryInt(r *http.Request, key string, def int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestReadRoot covers the root endpoint spec.
func TestReadRoot(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name               string
		wantStatus         int
		wantMessage        string
		wantVersion        string
		wantDocsURLPresent bool
	}{
		{
			name:               "GET request to root path returns 200 with metadata",
			wantStatus:         http.StatusOK,
			wantMessage:        "Welcome to Book Catalog API",
			wantVersion:        "1.0.0",
			wantDocsURLPresent: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := newAPIClient(t)

			resp := c.get("/")

			assert.Equal(t, tc.wantStatus, resp.statusCode)
			body := resp.jsonMap()
			assert.Equal(t, tc.wantMessage, body["message"])
			assert.Equal(t, tc.wantVersion, body["version"])
			_, hasDocsURL := body["docs_url"]
			assert.True(t, hasDocsURL, "response should include docs_url field")
		})
	}
}

// TestHealthCheck covers the /health endpoint spec.
func TestHealthCheck(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		wantStatus    int
		wantStatusVal string
		wantService   string
	}{
		{
			name:          "GET request to health path returns 200 with status healthy",
			wantStatus:    http.StatusOK,
			wantStatusVal: "healthy",
			wantService:   "book-catalog-api",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := newAPIClient(t)

			resp := c.get("/health")

			assert.Equal(t, tc.wantStatus, resp.statusCode)
			body := resp.jsonMap()
			assert.Equal(t, tc.wantStatusVal, body["status"])
			assert.Equal(t, tc.wantService, body["service"])
		})
	}
}

// TestCreateBook covers all create_book behavioral specs.
func TestCreateBook(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name           string
		payload        map[string]any
		wantStatus     int
		wantDetailSubs []string // substrings expected in the detail field
		checkBody      func(t *testing.T, body map[string]any)
	}{
		{
			name: "valid book with all fields returns 201",
			payload: map[string]any{
				"title":          "The Go Programming Language",
				"author":         "Alan Donovan",
				"published_year": 2015,
				"summary":        "An intro to Go",
			},
			wantStatus: http.StatusCreated,
			checkBody: func(t *testing.T, body map[string]any) {
				t.Helper()
				assert.Equal(t, "The Go Programming Language", body["title"])
				assert.Equal(t, "Alan Donovan", body["author"])
				assert.EqualValues(t, 2015, toIntAny(body["published_year"]))
				assert.Equal(t, "An intro to Go", body["summary"])
				assert.NotNil(t, body["id"])
			},
		},
		{
			name: "valid book without summary has null summary",
			payload: map[string]any{
				"title":          "Clean Code",
				"author":         "Robert Martin",
				"published_year": 2008,
			},
			wantStatus: http.StatusCreated,
			checkBody: func(t *testing.T, body map[string]any) {
				t.Helper()
				assert.Equal(t, "Clean Code", body["title"])
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
			name: "published_year too early returns 422",
			payload: map[string]any{
				"title":          "Ancient Tome",
				"author":         "Someone",
				"published_year": 999,
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := newAPIClient(t)

			resp := c.post("/books/", tc.payload)

			assert.Equal(t, tc.wantStatus, resp.statusCode)
			if tc.checkBody != nil {
				tc.checkBody(t, resp.jsonMap())
			}
			for _, sub := range tc.wantDetailSubs {
				assert.Contains(t, resp.detail(), sub)
			}
		})
	}
}

// TestCreateBookDuplicate validates the unique title+author constraint.
func TestCreateBookDuplicate(t *testing.T) {
	t.Parallel()
	c := newAPIClient(t)

	payload := map[string]any{
		"title":          "Unique Book",
		"author":         "Author One",
		"published_year": 2020,
	}

	// First creation should succeed.
	first := c.post("/books/", payload)
	require.Equal(t, http.StatusCreated, first.statusCode)

	// Second creation with same title+author should fail with 400.
	second := c.post("/books/", payload)
	assert.Equal(t, http.StatusBadRequest, second.statusCode)
	assert.Contains(t, second.detail(), "already exists")
}