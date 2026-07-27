package tests

import (
	"strings"
	"testing"
	"time"

	"app/internal"
)

// MIGRATION_NOTE: The Python source was a pytest suite exercising Pydantic
// schema validation (BookCreate, BookUpdate, BookResponse). In Go these are
// plain structs in internal/schemas.go whose validation is performed by an
// explicit Validate method rather than by construction. Pydantic performs
// whitespace stripping and "empty summary -> None" coercion as part of
// validation, so the Go Validate methods are expected to mutate/normalise the
// receiver in place (title/author trimmed, blank summary cleared). These tests
// call Validate and then assert on the normalised field values.
//
// pytest.raises(ValidationError) is replaced by asserting that Validate returns
// a non-nil error whose message contains the expected text. The class-based
// grouping (TestBookCreate / TestBookUpdate / TestBookResponse) is flattened
// into individually named top-level test functions, and each is written
// table-driven where multiple cases share the same shape.

// helperStr returns a pointer to the given string, for optional fields.
func helperStr(s string) *string { return &s }

// helperInt returns a pointer to the given int, for optional fields.
func helperInt(i int) *int { return &i }

// TestBookCreateValid verifies that a fully populated BookCreate validates
// successfully and preserves its field values.
func TestBookCreateValid(t *testing.T) {
	bc := internal.BookCreate{
		Title:         "Valid Book",
		Author:        "Valid Author",
		PublishedYear: 2023,
		Summary:       helperStr("A valid book summary"),
	}

	if err := bc.Validate(); err != nil {
		t.Fatalf("Validate() returned unexpected error: %v", err)
	}

	if bc.Title != "Valid Book" {
		t.Errorf("Title = %q, want %q", bc.Title, "Valid Book")
	}
	if bc.Author != "Valid Author" {
		t.Errorf("Author = %q, want %q", bc.Author, "Valid Author")
	}
	if bc.PublishedYear != 2023 {
		t.Errorf("PublishedYear = %d, want %d", bc.PublishedYear, 2023)
	}
	if bc.Summary == nil || *bc.Summary != "A valid book summary" {
		t.Errorf("Summary = %v, want %q", bc.Summary, "A valid book summary")
	}
}

// TestBookCreateWithoutSummary verifies that summary is optional and remains
// nil when omitted.
func TestBookCreateWithoutSummary(t *testing.T) {
	bc := internal.BookCreate{
		Title:         "Book Without Summary",
		Author:        "Author",
		PublishedYear: 2023,
	}

	if err := bc.Validate(); err != nil {
		t.Fatalf("Validate() returned unexpected error: %v", err)
	}

	if bc.Title != "Book Without Summary" {
		t.Errorf("Title = %q, want %q", bc.Title, "Book Without Summary")
	}
	if bc.Author != "Author" {
		t.Errorf("Author = %q, want %q", bc.Author, "Author")
	}
	if bc.PublishedYear != 2023 {
		t.Errorf("PublishedYear = %d, want %d", bc.PublishedYear, 2023)
	}
	if bc.Summary != nil {
		t.Errorf("Summary = %v, want nil", *bc.Summary)
	}
}

// TestBookCreateStripsWhitespace verifies that title, author and summary are
// trimmed of surrounding whitespace during validation.
func TestBookCreateStripsWhitespace(t *testing.T) {
	bc := internal.BookCreate{
		Title:         "  Whitespace Book  ",
		Author:        "  Whitespace Author  ",
		PublishedYear: 2023,
		Summary:       helperStr("  Whitespace summary  "),
	}

	if err := bc.Validate(); err != nil {
		t.Fatalf("Validate() returned unexpected error: %v", err)
	}

	if bc.Title != "Whitespace Book" {
		t.Errorf("Title = %q, want %q", bc.Title, "Whitespace Book")
	}
	if bc.Author != "Whitespace Author" {
		t.Errorf("Author = %q, want %q", bc.Author, "Whitespace Author")
	}
	if bc.Summary == nil || *bc.Summary != "Whitespace summary" {
		t.Errorf("Summary = %v, want %q", bc.Summary, "Whitespace summary")
	}
}

