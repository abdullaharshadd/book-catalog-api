// internal/database.go
package internal

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"

	"database/sql"
	"github.com/jmoiron/sqlx"
	"migrated-app/internal/model"
)

var (
	syncDBOnce sync.Once
	asyncDBOnce sync.Once
	syncDB     *sql.DB
	asyncDB    *sqlx.DB
)

func InitializeDatabase(ctx context.Context) error {
	var err error
	syncDBOnce.Do(func() {
		databaseURL := os.Getenv("DATABASE_URL")
		if databaseURL == "" {
			databaseURL = "postgres://user:password@localhost:5432/books"
		}
		syncDB, err = sql.Open("postgres", databaseURL)
		if err != nil {
			log.Fatalf("failed to open sync database connection: %v", err)
		}
		if err := syncDB.PingContext(ctx); err != nil {
			log.Fatalf("failed to ping sync database: %v", err)
		}
	})

	asyncDBOnce.Do(func() {
		asyncDatabaseURL := os.Getenv("ASYNC_DATABASE_URL")
		if asyncDatabaseURL == "" {
			asyncDatabaseURL = "postgres://user:password@localhost:5432/books?sslmode=disable"
		}
		asyncDB, err = sql.Open("postgres", asyncDatabaseURL)
		if err != nil {
			log.Fatalf("failed to open async database connection: %v", err)
		}
		asyncSqlxDB := sqlx.NewDb(asyncDB, "postgres")
		if err := asyncSqlxDB.PingContext(ctx); err != nil {
			log.Fatalf("failed to ping async database: %v", err)
		}
		asyncDB = asyncSqlxDB
	})

	if err != nil {
		return errors.Wrap(err, "error initializing database")
	}

	return model.CreateSchema(ctx, syncDB)
}

func GetSyncDB() (*sql.DB, error) {
	if syncDB == nil {
		return nil, errors.New("sync database not initialized")
	}
	return syncDB, nil
}

func GetAsyncDB(ctx context.Context) (*sqlx.DB, error) {
	if asyncDB == nil {
		return nil, errors.New("async database not initialized")
	}
	return asyncDB, nil
}

// MIGRATION_NOTE: The original Python code used a synchronous session for schema creation.
// In Go, we use a dedicated function to create the schema at startup.