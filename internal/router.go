package internal

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

type Book struct {
	ID            int       `db:"id" json:"id"`
	Title         string    `db:"title" json:"title"`
	Author        string    `db:"author" json:"author"`
	PublishedYear *int      `db:"published_year" json:"published_year,omitempty"`
	ISBN          *string   `db:"isbn" json:"isbn,omitempty"`
	Description   *string   `db:"description" json:"description,omitempty"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

type BookCreate struct {
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	PublishedYear *int    `json:"published_year,omitempty"`
	ISBN          *string `json:"isbn,omitempty"`
	Description   *string `json:"description,omitempty"`
}

func BuildRouter(db *sqlx.DB) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	r.Get("/books", func(w http.ResponseWriter, r *http.Request) {
		var books []Book
		err := db.Select(&books, `SELECT id, title, author, published_year, isbn, description, created_at, updated_at FROM books ORDER BY id`)
		if err != nil {
			log.Error().Err(err).Msg("failed to list books")
			http.Error(w, `{"detail":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		if books == nil {
			books = []Book{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(books)
	})

	r.Post("/books", func(w http.ResponseWriter, r *http.Request) {
		var input BookCreate
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, `{"detail":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		if input.Title == "" || input.Author == "" {
			http.Error(w, `{"detail":"title and author are required"}`, http.StatusBadRequest)
			return
		}
		var book Book
		err := db.QueryRowx(
			`INSERT INTO books (title, author, published_year, isbn, description) VALUES ($1, $2, $3, $4, $5) RETURNING id, title, author, published_year, isbn, description, created_at, updated_at`,
			input.Title, input.Author, input.PublishedYear, input.ISBN, input.Description,
		).StructScan(&book)
		if err != nil {
			log.Error().Err(err).Msg("failed to create book")
			http.Error(w, `{"detail":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(book)
	})

	r.Get("/books/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, `{"detail":"invalid id"}`, http.StatusBadRequest)
			return
		}
		var book Book
		err = db.QueryRowx(`SELECT id, title, author, published_year, isbn, description, created_at, updated_at FROM books WHERE id=$1`, id).StructScan(&book)
		if err != nil {
			http.Error(w, `{"detail":"book not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(book)
	})

	r.Put("/books/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, `{"detail":"invalid id"}`, http.StatusBadRequest)
			return
		}
		var input BookCreate
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, `{"detail":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		var book Book
		err = db.QueryRowx(
			`UPDATE books SET title=$1, author=$2, published_year=$3, isbn=$4, description=$5, updated_at=NOW() WHERE id=$6 RETURNING id, title, author, published_year, isbn, description, created_at, updated_at`,
			input.Title, input.Author, input.PublishedYear, input.ISBN, input.Description, id,
		).StructScan(&book)
		if err != nil {
			http.Error(w, `{"detail":"book not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(book)
	})

	r.Delete("/books/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, `{"detail":"invalid id"}`, http.StatusBadRequest)
			return
		}
		result, err := db.Exec(`DELETE FROM books WHERE id=$1`, id)
		if err != nil {
			http.Error(w, `{"detail":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			http.Error(w, `{"detail":"book not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	return r
}