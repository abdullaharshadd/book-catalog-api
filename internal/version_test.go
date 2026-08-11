```go
package internal_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yourusername/bookcatalog/internal"
)

// semVerPattern matches a standard semantic version string: major.minor.patch
var semVerPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// emailPattern performs a basic RFC-5321-style sanity check on an e-mail address.
var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// ---------------------------------------------------------------------------
// Version constant tests
// ---------------------------------------------------------------------------

func TestVersion(t *testing.T) {
	tests := []struct {
		name     string
		scenario string
		got      string
		// invariants
		wantExact    string
		wantNonEmpty bool
		wantSemVer   bool
	}{
		{
			name:         "accessing the package version attribute returns the correct string",
			scenario:     "accessing the package version attribute",
			got:          internal.Version,
			wantExact:    "1.0.0",
			wantNonEmpty: true,
			wantSemVer:   true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// exact value
			assert.Equal(t, tc.wantExact, tc.got,
				"Version should equal %q", tc.wantExact)

			// invariant: non-empty
			if tc.wantNonEmpty {
				assert.NotEmpty(t, tc.got, "Version must be a non-empty string")
			}

			// invariant: semantic version format
			if tc.wantSemVer {
				assert.Regexp(t, semVerPattern, tc.got,
					"Version must follow semver format (major.minor.patch)")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Author constant tests
// ---------------------------------------------------------------------------

func TestAuthor(t *testing.T) {
	tests := []struct {
		name         string
		scenario     string
		got          string
		wantExact    string
		wantNonEmpty bool
	}{
		{
			name:         "accessing the package author attribute returns the correct string",
			scenario:     "accessing the package author attribute",
			got:          internal.Author,
			wantExact:    "Abdullah Arshad",
			wantNonEmpty: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// exact value
			assert.Equal(t, tc.wantExact, tc.got,
				"Author should equal %q", tc.wantExact)

			// invariant: non-empty
			if tc.wantNonEmpty {
				assert.NotEmpty(t, tc.got, "Author must be a non-empty string")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Email constant tests
// ---------------------------------------------------------------------------

func TestEmail(t *testing.T) {
	tests := []struct {
		name          string
		scenario      string
		got           string
		wantExact     string
		wantNonEmpty  bool
		wantValidMail bool
	}{
		{
			name:          "accessing the package email attribute returns the correct string",
			scenario:      "accessing the package email attribute",
			got:           internal.Email,
			wantExact:     "abdullah.arshad.314@gmail.com",
			wantNonEmpty:  true,
			wantValidMail: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// exact value
			assert.Equal(t, tc.wantExact, tc.got,
				"Email should equal %q", tc.wantExact)

			// invariant: non-empty
			if tc.wantNonEmpty {
				assert.NotEmpty(t, tc.got, "Email must be a non-empty string")
			}

			// invariant: valid e-mail format
			if tc.wantValidMail {
				assert.Regexp(t, emailPattern, tc.got,
					"Email must be in a valid e-mail address format")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Global invariant tests
// ---------------------------------------------------------------------------

// TestMetadataConstantsExposed verifies that all three metadata constants are
// exported from the package (i.e. they compile and are accessible).
func TestMetadataConstantsExposed(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"Version is exported", internal.Version},
		{"Author is exported", internal.Author},
		{"Email is exported", internal.Email},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// If the constant were unexported the file would not compile, but
			// we also assert non-empty as an additional runtime guard.
			assert.NotEmpty(t, tc.value)
		})
	}
}

// TestNoSideEffectsOnImport verifies that importing the package and reading
// constants produces no observable side effects (no panic, no mutation of
// global state that would cause subsequent reads to differ).
func TestNoSideEffectsOnImport(t *testing.T) {
	// Read each constant twice; both reads must yield identical values.
	tests := []struct {
		name   string
		read1  func() string
		read2  func() string
	}{
		{
			name:  "Version is stable across multiple reads",
			read1: func() string { return internal.Version },
			read2: func() string { return internal.Version },
		},
		{
			name:  "Author is stable across multiple reads",
			read1: func() string { return internal.Author },
			read2: func() string { return internal.Author },
		},
		{
			name:  "Email is stable across multiple reads",
			read1: func() string { return internal.Email },
			read2: func() string { return internal.Email },
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.read1(), tc.read2(),
				"Constant value must not change between reads")
		})
	}
}

// TestVersionSemVerComponents validates each numeric component of the version
// string individually so that the table entry stays self-documenting.
func TestVersionSemVerComponents(t *testing.T) {
	parts := strings.Split(internal.Version, ".")

	tests := []struct {
		name          string
		index         int
		expectedPart  string
	}{
		{"major version component is '1'", 0, "1"},
		{"minor version component is '0'", 1, "0"},
		{"patch version component is '0'", 2, "0"},
	}

	assert.Len(t, parts, 3, "Version must have exactly three dot-separated components")

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.index < len(parts) {
				assert.Equal(t, tc.expectedPart, parts[tc.index])
			} else {
				t.Errorf("version string %q does not have component at index %d", internal.Version, tc.index)
			}
		})
	}
}
```