package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type ConfigTest struct {
	// Define the fields of your ConfigTest here
	Field1 string `yaml:"field1"`
	Field2 int    `yaml:"field2"`
}

func TestParseYamlData(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expected    *ConfigTest
		expectError bool
	}{
		{
			name: "Valid YAML",
			input: []byte(`
field1: "value1"
field2: 2
`),
			expected: &ConfigTest{
				Field1: "value1",
				Field2: 2,
			},
			expectError: false,
		},
		{
			name:        "Empty YAML",
			input:       []byte(``),
			expected:    &ConfigTest{},
			expectError: false,
		},
		{
			name: "Invalid YAML",
			input: []byte(`
field1: "value1"
field2: "invalid_int"
`),
			expected:    nil,
			expectError: true,
		},
		{
			name: "Unknown Field",
			input: []byte(`
field1: "value1"
field2: 2
unknown_field: "value"
`),
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseYamlData[ConfigTest](tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, config)
			}
		})
	}
}
