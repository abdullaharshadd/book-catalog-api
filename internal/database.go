package internal

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"internal/model"
)

// Database represents the database connection and session factories.
type Database struct {
	syncDB  *sqlx.DB
	asyncDB *sqlx.DB
}

// NewDatabase initializes and returns a new Database instance.
func NewDatabase(ctx context.Context) (*Database, error) {
	dbConfig := viper.GetString("DATABASE_URL")
	asyncDBConfig := viper.GetString("ASYNC_DATABASE_URL")

	syncDB, err := sqlx.Open("postgres", dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open sync database: %w", err)
	}
	defer syncDB.Close()

	asyncDB, err := sqlx.Open("postgres", asyncDBConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open async database: %w", err)
	}
	defer asyncDB.Close()

	return &Database{
		syncDB:  syncDB,
		asyncDB: asyncDB,
	}, nil
}

// InitDB initializes the database by creating all tables.
func (db *Database) InitDB(ctx context.Context) error {
	query := `CREATE TABLE IF NOT EXISTS books (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		author TEXT NOT NULL,
		published_year INT NOT NULL,
		summary TEXT
	)`
	_, err := db.syncDB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	return nil
}

// GetDB returns a new async database session for each request.
func (db *Database) GetDB(ctx context.Context) (*sqlx.Tx, error) {
	tx, err := db.asyncDB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

// GetSyncDB returns a new sync database session for testing and synchronous operations.
func (db *Database) GetSyncDB() (*sqlx.Tx, error) {
	tx, err := db.syncDB.BeginTxx(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin sync transaction: %w", err)
	}
	return tx, nil
}
