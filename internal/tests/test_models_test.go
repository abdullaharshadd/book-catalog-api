```go
package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"myapp/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// newTestDB obtains a fresh, schema-initialised DB for the duration of a
// single test. It registers a t.Cleanup that tears the database down so each
// test is fully isolated.
//
// We call internal.NewTestDB which is declared in internal/conftest.go and
// is responsible for provisioning a PostgreSQL database, running migrations,
// and returning an *internal.DB.
func newTestDB(t *testing.T) *internal.DB {
	t.Helper()
	db, err := internal.NewTestDB(t)
	require.NoError(t, err, "NewTestDB should succeed")
	return db
}

// ---------------------------------------------------------------------------
// Book.create
// ---------------------------------------------------------------------------

func TestBookCreate(t *testing.T) {
	t.Parallel()

	type tc struct {
		name          string
		title         string
		author        string
		publishedYear int
		summary       *string
	}

	cases := []tc{
		{
			name:          "all fields including summary",
			title:         "The Go Programming Language",
			author:        "Alan Donovan",
			publishedYear: 2015,
			summary:       strPtr("An introduction to the Go programming language."),
		},
		{
			name:          "numeric title",
			title:         "1984",
			author:        "George Orwell",
			publishedYear: 1949,
			summary:       strPtr("A dystopian novel."),
		},
		{
			name:          "long summary",
			title:         "War and Peace",
			author:        "Leo Tolstoy",
			publishedYear: 1869,
			summary:       strPtr(strings.Repeat("a", 1000)),
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := newTestDB(t)

			row, err := insertBook(ctx, db, c.title, c.author, c.publishedYear, c.summary)
			require.NoError(t, err, "insertBook should succeed")

			// id must be non-zero (auto-generated)
			assert.Greater(t, row.ID, int64(0), "id must be non-null/non-zero")

			// All provided field values must be stored unchanged.
			assert.Equal(t, c.title, row.Title, "title must match")
			assert.Equal(t, c.author, row.Author, "author must match")
			assert.Equal(t, c.publishedYear, row.PublishedYear, "published_year must match")

			if c.summary != nil {
				require.NotNil(t, row.Summary, "summary must not be nil")
				assert.Equal(t, *c.summary, *row.Summary, "summary content must match")
			} else {
				assert.Nil(t, row.Summary, "summary must be nil when not provided")
			}

			// Verify the data is actually readable back from the DB.
			got, found, err := getBook(ctx, db, row.ID)
			require.NoError(t, err)
			require.True(t, found, "book should be retrievable by id")
			assert.Equal(t, row, got, "retrieved row must equal inserted row")
		})
	}
}

// ---------------------------------------------------------------------------
// Book.create_without_summary
// ---------------------------------------------------------------------------

func TestBookCreateWithoutSummary(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		title         string
		author        string
		publishedYear int
	}{
		{
			name:          "nil summary stored as NULL",
			title:         "Invisible Man",
			author:        "Ralph Ellison",
			publishedYear: 1952,
		},
		{
			name:          "another book without summary",
			title:         "Beloved",
			author:        "Toni Morrison",
			publishedYear: 1987,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := newTestDB(t)

			row, err := insertBook(ctx, db, c.title, c.author, c.publishedYear, nil)
			require.NoError(t, err, "insertBook without summary should succeed")

			assert.Greater(t, row.ID, int64(0), "id must be non-null/non-zero")
			assert.Equal(t, c.title, row.Title)
			assert.Equal(t, c.author, row.Author)
			assert.Equal(t, c.publishedYear, row.PublishedYear)
			assert.Nil(t, row.Summary, "summary must be nil (NULL) when not provided")

			// Round-trip: read back and confirm NULL survives the DB round-trip.
			got, found, err := getBook(ctx, db, row.ID)
			require.NoError(t, err)
			require.True(t, found)
			assert.Nil(t, got.Summary, "summary must still be nil after retrieval")
		})
	}
}

// ---------------------------------------------------------------------------
// Book.__repr__   (GoString / developer-facing representation)
// ---------------------------------------------------------------------------

func TestBookRepr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		title         string
		author        string
		publishedYear int
		summary       *string
	}{
		{
			name:          "book with summary",
			title:         "Dune",
			author:        "Frank Herbert",
			publishedYear: 1965,
			summary:       strPtr("A science fiction epic."),
		},
		{
			name:          "book without summary",
			title:         "Foundation",
			author:        "Isaac Asimov",
			publishedYear: 1951,
			summary:       nil,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := newTestDB(t)

			row, err := insertBook(ctx, db, c.title, c.author, c.publishedYear, c.summary)
			require.NoError(t, err)

			book := row.toModel()
			want := fmt.Sprintf("<Book(id=%d, title='%s', author='%s', year=%d)>",
				book.ID, book.Title, book.Author, book.PublishedYear)

			repr := book.GoString()
			assert.Equal(t, want, repr, "GoString() must match expected format")

			// Format invariants
			assert.Contains(t, repr, fmt.Sprintf("id=%d", book.ID))
			assert.Contains(t, repr, fmt.Sprintf("title='%s'", book.Title))
			assert.Contains(t, repr, fmt.Sprintf("author='%s'", book.Author))
			assert.Contains(t, repr, fmt.Sprintf("year=%d", book.PublishedYear))
		})
	}
}

// ---------------------------------------------------------------------------
// Book.__str__   (human-facing representation)
// ---------------------------------------------------------------------------

func TestBookStr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		title         string
		author        string
		publishedYear int
		summary       *string
	}{
		{
			name:          "standard book",
			title:         "The Hobbit",
			author:        "J.R.R. Tolkien",
			publishedYear: 1937,
			summary:       strPtr("There and back again."),
		},
		{
			name:          "book without summary",
			title:         "Neuromancer",
			author:        "William Gibson",
			publishedYear: 1984,
			summary:       nil,
		},
		{
			name:          "book with special characters",
			title:         "Don Quixote",
			author:        "Miguel de Cervantes",
			publishedYear: 1605,
			summary:       nil,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := newTestDB(t)

			row, err := insertBook(ctx, db, c.title, c.author, c.publishedYear, c.summary)
			require.NoError(t, err)

			book := row.toModel()
			want := fmt.Sprintf("%s by %s (%d)", book.Title, book.Author, book.PublishedYear)

			str := book.String()
			assert.Equal(t, want, str, "String() must match expected format")

			// Pattern invariants
			assert.Contains(t, str, book.Title)
			assert.Contains(t, str, "by")
			assert.Contains(t, str, book.Author)
			assert.Contains(t, str, fmt.Sprintf("(%d)", book.PublishedYear))
		})
	}
}

// ---------------------------------------------------------------------------
// Book.unique_title_author_constraint
// ---------------------------------------------------------------------------

func TestBookUniqueTitleAuthorConstraint(t *testing.T) {
	t.Parallel()

	t.Run("duplicate title and author fails", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		db := newTestDB(t)

		_, err := insertBook(ctx, db, "Hamlet", "Shakespeare", 1600, strPtr("A tragedy."))
		require.NoError(t, err, "first insert should succeed")

		// Same title + author, different year – must fail.
		_, err = insertBook(ctx, db, "Hamlet", "Shakespeare", 1601, nil)
		require.Error(t, err, "duplicate (title, author) must raise an error")

		// The error should hint at a constraint / uniqueness violation.
		errMsg := strings.ToLower(err.Error())
		isConstraintErr := strings.Contains(errMsg, "unique") ||
			strings.Contains(errMsg, "duplicate") ||
			strings.Contains(errMsg, "constraint") ||
			strings.Contains(errMsg, "violat")
		assert.True(t, isConstraintErr,
			"error message should indicate a uniqueness/constraint violation, got: %s", err.Error())
	})

	t.Run("same title different authors is allowed", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		db := newTestDB(t)

		row1, err := insertBook(ctx, db, "Hamlet", "Shakespeare", 1600, nil)
		require.NoError(t, err, "first insert should succeed")

		row2, err := insertBook(ctx, db, "Hamlet", "Somebody Else", 1999, nil)
		require.NoError(t, err, "second insert with different author should succeed")

		assert.Greater(t, row1.ID, int64(0))
		assert.Greater(t, row2.ID, int64(0))
		assert.NotEqual(t, row1.ID, row2.ID, "distinct books must have distinct ids")
	})

	t.Run("same author different titles is allowed", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		db := newTestDB(t)

		row1, err := insertBook(ctx, db, "Hamlet", "Shakespeare", 1600, nil)
		require.NoError(t, err, "first insert should succeed")

		row2, err := insertBook(ctx, db, "Othello", "Shakespeare", 1603, nil)
		require.NoError(t, err, "second insert with different title should succeed")

		assert.Greater(t, row1.ID, int64(0))
		assert.Greater(t, row2.ID, int64(0))
		assert.NotEqual(t, row1.ID, row2.ID, "distinct books must have distinct ids")
	})

	t.Run("uniqueness is independent of published_year", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		db := newTestDB(t)

		_, err := insertBook(ctx, db, "Macbeth", "Shakespeare", 1606, nil)
		require.NoError(t, err)

		// Different year must NOT bypass the constraint.
		_, err = insertBook(ctx, db, "Macbeth", "Shakespeare", 2023, nil)
		require.Error(t, err, "same title+author with different year should still fail")
	})

	t.Run("uniqueness is independent of summary", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		db := newTestDB(t)

		_, err := insertBook(ctx, db, "King Lear", "Shakespeare", 1606, strPtr("A tragedy"))
		require.NoError(t, err)

		_, err = insertBook(ctx, db, "King Lear", "Shakespeare", 1606, nil)
		require.Error(t, err, "same title+author with different summary should still fail")
	})
}

// ---------------------------------------------------------------------------
// Global invariants
// ---------------------------------------------------------------------------

func TestGlobalInvariants(t *testing.T) {
	t.Parallel()

	t.Run("every persisted book has a unique non-null id", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		db := newTestDB(t)

		titles := []string{"Book A", "Book B", "Book C", "Book D", "Book E"}
		ids := make(map[int64]struct{}, len(titles))

		for i, title := range titles {
			row, err := insertBook(ctx, db, title, "Same Author", 2000+i, nil)
			require.NoError(t, err)
			assert.Greater(t, row.ID, int64(0), "id must be > 0")
			_, dup := ids[row.ID]
			assert.False(t, dup, "id %d is duplicated", row.ID)
			ids[row.ID] = struct{}{}
		}
	})

	t.Run("field values are preserved exactly after retrieval", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		db := newTestDB(t)

		summary := strPtr("Exact content preservation test.")
		original, err := insertBook(ctx, db, "Precision", "Author Q", 2021, summary)
		require.NoError(t, err)

		retrieved, found, err := getBook(ctx, db, original.ID)
		require.NoError(t, err)
		require.True(t, found)

		assert.Equal(t, original.Title, retrieved.Title)
		assert.Equal(t, original.Author, retrieved.Author)
		assert.Equal(t, original.PublishedYear, retrieved.PublishedYear)
		require.NotNil(t, retrieved.Summary)
		assert.Equal(t, *summary, *retrieved.Summary)
	})

	t.Run("title author and published_year are required", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		db := newTestDB(t)

		// A well-formed insert with all required fields succeeds.
		row, err := insertBook(ctx, db, "Required Fields", "Author R", 2022, nil)
		require.NoError(t, err)
		assert.Greater(t, row.ID, int64(0))
		assert.Equal(t, "Required Fields", row.Title)
		assert.Equal(t, "Author R", row.Author)
		assert.Equal(t, 2022, row.PublishedYear)
	})

	t.Run("summary is optional