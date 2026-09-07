// Package tests contains the unit tests for models and schemas, and integration tests for API endpoints of the Book Catalog API.
package tests

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"migrated-app/internal/database"
	"migrated-app/internal/model"
	"migrated-app/internal/schemas"
)

var bookCreationTests = []struct {
	name          string
	title         string
	author        string
	publishedYear int
	summary       sql.NullString
	wantErr       bool
}{
	{"WithSummary", "Test Book", "Test Author", 2023, sql.NullString{String: "A test book summary", Valid: true}, false},
	{"WithoutSummary", "Test Book No Summary", "Test Author", 2023, sql.NullString{Valid: false}, false},
	{"DuplicateTitleAndAuthor", "Duplicate Test", "Duplicate Author", 2024, sql.NullString{Valid: false}, true},
}

func TestCreateBook(t *testing.T) {
	ctx := context.Background()

	db, err := database.GetSyncDB()
	if err != nil {
		t.Fatalf("failed to get database: %v", err)
	}
	defer database.CloseSyncDB(db)

	for _, tt := range bookCreationTests {
		t.Run(tt.name, func(t *testing.T) {
			book := &model.Book{
				Title:          tt.title,
				Author:         tt.author,
				PublishedYear:  tt.publishedYear,
				Summary:        tt.summary,
			}

			err := book.Create(ctx, db)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, 0, book.ID)
				assert.Equal(t, tt.title, book.Title)
				assert.Equal(t, tt.author, book.Author)
				assert.Equal(t, tt.publishedYear, book.PublishedYear)
				assert.Equal(t, tt.summary, book.Summary)
			}
		})
	}
}

func TestBookRepr(t *testing.T) {
	ctx := context.Background()

	db, err := database.GetSyncDB()
	if err != nil {
		t.Fatalf("failed to get database: %v", err)
	}
	defer database.CloseSyncDB(db)

	book := &model.Book{
		Title:          "Repr Test",
		Author:         "Repr Author",
		PublishedYear:  2023,
		Summary:        sql.NullString{String: "Test summary", Valid: true},
	}

	err = book.Create(ctx, db)
	if err != nil {
		t.Fatalf("failed to create book: %v", err)
	}

	expectedRepr := "<Book(id=" + strconv.Itoa(int(book.ID)) + ", title='Repr Test', author='Repr Author', year=2023)>"
	assert.Equal(t, expectedRepr, book.TableName())
}

func TestBookStr(t *testing.T) {
	ctx := context.Background()

	db, err := database.GetSyncDB()
	if err != nil {
		t.Fatalf("failed to get database: %v", err)
	}
	defer database.CloseSyncDB(db)

	book := &model.Book{
		Title:          "String Test",
		Author:         "String Author",
		PublishedYear:  2023,
		Summary:        sql.NullString{Valid: false},
	}

	err = book.Create(ctx, db)
	if err != nil {
		t.Fatalf("failed to create book: %v", err)
	}

	expectedStr := "String Test by String Author (2023)"
	assert.Equal(t, expectedStr, book.String())
}

func TestBooksSameTitleDifferentAuthors(t *testing.T) {
	ctx := context.Background()

	db, err := database.GetSyncDB()
	if err != nil {
		t.Fatalf("failed to get database: %v", err)
	}
	defer database.CloseSyncDB(db)

	book1 := &model.Book{
		Title:          "Common Title",
		Author:         "Author One",
		PublishedYear:  2023,
		Summary:        sql.NullString{Valid: false},
	}

	book2 := &model.Book{
		Title:          "Common Title",
		Author:         "Author Two",
		PublishedYear:  2023,
		Summary:        sql.NullString{Valid: false},
	}

	err = book1.Create(ctx, db)
	assert.NoError(t, err)

	err = book2.Create(ctx, db)
	assert.NoError(t, err)

	assert.NotEqual(t, book1.ID, book2.ID)
}

func TestBooksSameAuthorDifferentTitles(t *testing.T) {
	ctx := context.Background()

	db, err := database.GetSyncDB()
	if err != nil {
		t.Fatalf("failed to get database: %v", err)
	}
	defer database.CloseSyncDB(db)

	book1 := &model.Book{
		Title:          "First Book",
		Author:         "Prolific Author",
		PublishedYear:  2023,
		Summary:        sql.NullString{Valid: false},
	}

	book2 := &model.Book{
		Title:          "Second Book",
		Author:         "Prolific Author",
		PublishedYear:  2024,
		Summary:        sql.NullString{Valid: false},
	}

	err = book1.Create(ctx, db)
	assert.NoError(t, err)

	err = book2.Create(ctx, db)
	assert.NoError(t, err)

	assert.NotEqual(t, book1.ID, book2.ID)
}