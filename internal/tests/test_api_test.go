package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	name         string
	method       string
	path         string
	body         []byte
	expectedCode int
	expectedBody interface{}
}

func TestReadRoot(t *testing.T) {
	testCases := []testCase{
		{
			name:         "no input provided",
			method:       http.MethodGet,
			path:         "/",
			body:         nil,
			expectedCode: http.StatusOK,
			expectedBody: map[string]string{
				"message":   "Welcome to our API",
				"version":   "v1.0.0",
				"docs_url":  "https://example.com/docs",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			readRootHandler(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			var respBody map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &respBody)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedBody, respBody)
		})
	}
}

func TestHealthCheck(t *testing.T) {
	testCases := []testCase{
		{
			name:         "no input provided",
			method:       http.MethodGet,
			path:         "/health",
			body:         nil,
			expectedCode: http.StatusOK,
			expectedBody: map[string]string{
				"status": "OK",
				"service": "BookAPI",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			healthCheckHandler(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			var respBody map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &respBody)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedBody, respBody)
		})
	}
}

func TestCreateBook(t *testing.T) {
	testCases := []testCase{
		{
			name:         "valid book data provided",
			method:       http.MethodPost,
			path:         "/books/",
			body:         []byte(`{"title":"Test Book","author":"John Doe","published_year":2023,"summary":"This is a test book."}`),
			expectedCode: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"id":           1,
				"title":        "Test Book",
				"author":       "John Doe",
				"published_year": 2023,
				"summary":      "This is a test book.",
			},
		},
		{
			name:         "missing required fields",
			method:       http.MethodPost,
			path:         "/books/",
			body:         []byte(`{"title":"Test Book","published_year":2023}`),
			expectedCode: http.StatusUnprocessableEntity,
			expectedBody: map[string]string{},
		},
		{
			name:         "invalid published year",
			method:       http.MethodPost,
			path:         "/books/",
			body:         []byte(`{"title":"Test Book","author":"John Doe","published_year":"invalid","summary":"This is a test book."}`),
			expectedCode: http.StatusUnprocessableEntity,
			expectedBody: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("Content-Type", "application/json")
			req.Body = httptest.NopCloser(jsonReader(tc.body))
			w := httptest.NewRecorder()
			createBookHandler(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			var respBody map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &respBody)
			assert.NoError(t, err)
			if tc.expectedCode == http.StatusCreated {
				assert.Equal(t, tc.expectedBody["title"], respBody["title"])
				assert.Equal(t, tc.expectedBody["author"], respBody["author"])
				assert.Equal(t, tc.expectedBody["published_year"], respBody["published_year"])
				assert.Equal(t, tc.expectedBody["summary"], respBody["summary"])
			} else {
				assert.Equal(t, tc.expectedBody, respBody)
			}
		})
	}
}

func TestGetBooks(t *testing.T) {
	testCases := []testCase{
		{
			name:         "database is empty",
			method:       http.MethodGet,
			path:         "/books/",
			body:         nil,
			expectedCode: http.StatusOK,
			expectedBody: []interface{}{},
		},
		{
			name:         "database has data",
			method:       http.MethodGet,
			path:         "/books/",
			body:         nil,
			expectedCode: http.StatusOK,
			expectedBody: []interface{}{
				map[string]interface{}{
					"id":           1,
					"title":        "Book One",
					"author":       "Author One",
					"published_year": 2020,
					"summary":      "Summary One",
				},
				map[string]interface{}{
					"id":           2,
					"title":        "Book Two",
					"author":       "Author Two",
					"published_year": 2021,
					"summary":      "Summary Two",
				},
			},
		},
		{
			name:         "pagination parameters are provided",
			method:       http.MethodGet,
			path:         "/books/?skip=1&limit=1",
			body:         nil,
			expectedCode: http.StatusOK,
			expectedBody: []interface{}{
				map[string]interface{}{
					"id":           2,
					"title":        "Book Two",
					"author":       "Author Two",
					"published_year": 2021,
					"summary":      "Summary Two",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			getBooksHandler(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			var respBody []interface{}
			err := json.Unmarshal(w.Body.Bytes(), &respBody)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedBody, respBody)
		})
	}
}

func TestGetBookByID(t *testing.T) {
	testCases := []testCase{
		{
			name:         "valid book ID provided",
			method:       http.MethodGet,
			path:         "/books/1",
			body:         nil,
			expectedCode: http.StatusOK,
			expectedBody: map[string]interface{}{
				"id":           1,
				"title":        "Book One",
				"author":       "Author One",
				"published_year": 2020,
				"summary":      "Summary One",
			},
		},
		{
			name:         "book ID does not exist",
			method:       http.MethodGet,
			path:         "/books/999",
			body:         nil,
			expectedCode: http.StatusNotFound,
			expectedBody: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			getBookByIDHandler(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			var respBody map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &respBody)
			assert.NoError(t, err)
			if tc.expectedCode == http.StatusOK {
				assert.Equal(t, tc.expectedBody, respBody)
			} else {
				assert.Equal(t, tc.expectedBody, respBody)
			}
		})
	}
}

func TestUpdateBook(t *testing.T) {
	testCases := []testCase{
		{
			name:         "valid book ID and update data provided",
			method:       http.MethodPut,
			path:         "/books/1",
			body:         []byte(`{"title":"Updated Title","author":"Updated Author","published_year":2023,"summary":"Updated Summary."}`),
			expectedCode: http.StatusOK,
			expectedBody: map[string]interface{}{
				"id":           1,
				"title":        "Updated Title",
				"author":       "Updated Author",
				"published_year": 2023,
				"summary":      "Updated Summary.",
			},
		},
		{
			name:         "book ID does not exist",
			method:       http.MethodPut,
			path:         "/books/999",
			body:         []byte(`{"title":"Updated Title","author":"Updated Author","published_year":2023,"summary":"Updated Summary."}`),
			expectedCode: http.StatusNotFound,
			expectedBody: map[string]string{},
		},
		{
			name:         "provided invalid data",
			method:       http.MethodPut,
			path:         "/books/1",
			body:         []byte(`{"title":"Updated Title","author":"Updated Author","published_year":"invalid","summary":"Updated Summary."}`),
			expectedCode: http.StatusUnprocessableEntity,
			expectedBody: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("Content-Type", "application/json")
			req.Body = httptest.NopCloser(jsonReader(tc.body))
			w := httptest.NewRecorder()
			updateBookHandler(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			var respBody map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &respBody)
			assert.NoError(t, err)
			if tc.expectedCode == http.StatusOK {
				assert.Equal(t, tc.expectedBody, respBody)
			} else {
				assert.Equal(t, tc.expectedBody, respBody)
			}
		})
	}
}

func TestDeleteBook(t *testing.T) {
	testCases := []testCase{
		{
			name:         "valid book ID provided",
			method:       http.MethodDelete,
			path:         "/books/1",
			body:         nil,
			expectedCode: http.StatusNoContent,
			expectedBody: map[string]interface{}{},
		},
		{
			name:         "book ID does not exist",
			method:       http.MethodDelete,
			path:         "/books/999",
			body:         nil,
			expectedCode: http.StatusNotFound,
			expectedBody: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			deleteBookHandler(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			var respBody map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &respBody)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedBody, respBody)
		})
	}
}

func jsonReader(b []byte) *json.Reader {
	return json.NewReader(nil)
}