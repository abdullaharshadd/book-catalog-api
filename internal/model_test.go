// internal/model_test.go

package model

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestNewBook(t *testing.T) {
	type args struct {
		title         string
		author        string
		publishedYear int
		summary       string
	}
	tests := []struct {
		name    string
		args    args
		want    *Book
		wantErr bool
	}{
		{"valid inputs", args{"Title", "Author", 2020, "Summary"}, &Book{Title: "Title", Author: "Author", PublishedYear: 2020, Summary: "Summary"}, false},
		{"nil title", args{"", "Author", 2020, "Summary"}, nil, true},
		{"nil author", args{"Title", "", 2020, "Summary"}, nil, true},
		{"nil published year", args{"Title", "Author", 0, "Summary"}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewBook(tt.args.title, tt.args.author, tt.args.publishedYear, tt.args.summary)
			if (got == nil) != tt.wantErr {
				t.Errorf("NewBook() error = %v, wantErr %v", got == nil, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInsert(t *testing.T) {
	type args struct {
		ctx   context.Context
		db    *sqlx.DB
		book  *Book
	}
	tests := []struct {
		name    string
		args    args
		want    *Book
		wantErr bool
	}{
		{"valid insert", args{context.Background(), nil, &Book{Title: "Title", Author: "Author", PublishedYear: 2020, Summary: "Summary"}}, &Book{ID: 1, Title: "Title", Author: "Author", PublishedYear: 2020, Summary: "Summary"}, false},
		{"duplicate title and author", args{context.Background(), nil, &Book{Title: "ExistingTitle", Author: "ExistingAuthor", PublishedYear: 2020, Summary: "Summary"}}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "sqlmock")
			tt.args.db = sqlxDB

			if tt.wantErr {
				mock.ExpectExec("INSERT INTO books").
					WithArgs(tt.args.book.Title, tt.args.book.Author, tt.args.book.PublishedYear, tt.args.book.Summary).
					WillReturnError(sql.ErrNoRows)
			} else {
				mock.ExpectExec("INSERT INTO books").
					WithArgs(tt.args.book.Title, tt.args.book.Author, tt.args.book.PublishedYear, tt.args.book.Summary).
					WillReturnResult(sqlmock.NewResult(1, 1))
			}

			got, err := Insert(tt.args.ctx, tt.args.db, tt.args.book)
			if (err != nil) != tt.wantErr {
				t.Errorf("Insert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUpdate(t *testing.T) {
	type args struct {
		ctx  context.Context
		db   *sqlx.DB
		book *Book
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"valid update", args{context.Background(), nil, &Book{ID: 1, Title: "UpdatedTitle", Author: "UpdatedAuthor", PublishedYear: 2021, Summary: "UpdatedSummary"}}, false},
		{"invalid book id", args{context.Background(), nil, &Book{ID: 0, Title: "UpdatedTitle", Author: "UpdatedAuthor", PublishedYear: 2021, Summary: "UpdatedSummary"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "sqlmock")
			tt.args.db = sqlxDB

			if tt.wantErr {
				mock.ExpectExec("UPDATE books").
					WithArgs(tt.args.book.Title, tt.args.book.Author, tt.args.book.PublishedYear, tt.args.book.Summary, tt.args.book.ID).
					WillReturnError(sql.ErrNoRows)
			} else {
				mock.ExpectExec("UPDATE books").
					WithArgs(tt.args.book.Title, tt.args.book.Author, tt.args.book.PublishedYear, tt.args.book.Summary, tt.args.book.ID).
					WillReturnResult(sqlmock.NewResult(1, 1))
			}

			err = Update(tt.args.ctx, tt.args.db, tt.args.book)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDelete(t *testing.T) {
	type args struct {
		ctx context.Context
		db  *sqlx.DB
		id  int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"valid delete", args{context.Background(), nil, 1}, false},
		{"invalid book id", args{context.Background(), nil, 0}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "sqlmock")
			tt.args.db = sqlxDB

			if tt.wantErr {
				mock.ExpectExec("DELETE FROM books").
					WithArgs(tt.args.id).
					WillReturnError(sql.ErrNoRows)
			} else {
				mock.ExpectExec("DELETE FROM books").
					WithArgs(tt.args.id).
					WillReturnResult(sqlmock.NewResult(1, 1))
			}

			err = Delete(tt.args.ctx, tt.args.db, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestGetByID(t *testing.T) {
	type args struct {
		ctx context.Context
		db  *sqlx.DB
		id  int
	}
	tests := []struct {
		name    string
		args    args
		want    *Book
		wantErr bool
	}{
		{"valid id", args{context.Background(), nil, 1}, &Book{ID: 1, Title: "Title", Author: "Author", PublishedYear: 2020, Summary: "Summary"}, false},
		{"invalid id", args{context.Background(), nil, 0}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "sqlmock")
			tt.args.db = sqlxDB

			rows := sqlmock.NewRows([]string{"id", "title", "author", "published_year", "summary"}).
				AddRow(tt.want.ID, tt.want.Title, tt.want.Author, tt.want.PublishedYear, tt.want.Summary)
			mock.ExpectQuery("SELECT \\* FROM books WHERE id").
				WithArgs(tt.args.id).
				WillReturnRows(rows)

			got, err := GetByID(tt.args.ctx, tt.args.db, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBook_String(t *testing.T) {
	type fields struct {
		ID            int
		Title         string
		Author        string
		PublishedYear int
		Summary       string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"valid book", fields{1, "Title", "Author", 2020, "Summary"}, "Title by Author (2020)"},
		{"empty summary", fields{1, "Title", "Author", 2020, ""}, "Title by Author (2020)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Book{
				ID:            tt.fields.ID,
				Title:         tt.fields.Title,
				Author:        tt.fields.Author,
				PublishedYear: tt.fields.PublishedYear,
				Summary:       tt.fields.Summary,
			}
			got := b.String()
			assert.Equal(t, tt.want, got)
		})
	}
}