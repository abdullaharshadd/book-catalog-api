package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func getDBURL() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	if v := os.Getenv("DB_URL"); v != "" {
		return v
	}
	return "postgres://app:app@db:5432/app?sslmode=disable"
}

func InitDB() (*sql.DB, error) {
	dsn := getDBURL()

	var db *sql.DB
	var err error

	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			break
		}
		log.Printf("Database not ready (attempt %d/10): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := createSchema(db); err != nil {
		return nil, fmt.Errorf("could not create schema: %w", err)
	}

	log.Println("Database connected and schema ready")
	return db, nil
}

func createSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS books (
			id            SERIAL PRIMARY KEY,
			title         VARCHAR(255) NOT NULL,
			author        VARCHAR(255) NOT NULL,
			isbn          VARCHAR(20)  UNIQUE,
			description   TEXT,
			published_year INTEGER,
			price         NUMERIC(10,2),
			created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)
	`)
	return err
}