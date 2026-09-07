package internal

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

type BookCreate struct {
	Title          string `json:"title"`
	Author         string `json:"author"`
	PublishedYear  int    `json:"published_year"`
	Summary        *string `json:"summary,omitempty"`
}

// Validate ensures that the fields of a BookCreate struct meet the required constraints.
func (b *BookCreate) Validate() error {
	if strings.TrimSpace(b.Title) == "" {
		return errors.New("Title cannot be empty")
	}
	if len(strings.TrimSpace(b.Title)) > 255 {
		return errors.New("ensure this value has at most 255 characters")
	}
	b.Title = strings.TrimSpace(b.Title)

	if strings.TrimSpace(b.Author) == "" {
		return errors.New("Author cannot be empty")
	}
	if len(strings.TrimSpace(b.Author)) > 255 {
		return errors.New("ensure this value has at most 255 characters")
	}
	b.Author = strings.TrimSpace(b.Author)

	currentYear := time.Now().Year()
	if b.PublishedYear < 1000 {
		return errors.New("Published year must be after year 1000")
	}
	if b.PublishedYear > currentYear {
		return errors.New("Published year cannot be in the future (current year: " + strconv.Itoa(currentYear) + ")")
	}

	if b.Summary != nil && strings.TrimSpace(*b.Summary) == "" {
		*b.Summary = ""
	}
	if b.Summary != nil && len(strings.TrimSpace(*b.Summary)) > 2000 {
		return errors.New("ensure this value has at most 2000 characters")
	}
	if b.Summary != nil {
		*b.Summary = strings.TrimSpace(*b.Summary)
	}
	return nil
}

type BookUpdate struct {
	Title          *string `json:"title,omitempty"`
	Author         *string `json:"author,omitempty"`
	PublishedYear  *int    `json:"published_year,omitempty"`
	Summary        *string `json:"summary,omitempty"`
}

// Validate ensures that the fields of a BookUpdate struct meet the required constraints.
func (b *BookUpdate) Validate() error {
	if b.Title != nil && strings.TrimSpace(*b.Title) == "" {
		return errors.New("Title cannot be empty")
	}
	if b.Title != nil && len(strings.TrimSpace(*b.Title)) > 255 {
		return errors.New("ensure this value has at most 255 characters")
	}
	if b.Title != nil {
		*b.Title = strings.TrimSpace(*b.Title)
	}

	if b.Author != nil && strings.TrimSpace(*b.Author) == "" {
		return errors.New("Author cannot be empty")
	}
	if b.Author != nil && len(strings.TrimSpace(*b.Author)) > 255 {
		return errors.New("ensure this value has at most 255 characters")
	}
	if b.Author != nil {
		*b.Author = strings.TrimSpace(*b.Author)
	}

	currentYear := time.Now().Year()
	if b.PublishedYear != nil && *b.PublishedYear < 1000 {
		return errors.New("Published year must be after year 1000")
	}
	if b.PublishedYear != nil && *b.PublishedYear > currentYear {
		return errors.New("Published year cannot be in the future (current year: " + strconv.Itoa(currentYear) + ")")
	}

	if b.Summary != nil && strings.TrimSpace(*b.Summary) == "" {
		*b.Summary = ""
	}
	if b.Summary != nil && len(strings.TrimSpace(*b.Summary)) > 2000 {
		return errors.New("ensure this value has at most 2000 characters")
	}
	if b.Summary != nil {
		*b.Summary = strings.TrimSpace(*b.Summary)
	}
	return nil
}

type BookResponse struct {
	ID             int    `json:"id"`
	Title          string `json:"title"`
	Author         string `json:"author"`
	PublishedYear  int    `json:"published_year"`
	Summary        *string `json:"summary,omitempty"`
}
