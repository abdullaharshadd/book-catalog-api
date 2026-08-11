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

// This file is the Go analog of the source project's app/main.py, a FastAPI
// application exposing a CRUD REST API over a book catalog.
//
// MIGRATION_NOTE: FastAPI's Depends() dependency-injection of request-scoped
// AsyncSession / Session objects has no idiomatic Go equivalent. Instead, a
// single *sqlx.DB (see internal/database.go) is captured by the BookServer and
// shared across handlers. The async/sync split in the source is irrelevant in
// Go: every query runs synchronously against the shared pool.
//
// MIGRATION_NOTE: FastAPI's custom HTTPException handler + Pydantic
// response_model serialization are replaced by explicit JSON encoding helpers
// (writeJSON / writeError) that emit {"detail": "..."} error bodies, matching
// the source's http_exception_handler shape.

// BookServer holds the dependencies (database handle) needed by the HTTP
// handlers migrated from app/main.py.
type BookServer struct {
	db *sqlx.DB
}

// NewBookServer constructs a BookServer over the given database handle.
func NewBookServer(db *sqlx.DB) *BookServer {
	return &BookServer{db: db}
}

// errorBody mirrors FastAPI's {"detail": "..."} error response shape.
type errorBody struct {
	Detail string `json:"detail"`
}

// writeJSON encodes v as JSON with the given status code.
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

// writeError emits a {"detail": msg} body with the given status code, matching
// the source's custom HTTPException handler.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorBody{Detail: msg})
}

// isUniqueViolation reports whether err is a PostgreSQL unique-constraint
// violation (SQLSTATE 23505), the Go analog of SQLAlchemy's IntegrityError
// raised on a duplicate (title, author).
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}

// root is GET / — returns welcome message and API information.
func (s *BookServer) root(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"message":  "Welcome to Book Catalog API",
		"version":  "1.0.0",
		"docs_url": "/docs",
	})
}

// healthCheck is GET /health — health check endpoint.
func (s *BookServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "book-catalog-api",
	})
}

// parsePaginationParam parses an optional integer query param. An empty string
// yields the provided default; a non-integer value yields ok=false so the
// caller can reply 422 (matching FastAPI's query-param validation).
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

// listBooks is GET /books/ — lists all books with pagination (skip/limit).
//
// MIGRATION_NOTE: the source clamps limit to 1000 INLINE (limit = min(limit,
// 1000)) and returns 200 — it does NOT reject an over-large limit with 422.
// Only a non-integer (or otherwise unparseable) skip/limit triggers 422.
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

	// Enforce reasonable limits (inline clamp, HTTP 200).
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

	// nil/empty slice must serialize as [] not null.
	resp := make([]BookResponse, 0, len(books))
	for i := range books {
		resp = append(resp, NewBookResponse(books[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

// parseBookID extracts and validates the {book_id} path parameter.
func parseBookID(r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "book_id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

// getBook is GET /books/{book_id} — retrieves a single book by ID, 404 if not
// found.
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

// createBook is POST /books/ — creates a new book, 201 on success, 400 on
// duplicate (title, author).
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
	book.Summary = in.Summary

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

// updateBook is PUT /books/{book_id} — partially updates an existing book
// (only provided fields), 404 if not found, 400 on duplicate (title, author).
//
// MIGRATION_NOTE: Pydantic's exclude_unset partial-update semantics are
// reproduced by treating BookUpdate's fields as pointers: a nil pointer means
// "not provided" and is left untouched; a non-nil pointer overwrites the
// corresponding column.
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

	// Apply only provided fields.
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
		book.Summary = in.Summary
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

// deleteBook is DELETE /books/{book_id} — deletes a book by ID, 204 on
// success, 404 if not found.
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

// serverDB is the package-level database handle wired into BuildRouter. It is
// initialized by InitServer (called from cmd/server/main.go after opening the
// pool) so BuildRouter can keep the exact signature required by the entry
// point.
//
// MIGRATION_NOTE: BuildRouter's required signature is func() http.Handler with
// no arguments, so the *sqlx.DB is threaded in via this package variable rather
// than a parameter. cmd/server/main.go must call InitServer(db) before
// BuildRouter().
var serverDB *sqlx.DB

// InitServer records the database handle used by the HTTP handlers. Call this
// once at startup before BuildRouter().
func InitServer(db *sqlx.DB) {
	serverDB = db
}

// BuildRouter wires all routes migrated from app/main.py onto a chi router and
// returns it as an http.Handler. cmd/server/main.go calls this directly.
func BuildRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	srv := NewBookServer(serverDB)

	r.Get("/", srv.root)
	r.Get("/health", srv.healthCheck)

	// GET /healthz → returns "ok" (infra liveness probe).
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	// Book catalog routes. chi treats "/books" and "/books/" distinctly; the
	// source registers the trailing-slash forms, so we mirror them exactly and
	// also accept the non-slash collection variants for convenience.
	r.Get("/books/", srv.listBooks)
	r.Get("/books", srv.listBooks)
	r.Post("/books/", srv.createBook)
	r.Post("/books", srv.createBook)
	r.Get("/books/{book_id}", srv.getBook)
	r.Put("/books/{book_id}", srv.updateBook)
	r.Delete("/books/{book_id}", srv.deleteBook)

	return r
}

// buildRouter is kept as an unexported alias for internal use.
func buildRouter() http.Handler {
	return BuildRouter()
}

// ensure strings is referenced (used by potential future validation helpers);
// kept to avoid accidental unused-import churn if handlers evolve.
var _ = strings.TrimSpace