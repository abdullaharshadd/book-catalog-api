```go
package internal_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yourusername/bookcatalog/internal"
)

func TestVersion(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{
			name:     "package is imported and Version is accessed",
			got:      internal.Version,
			expected: "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.got, "Version should equal '1.0.0'")
		})
	}
}

func TestVersionInvariants(t *testing.T) {
	tests := []struct {
		name      string
		checkFunc func(t *testing.T)
	}{
		{
			name: "Version is always a non-empty string",
			checkFunc: func(t *testing.T) {
				assert.NotEmpty(t, internal.Version, "Version must not be empty")
			},
		},
		{
			name: "Version follows semantic versioning format MAJOR.MINOR.PATCH",
			checkFunc: func(t *testing.T) {
				semverRegex := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
				assert.True(t, semverRegex.MatchString(internal.Version),
					"Version '%s' does not match semantic versioning format MAJOR.MINOR.PATCH", internal.Version)
			},
		},
		{
			name: "Version equals 1.0.0",
			checkFunc: func(t *testing.T) {
				assert.Equal(t, "1.0.0", internal.Version)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.checkFunc(t)
		})
	}
}

func TestAuthor(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{
			name:     "package is imported and Author is accessed",
			got:      internal.Author,
			expected: "Abdullah Arshad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.got, "Author should equal 'Abdullah Arshad'")
		})
	}
}

func TestAuthorInvariants(t *testing.T) {
	tests := []struct {
		name      string
		checkFunc func(t *testing.T)
	}{
		{
			name: "Author is always a non-empty string",
			checkFunc: func(t *testing.T) {
				assert.NotEmpty(t, internal.Author, "Author must not be empty")
			},
		},
		{
			name: "Author equals Abdullah Arshad",
			checkFunc: func(t *testing.T) {
				assert.Equal(t, "Abdullah Arshad", internal.Author)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.checkFunc(t)
		})
	}
}

func TestEmail(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{
			name:     "package is imported and Email is accessed",
			got:      internal.Email,
			expected: "abdullah.arshad.314@gmail.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.got, "Email should equal 'abdullah.arshad.314@gmail.com'")
		})
	}
}

func TestEmailInvariants(t *testing.T) {
	tests := []struct {
		name      string
		checkFunc func(t *testing.T)
	}{
		{
			name: "Email is always a non-empty string",
			checkFunc: func(t *testing.T) {
				assert.NotEmpty(t, internal.Email, "Email must not be empty")
			},
		},
		{
			name: "Email is a valid email address format",
			checkFunc: func(t *testing.T) {
				emailRegex := regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
				assert.True(t, emailRegex.MatchString(internal.Email),
					"Email '%s' does not match a valid email address format", internal.Email)
			},
		},
		{
			name: "Email equals abdullah.arshad.314@gmail.com",
			checkFunc: func(t *testing.T) {
				assert.Equal(t, "abdullah.arshad.314@gmail.com", internal.Email)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.checkFunc(t)
		})
	}
}

func TestGlobalInvariants(t *testing.T) {
	tests := []struct {
		name      string
		checkFunc func(t *testing.T)
	}{
		{
			name: "Version is defined and accessible after import",
			checkFunc: func(t *testing.T) {
				v := internal.Version
				assert.NotEmpty(t, v, "Version must be defined and non-empty")
			},
		},
		{
			name: "Author is defined and accessible after import",
			checkFunc: func(t *testing.T) {
				a := internal.Author
				assert.NotEmpty(t, a, "Author must be defined and non-empty")
			},
		},
		{
			name: "Email is defined and accessible after import",
			checkFunc: func(t *testing.T) {
				e := internal.Email
				assert.NotEmpty(t, e, "Email must be defined and non-empty")
			},
		},
		{
			name: "all metadata constants are simultaneously accessible",
			checkFunc: func(t *testing.T) {
				assert.NotEmpty(t, internal.Version)
				assert.NotEmpty(t, internal.Author)
				assert.NotEmpty(t, internal.Email)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.checkFunc(t)
		})
	}
}

func TestAllMetadataValues(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "Version constant value",
			constant: internal.Version,
			expected: "1.0.0",
		},
		{
			name:     "Author constant value",
			constant: internal.Author,
			expected: "Abdullah Arshad",
		},
		{
			name:     "Email constant value",
			constant: internal.Email,
			expected: "abdullah.arshad.314@gmail.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
			assert.NotEmpty(t, tt.constant)
		})
	}
}
```