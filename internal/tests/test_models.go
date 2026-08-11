package tests

// This file is the Go analog of the source project's tests/test_models.py, a
// pytest suite for the SQLAlchemy Book model. It verifies basic creation,
// optional-field (summary) handling, the string/GoString representations, the
// NOT NULL constraint on published_year, and the composite UNIQUE constraint on
// (title, author).
//
// MIGRATION_NOTE: The Python source relied on pytest fixtures that built an
// in-memory SQLite database via Base.metadata.create_all. The Go equivalent
// uses the NewTestDB helper (see internal/conftest.go), which builds a real
// *sqlx.DB against the target PostgreSQL dialect with the schema already
// created. There is no ORM "session" with add/commit/refresh: rows are inserted
// with an explicit INSERT ... RETURNING id (PostgreSQL has no LastInsertId),
// and the generated id is scanned back into the model to mirror the source's
// db_session.refresh(book) call.
//
// MIGRATION_NOTE: repr()/str() on the SQLAlchemy Book map to the already-
// migrated GoString() and String() methods on internal.Book (see
// internal/model.go), so those cases assert against those methods directly.

import (
	"context"
	"testing"

	"github.com/example/bookcatalog/internal"
)

// insertBook inserts a Book and populates its ID via RETURNING id. It mirrors
// the source's db_session.add / commit / refresh sequence. The returned error
// lets individual tests assert on constraint violations (NOT NULL, UNIQUE).
func insertBook(ctx context.Context, db interface {
	GetContext(context.Context, interface{}, string, ...interface{}) error
}, b *internal.Book) error {
	const q = `INSERT INTO books (title, author, published_year, summary)
	           VALUES ($1, $2, $3, $4) RETURNING id`
	return db.GetContext(ctx, &b.ID, q, b.Title, b.Author, b.PublishedYear, b.Summary)
}

// TestCreateBook verifies that a fully-populated book persists and round-trips.
func TestCreateBook(t *testing.T) {
	ctx := context.Background()
	db := internal.NewTestDB(t)

	book := &internal.Book{
		Title:         "Test Book",
		Author:        "Test Author",
		PublishedYear: 2023,
		Summary:       ptr("A test book summary"),
	}

	if err := insertBook(ctx, db, book); err != nil {
		t.Fatalf("insertBook: %v", err)
	}

	if book.ID == 0 {
		t.Errorf("expected non-zero id, got %d", book.ID)
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

// TestCreateBookWithoutSummary verifies the optional summary field persists as
// NULL when omitted.
func TestCreateBookWithoutSummary(t *testing.T) {
	ctx := context.Background()
	db := internal.NewTestDB(t)

	book := &internal.Book{
		Title:         "Test Book No Summary",
		Author:        "Test Author",
		PublishedYear: 2023,
	}

	if err := insertBook(ctx, db, book); err != nil {
		t.Fatalf("insertBook: %v", err)
	}

	if book.ID == 0 {
		t.Errorf("expected non-zero id, got %d", book.ID)
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
		t.Errorf("summary = %v, want nil", book.Summary)
	}
}

// TestBookGoString exercises the GoString method (the Go analog of Python's
// repr()).
func TestBookGoString(t *testing.T) {
	ctx := context.Background()
	db := internal.NewTestDB(t)

	book := &internal.Book{
		Title:         "Repr Test",
		Author:        "Repr Author",
		PublishedYear: 2023,
		Summary:       ptr("Test summary"),
	}

	if err := insertBook(ctx, db, book); err != nil {
		t.Fatalf("insertBook: %v", err)
	}

	expected := "<Book(id=" + itoa(book.ID) + ", title='Repr Test', author='Repr Author', year=2023)>"
	if got := book.GoString(); got != expected {
		t.Errorf("GoString() = %q, want %q", got, expected)
	}
}

// TestBookString exercises the String method (the Go analog of Python's str()).
func TestBookString(t *testing.T) {
	ctx := context.Background()
	db := internal.NewTestDB(t)

	book := &internal.Book{
		Title:         "String Test",
		Author:        "String Author",
		PublishedYear: 2023,
	}

	if err := insertBook(ctx, db, book); err != nil {
		t.Fatalf("insertBook: %v", err)
	}

	const expected = "String Test by String Author (2023)"
	if got := book.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}

// TestUniqueConstraintViolation verifies that duplicate (title, author) pairs
// are rejected by the composite UNIQUE constraint.
func TestUniqueConstraintViolation(t *testing.T) {
	ctx := context.Background()
	db := internal.NewTestDB(t)

	book1 := &internal.Book{
		Title:         "Duplicate Test",
		Author:        "Duplicate Author",
		PublishedYear: 2023,
	}
	if err := insertBook(ctx, db, book1); err != nil {
		t.Fatalf("insert first book: %v", err)
	}

	book2 := &internal.Book{
		Title:         "Duplicate Test",
		Author:        "Duplicate Author",
		PublishedYear: 2024, // different year, same title/author
	}
	if err := insertBook(ctx, db, book2); err == nil {
		t.Fatal("expected unique constraint violation, got nil error")
	}
}

// TestBooksWithSameTitleDifferentAuthors verifies that identical titles under
// different authors are permitted.
func TestBooksWithSameTitleDifferentAuthors(t *testing.T) {
	ctx := context.Background()
	db := internal.NewTestDB(t)

	book1 := &internal.Book{Title: "Common Title", Author: "Author One", PublishedYear: 2023}
	book2 := &internal.Book{Title: "Common Title", Author: "Author Two", PublishedYear: 2023}

	if err := insertBook(ctx, db, book1); err != nil {
		t.Fatalf("insert first book: %v", err)
	}
	if err := insertBook(ctx, db, book2); err != nil {
		t.Fatalf("insert second book: %v", err)
	}

	if book1.ID == 0 || book2.ID == 0 {
		t.Errorf("expected non-zero ids, got %d and %d", book1.ID, book2.ID)
	}
	if book1.ID == book2.ID {
		t.Errorf("expected distinct ids, both were %d", book1.ID)
	}
}

// TestBooksWithSameAuthorDifferentTitles verifies that identical authors under
// different titles are permitted.
func TestBooksWithSameAuthorDifferentTitles(t *testing.T) {
	ctx := context.Background()
	db := internal.NewTestDB(t)

	book1 := &internal.Book{Title: "First Book", Author: "Prolific Author", PublishedYear: 2023}
	book2 := &internal.Book{Title: "Second Book", Author: "Prolific Author", PublishedYear: 2024}

	if err := insertBook(ctx, db, book1); err != nil {
		t.Fatalf("insert first book: %v", err)
	}
	if err := insertBook(ctx, db, book2); err != nil {
		t.Fatalf("insert second book: %v", err)
	}

	if book1.ID == 0 || book2.ID == 0 {
		t.Errorf("expected non-zero ids, got %d and %d", book1.ID, book2.ID)
	}
	if book1.ID == book2.ID {
		t.Errorf("expected distinct ids, both were %d", book1.ID)
	}
}

// ptr returns a pointer to v, used for the optional Summary field.
func ptr[T any](v T) *T { return &v }

// itoa renders an int64 id for interpolation into the expected GoString output.
func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
