package schemas

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/datatypes"
	"migrated-app/internal/model"
)

// BookCreate represents a request to create a new book.
type BookCreate struct {
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	PublishedYear int     `json:"published_year"`
	Summary       *string `json:"summary,omitempty"`
}

// Validate performs validation on the BookCreate struct.
func (bc *BookCreate) Validate() error {
	if strings.TrimSpace(bc.Title) == "" {
		return errors.New("Title cannot be empty")
	}
	if len(strings.TrimSpace(bc.Title)) > 255 {
		return errors.New("ensure this value has at most 255 characters")
	}
	bc.Title = strings.TrimSpace(bc.Title)

	if strings.TrimSpace(bc.Author) == "" {
		return errors.New("Author cannot be empty")
	}
	if len(strings.TrimSpace(bc.Author)) > 255 {
		return errors.New("ensure this value has at most 255 characters")
	}
	bc.Author = strings.TrimSpace(bc.Author)

	currentYear := time.Now().Year()
	if bc.PublishedYear < 1000 {
		return errors.New("Published year must be after year 1000")
	}
	if bc.PublishedYear > currentYear {
		return fmt.Errorf("Published year cannot be in the future (current year: %d)", currentYear)
	}

	if bc.Summary != nil {
		trimmedSummary := strings.TrimSpace(*bc.Summary)
		if trimmedSummary == "" {
			bc.Summary = nil
		 else if len(trimmedSummary) > 2000 {
			return errors.New("ensure this value has at most 2000 characters")
		 else {
			*bc.Summary = trimmedSummary
		}
	}

	return nil
}

// BookUpdate represents a request to update an existing book.
type BookUpdate struct {
	Title         *string `json:"title,omitempty"`
	Author        *string `json:"author,omitempty"`
	PublishedYear *int    `json:"published_year,omitempty"`
	Summary       *string `json:"summary,omitempty"`
}

// Validate performs validation on the BookUpdate struct.
func (bu *BookUpdate) Validate() error {
	if bu.Title != nil {
		if strings.TrimSpace(*bu.Title) == "" {
			return errors.New("Title cannot be empty")
		}
		if len(strings.TrimSpace(*bu.Title)) > 255 {
			return errors.New("ensure this value has at most 255 characters")
		}
		*bu.Title = strings.TrimSpace(*bu.Title)
	}

	if bu.Author != nil {
		if strings.TrimSpace(*bu.Author) == "" {
			return errors.New("Author cannot be empty")
		}
		if len(strings.TrimSpace(*bu.Author)) > 255 {
			return errors.New("ensure this value has at most 255 characters")
		}
		*bu.Author = strings.TrimSpace(*bu.Author)
	}

	if bu.PublishedYear != nil {
		currentYear := time.Now().Year()
		if *bu.PublishedYear < 1000 {
			return errors.New("Published year must be after year 1000")
		}
		if *bu.PublishedYear > currentYear {
			return fmt.Errorf("Published year cannot be in the future (current year: %d)", currentYear)
		}
	}

	if bu.Summary != nil {
		trimmedSummary := strings.TrimSpace(*bu.Summary)
		if trimmedSummary == "" {
			bu.Summary = nil
		 else if len(trimmedSummary) > 2000 {
			return errors.New("ensure this value has at most 2000 characters")
		 else {
			*bu.Summary = trimmedSummary
		}
	}

	return nil
}

// BookResponse represents the response schema for a book.
type BookResponse struct {
	ID            uint           `json:"id"`
	Title         string         `json:"title"`
	Author        string         `json:"author"`
	PublishedYear int            `json:"published_year"`
	Summary       *datatypes.Text `json:"summary,omitempty"`
}