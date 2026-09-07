package schemas_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// Embedded implementation (schemas.go is empty; we define the types here so
// the test file is self-contained and can be compiled/run immediately).
// Move these types to internal/schemas.go and change the package to "schemas"
// (or whatever the module path resolves to) before running against real code.
// ─────────────────────────────────────────────────────────────────────────────

// ValidationError holds one or more field-level error messages.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field %q: %s", e.Field, e.Message)
}

// ValidationErrors is a slice of ValidationError returned from schema
// validation.
type ValidationErrors []*ValidationError

func (ve ValidationErrors) Error() string {
	msgs := make([]string, len(ve))
	for i, e := range ve {
		msgs[i] = e.Error()
	}
	return strings.Join(msgs, "; ")
}

func (ve ValidationErrors) HasField(field string) bool {
	for _, e := range ve {
		if e.Field == field {
			return true
		}
	}
	return false
}

func (ve ValidationErrors) MessageFor(field string) string {
	for _, e := range ve {
		if e.Field == field {
			return e.Message
		}
	}
	return ""
}

// ─── BookCreate ───────────────────────────────────────────────────────────────

// BookCreateInput is the raw input map fed into BookCreate validation.
type BookCreateInput struct {
	Title         *string
	Author        *string
	PublishedYear *int
	Summary       *string
}

// BookCreate is the validated/normalised result of a create-book request.
type BookCreate struct {
	Title         string
	Author        string
	PublishedYear int
	Summary       *string // nil when not provided or blank
}

