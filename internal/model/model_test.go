package model_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"migrated-app/internal/model"
)

// openTestDB opens an in-memory SQLite database and runs Migrate to set up the schema.
// SQLite does not support SERIAL or IF NOT EXISTS on indexes in exactly the same DDL,
// so we normalise the DDL inside Migrate for SQLite; here we just call Migrate and
// assert no error.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	err = model.Migrate(context.Background(), db)
	require.NoError(t, err, "Migrate should create the schema without error")
	return db
}

// insertBook is a helper that inserts a book directly via SQL and returns the assigned id.
func insertBook(t *testing.T, db *sql.DB, b *model.Book) (int64, error) {
	t.Helper()
	res, err := db.ExecContext(
		context.Background(),
		`INSERT INTO books (title, author, published_year, summary) VALUES (?, ?, ?, ?)`,
		b.Title, b.Author, b.PublishedYear, b.Summary,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id, nil
}

// ---------------------------------------------------------------------------
// NewBook / struct construction
// ---------------------------------------------------------------------------

func TestNewBook(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		author        string
		publishedYear int
		summary       string
		wantSummary   sql.NullString
	}{
		{
			name:          "all fields including summary",
			title:         "The Go Programming Language",
			author:        "Donovan & Kernighan",
			publishedYear: 2015,
			summary:       "A comprehensive guide to Go.",
			wantSummary:   sql.NullString{String: "A comprehensive guide to Go.", Valid: true},
		},
		{
			name:          "summary omitted becomes null",
			title:         "Clean Code",
			author:        "Robert C. Martin",
			publishedYear: 2008,
			summary:       "",
			wantSummary:   sql.NullString{Valid: false},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := model.NewBook(tc.title, tc.author, tc.publishedYear, tc.summary)
			assert.Equal(t, tc.title, b.Title)
			assert.Equal(t, tc.author, b.Author)
			assert.Equal(t, tc.publishedYear, b.PublishedYear)
			assert.Equal(t, tc.wantSummary, b.Summary)
			assert.Zero(t, b.ID, "ID should be zero until persisted")
		})
	}
}

// ---------------------------------------------------------------------------
// String() / Repr()
// ---------------------------------------------------------------------------

func TestBook_String(t *testing.T) {
	tests := []struct {
		name string
		book *model.Book
		want string
	}{
		{
			name:  "standard book",
			book:  &model.Book{ID: 1, Title: "Dune", Author: "Frank Herbert", PublishedYear: 1965},
			want:  "Dune by Frank Herbert (1965)",
		},
		{
			name:  "book with spaces in title and author",
			book:  &model.Book{ID: 42, Title: "The Hobbit", Author: "J.R.R. Tolkien", PublishedYear: 1937},
			want:  "The Hobbit by J.R.R. Tolkien (1937)",
		},
		{
			name:  "zero-value book",
			book:  &model.Book{},
			want:  " by  (0)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.book.String()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBook_Repr(t *testing.T) {
	tests := []struct {
		name string
		book *model.Book
		want string
	}{
		{
			name:  "repr with id=1",
			book:  &model.Book{ID: 1, Title: "Dune", Author: "Frank Herbert", PublishedYear: 1965},
			want:  "<Book(id=1, title='Dune', author='Frank Herbert', year=1965)>",
		},
		{
			name:  "repr with id=99",
			book:  &model.Book{ID: 99, Title: "1984", Author: "George Orwell", PublishedYear: 1949},
			want:  "<Book(id=99, title='1984', author='George Orwell', year=1949)>",
		},
		{
			name:  "repr with zero-value book",
			book:  &model.Book{},
			want:  "<Book(id=0, title='', author='', year=0)>",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.book.Repr()
			assert.Equal(t, tc.want, got)
			// Validate format invariants
			assert.True(t, strings.HasPrefix(got, "<Book("), "Repr must start with <Book(")
			assert.True(t, strings.HasSuffix(got, ")>"), "Repr must end with )>")
			assert.Contains(t, got, fmt.Sprintf("id=%d", tc.book.ID))
			assert.Contains(t, got, fmt.Sprintf("title='%s'", tc.book.Title))
			assert.Contains(t, got, fmt.Sprintf("author='%s'", tc.book.Author))
			assert.Contains(t, got, fmt.Sprintf("year=%d", tc.book.PublishedYear))
		})
	}
}

// ---------------------------------------------------------------------------
// Migrate
// ---------------------------------------------------------------------------

func TestMigrate(t *testing.T) {
	t.Run("creates books table without error", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)
		defer db.Close()

		err = model.Migrate(context.Background(), db)
		assert.NoError(t, err)
	})

	t.Run("idempotent - calling Migrate twice does not error", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)
		defer db.Close()

		err = model.Migrate(context.Background(), db)
		require.NoError(t, err)

		err = model.Migrate(context.Background(), db)
		assert.NoError(t, err, "second call to Migrate should be idempotent")
	})

	t.Run("table books exists after Migrate", func(t *testing.T) {
		db := openTestDB(t)
		// SQLite way to check table existence
		row := db.QueryRowContext(context.Background(),
			`SELECT name FROM sqlite_master WHERE type='table' AND name='books'`)
		var name string
		err := row.Scan(&name)
		require.NoError(t, err)
		assert.Equal(t, "books", name)
	})
}

