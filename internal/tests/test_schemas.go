// Package tests contains the unit tests for models and schemas, and integration tests for API endpoints of the Book Catalog API.
package tests

import (
	"testing"
	"time"
	"migrated-app/internal/schemas"
)

// TestBookCreate tests the BookCreate schema.
func TestBookCreate(t *testing.T) {
	t.Run("valid book create", func(t *testing.T) {
		bookData := schemas.BookCreate{
			Title:          "Valid Book",
			Author:         "Valid Author",
			PublishedYear:  2023,
			Summary:        "A valid book summary",
		}

		err := bookData.Validate()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("book create without summary", func(t *testing.T) {
		bookData := schemas.BookCreate{
			Title:          "Book Without Summary",
			Author:         "Author",
			PublishedYear:  2023,
		}

		err := bookData.Validate()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if bookData.Summary != nil {
			t.Errorf("expected summary to be nil, got %s", *bookData.Summary)
		}
	})

	t.Run("whitespace stripping", func(t *testing.T) {
		bookData := schemas.BookCreate{
			Title:          "  Whitespace Book  ",
			Author:         "  Whitespace Author  ",
			PublishedYear:  2023,
			Summary:        &"  Whitespace summary  ",
		}

		err := bookData.Validate()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if bookData.Title != "Whitespace Book" || bookData.Author != "Whitespace Author" || *bookData.Summary != "Whitespace summary" {
			t.Errorf("expected whitespace to be stripped, got %+v", bookData)
		}
	})

	t.Run("empty summary becomes nil", func(t *testing.T) {
		bookData := schemas.BookCreate{
			Title:          "Book",
			Author:         "Author",
			PublishedYear:  2023,
			Summary:        &"   ", // Only whitespace
		}

		err := bookData.Validate()
		if err != nil {
			t.Errorf("expected no error for empty summary, got %v", err)
		}
		if bookData.Summary != nil {
			t.Errorf("expected summary to be nil, got %s", *bookData.Summary)
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		testCases := []struct {
			bookData schemas.BookCreate
			field    string
		}{
			{schemas.BookCreate{Author: "Author", PublishedYear: 2023}, "title"},
			{schemas.BookCreate{Title: "Title", PublishedYear: 2023}, "author"},
			{schemas.BookCreate{Title: "Title", Author: "Author"}, "published_year"},
		}

		for _, tc := range testCases {
			err := tc.bookData.Validate()
			if err == nil {
				t.Errorf("expected error for missing %s, got none", tc.field)
			 else if !tc.bookData.ValidationError(err, tc.field) {
				t.Errorf("expected error for missing %s, got %v", tc.field, err)
			}
		}
	})

	t.Run("empty title validation", func(t *testing.T) {
		testCases := []struct {
			title string
		}{
			{"   "},
			{""},
		}

		for _, tc := range testCases {
			bookData := schemas.BookCreate{
				Title:          tc.title,
				Author:         "Author",
				PublishedYear:  2023,
			}

			err := bookData.Validate()
			if err == nil {
				t.Errorf("expected error for empty title, got none")
			 else if !bookData.ValidationError(err, "Title cannot be empty") {
				t.Errorf("expected error for empty title, got %v", err)
			}
		}
	})

	t.Run("empty author validation", func(t *testing.T) {
		testCases := []struct {
			author string
		}{
			{"   "},
			{""},
		}

		for _, tc := range testCases {
			bookData := schemas.BookCreate{
				Title:          "Title",
				Author:         tc.author,
				PublishedYear:  2023,
			}

			err := bookData.Validate()
			if err == nil {
				t.Errorf("expected error for empty author, got none")
			 else if !bookData.ValidationError(err, "Author cannot be empty") {
				t.Errorf("expected error for empty author, got %v", err)
			}
		}
	})

	t.Run("published year validation", func(t *testing.T) {
		currentYear := time.Now().Year()

		testCases := []struct {
			year int
			err  string
		}{
			{999, "Published year must be after year 1000"},
			{currentYear + 1, "Published year cannot be in the future (current year: " + strconv.Itoa(currentYear) + ")"},
			{1000, ""},
			{currentYear, ""},
		}

		for _, tc := range testCases {
			bookData := schemas.BookCreate{
				Title:          "Title",
				Author:         "Author",
				PublishedYear:  tc.year,
			}

			err := bookData.Validate()
			if tc.err == "" && err != nil {
				t.Errorf("expected no error for year %d, got %v", tc.year, err)
			 else if tc.err != "" && err == nil {
				t.Errorf("expected error for year %d, got none", tc.year)
			 else if tc.err != "" && !bookData.ValidationError(err, tc.err) {
				t.Errorf("expected error %q for year %d, got %v", tc.err, tc.year, err)
			}
		}
	})

	t.Run("title length validation", func(t *testing.T) {
		longTitle := "A" + string(make([]byte, 255))
		bookData := schemas.BookCreate{
			Title:          longTitle,
			Author:         "Author",
			PublishedYear:  2023,
		}

		err := bookData.Validate()
		if err == nil {
			t.Errorf("expected error for title length, got none")
		 else if !bookData.ValidationError(err, "ensure this value has at most 255 characters") {
			t.Errorf("expected error for title length, got %v", err)
		}
	})

	t.Run("author length validation", func(t *testing.T) {
		longAuthor := "B" + string(make([]byte, 255))
		bookData := schemas.BookCreate{
			Title:          "Title",
			Author:         longAuthor,
			PublishedYear:  2023,
		}

		err := bookData.Validate()
		if err == nil {
			t.Errorf("expected error for author length, got none")
		 else if !bookData.ValidationError(err, "ensure this value has at most 255 characters") {
			t.Errorf("expected error for author length, got %v", err)
		}
	})

	t.Run("summary length validation", func(t *testing.T) {
		longSummary := "C" + string(make([]byte, 2000))
		bookData := schemas.BookCreate{
			Title:          "Title",
			Author:         "Author",
			PublishedYear:  2023,
			Summary:        &longSummary,
		}

		err := bookData.Validate()
		if err == nil {
			t.Errorf("expected error for summary length, got none")
		 else if !bookData.ValidationError(err, "ensure this value has at most 2000 characters") {
			t.Errorf("expected error for summary length, got %v", err)
		}
	})
}

// TestBookUpdate tests the BookUpdate schema.
func TestBookUpdate(t *testing.T) {
	t.Run("valid partial update", func(t *testing.T) {
		updateData := schemas.BookUpdate{
			Title:          "Updated Title",
			PublishedYear:  &2024,
		}

		err := updateData.Validate()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if updateData.Author != nil || updateData.Summary != nil {
			t.Errorf("expected optional fields to be nil, got %+v", updateData)
		}
	})

	t.Run("empty update", func(t *testing.T) {
		updateData := schemas.BookUpdate{}

		err := updateData.Validate()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("update validation same as create", func(t *testing.T) {
		testCases := []struct {
			updateData schemas.BookUpdate
			err        string
		}{
			{schemas.BookUpdate{Title: &""}, "Title cannot be empty"},
			{schemas.BookUpdate{PublishedYear: &999}, "Published year must be after year 1000"},
		}

		for _, tc := range testCases {
			err := tc.updateData.Validate()
			if err == nil {
				t.Errorf("expected error for invalid data, got none")
			 else if !tc.updateData.ValidationError(err, tc.err) {
				t.Errorf("expected error %q, got %v", tc.err, err)
			}
		}
	})
}

// TestBookResponse tests the BookResponse schema.
func TestBookResponse(t *testing.T) {
	t.Run("valid book response", func(t *testing.T) {
		bookData := schemas.BookResponse{
			ID:             1,
			Title:          "Response Book",
			Author:         "Response Author",
			PublishedYear:  2023,
			Summary:        &"Response summary",
		}

		err := bookData.Validate()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("book response missing id", func(t *testing.T) {
		bookData := schemas.BookResponse{
			Title:          "Title",
			Author:         "Author",
			PublishedYear:  2023,
		}

		err := bookData.Validate()
		if err == nil {
			t.Errorf("expected error for missing id, got none")
		 else if !bookData.ValidationError(err, "id") {
			t.Errorf("expected error for missing id, got %v", err)
		}
	})
}