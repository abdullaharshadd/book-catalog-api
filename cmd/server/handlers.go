package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

type Book struct {
	ID            int    `db:"id" json:"id"`
	Title         string `db:"title" json:"title"`
	Author        string `db:"author" json:"author"`
	PublishedYear int    `db:"published_year" json:"published_year"`
	Summary       string `db:"summary" json:"summary,omitempty"`
}

type BookHandler struct {
	db *sqlx.DB
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"detail": msg})
}

func (h *BookHandler) ListBooks(w http.ResponseWriter, r *http.Request) {
	skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	var books []Book
	err := h.db.Select(&books, "SELECT id, title, author, published_year, COALESCE(summary, '') as summary FROM books ORDER BY id LIMIT $1 OFFSET $2", limit, skip)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if books == nil {
		books = []Book{}
	}
	writeJSON(w, http.StatusOK, books)
}

func (h *BookHandler) GetBook(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "bookID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid book ID")
		return
	}
	var book Book
	err = h.db.Get(&book, "SELECT id, title, author, published_year, COALESCE(summary, '') as summary FROM books WHERE id = $1", id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Book not found")
		return
	}
	writeJSON(w, http.StatusOK, book)
}

func (h *BookHandler) CreateBook(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title         string `json:"title"`
		Author        string `json:"author"`
		PublishedYear int    `json:"published_year"`
		Summary       string `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if input.Title == "" || input.Author == "" {
		writeError(w, http.StatusBadRequest, "title and author are required")
		return
	}
	var book Book
	err := h.db.QueryRowx(
		"INSERT INTO books (title, author, published_year, summary) VALUES ($1, $2, $3, $4) RETURNING id, title, author, published_year, COALESCE(summary, '') as summary",
		input.Title, input.Author, input.PublishedYear, input.Summary,
	).StructScan(&book)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, book)
}

func (h *BookHandler) UpdateBook(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "bookID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid book ID")
		return
	}
	var existing Book
	err = h.db.Get(&existing, "SELECT id, title, author, published_year, COALESCE(summary, '') as summary FROM books WHERE id = $1", id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Book not found")
		return
	}
	var input struct {
		Title         *string `json:"title"`
		Author        *string `json:"author"`
		PublishedYear *int    `json:"published_year"`
		Summary       *string `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
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
		existing.Summary = *input.Summary
	}
	var updated Book
	err = h.db.QueryRowx(
		"UPDATE books SET title=$1, author=$2, published_year=$3, summary=$4 WHERE id=$5 RETURNING id, title, author, published_year, COALESCE(summary, '') as summary",
		existing.Title, existing.Author, existing.PublishedYear, existing.Summary, id,
	).StructScan(&updated)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *BookHandler) DeleteBook(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "bookID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid book ID")
		return
	}
	result, err := h.db.Exec("DELETE FROM books WHERE id = $1", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeError(w, http.StatusNotFound, "Book not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}