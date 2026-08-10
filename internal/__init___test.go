```go
package internal_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yourusername/bookcatalog/internal"
)

// semVerPattern matches a strict major.minor.patch semantic version string.
var semVerPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// emailPattern is a simple RFC-5322-ish pattern sufficient for the invariant
// check specified in the behavioral specs.
var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// ---------------------------------------------------------------------------
// Version constant
// ---------------------------------------------------------------------------

func TestVersion(t *testing.T) {
	tests := []struct {
		name            string
		scenario        string
		wantValue       string
		wantNonEmpty    bool
		wantSemVer      bool
		wantConstantStr bool
	}{
		{
			name:            "accessing the package version attribute returns 1.0.0",
			scenario:        "accessing the package version attribute",
			wantValue:       "1.0.0",
			wantNonEmpty:    true,
			wantSemVer:      true,
			wantConstantStr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// The constant value must equal exactly the expected string.
			assert.Equal(t, tc.wantValue, internal.Version,
				"Version should equal %q (scenario: %s)", tc.wantValue, tc.scenario)

			// Invariant: Version is always a non-empty string.
			if tc.wantNonEmpty {
				assert.NotEmpty(t, internal.Version, "Version must not be empty")
			}

			// Invariant: Version follows semantic versioning format (major.minor.patch).
			if tc.wantSemVer {
				assert.Regexp(t, semVerPattern, internal.Version,
					"Version must follow semantic versioning format (major.minor.patch)")
			}

			// Invariant: Value remains constant across repeated accesses.
			if tc.wantConstantStr {
				first := internal.Version
				second := internal.Version
				assert.Equal(t, first, second,
					"Version must be constant across multiple accesses")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Author constant
// ---------------------------------------------------------------------------

func TestAuthor(t *testing.T) {
	tests := []struct {
		name            string
		scenario        string
		wantValue       string
		wantNonEmpty    bool
		wantConstantStr bool
	}{
		{
			name:            "accessing the package author attribute returns Abdullah Arshad",
			scenario:        "accessing the package author attribute",
			wantValue:       "Abdullah Arshad",
			wantNonEmpty:    true,
			wantConstantStr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// The constant value must equal exactly the expected string.
			assert.Equal(t, tc.wantValue, internal.Author,
				"Author should equal %q (scenario: %s)", tc.wantValue, tc.scenario)

			// Invariant: Author is always a non-empty string.
			if tc.wantNonEmpty {
				assert.NotEmpty(t, internal.Author, "Author must not be empty")
			}

			// Invariant: Value remains constant across repeated accesses.
			if tc.wantConstantStr {
				first := internal.Author
				second := internal.Author
				assert.Equal(t, first, second,
					"Author must be constant across multiple accesses")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Email constant
// ---------------------------------------------------------------------------

func TestEmail(t *testing.T) {
	tests := []struct {
		name             string
		scenario         string
		wantValue        string
		wantNonEmpty     bool
		wantValidEmail   bool
		wantConstantStr  bool
	}{
		{
			name:            "accessing the package email attribute returns abdullah.arshad.314@gmail.com",
			scenario:        "accessing the package email attribute",
			wantValue:       "abdullah.arshad.314@gmail.com",
			wantNonEmpty:    true,
			wantValidEmail:  true,
			wantConstantStr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// The constant value must equal exactly the expected string.
			assert.Equal(t, tc.wantValue, internal.Email,
				"Email should equal %q (scenario: %s)", tc.wantValue, tc.scenario)

			// Invariant: Email is always a non-empty string.
			if tc.wantNonEmpty {
				assert.NotEmpty(t, internal.Email, "Email must not be empty")
			}

			// Invariant: Email is a valid email address format.
			if tc.wantValidEmail {
				assert.Regexp(t, emailPattern, internal.Email,
					"Email must be a valid email address format")

				// Additional structural checks.
				assert.True(t, strings.Contains(internal.Email, "@"),
					"Email must contain '@'")
				parts := strings.SplitN(internal.Email, "@", 2)
				assert.Len(t, parts, 2, "Email must have exactly one '@'")
				assert.NotEmpty(t, parts[0], "Email local-part must not be empty")
				assert.NotEmpty(t, parts[1], "Email domain must not be empty")
				assert.True(t, strings.Contains(parts[1], "."),
					"Email domain must contain a dot")
			}

			// Invariant: Value remains constant across repeated accesses.
			if tc.wantConstantStr {
				first := internal.Email
				second := internal.Email
				assert.Equal(t, first, second,
					"Email must be constant across multiple accesses")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Global invariants
// ---------------------------------------------------------------------------

func TestGlobalInvariants(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "package is importable without side effects",
			fn: func(t *testing.T) {
				// If we reached this point, the package was imported successfully.
				// The mere existence of the test binary proves no import-time panic
				// or side-effect occurred.
				assert.NotPanics(t, func() {
					_ = internal.Version
					_ = internal.Author
					_ = internal.Email
				}, "Accessing package constants must not panic")
			},
		},
		{
			name: "all three metadata constants are exposed",
			fn: func(t *testing.T) {
				assert.NotEmpty(t, internal.Version, "Version must be exposed and non-empty")
				assert.NotEmpty(t, internal.Author, "Author must be exposed and non-empty")
				assert.NotEmpty(t, internal.Email, "Email must be exposed and non-empty")
			},
		},
		{
			name: "metadata values remain constant across multiple imports",
			fn: func(t *testing.T) {
				v1, v2 := internal.Version, internal.Version
				a1, a2 := internal.Author, internal.Author
				e1, e2 := internal.Email, internal.Email

				assert.Equal(t, v1, v2, "Version is not constant")
				assert.Equal(t, a1, a2, "Author is not constant")
				assert.Equal(t, e1, e2, "Email is not constant")
			},
		},
		{
			name: "no executable logic or state mutation occurs beyond constant definitions",
			fn: func(t *testing.T) {
				// Accessing constants multiple times must always yield the same
				// results, proving no mutable state is involved.
				for i := 0; i < 5; i++ {
					assert.Equal(t, "1.0.0", internal.Version)
					assert.Equal(t, "Abdullah Arshad", internal.Author)
					assert.Equal(t, "abdullah.arshad.314@gmail.com", internal.Email)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, tc.fn)
	}
}
```