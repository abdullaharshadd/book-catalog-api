package tests

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"internal/schemas"
)

// TestSuite tests the schemas for the BookCatalogAPI
func TestSuite(t *testing.T) {
	testBookCreate(t)
	testBookUpdate(t)
	testBookResponse(t)
}

func testBookCreate(t *testing.T) {
	// Test creating a valid book
	bookData := schemas.BookCreate{
		Title:         "Valid Book",
		Author:       "Valid Author",
		PublishedYear: 2023,
		Summary:      "A valid book summary",
	}
	assert.NoError(t, bookData.Validate(context.Background()))
	assert.Equal(t, "Valid Book", bookData.Title)
	assert.Equal(t, "Valid Author", bookData.Author)
	assert.Equal(t, 2023, bookData.PublishedYear)
	assert.Equal(t, "A valid book summary", bookData.Summary)

	// Test creating a book without summary (optional)
	bookData = schemas.BookCreate{
		Title:         "Book Without Summary",
		Author:       "Author",
		PublishedYear: 2023,
	}
	assert.NoError(t, bookData.Validate(context.Background()))
	assert.Equal(t, "Book Without Summary", bookData.Title)
	assert.Equal(t, "Author", bookData.Author)
	assert.Equal(t, 2023, bookData.PublishedYear)
	assert.Equal(t, (*string)(nil), bookData.Summary)

	// Test that whitespace is stripped from title and author
	bookData = schemas.BookCreate{
		Title:         "  Whitespace Book  ",
		Author:       "  Whitespace Author  ",
		PublishedYear: 2023,
		Summary:      "  Whitespace summary  ",
	}
	assert.NoError(t, bookData.Validate(context.Background()))
	assert.Equal(t, "Whitespace Book", *bookData.Title)
	assert.Equal(t, "Whitespace Author", *bookData.Author)
	assert.Equal(t, "Whitespace summary", *bookData.Summary)

	// Test that empty summary becomes nil
	bookData = schemas.BookCreate{
		Title:         "Book",
		Author:       "Author",
		PublishedYear: 2023,
		Summary:      "   ", // Only whitespace
	}
	assert.NoError(t, bookData.Validate(context.Background()))
	assert.Equal(t, (*string)(nil), bookData.Summary)

	// Test validation errors for missing required fields
	cases := []struct {
		bookData schemas.BookCreate
		missingField string
		expectError bool
	}{
		{bookData: schemas.BookCreate{Author: "Author", PublishedYear: 2023}, missingField: "title", expectError: true},
		{bookData: schemas.BookCreate{Title: "Title", PublishedYear: 2023}, missingField: "author", expectError: true},
		{bookData: schemas.BookCreate{Title: "Title", Author: "Author"}, missingField: "published_year", expectError: true},
	}
	for _, tc := range cases {
		if tc.expectError {
			assert.Error(t, tc.bookData.Validate(context.Background()))
			assert.Contains(t, tc.bookData.Validate(context.Background()).Error(), tc.missingField)
		} else {
			assert.NoError(t, tc.bookData.Validate(context.Background()))
		}
	}

	// Test validation for empty title
	cases = []struct {
		bookData schemas.BookCreate
		expectError bool
	}{
		{bookData: schemas.BookCreate{Title: "", Author: "Author", PublishedYear: 2023}, expectError: true},
		{bookData: schemas.BookCreate{Title: "   ", Author: "Author", PublishedYear: 2023}, expectError: true},
	}
	for _, tc := range cases {
		if tc.expectError {
			assert.Error(t, tc.bookData.Validate(context.Background()))
			assert.Contains(t, tc.bookData.Validate(context.Background()).Error(), "Title cannot be empty")
		} else {
			assert.NoError(t, tc.bookData.Validate(context.Background()))
		}
	}

	// Test validation for empty author
	cases = []struct {
		bookData schemas.BookCreate
		expectError bool
	}{
		{bookData: schemas.BookCreate{Title: "Title", Author: "", PublishedYear: 2023}, expectError: true},
		{bookData: schemas.BookCreate{Title: "Title", Author: "   ", PublishedYear: 2023}, expectError: true},
	}
	for _, tc := range cases {
		if tc.expectError {
			assert.Error(t, tc.bookData.Validate(context.Background()))
			assert.Contains(t, tc.bookData.Validate(context.Background()).Error(), "Author cannot be empty")
		} else {
			assert.NoError(t, tc.bookData.Validate(context.Background()))
		}
	}

	// Test validation for published year
	cases = []struct {
		bookData schemas.BookCreate
		expectError bool
		expectedYear int
	}{
		{bookData: schemas.BookCreate{Title: "Title", Author: "Author", PublishedYear: 999}, expectError: true, expectedYear: 0},
		{bookData: schemas.BookCreate{Title: "Title", Author: "Author", PublishedYear: 2024}, expectError: true, expectedYear: 0},
		{bookData: schemas.BookCreate{Title: "Title", Author: "Author", PublishedYear: 1000}, expectError: false, expectedYear: 1000},
		{bookData: schemas.BookCreate{Title: "Title", Author: "Author", PublishedYear: 2023}, expectError: false, expectedYear: 2023},
	}
	for _, tc := range cases {
		if tc.expectError {
			assert.Error(t, tc.bookData.Validate(context.Background()))
			if tc.expectedYear == 999 {
				assert.Contains(t, tc.bookData.Validate(context.Background()).Error(), "Published year must be after year 1000")
			} else {
				assert.Contains(t, tc.bookData.Validate(context.Background()).Error(), "cannot be in the future")
			}
		} else {
			assert.NoError(t, tc.bookData.Validate(context.Background()))
			assert.Equal(t, tc.expectedYear, tc.bookData.PublishedYear)
		}
	}

	// Test title length validation
	bookData = schemas.BookCreate{
		Title:         "A" + "A"*255,
		Author:       "Author",
		PublishedYear: 2023,
	}
	assert.Error(t, bookData.Validate(context.Background()))
	assert.Contains(t, bookData.Validate(context.Background()).Error(), "ensure this value has at most 255 characters")

	// Test author length validation
	bookData = schemas.BookCreate{
		Title:         "Title",
		Author:       "B" + "B"*255,
		PublishedYear: 2023,
	}
	assert.Error(t, bookData.Validate(context.Background()))
	assert.Contains(t, bookData.Validate(context.Background()).Error(), "ensure this value has at most 255 characters")

	// Test summary length validation
	bookData = schemas.BookCreate{
		Title:         "Title",
		Author:       "Author",
		PublishedYear: 2023,
		Summary:      &("C" + "C"*2000),
	}
	assert.Error(t, bookData.Validate(context.Background()))
	assert.Contains(t, bookData.Validate(context.Background()).Error(), "ensure this value has at most 2000 characters")
}

