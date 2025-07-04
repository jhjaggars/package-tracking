package config

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBoolOrDefault returns environment variable as boolean or default
func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getEnvIntOrDefault returns environment variable as integer or default
func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getEnvDurationOrDefault returns environment variable as duration or default
func getEnvDurationOrDefault(key string, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	
	// Parse default value
	duration, err := time.ParseDuration(defaultValue)
	if err != nil {
		return time.Hour // Fallback to 1 hour
	}
	return duration
}

// validateEnvFilePath validates that the env file path is safe and prevents directory traversal
func validateEnvFilePath(filename string) error {
	if filename == "" {
		return nil
	}
	
	// Clean the path and check for directory traversal attempts
	cleanPath := filepath.Clean(filename)
	
	// Check for absolute paths outside of current working directory
	if filepath.IsAbs(cleanPath) {
		// Allow temporary directories (common for testing)
		tmpDir := os.TempDir()
		if strings.HasPrefix(cleanPath, filepath.Clean(tmpDir)) {
			// Allow files in temp directory
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cannot determine current directory: %w", err)
			}
			
			// Ensure absolute path is within or below current directory
			relPath, err := filepath.Rel(cwd, cleanPath)
			if err != nil {
				return fmt.Errorf("invalid file path: %w", err)
			}
			
			if strings.HasPrefix(relPath, "..") {
				return fmt.Errorf("file path cannot access parent directories: %s", filename)
			}
		}
	} else {
		// For relative paths, ensure they don't traverse up
		if strings.Contains(cleanPath, "..") {
			return fmt.Errorf("file path cannot contain '..': %s", filename)
		}
	}
	
	// Ensure the file has a reasonable extension (.env or starts with .env)
	if ext := filepath.Ext(cleanPath); ext != "" && ext != ".env" && !strings.HasPrefix(filepath.Base(cleanPath), ".env") {
		return fmt.Errorf("env file must have .env extension or no extension: %s", filename)
	}
	
	return nil
}

// loadEnvFile loads environment variables from a .env file if it exists
func loadEnvFile(filename string) error {
	// Validate file path for security
	if err := validateEnvFilePath(filename); err != nil {
		return fmt.Errorf("invalid env file path: %w", err)
	}
	
	file, err := os.Open(filename)
	if err != nil {
		// .env file doesn't exist or can't be opened, which is fine
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Split on first '=' character
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Remove quotes if present
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		   (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}
		
		// Only set if not already set in environment
		if existing := os.Getenv(key); existing == "" {
			os.Setenv(key, value)
			slog.Debug("Loaded env var from .env file", "key", key, "value", value)
		} else {
			slog.Debug("Env var already set, skipping .env file value", "key", key, "existing", existing, "env_file_value", value)
		}
	}
	
	return nil
}

// LoadEnvFile is a public wrapper around loadEnvFile for external use
func LoadEnvFile(filename string) error {
	return loadEnvFile(filename)
}

// getEnvInt64OrDefault returns environment variable as int64 or default
func getEnvInt64OrDefault(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getEnvFloatOrDefault returns environment variable as float64 or default
func getEnvFloatOrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}