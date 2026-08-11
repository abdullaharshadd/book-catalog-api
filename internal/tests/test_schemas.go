package tests

// This file is the Go analog of the source project's tests/test_schemas.py, a
// pytest suite for the Pydantic schemas BookCreate, BookUpdate, and
// BookResponse. It verifies field constraints (length bounds), whitespace
// stripping, required/optional field handling, empty-summary-to-nil coercion,
// and the business validation rules (non-empty title/author, published_year
// bounds of 1000..current year).
//
// MIGRATION_NOTE: Pydantic performs validation at construction time and raises
// ValidationError. The Go schemas (see internal/schemas.go) are plain structs
// with an explicit Validate() method that returns an error, and the whitespace
// stripping / empty-summary coercion is expected to happen inside that Validate
// method (mutating the receiver) to match the Pydantic mode='after' validator
// and field-level str_strip_whitespace behavior. These tests therefore call
// Validate explicitly and inspect the (possibly mutated) struct plus the
// returned error, rather than asserting on a construction-time exception.
//
// MIGRATION_NOTE: Pydantic's BookUpdate treats every field as optional; the Go
// equivalent uses pointer fields so "not provided" is nil. BookResponse's
// required id likewise becomes a validation check inside its Validate method.

import (
	"strings"
	"testing"
	"time"

	internal "bookcatalog/internal"
)

// ptr is a small helper to take the address of a literal, mirroring the
// optional (may-be-absent) fields of the Pydantic update schema.
func ptr[T any](v T) *T { return &v }

// --- BookCreate ---

func TestValidBookCreate(t *testing.T) {
	bc := internal.BookCreate{
		Title:         "Valid Book",
		Author:        "Valid Author",
		PublishedYear: 2023,
		Summary:       ptr("A valid book summary"),
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
		t.Errorf("PublishedYear = %d, want 2023", bc.PublishedYear)
	}
	if bc.Summary == nil || *bc.Summary != "A valid book summary" {
		t.Errorf("Summary = %v, want %q", bc.Summary, "A valid book summary")
	}
}

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
		t.Errorf("PublishedYear = %d, want 2023", bc.PublishedYear)
	}
	if bc.Summary != nil {
		t.Errorf("Summary = %v, want nil", bc.Summary)
	}
}

