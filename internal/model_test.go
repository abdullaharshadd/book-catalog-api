// internal/model_test.go
package internal

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var (
	errMock = errors.New("mock error")
)

type MockDB struct {
	mock.Mock
}

func (m *MockDB) Exec(query string) *gorm.DB {
	args := m.Called(query)
	return args.Get(0).(*gorm.DB)
}

func (m *MockDB) DB() (*sql.DB, error) {
	args := m.Called()
	return args.Get(0).(*sql.DB), args.Error(1)
}

func TestNewBook(t *testing.T) {
	type args struct {
		title, author string
		publishedYear int
		summary       string
	}
	type want struct {
		book *Book
		err  error
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"Valid book", args{"The Great Gatsby", "F. Scott Fitzgerald", 1925, "A classic novel about ..."}, want{&Book{Title: "The Great Gatsby", Author: "F. Scott Fitzgerald", PublishedYear: 1925, Summary: "A classic novel about ..."}, nil}},
		{"Empty title", args{"", "F. Scott Fitzgerald", 1925, "A classic novel about ..."}, want{nil, errors.New("title, author, and publishedYear are required")}},
		{"Empty author", args{"The Great Gatsby", "", 1925, "A classic novel about ..."}, want{nil, errors.New("title, author, and publishedYear are required")}},
		{"Invalid published year", args{"The Great Gatsby", "F. Scott Fitzgerald", -1, "A classic novel about ..."}, want{nil, errors.New("title, author, and publishedYear are required")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewBook(tt.args.title, tt.args.author, tt.args.publishedYear, tt.args.summary)
			if tt.want.err != nil {
				require.EqualError(t, err, tt.want.err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want.book.Title, got.Title)
				assert.Equal(t, tt.want.book.Author, got.Author)
				assert.Equal(t, tt.want.book.PublishedYear, got.PublishedYear)
				assert.Equal(t, tt.want.book.Summary, got.Summary)
			}
		})
	}
}

func TestCreateBooksTable(t *testing.T) {
	type want struct {
		err error
	}
	tests := []struct {
		name string
		want want
	}{
		{"Success", want{nil}},
		{"Failure", want{errMock}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDB)
			if tt.name == "Success" {
				mockDB.On("Exec", mock.Anything).Return(&gorm.DB{})
			} else {
				mockDB.On("Exec", mock.Anything).Return(&gorm.DB{DB: sql.DB{}, Error: errMock})
			}

			err := CreateBooksTable(mockDB)
			if tt.want.err != nil {
				require.EqualError(t, err, tt.want.err.Error())
			} else {
				require.NoError(t, err)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestInitModel(t *testing.T) {
	type want struct {
		err error
	}
	tests := []struct {
		name string
		want want
	}{
		{"Success", want{nil}},
		{"Failure", want{errMock}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDB)
			if tt.name == "Success" {
				mockDB.On("Exec", mock.Anything).Return(&gorm.DB{})
			} else {
				mockDB.On("Exec", mock.Anything).Return(&gorm.DB{DB: sql.DB{}, Error: errMock})
			}

			err := InitModel(mockDB)
			if tt.want.err != nil {
				require.EqualError(t, err, tt.want.err.Error())
			} else {
				require.NoError(t, err)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestBook_String(t *testing.T) {
	book := &Book{
		Title:         "The Great Gatsby",
		Author:        "F. Scott Fitzgerald",
		PublishedYear: 1925,
		Summary:       "A classic novel about ...",
	}

	expectedString := "The Great Gatsby by F. Scott Fitzgerald (1925)"
	assert.Equal(t, expectedString, book.String())
}