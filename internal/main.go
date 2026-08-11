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

	"migrated-app/internal/config"
)

// Handlers bundles the dependencies shared by every HTTP handler in the Book
// Catalog API.
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

// writeError mirrors FastAPI's custom HTTPException handler.
func writeError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}

// BuildRouter constructs the fully-wired HTTP handler for the Book Catalog API
// using configuration from environment variables. It is the exported entry
// point called by cmd/server/main.go.
func BuildRouter(ctx context.Context) http.Handler {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	if err := db.InitSchema(ctx); err != nil {
		log.Fatalf("failed to init schema: %v", err)
	}
	log.Println("Database initialized successfully")

	h := NewHandlers(db)
	return h.Router()
}

// Router builds and returns the chi router for these handlers.
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

// Root handles GET /.
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

// ListBooks handles GET /books/.
func (h *Handlers) ListBooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	skip := parseIntQuery(r, "skip", 0)
	limit := parseIntQuery(r, "limit", 100)
	if limit > 1000 {
		limit = 1000
	}

	books, err := h.db.ListBooks(ctx, skip, limit)
	if err != nil {
		log.Printf("Error retrieving books: %v", err)
		writeError(w, http.StatusInternalServerError, "Internal server error while retrieving books")
		return
	}

	resp := make([]*BookResponse, 0, len(books))
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

// CreateBook handles POST /books/.
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

// UpdateBook handles PUT /books/{book_id}.
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

// DeleteBook handles DELETE /books/{book_id}.
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

// parseBookID extracts and validates the {book_id} path parameter.
func parseBookID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "book_id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("Invalid book ID: %q", raw))
		return 0, false
	}
	return id, true
}