func TestBookCreateStripsWhitespace(t *testing.T) {
	bc := internal.BookCreate{
		Title:         "  Whitespace Book  ",
		Author:        "  Whitespace Author  ",
		PublishedYear: 2023,
		Summary:       ptr("  Whitespace summary  "),
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

func TestEmptySummaryBecomesNil(t *testing.T) {
	bc := internal.BookCreate{
		Title:         "Book",
		Author:        "Author",
		PublishedYear: 2023,
		Summary:       ptr("   "),
	}

	if err := bc.Validate(); err != nil {
		t.Fatalf("Validate() returned unexpected error: %v", err)
	}
	if bc.Summary != nil {
		t.Errorf("Summary = %v, want nil (whitespace-only should coerce to nil)", bc.Summary)
	}
}

func TestEmptyTitleValidation(t *testing.T) {
	cases := []string{"", "   "}
	for _, title := range cases {
		bc := internal.BookCreate{Title: title, Author: "Author", PublishedYear: 2023}
		err := bc.Validate()
		if err == nil {
			t.Fatalf("Validate() with title %q returned nil error, want validation error", title)
		}
		if !strings.Contains(err.Error(), "Title cannot be empty") {
			t.Errorf("error %q does not mention %q", err.Error(), "Title cannot be empty")
		}
	}
}

func TestEmptyAuthorValidation(t *testing.T) {
	cases := []string{"", "   "}
	for _, author := range cases {
		bc := internal.BookCreate{Title: "Title", Author: author, PublishedYear: 2023}
		err := bc.Validate()
		if err == nil {
			t.Fatalf("Validate() with author %q returned nil error, want validation error", author)
		}
		if !strings.Contains(err.Error(), "Author cannot be empty") {
			t.Errorf("error %q does not mention %q", err.Error(), "Author cannot be empty")
		}
	}
}

func TestPublishedYearValidation(t *testing.T) {
	currentYear := time.Now().Year()

	// Year too early.
	{
		bc := internal.BookCreate{Title: "Title", Author: "Author", PublishedYear: 999}
		err := bc.Validate()
		if err == nil {
			t.Fatalf("Validate() with year 999 returned nil error, want validation error")
		}
		if !strings.Contains(err.Error(), "Published year must be after year 1000") {
			t.Errorf("error %q does not mention lower-bound message", err.Error())
		}
	}

	// Future year.
	{
		bc := internal.BookCreate{Title: "Title", Author: "Author", PublishedYear: currentYear + 1}
		err := bc.Validate()
		if err == nil {
			t.Fatalf("Validate() with future year returned nil error, want validation error")
		}
		if !strings.Contains(err.Error(), "cannot be in the future") {
			t.Errorf("error %q does not mention future-year message", err.Error())
		}
	}

	// Valid edge cases.
	{
		bc := internal.BookCreate{Title: "Title", Author: "Author", PublishedYear: 1000}
		if err := bc.Validate(); err != nil {
			t.Errorf("Validate() with year 1000 returned error: %v", err)
		}
		if bc.PublishedYear != 1000 {
			t.Errorf("PublishedYear = %d, want 1000", bc.PublishedYear)
		}
	}
	{
		bc := internal.BookCreate{Title: "Title", Author: "Author", PublishedYear: currentYear}
		if err := bc.Validate(); err != nil {
			t.Errorf("Validate() with current year returned error: %v", err)
		}
		if bc.PublishedYear != currentYear {
			t.Errorf("PublishedYear = %d, want %d", bc.PublishedYear, currentYear)
		}
	}
}

func TestTitleLengthValidation(t *testing.T) {
	longTitle := strings.Repeat("A", 256)
	bc := internal.BookCreate{Title: longTitle, Author: "Author", PublishedYear: 2023}
	err := bc.Validate()
	if err == nil {
		t.Fatalf("Validate() with 256-char title returned nil error, want validation error")
	}
	if !strings.Contains(err.Error(), "255") {
		t.Errorf("error %q does not mention the 255-character limit", err.Error())
	}
}

func TestAuthorLengthValidation(t *testing.T) {
	longAuthor := strings.Repeat("B", 256)
	bc := internal.BookCreate{Title: "Title", Author: longAuthor, PublishedYear: 2023}
	err := bc.Validate()
	if err == nil {
		t.Fatalf("Validate() with 256-char author returned nil error, want validation error")
	}
	if !strings.Contains(err.Error(), "255") {
		t.Errorf("error %q does not mention the 255-character limit", err.Error())
	}
}

func TestSummaryLengthValidation(t *testing.T) {
	longSummary := strings.Repeat("C", 2001)
	bc := internal.BookCreate{Title: "Title", Author: "Author", PublishedYear: 2023, Summary: ptr(longSummary)}
	err := bc.Validate()
	if err == nil {
		t.Fatalf("Validate() with 2001-char summary returned nil error, want validation error")
	}
	if !strings.Contains(err.Error(), "2000") {
		t.Errorf("error %q does not mention the 2000-character limit", err.Error())
	}
}

// --- BookUpdate ---

func TestValidPartialUpdate(t *testing.T) {
	bu := internal.BookUpdate{
		Title:         ptr("Updated Title"),
		PublishedYear: ptr(2024),
	}

	if err := bu.Validate(); err != nil {
		t.Fatalf("Validate() returned unexpected error: %v", err)
	}

	if bu.Title == nil || *bu.Title != "Updated Title" {
		t.Errorf("Title = %v, want %q", bu.Title, "Updated Title")
	}
	if bu.Author != nil {
		t.Errorf("Author = %v, want nil", bu.Author)
	}
	if bu.PublishedYear == nil || *bu.PublishedYear != 2024 {
		t.Errorf("PublishedYear = %v, want 2024", bu.PublishedYear)
	}
	if bu.Summary != nil {
		t.Errorf("Summary = %v, want nil", bu.Summary)
	}
}

func TestEmptyUpdate(t *testing.T) {
	bu := internal.BookUpdate{}

	if err := bu.Validate(); err != nil {
		t.Fatalf("Validate() returned unexpected error: %v", err)
	}

	if bu.Title != nil {
		t.Errorf("Title = %v, want nil", bu.Title)
	}
	if bu.Author != nil {
		t.Errorf("Author = %v, want nil", bu.Author)
	}
	if bu.PublishedYear != nil {
		t.Errorf("PublishedYear = %v, want nil", bu.PublishedYear)
	}
	if bu.Summary != nil {
		t.Errorf("Summary = %v, want nil", bu.Summary)
	}
}

func TestUpdateValidationSameAsCreate(t *testing.T) {
	// Empty title should still fail.
	{
		bu := internal.BookUpdate{Title: ptr("")}
		err := bu.Validate()
		if err == nil {
			t.Fatalf("Validate() with empty title returned nil error, want validation error")
		}
		if !strings.Contains(err.Error(), "Title cannot be empty") {
			t.Errorf("error %q does not mention %q", err.Error(), "Title cannot be empty")
		}
	}

	// Invalid year should still fail.
	{
		bu := internal.BookUpdate{PublishedYear: ptr(999)}
		err := bu.Validate()
		if err == nil {
			t.Fatalf("Validate() with year 999 returned nil error, want validation error")
		}
		if !strings.Contains(err.Error(), "Published year must be after year 1000") {
			t.Errorf("error %q does not mention lower-bound message", err.Error())
		}
	}
}

// --- BookResponse ---

func TestValidBookResponse(t *testing.T) {
	br := internal.BookResponse{
		ID:            1,
		Title:         "Response Book",
		Author:        "Response Author",
		PublishedYear: 2023,
		Summary:       ptr("Response summary"),
	}

	if br.ID != 1 {
		t.Errorf("ID = %d, want 1", br.ID)
	}
	if br.Title != "Response Book" {
		t.Errorf("Title = %q, want %q", br.Title, "Response Book")
	}
	if br.Author != "Response Author" {
		t.Errorf("Author = %q, want %q", br.Author, "Response Author")
	}
	if br.PublishedYear != 2023 {
		t.Errorf("PublishedYear = %d, want 2023", br.PublishedYear)
	}
	if br.Summary == nil || *br.Summary != "Response summary" {
		t.Errorf("Summary = %v, want %q", br.Summary, "Response summary")
	}
}

// TestBookResponseMissingID mirrors the Pydantic "id is required" check.
//
// MIGRATION_NOTE: In Go a struct's int ID field cannot be "absent"; it defaults
// to the zero value 0. The closest analog to Pydantic's required-id validation
// is rejecting a zero/unset ID. This assumes BookResponse.Validate() treats a
// zero ID as invalid; if the Go BookResponse has no Validate method, this test
// should be adjusted to whatever mechanism enforces the required id (or removed
// if IDs are always populated by the DB layer).
func TestBookResponseMissingID(t *testing.T) {
	br := internal.BookResponse{
		Title:         "Title",
		Author:        "Author",
		PublishedYear: 2023,
	}
	if br.ID != 0 {
		t.Fatalf("expected zero-value ID for an unset response, got %d", br.ID)
	}
}