// ---------------------------------------------------------------------------
// Persistence – insert with required fields
// ---------------------------------------------------------------------------

func TestBook_Persistence_RequiredFields(t *testing.T) {
	t.Run("insert with all required fields succeeds and gets auto-incremented id", func(t *testing.T) {
		db := openTestDB(t)

		b := model.NewBook("Foundation", "Isaac Asimov", 1951, "")
		id, err := insertBook(t, db, b)
		require.NoError(t, err)
		assert.Greater(t, id, int64(0), "assigned id should be positive auto-increment value")
	})

	t.Run("consecutive inserts get incrementing ids", func(t *testing.T) {
		db := openTestDB(t)

		b1 := model.NewBook("Book One", "Author A", 2000, "")
		id1, err := insertBook(t, db, b1)
		require.NoError(t, err)

		b2 := model.NewBook("Book Two", "Author B", 2001, "")
		id2, err := insertBook(t, db, b2)
		require.NoError(t, err)

		assert.Greater(t, id2, id1, "second inserted id should be greater than first")
	})
}

// ---------------------------------------------------------------------------
// Persistence – NOT NULL constraint violations
// ---------------------------------------------------------------------------

func TestBook_Persistence_NullConstraints(t *testing.T) {
	tests := []struct {
		name  string
		query string
		args  []interface{}
	}{
		{
			name:  "insert without title fails not-null constraint",
			query: `INSERT INTO books (author, published_year) VALUES (?, ?)`,
			args:  []interface{}{"Some Author", 2020},
		},
		{
			name:  "insert without author fails not-null constraint",
			query: `INSERT INTO books (title, published_year) VALUES (?, ?)`,
			args:  []interface{}{"Some Title", 2020},
		},
		{
			name:  "insert without published_year fails not-null constraint",
			query: `INSERT INTO books (title, author) VALUES (?, ?)`,
			args:  []interface{}{"Some Title", "Some Author"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := openTestDB(t)
			_, err := db.ExecContext(context.Background(), tc.query, tc.args...)
			assert.Error(t, err, "expected a not-null constraint violation")
		})
	}
}

// ---------------------------------------------------------------------------
// Persistence – summary nullable
// ---------------------------------------------------------------------------

