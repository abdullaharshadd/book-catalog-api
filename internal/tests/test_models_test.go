```go
package tests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"app/internal"
)

func init() {
	// Wire up the real os.LookupEnv for the test binary.
	lookupEnv = os.LookupEnv
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// skipIfNoDB skips the test when TEST_DATABASE_URL is not set.
func skipIfNoDB(t *testing.T) {
	t.Helper()
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping database integration tests")
	}
}

// ---------------------------------------------------------------------------
// TestEnvLookup – unit test for the environment-variable seam (no DB needed)
// ---------------------------------------------------------------------------

func TestEnvLookup(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		key     string
		environ map[string]string
		want    string
	}{
		{
			name:    "key present",
			key:     "MY_TEST_KEY_PRESENT",
			environ: map[string]string{"MY_TEST_KEY_PRESENT": "hello"},
			want:    "hello",
		},
		{
			name:    "key absent",
			key:     "MY_TEST_KEY_ABSENT",
			environ: map[string]string{},
			want:    "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Stub lookupEnv for this sub-test.
			orig := lookupEnv
			lookupEnv = func(key string) (string, bool) {
				v, ok := tc.environ[key]
				return v, ok
			}
			t.Cleanup(func() { lookupEnv = orig })

			got := envLookup(tc.key)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestOsGetenv – unit test for the thin osGetenv wrapper (no DB needed)
// ---------------------------------------------------------------------------

func TestOsGetenv(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		key     string
		environ map[string]string
		want    string
	}{
		{
			name:    "variable set",
			key:     "OSGETENV_TEST_SET",
			environ: map[string]string{"OSGETENV_TEST_SET": "42"},
			want:    "42",
		},
		{
			name:    "variable not set",
			key:     "OSGETENV_TEST_NOTSET",
			environ: map[string]string{},
			want:    "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orig := lookupEnv
			lookupEnv = func(key string) (string, bool) {
				v, ok := tc.environ[key]
				return v, ok
			}
			t.Cleanup(func() { lookupEnv = orig })

			got := osGetenv(tc.key)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestBookString – human-readable representation (no DB needed)
// ---------------------------------------------------------------------------

func TestBookString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		book internal.Book
		want string
	}{
		{
			name: "typical book",
			book: internal.Book{
				ID:            1,
				Title:         "String Test",
				Author:        "String Author",
				PublishedYear: 2023,
			},
			want: "String Test by String Author (2023)",
		},
		{
			name: "different year",
			book: internal.Book{
				ID:            2,
				Title:         "Old Book",
				Author:        "Old Author",
				PublishedYear: 1984,
			},
			want: "Old Book by Old Author (1984)",
		},
		{
			name: "book without id (unpersisted)",
			book: internal.Book{
				Title:         "Draft",
				Author:        "Nobody",
				PublishedYear: 2000,
			},
			want: "Draft by Nobody (2000)",
		},
		{
			name: "book with summary field set (summary not part of String)",
			book: internal.Book{
				ID:            3,
				Title:         "Summary Book",
				Author:        "Summary Author",
				PublishedYear: 2021,
				Summary:       strPtr("some summary"),
			},
			want: "Summary Book by Summary Author (2021)",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.book.String())
		})
	}
}

// ---------------------------------------------------------------------------
// TestBookGoString – developer-facing representation (no DB needed)
// ---------------------------------------------------------------------------

func TestBookGoString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		book internal.Book
		want string
	}{
		{
			name: "fully populated book",
			book: internal.Book{
				ID:            7,
				Title:         "Repr Test",
				Author:        "Repr Author",
				PublishedYear: 2023,
				Summary:       strPtr("Test summary"),
			},
			want: "<Book(id=7, title='Repr Test', author='Repr Author', year=2023)>",
		},
		{
			name: "book without summary",
			book: internal.Book{
				ID:            42,
				Title:         "No Summary",
				Author:        "Ghost Writer",
				PublishedYear: 2000,
			},
			want: "<Book(id=42, title='No Summary', author='Ghost Writer', year=2000)>",
		},
		{
			name: "zero id (unpersisted)",
			book: internal.Book{
				ID:            0,
				Title:         "Draft",
				Author:        "Nobody",
				PublishedYear: 2024,
			},
			want: "<Book(id=0, title='Draft', author='Nobody', year=2024)>",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.book.GoString())
		})
	}
}

// ---------------------------------------------------------------------------
// TestCreateBook – integration: full field set (requires DB)
// ---------------------------------------------------------------------------

func TestCreateBook_TableDriven(t *testing.T) {
	skipIfNoDB(t)

	cases := []struct {
		name          string
		title         string
		author        string
		publishedYear int
		summary       *string
	}{
		{
			name:          "all fields including summary",
			title:         "Full Book",
			author:        "Full Author",
			publishedYear: 2023,
			summary:       strPtr("A test book summary"),
		},
		{
			name:          "without summary",
			title:         "No Summary Book",
			author:        "No Summary Author",
			publishedYear: 2022,
			summary:       nil,
		},
		{
			name:          "empty summary pointer is nil",
			title:         "Nil Summary Book",
			author:        "Nil Summary Author",
			publishedYear: 2021,
			summary:       nil,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Do NOT run in parallel – each sub-test needs its own clean DB state.
			db := newTestDB(t)
			ctx := context.Background()

			input := internal.Book{
				Title:         tc.title,
				Author:        tc.author,
				PublishedYear: tc.publishedYear,
				Summary:       tc.summary,
			}

			saved, err := insertBook(ctx, db, input)
			require.NoError(t, err, "insertBook should succeed")
			assert.NotZero(t, saved.ID, "id must be non-zero after insert")

			got, err := getBook(ctx, db, saved.ID)
			require.NoError(t, err, "getBook should succeed")

			assert.Equal(t, tc.title, got.Title)
			assert.Equal(t, tc.author, got.Author)
			assert.Equal(t, tc.publishedYear, got.PublishedYear)

			if tc.summary == nil {
				assert.Nil(t, got.Summary, "summary should be nil")
			} else {
				require.NotNil(t, got.Summary, "summary should not be nil")
				assert.Equal(t, *tc.summary, *got.Summary)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestUniqueConstraint_TableDriven – integration: constraint scenarios
// ---------------------------------------------------------------------------

func TestUniqueConstraint_TableDriven(t *testing.T) {
	skipIfNoDB(t)

	cases := []struct {
		name        string
		first       internal.Book
		second      internal.Book
		expectError bool
	}{
		{
			name: "duplicate title and author (different year) → error",
			first: internal.Book{
				Title:         "Dup Title",
				Author:        "Dup Author",
				PublishedYear: 2023,
			},
			second: internal.Book{
				Title:         "Dup Title",
				Author:        "Dup Author",
				PublishedYear: 2024,
			},
			expectError: true,
		},
		{
			name: "same title different author → ok",
			first: internal.Book{
				Title:         "Common Title",
				Author:        "Author One",
				PublishedYear: 2023,
			},
			second: internal.Book{
				Title:         "Common Title",
				Author:        "Author Two",
				PublishedYear: 2023,
			},
			expectError: false,
		},
		{
			name: "same author different title → ok",
			first: internal.Book{
				Title:         "First Book",
				Author:        "Prolific Author",
				PublishedYear: 2023,
			},
			second: internal.Book{
				Title:         "Second Book",
				Author:        "Prolific Author",
				PublishedYear: 2024,
			},
			expectError: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db := newTestDB(t)
			ctx := context.Background()

			b1, err := insertBook(ctx, db, tc.first)
			require.NoError(t, err, "first insertBook should succeed")
			assert.NotZero(t, b1.ID, "first book should have a non-zero id")

			b2, err := insertBook(ctx, db, tc.second)

			if tc.expectError {
				assert.Error(t, err, "second insertBook should fail on unique constraint violation")
			} else {
				require.NoError(t, err, "second insertBook should succeed")
				assert.NotZero(t, b2.ID, "second book should have a non-zero id")
				assert.NotEqual(t, b1.ID, b2.ID, "distinct books must have distinct ids")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestBookGoString_AfterInsert – GoString on persisted book (requires DB)
// ---------------------------------------------------------------------------

func TestBookGoString_AfterInsert(t *testing.T) {
	skipIfNoDB(t)

	cases := []struct {
		name  string
		input internal.Book
	}{
		{
			name: "book with summary",
			input: internal.Book{
				Title:         "Repr Test",
				Author:        "Repr Author",
				PublishedYear: 2023,
				Summary:       strPtr("Test summary"),
			},
		},
		{
			name: "book without summary",
			input: internal.Book{
				Title:         "Plain Repr",
				Author:        "Plain Author",
				PublishedYear: 2000,
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db := newTestDB(t)
			ctx := context.Background()

			saved, err := insertBook(ctx, db, tc.input)
			require.NoError(t, err)
			assert.NotZero(t, saved.ID)

			want := fmt.Sprintf(
				"<Book(id=%d, title='%s', author='%s', year=%d)>",
				saved.ID, tc.input.Title, tc.input.Author, tc.input.PublishedYear,
			)
			assert.Equal(t, want, saved.GoString())
		})
	}
}

// ---------------------------------------------------------------------------
// TestBookString_AfterInsert – String() on persisted book (requires DB)
// ---------------------------------------------------------------------------

func TestBookString_AfterInsert(t *testing.T) {
	skipIfNoDB(t)

	cases := []struct {
		name  string
		input internal.Book
	}{
		{
			name: "standard book",
			input: internal.Book{
				Title:         "String Test",
				Author:        "String Author",
				PublishedYear: 2023,
			},
		},
		{
			name: "older publication",
			input: internal.Book{
				Title:         "Old Classic",
				Author:        "Classic Author",
				PublishedYear: 1950,
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db := newTestDB(t)
			ctx := context.Background()

			saved, err := insertBook(ctx, db, tc.input)
			require.NoError(t, err)

			want := fmt.Sprintf("%s by %s (%d)",
				tc.input.Title, tc.input.Author, tc.input.PublishedYear)
			assert.Equal(t, want, saved.String())
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetBook_NotFound – fetching a non-existent id returns an error
// ---------------------------------------------------------------------------

func TestGetBook_NotFound(t *testing.T) {
	skipIfNoDB(t)

	db := newTestDB(t)
	ctx := context.Background()

	_, err := getBook(ctx, db, 999999999)
	assert.Error(t, err, "getBook with non-existent id should return an error")
}

// ---------------------------------------------------------------------------
// TestAutoIncrementIds – every insert produces a unique id
// ---------------------------------------------------------------------------

func TestAutoIncrementIds(t *testing.T) {
	skipIfNoDB(t)

	db := newTestDB(t)
	ctx := context.Background()

	books := []internal.Book{
		{Title: "Book A", Author: "Author A", PublishedYear: 2001},
		{Title: "Book B", Author: "Author B", PublishedYear: 2002},
		{Title: "Book C", Author: "Author C", PublishedYear: 2003},
	}

	ids := make(map[int64]struct{})
	for _, b := range books {
		saved, err := insertBook(ctx, db, b)
		require.NoError(t, err)
		assert.NotZero(t, saved.ID)

		_, dup := ids[saved.ID]
		