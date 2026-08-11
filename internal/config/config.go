package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL string
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DB_URL")
	}
	if dbURL == "" {
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
		dbName := os.Getenv("DB_NAME")
		if dbName == "" {
			dbName = os.Getenv("DB_DATABASE")
		}
		if dbName == "" {
			dbName = os.Getenv("POSTGRES_DB")
		}
		if dbName == "" {
			dbName = "app"
		}
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbName)
	}
	return &Config{
		DatabaseURL: dbURL,
	}, nil
}