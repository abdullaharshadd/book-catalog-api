package tests

import (
	"context"
	"encoding/json"
	"net/http"
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

// TestSuite represents the test suite for the Book Catalog API.
// It contains unit tests for models and schemas, plus integration tests for API endpoints.
func TestSuite(t *testing.T) {
	// Initialize API and database for testing
	a := assert.New(t)
	req := require.New(t)
	api := internal.NewBookCatalogAPI()
	db, err := database.NewDatabase()
	req.NoError(err)
	db = getSyncDB(t, db)
	defer db.Close()
	dropAllTables(t, db)
	testAPIEndpoints(t, api, db)
}

func testAPIEndpoints(t *testing.T, api *internal.BookCatalogAPI, db *sqlx.DB) {
	// Setup
	client := &httptest.Server{
		Config: &http.Server{Handler: api.Router()},
	}
	defer client.Close()

	// Test root endpoint
	testReadRoot(t, client)

	// Test health check endpoint
	testHealthCheck(t, client)

	// Test create book endpoint
	testCreateBook(t, client, db)
	testCreateBookWithoutSummary(t, client, db)
	testCreateBookValidationError(t, client, db)
	testCreateDuplicateBook(t, client, db)
	testCreateBooksSameTitleDifferentAuthors(t, client, db)
	testFullCRUDWorkflow(t, client, db)

	// Test get books endpoint
	testGetBooksEmpty(t, client, db)
	testGetBooksWithData(t, client, db)
	testGetBooksWithPagination(t, client, db)

	// Test get book by ID endpoint
	testGetBookByID(t, client, db)
	testGetBookNotFound(t, client, db)

	// Test update book endpoint
	testUpdateBook(t, client, db)
	testUpdateBookNotFound(t, client, db)
	testUpdateBookValidationError(t, client, db)

	// Test delete book endpoint
	testDeleteBook(t, client, db)
	testDeleteBookNotFound(t, client, db)
}

func testReadRoot(t *testing.T, client *httptest.Server) {
	resp, err := client.Client().Get(client.URL + "/")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var data map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	require.NoError(t, err)
	assert.Equal(t, "Welcome to Book Catalog API", data["message"])
	assert.Equal(t, "1.0.0", data["version"])
	assert.Contains(t, data, "docs_url")
}

func testHealthCheck(t *testing.T, client *httptest.Server) {
	resp, err := client.Client().Get(client.URL + "/health")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var data map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	require.NoError(t, err)
	assert.Equal(t, "healthy", data["status"])
	assert.Equal(t, "book-catalog-api", data["service"])
}

func testCreateBook(t *testing.T, client *httptest.Server, db *sqlx.DB) {
	bookData := schemas.BookCreate{
		Title:          "Test Book",
		Author:         "Test Author",
		PublishedYear:  2023,
		Summary:        &"A test book summary",
	}
	req := require.New(t)
	req.NoError(bookData.Validate())

	resp, err := client.Client().Post(client.URL+"/books/", "application/json", readerOfJSON(t, bookData))
	req.NoError(err)
	req.Equal(http.StatusCreated, resp.StatusCode)

	var createdBook schemas.BookResponse
	err = json.NewDecoder(resp.Body).Decode(&createdBook)
	req.NoError(err)
	req.Equal(bookData.Title, createdBook.Title)
	req.Equal(bookData.Author, createdBook.Author)
	req.Equal(bookData.PublishedYear, createdBook.PublishedYear)
	req.Equal(bookData.Summary, &createdBook.Summary)
	req.NotNil(createdBook.ID)
}

func testCreateBookWithoutSummary(t *testing.T, client *httptest.Server, db *sqlx.DB) {
	bookData := schemas.BookCreate{
		Title:          "Book Without Summary",
		Author:         "Author",
		PublishedYear:  2023,
	}
	req := require.New(t)
	req.NoError(bookData.Validate())

	resp, err := client.Client().Post(client.URL+"/books/", "application/json", readerOfJSON(t, bookData))
	req.NoError(err)
	req.Equal(http.StatusCreated, resp.StatusCode)

	var createdBook schemas.BookResponse
	err = json.NewDecoder(resp.Body).Decode(&createdBook)
	req.NoError(err)
	req.Equal(bookData.Title, createdBook.Title)
	req.Equal(bookData.Author, createdBook.Author)
	req.Equal(bookData.PublishedYear, createdBook.PublishedYear)
	req.Nil(createdBook.Summary)
}

func testCreateBookValidationError(t *testing.T, client *httptest.Server, db *sqlx.DB) {
	// Missing required field
	bookData := schemas.BookCreate{
		Title: "Test Book",
	}
	req := require.New(t)
	req.Error(bookData.Validate())

	resp, err := client.Client().Post(client.URL+"/books/", "application/json", readerOfJSON(t, bookData))
	req.NoError(err)
	req.Equal(http.StatusUnprocessableEntity, resp.StatusCode)

	// Invalid published year
	bookData = schemas.BookCreate{
		Title:          "Test Book",
		Author:         "Test Author",
		PublishedYear:  999,
	}
	req.Error(bookData.Validate())

	resp, err = client.Client().Post(client.URL+"/books/", "application/json", readerOfJSON(t, bookData))
	req.NoError(err)
	req.Equal(http.StatusUnprocessableEntity, resp.StatusCode)
}

func testGetBooksEmpty(t *testing.T, client *httptest.Server, db *sqlx.DB) {
	resp, err := client.Client().Get(client.URL + "/books/")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var books []schemas.BookResponse
	err = json.NewDecoder(resp.Body).Decode(&books)
	require.NoError(t, err)
	assert.Len(t, books, 0)
}