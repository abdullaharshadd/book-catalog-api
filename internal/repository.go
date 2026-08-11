package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrDuplicateBook is returned when a book with the same title and author already exists.
var ErrDuplicateBook = errors.New("duplicate book")

// BookResponse is the JSON-serialisable representation of a Book.
type BookResponse struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Author      string  `json:"author"`
	Description *string `json:"description,omitempty"`
	Published   bool    `json:"published"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// NewBookResponse converts a Book to a BookResponse.
func NewBookResponse(b *Book) *BookResponse {
	resp := &BookResponse{
		ID:        b.ID,
		Title:     b.Title,
		Author:    b.Author,
		Published: b.Published,
		CreatedAt: b.CreatedAt.Format(time.RFC3339),
		UpdatedAt: b.UpdatedAt.Format(time.RFC3339),
	}
	if b.Description.Valid {
		resp.Description = &b.Description.String
	}
	return resp
}

// CreateRequest is the payload for POST /books/.
type CreateRequest struct {
	Title       string  `json:"title"`
	Author      string  `json:"author"`
	Description *string `json:"description"`
	Published   bool    `json:"published"`
}

// Validate checks required fields.
func (r *CreateRequest) Validate() error {
	if strings.TrimSpace(r.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if strings.TrimSpace(r.Author) == "" {
		return fmt.Errorf("author is required")
	}
	return nil
}

// UpdateRequest is the payload for PUT /books/{book_id}.
type UpdateRequest struct {
	Title       *string `json:"title"`
	Author      *string `json:"author"`
	Description *string `json:"description"`
	Published   *bool   `json:"published"`
}

// Validate checks that at least one field is provided.
func (r *UpdateRequest) Validate() error {
	if r.Title == nil && r.Author == nil && r.Description == nil && r.Published == nil {
		return fmt.Errorf("at least one field must be provided for update")
	}
	if r.Title != nil && strings.TrimSpace(*r.Title) == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if r.Author != nil && strings.TrimSpace(*r.Author) == "" {
		return fmt.Errorf("author cannot be empty")
	}
	return nil
}

// ListBooks returns a paginated list of books.
func (db *DB) ListBooks(ctx context.Context, skip, limit int) ([]Book, error) {
	rows, err := db.pool.QueryContext(ctx,
		`SELECT id, title, author, description, published, created_at, updated_at
		 FROM books ORDER BY id LIMIT $1 OFFSET $2`,
		limit, skip)
	if err != nil {
		return nil, fmt.Errorf("list books: %w", err)
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.Description, &b.Published, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("list books scan: %w", err)
		}
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list books rows: %w", err)
	}
	return books, nil
}

// GetBook retrieves a single book by ID. Returns sql.ErrNoRows if not found.
func (db *DB) GetBook(ctx context.Context, id int64) (*Book, error) {
	var b Book
	err := db.pool.QueryRowContext(ctx,
		`SELECT id, title, author, description, published, created_at, updated_at
		 FROM books WHERE id = $1`, id).
		Scan(&b.ID, &b.Title, &b.Author, &b.Description, &b.Published, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// CreateBook inserts a new book and returns it. Returns ErrDuplicateBook on uniqueness violation.
func (db *DB) CreateBook(ctx context.Context, req *CreateRequest) (*Book, error) {
	var desc sql.NullString
	if req.Description != nil {
		desc = sql.NullString{String: *req.Description, Valid: true}
	}

	var b Book
	err := db.pool.QueryRowContext(ctx,
		`INSERT INTO books (title, author, description, published)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, title, author, description, published, created_at, updated_at`,
		req.Title, req.Author, desc, req.Published).
		Scan(&b.ID, &b.Title, &b.Author, &b.Description, &b.Published, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if isDuplicateError(err) {
			return nil, ErrDuplicateBook
		}
		return nil, fmt.Errorf("create book: %w", err)
	}
	return &b, nil
}

// UpdateBook updates an existing book. Returns sql.ErrNoRows if not found,
// ErrDuplicateBook on uniqueness violation.
func (db *DB) UpdateBook(ctx context.Context, id int64, req *UpdateRequest) (*Book, error) {
	existing, err := db.GetBook(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Author != nil {
		existing.Author = *req.Author
	}
	if req.Description != nil {
		existing.Description = sql.NullString{String: *req.Description, Valid: true}
	}
	if req.Published != nil {
		existing.Published = *req.Published
	}

	var b Book
	err = db.pool.QueryRowContext(ctx,
		`UPDATE books SET title=$1, author=$2, description=$3, published=$4, updated_at=now()
		 WHERE id=$5
		 RETURNING id, title, author, description, published, created_at, updated_at`,
		existing.Title, existing.Author, existing.Description, existing.Published, id).
		Scan(&b.ID, &b.Title, &b.Author, &b.Description, &b.Published, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if isDuplicateError(err) {
			return nil, ErrDuplicateBook
		}
		return nil, fmt.Errorf("update book: %w", err)
	}
	return &b, nil
}

// DeleteBook removes a book by ID and returns the deleted book.
// Returns sql.ErrNoRows if not found.
func (db *DB) DeleteBook(ctx context.Context, id int64) (*Book, error) {
	var b Book
	err := db.pool.QueryRowContext(ctx,
		`DELETE FROM books WHERE id=$1
		 RETURNING id, title, author, description, published, created_at, updated_at`, id).
		Scan(&b.ID, &b.Title, &b.Author, &b.Description, &b.Published, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// isDuplicateError checks if the error is a PostgreSQL unique constraint violation.
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "unique constraint") ||
		strings.Contains(err.Error(), "23505")
}