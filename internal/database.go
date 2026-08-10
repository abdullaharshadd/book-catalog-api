package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	// MIGRATION_NOTE: PostgreSQL driver. The Python source defaulted to SQLite;
	// per the target-dialect directive we standardize on PostgreSQL (lib/pq).
	_ "github.com/lib/pq"
)

// DB wraps a sqlx connection pool for the Book Catalog service.
//
// MIGRATION_NOTE: The Python source configured two SQLAlchemy engines — an
// async engine (asyncpg/aiosqlite) and a sync engine — plus two sessionmaker
// factories and FastAPI-style dependency generators (get_db / get_sync_db).
// Go's database/sql (and sqlx on top of it) is already a concurrency-safe
// connection pool with context-based cancellation, so there is no meaningful
// async/sync split to preserve. A single *sqlx.DB replaces both engines and
// both session factories. The FastAPI request-scoped session generators are
// replaced by passing context.Context into each query — callers obtain the
// shared pool via NewDB and thread ctx through repository methods.
type DB struct {
	*sqlx.DB
}

// NewDB opens a PostgreSQL connection pool using the provided database URL,
// verifies connectivity with a ping, and returns a ready-to-use *DB.
//
// MIGRATION_NOTE: SQLAlchemy's echo=True (SQL statement logging) has no direct
// database/sql equivalent; enable it via a logging driver wrapper or a
// zerolog-instrumented sqlx wrapper if query logging is required in dev.
func NewDB(ctx context.Context, databaseURL string) (*DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database url is empty")
	}

	db, err := sqlx.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{DB: db}, nil
}

// Close releases all resources held by the connection pool.
func (d *DB) Close() error {
	if d == nil || d.DB == nil {
		return nil
	}
	if err := d.DB.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}
	return nil
}

// InitSchema creates the required database tables if they do not already exist.
//
// MIGRATION_NOTE: The Python source called Base.metadata.create_all(...) to
// materialize SQLAlchemy-declared models. There is no ORM metadata in this Go
// migration, so the schema is expressed as explicit DDL here. This uses
// PostgreSQL GENERATED ALWAYS AS IDENTITY (not MySQL AUTO_INCREMENT / SQLite
// AUTOINCREMENT) for the primary key. If the models package (app/models.py)
// defines additional columns, this DDL must be reconciled with it during
// manual review.
func (d *DB) InitSchema(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS books (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title       TEXT    NOT NULL,
    author      TEXT    NOT NULL,
    isbn        TEXT,
    published   DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);`

	if _, err := d.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}
