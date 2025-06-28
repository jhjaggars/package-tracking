package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"package-tracking/internal/database"
)

func TestOutputFormatterPrintShipments(t *testing.T) {
	shipments := []database.Shipment{
		{
			ID:             1,
			TrackingNumber: "1Z999AA1234567890",
			Carrier:        "ups",
			Description:    "Test package",
			Status:         "in_transit",
			CreatedAt:      time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:             2,
			TrackingNumber: "1234567890",
			Carrier:        "fedex",
			Description:    "Another package",
			Status:         "delivered",
			CreatedAt:      time.Date(2023, 12, 2, 11, 0, 0, 0, time.UTC),
		},
	}

	tests := []struct {
		name     string
		format   string
		quiet    bool
		contains []string
	}{
		{
			name:   "table format",
			format: "table",
			quiet:  false,
			contains: []string{"ID", "TRACKING", "CARRIER", "STATUS", "1Z999AA12345", "UPS", "in_transit"},
		},
		{
			name:   "json format",
			format: "json",
			quiet:  false,
			contains: []string{`"id":1`, `"tracking_number":"1Z999AA1234567890"`, `"carrier":"ups"`},
		},
		{
			name:     "quiet mode",
			format:   "table",
			quiet:    true,
			contains: []string{"1", "2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			formatter := NewOutputFormatter(tt.format, tt.quiet)
			err := formatter.PrintShipments(shipments)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if err != nil {
				t.Fatalf("PrintShipments failed: %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Output should contain '%s', but got: %s", expected, output)
				}
			}
		})
	}
}

func TestOutputFormatterPrintSuccess(t *testing.T) {
	tests := []struct {
		name     string
		quiet    bool
		message  string
		expected string
	}{
		{
			name:     "normal mode",
			quiet:    false,
			message:  "Operation successful",
			expected: "✓ Operation successful",
		},
		{
			name:     "quiet mode",
			quiet:    true,
			message:  "Operation successful",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			formatter := NewOutputFormatter("table", tt.quiet)
			formatter.PrintSuccess(tt.message)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if tt.expected == "" {
				if output != "" {
					t.Errorf("Expected no output in quiet mode, but got: %s", output)
				}
			} else {
				if !strings.Contains(output, tt.expected) {
					t.Errorf("Expected output to contain '%s', but got: %s", tt.expected, output)
				}
			}
		})
	}
}

func TestTruncateFunction(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly ten chars", 17, "exactly ten chars"},
		{"this is a very long string that should be truncated", 20, "this is a very lo..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "..."},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}