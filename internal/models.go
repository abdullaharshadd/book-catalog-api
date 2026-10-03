package internal

import "time"

// Book is the persisted book record.
type Book struct {
	ID            int64     `json:"id" db:"id"`
	Title         string    `json:"title" db:"title"`
	Author        string    `json:"author" db:"author"`
	PublishedYear int       `json:"published_year" db:"published_year"`
	Summary       string    `json:"summary" db:"summary"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}
