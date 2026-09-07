package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"migrated-app/internal/database"
	"migrated-app/internal/model"
	"migrated-app/internal/schemas"
	"migrated-app/internal/main"
	"migrated-app/internal/tests/conftest.go"
)

const (
	rootURL         = "/"
	healthURL       = "/health"
	booksURL        = "/books/"
	docsURL         = "/docs"
	welcomeMessage  = "Welcome to Book Catalog API"
	version         = "1.0.0"
	healthyStatus   = "healthy"
	serviceName     = "book-catalog-api"
	invalidYear     = 999
)

var db database.DBInterface

func setupTest(t *testing.T) {
	db = conftest.NewMockDB(t)
	main.SetDB(db)
}

func teardownTest() {
	db.Close()
}

func TestClient(t *testing.T) *httptest.Client {
	t.Helper()

	server := httptest.NewServer(main.buildRouter())
	return server.Client()
}

func TestRootEndpoint(t *testing.T) {
	setupTest(t)
	defer teardownTest()

	tests := []struct {
		name      string
		url       string
		statusCode int
	}{
		{"root endpoint", rootURL, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := TestClient(t)
			resp, err := client.Get(tt.url)
			assert.NoError(t, err)
			assert.Equal(t, tt.statusCode, resp.StatusCode)

			body, err := ioutil.ReadAll(resp.Body)
			assert.NoError(t, err)
			defer resp.Body.Close()

			var data map[string]interface{}
			err = json.Unmarshal(body, &data)
			assert.NoError(t, err)

			assert.Equal(t, welcomeMessage, data["message"])
			assert.Equal(t, version, data["version"])
			assert.Contains(t, data, "docs_url")
		})
	}
}

func TestHealthCheck(t *testing.T) {
	setupTest(t)
	defer teardownTest()

	tests := []struct {
		name      string
		url       string
		statusCode int
	}{
		{"health check", healthURL, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := TestClient(t)
			resp, err := client.Get(tt.url)
			assert.NoError(t, err)
			assert.Equal(t, tt.statusCode, resp.StatusCode)

			body, err := ioutil.ReadAll(resp.Body)
			assert.NoError(t, err)
			defer resp.Body.Close()

			var data map[string]interface{}
			err = json.Unmarshal(body, &data)
			assert.NoError(t, err)

			assert.Equal(t, healthyStatus, data["status"])
			assert.Equal(t, serviceName, data["service"])
		})
	}
}

func TestCreateBook(t *testing.T) {
	setupTest(t)
	defer teardownTest()

	tests := []struct {
		name          string
		input         model.Book
		expectedError bool
		expectedCode  int
	}{
		{"valid book data", model.Book{Title: "Test Title", Author: "Test Author", PublishedYear: 2023, Summary: "Test Summary"}, false, http.StatusCreated},
		{"missing required fields", model.Book{PublishedYear: 2023, Summary: "Test Summary"}, true, http.StatusUnprocessableEntity},
		{"invalid year", model.Book{Title: "Test Title", Author: "Test Author", PublishedYear: invalidYear, Summary: "Test Summary"}, true, http.StatusUnprocessableEntity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := TestClient(t)
			data, err := json.Marshal(tt.input)
			assert.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, booksURL, bytes.NewBuffer(data))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			if !tt.expectedError {
				body, err := ioutil.ReadAll(resp.Body)
				assert.NoError(t, err)
				defer resp.Body.Close()

				var book model.Book
				err = json.Unmarshal(body, &book)
				assert.NoError(t, err)

				assert.NotEmpty(t, book.ID)
				assert.Equal(t, tt.input.Title, book.Title)
				assert.Equal(t, tt.input.Author, book.Author)
				assert.Equal(t, tt.input.PublishedYear, book.PublishedYear)
				assert.Equal(t, tt.input.Summary, book.Summary)
			}
		})
	}
}

