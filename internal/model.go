package internal

import (
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

// Book represents a book in the catalog.
type Book struct {
	ID             int    `db:"id" json:"id"`
	Title          string `db:"title" json:"title"`
	Author         string `db:"author" json:"author"`
	PublishedYear  int    `db:"published_year" json:"published_year"`
	Summary        sql.NullString `db:"summary" json:"summary"`
}

// CreateBook inserts a new book into the database and returns the ID of the newly created book.
func CreateBook(ctx context.Context, db *sqlx.DB, book Book) (int, error) {
	var id int
	query := `INSERT INTO books (title, author, published_year, summary) VALUES ($1, $2, $3, $4) RETURNING id`
	x := db.MustRebind(query)
	
	// Handle optional summary
	summary := pq.NullString{String: book.Summary.String, Valid: book.Summary.Valid}
	
	err := db.GetContext(ctx, &id, x, book.Title, book.Author, book.PublishedYear, summary)
	if err != nil {
		log.Error().Err(err).Msg("failed to create book")
		return 0, err
	}
	return id, nil
}

// GetBook retrieves a book from the database by its ID.
func GetBook(ctx context.Context, db *sqlx.DB, id int) (*Book, error) {
	var book Book
	query := `SELECT id, title, author, published_year, summary FROM books WHERE id = $1`
	x := db.MustRebind(query)
	
	err := db.GetContext(ctx, &book, x, id)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debug().Msg("book not found")
			return nil, nil
		}
		log.Error().Err(err).Msg("failed to get book")
		return nil, err
	}
	return &book, nil
}

// ListBooks retrieves all books from the database.
func ListBooks(ctx context.Context, db *sqlx.DB) ([]Book, error) {
	var books []Book
	query := `SELECT id, title, author, published_year, summary FROM books`
	x := db.MustRebind(query)
	
	err := db.SelectContext(ctx, &books, x)
	if err != nil {
		log.Error().Err(err).Msg("failed to list books")
		return nil, err
	}
	return books, nil
}

// DeleteBook removes a book from the database by its ID.
func DeleteBook(ctx context.Context, db *sqlx.DB, id int) error {
	query := `DELETE FROM books WHERE id = $1`
	x := db.MustRebind(query)
	
	err := db.ExecContext(ctx, x, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete book")
		return err
	}
	return nil
}

// UpdateBook updates a book in the database by its ID.
func UpdateBook(ctx context.Context, db *sqlx.DB, book Book) error {
	// Handle optional summary
	summary := pq.NullString{String: book.Summary.String, Valid: book.Summary.Valid}
	query := `UPDATE books SET title = $1, author = $2, published_year = $3, summary = $4 WHERE id = $5`
	x := db.MustRebind(query)
	
	err := db.ExecContext(ctx, x, book.Title, book.Author, book.PublishedYear, summary, book.ID)
	if err != nil {
		log.Error().Err(err).Msg("failed to update book")
		return err
	}
	return nil
}
