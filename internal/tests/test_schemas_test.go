```go
package tests

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"internal/schemas"
)

func TestSuite(t *testing.T) {
	testBookCreate(t)
	testBookUpdate(t)
	testBookResponse(t)
}

func testBookCreate(t *testing.T) {
	cases := []struct {
		bookData schemas.BookCreate
		validate bool
		check    func(*testing.T, error, schemas.BookCreate)
	}{
		{bookData: schemas.BookCreate{Title: "Valid Book", Author: "Valid Author", PublishedYear: 2023, Summary: "A valid book summary"}, validate: true, check: checkBookCreateValid},
		{bookData: schemas.BookCreate{Title: "Book Without Summary", Author: "Author", PublishedYear: 2023}, validate: true, check: checkBookCreateNoSummary},
		{bookData: schemas.BookCreate{Title: "  Whitespace Book  ", Author: "  Whitespace Author  ", PublishedYear: 2023, Summary: "  Whitespace summary  "}, validate: true, check: checkBookCreateWhitespace},
		{bookData: schemas.BookCreate{Title: "Book", Author: "Author", PublishedYear: 2023, Summary: "   "}, validate: true, check: checkBookCreateEmptySummary},
		{bookData: schemas.BookCreate{Author: "Author", PublishedYear: 2023}, validate: false, check: checkBookCreateMissingTitle},
		{bookData: schemas.BookCreate{Title: "Title", PublishedYear: 2023}, validate: false, check: checkBookCreateMissingAuthor},
		{bookData: schemas.BookCreate{Title: "Title", Author: "Author"}, validate: false, check: checkBookCreateMissingPublishedYear},
		{bookData: schemas.BookCreate{Title: "", Author: "Author", PublishedYear: 2023}, validate: false, check: checkBookCreateEmptyTitle},
		{bookData: schemas.BookCreate{Title: "   ", Author: "Author", PublishedYear: 2023}, validate: false, check: checkBookCreateEmptyTitle},
		{bookData: schemas.BookCreate{Title: "Title", Author: "", PublishedYear: 2023}, validate: false, check: checkBookCreateEmptyAuthor},
		{bookData: schemas.BookCreate{Title: "Title", Author: "   ", PublishedYear: 2023}, validate: false, check: checkBookCreateEmptyAuthor},
		{bookData: schemas.BookCreate{Title: "Title", Author: "Author", PublishedYear: 999}, validate: false, check: checkBookCreateInvalidYear},
		{bookData: schemas.BookCreate{Title: "Title", Author: "Author", PublishedYear: 2024}, validate: false, check: checkBookCreateFutureYear},
		{bookData: schemas.BookCreate{Title: "A" + "A"*255, Author: "Author", PublishedYear: 2023}, validate: false, check: checkBookCreateTitleTooLong},
		{bookData: schemas.BookCreate{Title: "Title", Author: "B" + "B"*255, PublishedYear: 2023}, validate: false, check: checkBookCreateAuthorTooLong},
		{bookData: schemas.BookCreate{Title: "Title", Author: "Author", PublishedYear: 2023, Summary: &("C" + "C"*2000)}, validate: false, check: checkBookCreateSummaryTooLong},
	}

	for _, tc := range cases {
		err := tc.bookData.Validate(context.Background())
		if tc.validate {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
		tc.check(t, err, tc.bookData)
	}
}

func checkBookCreateValid(t *testing.T, err error, bookData schemas.BookCreate) {
	assert.NoError(t, err)
	assert.Equal(t, "Valid Book", *bookData.Title)
	assert.Equal(t, "Valid Author", *bookData.Author)
	assert.Equal(t, 2023, bookData.PublishedYear)
	assert.Equal(t, "A valid book summary", *bookData.Summary)
}

func checkBookCreateNoSummary(t *testing.T, err error, bookData schemas.BookCreate) {
	assert.NoError(t, err)
	assert.Equal(t, "Book Without Summary", *bookData.Title)
	assert.Equal(t, "Author", *bookData.Author)
	assert.Equal(t, 2023, bookData.PublishedYear)
	assert.Equal(t, (*string)(nil), bookData.Summary)
}

func checkBookCreateWhitespace(t *testing.T, err error, bookData schemas.BookCreate) {
	assert.NoError(t, err)
	assert.Equal(t, "Whitespace Book", *bookData.Title)
	assert.Equal(t, "Whitespace Author", *bookData.Author)
	assert.Equal(t, "Whitespace summary", *bookData.Summary)
}

func checkBookCreateEmptySummary(t *testing.T, err error, bookData schemas.BookCreate) {
	assert.NoError(t, err)
	assert.Equal(t, (*string)(nil), bookData.Summary)
}

func checkBookCreateMissingTitle(t *testing.T, err error, bookData schemas.BookCreate) {
	assert.Contains(t, err.Error(), "title")
}

func checkBookCreateMissingAuthor(t *testing.T, err error, bookData schemas.BookCreate) {
	assert.Contains(t, err.Error(), "author")
}

func checkBookCreateMissingPublishedYear(t *testing.T, err error, bookData schemas.BookCreate) {
	assert.Contains(t, err.Error(), "published_year")
}

func checkBookCreateEmptyTitle(t *testing.T, err error, bookData schemas.BookCreate) {
	assert.Contains(t, err.Error(), "Title cannot be empty")
}

func checkBookCreateEmptyAuthor(t *testing.T, err error, bookData schemas.BookCreate) {
	assert.Contains(t, err.Error(), "Author cannot be empty")
}

func checkBookCreateInvalidYear(t *testing.T, err error, bookData schemas.BookCreate) {
	assert.Contains(t, err.Error(), "Published year must be after year 1000")
}

func checkBookCreateFutureYear(t *testing.T, err error, bookData schemas.BookCreate) {
	assert.Contains(t, err.Error(), "cannot be in the future")
}

func checkBookCreateTitleTooLong(t *testing.T, err error, bookData schemas.BookCreate) {
	assert.Contains(t, err.Error(), "ensure this value has at most 255 characters")
}

func checkBookCreateAuthorTooLong(t *testing.T, err error, bookData schemas.BookCreate) {
	assert.Contains(t, err.Error(), "ensure this value has at most 255 characters")
}

func checkBookCreateSummaryTooLong(t *testing.T, err error, bookData schemas.BookCreate) {
	assert.Contains(t, err.Error(), "ensure this value has at most 2000 characters")
}

func testBookUpdate(t *testing.T) {
	cases := []struct {
		updateData schemas.BookUpdate
		validate   bool
		check      func(*testing.T, error, schemas.BookUpdate)
	}{
		{updateData: schemas.BookUpdate{Title: (*string)(nil), PublishedYear: (*int)(nil)}, validate: true, check: checkBookUpdateNoFields},
		{updateData: schemas.BookUpdate{Title: (*string)(nil), PublishedYear: (*int)(nil)}, validate: true, check: checkBookUpdateEmptyTitle},
		{updateData: schemas.BookUpdate{PublishedYear: (*int)(nil)}, validate: false, check: checkBookUpdateInvalidYear},
	}

	for _, tc := range cases {
		if tc.updateData.Title != nil {
			*tc.updateData.Title = "Updated Title"
		}
		if tc.updateData.PublishedYear != nil {
			*tc.updateData.PublishedYear = 2024
		}
		err := tc.updateData.Validate(context.Background())
		if tc.validate {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
		tc.check(t, err, tc.updateData)
	}
}

func checkBookUpdateNoFields(t *testing.T, err error, updateData schemas.BookUpdate) {
	assert.NoError(t, err)
	assert.Equal(t, (*string)(nil), updateData.Title)
	assert.Equal(t, (*string)(nil), updateData.Author)
	assert.Equal(t, (*int)(nil), updateData.PublishedYear)
	assert.Equal(t, (*string)(nil), updateData.Summary)
}

func checkBookUpdateEmptyTitle(t *testing.T, err error, updateData schemas.BookUpdate) {
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Title cannot be empty")
}

func checkBookUpdateInvalidYear(t *testing.T, err error, updateData schemas.BookUpdate) {
	*updateData.PublishedYear = 999
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Published year must be after year 1000")
}

func testBookResponse(t *testing.T) {
	cases := []struct {
		bookData schemas.BookResponse
		validate bool
		check    func(*testing.T, error, schemas.BookResponse)
	}{
		{bookData: schemas.BookResponse{ID: 1, Title: "Response Book", Author: "Response Author", PublishedYear: 2023, Summary: "Response summary"}, validate: true, check: checkBookResponseValid},
		{bookData: schemas.BookResponse{Title: "Title", Author: "Author", PublishedYear: 2023, Summary: "Summary"}, validate: false, check: checkBookResponseMissingID},
	}

	for _, tc := range cases {
		err := tc.bookData.Validate(context.Background())
		if tc.validate {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
		tc.check(t, err, tc.bookData)
	}
}

func checkBookResponseValid(t *testing.T, err error, bookData schemas.BookResponse) {
	assert.NoError(t, err)
	assert.Equal(t, 1, bookData.ID)
	assert.Equal(t, "Response Book", bookData.Title)
	assert.Equal(t, "Response Author", bookData.Author)
	assert.Equal(t, 2023, bookData.PublishedYear)
	assert.Equal(t, "Response summary", bookData.Summary)
}

func checkBookResponseMissingID(t *testing.T, err error, bookData schemas.BookResponse) {
	assert.Contains(t, err.Error(), "id")
}
```