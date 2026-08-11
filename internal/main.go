package internal

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// MIGRATION_NOTE: The Python source was a FastAPI application (app/main.py) that
// mixed async and sync SQLAlchemy sessions. As documented in database.go, the
// dual async/sync split does not carry over to Go: database/sql + sqlx already
// manages a connection pool and runs queries synchronously per goroutine. All
// handlers below therefore use a single *sqlx.DB.
//
// MIGRATION_NOTE: FastAPI's Depends() dependency injection is replaced by an
// explicit BookHandler struct constructed with NewBookHandler(db). Routes are
// registered explicitly on a chi router in buildRouter().
//
// MIGRATION_NOTE: FastAPI's automatic Pydantic request/response serialization
// and the custom HTTPException handler are replaced by explicit JSON decode,
// Validate(), and helper writers (writeJSON / writeError) below.

// BookHandler holds the dependencies required to serve the book-catalog HTTP
// endpoints. It is the Go equivalent of the FastAPI route functions plus their
// Depends()-injected database sessions.
type BookHandler struct {
	db *sqlx.DB
}

// NewBookHandler constructs a BookHandler backed by the given database handle.
func NewBookHandler(db *sqlx.DB) *BookHandler {
	return &BookHandler{db: db}
}

// errorBody is the JSON error envelope, mirroring FastAPI's {"detail": ...}.
type errorBody struct {
	Detail interface{} `json:"detail"`
}

// writeJSON serializes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("failed to encode JSON response")
	}
}

// writeError writes a FastAPI-style {"detail": ...} error body.
func writeError(w http.ResponseWriter, status int, detail interface{}) {
	writeJSON(w, status, errorBody{Detail: detail})
}

// Root handles GET / and returns API metadata and a welcome message.
//
// MIGRATION_NOTE: FastAPI auto-generated /docs and /redoc. There is no direct
// Go equivalent bundled here; docs_url is reported as "/docs" for parity but is
// not served by this application.
func (h *BookHandler) Root(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"message":  "Welcome to Book Catalog API",
		"version":  "1.0.0",
		"docs_url": "/docs",
	})
}

// HealthCheck handles GET /health and reports service health.
func (h *BookHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "book-catalog-api",
	})
}

// ListBooks handles GET /books/ and lists books with skip/limit pagination.
// The limit is capped at 1000, matching the source.
func (h *BookHandler) ListBooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	skip := parseIntDefault(r.URL.Query().Get("skip"), 0)
	limit := parseIntDefault(r.URL.Query().Get("limit"), 100)
	if limit > 1000 {
		limit = 1000
	}
	if skip < 0 {
		skip = 0
	}
	if limit < 0 {
		limit = 0
	}

	const query = `SELECT id, title, author, published_year, summary, created_at, updated_at
		FROM books ORDER BY id OFFSET $1 LIMIT $2`

	var books []Book
	if err := h.db.SelectContext(ctx, &books, query, skip, limit); err != nil {
		log.Error().Err(err).Msg("error retrieving books")
		writeError(w, http.StatusInternalServerError, "Internal server error while retrieving books")
		return
	}

	log.Info().Int("count", len(books)).Int("skip", skip).Int("limit", limit).Msg("retrieved books")

	responses := make([]BookResponse, 0, len(books))
	for i := range books {
		responses = append(responses, NewBookResponse(&books[i]))
	}
	writeJSON(w, http.StatusOK, responses)
}

// GetBook handles GET /books/{book_id} and returns a single book, or 404.
func (h *BookHandler) GetBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bookID, ok := parsePathID(w, r)
	if !ok {
		return
	}

	book, err := h.fetchBook(ctx, h.db, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn().Int64("book_id", bookID).Msg("book not found")
			writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %d not found", bookID))
			return
		}
		log.Error().Err(err).Int64("book_id", bookID).Msg("error retrieving book")
		writeError(w, http.StatusInternalServerError, "Internal server error while retrieving book")
		return
	}

	log.Info().Str("title", book.Title).Msg("retrieved book")
	writeJSON(w, http.StatusOK, NewBookResponse(book))
}

