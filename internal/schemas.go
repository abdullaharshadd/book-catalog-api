package internal

import (
	"fmt"
	"strings"
	"time"
)

// MIGRATION_NOTE: The Python source used Pydantic v2 BaseModel schemas with
// field_validator classmethods and ConfigDict options. Go has no runtime
// validation framework equivalent to Pydantic, so the DTOs are expressed as
// plain structs and the validators are re-implemented as explicit Validate()
// methods returning errors. Two behaviours from Pydantic's ConfigDict are
// replicated manually:
//
//   - str_strip_whitespace=True: whitespace is stripped from string fields
//     during validation (mutating the struct in place).
//   - from_attributes=True (BookResponse): in Go this is just constructing a
//     BookResponse from a Book; see NewBookResponse.
//
// The critical null-vs-omitted distinction between BookCreate and BookUpdate
// is preserved with pointer types on BookUpdate: a nil pointer means the field
// was omitted (no change), while a non-nil pointer means the caller supplied a
// value. On BookCreate every field except summary is required, so those are
// value types; summary is a *string because it is optional and nullable.

// maxTitleLen is the maximum allowed length of a book title.
const maxTitleLen = 255

// maxAuthorLen is the maximum allowed length of a book author.
const maxAuthorLen = 255

// maxSummaryLen is the maximum allowed length of a book summary.
const maxSummaryLen = 2000

// minPublishedYear is the earliest allowed publication year.
const minPublishedYear = 1000

// BookCreate is the request payload for creating a new book.
//
// MIGRATION_NOTE: Mirrors Pydantic BookCreate. Title, Author and PublishedYear
// are required (value types); Summary is optional and nullable (*string).
type BookCreate struct {
	// Title is the book title. Required, 1-255 characters after trimming.
	Title string `json:"title"`

	// Author is the book author. Required, 1-255 characters after trimming.
	Author string `json:"author"`

	// PublishedYear is the year the book was published. Required, must be in
	// the range [1000, current year].
	PublishedYear int `json:"published_year"`

	// Summary is an optional free-text summary, at most 2000 characters after
	// trimming. A nil pointer or a blank string is normalized to nil.
	Summary *string `json:"summary"`
}

// Validate checks the BookCreate payload and normalizes its string fields,
// stripping surrounding whitespace in the same way Pydantic's
// str_strip_whitespace config did. It returns the first validation error
// encountered, or nil if the payload is valid.
func (b *BookCreate) Validate() error {
	title, err := validateRequiredString(b.Title, "Title", maxTitleLen)
	if err != nil {
		return err
	}
	b.Title = title

	author, err := validateRequiredString(b.Author, "Author", maxAuthorLen)
	if err != nil {
		return err
	}
	b.Author = author

	if err := validatePublishedYear(b.PublishedYear); err != nil {
		return err
	}

	b.Summary = normalizeSummary(b.Summary)
	if err := validateSummaryLen(b.Summary); err != nil {
		return err
	}

	return nil
}

// BookUpdate is the request payload for a partial update of an existing book.
//
// MIGRATION_NOTE: Every field is a pointer to preserve Pydantic's Optional
// semantics. A nil pointer means "field omitted, leave unchanged"; a non-nil
// pointer means "apply this value". This is the hard prerequisite extraction
// target called out in the migration notes: the null-vs-omitted distinction
// per field is encoded by the pointer being nil vs non-nil.
type BookUpdate struct {
	// Title, when non-nil, replaces the book title. Must be 1-255 characters
	// after trimming.
	Title *string `json:"title"`

	// Author, when non-nil, replaces the book author. Must be 1-255 characters
	// after trimming.
	Author *string `json:"author"`

	// PublishedYear, when non-nil, replaces the publication year. Must be in
	// the range [1000, current year].
	PublishedYear *int `json:"published_year"`

	// Summary, when non-nil, replaces the summary. A blank value is normalized
	// to an empty (but non-nil) pointer, matching the Python validator which
	// returned None for blank input.
	Summary *string `json:"summary"`
}

