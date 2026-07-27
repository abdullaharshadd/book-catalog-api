package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// MIGRATION_NOTE: The Python source used Pydantic v2 BaseModel DTOs with
// field_validator decorators and ConfigDict(str_strip_whitespace=True,
// from_attributes=True). Go has no declarative validation framework in the
// standard library, so validation is expressed as explicit Validate methods
// that return an error. The str_strip_whitespace behaviour is reproduced by
// trimming inside the validators. from_attributes (ORM mode) is reproduced by
// an explicit constructor that maps a Book domain struct to a BookResponse.
//
// Each schema returns (T, error) rather than raising, per Go conventions. The
// error messages are preserved verbatim from the Python source so API clients
// observe identical validation feedback.

const (
	maxTitleLen   = 255
	maxAuthorLen  = 255
	maxSummaryLen = 2000
	minPublishedYear = 1000
)

// ValidationError describes a single field validation failure. It mirrors the
// ValueError messages raised by the original Pydantic validators.
type ValidationError struct {
	// Field is the name of the offending field (e.g. "title").
	Field string
	// Message is the human-readable validation message.
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// newValidationError constructs a *ValidationError for the given field.
func newValidationError(field, msg string) *ValidationError {
	return &ValidationError{Field: field, Message: msg}
}

// validateTitle applies the shared title rules: non-empty after trimming and at
// most 255 characters. It returns the trimmed value.
func validateTitle(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", newValidationError("title", "Title cannot be empty")
	}
	if len(v) > maxTitleLen {
		return "", newValidationError("title", "ensure this value has at most 255 characters")
	}
	return v, nil
}

// validateAuthor applies the shared author rules: non-empty after trimming and
// at most 255 characters. It returns the trimmed value.
func validateAuthor(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", newValidationError("author", "Author cannot be empty")
	}
	if len(v) > maxAuthorLen {
		return "", newValidationError("author", "ensure this value has at most 255 characters")
	}
	return v, nil
}

// validatePublishedYear applies the shared published_year rules: not before
// year 1000 and not in the future relative to the current year.
func validatePublishedYear(v int) (int, error) {
	currentYear := time.Now().Year()
	if v < minPublishedYear {
		return 0, newValidationError("published_year", "Published year must be after year 1000")
	}
	if v > currentYear {
		return 0, newValidationError("published_year",
			fmt.Sprintf("Published year cannot be in the future (current year: %d)", currentYear))
	}
	return v, nil
}

// validateSummary applies the shared summary rules. A summary that is nil or
// blank after trimming normalises to nil (SQL NULL). A trimmed summary longer
// than 2000 characters is rejected. It returns the normalised value.
func validateSummary(v *string) (*string, error) {
	if v == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil, nil
	}
	if len(trimmed) > maxSummaryLen {
		return nil, newValidationError("summary", "ensure this value has at most 2000 characters")
	}
	return &trimmed, nil
}

// BookCreate is the request DTO for creating a book. All fields except Summary
// are required.
//
// MIGRATION_NOTE: Corresponds to Pydantic's BookCreate. Summary uses a pointer
// so the absence of a value (JSON null / missing) is distinguishable from an
// empty string, matching Optional[str] = None semantics.
type BookCreate struct {
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	PublishedYear int     `json:"published_year"`
	Summary       *string `json:"summary"`
}

// Validate normalises and validates the BookCreate payload in place, returning
// the first validation error encountered (or nil if valid). On success the
// receiver's string fields are trimmed and Summary is normalised.
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

// BookUpdate is the request DTO for partially updating a book (PATCH
// semantics). Every field is optional; only fields explicitly present in the
// request JSON are considered set.
//
// MIGRATION_NOTE: Pydantic distinguishes "field omitted" from "field set to
// null" via model_fields_set. Go's encoding/json cannot express that with a
// plain struct of pointers alone, because a JSON null and an absent key both
// decode to a nil pointer. To preserve PATCH semantics we implement a custom
// UnmarshalJSON that records which keys were physically present in the payload
// (the `present` set), and expose Present() so the repository layer can build
// an UPDATE touching only the supplied columns. The id key never enters the
// present set.
type BookUpdate struct {
	Title         *string `json:"title"`
	Author        *string `json:"author"`
	PublishedYear *int    `json:"published_year"`
	Summary       *string `json:"summary"`

	// present records which updatable fields were physically present in the
	// decoded JSON. Only "title", "author", "published_year" and "summary" may
	// ever appear here.
	present map[string]bool
}

