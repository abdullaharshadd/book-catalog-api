```go
package internal_test

import (
	"testing"
	"internal"
	"github.com/stretchr/testify/assert"
)

func TestNewBookCatalogAPI(t *testing.T) {
	tests := []struct {
		name string
		want *internal.BookCatalogAPI
	}{
		{
			name: "valid metadata",
			want: &internal.BookCatalogAPI{
				Version: "1.0.0",
				Author:  "Abdullah Arshad",
				Email:   "abdullah.arshad.314@gmail.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := internal.NewBookCatalogAPI()
			assert.Equal(t, tt.want.Version, got.Version, "Version should match")
			assert.Equal(t, tt.want.Author, got.Author, "Author should match")
			assert.Equal(t, tt.want.Email, got.Email, "Email should match")
		})
	}
}

func TestGetBookCatalogAPI(t *testing.T) {
	want := &internal.BookCatalogAPI{
		Version: "1.0.0",
		Author:  "Abdullah Arshad",
		Email:   "abdullah.arshad.314@gmail.com",
	}

	got := internal.GetBookCatalogAPI()
	assert.Equal(t, want.Version, got.Version, "Version should match")
	assert.Equal(t, want.Author, got.Author, "Author should match")
	assert.Equal(t, want.Email, got.Email, "Email should match")
}
```