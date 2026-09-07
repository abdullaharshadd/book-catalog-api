// internal/main.go

package internal

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"migrated-app/internal/database"
	"migrated-app/internal/handler"
	"migrated-app/internal/model"
	"migrated-app/internal/schemas"
)

const (
	Version = "1.0.0"
	Author  = "Your Name"
	Email   = "you@example.com"
)

func buildRouter() http.Handler {
	r := chi.NewRouter()

	// Middleware setup
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health Check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, map[string]string{"status": "healthy", "service": "book-catalog-api"})
	})

	// Root Endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, map[string]interface{}{
			"message":    "Welcome to Book Catalog API",
			"version":    Version,
			"docs_url":   "/docs",
		})
	})

	// Books Endpoints
	r.Route("/books", func(r chi.Router) {
		r.Get("/", handler.ListBooks)
		r.Get("/{book_id}", handler.GetBook)
		r.Post("/", handler.CreateBook)
		r.Put("/{book_id}", handler.UpdateBook)
		r.Delete("/{book_id}", handler.DeleteBook)
	})

	return r
}

// Handler functions
func ListBooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if limit > 1000 {
		limit = 1000
	}

	db, ok := database.FromContext(ctx)
	if !ok {
		log.Error().Msg("Database not found in context")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	books, err := model.GetBooks(ctx, db, skip, limit)
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving books (skip=%d, limit=%d)", skip, limit)
		http.Error(w, "Internal server error while retrieving books", http.StatusInternalServerError)
		return
	}

	log.Info().Msgf("Retrieved %d books (skip=%d, limit=%d)", len(books), skip, limit)
	render.JSON(w, r, books)
}

func GetBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	bookIDStr := chi.URLParam(r, "book_id")
	bookID, err := strconv.Atoi(bookIDStr)
	if err != nil {
		log.Error().Err(err).Msgf("Invalid book ID: %s", bookIDStr)
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	db, ok := database.FromContext(ctx)
	if !ok {
		log.Error().Msg("Database not found in context")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	book, err := model.GetByID(ctx, db, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn().Msgf("Book with ID %d not found", bookID)
			http.Error(w, fmt.Sprintf("Book with ID %d not found", bookID), http.StatusNotFound)
			return
		}
		log.Error().Err(err).Msgf("Error retrieving book with ID %d", bookID)
		http.Error(w, "Internal server error while retrieving book", http.StatusInternalServerError)
		return
	}

	log.Info().Msgf("Retrieved book: %s", book.Title)
	render.JSON(w, r, schemas.FromModel(book))
}

func CreateBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var bookCreate schemas.BookCreate
	if err := render.DecodeJSON(r.Body, &bookCreate); err != nil {
		log.Error().Err(err).Msg("Failed to decode request body")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := bookCreate.Validate(); err != nil {
		log.Error().Err(err).Msg("Validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db, ok := database.FromContext(ctx)
	if !ok {
		log.Error().Msg("Database not found in context")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	book, err := model.NewBook(bookCreate.Title, bookCreate.Author, bookCreate.PublishedYear, bookCreate.Summary)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create new book")
		http.Error(w, "Internal server error while creating book", http.StatusInternalServerError)
		return
	}

	if err := model.Insert(ctx, db, book); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn().Msgf("Failed to insert book due to unique constraint violation")
			http.Error(w, "Book with this title and author already exists", http.StatusBadRequest)
			return
		}
		log.Error().Err(err).Msg("Failed to insert book")
		http.Error(w, "Internal server error while creating book", http.StatusInternalServerError)
		return
	}

	log.Info().Msgf("Created new book: %s by %s", book.Title, book.Author)
	w.WriteHeader(http.StatusCreated)
	render.JSON(w, r, schemas.FromModel(book))
}

func UpdateBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	bookIDStr := chi.URLParam(r, "book_id")
	bookID, err := strconv.Atoi(bookIDStr)
	if err != nil {
		log.Error().Err(err).Msgf("Invalid book ID: %s", bookIDStr)
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	var bookUpdate schemas.BookUpdate
	if err := render.DecodeJSON(r.Body, &bookUpdate); err != nil {
		log.Error().Err(err).Msg("Failed to decode request body")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	db, ok := database.FromContext(ctx)
	if !ok {
		log.Error().Msg("Database not found in context")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	book, err := model.GetByID(ctx, db, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn().Msgf("Book with ID %d not found for update", bookID)
			http.Error(w, fmt.Sprintf("Book with ID %d not found", bookID), http.StatusNotFound)
			return
		}
		log.Error().Err(err).Msgf("Error retrieving book with ID %d for update", bookID)
		http.Error(w, "Internal server error while retrieving book", http.StatusInternalServerError)
		return
	}

	if bookUpdate.Title.Valid {
		book.Title = bookUpdate.Title.String
	}
	if bookUpdate.Author.Valid {
		book.Author = bookUpdate.Author.String
	}
	if bookUpdate.PublishedYear.Valid {
		book.PublishedYear = bookUpdate.PublishedYear.Int32
	}
	if bookUpdate.Summary.Valid {
		book.Summary = sql.NullString{String: bookUpdate.Summary.String, Valid: true}
	}

	if err := model.Update(ctx, db, book); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn().Msgf("Failed to update book due to unique constraint violation")
			http.Error(w, "Book with this title and author already exists", http.StatusBadRequest)
			return
		}
		log.Error().Err(err).Msgf("Error updating book with ID %d", bookID)
		http.Error(w, "Internal server error while updating book", http.StatusInternalServerError)
		return
	}

	log.Info().Msgf("Updated book: %s", book.Title)
	render.JSON(w, r, schemas.FromModel(book))
}

func DeleteBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	bookIDStr := chi.URLParam(r, "book_id")
	bookID, err := strconv.Atoi(bookIDStr)
	if err != nil {
		log.Error().Err(err).Msgf("Invalid book ID: %s", bookIDStr)
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	db, ok := database.FromContext(ctx)
	if !ok {
		log.Error().Msg("Database not found in context")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	book, err := model.GetByID(ctx, db, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn().Msgf("Book with ID %d not found for deletion", bookID)
			http.Error(w, fmt.Sprintf("Book with ID %d not found", bookID), http.StatusNotFound)
			return
		}
		log.Error().Err(err).Msgf("Error retrieving book with ID %d for deletion", bookID)
		http.Error(w, "Internal server error while retrieving book", http.StatusInternalServerError)
		return
	}

	if err := model.Delete(ctx, db, book); err != nil {
		log.Error().Err(err).Msgf("Error deleting book with ID %d", bookID)
		http.Error(w, "Internal server error while deleting book", http.StatusInternalServerError)
		return
	}

	log.Info().Msgf("Deleted book: %s", book.Title)
	w.WriteHeader(http.StatusNoContent)
}

// Startup function
func InitializeDatabase() {
	db, err := database.OpenDatabase("postgres://user:password@localhost/dbname?sslmode=disable")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer db.Close()

	if err := model.CreateSchema(db); err != nil {
		log.Fatal().Err(err).Msg("Failed to create schema")
	}

	log.Info().Msg("Database initialized successfully")
}