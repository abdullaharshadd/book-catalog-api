package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func getDSN() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	if v := os.Getenv("DB_URL"); v != "" {
		return v
	}
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "db"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = os.Getenv("DB_USERNAME")
	}
	if user == "" {
		user = os.Getenv("POSTGRES_USER")
	}
	if user == "" {
		user = "app"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = os.Getenv("POSTGRES_PASSWORD")
	}
	if password == "" {
		password = "app"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = os.Getenv("DB_DATABASE")
	}
	if dbname == "" {
		dbname = os.Getenv("POSTGRES_DB")
	}
	if dbname == "" {
		dbname = "app"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
}

func connectDB() (*sqlx.DB, error) {
	dsn := getDSN()
	var db *sqlx.DB
	var err error
	for i := 0; i < 10; i++ {
		db, err = sqlx.Connect("postgres", dsn)
		if err == nil {
			return db, nil
		}
		log.Printf("waiting for database (attempt %d/10): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("ping database: %w", err)
}

func main() {
	db, err := connectDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create tables if not exists
	schema := `
	CREATE TABLE IF NOT EXISTS books (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		author TEXT NOT NULL,
		published_year INT NOT NULL,
		summary TEXT
	);`
	if _, err := db.Exec(schema); err != nil {
		log.Fatalf("failed to create schema: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	h := &BookHandler{db: db}

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"Welcome to Book Catalog API","version":"1.0.0"}`))
	})
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"healthy","service":"book-catalog-api"}`))
	})
	r.Get("/books/", h.ListBooks)
	r.Get("/books/{bookID}", h.GetBook)
	r.Post("/books/", h.CreateBook)
	r.Put("/books/{bookID}", h.UpdateBook)
	r.Delete("/books/{bookID}", h.DeleteBook)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	log.Printf("starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}