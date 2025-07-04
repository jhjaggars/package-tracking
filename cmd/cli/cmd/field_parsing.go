package cmd

import (
	"fmt"
	"strings"
)

// defaultFields represents the default fields to display in the table
var defaultFields = []string{"id", "tracking", "carrier", "status", "description", "created"}

// availableFields maps field names to their display names
var availableFields = map[string]string{
	"id":          "ID",
	"tracking":    "TRACKING",
	"carrier":     "CARRIER",
	"status":      "STATUS",
	"description": "DESCRIPTION",
	"created":     "CREATED",
	"updated":     "UPDATED",
	"delivery":    "DELIVERY",
	"delivered":   "DELIVERED",
}

// parseFields parses the fields flag and returns a slice of field names
func parseFields(fieldsFlag string) []string {
	if fieldsFlag == "" {
		return defaultFields
	}

	fields := strings.Split(fieldsFlag, ",")
	result := make([]string, 0, len(fields))

	for _, field := range fields {
		trimmed := strings.TrimSpace(field)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// validateFields validates that all provided fields are valid
func validateFields(fields []string) error {
	var invalid []string

	for _, field := range fields {
		if _, exists := availableFields[field]; !exists {
			invalid = append(invalid, field)
		}
	}

	if len(invalid) > 0 {
		return fmt.Errorf("invalid field(s): %s. Available fields: %s",
			strings.Join(invalid, ", "),
			strings.Join(getAvailableFieldNames(), ", "))
	}

	return nil
}

// getFieldDisplayName returns the display name for a field
func getFieldDisplayName(field string) string {
	if displayName, exists := availableFields[field]; exists {
		return displayName
	}
	return field
}

// getAvailableFieldNames returns a slice of all available field names
func getAvailableFieldNames() []string {
	names := make([]string, 0, len(availableFields))
	for name := range availableFields {
		names = append(names, name)
	}
	return names
}