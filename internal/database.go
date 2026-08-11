package internal

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// MIGRATION_NOTE: The source (app/database.py) configured BOTH an async and a
// sync SQLAlchemy engine against SQLite, plus FastAPI-style dependency-injection
// generators (get_db / get_sync_db). In idiomatic Go there is no async/sync
// engine split and no generator-based request-scoped session: a single
// *sqlx.DB is a connection pool that is safe for concurrent use, and callers
// pass context.Context per operation. The target database is PostgreSQL
// (per project requirements), so the SQLite-specific check_same_thread /
// busy_timeout / SetMaxOpenConns(1) :memory: workarounds do NOT apply here
// and are intentionally dropped.
//
// MIGRATION_NOTE: The source model file (app/models.py) was not provided in
// context. The schema below is derived from the source project's Book model
// using its ORM's own snake_case naming convention (table "books"). Review
// the real app/models.py and reconcile any column differences before running
// in production.

// DB wraps the application's database connection pool. It is the Go analog of
// the SQLAlchemy engine/session-factory pair from app/database.py.
type DB struct {
	*sqlx.DB
}

// NewDB opens a PostgreSQL connection pool using the provided DSN and verifies
// connectivity with a Ping. It replaces the module-level create_engine /
// create_async_engine calls from the source.
func NewDB(ctx context.Context, dsn string) (*DB, error) {
	sqlDB, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{DB: sqlDB}, nil
}

// InitSchema creates all required tables if they do not already exist. It is
// the Go equivalent of the source's init_db() / Base.metadata.create_all().
//
// The DDL is executed inline at startup so a freshly-provisioned database is
// usable without an external migration step.
func (db *DB) InitSchema(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS books (
	id          SERIAL PRIMARY KEY,
	title       TEXT NOT NULL,
	author      TEXT NOT NULL,
	description TEXT,
	published_year INTEGER,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);`

	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}

// WithTx runs fn inside a database transaction, committing on success and
// rolling back on error. This replaces the try/rollback/close pattern that the
// source's get_db / get_sync_db generators implemented manually per request.
func (db *DB) WithTx(ctx context.Context, fn func(tx *sqlx.Tx) error) (err error) {
	tx, err := db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
			}
			return
		}
		if cErr := tx.Commit(); cErr != nil {
			err = fmt.Errorf("commit tx: %w", cErr)
		}
	}()

	return fn(tx)
}

// Close releases the underlying connection pool.
func (db *DB) Close() error {
	return db.DB.Close()
}