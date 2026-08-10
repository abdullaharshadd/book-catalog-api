```go
package internal_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/your/module/internal"
)

func TestVersion(t *testing.T) {
	semverRe := regexp.MustCompile(`^\d+\.\d+\.\d+$`)

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{
			name:     "accessing the package version attribute returns 1.0.0",
			got:      internal.Version,
			expected: "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.got, "Version should equal the expected semantic version string")
			assert.True(t, semverRe.MatchString(tt.got), "Version must be in semantic-version form MAJOR.MINOR.PATCH, got: %q", tt.got)
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
			name:     "accessing the package author attribute returns 'Abdullah Arshad'",
			got:      internal.Author,
			expected: "Abdullah Arshad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.got, "Author should equal the expected author name")
			assert.NotEmpty(t, tt.got, "Author must be a non-empty string")
		})
	}
}

func TestEmail(t *testing.T) {
	// Simple email format validator: requires at least one char, @, domain with dot.
	emailRe := regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{
			name:     "accessing the package email attribute returns 'abdullah.arshad.314@gmail.com'",
			got:      internal.Email,
			expected: "abdullah.arshad.314@gmail.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.got, "Email should equal the expected contact email")
			assert.NotEmpty(t, tt.got, "Email must be a non-empty string")
			assert.True(t, emailRe.MatchString(tt.got), "Email must be in a valid email address format, got: %q", tt.got)
		})
	}
}

func TestConstantsImmutability(t *testing.T) {
	// Verify that constants do not change between multiple accesses (immutability invariant).
	tests := []struct {
		name        string
		firstRead   string
		secondRead  string
		description string
	}{
		{
			name:        "Version is immutable across multiple reads",
			firstRead:   internal.Version,
			secondRead:  internal.Version,
			description: "Version constant must not change between accesses",
		},
		{
			name:        "Author is immutable across multiple reads",
			firstRead:   internal.Author,
			secondRead:  internal.Author,
			description: "Author constant must not change between accesses",
		},
		{
			name:        "Email is immutable across multiple reads",
			firstRead:   internal.Email,
			secondRead:  internal.Email,
			description: "Email constant must not change between accesses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.firstRead, tt.secondRead, tt.description)
		})
	}
}

func TestPackageExposesExactlyThreeMetadataConstants(t *testing.T) {
	// Validate that all three required metadata constants are exposed and non-empty.
	tests := []struct {
		name        string
		value       string
		description string
	}{
		{
			name:        "Version constant is exposed",
			value:       internal.Version,
			description: "package must expose the Version constant",
		},
		{
			name:        "Author constant is exposed",
			value:       internal.Author,
			description: "package must expose the Author constant",
		},
		{
			name:        "Email constant is exposed",
			value:       internal.Email,
			description: "package must expose the Email constant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.value, tt.description)
		})
	}
}

func TestVersionSemanticFormat(t *testing.T) {
	// Additional invariant: version must be parseable as MAJOR.MINOR.PATCH with numeric parts.
	tests := []struct {
		name    string
		version string
	}{
		{
			name:    "Version follows strict semantic versioning format",
			version: internal.Version,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Split(tt.version, ".")
			assert.Len(t, parts, 3, "semantic version must have exactly three dot-separated parts")
			for _, part := range parts {
				assert.NotEmpty(t, part, "each part of semantic version must be non-empty")
				for _, ch := range part {
					assert.True(t, ch >= '0' && ch <= '9', "each part of semantic version must be numeric, got char %q in %q", string(ch), part)
				}
			}
		})
	}
}

func TestEmailContainsAtSymbol(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{
			name:  "Email contains exactly one @ symbol",
			email: internal.Email,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := strings.Count(tt.email, "@")
			assert.Equal(t, 1, count, "email address must contain exactly one '@' symbol, got %d in %q", count, tt.email)
		})
	}
}
```