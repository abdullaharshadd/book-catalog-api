// internal/schemas.go

package schemas

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"migrated-app/internal/model"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// BookCreate represents the schema for creating a new book.
type BookCreate struct {
	Title         string `validate:"required,max=255"`
	Author        string `validate:"required,max=255"`
	PublishedYear int    `validate:"gt=999,lte=current_year"`
	Summary       *string `validate:"max=2000"`
}

// Validate ensures the BookCreate struct is valid.
func (b *BookCreate) Validate() error {
	if err := validate.Var(b.Title, "required,max=255"); err != nil {
		return errors.New("Title cannot be empty and must be at most 255 characters")
	}
	if err := validate.Var(b.Author, "required,max=255"); err != nil {
		return errors.New("Author cannot be empty and must be at most 255 characters")
	}
	currentYear := time.Now().Year()
	if b.PublishedYear < 1000 || b.PublishedYear > currentYear {
		return errors.New(fmt.Sprintf("Published year must be between 1000 and %d", currentYear))
	}
	if b.Summary != nil {
		if err := validate.Var(*b.Summary, "max=2000"); err != nil {
			return errors.New("Summary must be at most 2000 characters")
		}
		*b.Summary = strings.TrimSpace(*b.Summary)
	}
	return nil
}

// BookUpdate represents the schema for updating a book.
type BookUpdate struct {
	Title         *string `validate:"omitempty,max=255"`
	Author        *string `validate:"omitempty,max=255"`
	PublishedYear *int    `validate:"omitempty,gt=999,lte=current_year"`
	Summary       *string `validate:"omitempty,max=2000"`
}

// Validate ensures the BookUpdate struct is valid.
func (b *BookUpdate) Validate() error {
	if b.Title != nil {
		if err := validate.Var(*b.Title, "max=255"); err != nil {
			return errors.New("Title must be at most 255 characters")
		}
		*b.Title = strings.TrimSpace(*b.Title)
		if *b.Title == "" {
			return errors.New("Title cannot be empty")
		}
	}
	if b.Author != nil {
		if err := validate.Var(*b.Author, "max=255"); err != nil {
			return errors.New("Author must be at most 255 characters")
		}
		*b.Author = strings.TrimSpace(*b.Author)
		if *b.Author == "" {
			return errors.New("Author cannot be empty")
		}
	}
	if b.PublishedYear != nil {
		currentYear := time.Now().Year()
		if *b.PublishedYear < 1000 || *b.PublishedYear > currentYear {
			return errors.New(fmt.Sprintf("Published year must be between 1000 and %d", currentYear))
		}
	}
	if b.Summary != nil {
		if err := validate.Var(*b.Summary, "max=2000"); err != nil {
			return errors.New("Summary must be at most 2000 characters")
		}
		*b.Summary = strings.TrimSpace(*b.Summary)
	}
	return nil
}

// BookResponse represents the schema for responding with book information.
type BookResponse struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	Author        string `json:"author"`
	PublishedYear int    `json:"published_year"`
	Summary       *string `json:"summary,omitempty"`
}

// FromModel converts a model.Book to a BookResponse.
func FromModel(book *model.Book) *BookResponse {
	return &BookResponse{
		ID:            book.ID,
		Title:         book.Title,
		Author:        book.Author,
		PublishedYear: book.PublishedYear,
		Summary:       model.NewOptionalString(book.Summary),
	}
}

// NewOptionalString creates a new optional string.
func NewOptionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// MIGRATION_NOTE: The validation logic in the source Pydantic models has been
// translated into Go using the go-playground/validator package. The validation
// methods are called explicitly in the Validate functions to ensure that all
// business logic is preserved. The current year validation is handled by
// fetching the current year during validation.