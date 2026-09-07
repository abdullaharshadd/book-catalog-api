package internal

import (
	"testing"
	"regexp"
	"github.com/stretchr/testify/assert"
)

var versionSpecs = []struct {
	expectedVersion string
}{
	{"1.0.0"},
}

func TestVersion(t *testing.T) {
	for _, spec := range versionSpecs {
		assert.Equal(t, spec.expectedVersion, Version)
	}
}

var authorSpecs = []struct {
	expectedAuthor string
}{
	{"Abdullah Arshad"},
}

func TestAuthor(t *testing.T) {
	for _, spec := range authorSpecs {
		assert.Equal(t, spec.expectedAuthor, Author)
	}
}

var emailSpecs = []struct {
	email      string
	shouldPass bool
}{
	{"abdullah.arshad.314@gmail.com", true},
	{"invalid-email", false},
	{"@missingusername.com", false},
	{"username@missingdomain", false},
	{"username@domain.com@", false},
}

func TestEmail(t *testing.T) {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	for _, spec := range emailSpecs {
		if spec.shouldPass {
			assert.True(t, emailRegex.MatchString(spec.email), "Expected email to be valid")
		} else {
			assert.False(t, emailRegex.MatchString(spec.email), "Expected email to be invalid")
		}
	}
	assert.True(t, emailRegex.MatchString(Email), "Application email should be valid")
}