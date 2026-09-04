package internal

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

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

	if b.Summary.Valid {
		trimmedSummary := strings.TrimSpace(b.Summary.String)
		if trimmedSummary == "" {
			b.Summary = sql.NullString{Valid: false}
		} else if len(trimmedSummary) > 2000 {
			return errors.New("Summary must have at most 2000 characters")
		} else {
			b.Summary = sql.NullString{String: trimmedSummary, Valid: true}
		}
	}
	return nil
}

// Validate checks the validity of the BookUpdate struct.
func (b *BookUpdate) Validate() error {
	currentYear := time.Now().Year()
	if b.Title != "" {
		if strings.TrimSpace(b.Title) == "" {
			return errors.New("Title cannot be empty")
		}
		if len(b.Title) > 255 {
			return errors.New("Title must have at most 255 characters")
		}
		b.Title = strings.TrimSpace(b.Title)
	}

	if b.Author != "" {
		if strings.TrimSpace(b.Author) == "" {
			return errors.New("Author cannot be empty")
		}
		if len(b.Author) > 255 {
			return errors.New("Author must have at most 255 characters")
		}
		b.Author = strings.TrimSpace(b.Author)
	}

	if b.PublishedYear != 0 && (b.PublishedYear < 1000 || b.PublishedYear > currentYear) {
		return errors.New("Published year must be between 1000 and the current year")
	}

	if b.Summary.Valid {
		trimmedSummary := strings.TrimSpace(b.Summary.String)
		if trimmedSummary == "" {
			b.Summary = sql.NullString{Valid: false}
		} else if len(trimmedSummary) > 2000 {
			return errors.New("Summary must have at most 2000 characters")
		} else {
			b.Summary = sql.NullString{String: trimmedSummary, Valid: true}
		}
	}
	return nil
}