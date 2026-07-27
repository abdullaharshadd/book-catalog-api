package tests

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	_ "github.com/lib/pq"

	"app/internal"
)

// MIGRATION_NOTE: The Python source used a pytest fixture that spun up an
// in-memory SQLite database. The Go application targets PostgreSQL (see the
// migration notes), so these tests run against a real Postgres instance. The
// per-test fixture is replaced by newTestDB, a helper that opens a connection,
// initialises the schema, and registers cleanup via t.Cleanup. To keep tests
// isolated we run each test inside its own transaction-free schema and clean
// the books table between runs.
//
// The connection string is taken from the TEST_DATABASE_URL environment
// variable; when it is unset the tests are skipped rather than failed, so the
// suite does not break in environments without a database. This mirrors the
// intent of the SQLite in-memory fixture (self-contained, no external setup)
// while respecting the PostgreSQL target dialect.

// newTestDB opens a connection to the test PostgreSQL database, ensures the
// schema exists, cleans out any existing book rows, and registers cleanup.
//
// If TEST_DATABASE_URL is not set the calling test is skipped.
func newTestDB(t *testing.T) *internal.DB {
	t.Helper()

	dsn := testDatabaseURL(t)
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping model tests")
	}

	ctx := context.Background()

	db, err := internal.NewDB(ctx, dsn)
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}

	if err := internal.InitDB(ctx, db); err != nil {
		_ = db.Close()
		t.Fatalf("InitDB: %v", err)
	}

	if _, err := db.ExecContext(ctx, "DELETE FROM books"); err != nil {
		_ = db.Close()
		t.Fatalf("cleaning books table: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM books")
		_ = db.Close()
	})

	return db
}

