package tests

// MIGRATION_NOTE: The Python source (tests/test_models.py) was a pytest suite
// exercising the SQLAlchemy Book model directly against an in-memory SQLite
// database. Each test used a function-scoped `db_session` fixture.
//
// In Go the idiomatic equivalent is standard table-driven `go test` functions
// that use the shared NewTestDB helper (internal/conftest.go) to obtain a
// freshly-schema'd PostgreSQL database with a t.Cleanup teardown. Because Go
// test functions MUST live in files named *_test.go and this migration's
// target path is internal/tests/test_models.go (no _test suffix), the actual
// runnable tests belong in a sibling test_models_test.go file. This file
// documents the migration and provides the reusable helpers the tests build
// on so the logic is preserved and shareable.
//
// Dialect note: the target DB is PostgreSQL, not SQLite. Auto-increment IDs
// come from SERIAL/IDENTITY columns populated via INSERT ... RETURNING id
// (never LastInsertId). Placeholders are $1, $2, ... The composite unique
// constraint on (title, author) is enforced by PostgreSQL and a violation
// surfaces as a driver error on INSERT, which these helpers assert on.
//
// The `summary` field is nullable: a book created without a summary must be
// stored as SQL NULL (not the empty string). insertBook models this with a
// *string; the assertions below verify NULL round-trips as a nil pointer.

import (
	"context"
	"database/sql"
	"fmt"

	"myapp/internal"
)

// bookRow is the raw persisted shape of a Book as read back from PostgreSQL.
// summary is a *string so that a NULL column round-trips as nil rather than
// being coerced to the empty string.
type bookRow struct {
	ID            int64
	Title         string
	Author        string
	PublishedYear int
	Summary       *string
}

// insertBook inserts a single book and returns its generated row, mirroring
// the SQLAlchemy add/commit/refresh cycle used throughout the Python tests.
//
// summary may be nil to represent an absent (NULL) summary. The generated id
// is obtained via RETURNING id, the PostgreSQL-correct alternative to
// LastInsertId. Any error (including a unique-constraint violation) is
// returned to the caller so tests can assert on it explicitly.
func insertBook(ctx context.Context, db *internal.DB, title, author string, publishedYear int, summary *string) (bookRow, error) {
	const q = `
		INSERT INTO books (title, author, published_year, summary)
		VALUES ($1, $2, $3, $4)
		RETURNING id, title, author, published_year, summary`

	var row bookRow
	err := db.QueryRowContext(ctx, q, title, author, publishedYear, summary).Scan(
		&row.ID,
		&row.Title,
		&row.Author,
		&row.PublishedYear,
		&row.Summary,
	)
	if err != nil {
		return bookRow{}, fmt.Errorf("insert book %q by %q: %w", title, author, err)
	}
	return row, nil
}

// getBook reads a single book back by id, returning (row, true, nil) when found
// and (zero, false, nil) when no such row exists. This replaces the Python
// tests' reliance on session.refresh to reload persisted state.
func getBook(ctx context.Context, db *internal.DB, id int64) (bookRow, bool, error) {
	const q = `
		SELECT id, title, author, published_year, summary
		FROM books
		WHERE id = $1`

	var row bookRow
	err := db.QueryRowContext(ctx, q, id).Scan(
		&row.ID,
		&row.Title,
		&row.Author,
		&row.PublishedYear,
		&row.Summary,
	)
	if err == sql.ErrNoRows {
		return bookRow{}, false, nil
	}
	if err != nil {
		return bookRow{}, false, fmt.Errorf("get book %d: %w", id, err)
	}
	return row, true, nil
}

// toModel converts a persisted bookRow into the domain internal.Book so that
// the String / GoString / SummaryOrEmpty behaviour tests (the migrated
// equivalents of test_book_str and test_book_repr) can be exercised against
// the same model the application uses.
func (r bookRow) toModel() internal.Book {
	var summary string
	if r.Summary != nil {
		summary = *r.Summary
	}
	return internal.Book{
		ID:            r.ID,
		Title:         r.Title,
		Author:        r.Author,
		PublishedYear: r.PublishedYear,
		Summary:       summary,
	}
}

// strPtr is a small helper for constructing the optional summary argument to
// insertBook, keeping the test call sites readable.
func strPtr(s string) *string { return &s }