func TestGetBooks(t *testing.T) {
	setupTest(t)
	defer teardownTest()

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedCount  int
	}{
		{"empty database", booksURL, http.StatusOK, 0},
		{"with data", fmt.Sprintf("%s?skip=0&limit=5", booksURL), http.StatusOK, 5},
		{"with pagination", fmt.Sprintf("%s?skip=5&limit=5", booksURL), http.StatusOK, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := TestClient(t)
			resp, err := client.Get(tt.url)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			body, err := ioutil.ReadAll(resp.Body)
			assert.NoError(t, err)
			defer resp.Body.Close()

			var books []model.Book
			err = json.Unmarshal(body, &books)
			assert.NoError(t, err)

			assert.Len(t, books, tt.expectedCount)
		})
	}
}

func TestGetBookByID(t *testing.T) {
	setupTest(t)
	defer teardownTest()

	book := model.Book{Title: "Test Title", Author: "Test Author", PublishedYear: 2023, Summary: "Test Summary"}
	db.Create(&book)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{"valid id", fmt.Sprintf("%s%d", booksURL, book.ID), http.StatusOK},
		{"invalid id", fmt.Sprintf("%s%d", booksURL, -1), http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := TestClient(t)
			resp, err := client.Get(tt.url)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				body, err := ioutil.ReadAll(resp.Body)
				assert.NoError(t, err)
				defer resp.Body.Close()

				var data map[string]interface{}
				err = json.Unmarshal(body, &data)
				assert.NoError(t, err)

				assert.Equal(t, book.Title, data["title"])
				assert.Equal(t, book.Author, data["author"])
				assert.Equal(t, book.PublishedYear, data["published_year"])
				assert.Equal(t, book.Summary, data["summary"])
				assert.NotEmpty(t, data["id"])
			}
		})
	}
}

func TestUpdateBook(t *testing.T) {
	setupTest(t)
	defer teardownTest()

	book := model.Book{Title: "Test Title", Author: "Test Author", PublishedYear: 2023, Summary: "Test Summary"}
	db.Create(&book)

	updateData := model.Book{Title: "Updated Title", Author: "Updated Author", PublishedYear: 2024, Summary: "Updated Summary"}

	tests := []struct {
		name          string
		bookID        int
		input         model.Book
		expectedError bool
		expectedCode  int
	}{
		{"valid update", book.ID, updateData, false, http.StatusOK},
		{"invalid id", -1, updateData, true, http.StatusNotFound},
		{"invalid year", book.ID, model.Book{Title: "Invalid Year", Author: "Invalid Author", PublishedYear: invalidYear, Summary: "Invalid Summary"}, true, http.StatusUnprocessableEntity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := TestClient(t)
			data, err := json.Marshal(tt.input)
			assert.NoError(t, err)

			req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s%d", booksURL, tt.bookID), bytes.NewBuffer(data))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			if !tt.expectedError {
				body, err := ioutil.ReadAll(resp.Body)
				assert.NoError(t, err)
				defer resp.Body.Close()

				var updatedBook model.Book
				err = json.Unmarshal(body, &updatedBook)
				assert.NoError(t, err)

				assert.Equal(t, tt.input.Title, updatedBook.Title)
				assert.Equal(t, tt.input.Author, updatedBook.Author)
				assert.Equal(t, tt.input.PublishedYear, updatedBook.PublishedYear)
				assert.Equal(t, tt.input.Summary, updatedBook.Summary)
				assert.NotEmpty(t, updatedBook.ID)
			}
		})
	}
}

func TestDeleteBook(t *testing.T) {
	setupTest(t)
	defer teardownTest()

	book := model.Book{Title: "Test Title", Author: "Test Author", PublishedYear: 2023, Summary: "Test Summary"}
	db.Create(&book)

	tests := []struct {
		name           string
		bookID         int
		expectedStatus int
	}{
		{"valid id", book.ID, http.StatusNoContent},
		{"invalid id", -1, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := TestClient(t)
			resp, err := client.Delete(fmt.Sprintf("%s%d", booksURL, tt.bookID))
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusNoContent {
				resp, err := client.Get(fmt.Sprintf("%s%d", booksURL, tt.bookID))
				assert.NoError(t, err)
				assert.Equal(t, http.StatusNotFound, resp.StatusCode)
			}
		})
	}
}