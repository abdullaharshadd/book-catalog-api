```go
package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"internal"
	"internal/database"
	"internal/model"
	"internal/schemas"
)

type testEntry struct {
	name               string
	method             string
	path               string
	body               interface{}
	expectedStatusCode int
	expectedResponse   interface{}
}

func TestSuite(t *testing.T) {
	a := assert.New(t)
	req := require.New(t)
	api := internal.NewBookCatalogAPI()
	db, err := database.NewDatabase()
	req.NoError(err)
	db = getSyncDB(t, db)
	defer db.Close()
	dropAllTables(t, db)

	testCases := []testEntry{
		{"root", "GET", "/", nil, http.StatusOK, map[string]interface{}{
			"message": "Welcome to Book Catalog API",
			"version": "1.0.0",
			"docs_url": "/docs",
		}},
		{"health_check", "GET", "/health", nil, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": "book-catalog-api",
		}},
		{"create_book", "POST", "/books/", schemas.BookCreate{
			Title:          "Test Book",
			Author:         "Test Author",
			PublishedYear:  2023,
			Summary:        &"A test book summary",
		}, http.StatusCreated, schemas.BookResponse{
			Title:          "Test Book",
			Author:         "Test Author",
			PublishedYear:  2023,
			Summary:        "A test book summary",
		}},
		{"create_book_without_summary", "POST", "/books/", schemas.BookCreate{
			Title:          "Book Without Summary",
			Author:         "Author",
			PublishedYear:  2023,
		}, http.StatusCreated, schemas.BookResponse{
			Title:          "Book Without Summary",
			Author:         "Author",
			PublishedYear:  2023,
			Summary:        "",
		}},
		{"create_book_validation_error", "POST", "/books/", schemas.BookCreate{
			Title: "Test Book",
		}, http.StatusUnprocessableEntity, map[string]interface{}{
			"message": "Validation failed",
		}},
		{"get_books_empty", "GET", "/books/", nil, http.StatusOK, []schemas.BookResponse{}},
		{"get_books_with_data", "GET", "/books/", nil, http.StatusOK, []schemas.BookResponse{
			{ID: 1, Title: "Test Book", Author: "Test Author", PublishedYear: 2023, Summary: "A test book summary"},
			{ID: 2, Title: "Book Without Summary", Author: "Author", PublishedYear: 2023},
		}},
		{"get_books_with_pagination", "GET", "/books/?skip=1&limit=1", nil, http.StatusOK, []schemas.BookResponse{
			{ID: 2, Title: "Book Without Summary", Author: "Author", PublishedYear: 2023},
		}},
		{"get_book_by_id", "GET", "/books/1", nil, http.StatusOK, schemas.BookResponse{
			ID:            1,
			Title:         "Test Book",
			Author:        "Test Author",
			PublishedYear: 2023,
			Summary:       "A test book summary",
		}},
		{"get_book_not_found", "GET", "/books/999", nil, http.StatusNotFound, map[string]interface{}{
			"message": "Book not found",
		}},
		{"update_book", "PUT", "/books/1", schemas.BookUpdate{
			Title:          "Updated Test Book",
			PublishedYear:  2024,
		}, http.StatusOK, schemas.BookResponse{
			ID:            1,
			Title:         "Updated Test Book",
			Author:        "Test Author",
			PublishedYear: 2024,
			Summary:       "A test book summary",
		}},
		{"update_book_not_found", "PUT", "/books/999", schemas.BookUpdate{
			Title: "Updated Test Book",
		}, http.StatusNotFound, map[string]interface{}{
			"message": "Book not found",
		}},
		{"update_book_validation_error", "PUT", "/books/1", schemas.BookUpdate{
			Title: "Updated Test Book",
			PublishedYear: 0,
		}, http.StatusUnprocessableEntity, map[string]interface{}{
			"message": "Validation failed",
		}},
		{"delete_book", "DELETE", "/books/1", nil, http.StatusNoContent, nil},
		{"delete_book_not_found", "DELETE", "/books/999", nil, http.StatusNotFound, map[string]interface{}{
			"message": "Book not found",
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := httptest.NewServer(api.Router())
			defer client.Close()

			var resp *http.Response
			var err error

			if tc.method == "GET" {
				resp, err = client.Client().Get(client.URL + tc.path)
			} else {
				resp, err = client.Client().Post(client.URL+tc.path, "application/json", readerOfJSON(t, tc.body))
			}

			req.NoError(err)
			req.Equal(tc.expectedStatusCode, resp.StatusCode)

			if tc.expectedStatusCode != http.StatusNoContent {
				var data interface{}
				err = json.NewDecoder(resp.Body).Decode(&data)
				req.NoError(err)

				if len(tc.expectedResponse.([]schemas.BookResponse)) == 0 {
					assert.Empty(t, data)
				} else {
					a.Equal(tc.expectedResponse, data)
				}
			}
		})
	}
}

func readerOfJSON(t *testing.T, v interface{}) *json.Reader {
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return json.NewReader(data)
}

func getSyncDB(t *testing.T, db *database.DB) *database.DB {
	// Mock DB logic here
	return db
}

func dropAllTables(t *testing.T, db *database.DB) {
	// Mock DB logic here
}

```