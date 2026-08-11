package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func getPort() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8000"
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"detail": msg})
}

// NewRouter builds and returns the ServeMux wired to the given DB.
func NewRouter(db *sql.DB) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// /books  — list + create
	mux.HandleFunc("/books", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listBooks(w, r, db)
		case http.MethodPost:
			createBook(w, r, db)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	// /books/{id}  — get, update, delete
	mux.HandleFunc("/books/", func(w http.ResponseWriter, r *http.Request) {
		idStr := strings.TrimPrefix(r.URL.Path, "/books/")
		if idStr == "" {
			writeError(w, http.StatusBadRequest, "missing book id")
			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid book id")
			return
		}
		switch r.Method {
		case http.MethodGet:
			getBook(w, r, db, id)
		case http.MethodPut:
			updateBook(w, r, db, id)
		case http.MethodDelete:
			deleteBook(w, r, db, id)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	return mux
}

func listBooks(w http.ResponseWriter, _ *http.Request, db *sql.DB) {
	rows, err := db.Query(`SELECT id, title, author, isbn, description, published_year, price, created_at, updated_at FROM books ORDER BY id`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	books := []Book{}
	for rows.Next() {
		var b Book
		var isbn, description sql.NullString
		var publishedYear sql.NullInt64
		var price sql.NullFloat64
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &isbn, &description, &publishedYear, &price, &b.CreatedAt, &b.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if isbn.Valid {
			b.ISBN = isbn.String
		}
		if description.Valid {
			b.Description = description.String
		}
		if publishedYear.Valid {
			b.PublishedYear = int(publishedYear.Int64)
		}
		if price.Valid {
			b.Price = price.Float64
		}
		books = append(books, b)
	}
	writeJSON(w, http.StatusOK, books)
}

func getBook(w http.ResponseWriter, _ *http.Request, db *sql.DB, id int) {
	var b Book
	var isbn, description sql.NullString
	var publishedYear sql.NullInt64
	var price sql.NullFloat64
	err := db.QueryRow(`SELECT id, title, author, isbn, description, published_year, price, created_at, updated_at FROM books WHERE id = $1`, id).
		Scan(&b.ID, &b.Title, &b.Author, &isbn, &description, &publishedYear, &price, &b.CreatedAt, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, fmt.Sprintf("book %d not found", id))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if isbn.Valid {
		b.ISBN = isbn.String
	}
	if description.Valid {
		b.Description = description.String
	}
	if publishedYear.Valid {
		b.PublishedYear = int(publishedYear.Int64)
	}
	if price.Valid {
		b.Price = price.Float64
	}
	writeJSON(w, http.StatusOK, b)
}

func createBook(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var input BookCreate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if input.Title == "" || input.Author == "" {
		writeError(w, http.StatusUnprocessableEntity, "title and author are required")
		return
	}

	var b Book
	now := time.Now()
	err := db.QueryRow(
		`INSERT INTO books (title, author, isbn, description, published_year, price, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING id, title, author, isbn, description, published_year, price, created_at, updated_at`,
		input.Title, input.Author,
		nullString(input.ISBN), nullString(input.Description),
		nullInt(input.PublishedYear), nullFloat(input.Price),
		now, now,
	).Scan(&b.ID, &b.Title, &b.Author,
		nullableString(&b.ISBN), nullableString(&b.Description),
		nullableInt(&b.PublishedYear), nullableFloat(&b.Price),
		&b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func updateBook(w http.ResponseWriter, r *http.Request, db *sql.DB, id int) {
	var input BookUpdate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Build dynamic SET clause
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIdx := 1

	if input.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *input.Title)
		argIdx++
	}
	if input.Author != nil {
		setClauses = append(setClauses, fmt.Sprintf("author = $%d", argIdx))
		args = append(args, *input.Author)
		argIdx++
	}
	if input.ISBN != nil {
		setClauses = append(setClauses, fmt.Sprintf("isbn = $%d", argIdx))
		args = append(args, *input.ISBN)
		argIdx++
	}
	if input.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *input.Description)
		argIdx++
	}
	if input.PublishedYear != nil {
		setClauses = append(setClauses, fmt.Sprintf("published_year = $%d", argIdx))
		args = append(args, *input.PublishedYear)
		argIdx++
	}
	if input.Price != nil {
		setClauses = append(setClauses, fmt.Sprintf("price = $%d", argIdx))
		args = append(args, *input.Price)
		argIdx++
	}

	args = append(args, id)
	query := fmt.Sprintf(
		`UPDATE books SET %s WHERE id = $%d
		 RETURNING id, title, author, isbn, description, published_year, price, created_at, updated_at`,
		strings.Join(setClauses, ", "), argIdx,
	)

	var b Book
	err := db.QueryRow(query, args...).Scan(
		&b.ID, &b.Title, &b.Author,
		nullableString(&b.ISBN), nullableString(&b.Description),
		nullableInt(&b.PublishedYear), nullableFloat(&b.Price),
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, fmt.Sprintf("book %d not found", id))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func deleteBook(w http.ResponseWriter, _ *http.Request, db *sql.DB, id int) {
	res, err := db.Exec(`DELETE FROM books WHERE id = $1`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, http.StatusNotFound, fmt.Sprintf("book %d not found", id))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// helpers for nullable SQL values
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
func nullInt(i int) interface{} {
	if i == 0 {
		return nil
	}
	return i
}
func nullFloat(f float64) interface{} {
	if f == 0 {
		return nil
	}
	return f
}
func nullableString(dst *string) *sql.NullString {
	return &sql.NullString{}
}
func nullableInt(dst *int) *sql.NullInt64 {
	return &sql.NullInt64{}
}
func nullableFloat(dst *float64) *sql.NullFloat64 {
	return &sql.NullFloat64{}
}

func main() {
	db, err := InitDB()
	if err != nil {
		log.Fatalf("failed to initialise database: %v", err)
	}
	defer db.Close()

	DB = db
	router := NewRouter(db)
	addr := ":" + getPort()
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}