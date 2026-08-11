package internal

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func getDBURL() string {
	for _, env := range []string{"DATABASE_URL", "DB_URL"} {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = os.Getenv("DB_USER")
	}
	if user == "" {
		user = "app"
	}
	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		password = os.Getenv("DB_PASSWORD")
	}
	if password == "" {
		password = "app"
	}
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "db"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	dbname := os.Getenv("POSTGRES_DB")
	if dbname == "" {
		dbname = os.Getenv("DB_NAME")
	}
	if dbname == "" {
		dbname = "app"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
}

func NewDB() (*sql.DB, error) {
	dsn := getDBURL()
	log.Info().Str("dsn", dsn).Msg("connecting to database")

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	for i := 0; i < 10; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		log.Warn().Err(err).Int("attempt", i+1).Msg("waiting for database")
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	return db, nil
}