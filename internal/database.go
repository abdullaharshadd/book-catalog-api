package internal

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	// MIGRATION_NOTE: pgx's stdlib driver registers the "pgx" database/sql
	// driver. Import for side effects. Swap for lib/pq if preferred.
	_ "github.com/jackc/pgx/v5/stdlib"
)

// MIGRATION_NOTE: The Python source configured TWO SQLAlchemy engines (an
// async engine + a sync engine) plus two sessionmaker factories. That split
// exists purely because Python needs separate sync/async I/O stacks. Go's
// database/sql is inherently concurrency-safe and uses context for
// cancellation, so a single *sqlx.DB (a connection pool) replaces BOTH
// engines and BOTH session factories. FastAPI's get_db / get_sync_db
// dependency-injection generators are replaced by passing the *DB (or a
// context-scoped *sql.Tx) explicitly to callers.
//
// MIGRATION_NOTE: The source targeted SQLite. Per migration directive the
// target dialect is PostgreSQL, so queries elsewhere use $1,$2 placeholders
// and RETURNING id. The SQLite-specific connect_args (check_same_thread) have
// no PostgreSQL equivalent and were dropped.

// DB wraps a PostgreSQL connection pool used by the application.
type DB struct {
	*sqlx.DB
}

// NewDB opens a PostgreSQL connection pool using the provided DSN, verifies
// connectivity with a ping, and returns a ready-to-use *DB.
//
// The caller is responsible for calling Close when the pool is no longer
// needed (typically during graceful shutdown).
func NewDB(ctx context.Context, dsn string) (*DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("database: empty DSN")
	}

	pool, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("database: open pool: %w", err)
	}

	pool.SetMaxOpenConns(25)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.PingContext(pingCtx); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("database: ping: %w", err)
	}

	return &DB{DB: pool}, nil
}

// Close releases all resources held by the connection pool.
func (db *DB) Close() error {
	if db == nil || db.DB == nil {
		return nil
	}
	return db.DB.Close()
}

// booksSchema is the DDL used by InitDB to ensure the books table exists.
//
// MIGRATION_NOTE: The Python source used Base.metadata.create_all to derive
// the schema from ORM models. Go has no metaclass-driven ORM here, so the
// schema is expressed explicitly. In production prefer a real migration tool
// (golang-migrate, goose, atlas) over create-if-not-exists; this mirrors the
// original's convenience behaviour only. Uses GENERATED ALWAYS AS IDENTITY
// for the PK per PostgreSQL conventions (not AUTO_INCREMENT/SERIAL).
const booksSchema = `
CREATE TABLE IF NOT EXISTS books (
	id             INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	title          TEXT NOT NULL,
	author         TEXT NOT NULL,
	published_year INTEGER,
	summary        TEXT
);`

// InitDB initializes the database by creating the required tables if they do
// not already exist. It is the Go equivalent of the source's init_db().
func InitDB(ctx context.Context, db *DB) error {
	if db == nil {
		return fmt.Errorf("database: nil DB")
	}
	if _, err := db.ExecContext(ctx, booksSchema); err != nil {
		return fmt.Errorf("database: init schema: %w", err)
	}
	return nil
}

// WithTx runs fn inside a database transaction, committing on success and
// rolling back if fn returns an error or panics.
//
// MIGRATION_NOTE: This replaces the try/except-rollback/finally-close pattern
// in the source's get_db and get_sync_db generators. Instead of a
// per-request session leased by a DI framework, callers wrap the unit of
// work explicitly. Connection acquisition and release are handled by the
// database/sql pool automatically.
func WithTx(ctx context.Context, db *DB, fn func(tx *sqlx.Tx) error) (err error) {
	if db == nil {
		return fmt.Errorf("database: nil DB")
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("database: begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = fmt.Errorf("database: rollback failed: %v (original error: %w)", rbErr, err)
			}
			return
		}
		if cErr := tx.Commit(); cErr != nil {
			err = fmt.Errorf("database: commit: %w", cErr)
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}
	return nil
}

// Compile-time assertion that *DB satisfies the minimal executor contract
// expected by repositories.
var _ interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
} = (*DB)(nil)
