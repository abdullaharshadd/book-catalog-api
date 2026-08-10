package internal

import "time"

// Book is the database row model for the books table.
type Book struct {
	ID            int64      `db:"id"`
	Title         string     `db:"title"`
	Author        string     `db:"author"`
	PublishedYear *int       `db:"published_year"`
	Summary       *string    `db:"summary"`
	CreatedAt     time.Time  `db:"created_at"`
}