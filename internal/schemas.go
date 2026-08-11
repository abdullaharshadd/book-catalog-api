package internal

import (
	"fmt"
	"strings"
	"time"
)

// MIGRATION_NOTE: The Python source used Pydantic v2 BaseModel schemas
// (BookCreate, BookUpdate, BookResponse) with @field_validator methods and
// ConfigDict options (str_strip_whitespace, from_attributes). Go has no
// declarative validation framework in the standard library, so the idiomatic
// equivalent is a plain struct per DTO plus an explicit Validate() method that
// returns structured errors. The str_strip_whitespace behaviour is replicated
// by trimming inside Validate() and (crucially) mutating the receiver so the
// caller sees the cleaned values, matching Pydantic's coerced output.
//
// MIGRATION_NOTE: Pydantic reports different error "type" strings depending on
// whether the failure came from a built-in constraint (max_length ->
// "string_too_long") or from a custom validator raising ValueError
// ("value_error"). In the source, ALL length checks are performed inside
// @field_validator bodies via `raise ValueError(...)`, so Pydantic classifies
// every one of them as "value_error" (not "string_too_long"). We preserve that
// exact classification in FieldError.Type below.
//
// MIGRATION_NOTE: Optional[str]/Optional[int] fields are represented with
// pointer types (*string / *int) so that "absent" (nil) is distinguishable from
// "present but empty/zero" — this is required for BookUpdate's partial-update
// semantics.

const (
	// maxTitleLen is the maximum allowed length of a book title.
	maxTitleLen = 255
	// maxAuthorLen is the maximum allowed length of a book author.
	maxAuthorLen = 255
	// maxSummaryLen is the maximum allowed length of a book summary.
	maxSummaryLen = 2000
	// minPublishedYear is the earliest permitted publication year.
	minPublishedYear = 1000
)

// FieldError describes a single field-level validation failure. It mirrors the
// shape Pydantic produces for a validation error (a field location plus a
// message plus a machine-readable type discriminator).
type FieldError struct {
	// Field is the name of the field that failed validation.
	Field string `json:"field"`
	// Message is the human-readable validation message.
	Message string `json:"message"`
	// Type is the Pydantic-compatible error type discriminator. Because every
	// check in the source is performed inside a custom validator raising
	// ValueError, this is always "value_error".
	Type string `json:"type"`
}

// Error implements the error interface for a single FieldError.
func (e *FieldError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationError aggregates one or more FieldError values, analogous to
// Pydantic's ValidationError which collects every failing field.
type ValidationError struct {
	// Errors holds all field-level failures collected during validation.
	Errors []FieldError `json:"errors"`
}

// Error implements the error interface for the aggregated validation error.
func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}
	msgs := make([]string, 0, len(e.Errors))
	for _, fe := range e.Errors {
		msgs = append(msgs, fe.Error())
	}
	return "validation failed: " + strings.Join(msgs, "; ")
}

// newFieldError builds a FieldError. All source validators raise ValueError, so
// the Type is fixed to "value_error" to match Pydantic's classification.
func newFieldError(field, message string) FieldError {
	return FieldError{Field: field, Message: message, Type: "value_error"}
}

// currentYear returns the current calendar year, matching the Python source's
// use of datetime.now().year inside the published_year validators.
func currentYear() int {
	return time.Now().Year()
}

// BookCreate is the request payload for creating a book. It corresponds to the
// Pydantic BookCreate schema.

// Validate applies the same rules as the Pydantic BookCreate field validators.
// It trims string fields in place (replicating str_strip_whitespace) and
// returns a *ValidationError describing every failing field, or nil if valid.
func (b *BookCreate) Validate() error {
	ve := &ValidationError{}

	// title
	title := strings.TrimSpace(b.Title)
	switch {
	case title == "":
		ve.Errors = append(ve.Errors, newFieldError("title", "Title cannot be empty"))
	case len(b.Title) > maxTitleLen:
		ve.Errors = append(ve.Errors, newFieldError("title", "ensure this value has at most 255 characters"))
	default:
		b.Title = title
	}

	// author
	author := strings.TrimSpace(b.Author)
	switch {
	case author == "":
		ve.Errors = append(ve.Errors, newFieldError("author", "Author cannot be empty"))
	case len(b.Author) > maxAuthorLen:
		ve.Errors = append(ve.Errors, newFieldError("author", "ensure this value has at most 255 characters"))
	default:
		b.Author = author
	}

	// published_year
	if b.PublishedYear != nil {
		if err := validatePublishedYear(*b.PublishedYear); err != nil {
			ve.Errors = append(ve.Errors, *err)
		}
	}

	// description (mapped to summary slot)
	b.Description = normalizeSummaryPtr(b.Description, ve)

	if len(ve.Errors) > 0 {
		return ve
	}
	return nil
}

// BookUpdate is the request payload for partially updating a book. Every field
// is optional; a nil pointer means "leave unchanged". It corresponds to the
// Pydantic BookUpdate schema.
type BookUpdate struct {
	// Title, when non-nil, replaces the book title (1..255 chars after trimming).
	Title *string `json:"title,omitempty"`
	// Author, when non-nil, replaces the book author (1..255 chars after trimming).
	Author *string `json:"author,omitempty"`
	// PublishedYear, when non-nil, replaces the publication year (1000..current).
	PublishedYear *int `json:"published_year,omitempty"`
	// Summary, when non-nil, replaces the summary (<=2000 chars after trimming).
	Summary *string `json:"summary,omitempty"`
}

