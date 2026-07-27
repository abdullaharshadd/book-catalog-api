```go
package internal_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/your/module/internal"
)

func TestVersion(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{
			name:     "Version equals 1.0.0",
			got:      internal.Version,
			expected: "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.got, "Version constant should equal '1.0.0'")
		})
	}
}

func TestVersionIsString(t *testing.T) {
	tests := []struct {
		name string
		got  interface{}
	}{
		{
			name: "Version is a string type",
			got:  internal.Version,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.IsType(t, "", tt.got, "Version should be of type string")
		})
	}
}

func TestVersionFollowsSemanticVersioning(t *testing.T) {
	semverRegex := regexp.MustCompile(`^\d+\.\d+\.\d+$`)

	tests := []struct {
		name    string
		version string
	}{
		{
			name:    "Version follows MAJOR.MINOR.PATCH format",
			version: internal.Version,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Regexp(t, semverRegex, tt.version, "Version should follow semantic versioning format (MAJOR.MINOR.PATCH)")
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
			name:     "Author equals 'Abdullah Arshad'",
			got:      internal.Author,
			expected: "Abdullah Arshad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.got, "Author constant should equal 'Abdullah Arshad'")
		})
	}
}

func TestAuthorIsString(t *testing.T) {
	tests := []struct {
		name string
		got  interface{}
	}{
		{
			name: "Author is a string type",
			got:  internal.Author,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.IsType(t, "", tt.got, "Author should be of type string")
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
			name:     "Email equals 'abdullah.arshad.314@gmail.com'",
			got:      internal.Email,
			expected: "abdullah.arshad.314@gmail.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.got, "Email constant should equal 'abdullah.arshad.314@gmail.com'")
		})
	}
}

func TestEmailIsString(t *testing.T) {
	tests := []struct {
		name string
		got  interface{}
	}{
		{
			name: "Email is a string type",
			got:  internal.Email,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.IsType(t, "", tt.got, "Email should be of type string")
		})
	}
}

func TestEmailIsValidFormat(t *testing.T) {
	// RFC 5322-compatible simplified email regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

	tests := []struct {
		name  string
		email string
	}{
		{
			name:  "Email follows valid email address format",
			email: internal.Email,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Regexp(t, emailRegex, tt.email, "Email should be a valid email address format")
		})
	}
}

func TestAllConstantsDefined(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		label    string
	}{
		{
			name:     "Version is defined and non-empty",
			constant: internal.Version,
			label:    "Version",
		},
		{
			name:     "Author is defined and non-empty",
			constant: internal.Author,
			label:    "Author",
		},
		{
			name:     "Email is defined and non-empty",
			constant: internal.Email,
			label:    "Email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.constant, "%s constant should be defined and non-empty", tt.label)
		})
	}
}

func TestPackageMetadataConsistency(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		value    string
		expected string
	}{
		{
			name:     "Version constant value",
			field:    "Version",
			value:    internal.Version,
			expected: "1.0.0",
		},
		{
			name:     "Author constant value",
			field:    "Author",
			value:    internal.Author,
			expected: "Abdullah Arshad",
		},
		{
			name:     "Email constant value",
			field:    "Email",
			value:    internal.Email,
			expected: "abdullah.arshad.314@gmail.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.value, "Constant %s should have expected value", tt.field)
		})
	}
}
```