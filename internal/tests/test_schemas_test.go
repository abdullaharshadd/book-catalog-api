package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type BookCreate struct {
	Title         string
	Author        string
	PublishedYear int
	Summary       *string
}

func (b *BookCreate) Validate() error {
	if b.Title == "" || b.Title == " " {
		return ValidationError{"Title cannot be empty"}
	}
	if b.Author == "" || b.Author == " " {
		return ValidationError{"Author cannot be empty"}
	}
	if b.PublishedYear < 1000 {
		return ValidationError{"Published year must be after year 1000"}
	}
	if b.PublishedYear > time.Now().Year() {
		return ValidationError{"cannot be in the future"}
	}
	if len(b.Title) > 255 {
		return ValidationError{"ensure this value has at most 255 characters"}
	}
	if len(b.Author) > 255 {
		return ValidationError{"ensure this value has at most 255 characters"}
	}
	if b.Summary != nil && len(*b.Summary) > 2000 {
		return ValidationError{"ensure this value has at most 2000 characters"}
	}
	return nil
}

type ValidationError struct {
	Message string
}

func (ve ValidationError) Error() string {
	return ve.Message
}

type BookUpdate struct {
	Title         *string
	Author        *string
	PublishedYear *int
	Summary       *string
}

func (b *BookUpdate) Validate() error {
	if b.Title != nil && (*b.Title == "" || *b.Title == " ") {
		return ValidationError{"Title cannot be empty"}
	}
	if b.PublishedYear != nil && (*b.PublishedYear < 1000) {
		return ValidationError{"Published year must be after year 1000"}
	}
	return nil
}

type BookResponse struct {
	ID            int
	Title         string
	Author        string
	PublishedYear int
	Summary       *string
}

func (b *BookResponse) Validate() error {
	if b.ID <= 0 {
		return ValidationError{"id"}
	}
	if b.Title == "" || b.Title == " " {
		return ValidationError{"Title cannot be empty"}
	}
	if b.Author == "" || b.Author == " " {
		return ValidationError{"Author cannot be empty"}
	}
	if b.PublishedYear < 1000 {
		return ValidationError{"Published year must be after year 1000"}
	}
	if b.PublishedYear > time.Now().Year() {
		return ValidationError{"cannot be in the future"}
	}
	if len(b.Title) > 255 {
		return ValidationError{"ensure this value has at most 255 characters"}
	}
	if len(b.Author) > 255 {
		return ValidationError{"ensure this value has at most 255 characters"}
	}
	if b.Summary != nil && len(*b.Summary) > 2000 {
		return ValidationError{"ensure this value has at most 2000 characters"}
	}
	return nil
}

var bookCreateTests = []struct {
	name          string
	input         BookCreate
	expectedError string
}{
	{"valid inputs", BookCreate{Title: "Test Title", Author: "Test Author", PublishedYear: 2023, Summary: strPtr("Summary")}, ""},
	{"missing summary", BookCreate{Title: "Test Title", Author: "Test Author", PublishedYear: 2023, Summary: nil}, ""},
	{"whitespace in inputs", BookCreate{Title: " Test Title ", Author: " Test Author ", PublishedYear: 2023, Summary: strPtr(" Summary ")}, ""},
	{"empty title", BookCreate{Title: "", Author: "Test Author", PublishedYear: 2023, Summary: nil}, "Title cannot be empty"},
	{"empty author", BookCreate{Title: "Test Title", Author: "", PublishedYear: 2023, Summary: nil}, "Author cannot be empty"},
	{"year before 1000", BookCreate{Title: "Test Title", Author: "Test Author", PublishedYear: 999, Summary: nil}, "Published year must be after year 1000"},
	{"future year", BookCreate{Title: "Test Title", Author: "Test Author", PublishedYear: time.Now().Year() + 1, Summary: nil}, "cannot be in the future"},
	{"long title", BookCreate{Title: "T" + "est Title".Repeat(25), Author: "Test Author", PublishedYear: 2023, Summary: nil}, "ensure this value has at most 255 characters"},
	{"long author", BookCreate{Title: "Test Title", Author: "T" + "est Author".Repeat(25), PublishedYear: 2023, Summary: nil}, "ensure this value has at most 255 characters"},
	{"long summary", BookCreate{Title: "Test Title", Author: "Test Author", PublishedYear: 2023, Summary: strPtr("S" + "ummary".Repeat(200))}, "ensure this value has at most 2000 characters"},
}

func TestBookCreate(t *testing.T) {
	for _, tt := range bookCreateTests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if err != nil {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

var bookUpdateTests = []struct {
	name          string
	input         BookUpdate
	expectedError string
}{
	{"partial update", BookUpdate{Title: strPtr("Updated Title"), Author: nil, PublishedYear: nil, Summary: nil}, ""},
	{"no update data", BookUpdate{Title: nil, Author: nil, PublishedYear: nil, Summary: nil}, ""},
	{"empty title", BookUpdate{Title: strPtr(""), Author: nil, PublishedYear: nil, Summary: nil}, "Title cannot be empty"},
	{"year before 1000", BookUpdate{Title: nil, Author: nil, PublishedYear: intPtr(999), Summary: nil}, "Published year must be after year 1000"},
}

func TestBookUpdate(t *testing.T) {
	for _, tt := range bookUpdateTests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if err != nil {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

var bookResponseTests = []struct {
	name          string
	input         BookResponse
	expectedError string
}{
	{"valid response", BookResponse{ID: 1, Title: "Test Title", Author: "Test Author", PublishedYear: 2023, Summary: strPtr("Summary")}, ""},
	{"missing id", BookResponse{ID: 0, Title: "Test Title", Author: "Test Author", PublishedYear: 2023, Summary: strPtr("Summary")}, "id"},
}

func TestBookResponse(t *testing.T) {
	for _, tt := range bookResponseTests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if err != nil {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func (s string) Repeat(count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}