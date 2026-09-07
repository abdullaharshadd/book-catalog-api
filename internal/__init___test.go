package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	tests := []struct {
		name    string
		constant string
		expected string
	}{
		{"Version", Version, "1.0.0"},
		{"Author", Author, "Abdullah Arshad"},
		{"Email", Email, "abdullah.arshad.314@gmail.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.name {
			case "Version":
				assert.Equal(t, tt.expected, Version)
			case "Author":
				assert.Equal(t, tt.expected, Author)
			case "Email":
				assert.Equal(t, tt.expected, Email)
			default:
				t.Errorf("Unknown constant: %s", tt.name)
			}
		})
	}
}