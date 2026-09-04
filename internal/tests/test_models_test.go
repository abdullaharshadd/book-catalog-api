```go
package tests

import (
	"context"
	"testing"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"internal"
	"internal/database"
	"internal/model"
	"internal/schemas"
	"internal/conftest"
	"database/sql"
	"fmt"
)

type bookTestCase struct {
	name                  string
	input                 *schemas.BookCreate
	expectedErrorContains string
	expectedBook          *model.Book
}

func TestCreateBook(t *testing.T) {
	testCases := []bookTestCase{
		{
			name: "Create book with all fields",
			input: &schemas.BookCreate{
				Title:         "Test Book",
				Author:       "Test Author",
				PublishedYear: 2023,
				Summary:      "A test book summary",
			},
			expectedBook: &model.Book{
				Title:         "Test Book",
				Author:       "Test Author",
				PublishedYear: 2023,
				Summary:      "A test book summary",
			},
		},
		{
			name: "Create book without summary",
			input: &schemas.BookCreate{
				Title:         "Test Book No Summary",
				Author:       "Test Author",
				PublishedYear: 2023,
			},
			expectedBook: &model.Book{
				Title:         "Test Book No Summary",
				Author:       "Test Author",
				PublishedYear: 2023,
				Summary:      nil,
			},
		},
		{
			name: "Create duplicate book",
			input: &schemas.BookCreate{
				Title:         "Duplicate Book",
				Author:       "Duplicate Author",
				PublishedYear: 2024,
			},
			expectedErrorContains: "duplicate key value violates unique constraint",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := database.NewDatabase()
			require.NoError(t, err)
			db = conftest.GetSyncDB(t, db)
			defer db.Close()

			conftest.DropAllTables(context.Background(), db)
			InitDB(context.Background(), db)

			if tc.expectedErrorContains == "" {
				book, err := model.CreateBook(context.Background(), db, tc.input)
				require.NoError(t, err)
				assert.NotNil(t, book)
				assert.Equal(t, tc.expectedBook.Title, book.Title)
				assert.Equal(t, tc.expectedBook.Author, book.Author)
				assert.Equal(t, tc.expectedBook.PublishedYear, book.PublishedYear)
				assert.Equal(t, tc.expectedBook.Summary, book.Summary)
			} else {
				_, err := model.CreateBook(context.Background(), db, &schemas.BookCreate{
					Title:         "Duplicate Book",
					Author:       "Duplicate Author",
					PublishedYear: 2023,
				})
				require.NoError(t, err)

				_, err = model.CreateBook(context.Background(), db, tc.input)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrorContains)
			}
		})
	}
}

func TestBookRepr(t *testing.T) {
	db, err := database.NewDatabase()
	require.NoError(t, err)
	db = conftest.GetSyncDB(t, db)
	defer db.Close()

	conftest.DropAllTables(context.Background(), db)
	InitDB(context.Background(), db)

	bookCreate := schemas.BookCreate{
		Title:         "Repr Test",
		Author:       "Repr Author",
		PublishedYear: 2023,
		Summary:      "Test summary",
	}
	book, err := model.CreateBook(context.Background(), db, &bookCreate)
	require.NoError(t, err)
	assert.NotNil(t, book)

	expectedRepr := fmt.Sprintf("<Book(id=%d, title='Repr Test', author='Repr Author', year=2023)>", book.ID)
	assert.Equal(t, expectedRepr, book.String())
}

func TestBookStr(t *testing.T) {
	db, err := database.NewDatabase()
	require.NoError(t, err)
	db = conftest.GetSyncDB(t, db)
	defer db.Close()

	conftest.DropAllTables(context.Background(), db)
	InitDB(context.Background(), db)

	bookCreate := schemas.BookCreate{
		Title:         "String Test",
		Author:       "String Author",
		PublishedYear: 2023,
	}
	book, err := model.CreateBook(context.Background(), db, &bookCreate)
	require.NoError(t, err)
	assert.NotNil(t, book)

	expectedStr := "String Test by String Author (2023)"
	assert.Equal(t, expectedStr, book.String())
}

func TestCreateBooksSameTitleDifferentAuthors(t *testing.T) {
	db, err := database.NewDatabase()
	require.NoError(t, err)
	db = conftest.GetSyncDB(t, db)
	defer db.Close()

	conftest.DropAllTables(context.Background(), db)
	InitDB(context.Background(), db)

	bookCreate1 := schemas.BookCreate{
		Title:         "Common Title",
		Author:       "Author One",
		PublishedYear: 2023,
	}
	book1, err := model.CreateBook(context.Background(), db, &bookCreate1)
	require.NoError(t, err)
	assert.NotNil(t, book1)

	bookCreate2 := schemas.BookCreate{
		Title:         "Common Title",
		Author:       "Author Two",
		PublishedYear: 2023,
	}
	book2, err := model.CreateBook(context.Background(), db, &bookCreate2)
	require.NoError(t, err)
	assert.NotNil(t, book2)

	assert.NotEqual(t, book1.ID, book2.ID)
}

func TestCreateBooksSameAuthorDifferentTitles(t *testing.T) {
	db, err := database.NewDatabase()
	require.NoError(t, err)
	db = conftest.GetSyncDB(t, db)
	defer db.Close()

	conftest.DropAllTables(context.Background(), db)
	InitDB(context.Background(), db)

	bookCreate1 := schemas.BookCreate{
		Title:         "First Book",
		Author:       "Prolific Author",
		PublishedYear: 2023,
	}
	book1, err := model.CreateBook(context.Background(), db, &bookCreate1)
	require.NoError(t, err)
	assert.NotNil(t, book1)

	bookCreate2 := schemas.BookCreate{
		Title:         "Second Book",
		Author:       "Prolific Author",
		PublishedYear: 2024,
	}
	book2, err := model.CreateBook(context.Background(), db, &bookCreate2)
	require.NoError(t, err)
	assert.NotNil(t, book2)

	assert.NotEqual(t, book1.ID, book2.ID)
}
```