package internal

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// DB wraps a *sql.DB connection pool for the Book Catalog API.
//
// This replaces the SQLAlchemy engine/session machinery from app/database.py.
// In Go we do not maintain a separate "async" and "sync" engine: database/sql
// is inherently connection-pooled and every query already accepts a
// context.Context, so a single pool serves both roles cleanly.
//
// The SQLAlchemy dependency-injection generators (get_db / get_sync_db) were
// per-request session-with-rollback helpers. In idiomatic Go, transaction
// scoping is handled explicitly at the call site via WithTx (below) rather
// than through framework-managed generators, so those helpers have no direct
// analogue and are intentionally not reproduced verbatim.
type DB struct {
	pool *sql.DB
}

// schemaDDL is the schema-bootstrap DDL, replacing SQLAlchemy's
// Base.metadata.create_all(). It is derived from the source ORM model
// (app/models.py Book entity) and targets PostgreSQL.
//
// MIGRATION_NOTE: app/models.py was not included in the provided context.
// The columns below reflect the standard Book Catalog model fields for this
// project. Verify against the real app/models.py Book model before deploying;
// adjust column names/types here if they differ. The table name follows the
// SQLAlchemy default of the lowercased class name ("books").
const schemaDDL = `
CREATE TABLE IF NOT EXISTS books (
    id          SERIAL PRIMARY KEY,
    title       TEXT NOT NULL,
    author      TEXT NOT NULL,
    description TEXT,
    published   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);`

// NewDB opens a PostgreSQL connection pool using the supplied DSN, verifies
// connectivity, and returns a ready-to-use *DB.
//
// The DSN replaces the DATABASE_URL / ASYNC_DATABASE_URL environment variables
// from the source; callers should source it from config.Config.DatabaseURL.
func NewDB(ctx context.Context, dsn string) (*DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("database: empty DSN")
	}

	pool, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("database: open pool: %w", err)
	}

	pool.SetMaxOpenConns(25)
	pool.SetMaxIdleConns(25)
	pool.SetConnMaxLifetime(5 * time.Minute)

	// Retry ping with backoff to handle DB container startup race.
	var pingErr error
	for i := 0; i < 10; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		pingErr = pool.PingContext(pingCtx)
		cancel()
		if pingErr == nil {
			break
		}
		select {
		case <-ctx.Done():
			_ = pool.Close()
			return nil, fmt.Errorf("database: ping cancelled: %w", ctx.Err())
		case <-time.After(time.Duration(i+1) * time.Second):
		}
	}
	if pingErr != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("database: ping: %w", pingErr)
	}

	return &DB{pool: pool}, nil
}

// InitSchema creates all tables if they do not already exist. This is the Go
// equivalent of the source's init_db()/Base.metadata.create_all() and should
// be called once at application startup.
func (db *DB) InitSchema(ctx context.Context) error {
	if _, err := db.pool.ExecContext(ctx, schemaDDL); err != nil {
		return fmt.Errorf("database: init schema: %w", err)
	}
	return nil
}

// Pool exposes the underlying *sql.DB for repositories that need direct access.
func (db *DB) Pool() *sql.DB {
	return db.pool
}

// Close releases the underlying connection pool.
func (db *DB) Close() error {
	return db.pool.Close()
}

// WithTx runs fn inside a single transaction, committing on success and rolling
// back on error or panic. This replaces the per-request session-with-rollback
// behavior of the source's get_db / get_sync_db dependency generators, but
// scoped explicitly to a unit of work rather than to an HTTP request.
func (db *DB) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) (err error) {
	tx, err := db.pool.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("database: begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil && rbErr != sql.ErrTxDone {
				err = fmt.Errorf("database: rollback (original: %v): %w", err, rbErr)
			}
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("database: commit: %w", err)
	}
	return nil
}

// Pagination-related constants used to defensively clamp query limits.
//
// MIGRATION_NOTE: The source relied on SQLite semantics where LIMIT -1 means
// "no limit". PostgreSQL rejects a negative LIMIT, so any repository building
// paginated queries must clamp limit/offset via ClampPagination below rather
// than passing user input straight through.
const (
	// DefaultPageLimit is the fallback page size when none is provided.
	DefaultPageLimit = 20
	// MaxPageLimit is the upper bound on a single page size.
	MaxPageLimit = 100
)

// ClampPagination normalizes a requested limit/offset into safe bounds for
// PostgreSQL. A non-positive limit becomes DefaultPageLimit, values above
// MaxPageLimit are capped, and a negative offset becomes 0.
func ClampPagination(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = DefaultPageLimit
	}
	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}