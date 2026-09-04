package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"migrated-app/internal/model"
)

// BuildRouter constructs the main HTTP router for the book catalog service.
func BuildRouter() http.Handler {
	return buildRouter()
}

// buildRouter constructs the main HTTP router for the book catalog service.
func buildRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", root)
	r.Route("/books", func(r chi.Router) {
		r.Get("/", listBooks)
		r.Get("/{book_id}", getBook)
		r.Post("/", createBook)
		r.Put("/{book_id}", updateBook)
		r.Delete("/{book_id}", deleteBook)
	})
	r.Get("/health", healthCheck)

	return r
}

// getSyncDB returns a sqlx.DB connected to the configured postgres database.
func getSyncDB() *sqlx.DB {
	dbURL := viper.GetString("DATABASE_URL")
	db, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to open database")
	}
	return db
}

// root is the root endpoint with API information.
func root(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message":  "Welcome to Book Catalog API",
		"version":  "1.0.0",
		"docs_url": "/docs",
	})
}

// listBooks retrieves all books with pagination.
func listBooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 100
	}
	limit = minInt(limit, 1000)
	db := getSyncDB()
	defer db.Close()

	query := "SELECT * FROM book OFFSET $1 LIMIT $2"
	rows, err := db.QueryContext(ctx, query, skip, limit)
	if err != nil {
		log.Error().Err(err).Msg("Error retrieving books")
		http.Error(w, "Internal server error while retrieving books", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	posts := []model.BookResponse{}
	for rows.Next() {
		var book model.BookResponse
		if err := rows.Scan(&book.ID, &book.Title, &book.Author, &book.PublishedYear, &book.Summary); err != nil {
			log.Error().Err(err).Msg("Error scanning book")
			http.Error(w, "Internal server error while retrieving books", http.StatusInternalServerError)
			return
		}
		posts = append(posts, book)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

// getBook retrieves a single book by its ID.
func getBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	bookID := chi.URLParam(r, "book_id")
	db := getSyncDB()
	defer db.Close()

	query := "SELECT * FROM book WHERE id=$1"
	var book model.BookResponse
	if err := db.GetContext(ctx, &book, query, bookID); err != nil {
		if err == sql.ErrNoRows {
			log.Warn().Msgf("Book with ID %s not found", bookID)
			http.Error(w, fmt.Sprintf("Book with ID %s not found", bookID), http.StatusNotFound)
			return
		}
		log.Error().Err(err).Msgf("Error retrieving book %s", bookID)
		http.Error(w, "Internal server error while retrieving book", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

// createBook creates a new book.
func createBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var bookCreate model.BookCreate
	if err := json.NewDecoder(r.Body).Decode(&bookCreate); err != nil {
		log.Error().Err(err).Msg("Error decoding request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	db := getSyncDB()
	defer db.Close()

	query := "INSERT INTO book (title, author, published_year, summary) VALUES ($1, $2, $3, $4) RETURNING id"
	var newID int64
	if err := db.QueryRowContext(ctx, query, bookCreate.Title, bookCreate.Author, bookCreate.PublishedYear, bookCreate.Summary).Scan(&newID); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
			log.Error().Err(err).Msg("Integrity error creating book")
			http.Error(w, "Book with this title and author already exists", http.StatusBadRequest)
			return
		}
		log.Error().Err(err).Msg("Error creating book")
		http.Error(w, "Internal server error while creating book", http.StatusInternalServerError)
		return
	}
	newBook := model.BookResponse{ID: newID, Title: bookCreate.Title, Author: bookCreate.Author, PublishedYear: bookCreate.PublishedYear, Summary: bookCreate.Summary}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newBook)
}

// updateBook updates an existing book.
func updateBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	bookID := chi.URLParam(r, "book_id")
	var bookUpdate model.BookUpdate
	if err := json.NewDecoder(r.Body).Decode(&bookUpdate); err != nil {
		log.Error().Err(err).Msg("Error decoding request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	db := getSyncDB()
	defer db.Close()

	query := "UPDATE book SET title=$1, author=$2, published_year=$3, summary=$4 WHERE id=$5 RETURNING id"
	var updatedID int64
	if err := db.QueryRowContext(ctx, query, bookUpdate.Title, bookUpdate.Author, bookUpdate.PublishedYear, bookUpdate.Summary, bookID).Scan(&updatedID); err != nil {
		if err == sql.ErrNoRows {
			log.Warn().Msgf("Book with ID %s not found for update", bookID)
			http.Error(w, fmt.Sprintf("Book with ID %s not found", bookID), http.StatusNotFound)
			return
		}
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
			log.Error().Err(err).Msg("Integrity error updating book")
			http.Error(w, "Book with this title and author already exists", http.StatusBadRequest)
			return
		}
		log.Error().Err(err).Msgf("Error updating book %s", bookID)
		http.Error(w, "Internal server error while updating book", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.BookResponse{ID: updatedID, Title: bookUpdate.Title, Author: bookUpdate.Author, PublishedYear: bookUpdate.PublishedYear, Summary: bookUpdate.Summary})
}

// deleteBook deletes a book by its ID.
func deleteBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	bookID := chi.URLParam(r, "book_id")
	db := getSyncDB()
	defer db.Close()

	query := "DELETE FROM book WHERE id=$1 RETURNING id"
	var deletedID int64
	if err := db.QueryRowContext(ctx, query, bookID).Scan(&deletedID); err != nil {
		if err == sql.ErrNoRows {
			log.Warn().Msgf("Book with ID %s not found for deletion", bookID)
			http.Error(w, fmt.Sprintf("Book with ID %s not found", bookID), http.StatusNotFound)
			return
		}
		log.Error().Err(err).Msgf("Error deleting book %s", bookID)
		http.Error(w, "Internal server error while deleting book", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// healthCheck is the health check endpoint.
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "book-catalog-api",
	})
}

// minInt returns the minimum of two integers.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Ensure unused imports are referenced.
var _ = context.Background
var _ *sqlx.DB
