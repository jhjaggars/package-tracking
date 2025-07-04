package cmd

import (
	"fmt"
	"os"
	"testing"

	cliapi "package-tracking/internal/cli"
)

func TestShouldUseInteractiveMode(t *testing.T) {
	tests := []struct {
		name          string
		config        *cliapi.Config
		explicitFlag  bool
		isTerminal    bool
		expected      bool
		description   string
	}{
		{
			name:         "explicit flag true",
			config:       &cliapi.Config{Format: "table", Quiet: false},
			explicitFlag: true,
			isTerminal:   false,
			expected:     true,
			description:  "Should use interactive mode when explicitly requested",
		},
		{
			name:         "explicit flag false",
			config:       &cliapi.Config{Format: "table", Quiet: false},
			explicitFlag: false,
			isTerminal:   true,
			expected:     true,
			description:  "Should use interactive mode when conditions are met",
		},
		{
			name:         "json format",
			config:       &cliapi.Config{Format: "json", Quiet: false},
			explicitFlag: false,
			isTerminal:   true,
			expected:     false,
			description:  "Should not use interactive mode with json format",
		},
		{
			name:         "quiet mode",
			config:       &cliapi.Config{Format: "table", Quiet: true},
			explicitFlag: false,
			isTerminal:   true,
			expected:     false,
			description:  "Should not use interactive mode in quiet mode",
		},
		{
			name:         "not a terminal",
			config:       &cliapi.Config{Format: "table", Quiet: false},
			explicitFlag: false,
			isTerminal:   false,
			expected:     false,
			description:  "Should not use interactive mode when not a terminal",
		},
		{
			name:         "CI environment",
			config:       &cliapi.Config{Format: "table", Quiet: false},
			explicitFlag: false,
			isTerminal:   true,
			expected:     false,
			description:  "Should not use interactive mode in CI environment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock terminal detection for testing
			origIsTerminal := isTerminalFunc
			isTerminalFunc = func() bool {
				return tt.isTerminal
			}
			defer func() {
				isTerminalFunc = origIsTerminal
			}()

			// Test CI environment scenario
			if tt.name == "CI environment" {
				oldCI := os.Getenv("CI")
				os.Setenv("CI", "true")
				defer func() {
					if oldCI == "" {
						os.Unsetenv("CI")
					} else {
						os.Setenv("CI", oldCI)
					}
				}()
			}

			result := shouldUseInteractiveMode(tt.config, tt.explicitFlag)
			if result != tt.expected {
				t.Errorf("shouldUseInteractiveMode() = %v, expected %v for %s", result, tt.expected, tt.description)
			}
		})
	}
}

func TestParseFields(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{"id", "tracking", "carrier", "status", "description", "created"},
		},
		{
			name:     "single field",
			input:    "id",
			expected: []string{"id"},
		},
		{
			name:     "multiple fields",
			input:    "id,tracking,status",
			expected: []string{"id", "tracking", "status"},
		},
		{
			name:     "fields with spaces",
			input:    "id, tracking, status",
			expected: []string{"id", " tracking", " status"},
		},
		{
			name:     "all default fields",
			input:    "id,tracking,carrier,status,description,created",
			expected: []string{"id", "tracking", "carrier", "status", "description", "created"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFields(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseFields() returned %d fields, expected %d", len(result), len(tt.expected))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("parseFields() field %d = %q, expected %q", i, result[i], expected)
				}
			}
		})
	}
}

func TestValidateFields(t *testing.T) {
	tests := []struct {
		name     string
		fields   []string
		expected error
	}{
		{
			name:     "valid fields",
			fields:   []string{"id", "tracking", "carrier"},
			expected: nil,
		},
		{
			name:     "all valid fields",
			fields:   []string{"id", "tracking", "carrier", "status", "description", "created", "updated", "delivery", "delivered"},
			expected: nil,
		},
		{
			name:     "invalid field",
			fields:   []string{"id", "invalid_field"},
			expected: fmt.Errorf("unknown field: invalid_field"),
		},
		{
			name:     "empty fields",
			fields:   []string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateFields(tt.fields)
			if (result == nil) != (tt.expected == nil) {
				t.Errorf("validateFields() error = %v, expected %v", result, tt.expected)
			}
			if result != nil && tt.expected != nil && result.Error() != tt.expected.Error() {
				t.Errorf("validateFields() error message = %q, expected %q", result.Error(), tt.expected.Error())
			}
		})
	}
}