package internal

import (
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func NewDB() (*sqlx.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DB_URL")
	}
	if dsn == "" {
		host := getEnvDefault("DB_HOST", "db")
		port := getEnvDefault("DB_PORT", "5432")
		user := getEnvDefault("DB_USER", getEnvDefault("DB_USERNAME", getEnvDefault("POSTGRES_USER", "app")))
		password := getEnvDefault("DB_PASSWORD", getEnvDefault("POSTGRES_PASSWORD", "app"))
		dbname := getEnvDefault("DB_NAME", getEnvDefault("DB_DATABASE", getEnvDefault("POSTGRES_DB", "app")))
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
	}

	var db *sqlx.DB
	var err error
	for i := 0; i < 10; i++ {
		db, err = sqlx.Open("pgx", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				return db, nil
			}
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	return nil, fmt.Errorf("ping: %w", err)
}