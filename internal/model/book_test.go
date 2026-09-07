package model

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"reflect"
	"strings"
)

func TestNewBook(t *testing.T) {
	type args struct {
		title         string
		author        string
		publishedYear int
		summary       *string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"valid inputs", args{"The Great Gatsby", "F. Scott Fitzgerald", 1925, nil}, false},
		{"valid inputs with summary", args{"The Great Gatsby", "F. Scott Fitzgerald", 1925, strPtr("A classic novel about the American Dream.")}, false},
		{"title too long", args{strings.Repeat("a", 256), "F. Scott Fitzgerald", 1925, nil}, true},
		{"author too long", args{"The Great Gatsby", strings.Repeat("a", 256), 1925, nil}, true},
		{"nil title", args{"", "F. Scott Fitzgerald", 1925, nil}, true},
		{"nil author", args{"The Great Gatsby", "", 1925, nil}, true},
		{"nil published year", args{"The Great Gatsby", "F. Scott Fitzgerald", 0, nil}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewBook(tt.args.title, tt.args.author, tt.args.publishedYear, tt.args.summary)
			if tt.wantErr {
				assert.Nil(t, got, "NewBook should return nil when inputs are invalid")
			} else {
				assert.NotNil(t, got, "NewBook should not return nil when inputs are valid")
				assert.Equal(t, tt.args.title, got.Title)
				assert.Equal(t, tt.args.author, got.Author)
				assert.Equal(t, tt.args.publishedYear, got.PublishedYear)
				if tt.args.summary != nil {
					assert.NotNil(t, got.Summary)
					assert.Equal(t, *tt.args.summary, *got.Summary)
				} else {
					assert.Nil(t, got.Summary)
				}
			}
		})
	}
}

func TestTableName(t *testing.T) {
	b := Book{}
	assert.Equal(t, "books", b.TableName())
}

func TestInitializeDatabase(t *testing.T) {
	db := &mockDB{}
	err := InitializeDatabase(db)
	assert.NoError(t, err)
	assert.True(t, db.migrated, "InitializeDatabase should call AutoMigrate")
}

type mockDB struct {
	migrated bool
}

func (m *mockDB) AutoMigrate(models ...interface{}) error {
	m.migrated = true
	return nil
}

func TestBookRepr(t *testing.T) {
	b := Book{ID: 1, Title: "The Great Gatsby", Author: "F. Scott Fitzgerald", PublishedYear: 1925}
	assert.Equal(t, "<Book(id=1, title='The Great Gatsby', author='F. Scott Fitzgerald', year=1925)>", b.Repr())
}

func TestBookStr(t *testing.T) {
	b := Book{Title: "The Great Gatsby", Author: "F. Scott Fitzgerald", PublishedYear: 1925}
	assert.Equal(t, "The Great Gatsby by F. Scott Fitzgerald (1925)", b.Str())
}

func strPtr(s string) *string {
	return &s
}

func (b *Book) Repr() string {
	return reflect.Sprintf("<Book(id=%d, title='%s', author='%s', year=%d)>", b.ID, b.Title, b.Author, b.PublishedYear)
}

func (b *Book) Str() string {
	return reflect.Sprintf("%s by %s (%d)", b.Title, b.Author, b.PublishedYear)
}