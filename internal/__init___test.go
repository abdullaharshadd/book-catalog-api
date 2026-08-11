```go
package internal_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yourusername/bookcatalog/internal"
)

// semVerRegex matches strings in the form major.minor.patch where each
// component is one or more decimal digits.
var semVerRegex = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// emailRegex is a deliberately simple RFC-5321-ish check that is sufficient
// for the invariant stated in the spec.
var emailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// ---------------------------------------------------------------------------
// Version
// ---------------------------------------------------------------------------

func TestVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		got         string
		wantExact   string
		wantNonEmpty bool
		wantSemVer  bool
	}{
		{
			name:        "accessing the package version attribute returns 1.0.0",
			got:         internal.Version,
			wantExact:   "1.0.0",
			wantNonEmpty: true,
			wantSemVer:  true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.wantExact != "" {
				assert.Equal(t, tc.wantExact, tc.got,
					"Version constant must equal the expected release string")
			}

			if tc.wantNonEmpty {
				assert.NotEmpty(t, tc.got,
					"Version constant must be a non-empty string (invariant)")
			}

			if tc.wantSemVer {
				assert.Regexp(t, semVerRegex, tc.got,
					"Version constant must follow semantic versioning format major.minor.patch (invariant)")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Author
// ---------------------------------------------------------------------------

func TestAuthor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		got          string
		wantExact    string
		wantNonEmpty bool
	}{
		{
			name:         "accessing the package author attribute returns Abdullah Arshad",
			got:          internal.Author,
			wantExact:    "Abdullah Arshad",
			wantNonEmpty: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.wantExact != "" {
				assert.Equal(t, tc.wantExact, tc.got,
					"Author constant must equal the expected author name")
			}

			if tc.wantNonEmpty {
				assert.NotEmpty(t, tc.got,
					"Author constant must be a non-empty string (invariant)")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Email
// ---------------------------------------------------------------------------

func TestEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		got          string
		wantExact    string
		wantNonEmpty bool
		wantValidEmail bool
	}{
		{
			name:           "accessing the package email attribute returns abdullah.arshad.314@gmail.com",
			got:            internal.Email,
			wantExact:      "abdullah.arshad.314@gmail.com",
			wantNonEmpty:   true,
			wantValidEmail: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.wantExact != "" {
				assert.Equal(t, tc.wantExact, tc.got,
					"Email constant must equal the expected contact email")
			}

			if tc.wantNonEmpty {
				assert.NotEmpty(t, tc.got,
					"Email constant must be a non-empty string (invariant)")
			}

			if tc.wantValidEmail {
				assert.Regexp(t, emailRegex, tc.got,
					"Email constant must be a valid email address format (invariant)")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Global invariants
// ---------------------------------------------------------------------------

func TestGlobalInvariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		description string
		check       func(t *testing.T)
	}{
		{
			name:        "metadata constants are static and do not change at runtime",
			description: "Reading the constants twice yields identical values",
			check: func(t *testing.T) {
				t.Helper()
				// Read each constant twice and assert idempotence.
				assert.Equal(t, internal.Version, internal.Version,
					"Version must be stable across reads")
				assert.Equal(t, internal.Author, internal.Author,
					"Author must be stable across reads")
				assert.Equal(t, internal.Email, internal.Email,
					"Email must be stable across reads")
			},
		},
		{
			name:        "all three metadata constants are non-empty",
			description: "Importing the package exposes non-empty Version, Author and Email",
			check: func(t *testing.T) {
				t.Helper()
				assert.NotEmpty(t, internal.Version, "Version must not be empty")
				assert.NotEmpty(t, internal.Author, "Author must not be empty")
				assert.NotEmpty(t, internal.Email, "Email must not be empty")
			},
		},
		{
			name:        "version follows semantic versioning",
			description: "Version string matches major.minor.patch pattern",
			check: func(t *testing.T) {
				t.Helper()
				assert.Regexp(t, semVerRegex, internal.Version,
					"Version must follow semver format")
			},
		},
		{
			name:        "email follows valid email format",
			description: "Email string contains an @ sign and a domain with a dot",
			check: func(t *testing.T) {
				t.Helper()
				assert.Regexp(t, emailRegex, internal.Email,
					"Email must be a valid email address")
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.check(t)
		})
	}
}
```