// TestBookCreateEmptySummaryBecomesNil verifies that a whitespace-only summary
// is normalised to nil.
func TestBookCreateEmptySummaryBecomesNil(t *testing.T) {
	bc := internal.BookCreate{
		Title:         "Book",
		Author:        "Author",
		PublishedYear: 2023,
		Summary:       helperStr("   "),
	}

	if err := bc.Validate(); err != nil {
		t.Fatalf("Validate() returned unexpected error: %v", err)
	}

	if bc.Summary != nil {
		t.Errorf("Summary = %v, want nil", *bc.Summary)
	}
}

// TestBookCreateEmptyTitleValidation verifies that empty or whitespace-only
// titles are rejected.
func TestBookCreateEmptyTitleValidation(t *testing.T) {
	tests := []struct {
		name  string
		title string
	}{
		{name: "empty", title: ""},
		{name: "whitespace", title: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := internal.BookCreate{
				Title:         tt.title,
				Author:        "Author",
				PublishedYear: 2023,
			}
			err := bc.Validate()
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", "Title cannot be empty")
			}
			if !strings.Contains(err.Error(), "Title cannot be empty") {
				t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), "Title cannot be empty")
			}
		})
	}
}

// TestBookCreateEmptyAuthorValidation verifies that empty or whitespace-only
// authors are rejected.
func TestBookCreateEmptyAuthorValidation(t *testing.T) {
	tests := []struct {
		name   string
		author string
	}{
		{name: "empty", author: ""},
		{name: "whitespace", author: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := internal.BookCreate{
				Title:         "Title",
				Author:        tt.author,
				PublishedYear: 2023,
			}
			err := bc.Validate()
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", "Author cannot be empty")
			}
			if !strings.Contains(err.Error(), "Author cannot be empty") {
				t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), "Author cannot be empty")
			}
		})
	}
}

// TestBookCreatePublishedYearValidation verifies the published-year bounds:
// years must be after 1000 and not in the future, with 1000 and the current
// year being valid edge cases.
func TestBookCreatePublishedYearValidation(t *testing.T) {
	currentYear := time.Now().Year()

	t.Run("too early", func(t *testing.T) {
		bc := internal.BookCreate{Title: "Title", Author: "Author", PublishedYear: 999}
		err := bc.Validate()
		if err == nil {
			t.Fatal("Validate() = nil, want error for year 999")
		}
		if !strings.Contains(err.Error(), "Published year must be after year 1000") {
			t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), "Published year must be after year 1000")
		}
	})

	t.Run("future year", func(t *testing.T) {
		bc := internal.BookCreate{Title: "Title", Author: "Author", PublishedYear: currentYear + 1}
		err := bc.Validate()
		if err == nil {
			t.Fatal("Validate() = nil, want error for future year")
		}
		if !strings.Contains(err.Error(), "cannot be in the future") {
			t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), "cannot be in the future")
		}
	})

	t.Run("min valid", func(t *testing.T) {
		bc := internal.BookCreate{Title: "Title", Author: "Author", PublishedYear: 1000}
		if err := bc.Validate(); err != nil {
			t.Fatalf("Validate() returned unexpected error: %v", err)
		}
		if bc.PublishedYear != 1000 {
			t.Errorf("PublishedYear = %d, want %d", bc.PublishedYear, 1000)
		}
	})

	t.Run("current year valid", func(t *testing.T) {
		bc := internal.BookCreate{Title: "Title", Author: "Author", PublishedYear: currentYear}
		if err := bc.Validate(); err != nil {
			t.Fatalf("Validate() returned unexpected error: %v", err)
		}
		if bc.PublishedYear != currentYear {
			t.Errorf("PublishedYear = %d, want %d", bc.PublishedYear, currentYear)
		}
	})
}