// CreateBook handles POST /books/ and creates a new book, returning 201 on
// success or 400 on a duplicate title/author.
//
// MIGRATION_NOTE: The handler enforces a strict decode -> Validate() -> toModel
// -> insert ordering. toModel is an explicit, unit-testable function symmetric
// with NewBookResponse (toResponse) rather than SQLAlchemy's Book(**dict).
func (h *BookHandler) CreateBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var in BookCreate
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if err := in.Validate(); err != nil {
		writeValidationError(w, err)
		return
	}

	book := bookFromCreate(&in)

	const query = `INSERT INTO books (title, author, published_year, summary)
		VALUES ($1, $2, $3, $4)
		RETURNING id, title, author, published_year, summary, created_at, updated_at`

	err := h.db.QueryRowxContext(ctx, query,
		book.Title, book.Author, book.PublishedYear, book.Summary,
	).StructScan(book)
	if err != nil {
		if IsUniqueViolation(err) {
			log.Error().Err(err).Msg("integrity error creating book")
			writeError(w, http.StatusBadRequest, "Book with this title and author already exists")
			return
		}
		log.Error().Err(err).Msg("error creating book")
		writeError(w, http.StatusInternalServerError, "Internal server error while creating book")
		return
	}

	log.Info().Str("title", book.Title).Str("author", book.Author).Msg("created new book")
	writeJSON(w, http.StatusCreated, NewBookResponse(book))
}

// UpdateBook handles PUT /books/{book_id} and applies a partial update,
// returning 404 if the book does not exist and 400 on an integrity error.
//
// MIGRATION_NOTE: Pydantic's exclude_unset partial update is replicated by only
// applying the pointer fields of BookUpdate that are non-nil.
func (h *BookHandler) UpdateBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bookID, ok := parsePathID(w, r)
	if !ok {
		return
	}

	var in BookUpdate
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if err := in.Validate(); err != nil {
		writeValidationError(w, err)
		return
	}

	// MIGRATION_NOTE: SQLAlchemy loaded the row, mutated the ORM object, and
	// committed. Here we do a fetch (for the 404 check) then a single UPDATE
	// that only touches the provided fields, followed by a re-fetch to return
	// the fresh row (equivalent to db.refresh()).
	book, err := h.fetchBook(ctx, h.db, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn().Int64("book_id", bookID).Msg("book not found for update")
			writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %d not found", bookID))
			return
		}
		log.Error().Err(err).Int64("book_id", bookID).Msg("error updating book")
		writeError(w, http.StatusInternalServerError, "Internal server error while updating book")
		return
	}

	applyBookUpdate(book, &in)

	const query = `UPDATE books
		SET title = $1, author = $2, published_year = $3, summary = $4, updated_at = now()
		WHERE id = $5
		RETURNING id, title, author, published_year, summary, created_at, updated_at`

	err = h.db.QueryRowxContext(ctx, query,
		book.Title, book.Author, book.PublishedYear, book.Summary, bookID,
	).StructScan(book)
	if err != nil {
		if IsUniqueViolation(err) {
			log.Error().Err(err).Msg("integrity error updating book")
			writeError(w, http.StatusBadRequest, "Book with this title and author already exists")
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %d not found", bookID))
			return
		}
		log.Error().Err(err).Int64("book_id", bookID).Msg("error updating book")
		writeError(w, http.StatusInternalServerError, "Internal server error while updating book")
		return
	}

	log.Info().Str("title", book.Title).Msg("updated book")
	writeJSON(w, http.StatusOK, NewBookResponse(book))
}

