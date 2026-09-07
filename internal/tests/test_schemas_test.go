// Package tests contains the unit tests for models and schemas, and integration tests for API endpoints of the Book Catalog API.
package tests

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"migrated-app/internal/schemas"
)

func TestBookCreate(t *testing.T) {
	testCases := []struct {
		name           string
		input          schemas.BookCreate
		expectedErr    bool
		expectedOutput schemas.BookCreate
	}{
		{"valid book create", schemas.BookCreate{Title: "Valid Book", Author: "Valid Author", PublishedYear: 2023, Summary: &"A valid book summary"}, false, schemas.BookCreate{Title: "Valid Book", Author: "Valid Author", PublishedYear: 2023, Summary: &"A valid book summary"}},
		{"book create without summary", schemas.BookCreate{Title: "Book Without Summary", Author: "Author", PublishedYear: 2023}, false, schemas.BookCreate{Title: "Book Without Summary", Author: "Author", PublishedYear: 2023, Summary: nil}},
		{"whitespace stripping", schemas.BookCreate{Title: "  Whitespace Book  ", Author: "  Whitespace Author  ", PublishedYear: 2023, Summary: &"  Whitespace summary  "}, false, schemas.BookCreate{Title: "Whitespace Book", Author: "Whitespace Author", PublishedYear: 2023, Summary: &"Whitespace summary"}},
		{"empty summary becomes nil", schemas.BookCreate{Title: "Book", Author: "Author", PublishedYear: 2023, Summary: &"   "}, false, schemas.BookCreate{Title: "Book", Author: "Author", PublishedYear: 2023, Summary: nil}},
		{"missing required fields - title", schemas.BookCreate{Author: "Author", PublishedYear: 2023}, true, schemas.BookCreate{}},
		{"missing required fields - author", schemas.BookCreate{Title: "Title", PublishedYear: 2023}, true, schemas.BookCreate{}},
		{"missing required fields - published_year", schemas.BookCreate{Title: "Title", Author: "Author"}, true, schemas.BookCreate{}},
		{"empty title validation", schemas.BookCreate{Title: "", Author: "Author", PublishedYear: 2023}, true, schemas.BookCreate{}},
		{"empty author validation", schemas.BookCreate{Title: "Title", Author: "", PublishedYear: 2023}, true, schemas.BookCreate{}},
		{"future published year validation", schemas.BookCreate{Title: "Title", Author: "Author", PublishedYear: time.Now().Year() + 1}, true, schemas.BookCreate{}},
		{"past published year validation", schemas.BookCreate{Title: "Title", Author: "Author", PublishedYear: 999}, true, schemas.BookCreate{}},
		{"title length validation", schemas.BookCreate{Title: "A" + string(make([]byte, 256)), Author: "Author", PublishedYear: 2023}, true, schemas.BookCreate{}},
		{"author length validation", schemas.BookCreate{Title: "Title", Author: "B" + string(make([]byte, 256)), PublishedYear: 2023}, true, schemas.BookCreate{}},
		{"summary length validation", schemas.BookCreate{Title: "Title", Author: "Author", PublishedYear: 2023, Summary: &("C" + string(make([]byte, 2001)))}, true, schemas.BookCreate{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()
			if tc.expectedErr {
				assert.NotNil(t, err, "Expected an error but got nil")
				if tc.name == "missing required fields - title" {
					assert.EqualError(t, err, "Title cannot be empty")
				} else if tc.name == "missing required fields - author" {
					assert.EqualError(t, err, "Author cannot be empty")
				} else if tc.name == "missing required fields - published_year" {
					assert.EqualError(t, err, "Published year must be after year 1000")
				} else if tc.name == "future published year validation" {
					assert.Contains(t, err.Error(), "cannot be in the future")
				} else if tc.name == "past published year validation" {
					assert.EqualError(t, err, "Published year must be after year 1000")
				} else if tc.name == "title length validation" {
					assert.Contains(t, err.Error(), "ensure this value has at most 255 characters")
				} else if tc.name == "author length validation" {
					assert.Contains(t, err.Error(), "ensure this value has at most 255 characters")
				} else if tc.name == "summary length validation" {
					assert.Contains(t, err.Error(), "ensure this value has at most 2000 characters")
				}
			} else {
				assert.Nil(t, err, "Expected no error but got %v", err)
				assert.Equal(t, tc.expectedOutput, tc.input, "Expected output mismatch")
			}
		})
	}
}

func TestBookUpdate(t *testing.T) {
	testCases := []struct {
		name           string
		input          schemas.BookUpdate
		expectedErr    bool
		expectedOutput schemas.BookUpdate
	}{
		{"valid partial update", schemas.BookUpdate{Title: "Updated Title", PublishedYear: &2024}, false, schemas.BookUpdate{Title: "Updated Title", PublishedYear: &2024, Author: nil, Summary: nil}},
		{"empty update", schemas.BookUpdate{}, false, schemas.BookUpdate{}},
		{"empty title validation", schemas.BookUpdate{Title: &""}, true, schemas.BookUpdate{}},
		{"past published year validation", schemas.BookUpdate{PublishedYear: &999}, true, schemas.BookUpdate{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()
			if tc.expectedErr {
				assert.NotNil(t, err, "Expected an error but got nil")
				if tc.name == "empty title validation" {
					assert.EqualError(t, err, "Title cannot be empty")
				} else if tc.name == "past published year validation" {
					assert.EqualError(t, err, "Published year must be after year 1000")
				}
			} else {
				assert.Nil(t, err, "Expected no error but got %v", err)
				assert.Equal(t, tc.expectedOutput, tc.input, "Expected output mismatch")
			}
		})
	}
}

func TestBookResponse(t *testing.T) {
	testCases := []struct {
		name           string
		input          schemas.BookResponse
		expectedErr    bool
		expectedOutput schemas.BookResponse
	}{
		{"valid book response", schemas.BookResponse{ID: 1, Title: "Response Book", Author: "Response Author", PublishedYear: 2023, Summary: &"Response summary"}, false, schemas.BookResponse{ID: 1, Title: "Response Book", Author: "Response Author", PublishedYear: 2023, Summary: &"Response summary"}},
		{"book response missing id", schemas.BookResponse{Title: "Title", Author: "Author", PublishedYear: 2023}, true, schemas.BookResponse{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()
			if tc.expectedErr {
				assert.NotNil(t, err, "Expected an error but got nil")
				if tc.name == "book response missing id" {
					assert.EqualError(t, err, "id")
				}
			} else {
				assert.Nil(t, err, "Expected no error but got %v", err)
				assert.Equal(t, tc.expectedOutput, tc.input, "Expected output mismatch")
			}
		})
	}
}