package main

import (
	"context"
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

func connectDB(dsn string) (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error
	for i := 0; i < 10; i++ {
		db, err = sqlx.Connect("postgres", dsn)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err = db.PingContext(ctx)
			cancel()
			if err == nil {
				return db, nil
			}
		}
		log.Printf("waiting for database (attempt %d/10): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("ping database: %w", err)
}

func initSchema(db *sqlx.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS books (
			id SERIAL PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			author VARCHAR(255) NOT NULL,
			published_year INTEGER NOT NULL,
			summary TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(title, author)
		)
	`)
	return err
}

func main() {
	dsn := getDSN()
	log.Printf("connecting to database...")

	db, err := connectDB(dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("database connected successfully")

	if err := initSchema(db); err != nil {
		log.Fatalf("failed to initialize schema: %v", err)
	}
	log.Println("database schema initialized")

	h := NewHandler(db)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/", h.Root)
	r.Get("/health", h.HealthCheck)
	r.Get("/books/", h.ListBooks)
	r.Get("/books/{book_id}", h.GetBook)
	r.Post("/books/", h.CreateBook)
	r.Put("/books/{book_id}", h.UpdateBook)
	r.Delete("/books/{book_id}", h.DeleteBook)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	addr := ":" + port
	log.Printf("starting server on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}