package tests

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"bookcatalog/internal"
)

// MIGRATION_NOTE: The Python source (tests/test_models.py) was a pytest suite
// that exercised the SQLAlchemy Book model against an in-memory SQLite
// database. The db_session fixture is replaced by internal.NewTestDB, which
// provisions an isolated PostgreSQL database (the target dialect) for each
// test. SQLAlchemy's session.add/commit/refresh round-trip is replaced by an
// explicit INSERT ... RETURNING id executed via the migrated internal.DB.
//
// MIGRATION_NOTE: The Python model carried the persistence behaviour directly
// (session.add on a Book instance). Go's internal.Book is a plain struct; the
// SQL lives here in the test as a small insertBook helper rather than on the
// model. The optional `summary` field maps to sql.NullString: a nil summary in
// Python becomes NullString{Valid:false}, a present summary becomes
// NullString{Valid:true}.
//
// MIGRATION_NOTE: The Python tests used repr(book) and str(book). These map to
// the already-migrated internal.Book.GoString() (repr) and
// internal.Book.String() (str) methods respectively.

// insertBook persists a Book using PostgreSQL's RETURNING clause to obtain the
// generated identifier, mirroring SQLAlchemy's session.add + commit + refresh.
// It returns the assigned id, or an error (including unique-constraint
// violations, surfaced via internal.IsUniqueViolation).
func insertBook(ctx context.Context, db *internal.DB, b internal.Book) (int64, error) {
	const query = `INSERT INTO books (title, author, published_year, summary)
	               VALUES ($1, $2, $3, $4)
	               RETURNING id`
	var id int64
	err := db.QueryRowxContext(ctx, query,
		b.Title, b.Author, b.PublishedYear, b.Summary,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert book: %w", err)
	}
	return id, nil
}

// getBook loads a single Book by id, returning (Book, false, nil) when no row
// exists. This replaces SQLAlchemy's session.refresh round-trip.
func getBook(ctx context.Context, db *internal.DB, id int64) (internal.Book, bool, error) {
	const query = `SELECT id, title, author, published_year, summary
	               FROM books WHERE id = $1`
	var b internal.Book
	err := db.QueryRowxContext(ctx, query, id).StructScan(&b)
	if errors.Is(err, sql.ErrNoRows) {
		return internal.Book{}, false, nil
	}
	if err != nil {
		return internal.Book{}, false, fmt.Errorf("get book: %w", err)
	}
	return b, true, nil
}

// TestCreateBook verifies that a book with all fields (including summary) is
// persisted and read back correctly.
func TestCreateBook(t *testing.T) {
	ctx := context.Background()
	db := internal.NewTestDB(t)

	id, err := insertBook(ctx, db, internal.Book{
		Title:         "Test Book",
		Author:        "Test Author",
		PublishedYear: 2023,
		Summary:       sql.NullString{String: "A test book summary", Valid: true},
	})
	if err != nil {
		t.Fatalf("insertBook: %v", err)
	}
	if id == 0 {
		t.Fatal("expected a non-zero generated id")
	}

	got, ok, err := getBook(ctx, db, id)
	if err != nil {
		t.Fatalf("getBook: %v", err)
	}
	if !ok {
		t.Fatal("expected book to exist")
	}

	if got.Title != "Test Book" {
		t.Errorf("title: got %q, want %q", got.Title, "Test Book")
	}
	if got.Author != "Test Author" {
		t.Errorf("author: got %q, want %q", got.Author, "Test Author")
	}
	if got.PublishedYear != 2023 {
		t.Errorf("published_year: got %d, want %d", got.PublishedYear, 2023)
	}
	if !got.Summary.Valid || got.Summary.String != "A test book summary" {
		t.Errorf("summary: got %+v, want valid %q", got.Summary, "A test book summary")
	}
}

// TestCreateBookWithoutSummary verifies that the optional summary field is
// stored as NULL (NullString{Valid:false}) when omitted.
func TestCreateBookWithoutSummary(t *testing.T) {
	ctx := context.Background()
	db := internal.NewTestDB(t)

	id, err := insertBook(ctx, db, internal.Book{
		Title:         "Test Book No Summary",
		Author:        "Test Author",
		PublishedYear: 2023,
		Summary:       sql.NullString{Valid: false},
	})
	if err != nil {
		t.Fatalf("insertBook: %v", err)
	}
	if id == 0 {
		t.Fatal("expected a non-zero generated id")
	}

	got, ok, err := getBook(ctx, db, id)
	if err != nil {
		t.Fatalf("getBook: %v", err)
	}
	if !ok {
		t.Fatal("expected book to exist")
	}

	if got.Title != "Test Book No Summary" {
		t.Errorf("title: got %q, want %q", got.Title, "Test Book No Summary")
	}
	if got.Author != "Test Author" {
		t.Errorf("author: got %q, want %q", got.Author, "Test Author")
	}
	if got.PublishedYear != 2023 {
		t.Errorf("published_year: got %d, want %d", got.PublishedYear, 2023)
	}
	if got.Summary.Valid {
		t.Errorf("summary: expected NULL, got %+v", got.Summary)
	}
}