// TestBookCreateLengthValidation verifies the maximum-length constraints on
// title (255), author (255) and summary (2000).
func TestBookCreateLengthValidation(t *testing.T) {
	tests := []struct {
		name    string
		book    internal.BookCreate
		wantMsg string
	}{
		{
			name:    "title too long",
			book:    internal.BookCreate{Title: strings.Repeat("A", 256), Author: "Author", PublishedYear: 2023},
			wantMsg: "at most 255 characters",
		},
		{
			name:    "author too long",
			book:    internal.BookCreate{Title: "Title", Author: strings.Repeat("B", 256), PublishedYear: 2023},
			wantMsg: "at most 255 characters",
		},
		{
			name:    "summary too long",
			book:    internal.BookCreate{Title: "Title", Author: "Author", PublishedYear: 2023, Summary: helperStr(strings.Repeat("C", 2001))},
			wantMsg: "at most 2000 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.book.Validate()
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tt.wantMsg)
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

// TestBookUpdateValidPartial verifies that a partial update validates and
// leaves unspecified fields nil.
func TestBookUpdateValidPartial(t *testing.T) {
	bu := internal.BookUpdate{
		Title:         helperStr("Updated Title"),
		PublishedYear: helperInt(2024),
	}

	if err := bu.Validate(); err != nil {
		t.Fatalf("Validate() returned unexpected error: %v", err)
	}

	if bu.Title == nil || *bu.Title != "Updated Title" {
		t.Errorf("Title = %v, want %q", bu.Title, "Updated Title")
	}
	if bu.Author != nil {
		t.Errorf("Author = %v, want nil", *bu.Author)
	}
	if bu.PublishedYear == nil || *bu.PublishedYear != 2024 {
		t.Errorf("PublishedYear = %v, want %d", bu.PublishedYear, 2024)
	}
	if bu.Summary != nil {
		t.Errorf("Summary = %v, want nil", *bu.Summary)
	}
}

// TestBookUpdateEmpty verifies that an update with no fields set validates and
// leaves every field nil.
func TestBookUpdateEmpty(t *testing.T) {
	bu := internal.BookUpdate{}

	if err := bu.Validate(); err != nil {
		t.Fatalf("Validate() returned unexpected error: %v", err)
	}

	if bu.Title != nil {
		t.Errorf("Title = %v, want nil", *bu.Title)
	}
	if bu.Author != nil {
		t.Errorf("Author = %v, want nil", *bu.Author)
	}
	if bu.PublishedYear != nil {
		t.Errorf("PublishedYear = %v, want nil", *bu.PublishedYear)
	}
	if bu.Summary != nil {
		t.Errorf("Summary = %v, want nil", *bu.Summary)
	}
}

// TestBookUpdateValidationSameAsCreate verifies that update validation applies
// the same field rules as create when a field is present.
func TestBookUpdateValidationSameAsCreate(t *testing.T) {
	t.Run("empty title", func(t *testing.T) {
		bu := internal.BookUpdate{Title: helperStr("")}
		err := bu.Validate()
		if err == nil {
			t.Fatal("Validate() = nil, want error for empty title")
		}
		if !strings.Contains(err.Error(), "Title cannot be empty") {
			t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), "Title cannot be empty")
		}
	})

	t.Run("invalid year", func(t *testing.T) {
		bu := internal.BookUpdate{PublishedYear: helperInt(999)}
		err := bu.Validate()
		if err == nil {
			t.Fatal("Validate() = nil, want error for year 999")
		}
		if !strings.Contains(err.Error(), "Published year must be after year 1000") {
			t.Errorf("Validate() error = %q, want it to contain %q", err.Error(), "Published year must be after year 1000")
		}
	})
}

// TestBookResponseValid verifies that a BookResponse constructed from a Book
// carries the expected field values.
//
// MIGRATION_NOTE: Pydantic's BookResponse validated raw kwargs (including a
// required id). In Go, BookResponse is produced from a persisted internal.Book
// via internal.NewBookResponse, so there is no separate "missing id"
// validation path at the schema level — id is guaranteed by the model. The
// TestBookResponseMissingID case from the source therefore has no Go analogue
// and is intentionally omitted; see unmigrable_components.
func TestBookResponseValid(t *testing.T) {
	summary := "Response summary"
	book := &internal.Book{
		ID:            1,
		Title:         "Response Book",
		Author:        "Response Author",
		PublishedYear: 2023,
		Summary:       &summary,
	}

	resp := internal.NewBookResponse(book)

	if resp.ID != 1 {
		t.Errorf("ID = %d, want %d", resp.ID, 1)
	}
	if resp.Title != "Response Book" {
		t.Errorf("Title = %q, want %q", resp.Title, "Response Book")
	}
	if resp.Author != "Response Author" {
		t.Errorf("Author = %q, want %q", resp.Author, "Response Author")
	}
	if resp.PublishedYear != 2023 {
		t.Errorf("PublishedYear = %d, want %d", resp.PublishedYear, 2023)
	}
	if resp.Summary == nil || *resp.Summary != "Response summary" {
		t.Errorf("Summary = %v, want %q", resp.Summary, "Response summary")
	}
}