// UnmarshalJSON decodes a BookUpdate while recording which updatable fields
// were physically present in the payload.
func (b *BookUpdate) UnmarshalJSON(data []byte) error {
	// First decode into a raw map so we can detect key presence.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	b.present = make(map[string]bool)

	if rawTitle, ok := raw["title"]; ok {
		var v *string
		if err := json.Unmarshal(rawTitle, &v); err != nil {
			return err
		}
		b.Title = v
		b.present["title"] = true
	}
	if rawAuthor, ok := raw["author"]; ok {
		var v *string
		if err := json.Unmarshal(rawAuthor, &v); err != nil {
			return err
		}
		b.Author = v
		b.present["author"] = true
	}
	if rawYear, ok := raw["published_year"]; ok {
		var v *int
		if err := json.Unmarshal(rawYear, &v); err != nil {
			return err
		}
		b.PublishedYear = v
		b.present["published_year"] = true
	}
	if rawSummary, ok := raw["summary"]; ok {
		var v *string
		if err := json.Unmarshal(rawSummary, &v); err != nil {
			return err
		}
		b.Summary = v
		b.present["summary"] = true
	}

	return nil
}

// Present reports whether the named field was physically present in the decoded
// JSON payload. Valid field names are "title", "author", "published_year" and
// "summary".
func (b *BookUpdate) Present(field string) bool {
	if b.present == nil {
		return false
	}
	return b.present[field]
}

// Validate normalises and validates only the fields that were present in the
// update payload. Fields that were omitted are skipped entirely, preserving
// PATCH semantics. It returns the first validation error encountered.
//
// Note: for a present title/author, a nil (JSON null) value is treated as an
// empty string and therefore rejected, matching the Python validators which
// short-circuit on None only — here presence implies the client intended to
// set the value, and a null title/author is invalid.
func (b *BookUpdate) Validate() error {
	if b.Present("title") {
		if b.Title == nil {
			return newValidationError("title", "Title cannot be empty")
		}
		title, err := validateTitle(*b.Title)
		if err != nil {
			return err
		}
		b.Title = &title
	}

	if b.Present("author") {
		if b.Author == nil {
			return newValidationError("author", "Author cannot be empty")
		}
		author, err := validateAuthor(*b.Author)
		if err != nil {
			return err
		}
		b.Author = &author
	}

	if b.Present("published_year") {
		if b.PublishedYear == nil {
			return newValidationError("published_year", "Published year must be after year 1000")
		}
		year, err := validatePublishedYear(*b.PublishedYear)
		if err != nil {
			return err
		}
		b.PublishedYear = &year
	}

	if b.Present("summary") {
		summary, err := validateSummary(b.Summary)
		if err != nil {
			return err
		}
		b.Summary = summary
	}

	return nil
}

// BookResponse is the response DTO returned to API clients.
//
// MIGRATION_NOTE: Corresponds to Pydantic's BookResponse with
// from_attributes=True (ORM mode). That behaviour is reproduced by
// NewBookResponse, which maps a domain Book to a response explicitly. Summary
// deliberately has NO omitempty tag so that a null summary serialises to a
// JSON `null` rather than being dropped from the object — matching the
// Pydantic model where summary is always present in the serialised output.
type BookResponse struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	PublishedYear int     `json:"published_year"`
	Summary       *string `json:"summary"`
}

// NewBookResponse builds a BookResponse from a domain Book, mapping fields
// explicitly. This is the Go equivalent of Pydantic's from_attributes ORM mode.
//
// MIGRATION_NOTE: The exact field names/types on Book are defined in
// internal/model.go. This constructor assumes Book exposes ID, Title, Author,
// PublishedYear and a nullable Summary. Adjust the mapping if model.go names
// differ.
func NewBookResponse(b *Book) (BookResponse, error) {
	if b == nil {
		return BookResponse{}, errors.New("cannot build BookResponse from nil Book")
	}
	return BookResponse{
		ID:            b.ID,
		Title:         b.Title,
		Author:        b.Author,
		PublishedYear: b.PublishedYear,
		Summary:       b.Summary,
	}, nil
}
