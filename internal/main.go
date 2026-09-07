package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"migrated-app/internal/database"
	"migrated-app/internal/model"
	"migrated-app/internal/schemas"
	"errors"
)

var logger *zap.Logger

func init() {
	var err error
	logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
}

// buildRouter creates and returns the main router for the application.
func buildRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", rootHandler)
	r.Get("/books/", listBooksHandler)
	r.Get("/books/{book_id}", getBookHandler)
	r.Post("/books/", createBookHandler)
	r.Put("/books/{book_id}", updateBookHandler)
	r.Delete("/books/{book_id}", deleteBookHandler)
	r.Get("/health", healthCheckHandler)

	return r
}

// rootHandler provides basic API information.
func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Welcome to Book Catalog API",
		"version": "1.0.0",
		"docs_url": "/docs",
	})
}

// listBooksHandler retrieves all books with pagination.
func listBooksHandler(w http.ResponseWriter, r *http.Request) {
	skip := r.URL.Query().Get("skip")
	limit := r.URL.Query().Get("limit")

	skipInt, _ := strconv.Atoi(skip)
	limitInt, _ := strconv.Atoi(limit)

	if limitInt > 1000 {
		limitInt = 1000
	}

	db, err := database.GetSyncDB()
	if err != nil {
		http.Error(w, "Failed to get database", http.StatusInternalServerError)
		return
	}
	defer database.CloseSyncDB(db)

	books, err := model.ListBooks(context.Background(), db, skipInt, limitInt)
	if err != nil {
		logger.Error("Error retrieving books", zap.Error(err))
		http.Error(w, "Internal server error while retrieving books", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

// getBookHandler retrieves a single book by its ID.
func getBookHandler(w http.ResponseWriter, r *http.Request) {
	bookIDStr := chi.URLParam(r, "book_id")
	bookID, err := strconv.Atoi(bookIDStr)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	db, err := database.GetSyncDB()
	if err != nil {
		http.Error(w, "Failed to get database", http.StatusInternalServerError)
		return
	}
	defer database.CloseSyncDB(db)

	book, err := model.GetBookByID(context.Background(), db, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Book not found", http.StatusNotFound)
			return
		}
		logger.Error("Error retrieving book", zap.Error(err))
		http.Error(w, "Internal server error while retrieving book", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

// createBookHandler creates a new book.
func createBookHandler(w http.ResponseWriter, r *http.Request) {
	var bookCreate schemas.BookCreate
	if err := json.NewDecoder(r.Body).Decode(&bookCreate); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := bookCreate.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db, err := database.GetSyncDB()
	if err != nil {
		http.Error(w, "Failed to get database", http.StatusInternalServerError)
		return
	}
	defer database.CloseSyncDB(db)

	book, _ := NewBook(bookCreate.Title, bookCreate.Author, bookCreate.PublishedYear, bookCreate.Summary)
	if err := book.Create(context.Background(), db); err != nil {
		if _, ok := err.(*model.UniqueConstraintViolationError); ok {
			http.Error(w, "Book with this title and author already exists", http.StatusBadRequest)
			return
		}
		logger.Error("Error creating book", zap.Error(err))
		http.Error(w, "Internal server error while creating book", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(book)
}

// updateBookHandler updates an existing book.
func updateBookHandler(w http.ResponseWriter, r *http.Request) {
	bookIDStr := chi.URLParam(r, "book_id")
	bookID, err := strconv.Atoi(bookIDStr)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	var bookUpdate schemas.BookUpdate
	if err := json.NewDecoder(r.Body).Decode(&bookUpdate); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	db, err := database.GetSyncDB()
	if err != nil {
		http.Error(w, "Failed to get database", http.StatusInternalServerError)
		return
	}
	defer database.CloseSyncDB(db)

	book, err := model.GetBookByID(context.Background(), db, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Book not found", http.StatusNotFound)
			return
		}
		logger.Error("Error retrieving book for update", zap.Error(err))
		http.Error(w, "Internal server error while retrieving book", http.StatusInternalServerError)
		return
	}

	if err := book.Update(context.Background(), db, bookUpdate); err != nil {
		if _, ok := err.(*model.UniqueConstraintViolationError); ok {
			http.Error(w, "Book with this title and author already exists", http.StatusBadRequest)
			return
		}
		logger.Error("Error updating book", zap.Error(err))
		http.Error(w, "Internal server error while updating book", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

// deleteBookHandler deletes a book by its ID.
func deleteBookHandler(w http.ResponseWriter, r *http.Request) {
	bookIDStr := chi.URLParam(r, "book_id")
	bookID, err := strconv.Atoi(bookIDStr)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	db, err := database.GetSyncDB()
	if err != nil {
		http.Error(w, "Failed to get database", http.StatusInternalServerError)
		return
	}
	defer database.CloseSyncDB(db)

	book, err := model.GetBookByID(context.Background(), db, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Book not found", http.StatusNotFound)
			return
		}
		logger.Error("Error retrieving book for deletion", zap.Error(err))
		http.Error(w, "Internal server error while retrieving book", http.StatusInternalServerError)
		return
	}

	if err := book.Delete(context.Background(), db); err != nil {
		logger.Error("Error deleting book", zap.Error(err))
		http.Error(w, "Internal server error while deleting book", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// healthCheckHandler provides a health check endpoint.
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "book-catalog-api",
	})
}