// Validate checks the BookUpdate payload and normalizes any supplied string
// fields. Omitted fields (nil pointers) are skipped, matching Pydantic's
// Optional-with-default-None behaviour. It returns the first validation error
// encountered, or nil if the payload is valid.
func (b *BookUpdate) Validate() error {
	if b.Title != nil {
		title, err := validateRequiredString(*b.Title, "Title", maxTitleLen)
		if err != nil {
			return err
		}
		b.Title = &title
	}

	if b.Author != nil {
		author, err := validateRequiredString(*b.Author, "Author", maxAuthorLen)
		if err != nil {
			return err
		}
		b.Author = &author
	}

	if b.PublishedYear != nil {
		if err := validatePublishedYear(*b.PublishedYear); err != nil {
			return err
		}
	}

	if b.Summary != nil {
		// MIGRATION_NOTE: The Python summary validator returned None for a
		// blank string. Here we keep the pointer non-nil (so the update still
		// applies) but normalize the value to empty, mirroring the intent that
		// a blank summary clears the field rather than being an error.
		b.Summary = normalizeSummary(b.Summary)
		if b.Summary == nil {
			empty := ""
			b.Summary = &empty
		}
		if err := validateSummaryLen(b.Summary); err != nil {
			return err
		}
	}

	return nil
}

// BookResponse is the response payload returned to clients for a book.
//
// MIGRATION_NOTE: Mirrors Pydantic BookResponse with from_attributes=True.
// In Go the "from attributes" conversion is an explicit constructor,
// NewBookResponse, that maps a persisted Book into the response DTO.
type BookResponse struct {
	// ID is the book's primary key.
	ID int `json:"id"`

	// Title is the book title.
	Title string `json:"title"`

	// Author is the book author.
	Author string `json:"author"`

	// PublishedYear is the year the book was published.
	PublishedYear int `json:"published_year"`

	// Summary is the optional book summary; nil when absent.
	Summary *string `json:"summary"`
}

// NewBookResponse builds a BookResponse from a persisted Book. This is the
// idiomatic Go replacement for Pydantic's from_attributes=True config, which
// allowed constructing a response schema directly from an ORM model instance.
//
// MIGRATION_NOTE: Adjust the field accesses below to match the actual Book
// struct defined in internal/model.go (field names, and whether Summary is a
// sql.NullString or *string). This requires manual review against model.go.
func NewBookResponse(b Book) BookResponse {
	return BookResponse{
		ID:            b.ID,
		Title:         b.Title,
		Author:        b.Author,
		PublishedYear: b.PublishedYear,
		Summary:       bookSummaryPtr(b),
	}
}

// bookSummaryPtr extracts an optional summary pointer from a Book.
//
// MIGRATION_NOTE: model.go exposes SummaryOrEmpty(); the underlying storage
// representation of the summary column (sql.NullString vs *string) is not
// visible from this file. We use SummaryOrEmpty and treat an empty string as
// "no summary" (nil) to match the Python validators that normalized blank
// summaries to None. Revisit if the model should distinguish empty from NULL.
func bookSummaryPtr(b Book) *string {
	s := strings.TrimSpace(b.SummaryOrEmpty())
	if s == "" {
		return nil
	}
	return &s
}

// validateRequiredString trims surrounding whitespace and enforces that the
// value is non-empty and no longer than maxLen. The fieldName is used in the
// "cannot be empty" error message to match the Python validators.
func validateRequiredString(v, fieldName string, maxLen int) (string, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return "", fmt.Errorf("%s cannot be empty", fieldName)
	}
	// MIGRATION_NOTE: The Python validator checked len(v) BEFORE stripping
	// (against the raw value) yet returned the stripped value. We enforce the
	// length against the pre-strip value to preserve that exact behaviour.
	if len(v) > maxLen {
		return "", fmt.Errorf("ensure this value has at most %d characters", maxLen)
	}
	return trimmed, nil
}

// validatePublishedYear enforces the [1000, current year] range, matching the
// Python published_year validators.
func validatePublishedYear(v int) error {
	if v < minPublishedYear {
		return fmt.Errorf("Published year must be after year %d", minPublishedYear)
	}
	currentYear := time.Now().Year()
	if v > currentYear {
		return fmt.Errorf("Published year cannot be in the future (current year: %d)", currentYear)
	}
	return nil
}

// normalizeSummary applies the Python summary validator's normalization:
// a nil pointer stays nil, and a value that is blank after trimming becomes
// nil. Otherwise the trimmed value is returned. Length is validated separately
// by validateSummaryLen.
func normalizeSummary(v *string) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// validateSummaryLen enforces the maximum summary length. A nil summary is
// always valid.
func validateSummaryLen(v *string) error {
	if v == nil {
		return nil
	}
	if len(*v) > maxSummaryLen {
		return fmt.Errorf("ensure this value has at most %d characters", maxSummaryLen)
	}
	return nil
}
