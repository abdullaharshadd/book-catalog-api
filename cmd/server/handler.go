package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

type Book struct {
	ID            int       `db:"id" json:"id"`
	Title         string    `db:"title" json:"title"`
	Author        string    `db:"author" json:"author"`
	PublishedYear int       `db:"published_year" json:"published_year"`
	Summary       *string   `db:"summary" json:"summary,omitempty"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

type BookCreate struct {
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	PublishedYear int     `json:"published_year"`
	Summary       *string `json:"summary,omitempty"`
}

type BookUpdate struct {
	Title         *string `json:"title,omitempty"`
	Author        *string `json:"author,omitempty"`
	PublishedYear *int    `json:"published_year,omitempty"`
	Summary       *string `json:"summary,omitempty"`
}

type Handler struct {
	db *sqlx.DB
}

func NewHandler(db *sqlx.DB) *Handler {
	return &Handler{db: db}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}

func (h *Handler) Root(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"message":  "Welcome to Book Catalog API",
		"version":  "1.0.0",
		"docs_url": "/docs",
	})
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "book-catalog-api",
	})
}

func (h *Handler) ListBooks(w http.ResponseWriter, r *http.Request) {
	skip := 0
	limit := 100

	if s := r.URL.Query().Get("skip"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			skip = v
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	if limit > 1000 {
		limit = 1000
	}

	var books []Book
	err := h.db.SelectContext(r.Context(), &books,
		`SELECT id, title, author, published_year, summary, created_at, updated_at FROM books ORDER BY id OFFSET $1 LIMIT $2`,
		skip, limit,
	)
	if err != nil {
		log.Printf("error retrieving books: %v", err)
		writeError(w, http.StatusInternalServerError, "Internal server error while retrieving books")
		return
	}
	if books == nil {
		books = []Book{}
	}
	log.Printf("Retrieved %d books (skip=%d, limit=%d)", len(books), skip, limit)
	writeJSON(w, http.StatusOK, books)
}

func (h *Handler) GetBook(w http.ResponseWriter, r *http.Request) {
	bookIDStr := chi.URLParam(r, "book_id")
	bookID, err := strconv.Atoi(bookIDStr)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid book ID")
		return
	}

	var book Book
	err = h.db.GetContext(r.Context(), &book,
		`SELECT id, title, author, published_year, summary, created_at, updated_at FROM books WHERE id = $1`,
		bookID,
	)
	if err == sql.ErrNoRows {
		log.Printf("Book with ID %d not found", bookID)
		writeError(w, http.StatusNotFound, "Book with ID "+bookIDStr+" not found")
		return
	}
	if err != nil {
		log.Printf("error retrieving book %d: %v", bookID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while retrieving book")
		return
	}

	log.Printf("Retrieved book: %s", book.Title)
	writeJSON(w, http.StatusOK, book)
}

func (h *Handler) CreateBook(w http.ResponseWriter, r *http.Request) {
	var input BookCreate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid request body")
		return
	}
	if input.Title == "" || input.Author == "" || input.PublishedYear == 0 {
		writeError(w, http.StatusUnprocessableEntity, "title, author, and published_year are required")
		return
	}

	var book Book
	err := h.db.QueryRowxContext(r.Context(),
		`INSERT INTO books (title, author, published_year, summary) VALUES ($1, $2, $3, $4)
		 RETURNING id, title, author, published_year, summary, created_at, updated_at`,
		input.Title, input.Author, input.PublishedYear, input.Summary,
	).StructScan(&book)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			writeError(w, http.StatusBadRequest, "Book with this title and author already exists")
			return
		}
		log.Printf("error creating book: %v", err)
		writeError(w, http.StatusInternalServerError, "Internal server error while creating book")
		return
	}

	log.Printf("Created new book: %s by %s", book.Title, book.Author)
	writeJSON(w, http.StatusCreated, book)
}

func (h *Handler) UpdateBook(w http.ResponseWriter, r *http.Request) {
	bookIDStr := chi.URLParam(r, "book_id")
	bookID, err := strconv.Atoi(bookIDStr)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid book ID")
		return
	}

	var input BookUpdate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid request body")
		return
	}

	var existing Book
	err = h.db.GetContext(r.Context(), &existing,
		`SELECT id, title, author, published_year, summary, created_at, updated_at FROM books WHERE id = $1`,
		bookID,
	)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Book with ID "+bookIDStr+" not found")
		return
	}
	if err != nil {
		log.Printf("error retrieving book %d for update: %v", bookID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while updating book")
		return
	}

	if input.Title != nil {
		existing.Title = *input.Title
	}
	if input.Author != nil {
		existing.Author = *input.Author
	}
	if input.PublishedYear != nil {
		existing.PublishedYear = *input.PublishedYear
	}
	if input.Summary != nil {
		existing.Summary = input.Summary
	}

	var updated Book
	err = h.db.QueryRowxContext(r.Context(),
		`UPDATE books SET title=$1, author=$2, published_year=$3, summary=$4, updated_at=NOW()
		 WHERE id=$5
		 RETURNING id, title, author, published_year, summary, created_at, updated_at`,
		existing.Title, existing.Author, existing.PublishedYear, existing.Summary, bookID,
	).StructScan(&updated)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			writeError(w, http.StatusBadRequest, "Book with this title and author already exists")
			return
		}
		log.Printf("error updating book %d: %v", bookID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while updating book")
		return
	}

	log.Printf("Updated book: %s", updated.Title)
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) DeleteBook(w http.ResponseWriter, r *http.Request) {
	bookIDStr := chi.URLParam(r, "book_id")
	bookID, err := strconv.Atoi(bookIDStr)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid book ID")
		return
	}

	var existing Book
	err = h.db.GetContext(r.Context(), &existing,
		`SELECT id FROM books WHERE id = $1`,
		bookID,
	)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Book with ID "+bookIDStr+" not found")
		return
	}
	if err != nil {
		log.Printf("error retrieving book %d for deletion: %v", bookID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while deleting book")
		return
	}

	_, err = h.db.ExecContext(r.Context(), `DELETE FROM books WHERE id = $1`, bookID)
	if err != nil {
		log.Printf("error deleting book %d: %v", bookID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while deleting book")
		return
	}

	log.Printf("Deleted book with ID %d", bookID)
	w.WriteHeader(http.StatusNoContent)
}