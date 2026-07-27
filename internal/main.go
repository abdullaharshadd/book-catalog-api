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
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type bookStore interface {
	List(ctx context.Context, skip, limit int) ([]Book, error)
	Get(ctx context.Context, id int64) (Book, bool, error)
	Create(ctx context.Context, b *Book) error
	Update(ctx context.Context, b *Book) error
	Delete(ctx context.Context, id int64) (bool, error)
}

var errDuplicate = errors.New("book with this title and author already exists")

type BookRepository struct {
	db *DB
}

func NewBookRepository(db *DB) *BookRepository {
	return &BookRepository{db: db}
}

func (r *BookRepository) List(ctx context.Context, skip, limit int) ([]Book, error) {
	const q = `SELECT id, title, author, published_year, summary
	           FROM books ORDER BY id OFFSET $1 LIMIT $2`
	books := []Book{}
	if err := r.db.SelectContext(ctx, &books, q, skip, limit); err != nil {
		return nil, fmt.Errorf("list books: %w", err)
	}
	return books, nil
}

func (r *BookRepository) Get(ctx context.Context, id int64) (Book, bool, error) {
	const q = `SELECT id, title, author, published_year, summary
	           FROM books WHERE id = $1`
	var b Book
	if err := r.db.GetContext(ctx, &b, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Book{}, false, nil
		}
		return Book{}, false, fmt.Errorf("get book %d: %w", id, err)
	}
	return b, true, nil
}

func (r *BookRepository) Create(ctx context.Context, b *Book) error {
	const q = `INSERT INTO books (title, author, published_year, summary)
	           VALUES ($1, $2, $3, $4) RETURNING id`
	err := r.db.QueryRowContext(ctx, q, b.Title, b.Author, b.PublishedYear, b.Summary).Scan(&b.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return errDuplicate
		}
		return fmt.Errorf("create book: %w", err)
	}
	return nil
}

func (r *BookRepository) Update(ctx context.Context, b *Book) error {
	const q = `UPDATE books
	           SET title = $1, author = $2, published_year = $3, summary = $4
	           WHERE id = $5`
	_, err := r.db.ExecContext(ctx, q, b.Title, b.Author, b.PublishedYear, b.Summary, b.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return errDuplicate
		}
		return fmt.Errorf("update book %d: %w", b.ID, err)
	}
	return nil
}

