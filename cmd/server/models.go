package main

// Book is the model for a book record.
type Book struct {
	ID            int    `db:"id" json:"id"`
	Title         string `db:"title" json:"title"`
	Author        string `db:"author" json:"author"`
	PublishedYear int    `db:"published_year" json:"published_year"`
	Summary       string `db:"summary" json:"summary"`
}