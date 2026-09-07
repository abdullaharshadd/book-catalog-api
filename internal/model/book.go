package model

import (
	"gorm.io/gorm"
	"gorm.io/datatypes"
)

// Book represents a book in the catalog.
//
// Fields:
//   ID: Primary key, auto-incrementing integer.
//   Title: Book title (required).
//   Author: Book author (required).
//   PublishedYear: Year the book was published (required).
//   Summary: Optional summary/description of the book.
type Book struct {
	ID            uint           `gorm:\"primaryKey;autoIncrement\" json:\"id\"`
	Title         string         `gorm:\"uniqueIndex:idx_unique_title_author;not null\" json:\"title\"`
	Author        string         `gorm:\"uniqueIndex:idx_unique_title_author;not null\" json:\"author\"`
	PublishedYear int            `gorm:\"index;not null\" json:\"published_year\"`
	Summary       *datatypes.Text `gorm:\"type:text\" json:\"summary,omitempty\"`
}

// TableName overrides the table name used by Book to \`books\`.
func (Book) TableName() string {
	return "books"
}

// NewBook creates a new instance of Book.
//
// Parameters:
//   title: The title of the book.
//   author: The author of the book.
//   publishedYear: The year the book was published.
//   summary: An optional summary of the book.
//
// Returns:
//   A pointer to a new Book instance.
func NewBook(title string, author string, publishedYear int, summary *string) *Book {
	var bookSummary *datatypes.Text
	if summary != nil {
		bookSummary = datatypes.NewText(*summary)
	}
	return &Book{
		Title:         title,
		Author:        author,
		PublishedYear: publishedYear,
		Summary:       bookSummary,
	}
}

// InitializeDatabase initializes the database and creates the schema.
//
// Parameters:
//   db: The gorm database connection.
//
// Returns:
//   An error if the schema creation fails.
func InitializeDatabase(db *gorm.DB) error {
	err := db.AutoMigrate(&Book{})
	if err != nil {
		return err
	}
	return nil
}

// MIGRATION_NOTE: The uniqueness constraint for the combination of title and author is preserved using the `gorm` tag `uniqueIndex:idx_unique_title_author`.