func (r *BookRepository) Delete(ctx context.Context, id int64) (bool, error) {
	const q = `DELETE FROM books WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return false, fmt.Errorf("delete book %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete book %d rows affected: %w", id, err)
	}
	return n > 0, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	type sqlStater interface{ SQLState() string }
	var ss sqlStater
	if errors.As(err, &ss) {
		return ss.SQLState() == "23505"
	}
	return strings.Contains(err.Error(), "23505") ||
		strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}

type BookHandler struct {
	store bookStore
}

func NewBookHandler(store bookStore) *BookHandler {
	return &BookHandler{store: store}
}

func toResponse(b Book) BookResponse {
	resp, _ := NewBookResponse(&b)
	return resp
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("error encoding response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}

func (h *BookHandler) Root(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"message":  "Welcome to Book Catalog API",
		"version":  "1.0.0",
		"docs_url": "/docs",
	})
}

func (h *BookHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "book-catalog-api",
	})
}

func (h *BookHandler) ListBooks(w http.ResponseWriter, r *http.Request) {
	skip := parseIntQuery(r, "skip", 0)
	if skip < 0 {
		skip = 0
	}
	limit := parseIntQuery(r, "limit", 100)
	if limit > 1000 {
		limit = 1000
	}
	if limit < 0 {
		limit = 0
	}

	books, err := h.store.List(r.Context(), skip, limit)
	if err != nil {
		log.Printf("error retrieving books: %v", err)
		writeError(w, http.StatusInternalServerError, "Internal server error while retrieving books")
		return
	}

	resp := make([]BookResponse, 0, len(books))
	for _, b := range books {
		resp = append(resp, toResponse(b))
	}
	log.Printf("Retrieved %d books (skip=%d, limit=%d)", len(books), skip, limit)
	writeJSON(w, http.StatusOK, resp)
}

func (h *BookHandler) GetBook(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}

	book, found, err := h.store.Get(r.Context(), id)
	if err != nil {
		log.Printf("error retrieving book %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while retrieving book")
		return
	}
	if !found {
		log.Printf("Book with ID %d not found", id)
		writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %d not found", id))
		return
	}
	log.Printf("Retrieved book: %s", book.Title)
	writeJSON(w, http.StatusOK, toResponse(book))
}

func (h *BookHandler) CreateBook(w http.ResponseWriter, r *http.Request) {
	var in BookCreate
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if err := in.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	book := Book{
		Title:         in.Title,
		Author:        in.Author,
		PublishedYear: in.PublishedYear,
		Summary:       in.Summary,
	}

	if err := h.store.Create(r.Context(), &book); err != nil {
		if errors.Is(err, errDuplicate) {
			log.Printf("integrity error creating book: %v", err)
			writeError(w, http.StatusBadRequest, "Book with this title and author already exists")
			return
		}
		log.Printf("error creating book: %v", err)
		writeError(w, http.StatusInternalServerError, "Internal server error while creating book")
		return
	}
	log.Printf("Created new book: %s by %s", book.Title, book.Author)
	writeJSON(w, http.StatusCreated, toResponse(book))
}

func (h *BookHandler) UpdateBook(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}

	var upd BookUpdate
	if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if err := upd.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	book, found, err := h.store.Get(r.Context(), id)
	if err != nil {
		log.Printf("error updating book %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while updating book")
		return
	}
	if !found {
		log.Printf("Book with ID %d not found for update", id)
		writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %d not found", id))
		return
	}

	if upd.Present("title") && upd.Title != nil {
		book.Title = *upd.Title
	}
	if upd.Present("author") && upd.Author != nil {
		book.Author = *upd.Author
	}
	if upd.Present("published_year") && upd.PublishedYear != nil {
		book.PublishedYear = *upd.PublishedYear
	}
	if upd.Present("summary") {
		book.Summary = upd.Summary
	}

	if err := h.store.Update(r.Context(), &book); err != nil {
		if errors.Is(err, errDuplicate) {
			log.Printf("integrity error updating book: %v", err)
			writeError(w, http.StatusBadRequest, "Book with this title and author already exists")
			return
		}
		log.Printf("error updating book %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while updating book")
		return
	}
	log.Printf("Updated book: %s", book.Title)
	writeJSON(w, http.StatusOK, toResponse(book))
}

func (h *BookHandler) DeleteBook(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}

	deleted, err := h.store.Delete(r.Context(), id)
	if err != nil {
		log.Printf("error deleting book %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "Internal server error while deleting book")
		return
	}
	if !deleted {
		log.Printf("Book with ID %d not found for deletion", id)
		writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %d not found", id))
		return
	}
	log.Printf("Deleted book %d", id)
	w.WriteHeader(http.StatusNoContent)
}

func parseIntQuery(r *http.Request, key string, def int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "book_id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Book with ID %s not found", raw))
		return 0, false
	}
	return id, true
}

var defaultHandler *BookHandler

func SetHandler(h *BookHandler) {
	defaultHandler = h
}

func BuildRouter() http.Handler {
	return buildRouter()
}

func buildRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	h := defaultHandler
	if h == nil {
		log.Printf("WARNING: buildRouter called before SetHandler; handlers will return 500")
		h = NewBookHandler(unconfiguredStore{})
	}

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

type unconfiguredStore struct{}

func (unconfiguredStore) List(context.Context, int, int) ([]Book, error) {
	return nil, errors.New("book store not configured")
}
func (unconfiguredStore) Get(context.Context, int64) (Book, bool, error) {
	return Book{}, false, errors.New("book store not configured")
}
func (unconfiguredStore) Create(context.Context, *Book) error {
	return errors.New("book store not configured")
}
func (unconfiguredStore) Update(context.Context, *Book) error {
	return errors.New("book store not configured")
}
func (unconfiguredStore) Delete(context.Context, int64) (bool, error) {
	return false, errors.New("book store not configured")
}