package internal

import (
	"errors"
	"strings"
	"time"
)

// BookCreate represents a request to create a new book.
type BookCreate struct {
	Title          string  `json:"title"`
	Author         string  `json:"author"`
	PublishedYear  int     `json:"published_year"`
	Summary        *string `json:"summary"`
}

// Validate checks the validity of the BookCreate struct.
func (b *BookCreate) Validate() error {
	currentYear := time.Now().Year()
	if b.Title == "" || strings.TrimSpace(b.Title) == "" {
		return errors.New("Title cannot be empty")
	}
	if len(b.Title) > 255 {
		return errors.New("Title must have at most 255 characters")
	}
	b.Title = strings.TrimSpace(b.Title)

	if b.Author == "" || strings.TrimSpace(b.Author) == "" {
		return errors.New("Author cannot be empty")
	}
	if len(b.Author) > 255 {
		return errors.New("Author must have at most 255 characters")
	}
	b.Author = strings.TrimSpace(b.Author)

	if b.PublishedYear < 1000 || b.PublishedYear > currentYear {
		return errors.New("Published year must be between 1000 and the current year")
	}

	if b.Summary != nil {
		trimmedSummary := strings.TrimSpace(*b.Summary)
		if trimmedSummary == "" {
			b.Summary = nil
		} else if len(trimmedSummary) > 2000 {
			return errors.New("Summary must have at most 2000 characters")
		}
		*b.Summary = trimmedSummary
	}
	return nil
}

// BookUpdate represents a request to update a book.
type BookUpdate struct {
	Title          *string `json:"title"`
	Author         *string `json:"author"`
	PublishedYear  *int    `json:"published_year"`
	Summary        *string `json:"summary"`
}

// Validate checks the validity of the BookUpdate struct.
func (b *BookUpdate) Validate() error {
	currentYear := time.Now().Year()
	if b.Title != nil {
		if *b.Title == "" || strings.TrimSpace(*b.Title) == "" {
			return errors.New("Title cannot be empty")
		}
		if len(*b.Title) > 255 {
			return errors.New("Title must have at most 255 characters")
		}
		*b.Title = strings.TrimSpace(*b.Title)
	}

	if b.Author != nil {
		if *b.Author == "" || strings.TrimSpace(*b.Author) == "" {
			return errors.New("Author cannot be empty")
		}
		if len(*b.Author) > 255 {
			return errors.New("Author must have at most 255 characters")
		}
		*b.Author = strings.TrimSpace(*b.Author)
	}

	if b.PublishedYear != nil && (*b.PublishedYear < 1000 || *b.PublishedYear > currentYear) {
		return errors.New("Published year must be between 1000 and the current year")
	}

	if b.Summary != nil {
		trimmedSummary := strings.TrimSpace(*b.Summary)
		if trimmedSummary == "" {
			b.Summary = nil
		} else if len(trimmedSummary) > 2000 {
			return errors.New("Summary must have at most 2000 characters")
		}
		*b.Summary = trimmedSummary
	}
	return nil
}

// BookResponse represents a response containing book information.
type BookResponse struct {
	ID             int     `json:"id"`
	Title          string  `json:"title"`
	Author         string  `json:"author"`
	PublishedYear  int     `json:"published_year"`
	Summary        *string `json:"summary"`
}
