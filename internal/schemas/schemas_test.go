// internal/schemas/schemas_test.go

package schemas

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/go-playground/validator/v10"
	"migrated-app/internal/model"
)

func TestBookCreateValidate(t *testing.T) {
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
		{"valid input", args{"The Great Gatsby", "F. Scott Fitzgerald", 1925, strPtr("A classic novel...")}, false},
		{"empty title", args{"", "F. Scott Fitzgerald", 1925, strPtr("A classic novel...")}, true},
		{"long title", args{strings.Repeat("T", 256), "F. Scott Fitzgerald", 1925, strPtr("A classic novel...")}, true},
		{"empty author", args{"The Great Gatsby", "", 1925, strPtr("A classic novel...")}, true},
		{"long author", args{"The Great Gatsby", strings.Repeat("A", 256), 1925, strPtr("A classic novel...")}, true},
		{"invalid published year", args{"The Great Gatsby", "F. Scott Fitzgerald", 999, strPtr("A classic novel...")}, true},
		{"future published year", args{"The Great Gatsby", "F. Scott Fitzgerald", time.Now().Year() + 1, strPtr("A classic novel...")}, true},
		{"long summary", args{"The Great Gatsby", "F. Scott Fitzgerald", 1925, strPtr(strings.Repeat("S", 2001))}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := &BookCreate{
				Title:         tt.args.title,
				Author:        tt.args.author,
				PublishedYear: tt.args.publishedYear,
				Summary:       tt.args.summary,
			}
			if err := bc.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("BookCreate.Validate() error = %v, wantErr %v", err, tt.wantErr)
			} else if !tt.wantErr && bc.Title != strings.TrimSpace(bc.Title) {
				t.Errorf("BookCreate.Validate() did not strip whitespace from title")
			} else if !tt.wantErr && bc.Author != strings.TrimSpace(bc.Author) {
				t.Errorf("BookCreate.Validate() did not strip whitespace from author")
			}
		})
	}
}

func TestBookUpdateValidate(t *testing.T) {
	type args struct {
		title         *string
		author        *string
		publishedYear *int
		summary       *string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"no update", args{nil, nil, nil, nil}, false},
		{"valid title update", args{strPtr("The Great Gatsby"), nil, nil, nil}, false},
		{"empty title update", args{strPtr(" "), nil, nil, nil}, true},
		{"long title update", args{strPtr(strings.Repeat("T", 256)), nil, nil, nil}, true},
		{"valid author update", args{nil, strPtr("F. Scott Fitzgerald"), nil, nil}, false},
		{"empty author update", args{nil, strPtr(" "), nil, nil}, true},
		{"long author update", args{nil, strPtr(strings.Repeat("A", 256)), nil, nil}, true},
		{"valid published year update", args{nil, nil, intPtr(1925), nil}, false},
		{"invalid published year update", args{nil, nil, intPtr(999), nil}, true},
		{"future published year update", args{nil, nil, intPtr(time.Now().Year() + 1), nil}, true},
		{"valid summary update", args{nil, nil, nil, strPtr("A classic novel...")}, false},
		{"long summary update", args{nil, nil, nil, strPtr(strings.Repeat("S", 2001))}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bu := &BookUpdate{
				Title:         tt.args.title,
				Author:        tt.args.author,
				PublishedYear: tt.args.publishedYear,
				Summary:       tt.args.summary,
			}
			if err := bu.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("BookUpdate.Validate() error = %v, wantErr %v", err, tt.wantErr)
			} else if bu.Title != nil && *bu.Title != strings.TrimSpace(*bu.Title) {
				t.Errorf("BookUpdate.Validate() did not strip whitespace from title")
			} else if bu.Author != nil && *bu.Author != strings.TrimSpace(*bu.Author) {
				t.Errorf("BookUpdate.Validate() did not strip whitespace from author")
			}
		})
	}
}

func TestFromModel(t *testing.T) {
	type args struct {
		book *model.Book
	}
	tests := []struct {
		name    string
		args    args
		want    *BookResponse
		wantErr bool
	}{
		{"valid book", args{&model.Book{ID: 1, Title: " The Great Gatsby ", Author: " F. Scott Fitzgerald ", PublishedYear: 1925, Summary: " A classic novel... "}}, &BookResponse{ID: 1, Title: "The Great Gatsby", Author: "F. Scott Fitzgerald", PublishedYear: 1925, Summary: strPtr("A classic novel...")}, false},
		{"no summary", args{&model.Book{ID: 1, Title: " The Great Gatsby ", Author: " F. Scott Fitzgerald ", PublishedYear: 1925, Summary: ""}}, &BookResponse{ID: 1, Title: "The Great Gatsby", Author: "F. Scott Fitzgerald", PublishedYear: 1925, Summary: nil}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromModel(tt.args.book)
			assert.Equal(t, tt.want, got)
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}