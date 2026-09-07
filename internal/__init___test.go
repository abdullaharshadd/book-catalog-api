package init_test

import (
	"testing"
	"internal/init"
	"github.com/stretchr/testify/assert"
)

func TestGlobalInvariants(t *testing.T) {
	tests := []struct {
		name     string
		variable string
		expected string
	}{
		{"Version", "internal/init.__version__", "1.0.0"},
		{"Author", "internal/init.__author__", "Abdullah Arshad"},
		{"Email", "internal/init.__email__", "abdullah.arshad.314@gmail.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := getVariableValue(tt.variable)
			assert.Equal(t, tt.expected, actual, "Expected %s to be '%s', got '%s'", tt.name, tt.expected, actual)
		})
	}
}

func getVariableValue(variableName string) string {
	switch variableName {
	case "internal/init.__version__":
		return init.__version__
	case "internal/init.__author__":
		return init.__author__
	case "internal/init.__email__":
		return init.__email__
	default:
		return ""
	}
}