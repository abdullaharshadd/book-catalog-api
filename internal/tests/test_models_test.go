package tests

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"migrated-app/internal/model"
	"migrated-app/internal/database"
)

var createBookTests = []struct {
	name          string
	title         string
	author        string
	publishedYear int
	summary       *string
	expectedErr   error
}{
	{"with summary", "Test Book", "Test Author", 2023, strPtr("A test book summary"), nil},
	{"without summary", "Test Book No Summary", "Test Author", 2023, nil, nil},
}

func TestCreateBook(t *testing.T) {
	db, err := database.GetDb()
	if err != nil {
		t.Fatalf("Failed to get DB: %v", err)
	}

	for _, tt := range createBookTests {
		t.Run(tt.name, func(t *testing.T) {
			book := model.NewBook(tt.title, tt.author, tt.publishedYear, tt.summary)
			err := db.Create(book).Error
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, book.ID)
				assert.Equal(t, tt.title, book.Title)
				assert.Equal(t, tt.author, book.Author)
				assert.Equal(t, tt.publishedYear, book.PublishedYear)
				if tt.summary != nil {
					assert.Equal(t, *tt.summary, *book.Summary)
				} else {
					assert.Nil(t, book.Summary)
				}
			}
		})
	}
}

var bookReprTests = []struct {
	name          string
	title         string
	author        string
	publishedYear int
	summary       *string
}{
	{"with summary", "Repr Test", "Repr Author", 2023, strPtr("Test summary")},
	{"without summary", "String Test", "String Author", 2023, nil},
}

func TestBookStringRepresentation(t *testing.T) {
	db, err := database.GetDb()
	if err != nil {
		t.Fatalf("Failed to get DB: %v", err)
	}

	for _, tt := range bookReprTests {
		t.Run(tt.name, func(t *testing.T) {
			book := model.NewBook(tt.title, tt.author, tt.publishedYear, tt.summary)
			err := db.Create(book).Error
			if err != nil {
				t.Fatalf("Failed to create book: %v", err)
			}

			expectedStr := ""
			if tt.summary == nil {
				expectedStr = tt.title + " by " + tt.author + " (" + strconv.Itoa(tt.publishedYear) + ")"
			} else {
				expectedStr = "<Book(id=" + book.ID.String() + ", title='" + tt.title + "', author='" + tt.author + "', year=" + strconv.Itoa(tt.publishedYear) + ")>"
			}
			assert.Equal(t, expectedStr, book.String())
		})
	}
}

var uniqueConstraintViolationTests = []struct {
	name          string
	title         string
	author        string
	publishedYear int
	summary       *string
}{
	{"duplicate title-author", "Duplicate Test", "Duplicate Author", 2024, nil},
}

func TestUniqueConstraintViolation(t *testing.T) {
	db, err := database.GetDb()
	if err != nil {
		t.Fatalf("Failed to get DB: %v", err)
	}

	for _, tt := range uniqueConstraintViolationTests {
		t.Run(tt.name, func(t *testing.T) {
			book1 := model.NewBook("Duplicate Test", "Duplicate Author", 2023, nil)
			err := db.Create(book1).Error
			if err != nil {
				t.Fatalf("Failed to create book1: %v", err)
			}

			book2 := model.NewBook(tt.title, tt.author, tt.publishedYear, tt.summary)
			err = db.Create(book2).Error
			if !errors.Is(err, gorm.ErrDuplicatedKey) {
				t.Fatalf("Expected ErrDuplicatedKey, got: %v", err)
			}
		})
	}
}

var booksWithSameTitleDifferentAuthorsTests = []struct {
	name          string
	title         string
	author        string
	publishedYear int
}{
	{"same title different authors", "Common Title", "Author One", 2023},
	{"same title different authors", "Common Title", "Author Two", 2023},
}

func TestBooksWithSameTitleDifferentAuthors(t *testing.T) {
	db, err := database.GetDb()
	if err != nil {
		t.Fatalf("Failed to get DB: %v", err)
	}

	var books []*model.Book
	for _, tt := range booksWithSameTitleDifferentAuthorsTests {
		t.Run(tt.name, func(t *testing.T) {
			book := model.NewBook(tt.title, tt.author, tt.publishedYear, nil)
			err := db.Create(book).Error
			if err != nil {
				t.Fatalf("Failed to create book: %v", err)
			}
			books = append(books, book)
		})
	}

	assert.NotEqual(t, books[0].ID, books[1].ID)
}

var booksWithSameAuthorDifferentTitlesTests = []struct {
	name          string
	title         string
	author        string
	publishedYear int
}{
	{"same author different titles", "First Book", "Prolific Author", 2023},
	{"same author different titles", "Second Book", "Prolific Author", 2024},
}

func TestBooksWithSameAuthorDifferentTitles(t *testing.T) {
	db, err := database.GetDb()
	if err != nil {
		t.Fatalf("Failed to get DB: %v", err)
	}

	var books []*model.Book
	for _, tt := range booksWithSameAuthorDifferentTitlesTests {
		t.Run(tt.name, func(t *testing.T) {
			book := model.NewBook(tt.title, tt.author, tt.publishedYear, nil)
			err := db.Create(book).Error
			if err != nil {
				t.Fatalf("Failed to create book: %v", err)
			}
			books = append(books, book)
		})
	}

	assert.NotEqual(t, books[0].ID, books[1].ID)
}

func strPtr(s string) *string {
	return &s
}