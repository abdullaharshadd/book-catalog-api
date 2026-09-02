```go
package internal_test

import (
	"testing"
	"regexp"
	"internal"
	"net/http"
	"net/http/httptest"
	"github.com/stretchr/testify/assert"
)

func TestPackageVersion(t *testing.T) {
	type test struct {
		name        string
		expected    string
		validation  func(string) bool
	}
	tests := []test{
		{
			name: "returns correct version",
			expected: "1.0.0",
			validation: func(v string) bool {
				re := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
				return re.MatchString(v)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := internal.PackageVersion
			assert.Equal(t, tt.expected, version)
			assert.True(t, tt.validation(version))
		})
	}
}

func TestPackageAuthor(t *testing.T) {
	type test struct {
		name        string
		expected    string
		validation  func(string) bool
	}
	tests := []test{
		{
			name: "returns correct author",
			expected: "Abdullah Arshad",
			validation: func(a string) bool {
				return a != ""
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			author := internal.PackageAuthor
			assert.Equal(t, tt.expected, author)
			assert.True(t, tt.validation(author))
		})
	}
}

func TestPackageEmail(t *testing.T) {
	type test struct {
		name        string
		expected    string
		validation  func(string) bool
	}
	tests := []test{
		{
			name: "returns correct email",
			expected: "abdullah.arshad.314@gmail.com",
			validation: func(e string) bool {
				re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
				return re.MatchString(e)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := internal.PackageEmail
			assert.Equal(t, tt.expected, email)
			assert.True(t, tt.validation(email))
		})
	}
}

// Mock HTTP handler tests
func TestMetadataHTTPHandler(t *testing.T) {
	// Assuming there is a handler function that returns metadata
	// For example purposes, these are mocked handlers
	versionHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(internal.PackageVersion))
	})
	authorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(internal.PackageAuthor))
	})
	emailHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(internal.PackageEmail))
	})

	tests := []struct {
		name        string
		handlerFunc http.HandlerFunc
		expected    string
	}{
		{"version handler returns correct version", versionHandler, "1.0.0"},
		{"author handler returns correct author", authorHandler, "Abdullah Arshad"},
		{"email handler returns correct email", emailHandler, "abdullah.arshad.314@gmail.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			rr := httptest.NewRecorder()
			tt.handlerFunc.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, tt.expected, rr.Body.String())
		})
	}
}
```