// internal/database.go
package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	_ "github.com/lib/pq"
	"log"
)

var (
	dbOnce sync.Once
	db *sql.DB
	asyncDB *sqlx.DB
)

// InitializeDatabase initializes the database connections and runs migrations.
func InitializeDatabase(ctx context.Context) error {
	// Ensure that the database is initialized only once
	dbOnce.Do(func() {
		var err error
		databaseURL := os.Getenv("DATABASE_URL")
		if databaseURL == "" {
			databaseURL = "postgres://user:password@localhost:5432/books?sslmode=disable"
		}

		// Initialize the sync database connection
		db, err = sql.Open("postgres", databaseURL)
		if err != nil {
			log.Fatalf("Error opening database: %v", err)
			return
		}
		if err := db.PingContext(ctx); err != nil {
			log.Fatalf("Error pinging database: %v", err)
			return
		}

		// Initialize the async database connection
		asyncDB, err = sqlx.Connect("postgres", databaseURL)
		if err != nil {
			log.Fatalf("Error connecting to async database: %v", err)
			return
		}
		if err := asyncDB.PingContext(ctx); err != nil {
			log.Fatalf("Error pinging async database: %v", err)
			return
		}

		// Run migrations to create the schema
		if err := InitializeMigrations(ctx, db); err != nil && !errors.Is(err, goose.ErrMigrationAppliedAlready) {
			log.Fatalf("Error running migrations: %v", err)
			return err
		}
	})

	return nil
}

// GetDB returns the synchronous database connection.
func GetDB() *sql.DB {
	return db
}

// GetAsyncDB returns the asynchronous database connection.
func GetAsyncDB(ctx context.Context) (*sqlx.DB, error) {
	if asyncDB == nil {
		return nil, fmt.Errorf("async database not initialized")
	}
	return asyncDB, nil
}

// internal/database_migrations.go
package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/pressly/goose/v3"
	_ "github.com/lib/pq"
	"log"
)

//go:embed migrations/*.sql
var migrations embed.FS

func InitializeMigrations(ctx context.Context, db *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("Failed to set dialect: %v", err)
		return err
	}
	return goose.UpDB(db, migrations, "migrations")
}