// DeleteBook handles DELETE /books/{book_id} and removes a book, returning 204
// on success or 404 if the book does not exist.
func (h *BookHandler) DeleteBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bookID, ok := parsePathID(w, r)
	if !ok {
		return
	}

	const query = `DELETE FROM books WHERE id = $1`
	res, err := h.db.ExecContext(ctx, query, bookID)
	if err != nil {
		log.Error().Err(err).Int64("book_id", bookID).Msg("error deleting book")
		writeError(w, http.StatusInternalServerError, "Internal server error while deleting book")
		return
	}

	affected, err := res.RowsAffected()
	if err != nil {
		log.Error().Err(err).Int64("book_id", bookID).Msg("error deleting book")
		writeError(w, http.StatusInternalServerError, "Internal server error while deleting book")
		return
	}
	if affected == 0 {
		log.Warn().Int64("book_id", bookID).Msg("book not found for deletion")
		writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %d not found", bookID))
		return
	}

	log.Info().Int64("book_id", bookID).Msg("deleted book")
	w.WriteHeader(http.StatusNoContent)
}

// fetchBook loads a single book by ID, returning sql.ErrNoRows if absent.
func (h *BookHandler) fetchBook(ctx context, q *sqlx.DB, id int64) (*Book, error) {
	const query = `SELECT id, title, author, published_year, summary, created_at, updated_at
		FROM books WHERE id = $1`
	var book Book
	if err := q.GetContext(ctx, &book, query, id); err != nil {
		return nil, err
	}
	return &book, nil
}

// bookFromCreate is the explicit toModel translation of a validated BookCreate
// into a Book, symmetric with NewBookResponse (toResponse).
func bookFromCreate(in *BookCreate) *Book {
	b := &Book{
		Title:         in.Title,
		Author:        in.Author,
		PublishedYear: in.PublishedYear,
	}
	if in.Summary != nil {
		b.Summary = sql.NullString{String: *in.Summary, Valid: true}
	}
	return b
}

// applyBookUpdate applies only the set (non-nil) fields of a BookUpdate onto an
// existing Book, replicating Pydantic's exclude_unset semantics.
func applyBookUpdate(b *Book, in *BookUpdate) {
	if in.Title != nil {
		b.Title = *in.Title
	}
	if in.Author != nil {
		b.Author = *in.Author
	}
	if in.PublishedYear != nil {
		b.PublishedYear = *in.PublishedYear
	}
	if in.Summary != nil {
		b.Summary = sql.NullString{String: *in.Summary, Valid: true}
	}
}

// writeValidationError renders a *ValidationError as a 422 response, falling
// back to a generic 400 for any other error type.
func writeValidationError(w http.ResponseWriter, err error) {
	var ve *ValidationError
	if errors.As(err, &ve) {
		writeError(w, http.StatusUnprocessableEntity, ve.Errors)
		return
	}
	writeError(w, http.StatusBadRequest, err.Error())
}

// parsePathID extracts and validates the {book_id} URL parameter.
func parsePathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "book_id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid book ID")
		return 0, false
	}
	return id, true
}

// parseIntDefault parses s as an int, returning def when s is empty or invalid.
func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// buildRouter wires all HTTP routes for the Book Catalog API onto a chi router
// and returns it as an http.Handler. cmd/server/main.go calls this directly.
//
// MIGRATION_NOTE: The FastAPI @app.on_event("startup") hook called init_db().
// Startup/schema initialization (InitDB) is owned by the application entry
// point (cmd/server/main.go) which constructs the *sqlx.DB via NewDB before
// calling buildRouter, rather than being a router concern.
func buildRouter(db *sqlx.DB) http.Handler {
	h := NewBookHandler(db)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/", h.Root)
	r.Get("/health", h.HealthCheck)

	r.Get("/books/", h.ListBooks)
	r.Post("/books/", h.CreateBook)
	r.Get("/books/{book_id}", h.GetBook)
	r.Put("/books/{book_id}", h.UpdateBook)
	r.Delete("/books/{book_id}", h.DeleteBook)

	return r
}
