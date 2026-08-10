package internal

import (
	"errors"
	"time"
)

// BookCreate holds the fields required to create a new book.
type BookCreate struct {
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	PublishedYear *int    `json:"published_year"`
	Summary       *string `json:"summary"`
}

// Validate checks required fields.
func (b *BookCreate) Validate() error {
	if b.Title == "" {
		return errors.New("title is required")
	}
	if b.Author == "" {
		return errors.New("author is required")
	}
	return nil
}

// BookUpdate holds the optional fields for a partial update.
type BookUpdate struct {
	Title         *string `json:"title"`
	Author        *string `json:"author"`
	PublishedYear *int    `json:"published_year"`
	Summary       *string `json:"summary"`
}

// Validate checks that any supplied string fields are non-empty.
func (b *BookUpdate) Validate() error {
	if b.Title != nil && *b.Title == "" {
		return errors.New("title must not be empty")
	}
	if b.Author != nil && *b.Author == "" {
		return errors.New("author must not be empty")
	}
	return nil
}

// BookResponse is the API response shape for a book.
type BookResponse struct {
	ID            int64     `json:"id"`
	Title         string    `json:"title"`
	Author        string    `json:"author"`
	PublishedYear *int      `json:"published_year"`
	Summary       *string   `json:"summary"`
	CreatedAt     time.Time `json:"created_at"`
}

// NewBookResponse converts a Book model to a BookResponse.
func NewBookResponse(b Book) BookResponse {
	return BookResponse{
		ID:            b.ID,
		Title:         b.Title,
		Author:        b.Author,
		PublishedYear: b.PublishedYear,
		Summary:       b.Summary,
		CreatedAt:     b.CreatedAt,
	}
}