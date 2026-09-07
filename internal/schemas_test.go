package internal

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var bookCreateTestCases = []struct {
	name       string
	input      BookCreate
	wantErr    bool
	errMessage string
}{
	{"valid book", BookCreate{Title: " The Great Gatsby ", Author: " F. Scott Fitzgerald ", PublishedYear: 1925, Summary: strPtr(" A story about ... ")}, false, ""},
	{"empty title", BookCreate{Title: "", Author: "F. Scott Fitzgerald", PublishedYear: 1925}, true, "Title cannot be empty"},
	{"title too long", BookCreate{Title: strings.Repeat("a", 256), Author: "F. Scott Fitzgerald", PublishedYear: 1925}, true, "ensure this value has at most 255 characters"},
	{"empty author", BookCreate{Title: "The Great Gatsby", Author: "", PublishedYear: 1925}, true, "Author cannot be empty"},
	{"author too long", BookCreate{Title: "The Great Gatsby", Author: strings.Repeat("a", 256), PublishedYear: 1925}, true, "ensure this value has at most 255 characters"},
	{"published year before 1000", BookCreate{Title: "The Great Gatsby", Author: "F. Scott Fitzgerald", PublishedYear: 999}, true, "Published year must be after year 1000"},
	{"published year in the future", BookCreate{Title: "The Great Gatsby", Author: "F. Scott Fitzgerald", PublishedYear: time.Now().Year() + 1}, true, "Published year cannot be in the future"},
	{"summary too long", BookCreate{Title: "The Great Gatsby", Author: "F. Scott Fitzgerald", PublishedYear: 1925, Summary: strPtr(strings.Repeat("a", 2001))}, true, "ensure this value has at most 2000 characters"},
}

func TestBookCreateValidate(t *testing.T) {
	for _, tc := range bookCreateTestCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMessage)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, strings.TrimSpace(tc.input.Title), tc.input.Title)
				assert.Equal(t, strings.TrimSpace(tc.input.Author), tc.input.Author)
				if tc.input.Summary != nil {
					assert.Equal(t, strings.TrimSpace(*tc.input.Summary), *tc.input.Summary)
				}
			}
		})
	}
}

var bookUpdateTestCases = []struct {
	name       string
	input      BookUpdate
	wantErr    bool
	errMessage string
}{
	{"valid update", BookUpdate{Title: strPtr(" The Great Gatsby "), Author: strPtr(" F. Scott Fitzgerald "), PublishedYear: intPtr(1925)}, false, ""},
	{"empty title", BookUpdate{Title: strPtr("")}, true, "Title cannot be empty"},
	{"title too long", BookUpdate{Title: strPtr(strings.Repeat("a", 256))}, true, "ensure this value has at most 255 characters"},
	{"empty author", BookUpdate{Author: strPtr("")}, true, "Author cannot be empty"},
	{"author too long", BookUpdate{Author: strPtr(strings.Repeat("a", 256))}, true, "ensure this value has at most 255 characters"},
	{"published year before 1000", BookUpdate{PublishedYear: intPtr(999)}, true, "Published year must be after year 1000"},
	{"published year in the future", BookUpdate{PublishedYear: intPtr(time.Now().Year() + 1)}, true, "Published year cannot be in the future"},
	{"summary too long", BookUpdate{Summary: strPtr(strings.Repeat("a", 2001))}, true, "ensure this value has at most 2000 characters"},
}

func TestBookUpdateValidate(t *testing.T) {
	for _, tc := range bookUpdateTestCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMessage)
			} else {
				assert.NoError(t, err)
				if tc.input.Title != nil {
					assert.Equal(t, strings.TrimSpace(*tc.input.Title), *tc.input.Title)
				}
				if tc.input.Author != nil {
					assert.Equal(t, strings.TrimSpace(*tc.input.Author), *tc.input.Author)
				}
				if tc.input.Summary != nil {
					assert.Equal(t, strings.TrimSpace(*tc.input.Summary), *tc.input.Summary)
				}
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func TestBookResponse(t *testing.T) {
	br := BookResponse{
		ID:            1,
		Title:         " The Catcher in the Rye ",
		Author:        " J.D. Salinger ",
		PublishedYear: 1951,
		Summary:       strPtr(" A story about a teenage boy... "),
	}
	assert.Equal(t, strings.TrimSpace(br.Title), br.Title)
	assert.Equal(t, strings.TrimSpace(br.Author), br.Author)
	if br.Summary != nil {
		assert.Equal(t, strings.TrimSpace(*br.Summary), *br.Summary)
	}
}

// Mock HTTP handler for demonstration purposes
func mockHandler(w http.ResponseWriter, r *http.Request) {
	book := BookCreate{Title: "The Great Gatsby", Author: "F. Scott Fitzgerald", PublishedYear: 1925}
	err := book.Validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func TestHTTPHandlerWithBookCreateValidation(t *testing.T) {
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	mockHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if assert.Equal(t, http.StatusOK, resp.StatusCode) {
		// Additional checks can be added here if necessary
	}
}