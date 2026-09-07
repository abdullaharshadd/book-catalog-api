package tests

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"migrated-app/internal/model"
	"migrated-app/internal/database"
)

// MIGRATION_NOTE: Ensure the Book model implements the String() method to provide the expected string representation.
func TestCreateBook(t *testing.T) {
	db, err := database.GetDb()
	if err != nil {
		t.Fatalf("Failed to get DB: %v", err)
	}

	book := model.NewBook("Test Book", "Test Author", 2023, strPtr("A test book summary"))
	err = db.Create(book).Error
	if err != nil {
		t.Fatalf("Failed to create book: %v", err)
	}

	assert.NotNil(t, book.ID)
	assert.Equal(t, "Test Book", book.Title)
	assert.Equal(t, "Test Author", book.Author)
	assert.Equal(t, 2023, book.PublishedYear)
	assert.Equal(t, "A test book summary", *book.Summary)
}

func TestCreateBookWithoutSummary(t *testing.T) {
	db, err := database.GetDb()
	if err != nil {
		t.Fatalf("Failed to get DB: %v", err)
	}

	book := model.NewBook("Test Book No Summary", "Test Author", 2023, nil)
	err = db.Create(book).Error
	if err != nil {
		t.Fatalf("Failed to create book: %v", err)
	}

	assert.NotNil(t, book.ID)
	assert.Equal(t, "Test Book No Summary", book.Title)
	assert.Equal(t, "Test Author", book.Author)
	assert.Equal(t, 2023, book.PublishedYear)
	assert.Nil(t, book.Summary)
}

func TestBookStringRepresentation(t *testing.T) {
	db, err := database.GetDb()
	if err != nil {
		t.Fatalf("Failed to get DB: %v", err)
	}

	book := model.NewBook("Repr Test", "Repr Author", 2023, strPtr("Test summary"))
	err = db.Create(book).Error
	if err != nil {
		t.Fatalf("Failed to create book: %v", err)
	}

	expectedStr := "<Book(id=" + book.ID.String() + ", title='Repr Test', author='Repr Author', year=2023)>"
	assert.Equal(t, expectedStr, book.String())
}

func TestBookStringMethod(t *testing.T) {
	db, err := database.GetDb()
	if err != nil {
		t.Fatalf("Failed to get DB: %v", err)
	}

	book := model.NewBook("String Test", "String Author", 2023, nil)
	err = db.Create(book).Error
	if err != nil {
		t.Fatalf("Failed to create book: %v", err)
	}

	expectedStr := "String Test by String Author (2023)"
	assert.Equal(t, expectedStr, book.String())
}

func TestUniqueConstraintViolation(t *testing.T) {
	db, err := database.GetDb()
	if err != nil {
		t.Fatalf("Failed to get DB: %v", err)
	}

	book1 := model.NewBook("Duplicate Test", "Duplicate Author", 2023, nil)
	err = db.Create(book1).Error
	if err != nil {
		t.Fatalf("Failed to create book1: %v", err)
	}

	book2 := model.NewBook("Duplicate Test", "Duplicate Author", 2024, nil)
	err = db.Create(book2).Error
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("Expected ErrDuplicatedKey, got: %v", err)
	}
}

func TestBooksWithSameTitleDifferentAuthors(t *testing.T) {
	db, err := database.GetDb()
	if err != nil {
		t.Fatalf("Failed to get DB: %v", err)
	}

	book1 := model.NewBook("Common Title", "Author One", 2023, nil)
	book2 := model.NewBook("Common Title", "Author Two", 2023, nil)

	err = db.Create(book1).Error
	if err != nil {
		t.Fatalf("Failed to create book1: %v", err)
	}

	err = db.Create(book2).Error
	if err != nil {
		t.Fatalf("Failed to create book2: %v", err)
	}

	assert.NotEqual(t, book1.ID, book2.ID)
}

func TestBooksWithSameAuthorDifferentTitles(t *testing.T) {
	db, err := database.GetDb()
	if err != nil {
		t.Fatalf("Failed to get DB: %v", err)
	}

	book1 := model.NewBook("First Book", "Prolific Author", 2023, nil)
	book2 := model.NewBook("Second Book", "Prolific Author", 2024, nil)

	err = db.Create(book1).Error
	if err != nil {
		t.Fatalf("Failed to create book1: %v", err)
	}

	err = db.Create(book2).Error
	if err != nil {
		t.Fatalf("Failed to create book2: %v", err)
	}

	assert.NotEqual(t, book1.ID, book2.ID)
}

func strPtr(s string) *string {
	return &s
}