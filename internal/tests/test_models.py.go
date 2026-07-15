// Package tests contains the unit test suite for the Book Catalog models.
//
// This file was migrated from tests/test_models.py, a pytest suite that
// exercised the SQLAlchemy Book model against an in-memory SQLite database.
// The migration maps the Python testing concepts onto idiomatic Go as
// follows:
//
//   - pytest fixture (db_session)  -> newModelTestDB helper invoked per test,
//                                     with teardown registered via t.Cleanup.
//   - class-based grouping         -> individual Test* functions (Go convention).
//   - SQLAlchemy ORM add/commit    -> explicit INSERT via database/sql, since
//                                     the migrated Book model is a plain struct
//                                     with DDL exposed through CreateTableStmts.
//   - IntegrityError on duplicate  -> the INSERT returns a non-nil error which
//                                     is asserted explicitly.
//
// MIGRATION_NOTE: The original test relied on SQLAlchemy's session refresh to
// populate the auto-increment id and the summary default (NULL). In Go we read
// the generated id from the INSERT result (LastInsertId) and re-query the row
// to observe persisted values, which is closer to how database/sql works.
package tests

import (
	"database/sql"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"example.com/bookcatalog/internal"
)

// newModelTestDB creates an in-memory SQLite database with the Book schema
// applied. It registers cleanup so the connection is closed when the test
// finishes. The returned *sql.DB is ready for direct use.
//
// MIGRATION_NOTE: We use a plain *sql.DB rather than the project's NewTestDB
// helper because these are pure model/DDL tests that do not require the HTTP
// server or repository wiring. If you prefer to reuse conftest's NewTestDB,
// swap the body here — the exported test names are unaffected.
func newModelTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if cerr := db.Close(); cerr != nil {
			t.Errorf("close db: %v", cerr)
		}
	})

	for _, stmt := range internal.CreateTableStmts() {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("apply schema: %v", err)
		}
	}
	return db
}

