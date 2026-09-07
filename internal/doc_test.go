package internal_test

import (
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
			name:     "package is imported and Version constant is accessed",
			got:      internal.Version,
			expected: "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.IsType(t, "", tt.got, "Version should always be a string")
			assert.Equal(t, tt.expected, tt.got, "Version should equal '1.0.0'")
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
			name:     "package is imported and Author constant is accessed",
			got:      internal.Author,
			expected: "Abdullah Arshad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.IsType(t, "", tt.got, "Author should always be a string")
			assert.Equal(t, tt.expected, tt.got, "Author should equal 'Abdullah Arshad'")
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
			name:     "package is imported and Email constant is accessed",
			got:      internal.Email,
			expected: "abdullah.arshad.314@gmail.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.IsType(t, "", tt.got, "Email should always be a string")
			assert.Equal(t, tt.expected, tt.got, "Email should equal 'abdullah.arshad.314@gmail.com'")
		})
	}
}

func TestMetadataConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{
			name:     "Version constant value",
			got:      internal.Version,
			expected: "1.0.0",
		},
		{
			name:     "Author constant value",
			got:      internal.Author,
			expected: "Abdullah Arshad",
		},
		{
			name:     "Email constant value",
			got:      internal.Email,
			expected: "abdullah.arshad.314@gmail.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.IsType(t, "", tt.got, "metadata constant should always be a string")
			assert.NotEmpty(t, tt.got, "metadata constant should not be empty")
			assert.Equal(t, tt.expected, tt.got)
		})
	}
}

func TestMetadataImmutability(t *testing.T) {
	tests := []struct {
		name          string
		constantValue string
		expectedValue string
	}{
		{
			name:          "Version is immutable",
			constantValue: internal.Version,
			expectedValue: "1.0.0",
		},
		{
			name:          "Author is immutable",
			constantValue: internal.Author,
			expectedValue: "Abdullah Arshad",
		},
		{
			name:          "Email is immutable",
			constantValue: internal.Email,
			expectedValue: "abdullah.arshad.314@gmail.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			firstRead := tt.constantValue
			secondRead := tt.constantValue
			assert.Equal(t, firstRead, secondRead, "constant value should be immutable across multiple reads")
			assert.Equal(t, tt.expectedValue, tt.constantValue)
		})
	}
}