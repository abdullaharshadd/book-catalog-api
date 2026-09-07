// Package schemas defines the wire-format schemas for the Book resource:
// separate request schemas for creation (BookCreate) and partial update
// (BookUpdate), and a response schema (BookResponse) that reads attributes
// off a persistence model.
//
// This is the Go equivalent of the original Pydantic v2 schemas. Pydantic's
// field_validator classmethods become explicit validation functions;
// ConfigDict(str_strip_whitespace=True) becomes explicit strings.TrimSpace
// calls; and ConfigDict(from_attributes=True) (ORM mode) becomes the
// BookSource interface that any persistence model can satisfy.
package schemas

import (
	"fmt"
	"strings"
	"time"
)

// ValidationError holds a single field-level error message.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field %q: %s", e.Field, e.Message)
}

// ValidationErrors is a slice of ValidationError returned from schema
// validation. It aggregates every field-level failure so callers can report
// them all at once, matching Pydantic's behaviour of collecting all errors.
type ValidationErrors []*ValidationError

// Error implements the error interface by joining all field messages.
func (ve ValidationErrors) Error() string {
	msgs := make([]string, len(ve))
	for i, e := range ve {
		msgs[i] = e.Error()
	}
	return strings.Join(msgs, "; ")
}

// HasField reports whether any error targets the given field.
func (ve ValidationErrors) HasField(field string) bool {
	for _, e := range ve {
		if e.Field == field {
			return true
		}
	}
	return false
}

// MessageFor returns the first error message for the given field, or "".
func (ve ValidationErrors) MessageFor(field string) string {
	for _, e := range ve {
		if e.Field == field {
			return e.Message
		}
	}
	return ""
}

// Field length limits, derived from the source Pydantic validators.
const (
	maxTitleLen   = 255
	maxAuthorLen  = 255
	maxSummaryLen = 2000
	minYear       = 1000
)

// ─── BookCreate ──────────────────────────────────────────────────────────────

// BookCreateInput is the raw, unvalidated input for a create-book request.
// Pointer fields distinguish "not provided" (nil) from a provided value,
// mirroring Pydantic's required/optional field semantics.
type BookCreateInput struct {
	Title         *string
	Author        *string
	PublishedYear *int
	Summary       *string
}

// BookCreate is the validated and normalised result of a create-book request.
type BookCreate struct {
	Title         string
	Author        string
	PublishedYear int
	Summary       *string // nil when not provided or blank
}

