// Package catalog provides database configuration and session management
// for the Book Catalog API.
//
// This file was migrated from app/database.py, which configured SQLAlchemy
// async and sync engines plus FastAPI dependency-injection session generators.
//
// In idiomatic Go there is no async/sync engine split: database/sql manages a
// connection pool that is safe for concurrent use by multiple goroutines, and
// context.Context handles cancellation/deadlines instead of Python's asyncio.
// Therefore the two parallel engines (async aiosqlite + sync) collapse into a
// single *sql.DB. The FastAPI Depends() generators (get_db/get_sync_db) are
// replaced by passing the *sql.DB (or a repository built on it) explicitly
// through constructors and request handlers.
package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	// MIGRATION_NOTE: The original used SQLite via aiosqlite. This blank import
	// registers the pure-Go SQLite driver. Swap for the appropriate driver
	// (e.g. github.com/lib/pq or github.com/jackc/pgx) when moving to Postgres.
	_ "modernc.org/sqlite"
)

// Default connection strings mirror the Python defaults, which pointed at a
// local SQLite file (books.db). Real deployments should override via the
// DATABASE_URL environment variable.
const (
	defaultDatabaseURL = "file:./books.db"
	defaultDriverName  = "sqlite"
)

// Config holds database configuration resolved from the environment.
//
// MIGRATION_NOTE: The Python file exposed both DATABASE_URL and
// ASYNC_DATABASE_URL because SQLAlchemy needed separate sync/async engines.
// Go's database/sql pool covers both use cases, so only a single DSN is
// required. If a deployment set only ASYNC_DATABASE_URL, translate it to a
// standard driver DSN here.
type Config struct {
	// DriverName is the database/sql driver name (e.g. "sqlite", "postgres").
	DriverName string
	// DataSourceName is the driver-specific connection string (DSN).
	DataSourceName string
	// Echo enables SQL statement logging. The Python code hardcoded echo=True;
	// this is now controlled via the DB_ECHO environment variable and defaults
	// to false, which is the safe production default.
	Echo bool
}

// LoadConfig builds a Config from environment variables, applying the same
// SQLite-oriented defaults as the original Python module.
func LoadConfig() Config {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = defaultDatabaseURL
	}

	driver := os.Getenv("DATABASE_DRIVER")
	if driver == "" {
		driver = defaultDriverName
	}

	return Config{
		DriverName:     driver,
		DataSourceName: dsn,
		Echo:           strings.EqualFold(os.Getenv("DB_ECHO"), "true"),
	}
}

// DB wraps the standard *sql.DB connection pool.
//
// In the Python original, get_db/get_sync_db were FastAPI dependencies that
// yielded a per-request session and handled rollback/close. With database/sql,
// the pool is long-lived and shared; transaction scoping is done explicitly via
// WithTx below rather than per-request session objects.
type DB struct {
	*sql.DB
}

// NewDB opens a database connection pool using the supplied configuration and
// verifies connectivity with a ping. It replaces both create_async_engine and
// create_engine from the Python module.
//
// MIGRATION_NOTE: SQLAlchemy's connect_args={"check_same_thread": False} was a
// SQLite/threading workaround. database/sql manages concurrency safely, so no
// equivalent flag is needed. For file-based SQLite you may still want to limit
// the pool to a single connection to avoid "database is locked" errors.
func NewDB(ctx context.Context, cfg Config) (*DB, error) {
	sqlDB, err := sql.Open(cfg.DriverName, cfg.DataSourceName)
	if err != nil {
		return nil, fmt.Errorf("opening database %q: %w", cfg.DriverName, err)
	}

	// SQLite file databases behave best with a single writer connection.
	if cfg.DriverName == "sqlite" {
		sqlDB.SetMaxOpenConns(1)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{DB: sqlDB}, nil
}

// Close releases the underlying connection pool.
func (db *DB) Close() error {
	if err := db.DB.Close(); err != nil {
		return fmt.Errorf("closing database: %w", err)
	}
	return nil
}

// schemaStatements holds the DDL used to create the application's tables.
//
// MIGRATION_NOTE: The Python code called Base.metadata.create_all, where Base
// was the SQLAlchemy declarative base defined in app/models. Go has no ORM
// metadata registry by default, so the schema must be expressed explicitly
// here (or, preferably, managed by a migration tool such as goose or
// golang-migrate). This DDL is a placeholder derived from the Book domain and
// MUST be reviewed against the actual models package once it is migrated.
var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		author TEXT NOT NULL,
		published_year INTEGER
	);`,
}

// InitDB creates all tables if they do not already exist. It is the Go
// equivalent of the Python init_db coroutine.
//
// MIGRATION_NOTE: The Python init_db was declared async but performed a
// synchronous create_all on the sync engine — the async keyword was
// misleading. Here it is a plain method that accepts a context for
// cancellation, which is the idiomatic replacement.
func (db *DB) InitDB(ctx context.Context) error {
	for _, stmt := range schemaStatements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("creating schema: %w", err)
		}
	}
	return nil
}

// WithTx runs fn inside a database transaction, committing on success and
// rolling back on error or panic.
//
// This consolidates the rollback/close semantics that both get_db and
// get_sync_db implemented in Python. Callers scope a transaction explicitly
// instead of relying on FastAPI's Depends() session lifecycle.
//
// The expire_on_commit=False behaviour from the async SQLAlchemy session has no
// Go analogue: database/sql returns plain values, not managed entities, so
// objects read within the transaction remain valid after commit without any
// special flag.
func (db *DB) WithTx(ctx context.Context, fn func(*sql.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err = fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}