func testBookUpdate(t *testing.T) {
	// Test updating only some fields
	updateData := schemas.BookUpdate{
		Title:         "Updated Title",
		PublishedYear: 2024,
	}
	assert.NoError(t, updateData.Validate(context.Background()))
	assert.Equal(t, "Updated Title", *updateData.Title)
	assert.Equal(t, (*string)(nil), updateData.Author)
	assert.Equal(t, 2024, updateData.PublishedYear)
	assert.Equal(t, (*string)(nil), updateData.Summary)

	// Test update with no fields provided
	updateData = schemas.BookUpdate{}
	assert.NoError(t, updateData.Validate(context.Background()))
	assert.Equal(t, (*string)(nil), updateData.Title)
	assert.Equal(t, (*string)(nil), updateData.Author)
	assert.Equal(t, (*int)(nil), updateData.PublishedYear)
	assert.Equal(t, (*string)(nil), updateData.Summary)

	// Test that update validation follows same rules as create
	// Empty title should still fail
	updateData = schemas.BookUpdate{Title: (*string)(nil)}
	*updateData.Title = ""
	assert.Error(t, updateData.Validate(context.Background()))
	assert.Contains(t, updateData.Validate(context.Background()).Error(), "Title cannot be empty")

	// Invalid year should still fail
	updateData = schemas.BookUpdate{PublishedYear: (*int)(nil)}
	*updateData.PublishedYear = 999
	assert.Error(t, updateData.Validate(context.Background()))
	assert.Contains(t, updateData.Validate(context.Background()).Error(), "Published year must be after year 1000")
}

func testBookResponse(t *testing.T) {
	// Test creating a valid book response
	bookData := schemas.BookResponse{
		ID:            1,
		Title:         "Response Book",
		Author:       "Response Author",
		PublishedYear: 2023,
		Summary:      "Response summary",
	}
	assert.NoError(t, bookData.Validate(context.Background()))
	assert.Equal(t, 1, bookData.ID)
	assert.Equal(t, "Response Book", bookData.Title)
	assert.Equal(t, "Response Author", bookData.Author)
	assert.Equal(t, 2023, bookData.PublishedYear)
	assert.Equal(t, "Response summary", bookData.Summary)

	// Test that ID is required in response
	bookData = schemas.BookResponse{
		Title:         "Title",
		Author:       "Author",
		PublishedYear: 2023,
	}
	assert.Error(t, bookData.Validate(context.Background()))
	assert.Contains(t, bookData.Validate(context.Background()).Error(), "id")
}