package internal

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// This file is the Go analog of the source project's app/schemas.py, which
// defined Pydantic v2 DTOs (BookCreate, BookUpdate, BookResponse) with
// field_validator methods and ConfigDict options.
//
// MIGRATION_NOTE: Pydantic performs validation and coercion automatically
// during model construction. Go has no equivalent decorator/metaclass
// machinery, so each request struct exposes an explicit Validate method that
// callers must invoke after JSON decoding. Pydantic's str_strip_whitespace
// (which trims all incoming string fields) is replicated by trimming inside
// Validate and mutating the struct in place, so callers observe the cleaned
// values just as they would with Pydantic.

// Field length limits, mirroring the Pydantic validators.
const (
	maxTitleLen   = 255
	maxAuthorLen  = 255
	maxSummaryLen = 2000
	minPublishedYear = 1000
)

// Validation errors surfaced by the request schemas. Callers may use
// errors.Is to distinguish them; the per-field messages match the original
// Pydantic ValueError text.
var (
	// ErrTitleEmpty is returned when a title is missing or blank.
	ErrTitleEmpty = errors.New("Title cannot be empty")
	// ErrTitleTooLong is returned when a title exceeds maxTitleLen characters.
	ErrTitleTooLong = errors.New("ensure this value has at most 255 characters")
	// ErrAuthorEmpty is returned when an author is missing or blank.
	ErrAuthorEmpty = errors.New("Author cannot be empty")
	// ErrAuthorTooLong is returned when an author exceeds maxAuthorLen characters.
	ErrAuthorTooLong = errors.New("ensure this value has at most 255 characters")
	// ErrPublishedYearTooEarly is returned when the published year is before 1000.
	ErrPublishedYearTooEarly = errors.New("Published year must be after year 1000")
	// ErrSummaryTooLong is returned when a summary exceeds maxSummaryLen characters.
	ErrSummaryTooLong = errors.New("ensure this value has at most 2000 characters")
)

// BookCreate is the request DTO for creating a book. It is the Go analog of
// the Pydantic BookCreate model. summary is optional and modeled as a
// pointer so a caller can distinguish "absent" from "empty string".
type BookCreate struct {
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	PublishedYear int     `json:"published_year"`
	Summary       *string `json:"summary,omitempty"`
}

// Validate replicates the Pydantic BookCreate field_validators. It trims
// whitespace in place (mirroring str_strip_whitespace) and returns the first
// validation error encountered, or nil if the input is valid.
func (b *BookCreate) Validate() error {
	b.Title = strings.TrimSpace(b.Title)
	if b.Title == "" {
		return ErrTitleEmpty
	}
	if len(b.Title) > maxTitleLen {
		return ErrTitleTooLong
	}

	b.Author = strings.TrimSpace(b.Author)
	if b.Author == "" {
		return ErrAuthorEmpty
	}
	if len(b.Author) > maxAuthorLen {
		return ErrAuthorTooLong
	}

	if err := validatePublishedYear(b.PublishedYear); err != nil {
		return err
	}

	summary, err := normalizeSummary(b.Summary)
	if err != nil {
		return err
	}
	b.Summary = summary

	return nil
}

// BookUpdate is the request DTO for partially updating a book. It is the Go
// analog of the Pydantic BookUpdate model: every field is optional (a nil
// pointer means "leave unchanged"), and validation is skipped for fields that
// are absent.
type BookUpdate struct {
	Title         *string `json:"title,omitempty"`
	Author        *string `json:"author,omitempty"`
	PublishedYear *int    `json:"published_year,omitempty"`
	Summary       *string `json:"summary,omitempty"`
}

// Validate replicates the Pydantic BookUpdate field_validators. Each field is
// validated only when present (non-nil), mirroring the source's guard clauses
// that returned early when the value was None. Present string fields are
// trimmed in place.
func (b *BookUpdate) Validate() error {
	if b.Title != nil {
		title := strings.TrimSpace(*b.Title)
		if title == "" {
			return ErrTitleEmpty
		}
		if len(title) > maxTitleLen {
			return ErrTitleTooLong
		}
		b.Title = &title
	}

	if b.Author != nil {
		author := strings.TrimSpace(*b.Author)
		if author == "" {
			return ErrAuthorEmpty
		}
		if len(author) > maxAuthorLen {
			return ErrAuthorTooLong
		}
		b.Author = &author
	}

	// MIGRATION_NOTE: this is the "ValidateUpdate guard" called out in the
	// migration notes — published_year validation is skipped entirely when
	// the field is nil, matching the source's `if v is None: return None`.
	if b.PublishedYear != nil {
		if err := validatePublishedYear(*b.PublishedYear); err != nil {
			return err
		}
	}

	if b.Summary != nil {
		summary, err := normalizeSummary(b.Summary)
		if err != nil {
			return err
		}
		b.Summary = summary
	}

	return nil
}

// BookResponse is the response DTO returned to API clients. It is the Go
// analog of the Pydantic BookResponse model (from_attributes=True), which was
// built directly from the ORM Book. NewBookResponse constructs it from a Book.
type BookResponse struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	PublishedYear int     `json:"published_year"`
	Summary       *string `json:"summary,omitempty"`
}

// NewBookResponse builds a BookResponse from a Book model. This replaces
// Pydantic's from_attributes / model_validate(orm_obj) behavior.
func NewBookResponse(b Book) BookResponse {
	return BookResponse{
		ID:            b.ID,
		Title:         b.Title,
		Author:        b.Author,
		PublishedYear: b.PublishedYear,
		Summary:       b.Summary,
	}
}

// validatePublishedYear replicates the shared published_year validation logic
// from both Pydantic schemas: the year must be at least 1000 and not in the
// future relative to the current year.
func validatePublishedYear(year int) error {
	if year < minPublishedYear {
		return ErrPublishedYearTooEarly
	}
	currentYear := time.Now().Year()
	if year > currentYear {
		return fmt.Errorf("Published year cannot be in the future (current year: %d)", currentYear)
	}
	return nil
}

// normalizeSummary replicates the shared summary validator: nil stays nil, a
// blank-after-trim value collapses to nil, and an over-long value is rejected.
// The returned pointer holds the trimmed value.
func normalizeSummary(v *string) (*string, error) {
	if v == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil, nil
	}
	if len(trimmed) > maxSummaryLen {
		return nil, ErrSummaryTooLong
	}
	return &trimmed, nil
}