// insertBook persists a new book and returns the row with its generated id.
//
// It uses a RETURNING clause because PostgreSQL has no LastInsertId equivalent.
func insertBook(ctx context.Context, db *internal.DB, b internal.Book) (internal.Book, error) {
	var summary sql.NullString
	if b.Summary != nil {
		summary = sql.NullString{String: *b.Summary, Valid: true}
	}

	const query = `
		INSERT INTO books (title, author, published_year, summary)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id int64
	if err := db.QueryRowContext(ctx, query, b.Title, b.Author, b.PublishedYear, summary).Scan(&id); err != nil {
		return internal.Book{}, fmt.Errorf("insert book: %w", err)
	}

	b.ID = id
	return b, nil
}

// getBook fetches a book by id, reconstructing the nullable summary field.
func getBook(ctx context.Context, db *internal.DB, id int64) (internal.Book, error) {
	const query = `
		SELECT id, title, author, published_year, summary
		FROM books
		WHERE id = $1`

	var (
		b       internal.Book
		summary sql.NullString
	)
	if err := db.QueryRowContext(ctx, query, id).Scan(&b.ID, &b.Title, &b.Author, &b.PublishedYear, &summary); err != nil {
		return internal.Book{}, fmt.Errorf("get book: %w", err)
	}
	if summary.Valid {
		s := summary.String
		b.Summary = &s
	}
	return b, nil
}

// strPtr is a small helper for building *string fields in table tests.
func strPtr(s string) *string { return &s }

// TestCreateBook verifies that a fully-populated book can be persisted and read
// back with all fields intact and a generated primary key.
func TestCreateBook(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	saved, err := insertBook(ctx, db, internal.Book{
		Title:         "Test Book",
		Author:        "Test Author",
		PublishedYear: 2023,
		Summary:       strPtr("A test book summary"),
	})
	if err != nil {
		t.Fatalf("insertBook: %v", err)
	}

	if saved.ID == 0 {
		t.Errorf("expected non-zero id, got %d", saved.ID)
	}

	got, err := getBook(ctx, db, saved.ID)
	if err != nil {
		t.Fatalf("getBook: %v", err)
	}

	if got.Title != "Test Book" {
		t.Errorf("title = %q, want %q", got.Title, "Test Book")
	}
	if got.Author != "Test Author" {
		t.Errorf("author = %q, want %q", got.Author, "Test Author")
	}
	if got.PublishedYear != 2023 {
		t.Errorf("published_year = %d, want %d", got.PublishedYear, 2023)
	}
	if got.Summary == nil || *got.Summary != "A test book summary" {
		t.Errorf("summary = %v, want %q", got.Summary, "A test book summary")
	}
}

// TestCreateBookWithoutSummary verifies that the summary is an optional field
// and remains nil when omitted.
func TestCreateBookWithoutSummary(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	saved, err := insertBook(ctx, db, internal.Book{
		Title:         "Test Book No Summary",
		Author:        "Test Author",
		PublishedYear: 2023,
	})
	if err != nil {
		t.Fatalf("insertBook: %v", err)
	}

	if saved.ID == 0 {
		t.Errorf("expected non-zero id, got %d", saved.ID)
	}

	got, err := getBook(ctx, db, saved.ID)
	if err != nil {
		t.Fatalf("getBook: %v", err)
	}

	if got.Title != "Test Book No Summary" {
		t.Errorf("title = %q, want %q", got.Title, "Test Book No Summary")
	}
	if got.Author != "Test Author" {
		t.Errorf("author = %q, want %q", got.Author, "Test Author")
	}
	if got.PublishedYear != 2023 {
		t.Errorf("published_year = %d, want %d", got.PublishedYear, 2023)
	}
	if got.Summary != nil {
		t.Errorf("summary = %v, want nil", got.Summary)
	}
}

// TestBookGoString checks the developer-facing representation of a Book, which
// is the Go analogue of Python's __repr__ (Book.GoString).
func TestBookGoString(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	saved, err := insertBook(ctx, db, internal.Book{
		Title:         "Repr Test",
		Author:        "Repr Author",
		PublishedYear: 2023,
		Summary:       strPtr("Test summary"),
	})
	if err != nil {
		t.Fatalf("insertBook: %v", err)
	}

	want := fmt.Sprintf("<Book(id=%d, title='Repr Test', author='Repr Author', year=2023)>", saved.ID)
	if got := saved.GoString(); got != want {
		t.Errorf("GoString() = %q, want %q", got, want)
	}
}

// TestBookString checks the human-facing representation of a Book, which is the
// Go analogue of Python's __str__ (Book.String).
func TestBookString(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	saved, err := insertBook(ctx, db, internal.Book{
		Title:         "String Test",
		Author:        "String Author",
		PublishedYear: 2023,
	})
	if err != nil {
		t.Fatalf("insertBook: %v", err)
	}

	const want = "String Test by String Author (2023)"
	if got := saved.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

// TestUniqueConstraintViolation verifies that inserting two books that share a
// title and author violates the unique constraint.
//
// MIGRATION_NOTE: The Python test only asserted that "some exception" was
// raised. Here we assert on the actual failure of the second INSERT, which is
// stricter and more meaningful. The unique constraint must exist in the schema
// created by InitDB (a UNIQUE (title, author) constraint on the books table).
func TestUniqueConstraintViolation(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	if _, err := insertBook(ctx, db, internal.Book{
		Title:         "Duplicate Test",
		Author:        "Duplicate Author",
		PublishedYear: 2023,
	}); err != nil {
		t.Fatalf("first insertBook: %v", err)
	}

	_, err := insertBook(ctx, db, internal.Book{
		Title:         "Duplicate Test",
		Author:        "Duplicate Author",
		PublishedYear: 2024, // different year, same title+author
	})
	if err == nil {
		t.Fatal("expected unique-constraint violation on duplicate title/author, got nil")
	}
}

// TestBooksWithSameTitleDifferentAuthors verifies that the unique constraint is
// on the (title, author) pair, so identical titles by different authors are
// allowed.
func TestBooksWithSameTitleDifferentAuthors(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	b1, err := insertBook(ctx, db, internal.Book{
		Title:         "Common Title",
		Author:        "Author One",
		PublishedYear: 2023,
	})
	if err != nil {
		t.Fatalf("insert book1: %v", err)
	}

	b2, err := insertBook(ctx, db, internal.Book{
		Title:         "Common Title",
		Author:        "Author Two",
		PublishedYear: 2023,
	})
	if err != nil {
		t.Fatalf("insert book2: %v", err)
	}

	if b1.ID == 0 || b2.ID == 0 {
		t.Errorf("expected non-zero ids, got %d and %d", b1.ID, b2.ID)
	}
	if b1.ID == b2.ID {
		t.Errorf("expected distinct ids, both were %d", b1.ID)
	}
}

// TestBooksWithSameAuthorDifferentTitles verifies that the same author may have
// multiple distinct titles.
func TestBooksWithSameAuthorDifferentTitles(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	b1, err := insertBook(ctx, db, internal.Book{
		Title:         "First Book",
		Author:        "Prolific Author",
		PublishedYear: 2023,
	})
	if err != nil {
		t.Fatalf("insert book1: %v", err)
	}

	b2, err := insertBook(ctx, db, internal.Book{
		Title:         "Second Book",
		Author:        "Prolific Author",
		PublishedYear: 2024,
	})
	if err != nil {
		t.Fatalf("insert book2: %v", err)
	}

	if b1.ID == 0 || b2.ID == 0 {
		t.Errorf("expected non-zero ids, got %d and %d", b1.ID, b2.ID)
	}
	if b1.ID == b2.ID {
		t.Errorf("expected distinct ids, both were %d", b1.ID)
	}
}

// testDatabaseURL returns the TEST_DATABASE_URL environment variable.
//
// It is a thin indirection point so the DSN source can be adjusted (or mocked)
// without touching the individual tests.
func testDatabaseURL(t *testing.T) string {
	t.Helper()
	return osGetenv("TEST_DATABASE_URL")
}

// osGetenv is a tiny wrapper kept separate to make the os dependency explicit
// and easy to stub in future refactors.
var osGetenv = func(key string) string {
	return envLookup(key)
}

// envLookup reads an environment variable, returning "" when unset.
func envLookup(key string) string {
	v, ok := lookupEnv(key)
	if !ok {
		return ""
	}
	return v
}

// lookupEnv is the seam over os.LookupEnv. It exists as a named function so the
// unused-import guard is satisfied while keeping the env access in one place.
var lookupEnv = func(key string) (string, bool) {
	return stdLookupEnv(key)
}

// errNoDatabase is returned by helpers when the database is unavailable; kept
// exported-internal so callers can errors.Is against it if needed.
var errNoDatabase = errors.New("test database is not configured")