// insertBook persists a Book and returns its generated id. It is the Go
// equivalent of the SQLAlchemy add/commit/refresh cycle used throughout the
// original test suite. summary may be nil to represent the optional field.
func insertBook(db *sql.DB, title, author string, year int, summary *string) (int64, error) {
	const q = `INSERT INTO books (title, author, published_year, summary) VALUES (?, ?, ?, ?)`

	var summaryArg any
	if summary != nil {
		summaryArg = *summary
	}

	res, err := db.Exec(q, title, author, year, summaryArg)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// readBook loads a Book by id so tests can assert on persisted values,
// mirroring SQLAlchemy's session.refresh behaviour.
func readBook(db *sql.DB, id int64) (internal.Book, error) {
	const q = `SELECT id, title, author, published_year, summary FROM books WHERE id = ?`

	var (
		b       internal.Book
		summary sql.NullString
	)
	row := db.QueryRow(q, id)
	if err := row.Scan(&b.ID, &b.Title, &b.Author, &b.PublishedYear, &summary); err != nil {
		return internal.Book{}, err
	}
	if summary.Valid {
		s := summary.String
		b.Summary = &s
	}
	return b, nil
}

// TestCreateBook verifies that a fully populated book is persisted and can be
// read back with all fields intact.
func TestCreateBook(t *testing.T) {
	db := newModelTestDB(t)

	summary := "A test book summary"
	id, err := insertBook(db, "Test Book", "Test Author", 2023, &summary)
	if err != nil {
		t.Fatalf("insert book: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	book, err := readBook(db, id)
	if err != nil {
		t.Fatalf("read book: %v", err)
	}

	if book.Title != "Test Book" {
		t.Errorf("title = %q, want %q", book.Title, "Test Book")
	}
	if book.Author != "Test Author" {
		t.Errorf("author = %q, want %q", book.Author, "Test Author")
	}
	if book.PublishedYear != 2023 {
		t.Errorf("published_year = %d, want %d", book.PublishedYear, 2023)
	}
	if book.Summary == nil || *book.Summary != "A test book summary" {
		t.Errorf("summary = %v, want %q", book.Summary, "A test book summary")
	}
}

// TestCreateBookWithoutSummary verifies that the optional summary field may be
// omitted, in which case it is persisted as NULL.
func TestCreateBookWithoutSummary(t *testing.T) {
	db := newModelTestDB(t)

	id, err := insertBook(db, "Test Book No Summary", "Test Author", 2023, nil)
	if err != nil {
		t.Fatalf("insert book: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	book, err := readBook(db, id)
	if err != nil {
		t.Fatalf("read book: %v", err)
	}

	if book.Title != "Test Book No Summary" {
		t.Errorf("title = %q, want %q", book.Title, "Test Book No Summary")
	}
	if book.Author != "Test Author" {
		t.Errorf("author = %q, want %q", book.Author, "Test Author")
	}
	if book.PublishedYear != 2023 {
		t.Errorf("published_year = %d, want %d", book.PublishedYear, 2023)
	}
	if book.Summary != nil {
		t.Errorf("summary = %v, want nil", *book.Summary)
	}
}

// TestBookRepr verifies the developer-facing representation of a Book, which is
// the Go equivalent of Python's __repr__.
func TestBookRepr(t *testing.T) {
	db := newModelTestDB(t)

	summary := "Test summary"
	id, err := insertBook(db, "Repr Test", "Repr Author", 2023, &summary)
	if err != nil {
		t.Fatalf("insert book: %v", err)
	}

	book, err := readBook(db, id)
	if err != nil {
		t.Fatalf("read book: %v", err)
	}

	want := "<Book(id=" + itoa(book.ID) + ", title='Repr Test', author='Repr Author', year=2023)>"
	if got := book.Repr(); got != want {
		t.Errorf("Repr() = %q, want %q", got, want)
	}
}

// TestBookStr verifies the human-facing representation of a Book, which is the
// Go equivalent of Python's __str__ / fmt.Stringer.
func TestBookStr(t *testing.T) {
	db := newModelTestDB(t)

	if _, err := insertBook(db, "String Test", "String Author", 2023, nil); err != nil {
		t.Fatalf("insert book: %v", err)
	}

	book := internal.Book{
		Title:         "String Test",
		Author:        "String Author",
		PublishedYear: 2023,
	}

	const want = "String Test by String Author (2023)"
	if got := book.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

// TestUniqueConstraintViolation verifies that inserting a second book with the
// same title-author pair fails the unique constraint.
func TestUniqueConstraintViolation(t *testing.T) {
	db := newModelTestDB(t)

	if _, err := insertBook(db, "Duplicate Test", "Duplicate Author", 2023, nil); err != nil {
		t.Fatalf("insert first book: %v", err)
	}

	_, err := insertBook(db, "Duplicate Test", "Duplicate Author", 2024, nil)
	if err == nil {
		t.Fatal("expected unique constraint violation, got nil error")
	}
	// MIGRATION_NOTE: SQLAlchemy raised a generic IntegrityError. The go-sqlite3
	// driver returns a driver-specific error; we assert only that an error
	// occurred, matching the original test's broad pytest.raises(Exception).
	var _ = errors.Unwrap // keep errors import meaningful for future refinement
}

// TestBooksWithSameTitleDifferentAuthors verifies that the unique constraint is
// on the (title, author) pair, so a shared title with distinct authors is fine.
func TestBooksWithSameTitleDifferentAuthors(t *testing.T) {
	db := newModelTestDB(t)

	id1, err := insertBook(db, "Common Title", "Author One", 2023, nil)
	if err != nil {
		t.Fatalf("insert first book: %v", err)
	}
	id2, err := insertBook(db, "Common Title", "Author Two", 2023, nil)
	if err != nil {
		t.Fatalf("insert second book: %v", err)
	}

	if id1 == 0 || id2 == 0 {
		t.Fatal("expected non-zero ids for both books")
	}
	if id1 == id2 {
		t.Errorf("expected distinct ids, both are %d", id1)
	}
}

// TestBooksWithSameAuthorDifferentTitles verifies that a shared author with
// distinct titles is permitted by the unique constraint.
func TestBooksWithSameAuthorDifferentTitles(t *testing.T) {
	db := newModelTestDB(t)

	id1, err := insertBook(db, "First Book", "Prolific Author", 2023, nil)
	if err != nil {
		t.Fatalf("insert first book: %v", err)
	}
	id2, err := insertBook(db, "Second Book", "Prolific Author", 2024, nil)
	if err != nil {
		t.Fatalf("insert second book: %v", err)
	}

	if id1 == 0 || id2 == 0 {
		t.Fatal("expected non-zero ids for both books")
	}
	if id1 == id2 {
		t.Errorf("expected distinct ids, both are %d", id1)
	}
}

// itoa converts an int64 id to its decimal string form. It exists so the Repr
// assertion mirrors the exact formatting of the migrated Book.Repr method
// without pulling in fmt at the call site.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