// ValidateBookCreate validates and normalises raw input.
func ValidateBookCreate(in BookCreateInput) (*BookCreate, ValidationErrors) {
	var errs ValidationErrors

	// ── title ──────────────────────────────────────────────────────────────
	var title string
	if in.Title == nil {
		errs = append(errs, &ValidationError{Field: "title", Message: "field required"})
	} else {
		title = strings.TrimSpace(*in.Title)
		if title == "" {
			errs = append(errs, &ValidationError{Field: "title", Message: "Title cannot be empty"})
		} else if len(title) > 255 {
			errs = append(errs, &ValidationError{Field: "title", Message: "ensure this value has at most 255 characters"})
		}
	}

	// ── author ─────────────────────────────────────────────────────────────
	var author string
	if in.Author == nil {
		errs = append(errs, &ValidationError{Field: "author", Message: "field required"})
	} else {
		author = strings.TrimSpace(*in.Author)
		if author == "" {
			errs = append(errs, &ValidationError{Field: "author", Message: "Author cannot be empty"})
		} else if len(author) > 255 {
			errs = append(errs, &ValidationError{Field: "author", Message: "ensure this value has at most 255 characters"})
		}
	}

	// ── published_year ─────────────────────────────────────────────────────
	var publishedYear int
	currentYear := time.Now().Year()
	if in.PublishedYear == nil {
		errs = append(errs, &ValidationError{Field: "published_year", Message: "field required"})
	} else {
		publishedYear = *in.PublishedYear
		if publishedYear < 1000 {
			errs = append(errs, &ValidationError{Field: "published_year", Message: "Published year must be after year 1000"})
		} else if publishedYear > currentYear {
			errs = append(errs, &ValidationError{
				Field:   "published_year",
				Message: fmt.Sprintf("Published year cannot be in the future (current year: %d)", currentYear),
			})
		}
	}

	// ── summary ────────────────────────────────────────────────────────────
	var summary *string
	if in.Summary != nil {
		s := strings.TrimSpace(*in.Summary)
		if len(s) > 2000 {
			errs = append(errs, &ValidationError{Field: "summary", Message: "ensure this value has at most 2000 characters"})
		} else if s != "" {
			summary = &s
		}
		// empty/whitespace-only → leave summary as nil
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

// ─── BookUpdate ───────────────────────────────────────────────────────────────

// BookUpdateInput carries optional fields for a partial update.
type BookUpdateInput struct {
	Title         *string // nil = not provided
	Author        *string
	PublishedYear *int
	Summary       *string
}

// BookUpdate is the validated/normalised result of an update-book request.
type BookUpdate struct {
	Title         *string
	Author        *string
	PublishedYear *int
	Summary       *string // nil when omitted or blank
}

// ValidateBookUpdate validates and normalises optional fields.
func ValidateBookUpdate(in BookUpdateInput) (*BookUpdate, ValidationErrors) {
	var errs ValidationErrors
	currentYear := time.Now().Year()

	out := &BookUpdate{}

	// ── title ──────────────────────────────────────────────────────────────
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" {
			errs = append(errs, &ValidationError{Field: "title", Message: "Title cannot be empty"})
		} else if len(t) > 255 {
			errs = append(errs, &ValidationError{Field: "title", Message: "ensure this value has at most 255 characters"})
		} else {
			out.Title = &t
		}
	}

	// ── author ─────────────────────────────────────────────────────────────
	if in.Author != nil {
		a := strings.TrimSpace(*in.Author)
		if a == "" {
			errs = append(errs, &ValidationError{Field: "author", Message: "Author cannot be empty"})
		} else if len(a) > 255 {
			errs = append(errs, &ValidationError{Field: "author", Message: "ensure this value has at most 255 characters"})
		} else {
			out.Author = &a
		}
	}

	// ── published_year ─────────────────────────────────────────────────────
	if in.PublishedYear != nil {
		y := *in.PublishedYear
		if y < 1000 {
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

	// ── summary ────────────────────────────────────────────────────────────
	if in.Summary != nil {
		s := strings.TrimSpace(*in.Summary)
		if len(s) > 2000 {
			errs = append(errs, &ValidationError{Field: "summary", Message: "ensure this value has at most 2000 characters"})
		} else if s != "" {
			out.Summary = &s
		}
		// empty/whitespace-only → out.Summary stays nil
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return out, nil
}

// ─── BookResponse ─────────────────────────────────────────────────────────────

// BookSource is the interface satisfied by any object (e.g. a DB model) from
// which BookResponse can read attributes.
type BookSource interface {
	GetID() int
	GetTitle() string
	GetAuthor() string
	GetPublishedYear() int
	GetSummary() *string
}

// BookResponse is the serialised representation of a book resource.
type BookResponse struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	PublishedYear int     `json:"published_year"`
	Summary       *string `json:"summary"`
}

// NewBookResponse builds a BookResponse from any BookSource. Returns a
// ValidationError when a required field is missing/zero-valued.
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
	// Validate required fields
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

// ─────────────────────────────────────────────────────────────────────────────
// Helpers / test doubles
// ─────────────────────────────────────────────────────────────────────────────

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

// mockBookSource implements BookSource for tests.
type mockBookSource struct {
	id            int
	title         string
	author        string
	publishedYear int
	summary       *string
}

func (m *mockBookSource) GetID() int            { return m.id }
func (m *mockBookSource) GetTitle() string       { return m.title }
func (m *mockBookSource) GetAuthor() string      { return m.author }
func (m *mockBookSource) GetPublishedYear() int  { return m.publishedYear }
func (m *mockBookSource) GetSummary() *string    { return m.summary }

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

func currentYear() int { return time.Now().Year() }

// ══════════════════════════════════════════════════════════════════════════════
// BookCreate
// ══════════════════════════════════════════════════════════════════════════════

func TestValidateBookCreate(t *testing.T) {
	t.Parallel()

	year := currentYear()

	tests := []struct {
		name        string
		input       BookCreateInput
		wantErr     bool
		errField    string
		errContains string
		// expected output assertions (only checked when wantErr == false)
		wantTitle    string
		wantAuthor   string
		wantYear     int
		wantSummary  *string // nil means we expect nil
		hasSummary   bool    // true  → check wantSummary; false → expect nil
	}{
		// ── happy path ─────────────────────────────────────────────────────
		{
			name: "all valid fields – whitespace trimmed",
			input: BookCreateInput{
				Title:         strPtr("  Clean Code  "),
				Author:        strPtr("  Robert Martin  "),
				PublishedYear: intPtr(2008),
				Summary:       strPtr("  A great book.  "),
			},
			wantErr:    false,
			wantTitle:  "Clean Code",
			wantAuthor: "Robert Martin",
			wantYear:   2008,
			wantSummary: strPtr("A great book."),
			hasSummary:  true,
		},
		{
			name: "summary omitted – summary is nil",
			input: BookCreateInput{
				Title:         strPtr("Go Programming"),
				Author:        strPtr("Alan Donovan"),
				PublishedYear: intPtr(2015),
				Summary:       nil,
			},
			wantErr:    false,
			wantTitle:  "Go Programming",
			wantAuthor: "Alan Donovan",
			wantYear:   2015,
			hasSummary: false,
		},
		{
			name: "published_year equals current year – valid",
			input: BookCreateInput{
				Title:         strPtr("New Book"),
				Author:        strPtr("Someone"),
				PublishedYear: intPtr(year),
			},
			wantErr:    false,
			wantTitle:  "New Book",
			wantAuthor: "Someone",
			wantYear:   year,
			hasSummary: false,
		},
		{
			name: "published_year equals 1000 – valid boundary",
			input: BookCreateInput{
				Title:         strPtr("Old Book"),
				Author:        strPtr("Ancient Author"),
				PublishedYear: intPtr(1000),
			},
			wantErr:    false,
			wantTitle:  "Old Book",
			wantAuthor: "Ancient Author",
			wantYear:   1000,
			hasSummary: false,
		},
		{
			name: "summary empty string – normalised to nil",
			input: BookCreateInput{
				Title:         strPtr("Book"),
				Author:        strPtr("Author"),
				PublishedYear: intPtr(2000),
				Summary:       strPtr(""),
			},
			wantErr:    false,
			wantTitle:  "Book",
			wantAuthor: "Author",
			wantYear:   2000,
			hasSummary: false,
		},
		{
			name: "summary whitespace-only – normalised to nil",
			input: BookCreateInput{
				Title:         strPtr("Book"),
				Author:        strPtr("Author"),
				PublishedYear: intPtr(2000),
				Summary:       strPtr("   "),
			},
			wantErr:    false,
			wantTitle:  "Book",
			wantAuthor: "Author",
			wantYear:   2000,
			hasSummary: false,
		},

		// ── title errors ───────────────────────────────────────