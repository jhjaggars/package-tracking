package cmd

import (
	"reflect"
	"testing"
)

func TestParseFields(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  []string
	}{
		{
			name:     "empty string returns default fields",
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
			name:     "all available fields",
			input:    "id,tracking,carrier,status,description,created,updated,delivery,delivered",
			expected: []string{"id", "tracking", "carrier", "status", "description", "created", "updated", "delivery", "delivered"},
		},
		{
			name:     "fields with whitespace",
			input:    "id, tracking , status",
			expected: []string{"id", "tracking", "status"},
		},
		{
			name:     "duplicate fields are preserved",
			input:    "id,tracking,id",
			expected: []string{"id", "tracking", "id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFields(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseFields(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateFields(t *testing.T) {
	tests := []struct {
		name      string
		fields    []string
		expected  error
	}{
		{
			name:     "valid fields",
			fields:   []string{"id", "tracking", "status"},
			expected: nil,
		},
		{
			name:     "all valid fields",
			fields:   []string{"id", "tracking", "carrier", "status", "description", "created", "updated", "delivery", "delivered"},
			expected: nil,
		},
		{
			name:     "empty fields list",
			fields:   []string{},
			expected: nil,
		},
		{
			name:     "invalid field",
			fields:   []string{"id", "invalid", "status"},
			expected: nil, // We'll expect an error here when we implement
		},
		{
			name:     "multiple invalid fields",
			fields:   []string{"invalid1", "invalid2"},
			expected: nil, // We'll expect an error here when we implement
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateFields(tt.fields)
			if tt.name == "invalid field" || tt.name == "multiple invalid fields" {
				// These should return errors when we implement the function
				if result == nil {
					t.Error("validateFields() should return an error for invalid fields")
				}
			} else {
				if result != tt.expected {
					t.Errorf("validateFields(%v) = %v, expected %v", tt.fields, result, tt.expected)
				}
			}
		})
	}
}

func TestGetFieldDisplayName(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		expected  string
	}{
		{
			name:     "id field",
			field:    "id",
			expected: "ID",
		},
		{
			name:     "tracking field",
			field:    "tracking",
			expected: "TRACKING",
		},
		{
			name:     "carrier field",
			field:    "carrier",
			expected: "CARRIER",
		},
		{
			name:     "status field",
			field:    "status",
			expected: "STATUS",
		},
		{
			name:     "description field",
			field:    "description",
			expected: "DESCRIPTION",
		},
		{
			name:     "created field",
			field:    "created",
			expected: "CREATED",
		},
		{
			name:     "updated field",
			field:    "updated",
			expected: "UPDATED",
		},
		{
			name:     "delivery field",
			field:    "delivery",
			expected: "DELIVERY",
		},
		{
			name:     "delivered field",
			field:    "delivered",
			expected: "DELIVERED",
		},
		{
			name:     "unknown field",
			field:    "unknown",
			expected: "unknown", // Should return the field name as-is
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFieldDisplayName(tt.field)
			if result != tt.expected {
				t.Errorf("getFieldDisplayName(%q) = %q, expected %q", tt.field, result, tt.expected)
			}
		})
	}
}

// Helper functions are now implemented in field_parsing.go