package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lib/pq"
)

// MIGRATION_NOTE: The Python source was a FastAPI application with a mix of
// async (AsyncSession) and sync (Session) SQLAlchemy handlers. Go's
// database/sql pool is concurrency-safe and context-driven, so the
// async/sync split collapses into a single *DB (see internal/database.go).
// FastAPI's Depends-based dependency injection is replaced by an explicit
// Handlers struct holding the *DB dependency, wired in buildRouter.
//
// The custom FastAPI exception handler is replaced by writeError, which emits
// the same {"detail": "..."} JSON body the source produced.

// Handlers holds the dependencies shared by all HTTP handlers.
//
// MIGRATION_NOTE: This replaces FastAPI's Depends(get_db)/Depends(get_sync_db)
// injection — the database handle is stored once and reused by every route.
type Handlers struct {
	db *DB
}

// NewHandlers constructs a Handlers backed by the given database.
func NewHandlers(db *DB) *Handlers {
	return &Handlers{db: db}
}

// errorResponse mirrors FastAPI's {"detail": "..."} error body.
type errorResponse struct {
	Detail string `json:"detail"`
}

// writeError writes a JSON error body with the given status code, matching the
// shape produced by the source's custom HTTPException handler.
func writeError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, errorResponse{Detail: detail})
}

// writeJSON serializes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("error encoding response: %v", err)
	}
}

// isUniqueViolation reports whether err is a PostgreSQL unique-constraint
// violation (SQLSTATE 23505). This replaces the source's catch of
// SQLAlchemy IntegrityError for the (title, author) unique constraint.
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}

// Root handles GET / — returns welcome message and API metadata.
func (h *Handlers) Root(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"message":  "Welcome to Book Catalog API",
		"version":  "1.0.0",
		"docs_url": "/docs",
	})
}

// HealthCheck handles GET /health — returns service status.
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "book-catalog-api",
	})
}

// ListBooks handles GET /books/ — lists books with pagination.
//
// skip defaults to 0, limit defaults to 100 and is capped at 1000, matching
// the source's behaviour.
func (h *Handlers) ListBooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	skip := parseIntDefault(r.URL.Query().Get("skip"), 0)
	if skip < 0 {
		skip = 0
	}
	limit := parseIntDefault(r.URL.Query().Get("limit"), 100)
	if limit > 1000 {
		limit = 1000
	}
	if limit < 0 {
		limit = 0
	}

	const query = `SELECT id, title, author, published_year, summary, created_at
		FROM books ORDER BY id OFFSET $1 LIMIT $2`

	var books []Book
	if err := h.db.SQL().SelectContext(ctx, &books, query, skip, limit); err != nil {
		log.Printf("Error retrieving books: %v", err)
		writeError(w, http.StatusInternalServerError, "Internal server error while retrieving books")
		return
	}

	resp := make([]BookResponse, 0, len(books))
	for i := range books {
		resp = append(resp, NewBookResponse(&books[i]))
	}

	log.Printf("Retrieved %d books (skip=%d, limit=%d)", len(books), skip, limit)
	writeJSON(w, http.StatusOK, resp)
}

// GetBook handles GET /books/{book_id} — retrieves a single book by ID.
func (h *Handlers) GetBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bookID, ok := parseBookID(w, r)
	if !ok {
		return
	}

	book, err := h.findBook(ctx, bookID)
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

// CreateBook handles POST /books/ — creates a new book.
//
// Returns 201 on success and 400 when the (title, author) pair already
// exists, matching the source's IntegrityError handling.
func (h *Handlers) CreateBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var in BookCreate
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := in.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	const query = `INSERT INTO books (title, author, published_year, summary)
		VALUES ($1, $2, $3, $4)
		RETURNING id, title, author, published_year, summary, created_at`

	var book Book
	err := h.db.SQL().QueryRowxContext(ctx, query,
		in.Title, in.Author, in.PublishedYear, in.Summary,
	).StructScan(&book)
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
	writeJSON(w, http.StatusCreated, NewBookResponse(&book))
}