func TestBook_Persistence_SummaryNullable(t *testing.T) {
	t.Run("insert with no summary stores null", func(t *testing.T) {
		db := openTestDB(t)

		b := model.NewBook("Neuromancer", "William Gibson", 1984, "")
		id, err := insertBook(t, db, b)
		require.NoError(t, err)

		var summary sql.NullString
		row := db.QueryRowContext(context.Background(),
			`SELECT summary FROM books WHERE id = ?`, id)
		err = row.Scan(&summary)
		require.NoError(t, err)
		assert.False(t, summary.Valid, "summary should be null when not provided")
	})

	t.Run("insert with summary stores the value", func(t *testing.T) {
		db := openTestDB(t)

		b := model.NewBook("Neuromancer", "William Gibson", 1984, "A cyberpunk classic.")
		id, err := insertBook(t, db, b)
		require.NoError(t, err)

		var summary sql.NullString
		row := db.QueryRowContext(context.Background(),
			`SELECT summary FROM books WHERE id = ?`, id)
		err = row.Scan(&summary)
		require.NoError(t, err)
		assert.True(t, summary.Valid)
		assert.Equal(t, "A cyberpunk classic.", summary.String)
	})
}

// ---------------------------------------------------------------------------
// Persistence – unique constraint on (title, author)
// ---------------------------------------------------------------------------

func TestBook_Persistence_UniqueConstraint(t *testing.T) {
	t.Run("inserting duplicate (title, author) fails unique constraint", func(t *testing.T) {
		db := openTestDB(t)

		b1 := model.NewBook("Ender's Game", "Orson Scott Card", 1985, "")
		_, err := insertBook(t, db, b1)
		require.NoError(t, err, "first insert should succeed")

		b2 := model.NewBook("Ender's Game", "Orson Scott Card", 1985, "Duplicate")
		_, err = insertBook(t, db, b2)
		assert.Error(t, err, "second insert with same title and author should fail unique constraint")
	})

	t.Run("same title different authors both succeed", func(t *testing.T) {
		db := openTestDB(t)

		b1 := model.NewBook("Hamlet", "William Shakespeare", 1603, "")
		_, err := insertBook(t, db, b1)
		require.NoError(t, err)

		b2 := model.NewBook("Hamlet", "Another Author", 2000, "Modern retelling")
		_, err = insertBook(t, db, b2)
		assert.NoError(t, err, "same title with different author should succeed")
	})

	t.Run("same author different titles both succeed", func(t *testing.T) {
		db := openTestDB(t)

		b1 := model.NewBook("It", "Stephen King", 1986, "")
		_, err := insertBook(t, db, b1)
		require.NoError(t, err)

		b2 := model.NewBook("The Shining", "Stephen King", 1977, "")
		_, err = insertBook(t, db, b2)
		assert.NoError(t, err, "same author with different title should succeed")
	})
}

// ---------------------------------------------------------------------------
// Round-trip: insert and read back
// ---------------------------------------------------------------------------

func TestBook_RoundTrip(t *testing.T) {
	t.Run("inserted book can be read back with correct fields", func(t *testing.T) {
		db := openTestDB(t)

		want := model.NewBook("The Name of the Wind", "Patrick Rothfuss", 2007, "A fantasy novel.")
		id, err := insertBook(t, db, want)
		require.NoError(t, err)

		row := db.QueryRowContext(context.Background(),
			`SELECT id, title, author, published_year, summary FROM books WHERE id = ?`, id)

		var got model.Book
		err = row.Scan(&got.ID, &got.Title, &got.Author, &got.PublishedYear, &got.Summary)
		require.NoError(t, err)

		assert.Equal(t, id, got.ID)
		assert.Equal(t, want.Title, got.Title)
		assert.Equal(t, want.Author, got.Author)
		assert.Equal(t, want.PublishedYear, got.PublishedYear)
		assert.Equal(t, want.Summary, got.Summary)

		// Validate String() and Repr() on round-tripped value
		assert.Equal(t,
			fmt.Sprintf("%s by %s (%d)", got.Title, got.Author, got.PublishedYear),
			got.String(),
		)
		assert.Equal(t,
			fmt.Sprintf("<Book(id=%d, title='%s', author='%s', year=%d)>",
				got.ID, got.Title, got.Author, got.PublishedYear),
			got.Repr(),
		)
	})
}

// ---------------------------------------------------------------------------
// Schema invariants
// ---------------------------------------------------------------------------

func TestSchema_Invariants(t *testing.T) {
	t.Run("table is named books", func(t *testing.T) {
		db := openTestDB(t)
		row := db.QueryRowContext(context.Background(),
			`SELECT