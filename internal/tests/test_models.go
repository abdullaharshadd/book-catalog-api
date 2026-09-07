// Package tests contains the unit tests for models and schemas, and integration tests for API endpoints of the Book Catalog API.
package tests

import (
	"context"
	"database/sql"
	"testing"
	"migrated-app/internal/database"
	"migrated-app/internal/model"
	"migrated-app/internal/schemas"
)

func TestCreateBook(t *testing.T) {
	ctx := context.Background()

	db, err := database.GetSyncDB()
	if err != nil {
		t.Fatalf("failed to get database: %v", err)
	}
	defer database.CloseSyncDB(db)

	book := &model.Book{
		Title:          "Test Book",
		Author:         "Test Author",
		PublishedYear:  2023,
		Summary:        sql.NullString{String: "A test book summary", Valid: true},
	}

	err = book.Create(ctx, db)
	if err != nil {
		t.Fatalf("failed to create book: %v", err)
	}

	if book.ID == 0 {
		t.Errorf("expected non-zero ID after creation, got %d", book.ID)
	}
	if book.Title != "Test Book" {
		t.Errorf("expected title 'Test Book', got '%s'", book.Title)
	}
	if book.Author != "Test Author" {
		t.Errorf("expected author 'Test Author', got '%s'", book.Author)
	}
	if book.PublishedYear != 2023 {
		t.Errorf("expected published year 2023, got %d", book.PublishedYear)
	}
	if book.Summary.String != "A test book summary" || !book.Summary.Valid {
		t.Errorf("expected summary 'A test book summary', got '%s'", book.Summary.String)
	}
}

func TestCreateBookWithoutSummary(t *testing.T) {
	ctx := context.Background()

	db, err := database.GetSyncDB()
	if err != nil {
		t.Fatalf("failed to get database: %v", err)
	}
	defer database.CloseSyncDB(db)

	book := &model.Book{
		Title:          "Test Book No Summary",
		Author:         "Test Author",
		PublishedYear:  2023,
		Summary:        sql.NullString{Valid: false},
	}

	err = book.Create(ctx, db)
	if err != nil {
		t.Fatalf("failed to create book: %v", err)
	}

	if book.ID == 0 {
		t.Errorf("expected non-zero ID after creation, got %d", book.ID)
	}
	if book.Title != "Test Book No Summary" {
		t.Errorf("expected title 'Test Book No Summary', got '%s'", book.Title)
	}
	if book.Author != "Test Author" {
		t.Errorf("expected author 'Test Author', got '%s'", book.Author)
	}
	if book.PublishedYear != 2023 {
		t.Errorf("expected published year 2023, got %d", book.PublishedYear)
	}
	if book.Summary.Valid {
		t.Errorf("expected summary to be null, got '%s'", book.Summary.String)
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
	if book.TableName() != expectedRepr {
		t.Errorf("expected repr '%s', got '%s'", expectedRepr, book.TableName())
	}
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
	if book.String() != expectedStr {
		t.Errorf("expected string '%s', got '%s'", expectedStr, book.String())
	}
}

func TestUniqueConstraintViolation(t *testing.T) {
	ctx := context.Background()

	db, err := database.GetSyncDB()
	if err != nil {
		t.Fatalf("failed to get database: %v", err)
	}
	defer database.CloseSyncDB(db)

	book1 := &model.Book{
		Title:          "Duplicate Test",
		Author:         "Duplicate Author",
		PublishedYear:  2023,
		Summary:        sql.NullString{Valid: false},
	}

	err = book1.Create(ctx, db)
	if err != nil {
		t.Fatalf("failed to create book1: %v", err)
	}

	book2 := &model.Book{
		Title:          "Duplicate Test",
		Author:         "Duplicate Author",
		PublishedYear:  2024,
		Summary:        sql.NullString{Valid: false},
	}

	err = book2.Create(ctx, db)
	if err == nil {
		t.Errorf("expected unique constraint violation, got no error")
	}
}

func TestBooksWithSameTitleDifferentAuthors(t *testing.T) {
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
	if err != nil {
		t.Fatalf("failed to create book1: %v", err)
	}

	err = book2.Create(ctx, db)
	if err != nil {
		t.Fatalf("failed to create book2: %v", err)
	}

	if book1.ID == 0 {
		t.Errorf("expected non-zero ID after creation, got %d", book1.ID)
	}
	if book2.ID == 0 {
		t.Errorf("expected non-zero ID after creation, got %d", book2.ID)
	}
	if book1.ID == book2.ID {
		t.Errorf("expected different IDs, got %d and %d", book1.ID, book2.ID)
	}
}

func TestBooksWithSameAuthorDifferentTitles(t *testing.T) {
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
	if err != nil {
		t.Fatalf("failed to create book1: %v", err)
	}

	err = book2.Create(ctx, db)
	if err != nil {
		t.Fatalf("failed to create book2: %v", err)
	}

	if book1.ID == 0 {
		t.Errorf("expected non-zero ID after creation, got %d", book1.ID)
	}
	if book2.ID == 0 {
		t.Errorf("expected non-zero ID after creation, got %d", book2.ID)
	}
	if book1.ID == book2.ID {
		t.Errorf("expected different IDs, got %d and %d", book1.ID, book2.ID)
	}
}