// UpdateBook handles PUT /books/{book_id} — partially updates a book.
//
// MIGRATION_NOTE: The source used Pydantic's exclude_unset to apply only the
// fields present in the request body (the omitted/null/value three-way
// distinction). BookUpdate in internal/schemas.go carries pointer/presence
// fields for this; here we build a partial UPDATE from whichever fields were
// actually supplied. A field omitted from the JSON is left unchanged; a field
// present (including explicit null where the column is nullable) is applied.
func (h *Handlers) UpdateBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bookID, ok := parseBookID(w, r)
	if !ok {
		return
	}

	var in BookUpdate
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := in.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Ensure the book exists first (404 semantics from the source).
	if _, err := h.findBook(ctx, bookID); errors.Is(err, sql.ErrNoRows) {
		log.Printf("Book with ID %d not found for update", bookID)
		writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %d not found", bookID))
		return
	} else if err != nil {
		log.Printf("Error updating book %d: %v", bookID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while updating book")
		return
	}

	// Build the SET clause dynamically from the present fields. Each of the
	// BookUpdate pointer fields is nil when the field was omitted from the
	// request body and non-nil (possibly pointing at a null-equivalent) when
	// it was supplied.
	setClauses := make([]string, 0, 4)
	args := make([]interface{}, 0, 5)
	argIdx := 1

	if in.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *in.Title)
		argIdx++
	}
	if in.Author != nil {
		setClauses = append(setClauses, fmt.Sprintf("author = $%d", argIdx))
		args = append(args, *in.Author)
		argIdx++
	}
	if in.PublishedYear != nil {
		setClauses = append(setClauses, fmt.Sprintf("published_year = $%d", argIdx))
		args = append(args, *in.PublishedYear)
		argIdx++
	}
	if in.Summary != nil {
		setClauses = append(setClauses, fmt.Sprintf("summary = $%d", argIdx))
		// Summary is nullable: *in.Summary may itself represent an explicit
		// null depending on the schema's field type; it is passed straight
		// through so the source's set-to-null semantics are preserved.
		args = append(args, *in.Summary)
		argIdx++
	}

	// If nothing was supplied, re-fetch and return the unchanged book — this
	// matches the source, where an empty exclude_unset dict is a no-op update.
	if len(setClauses) == 0 {
		book, err := h.findBook(ctx, bookID)
		if err != nil {
			log.Printf("Error updating book %d: %v", bookID, err)
			writeError(w, http.StatusInternalServerError, "Internal server error while updating book")
			return
		}
		writeJSON(w, http.StatusOK, NewBookResponse(book))
		return
	}

	args = append(args, bookID)
	query := fmt.Sprintf(`UPDATE books SET %s WHERE id = $%d
		RETURNING id, title, author, published_year, summary, created_at`,
		join(setClauses, ", "), argIdx)

	var book Book
	err := h.db.SQL().QueryRowxContext(ctx, query, args...).StructScan(&book)
	if errors.Is(err, sql.ErrNoRows) {
		// Lost a race — the book was deleted between the existence check and
		// the update. Report as not found to stay consistent with the source.
		log.Printf("Book with ID %d not found for update", bookID)
		writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %d not found", bookID))
		return
	}
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
	writeJSON(w, http.StatusOK, NewBookResponse(&book))
}

// DeleteBook handles DELETE /books/{book_id} — deletes a book by ID.
//
// Returns 204 on success and 404 when the book does not exist.
func (h *Handlers) DeleteBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bookID, ok := parseBookID(w, r)
	if !ok {
		return
	}

	const query = `DELETE FROM books WHERE id = $1`
	res, err := h.db.SQL().ExecContext(ctx, query, bookID)
	if err != nil {
		log.Printf("Error deleting book %d: %v", bookID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while deleting book")
		return
	}

	affected, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error deleting book %d: %v", bookID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while deleting book")
		return
	}
	if affected == 0 {
		log.Printf("Book with ID %d not found for deletion", bookID)
		writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %d not found", bookID))
		return
	}

	log.Printf("Deleted book with ID %d", bookID)
	w.WriteHeader(http.StatusNoContent)
}

// findBook loads a single book by ID, returning sql.ErrNoRows when absent.
func (h *Handlers) findBook(ctx context.Context, id int64) (*Book, error) {
	const query = `SELECT id, title, author, published_year, summary, created_at
		FROM books WHERE id = $1`
	var book Book
	if err := h.db.SQL().GetContext(ctx, &book, query, id); err != nil {
		return nil, err
	}
	return &book, nil
}

// parseBookID extracts and validates the {book_id} path parameter. It writes a
// 404 response and returns ok=false when the value is not a valid integer,
// matching the source's routing behaviour for non-numeric IDs.
func parseBookID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "book_id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %s not found", raw))
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

// join concatenates parts with sep. (Small local helper to avoid importing
// strings solely for a single Join call alongside the rest of this file.)
func join(parts []string, sep string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += sep + p
	}
	return out
}

// buildRouter constructs the fully-wired HTTP handler for the Book Catalog
// service. cmd/server/main.go calls this directly.
//
// MIGRATION_NOTE: The FastAPI startup event that called init_db() is not run
// here — schema initialization (InitSchema) belongs in the application
// bootstrap (cmd/server/main.go), which also owns NewDB. buildRouter takes the
// already-constructed *DB so router construction stays free of side effects.
func buildRouter(db *DB) http.Handler {
	h := NewHandlers(db)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
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
