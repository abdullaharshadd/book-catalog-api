package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgconn"
	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib" // register the pgx database/sql driver
	"github.com/rs/zerolog/log"
)

// MIGRATION_NOTE: The Python source used SQLAlchemy with both an async and a
// sync engine (SQLite by default). The migration target is PostgreSQL, so the
// dual async/sync engine split from SQLAlchemy does not carry over: Go's
// database/sql (and sqlx on top of it) already manages a connection pool and
// executes queries synchronously per goroutine, with concurrency handled by
// the runtime rather than by a separate async engine. We therefore expose a
// single *sqlx.DB.
//
// MIGRATION_NOTE: SQLAlchemy's Base.metadata.create_all() auto-created tables
// from ORM models. In idiomatic Go schema management is done via explicit
// migration files (e.g. golang-migrate, goose, or checked-in SQL). InitDB
// below is intentionally a connectivity/ping check rather than a schema
// generator; wire in your migration tool of choice where noted.
//
// MIGRATION_NOTE: FastAPI's get_db / get_sync_db generator-based dependency
// injection (yield session, rollback on error, close in finally) has no direct
// Go equivalent. The idiomatic replacement is:
//   - hold a single long-lived *sqlx.DB (the pool) for the process lifetime;
//   - pass a context.Context to every query for cancellation/timeout;
//   - open an explicit *sqlx.Tx only when a request needs transactional
//     rollback-on-error semantics (see WithTx).

// DB wraps a sqlx connection pool for the application.
type DB struct {
	*sqlx.DB
}

// NewDB opens a PostgreSQL connection pool using the given DSN and verifies
// connectivity with a ping. The caller is responsible for calling Close.
func NewDB(ctx context.Context, databaseURL string) (*DB, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("database: empty database URL")
	}

	db, err := sqlx.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("database: open connection: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database: ping: %w", err)
	}

	log.Info().Msg("database connection established")
	return &DB{DB: db}, nil
}

// InitDB verifies the database is reachable and ready to serve queries.
//
// MIGRATION_NOTE: This replaces SQLAlchemy's init_db()/create_all(). It does
// NOT create tables; run your migration tool (golang-migrate/goose) here or as
// a separate deploy step.
func (d *DB) InitDB(ctx context.Context) error {
	if err := d.PingContext(ctx); err != nil {
		return fmt.Errorf("database: init: %w", err)
	}
	return nil
}

// Close releases all pooled connections.
func (d *DB) Close() error {
	if err := d.DB.Close(); err != nil {
		return fmt.Errorf("database: close: %w", err)
	}
	return nil
}

// WithTx runs fn inside a database transaction, committing on success and
// rolling back if fn returns an error or panics. This preserves the
// rollback-on-error / cleanup-on-finish semantics of the Python get_db
// dependency generator.
func (d *DB) WithTx(ctx context.Context, fn func(tx *sqlx.Tx) error) (err error) {
	tx, err := d.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("database: begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				err = fmt.Errorf("database: rollback (original error: %v): %w", err, rbErr)
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

// uniqueViolationCode is the PostgreSQL SQLSTATE for a unique_violation.
const uniqueViolationCode = "23505"

// IsUniqueViolation reports whether err (as returned from an ExecContext /
// QueryContext call via *sqlx.DB) is a PostgreSQL unique-constraint violation.
//
// MIGRATION_NOTE: The discovery test must exercise this against the same
// production call path (*sqlx.DB.ExecContext), because errors bubbling up
// through sqlx are the driver's *pgconn.PgError which we unwrap with errors.As.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == uniqueViolationCode
	}
	return false
}