// TestBookGoString verifies the repr-style representation produced by
// internal.Book.GoString(), the Go equivalent of Python's __repr__.
func TestBookGoString(t *testing.T) {
	ctx := context.Background()
	db := internal.NewTestDB(t)

	id, err := insertBook(ctx, db, internal.Book{
		Title:         "Repr Test",
		Author:        "Repr Author",
		PublishedYear: 2023,
		Summary:       sql.NullString{String: "Test summary", Valid: true},
	})
	if err != nil {
		t.Fatalf("insertBook: %v", err)
	}

	got, ok, err := getBook(ctx, db, id)
	if err != nil {
		t.Fatalf("getBook: %v", err)
	}
	if !ok {
		t.Fatal("expected book to exist")
	}

	want := fmt.Sprintf("<Book(id=%d, title='Repr Test', author='Repr Author', year=2023)>", id)
	if got.GoString() != want {
		t.Errorf("GoString: got %q, want %q", got.GoString(), want)
	}
}

// TestBookString verifies the str-style representation produced by
// internal.Book.String(), the Go equivalent of Python's __str__.
func TestBookString(t *testing.T) {
	ctx := context.Background()
	db := internal.NewTestDB(t)

	id, err := insertBook(ctx, db, internal.Book{
		Title:         "String Test",
		Author:        "String Author",
		PublishedYear: 2023,
		Summary:       sql.NullString{Valid: false},
	})
	if err != nil {
		t.Fatalf("insertBook: %v", err)
	}

	got, ok, err := getBook(ctx, db, id)
	if err != nil {
		t.Fatalf("getBook: %v", err)
	}
	if !ok {
		t.Fatal("expected book to exist")
	}

	const want = "String Test by String Author (2023)"
	if got.String() != want {
		t.Errorf("String: got %q, want %q", got.String(), want)
	}
}

// TestUniqueConstraintViolation verifies that inserting a second book with the
// same title and author violates the unique constraint. The Python test simply
// asserted that commit() raised; here we assert the error is recognised by
// internal.IsUniqueViolation.
func TestUniqueConstraintViolation(t *testing.T) {
	ctx := context.Background()
	db := internal.NewTestDB(t)

	if _, err := insertBook(ctx, db, internal.Book{
		Title:         "Duplicate Test",
		Author:        "Duplicate Author",
		PublishedYear: 2023,
	}); err != nil {
		t.Fatalf("insert first book: %v", err)
	}

	_, err := insertBook(ctx, db, internal.Book{
		Title:         "Duplicate Test",
		Author:        "Duplicate Author",
		PublishedYear: 2024, // different year, same title+author
	})
	if err == nil {
		t.Fatal("expected a unique-constraint violation, got nil")
	}
	if !internal.IsUniqueViolation(err) {
		t.Errorf("expected IsUniqueViolation, got %v", err)
	}
}

// TestBooksWithSameTitleDifferentAuthors verifies that the unique constraint is
// on (title, author) together: same title with different authors is allowed.
func TestBooksWithSameTitleDifferentAuthors(t *testing.T) {
	ctx := context.Background()
	db := internal.NewTestDB(t)

	id1, err := insertBook(ctx, db, internal.Book{
		Title:         "Common Title",
		Author:        "Author One",
		PublishedYear: 2023,
	})
	if err != nil {
		t.Fatalf("insert book1: %v", err)
	}

	id2, err := insertBook(ctx, db, internal.Book{
		Title:         "Common Title",
		Author:        "Author Two",
		PublishedYear: 2023,
	})
	if err != nil {
		t.Fatalf("insert book2: %v", err)
	}

	if id1 == 0 || id2 == 0 {
		t.Fatalf("expected non-zero ids, got %d and %d", id1, id2)
	}
	if id1 == id2 {
		t.Errorf("expected distinct ids, both were %d", id1)
	}
}

// TestBooksWithSameAuthorDifferentTitles verifies that the same author with
// different titles is allowed by the (title, author) unique constraint.
func TestBooksWithSameAuthorDifferentTitles(t *testing.T) {
	ctx := context.Background()
	db := internal.NewTestDB(t)

	id1, err := insertBook(ctx, db, internal.Book{
		Title:         "First Book",
		Author:        "Prolific Author",
		PublishedYear: 2023,
	})
	if err != nil {
		t.Fatalf("insert book1: %v", err)
	}

	id2, err := insertBook(ctx, db, internal.Book{
		Title:         "Second Book",
		Author:        "Prolific Author",
		PublishedYear: 2024,
	})
	if err != nil {
		t.Fatalf("insert book2: %v", err)
	}

	if id1 == 0 || id2 == 0 {
		t.Fatalf("expected non-zero ids, got %d and %d", id1, id2)
	}
	if id1 == id2 {
		t.Errorf("expected distinct ids, both were %d", id1)
	}
}
