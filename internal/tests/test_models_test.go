```go
package tests

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"bookcatalog/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateBook_TableDriven covers the Book.create behavioral specs:
//   - all fields including summary
//   - required fields but no summary (summary defaults to NULL)
func TestCreateBook_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		book          internal.Book
		wantSummary   sql.NullString
		wantNullSumm  bool
	}{
		{
			name: "all fields including summary",
			book: internal.Book{
				Title:         "All Fields Book",
				Author:        "Full Author",
				PublishedYear: 2021,
				Summary:       sql.NullString{String: "A complete summary", Valid: true},
			},
			wantSummary:  sql.NullString{String: "A complete summary", Valid: true},
			wantNullSumm: false,
		},
		{
			name: "required fields no summary",
			book: internal.Book{
				Title:         "No Summary Book",
				Author:        "Minimal Author",
				PublishedYear: 2022,
				Summary:       sql.NullString{Valid: false},
			},
			wantSummary:  sql.NullString{Valid: false},
			wantNullSumm: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db := internal.NewTestDB(t)

			id, err := insertBook(ctx, db, tc.book)
			require.NoError(t, err, "insertBook should succeed")
			assert.NotZero(t, id, "persisted book must have a non-zero auto-generated id")

			got, ok, err := getBook(ctx, db, id)
			require.NoError(t, err, "getBook should succeed")
			require.True(t, ok, "book should exist after insert")

			// Invariant: field values assigned at creation are unchanged after persistence
			assert.Equal(t, tc.book.Title, got.Title, "title should be preserved")
			assert.Equal(t, tc.book.Author, got.Author, "author should be preserved")
			assert.Equal(t, tc.book.PublishedYear, got.PublishedYear, "published_year should be preserved")

			if tc.wantNullSumm {
				assert.False(t, got.Summary.Valid, "summary should be NULL")
			} else {
				assert.True(t, got.Summary.Valid, "summary should be valid (non-NULL)")
				assert.Equal(t, tc.wantSummary.String, got.Summary.String, "summary string should match")
			}
		})
	}
}

// TestBookGoString_TableDriven covers Book.__repr__ behavioral spec:
// format "<Book(id={id}, title='{title}', author='{author}', year={published_year})>"
func TestBookGoString_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		book internal.Book
	}{
		{
			name: "repr with summary",
			book: internal.Book{
				Title:         "Repr Book One",
				Author:        "Repr Author One",
				PublishedYear: 2020,
				Summary:       sql.NullString{String: "Some summary", Valid: true},
			},
		},
		{
			name: "repr without summary",
			book: internal.Book{
				Title:         "Repr Book Two",
				Author:        "Repr Author Two",
				PublishedYear: 1999,
				Summary:       sql.NullString{Valid: false},
			},
		},
		{
			name: "repr with zero year",
			book: internal.Book{
				Title:         "Repr Book Three",
				Author:        "Repr Author Three",
				PublishedYear: 0,
				Summary:       sql.NullString{Valid: false},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db := internal.NewTestDB(t)

			id, err := insertBook(ctx, db, tc.book)
			require.NoError(t, err, "insertBook should succeed")
			require.NotZero(t, id, "id must be non-zero")

			got, ok, err := getBook(ctx, db, id)
			require.NoError(t, err)
			require.True(t, ok)

			// Invariant: repr always includes id, title, author, year in the specified format
			want := fmt.Sprintf(
				"<Book(id=%d, title='%s', author='%s', year=%d)>",
				id, tc.book.Title, tc.book.Author, tc.book.PublishedYear,
			)
			assert.Equal(t, want, got.GoString(), "GoString() must match repr format")
		})
	}
}

// TestBookString_TableDriven covers Book.__str__ behavioral spec:
// format "{title} by {author} ({published_year})"
func TestBookString_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		book internal.Book
	}{
		{
			name: "str with summary",
			book: internal.Book{
				Title:         "String Book Alpha",
				Author:        "String Author Alpha",
				PublishedYear: 2010,
				Summary:       sql.NullString{String: "Alpha summary", Valid: true},
			},
		},
		{
			name: "str without summary",
			book: internal.Book{
				Title:         "String Book Beta",
				Author:        "String Author Beta",
				PublishedYear: 2015,
				Summary:       sql.NullString{Valid: false},
			},
		},
		{
			name: "str with special characters in title",
			book: internal.Book{
				Title:         "The Go Programming Language",
				Author:        "Alan A. A. Donovan & Brian W. Kernighan",
				PublishedYear: 2015,
				Summary:       sql.NullString{Valid: false},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db := internal.NewTestDB(t)

			id, err := insertBook(ctx, db, tc.book)
			require.NoError(t, err, "insertBook should succeed")
			require.NotZero(t, id)

			got, ok, err := getBook(ctx, db, id)
			require.NoError(t, err)
			require.True(t, ok)

			// Invariant: str output always follows '<title> by <author> (<year>)'
			want := fmt.Sprintf("%s by %s (%d)", tc.book.Title, tc.book.Author, tc.book.PublishedYear)
			assert.Equal(t, want, got.String(), "String() must match str format")
		})
	}
}

// TestUniqueConstraint_TableDriven covers the unique (title, author) constraint:
//   - duplicate (title, author) → integrity error, even with different year
//   - same title, different author → both succeed with distinct ids
//   - same author, different title → both succeed with distinct ids
func TestUniqueConstraint_TableDriven(t *testing.T) {
	t.Run("duplicate title and author raises unique violation", func(t *testing.T) {
		tests := []struct {
			name   string
			first  internal.Book
			second internal.Book
		}{
			{
				name: "identical title and author different year",
				first: internal.Book{
					Title:         "Dup Title A",
					Author:        "Dup Author A",
					PublishedYear: 2000,
				},
				second: internal.Book{
					Title:         "Dup Title A",
					Author:        "Dup Author A",
					PublishedYear: 2001, // different year — still a violation
				},
			},
			{
				name: "identical title and author same year",
				first: internal.Book{
					Title:         "Dup Title B",
					Author:        "Dup Author B",
					PublishedYear: 2005,
				},
				second: internal.Book{
					Title:         "Dup Title B",
					Author:        "Dup Author B",
					PublishedYear: 2005,
				},
			},
			{
				name: "identical title and author with different summary",
				first: internal.Book{
					Title:         "Dup Title C",
					Author:        "Dup Author C",
					PublishedYear: 2010,
					Summary:       sql.NullString{Valid: false},
				},
				second: internal.Book{
					Title:         "Dup Title C",
					Author:        "Dup Author C",
					PublishedYear: 2011,
					Summary:       sql.NullString{String: "Some summary", Valid: true},
				},
			},
		}

		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				ctx := context.Background()
				db := internal.NewTestDB(t)

				_, err := insertBook(ctx, db, tc.first)
				require.NoError(t, err, "first insert should succeed")

				_, err = insertBook(ctx, db, tc.second)
				require.Error(t, err, "second insert should fail with unique violation")
				assert.True(t, internal.IsUniqueViolation(err),
					"error must be recognised as IsUniqueViolation, got: %v", err)
			})
		}
	})

	t.Run("same title different authors both persist", func(t *testing.T) {
		tests := []struct {
			name    string
			title   string
			author1 string
			author2 string
			year1   int
			year2   int
		}{
			{
				name:    "shared title two distinct authors",
				title:   "Shared Title X",
				author1: "Author X1",
				author2: "Author X2",
				year1:   2000,
				year2:   2001,
			},
			{
				name:    "shared title two distinct authors same year",
				title:   "Shared Title Y",
				author1: "Author Y1",
				author2: "Author Y2",
				year1:   2010,
				year2:   2010,
			},
		}

		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				ctx := context.Background()
				db := internal.NewTestDB(t)

				id1, err := insertBook(ctx, db, internal.Book{
					Title:         tc.title,
					Author:        tc.author1,
					PublishedYear: tc.year1,
				})
				require.NoError(t, err, "first book insert should succeed")
				assert.NotZero(t, id1)

				id2, err := insertBook(ctx, db, internal.Book{
					Title:         tc.title,
					Author:        tc.author2,
					PublishedYear: tc.year2,
				})
				require.NoError(t, err, "second book insert should succeed")
				assert.NotZero(t, id2)

				// Invariant: distinct persisted books always have distinct ids
				assert.NotEqual(t, id1, id2, "distinct books must have distinct ids")
			})
		}
	})

	t.Run("same author different titles both persist", func(t *testing.T) {
		tests := []struct {
			name   string
			author string
			title1 string
			title2 string
			year1  int
			year2  int
		}{
			{
				name:   "prolific author two books",
				author: "Prolific Author Z",
				title1: "First Title Z",
				title2: "Second Title Z",
				year1:  2018,
				year2:  2019,
			},
			{
				name:   "prolific author two books same year",
				author: "Prolific Author W",
				title1: "Title W Alpha",
				title2: "Title W Beta",
				year1:  2022,
				year2:  2022,
			},
		}

		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				ctx := context.Background()
				db := internal.NewTestDB(t)

				id1, err := insertBook(ctx, db, internal.Book{
					Title:         tc.title1,
					Author:        tc.author,
					PublishedYear: tc.year1,
				})
				require.NoError(t, err, "first book insert should succeed")
				assert.NotZero(t, id1)

				id2, err := insertBook(ctx, db, internal.Book{
					Title:         tc.title2,
					Author:        tc.author,
					PublishedYear: tc.year2,
				})
				require.NoError(t, err, "second book insert should succeed")
				assert.NotZero(t, id2)

				// Invariant: distinct persisted books always have distinct ids
				assert.NotEqual(t, id1, id2, "distinct books must have distinct ids")
			})
		}
	})
}

// TestInsertBook_Helper covers edge-cases of the insertBook helper itself.
func TestInsertBook_Helper(t *testing.T) {
	tests := []struct {
		name    string
		book    internal.Book
		wantErr bool
	}{
		{
			name: "valid minimal book",
			book: internal.Book{
				Title:         "Helper Test",
				Author:        "Helper Author",
				PublishedYear: 2023,
			},
			wantErr: false,
		},
		{
			name: "valid book with summary",
			book: internal.Book{
				Title:         "Helper Summary Book",
				Author:        "Helper Author Two",
				PublishedYear: 1990,
				Summary:       sql.NullString{String: "Helper summary text", Valid: true},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db := internal.NewTestDB(t)

			id, err := insertBook(ctx, db, tc.book)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotZero(t, id, "id must be non-zero after successful insert")
		})
	}
}

// TestGetBook_NotFound verifies getBook returns (zero, false, nil) when no row exists.
func TestGetBook_NotFound(t *testing.T) {
	ctx := context.Background()
	db := internal.NewTestDB(t)

	got, ok, err := getBook(ctx, db, 999999999)
	assert.NoError(t, err, "no error expected for missing row")
	assert.False(t, ok, "ok should be false when book does not exist")
	assert.Equal(t, internal.Book{}, got, "returned book should be zero value")
}

// TestAutoGeneratedIDsAreUnique verifies that multiple distinct inserts each
// receive a unique non-zero id — covering the global invariant that every
// persisted book has a unique auto-generated non-null id.
func TestAutoGeneratedIDsAreUnique(t *testing.T) {