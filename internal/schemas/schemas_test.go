package schemas_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"

	"schemas"
)

var (
	currentYear = time.Now().Year()
)

func TestBookCreateValidate(t *testing.T) {
	type args struct {
		title         string
		author        string
		publishedYear int
		summary       *string
	}
	type want struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"valid input", args{" The Great Gatsby ", " F. Scott Fitzgerald ", 1925, strPtr("A classic novel about the American Dream.")}, want{nil}},
		{"empty title", args{"", " F. Scott Fitzgerald ", 1925, strPtr("A classic novel about the American Dream.")}, want{errors.New("Title cannot be empty")}},
		{"whitespace title", args{"   ", " F. Scott Fitzgerald ", 1925, strPtr("A classic novel about the American Dream.")}, want{errors.New("Title cannot be empty")}},
		{"long title", args{strings.Repeat("a", 256), " F. Scott Fitzgerald ", 1925, strPtr("A classic novel about the American Dream.")}, want{errors.New("ensure this value has at most 255 characters")}},
		{"empty author", args{" The Great Gatsby ", "", 1925, strPtr("A classic novel about the American Dream.")}, want{errors.New("Author cannot be empty")}},
		{"whitespace author", args{" The Great Gatsby ", "   ", 1925, strPtr("A classic novel about the American Dream.")}, want{errors.New("Author cannot be empty")}},
		{"long author", args{" The Great Gatsby ", strings.Repeat("a", 256), 1925, strPtr("A classic novel about the American Dream.")}, want{errors.New("ensure this value has at most 255 characters")}},
		{"year before 1000", args{" The Great Gatsby ", " F. Scott Fitzgerald ", 999, strPtr("A classic novel about the American Dream.")}, want{errors.New("Published year must be after year 1000")}},
		{"year in future", args{" The Great Gatsby ", " F. Scott Fitzgerald ", currentYear + 1, strPtr("A classic novel about the American Dream.")}, want{fmt.Errorf("Published year cannot be in the future (current year: %d)", currentYear)}},
		{"empty summary", args{" The Great Gatsby ", " F. Scott Fitzgerald ", 1925, strPtr("   ")}, want{nil}},
		{"long summary", args{" The Great Gatsby ", " F. Scott Fitzgerald ", 1925, strPtr(strings.Repeat("a", 2001))}, want{errors.New("ensure this value has at most 2000 characters")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := &schemas.BookCreate{
				Title:         tt.args.title,
				Author:        tt.args.author,
				PublishedYear: tt.args.publishedYear,
				Summary:       tt.args.summary,
			}
			err := bc.Validate()
			assert.Equal(t, tt.want.err, err)
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
	type want struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"valid input", args{strPtr(" The Great Gatsby "), strPtr(" F. Scott Fitzgerald "), intPtr(1925), strPtr("A classic novel about the American Dream.")}, want{nil}},
		{"empty title", args{strPtr(""), strPtr(" F. Scott Fitzgerald "), intPtr(1925), strPtr("A classic novel about the American Dream.")}, want{errors.New("Title cannot be empty")}},
		{"whitespace title", args{strPtr("   "), strPtr(" F. Scott Fitzgerald "), intPtr(1925), strPtr("A classic novel about the American Dream.")}, want{errors.New("Title cannot be empty")}},
		{"long title", args{strPtr(strings.Repeat("a", 256)), strPtr(" F. Scott Fitzgerald "), intPtr(1925), strPtr("A classic novel about the American Dream.")}, want{errors.New("ensure this value has at most 255 characters")}},
		{"empty author", args{strPtr(" The Great Gatsby "), strPtr(""), intPtr(1925), strPtr("A classic novel about the American Dream.")}, want{errors.New("Author cannot be empty")}},
		{"whitespace author", args{strPtr(" The Great Gatsby "), strPtr("   "), intPtr(1925), strPtr("A classic novel about the American Dream.")}, want{errors.New("Author cannot be empty")}},
		{"long author", args{strPtr(" The Great Gatsby "), strPtr(strings.Repeat("a", 256)), intPtr(1925), strPtr("A classic novel about the American Dream.")}, want{errors.New("ensure this value has at most 255 characters")}},
		{"year before 1000", args{strPtr(" The Great Gatsby "), strPtr(" F. Scott Fitzgerald "), intPtr(999), strPtr("A classic novel about the American Dream.")}, want{errors.New("Published year must be after year 1000")}},
		{"year in future", args{strPtr(" The Great Gatsby "), strPtr(" F. Scott Fitzgerald "), intPtr(currentYear + 1), strPtr("A classic novel about the American Dream.")}, want{fmt.Errorf("Published year cannot be in the future (current year: %d)", currentYear)}},
		{"empty summary", args{strPtr(" The Great Gatsby "), strPtr(" F. Scott Fitzgerald "), intPtr(1925), strPtr("   ")}, want{nil}},
		{"long summary", args{strPtr(" The Great Gatsby "), strPtr(" F. Scott Fitzgerald "), intPtr(1925), strPtr(strings.Repeat("a", 2001))}, want{errors.New("ensure this value has at most 2000 characters")}},
		{"no title", args{nil, strPtr(" F. Scott Fitzgerald "), intPtr(1925), strPtr("A classic novel about the American Dream.")}, want{nil}},
		{"no author", args{strPtr(" The Great Gatsby "), nil, intPtr(1925), strPtr("A classic novel about the American Dream.")}, want{nil}},
		{"no published year", args{strPtr(" The Great Gatsby "), strPtr(" F. Scott Fitzgerald "), nil, strPtr("A classic novel about the American Dream.")}, want{nil}},
		{"no summary", args{strPtr(" The Great Gatsby "), strPtr(" F. Scott Fitzgerald "), intPtr(1925), nil}, want{nil}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bu := &schemas.BookUpdate{
				Title:         tt.args.title,
				Author:        tt.args.author,
				PublishedYear: tt.args.publishedYear,
				Summary:       tt.args.summary,
			}
			err := bu.Validate()
			assert.Equal(t, tt.want.err, err)
		})
	}
}

func TestBookResponse(t *testing.T) {
	br := &schemas.BookResponse{
		ID:            1,
		Title:         " The Great Gatsby ",
		Author:        " F. Scott Fitzgerald ",
		PublishedYear: 1925,
		Summary:       datatypes.TextPtr("A classic novel about the American Dream."),
	}

	assert.Equal(t, uint(1), br.ID)
	assert.Equal(t, "The Great Gatsby", br.Title)
	assert.Equal(t, "F. Scott Fitzgerald", br.Author)
	assert.Equal(t, 1925, br.PublishedYear)
	assert.Equal(t, "A classic novel about the American Dream.", *br.Summary)
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func datatypesTextPtr(s string) *datatypes.Text {
	return datatypes.TextPtr(s)
}