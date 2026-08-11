package internal

import (
	"os"
)

func getDBURL() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	if v := os.Getenv("DB_URL"); v != "" {
		return v
	}
	return "postgres://app:app@db:5432/app?sslmode=disable"
}