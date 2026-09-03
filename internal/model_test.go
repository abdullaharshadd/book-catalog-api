```go
package internal

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestCreateBook(t *testing.T) {
	tests := []struct {
		name       string
		book       Book
		wantErr    bool
		wantID     int
		dbMockRows sqlmock.Rows
		dbMockErr  error
	}{
		{
			name: "create book success",
			book: Book{
				Title:          "Sample Book",
				Author:         "Author Name",
				PublishedYear:  2023,
				Summary:        sql.NullString{String: "Sample Summary", Valid: true},
			},
			wantErr: false,
			wantID:  1,
			dbMockRows: sqlmock.NewRows([]string{"id"}).AddRow(1),
			dbMockErr: nil,
		},
		{
			name: "duplicate book error",
			book: Book{
				Title:          "Duplicate Book",
				Author:         "Duplicate Author",
				PublishedYear:  2023,
				Summary:        sql.NullString{String: "Sample Summary", Valid: true},
			},
			wantErr: true,
			wantID:  0,
			dbMockRows: nil,
			dbMockErr: pq.ErrDuplicateKey,
		},
		{
			name: "generic db error",
			book: Book{
				Title:          "Error Book",
				Author:         "Error Author",
				PublishedYear:  2023,
				Summary:        sql.NullString{String: "Sample Summary", Valid: true},
			},
			wantErr: true,
			wantID:  0,
			dbMockRows: nil,
			dbMockErr: errors.New("generic database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO books .+ RETURNING id`).WillReturnRows(tt.dbMockRows).WillReturnError(tt.dbMockErr)

			dbx := sqlx.NewDb(db, "sqlmock")
			ctx := context.Background()

			gotID, err := CreateBook(ctx, dbx, tt.book)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantID, gotID)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestGetBook(t *testing.T) {
	tests := []struct {
		name       string
		id         int
		wantBook   *Book
		wantErr    bool
		dbMockRows sqlmock.Rows
		dbMockErr  error
	}{
		{
			name: "get book success",
			id:   1,
			wantBook: &Book{
				ID:             1,
				Title:          "Sample Book",
				Author:         "Author Name",
				PublishedYear:  2023,
				Summary:        sql.NullString{String: "Sample Summary", Valid: true},
			},
			wantErr: false,
			dbMockRows: sqlmock.NewRows([]string{"id", "title", "author", "published_year", "summary"}).AddRow(1, "Sample Book", "Author Name", 2023, sql.NullString{String: "Sample Summary", Valid: true}),
			dbMockErr: nil,
		},
		{
			name: "get non-existing book",
			id:   2,
			wantBook: nil,
			wantErr: false,
			dbMockRows: nil,
			dbMockErr: sql.ErrNoRows,
		},
		{
			name: "generic db error",
			id:   3,
			wantBook: nil,
			wantErr: true,
			dbMockRows: nil,
			dbMockErr: errors.New("generic database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			mock.ExpectQuery(`SELECT id, title, author, published_year, summary FROM books WHERE id = \d+`).WillReturnRows(tt.dbMockRows).WillReturnError(tt.dbMockErr)

			dbx := sqlx.NewDb(db, "sqlmock")
			ctx := context.Background()

			gotBook, err := GetBook(ctx, dbx, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantBook, gotBook)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestListBooks(t *testing.T) {
	tests := []struct {
		name       string
		wantBooks  []Book
		wantErr    bool
		dbMockRows sqlmock.Rows
		dbMockErr  error
	}{
		{
			name: "list books success",
			wantBooks: []Book{
				{ID: 1, Title: "Book 1", Author: "Author 1", PublishedYear: 2020, Summary: sql.NullString{String: "Summary 1", Valid: true}},
				{ID: 2, Title: "Book 2", Author: "Author 2", PublishedYear: 2021, Summary: sql.NullString{String: "Summary 2", Valid: true}},
			},
			wantErr: false,
			dbMockRows: sqlmock.NewRows([]string{"id", "title", "author", "published_year", "summary"}).
				AddRow(1, "Book 1", "Author 1", 2020, sql.NullString{String: "Summary 1", Valid: true}).
				AddRow(2, "Book 2", "Author 2", 2021, sql.NullString{String: "Summary 2", Valid: true}),
			dbMockErr: nil,
		},
		{
			name: "generic db error",
			wantBooks: nil,
			wantErr: true,
			dbMockRows: nil,
			dbMockErr: errors.New("generic database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			mock.ExpectQuery(`SELECT id, title, author, published_year, summary FROM books`).WillReturnRows(tt.dbMockRows).WillReturnError(tt.dbMockErr)

			dbx := sqlx.NewDb(db, "sqlmock")
			ctx := context.Background()

			gotBooks, err := ListBooks(ctx, dbx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantBooks, gotBooks)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestDeleteBook(t *testing.T) {
	tests := []struct {
		name       string
		id         int
		wantErr    bool
		dbMockErr  error
	}{
		{
			name: "delete book success",
			id:   1,
			wantErr: false,
			dbMockErr: nil,
		},
		{
			name: "generic db error",
			id:   2,
			wantErr: true,
			dbMockErr: errors.New("generic database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			mock.ExpectExec(`DELETE FROM books WHERE id = \d+`).WillReturnResult(sqlmock.NewResult(1, 1)).WillReturnError(tt.dbMockErr)

			dbx := sqlx.NewDb(db, "sqlmock")
			ctx := context.Background()

			err = DeleteBook(ctx, dbx, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestUpdateBook(t *testing.T) {
	tests := []struct {
		name       string
		book       Book
		wantErr    bool
		dbMockErr  error
	}{
		{
			name: "update book success",
			book: Book{
				ID:             1,
				Title:          "Updated Book",
				Author:         "Updated Author",
				PublishedYear:  2024,
				Summary:        sql.NullString{String: "Updated Summary", Valid: true},
			},
			wantErr: false,
			dbMockErr: nil,
		},
		{
			name: "generic db error",
			book: Book{
				ID:             2,
				Title:          "Error Book",
				Author:         "Error Author",
				PublishedYear:  2025,
				Summary:        sql.NullString{String: "Error Summary", Valid: true},
			},
			wantErr: true,
			dbMockErr: errors.New("generic database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			mock.ExpectExec(`UPDATE books SET title = .+, author = .+, published_year = \d+, summary = .+ WHERE id = \d+`).WillReturnResult(sqlmock.NewResult(1, 1)).WillReturnError(tt.dbMockErr)

			dbx := sqlx.NewDb(db, "sqlmock")
			ctx := context.Background()

			err = UpdateBook(ctx, dbx, tt.book)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
```