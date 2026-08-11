```go
package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/example/bookcatalog/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateBook_TableDriven covers all Book.create behavioral specs using
// table-driven tests.
func TestCreateBook_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		book          *internal.Book
		wantID        bool   // expect a non-zero id
		wantSummary   bool   // expect a non-nil summary
		wantSummaryV  string // expected summary value when wantSummary==true
		wantErr       bool
	}{
		{
			name: "all fields including summary",
			book: &internal.Book{
				Title:         "Full Book",
				Author:        "Full Author",
				PublishedYear: 2020,
				Summary:       ptr("A great read"),
			},
			wantID:       true,
			wantSummary:  true,
			wantSummaryV: "A great read",
			wantErr:      false,
		},
		{
			name: "without summary",
			book: &internal.Book{
				Title:         "No Summary Book",
				Author:        "Silent Author",
				PublishedYear: 2021,
			},
			wantID:      true,
			wantSummary: false,
			wantErr:     false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := internal.NewTestDB(t)

			err := insertBook(ctx, db, tc.book)
			if tc.wantErr {
				assert.Error(t, err, "expected an error inserting book")
				return
			}
			require.NoError(t, err, "insertBook should succeed")

			if tc.wantID {
				assert.NotZero(t, tc.book.ID, "id should be non-zero after insert")
			}
			assert.Equal(t, tc.book.Title, tc.book.Title)
			assert.Equal(t, tc.book.Author, tc.book.Author)
			assert.Equal(t, tc.book.PublishedYear, tc.book.PublishedYear)

			if tc.wantSummary {
				require.NotNil(t, tc.book.Summary, "summary should not be nil")
				assert.Equal(t, tc.wantSummaryV, *tc.book.Summary)
			} else {
				assert.Nil(t, tc.book.Summary, "summary should be nil")
			}
		})
	}
}

// TestUniqueConstraint_TableDriven covers the UNIQUE (title, author) constraint
// scenarios.
func TestUniqueConstraint_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		first       *internal.Book
		second      *internal.Book
		wantErrOnB2 bool
		wantDistinctIDs bool
	}{
		{
			name: "duplicate (title, author) rejected",
			first: &internal.Book{
				Title:         "Dup Title",
				Author:        "Dup Author",
				PublishedYear: 2000,
			},
			second: &internal.Book{
				Title:         "Dup Title",
				Author:        "Dup Author",
				PublishedYear: 2001,
			},
			wantErrOnB2:     true,
			wantDistinctIDs: false,
		},
		{
			name: "same title different authors allowed",
			first: &internal.Book{
				Title:         "Shared Title",
				Author:        "Author Alpha",
				PublishedYear: 2010,
			},
			second: &internal.Book{
				Title:         "Shared Title",
				Author:        "Author Beta",
				PublishedYear: 2010,
			},
			wantErrOnB2:     false,
			wantDistinctIDs: true,
		},
		{
			name: "same author different titles allowed",
			first: &internal.Book{
				Title:         "Title One",
				Author:        "Prolific Writer",
				PublishedYear: 2015,
			},
			second: &internal.Book{
				Title:         "Title Two",
				Author:        "Prolific Writer",
				PublishedYear: 2016,
			},
			wantErrOnB2:     false,
			wantDistinctIDs: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := internal.NewTestDB(t)

			err := insertBook(ctx, db, tc.first)
			require.NoError(t, err, "first insertBook should succeed")
			assert.NotZero(t, tc.first.ID, "first book should have non-zero id")

			err = insertBook(ctx, db, tc.second)
			if tc.wantErrOnB2 {
				assert.Error(t, err, "expected constraint violation error on second insert")
			} else {
				require.NoError(t, err, "second insertBook should succeed")
				assert.NotZero(t, tc.second.ID, "second book should have non-zero id")
			}

			if tc.wantDistinctIDs {
				assert.NotEqual(t, tc.first.ID, tc.second.ID, "persisted books must have distinct ids")
			}
		})
	}
}

// TestBookGoString_TableDriven covers Book.__repr__ (GoString) behavioral specs.
func TestBookGoString_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		book    *internal.Book
		fmtWant func(id int64) string
	}{
		{
			name: "repr with summary",
			book: &internal.Book{
				Title:         "Repr Book",
				Author:        "Repr Author",
				PublishedYear: 2023,
				Summary:       ptr("Some summary"),
			},
			fmtWant: func(id int64) string {
				return fmt.Sprintf("<Book(id=%s, title='Repr Book', author='Repr Author', year=2023)>", itoa(id))
			},
		},
		{
			name: "repr without summary",
			book: &internal.Book{
				Title:         "Plain Repr Book",
				Author:        "Plain Author",
				PublishedYear: 1999,
			},
			fmtWant: func(id int64) string {
				return fmt.Sprintf("<Book(id=%s, title='Plain Repr Book', author='Plain Author', year=1999)>", itoa(id))
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := internal.NewTestDB(t)

			err := insertBook(ctx, db, tc.book)
			require.NoError(t, err, "insertBook should succeed")
			assert.NotZero(t, tc.book.ID, "id must be non-zero")

			want := tc.fmtWant(tc.book.ID)
			assert.Equal(t, want, tc.book.GoString(),
				"GoString() should match expected repr format")
		})
	}
}

// TestBookString_TableDriven covers Book.__str__ (String) behavioral specs.
func TestBookString_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		book    *internal.Book
		wantStr string
	}{
		{
			name: "str with summary",
			book: &internal.Book{
				Title:         "String Book",
				Author:        "String Author",
				PublishedYear: 2023,
				Summary:       ptr("A summary"),
			},
			wantStr: "String Book by String Author (2023)",
		},
		{
			name: "str without summary",
			book: &internal.Book{
				Title:         "No Summary Str",
				Author:        "Quiet Author",
				PublishedYear: 1985,
			},
			wantStr: "No Summary Str by Quiet Author (1985)",
		},
		{
			name: "str with special characters in title",
			book: &internal.Book{
				Title:         "It's Complicated",
				Author:        "Complex Author",
				PublishedYear: 2010,
			},
			wantStr: "It's Complicated by Complex Author (2010)",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := internal.NewTestDB(t)

			err := insertBook(ctx, db, tc.book)
			require.NoError(t, err, "insertBook should succeed")
			assert.NotZero(t, tc.book.ID, "id must be non-zero")

			assert.Equal(t, tc.wantStr, tc.book.String(),
				"String() should match expected human-readable format")
		})
	}
}

// TestSchemaCreation_TableDriven verifies that Base.metadata.create_all
// (NewTestDB) creates a usable schema with the expected constraints.
func TestSchemaCreation_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupBooks  []*internal.Book
		wantAllOK   bool
		wantLastErr bool
	}{
		{
			name: "schema allows inserting a valid book",
			setupBooks: []*internal.Book{
				{Title: "Schema Test", Author: "Schema Author", PublishedYear: 2000},
			},
			wantAllOK: true,
		},
		{
			name: "schema enforces unique (title, author) constraint",
			setupBooks: []*internal.Book{
				{Title: "Schema Dup", Author: "Schema Dup Author", PublishedYear: 2000},
				{Title: "Schema Dup", Author: "Schema Dup Author", PublishedYear: 2001},
			},
			wantAllOK:   false,
			wantLastErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := internal.NewTestDB(t)

			var lastErr error
			for i, b := range tc.setupBooks {
				err := insertBook(ctx, db, b)
				if err != nil {
					lastErr = err
					if tc.wantLastErr && i == len(tc.setupBooks)-1 {
						// expected
						break
					}
					t.Fatalf("unexpected error inserting book %d: %v", i, err)
				}
				assert.NotZero(t, b.ID, "each inserted book should have a non-zero id")
			}

			if tc.wantLastErr {
				assert.Error(t, lastErr, "last insert should produce a constraint error")
			} else {
				assert.NoError(t, lastErr)
			}
		})
	}
}

// TestBookInvariants_TableDriven exercises the global invariants:
//   - every persisted book has a non-null, unique id
//   - summary defaults to nil when not provided
//   - distinct persisted books always have distinct ids
func TestBookInvariants_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		books []*internal.Book
	}{
		{
			name: "all ids are non-zero and distinct after batch insert",
			books: []*internal.Book{
				{Title: "Inv Book 1", Author: "Inv Author 1", PublishedYear: 2001},
				{Title: "Inv Book 2", Author: "Inv Author 2", PublishedYear: 2002},
				{Title: "Inv Book 3", Author: "Inv Author 3", PublishedYear: 2003},
			},
		},
		{
			name: "summary defaults to nil",
			books: []*internal.Book{
				{Title: "Nil Summary", Author: "Nil Author", PublishedYear: 2022},
			},
		},
		{
			name: "book with summary retains summary value",
			books: []*internal.Book{
				{Title: "Has Summary", Author: "Has Author", PublishedYear: 2022, Summary: ptr("hello")},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := internal.NewTestDB(t)

			seenIDs := make(map[int64]bool)
			for _, b := range tc.books {
				hadSummary := b.Summary != nil
				var summaryVal string
				if hadSummary {
					summaryVal = *b.Summary
				}

				err := insertBook(ctx, db, b)
				require.NoError(t, err, "insertBook must not fail for valid book")
				assert.NotZero(t, b.ID, "id must be non-zero")
				assert.False(t, seenIDs[b.ID], "id must be unique across all inserts, got duplicate %d", b.ID)
				seenIDs[b.ID] = true

				if hadSummary {
					require.NotNil(t, b.Summary, "summary should remain non-nil")
					assert.Equal(t, summaryVal, *b.Summary, "summary value should be preserved")
				} else {
					assert.Nil(t, b.Summary, "summary should be nil when not provided")
				}
			}
		})
	}
}

// TestBookRepresentationPurity_TableDriven verifies that GoString and String are
// pure functions of the book's field values (same input → same output).
func TestBookRepresentationPurity_TableDriven(t *testing.T) {
	t.Parallel()

	type repCase struct {
		name          string
		title         string
		author        string
		publishedYear int
	}

	cases := []repCase{
		{"basic", "Pure Title", "Pure Author", 2023},
		{"year boundary low", "Old Book", "Old Author", 1},
		{"year boundary high", "Future Book", "Future Author", 9999},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := internal.NewTestDB(t)

			b := &internal.Book{
				Title:         c.title,
				Author:        c.author,
				PublishedYear: c.publishedYear,
			}
			err := insertBook(ctx, db, b)
			require.NoError(t, err)
			require.NotZero(t, b.ID)

			// Call each method twice and assert idempotency.
			gs1 := b.GoString()
			gs2 := b.GoString()
			assert.Equal(t, gs1, gs2, "GoString() must be deterministic")

			st1 := b.String()
			st2 := b.String()
			assert.Equal(t, st1, st2,