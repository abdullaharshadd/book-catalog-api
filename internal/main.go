package internal

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// This file replaces the FastAPI application defined in app/main.py. It wires
// the Book Catalog CRUD REST API onto a chi router.
//
// MIGRATION_NOTE: FastAPI's Depends()-based dependency injection is replaced by
// a Handlers struct that holds the *DB dependency and exposes methods as
// http.HandlerFunc. The source used a mix of async (AsyncSession) and sync
// (Session) SQLAlchemy sessions; in Go there is no such distinction, so every
// handler uses the same *DB and the request context for cancellation.
//
// MIGRATION_NOTE: FastAPI auto-generated OpenAPI docs (/docs, /redoc) have no
// direct Go equivalent and are intentionally omitted. The root endpoint still
// advertises "/docs" in its payload to preserve the original response body.

// Handlers bundles the dependencies shared by every HTTP handler in the Book
// Catalog API. It replaces FastAPI's Depends(get_db)/Depends(get_sync_db)
// injection with explicit constructor-based wiring.
type Handlers struct {
	db *DB
}

// NewHandlers constructs a Handlers value bound to the given database.
func NewHandlers(db *DB) *Handlers {
	return &Handlers{db: db}
}

// writeJSON serialises v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			log.Printf("error encoding JSON response: %v", err)
		}
	}
}

// writeError mirrors FastAPI's custom HTTPException handler, which returns a
// JSON body of the form {"detail": "..."} alongside the status code.
func writeError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}

// buildRouter constructs the fully-wired HTTP handler for the Book Catalog API.
//
// It is invoked directly by cmd/server/main.go, so the name and signature must
// remain stable.
func buildRouter() http.Handler {
	db, err := NewDB()
	if err != nil {
		// MIGRATION_NOTE: the FastAPI startup event called init_db() and would
		// crash the process on failure; we mirror that fail-fast behaviour here.
		log.Fatalf("failed to initialize database: %v", err)
	}
	log.Println("Database initialized successfully")

	h := NewHandlers(db)
	return h.Router()
}

// Router builds and returns the chi router for these handlers. It is separated
// from buildRouter so the routes can be exercised in tests with an injected DB.
func (h *Handlers) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	r.Get("/", h.Root)
	r.Get("/health", h.HealthCheck)

	r.Route("/books", func(r chi.Router) {
		r.Get("/", h.ListBooks)
		r.Post("/", h.CreateBook)
		r.Get("/{book_id}", h.GetBook)
		r.Put("/{book_id}", h.UpdateBook)
		r.Delete("/{book_id}", h.DeleteBook)
	})

	return r
}

// Root handles GET / and returns basic API information.
func (h *Handlers) Root(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"message":  "Welcome to Book Catalog API",
		"version":  "1.0.0",
		"docs_url": "/docs",
	})
}

// HealthCheck handles GET /health.
func (h *Handlers) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "book-catalog-api",
	})
}

// parseIntQuery reads an integer query parameter, falling back to def when the
// parameter is absent or malformed.
//
// MIGRATION_NOTE: the source used bare int defaults (skip=0, limit=100) WITHOUT
// Query(..., ge=0) constraints, so FastAPI did NOT reject negative values with
// 422 — they were passed straight to SQLAlchemy. We preserve that permissive
// behaviour: negative/garbage values are handled the same way the source did
// (limit is later clamped to <=1000; skip is passed through as-is).
func parseIntQuery(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// ListBooks handles GET /books/ with skip/limit pagination.
func (h *Handlers) ListBooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	skip := parseIntQuery(r, "skip", 0)
	limit := parseIntQuery(r, "limit", 100)

	// Enforce reasonable limits (max 1000), matching the source.
	if limit > 1000 {
		limit = 1000
	}

	books, err := h.db.ListBooks(ctx, skip, limit)
	if err != nil {
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

// GetBook handles GET /books/{book_id}.
func (h *Handlers) GetBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, ok := parseBookID(w, r)
	if !ok {
		return
	}

	book, err := h.db.GetBook(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("Book with ID %d not found", id)
		writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %d not found", id))
		return
	}
	if err != nil {
		log.Printf("Error retrieving book %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while retrieving book")
		return
	}

	log.Printf("Retrieved book: %s", book.Title)
	writeJSON(w, http.StatusOK, NewBookResponse(book))
}

// CreateBook handles POST /books/ and returns 201 on success.
func (h *Handlers) CreateBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("Invalid request body: %v", err))
		return
	}
	if verr := req.Validate(); verr != nil {
		writeError(w, http.StatusUnprocessableEntity, verr.Error())
		return
	}

	book, err := h.db.CreateBook(ctx, &req)
	if errors.Is(err, ErrDuplicateBook) {
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

// UpdateBook handles PUT /books/{book_id}, applying only the provided fields.
func (h *Handlers) UpdateBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, ok := parseBookID(w, r)
	if !ok {
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("Invalid request body: %v", err))
		return
	}
	if verr := req.Validate(); verr != nil {
		writeError(w, http.StatusUnprocessableEntity, verr.Error())
		return
	}

	book, err := h.db.UpdateBook(ctx, id, &req)
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("Book with ID %d not found for update", id)
		writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %d not found", id))
		return
	}
	if errors.Is(err, ErrDuplicateBook) {
		log.Printf("Integrity error updating book: %v", err)
		writeError(w, http.StatusBadRequest, "Book with this title and author already exists")
		return
	}
	if err != nil {
		log.Printf("Error updating book %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while updating book")
		return
	}

	log.Printf("Updated book: %s", book.Title)
	writeJSON(w, http.StatusOK, NewBookResponse(book))
}

// DeleteBook handles DELETE /books/{book_id} and returns 204 on success.
func (h *Handlers) DeleteBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, ok := parseBookID(w, r)
	if !ok {
		return
	}

	book, err := h.db.DeleteBook(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("Book with ID %d not found for deletion", id)
		writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %d not found", id))
		return
	}
	if err != nil {
		log.Printf("Error deleting book %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while deleting book")
		return
	}

	log.Printf("Deleted book: %s", book.Title)
	w.WriteHeader(http.StatusNoContent)
}

// parseBookID extracts and validates the {book_id} path parameter. On failure
// it writes a 422 response and returns ok=false.
//
// MIGRATION_NOTE: FastAPI coerces the path parameter to int and returns 422 for
// non-integer values; we reproduce that here rather than 404.
func parseBookID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "book_id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("Invalid book ID: %q", raw))
		return 0, false
	}
	return id, true
}
