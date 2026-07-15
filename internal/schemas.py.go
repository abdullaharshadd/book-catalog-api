// Package catalog defines the data-transfer objects (DTOs) and validation
// logic used by the Book Catalog HTTP API.
//
// This file was migrated from app/schemas.py, which declared Pydantic v2
// models for creating, updating, and returning Book resources. The Pydantic
// concepts map onto idiomatic Go as follows:
//
//   - BaseModel + field type hints        -> plain structs with json struct tags.
//   - Optional[T] = None fields           -> pointer fields (*T); a nil pointer
//                                            means "field not provided".
//   - ConfigDict(str_strip_whitespace)    -> explicit strings.TrimSpace in Validate.
//   - @field_validator methods            -> a Validate() (or Normalize) method on
//                                            the struct that returns an error.
//   - ConfigDict(from_attributes=True)    -> a constructor that maps a Book model
//                                            (see internal/models.py.go) into the
//                                            response DTO (NewBookResponse).
//
// Unlike Pydantic, Go does not validate automatically on deserialization; the
// caller (an HTTP handler) must invoke Validate explicitly after decoding JSON.
// Because validation also normalizes input (trimming whitespace, collapsing
// empty strings to nil), Validate mutates the receiver in place and returns the
// first validation error encountered — matching Pydantic's fail-fast behaviour.
package catalog

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Field length limits, mirroring the Pydantic max_length constraints.
const (
	maxTitleLen   = 255
	maxAuthorLen  = 255
	maxSummaryLen = 2000
	minPublishedYear = 1000
)

// Validation errors returned by the DTO Validate methods. Callers can use
// errors.Is to distinguish them (e.g. to map to an HTTP 422 response).
var (
	// ErrTitleEmpty is returned when a required title is empty or whitespace.
	ErrTitleEmpty = errors.New("title cannot be empty")
	// ErrTitleTooLong is returned when a title exceeds maxTitleLen characters.
	ErrTitleTooLong = fmt.Errorf("ensure this value has at most %d characters", maxTitleLen)
	// ErrAuthorEmpty is returned when a required author is empty or whitespace.
	ErrAuthorEmpty = errors.New("author cannot be empty")
	// ErrAuthorTooLong is returned when an author exceeds maxAuthorLen characters.
	ErrAuthorTooLong = fmt.Errorf("ensure this value has at most %d characters", maxAuthorLen)
	// ErrPublishedYearTooEarly is returned when published_year is before year 1000.
	ErrPublishedYearTooEarly = fmt.Errorf("published year must be after year %d", minPublishedYear)
	// ErrSummaryTooLong is returned when a summary exceeds maxSummaryLen characters.
	ErrSummaryTooLong = fmt.Errorf("ensure this value has at most %d characters", maxSummaryLen)
)

// currentYear is a package-level indirection over time.Now so tests can pin the
// "current year" used by published_year validation. Production code uses the
// real clock.
var currentYear = func() int { return time.Now().Year() }

// errPublishedYearFuture builds the future-year error, embedding the current
// year exactly as the Pydantic validator did.
func errPublishedYearFuture(year int) error {
	return fmt.Errorf("published year cannot be in the future (current year: %d)", year)
}

// validateTitle checks and normalizes a required title, returning the trimmed
// value or a validation error.
func validateTitle(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", ErrTitleEmpty
	}
	if len(v) > maxTitleLen {
		return "", ErrTitleTooLong
	}
	return v, nil
}

// validateAuthor checks and normalizes a required author, returning the trimmed
// value or a validation error.
func validateAuthor(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", ErrAuthorEmpty
	}
	if len(v) > maxAuthorLen {
		return "", ErrAuthorTooLong
	}
	return v, nil
}

// validatePublishedYear enforces the [1000, currentYear] range on a publication
// year.
func validatePublishedYear(v int) (int, error) {
	if v < minPublishedYear {
		return 0, ErrPublishedYearTooEarly
	}
	if cy := currentYear(); v > cy {
		return 0, errPublishedYearFuture(cy)
	}
	return v, nil
}

// validateSummary normalizes an optional summary. A nil input, or a value that
// is empty after trimming, collapses to nil (mirroring the Pydantic validator
// which returned None). A too-long summary yields ErrSummaryTooLong.
func validateSummary(v *string) (*string, error) {
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

// BookCreate is the request DTO for creating a Book. All fields are required
// except Summary. Callers must call Validate after decoding to normalize and
// validate the payload.
type BookCreate struct {
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	PublishedYear int     `json:"published_year"`
	Summary       *string `json:"summary,omitempty"`
}

// Validate normalizes and validates the BookCreate payload in place, returning
// the first validation error encountered. On success all fields are trimmed and
// Summary is collapsed to nil when empty.
//
// MIGRATION_NOTE: Pydantic runs validators automatically during model
// construction. Go has no such hook, so HTTP handlers must call Validate after
// json.Decode. See internal/main.py.go for the call site.
func (b *BookCreate) Validate() error {
	title, err := validateTitle(b.Title)
	if err != nil {
		return err
	}
	b.Title = title

	author, err := validateAuthor(b.Author)
	if err != nil {
		return err
	}
	b.Author = author

	year, err := validatePublishedYear(b.PublishedYear)
	if err != nil {
		return err
	}
	b.PublishedYear = year

	summary, err := validateSummary(b.Summary)
	if err != nil {
		return err
	}
	b.Summary = summary

	return nil
}

// BookUpdate is the request DTO for a partial update of a Book. Every field is
// optional; a nil pointer means "leave this field unchanged". Callers must call
// Validate after decoding.
type BookUpdate struct {
	Title         *string `json:"title,omitempty"`
	Author        *string `json:"author,omitempty"`
	PublishedYear *int    `json:"published_year,omitempty"`
	Summary       *string `json:"summary,omitempty"`
}

// Validate normalizes and validates the BookUpdate payload in place. Nil
// (omitted) fields are left untouched, matching the Pydantic validators that
// returned None early. It returns the first validation error encountered.
func (b *BookUpdate) Validate() error {
	if b.Title != nil {
		title, err := validateTitle(*b.Title)
		if err != nil {
			return err
		}
		b.Title = &title
	}

	if b.Author != nil {
		author, err := validateAuthor(*b.Author)
		if err != nil {
			return err
		}
		b.Author = &author
	}

	if b.PublishedYear != nil {
		year, err := validatePublishedYear(*b.PublishedYear)
		if err != nil {
			return err
		}
		b.PublishedYear = &year
	}

	// validateSummary already handles the nil case and the collapse-to-nil
	// behaviour for empty strings.
	summary, err := validateSummary(b.Summary)
	if err != nil {
		return err
	}
	b.Summary = summary

	return nil
}

// BookResponse is the response DTO returned to API clients. It is a projection
// of the Book domain model (see internal/models.py.go).
type BookResponse struct {
	ID            int64   `json:"id"`
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	PublishedYear int     `json:"published_year"`
	Summary       *string `json:"summary,omitempty"`
}

// NewBookResponse maps a Book domain model into a BookResponse DTO. This is the
// Go equivalent of Pydantic's ConfigDict(from_attributes=True), which populated
// a schema from an ORM object's attributes.
//
// MIGRATION_NOTE: The field types here (ID int64, Summary *string) must match
// the Book struct in internal/models.py.go. Adjust the mapping below if the
// migrated Book model uses different field names or types.
func NewBookResponse(b Book) BookResponse {
	return BookResponse{
		ID:            b.ID,
		Title:         b.Title,
		Author:        b.Author,
		PublishedYear: b.PublishedYear,
		Summary:       b.Summary,
	}
}