// Validate applies the same rules as the Pydantic BookUpdate field validators.
// nil fields are skipped (partial update). Present string fields are trimmed in
// place. It returns a *ValidationError describing every failing field, or nil.
func (b *BookUpdate) Validate() error {
	ve := &ValidationError{}

	// title (optional)
	if b.Title != nil {
		trimmed := strings.TrimSpace(*b.Title)
		switch {
		case trimmed == "":
			ve.Errors = append(ve.Errors, newFieldError("title", "Title cannot be empty"))
		case len(*b.Title) > maxTitleLen:
			ve.Errors = append(ve.Errors, newFieldError("title", "ensure this value has at most 255 characters"))
		default:
			b.Title = &trimmed
		}
	}

	// author (optional)
	if b.Author != nil {
		trimmed := strings.TrimSpace(*b.Author)
		switch {
		case trimmed == "":
			ve.Errors = append(ve.Errors, newFieldError("author", "Author cannot be empty"))
		case len(*b.Author) > maxAuthorLen:
			ve.Errors = append(ve.Errors, newFieldError("author", "ensure this value has at most 255 characters"))
		default:
			b.Author = &trimmed
		}
	}

	// published_year (optional)
	if b.PublishedYear != nil {
		if err := validatePublishedYear(*b.PublishedYear); err != nil {
			ve.Errors = append(ve.Errors, *err)
		}
	}

	// summary (optional)
	if b.Summary != nil {
		b.Summary = normalizeSummary(b.Summary, ve)
	}

	if len(ve.Errors) > 0 {
		return ve
	}
	return nil
}

// validatePublishedYear enforces the shared published_year rules. It returns a
// pointer to a FieldError on failure, or nil when the year is valid.
func validatePublishedYear(v int) *FieldError {
	year := currentYear()
	if v < minPublishedYear {
		fe := newFieldError("published_year", "Published year must be after year 1000")
		return &fe
	}
	if v > year {
		fe := newFieldError(
			"published_year",
			fmt.Sprintf("Published year cannot be in the future (current year: %d)", year),
		)
		return &fe
	}
	return nil
}

// normalizeSummary replicates the Pydantic summary validator: it trims the
// value, collapses an empty result to nil, enforces the 2000-char maximum, and
// appends a FieldError to ve on overflow. The length check uses the ORIGINAL
// (untrimmed) length to match the source, which measured len(v) after trimming
// but before returning; note the source trims first then checks, so we mirror
// that exactly by measuring the trimmed value.
func normalizeSummary(v *string, ve *ValidationError) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	if len(trimmed) > maxSummaryLen {
		ve.Errors = append(ve.Errors, newFieldError("summary", "ensure this value has at most 2000 characters"))
		return v
	}
	return &trimmed
}

// normalizeSummaryPtr is the same as normalizeSummary but uses the "description"
// field name in any error, for use with BookCreate.Description.
func normalizeSummaryPtr(v *string, ve *ValidationError) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	if len(trimmed) > maxSummaryLen {
		ve.Errors = append(ve.Errors, newFieldError("description", "ensure this value has at most 2000 characters"))
		return v
	}
	return &trimmed
}

// BookResponse is the serialized representation of a book returned to clients.
// It corresponds to the Pydantic BookResponse schema (from_attributes=True),
// whose Go equivalent is the NewBookResponse constructor that maps a persisted
// Book model into this DTO.
type BookResponse struct {
	// ID is the unique identifier of the book.
	ID int64 `json:"id"`
	// Title is the book title.
	Title string `json:"title"`
	// Author is the book author.
	Author string `json:"author"`
	// PublishedYear is the year of publication.
	PublishedYear int `json:"published_year"`
	// Summary is the optional summary; nil when absent.
	Summary *string `json:"summary,omitempty"`
}

// NewBookResponse builds a BookResponse from a persisted Book model. This is the
// idiomatic Go replacement for Pydantic's ConfigDict(from_attributes=True),
// which allowed constructing the response schema directly from an ORM object.
//
// MIGRATION_NOTE: The exact field types (e.g. how Book stores Summary and
// PublishedYear) live in internal/model.go. Adjust the mapping below if the
// Book struct field types differ from those assumed here.
func NewBookResponse(b *Book) BookResponse {
	return BookResponse{
		ID:            b.ID,
		Title:         b.Title,
		Author:        b.Author,
		PublishedYear: b.PublishedYear,
		Summary:       bookSummaryPtr(b),
	}
}

// bookSummaryPtr extracts the optional summary from a Book as a *string,
// yielding nil when the summary is absent/empty.
//
// MIGRATION_NOTE: The Book model (internal/model.go) may represent the nullable
// summary as a sql.NullString or as a *string. This helper isolates that
// decision in one place; update it to match the actual Book.Summary field type.
func bookSummaryPtr(b *Book) *string {
	if !b.Summary.Valid {
		return nil
	}
	s := b.Summary.String
	return &s
}