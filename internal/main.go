package internal

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type BookServer struct {
	db *sqlx.DB
}

func NewBookServer(db *sqlx.DB) *BookServer {
	return &BookServer{db: db}
}

type errorBody struct {
	Detail string `json:"detail"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("error encoding JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorBody{Detail: msg})
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}

func (s *BookServer) root(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"message":  "Welcome to Book Catalog API",
		"version":  "1.0.0",
		"docs_url": "/docs",
	})
}

func (s *BookServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "book-catalog-api",
	})
}

func parsePaginationParam(raw string, def int) (int, bool) {
	if raw == "" {
		return def, true
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return v, true
}

func (s *BookServer) listBooks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	skip, ok := parsePaginationParam(q.Get("skip"), 0)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "skip must be an integer")
		return
	}
	limit, ok := parsePaginationParam(q.Get("limit"), 100)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "limit must be an integer")
		return
	}

	if limit > 1000 {
		limit = 1000
	}

	books := []Book{}
	err := s.db.SelectContext(
		r.Context(),
		&books,
		`SELECT id, title, author, published_year, summary FROM books ORDER BY id OFFSET $1 LIMIT $2`,
		skip, limit,
	)
	if err != nil {
		log.Printf("Error retrieving books: %v", err)
		writeError(w, http.StatusInternalServerError, "Internal server error while retrieving books")
		return
	}

	log.Printf("Retrieved %d books (skip=%d, limit=%d)", len(books), skip, limit)

	resp := make([]BookResponse, 0, len(books))
	for i := range books {
		resp = append(resp, NewBookResponse(books[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

func parseBookID(r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "book_id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func (s *BookServer) getBook(w http.ResponseWriter, r *http.Request) {
	bookID, ok := parseBookID(r)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "book_id must be an integer")
		return
	}

	var book Book
	err := s.db.GetContext(
		r.Context(),
		&book,
		`SELECT id, title, author, published_year, summary FROM books WHERE id = $1`,
		bookID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("Book with ID %d not found", bookID)
		writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %d not found", bookID))
		return
	}
	if err != nil {
		log.Printf("Error retrieving book %d: %v", bookID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while retrieving book")
		return
	}

	log.Printf("Retrieved book: %s", book.Title)
	writeJSON(w, http.StatusOK, NewBookResponse(book))
}

func (s *BookServer) createBook(w http.ResponseWriter, r *http.Request) {
	var in BookCreate
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if err := in.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	var book Book
	book.Title = in.Title
	book.Author = in.Author
	book.PublishedYear = in.PublishedYear
	if in.Summary != nil {
		book.Summary = *in.Summary
	}

	err := s.db.QueryRowxContext(
		r.Context(),
		`INSERT INTO books (title, author, published_year, summary)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		book.Title, book.Author, book.PublishedYear, book.Summary,
	).Scan(&book.ID)
	if isUniqueViolation(err) {
		log.Printf("Integrity error creating book: %v", err)
		writeError(w, http.StatusBadRequest, "Book with this title and author already exists")
		return
	}
	if err != nil {
		log.Printf("Error creating book: %v", err)
		writeError(w, http.StatusInternalServerError, "Internal server error while creating book")
		return
	}

	log.Printf("Created new book: %s by %s", book.Title, book.Author)
	writeJSON(w, http.StatusCreated, NewBookResponse(book))
}

func (s *BookServer) updateBook(w http.ResponseWriter, r *http.Request) {
	bookID, ok := parseBookID(r)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "book_id must be an integer")
		return
	}

	var in BookUpdate
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if err := in.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	var book Book
	err := s.db.GetContext(
		r.Context(),
		&book,
		`SELECT id, title, author, published_year, summary FROM books WHERE id = $1`,
		bookID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("Book with ID %d not found for update", bookID)
		writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %d not found", bookID))
		return
	}
	if err != nil {
		log.Printf("Error updating book %d: %v", bookID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while updating book")
		return
	}

	if in.Title != nil {
		book.Title = *in.Title
	}
	if in.Author != nil {
		book.Author = *in.Author
	}
	if in.PublishedYear != nil {
		book.PublishedYear = *in.PublishedYear
	}
	if in.Summary != nil {
		book.Summary = *in.Summary
	}

	_, err = s.db.ExecContext(
		r.Context(),
		`UPDATE books SET title = $1, author = $2, published_year = $3, summary = $4 WHERE id = $5`,
		book.Title, book.Author, book.PublishedYear, book.Summary, book.ID,
	)
	if isUniqueViolation(err) {
		log.Printf("Integrity error updating book: %v", err)
		writeError(w, http.StatusBadRequest, "Book with this title and author already exists")
		return
	}
	if err != nil {
		log.Printf("Error updating book %d: %v", bookID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while updating book")
		return
	}

	log.Printf("Updated book: %s", book.Title)
	writeJSON(w, http.StatusOK, NewBookResponse(book))
}

func (s *BookServer) deleteBook(w http.ResponseWriter, r *http.Request) {
	bookID, ok := parseBookID(r)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "book_id must be an integer")
		return
	}

	var title string
	err := s.db.GetContext(
		r.Context(),
		&title,
		`SELECT title FROM books WHERE id = $1`,
		bookID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("Book with ID %d not found for deletion", bookID)
		writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %d not found", bookID))
		return
	}
	if err != nil {
		log.Printf("Error deleting book %d: %v", bookID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while deleting book")
		return
	}

	if _, err := s.db.ExecContext(r.Context(), `DELETE FROM books WHERE id = $1`, bookID); err != nil {
		log.Printf("Error deleting book %d: %v", bookID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while deleting book")
		return
	}

	log.Printf("Deleted book: %s", title)
	w.WriteHeader(http.StatusNoContent)
}

var serverDB *sqlx.DB

func InitServer(db *sqlx.DB) {
	serverDB = db
}

func BuildRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	srv := NewBookServer(serverDB)

	r.Get("/", srv.root)
	r.Get("/health", srv.healthCheck)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	r.Get("/books/", srv.listBooks)
	r.Get("/books", srv.listBooks)
	r.Post("/books/", srv.createBook)
	r.Post("/books", srv.createBook)
	r.Get("/books/{book_id}", srv.getBook)
	r.Put("/books/{book_id}", srv.updateBook)
	r.Delete("/books/{book_id}", srv.deleteBook)

	return r
}

func buildRouter() http.Handler {
	return BuildRouter()
}

var _ = strings.TrimSpace