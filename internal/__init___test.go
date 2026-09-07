package internal_test

import (
	"testing"
	"internal"
	"github.com/stretchr/testify/assert"
)

func TestNewBookCatalogAPI(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		author   string
		email    string
		expected *internal.BookCatalogAPI
	}{
		{
			name:     "Valid input",
			version:  "1.0.0",
			author:   "Abdullah Arshad",
			email:    "abdullah.arshad.314@gmail.com",
			expected: &internal.BookCatalogAPI{"1.0.0", "Abdullah Arshad", "abdullah.arshad.314@gmail.com"},
		},
		{
			name:     "Empty version",
			version:  "",
			author:   "Abdullah Arshad",
			email:    "abdullah.arshad.314@gmail.com",
			expected: &internal.BookCatalogAPI{"", "Abdullah Arshad", "abdullah.arshad.314@gmail.com"},
		},
		{
			name:     "Empty author",
			version:  "1.0.0",
			author:   "",
			email:    "abdullah.arshad.314@gmail.com",
			expected: &internal.BookCatalogAPI{"1.0.0", "", "abdullah.arshad.314@gmail.com"},
		},
		{
			name:     "Empty email",
			version:  "1.0.0",
			author:   "Abdullah Arshad",
			email:    "",
			expected: &internal.BookCatalogAPI{"1.0.0", "Abdullah Arshad", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := internal.NewBookCatalogAPI(tt.version, tt.author, tt.email)
			assert.Equal(t, tt.expected.Version, actual.Version)
			assert.Equal(t, tt.expected.Author, actual.Author)
			assert.Equal(t, tt.expected.Email, actual.Email)
		})
	}
}

func TestBookCatalogAPI_GlobalInvariants(t *testing.T) {
	api := internal.bookCatalogAPI

	assert.IsType(t, "", api.Version)
	assert.IsType(t, "", api.Author)
	assert.IsType(t, "", api.Email)
}