// ValidateBookCreate validates and normalises raw create input, collecting
// every field-level failure. On success it returns the normalised BookCreate
// and a nil error.
func ValidateBookCreate(in BookCreateInput) (*BookCreate, ValidationErrors) {
	var errs ValidationErrors

	// title — required
	var title string
	if in.Title == nil {
		errs = append(errs, &ValidationError{Field: "title", Message: "field required"})
	} else {
		title = strings.TrimSpace(*in.Title)
		if title == "" {
			errs = append(errs, &ValidationError{Field: "title", Message: "Title cannot be empty"})
		} else if len(title) > maxTitleLen {
			errs = append(errs, &ValidationError{Field: "title", Message: "ensure this value has at most 255 characters"})
		}
	}

	// author — required
	var author string
	if in.Author == nil {
		errs = append(errs, &ValidationError{Field: "author", Message: "field required"})
	} else {
		author = strings.TrimSpace(*in.Author)
		if author == "" {
			errs = append(errs, &ValidationError{Field: "author", Message: "Author cannot be empty"})
		} else if len(author) > maxAuthorLen {
			errs = append(errs, &ValidationError{Field: "author", Message: "ensure this value has at most 255 characters"})
		}
	}

	// published_year — required
	var publishedYear int
	currentYear := time.Now().Year()
	if in.PublishedYear == nil {
		errs = append(errs, &ValidationError{Field: "published_year", Message: "field required"})
	} else {
		publishedYear = *in.PublishedYear
		if publishedYear < minYear {
			errs = append(errs, &ValidationError{Field: "published_year", Message: "Published year must be after year 1000"})
		} else if publishedYear > currentYear {
			errs = append(errs, &ValidationError{
				Field:   "published_year",
				Message: fmt.Sprintf("Published year cannot be in the future (current year: %d)", currentYear),
			})
		}
	}

	// summary — optional; blank/whitespace normalises to nil
	var summary *string
	if in.Summary != nil {
		s := strings.TrimSpace(*in.Summary)
		if len(s) > maxSummaryLen {
			errs = append(errs, &ValidationError{Field: "summary", Message: "ensure this value has at most 2000 characters"})
		} else if s != "" {
			summary = &s
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return &BookCreate{
		Title:         title,
		Author:        author,
		PublishedYear: publishedYear,
		Summary:       summary,
	}, nil
}

// ─── BookUpdate ──────────────────────────────────────────────────────────────

// BookUpdateInput carries optional fields for a partial update. A nil pointer
// means the field was not provided and should be left untouched.
type BookUpdateInput struct {
	Title         *string
	Author        *string
	PublishedYear *int
	Summary       *string
}

// BookUpdate is the validated and normalised result of an update-book request.
// A nil field means "no change" (or, for summary, blank normalised away).
type BookUpdate struct {
	Title         *string
	Author        *string
	PublishedYear *int
	Summary       *string
}

// ValidateBookUpdate validates and normalises optional update fields,
// collecting every field-level failure. Fields left nil in the input are
// simply skipped.
func ValidateBookUpdate(in BookUpdateInput) (*BookUpdate, ValidationErrors) {
	var errs ValidationErrors
	currentYear := time.Now().Year()

	out := &BookUpdate{}

	// title
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" {
			errs = append(errs, &ValidationError{Field: "title", Message: "Title cannot be empty"})
		} else if len(t) > maxTitleLen {
			errs = append(errs, &ValidationError{Field: "title", Message: "ensure this value has at most 255 characters"})
		} else {
			out.Title = &t
		}
	}

	// author
	if in.Author != nil {
		a := strings.TrimSpace(*in.Author)
		if a == "" {
			errs = append(errs, &ValidationError{Field: "author", Message: "Author cannot be empty"})
		} else if len(a) > maxAuthorLen {
			errs = append(errs, &ValidationError{Field: "author", Message: "ensure this value has at most 255 characters"})
		} else {
			out.Author = &a
		}
	}

	// published_year
	if in.PublishedYear != nil {
		y := *in.PublishedYear
		if y < minYear {
			errs = append(errs, &ValidationError{Field: "published_year", Message: "Published year must be after year 1000"})
		} else if y > currentYear {
			errs = append(errs, &ValidationError{
				Field:   "published_year",
				Message: fmt.Sprintf("Published year cannot be in the future (current year: %d)", currentYear),
			})
		} else {
			out.PublishedYear = &y
		}
	}

	// summary — blank/whitespace normalises to nil
	if in.Summary != nil {
		s := strings.TrimSpace(*in.Summary)
		if len(s) > maxSummaryLen {
			errs = append(errs, &ValidationError{Field: "summary", Message: "ensure this value has at most 2000 characters"})
		} else if s != "" {
			out.Summary = &s
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return out, nil
}

// ─── BookResponse ────────────────────────────────────────────────────────────

// BookSource is satisfied by any object (typically a persistence model) from
// which a BookResponse can be built. This is the Go equivalent of Pydantic's
// ConfigDict(from_attributes=True) ORM mode.
type BookSource interface {
	GetID() int
	GetTitle() string
	GetAuthor() string
	GetPublishedYear() int
	GetSummary() *string
}

// BookResponse is the serialised representation of a book resource returned
// to clients.
type BookResponse struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	PublishedYear int     `json:"published_year"`
	Summary       *string `json:"summary"`
}

// NewBookResponse builds a BookResponse from any BookSource, reading its
// attributes just as Pydantic's from_attributes ORM mode would. It returns a
// ValidationErrors when a required field is missing/zero-valued.
func NewBookResponse(src BookSource) (*BookResponse, error) {
	if src == nil {
		return nil, &ValidationError{Field: "source", Message: "source object is required"}
	}
	resp := &BookResponse{
		ID:            src.GetID(),
		Title:         src.GetTitle(),
		Author:        src.GetAuthor(),
		PublishedYear: src.GetPublishedYear(),
		Summary:       src.GetSummary(),
	}

	var errs ValidationErrors
	if resp.ID == 0 {
		errs = append(errs, &ValidationError{Field: "id", Message: "field required"})
	}
	if resp.Title == "" {
		errs = append(errs, &ValidationError{Field: "title", Message: "field required"})
	}
	if resp.Author == "" {
		errs = append(errs, &ValidationError{Field: "author", Message: "field required"})
	}
	if resp.PublishedYear == 0 {
		errs = append(errs, &ValidationError{Field: "published_year", Message: "field required"})
	}
	if len(errs) > 0 {
		return nil, errs
	}
	return resp, nil
}
