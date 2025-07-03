package cmd

import (
	"fmt"
	"strconv"
	"strings"
)

// validateAndParseID validates that the argument is a non-empty, valid integer ID
func validateAndParseID(arg string) (int, error) {
	if strings.TrimSpace(arg) == "" {
		return 0, fmt.Errorf("ID cannot be empty")
	}
	
	id, err := strconv.Atoi(arg)
	if err != nil {
		return 0, fmt.Errorf("invalid ID '%s': must be a positive integer", arg)
	}
	
	if id <= 0 {
		return 0, fmt.Errorf("invalid ID '%d': must be a positive integer", id)
	}
	
	return id, nil
}