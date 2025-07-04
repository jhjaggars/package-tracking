package cmd

import (
	"os"
	"testing"

	"github.com/mattn/go-isatty"
	cliapi "package-tracking/internal/cli"
)

func TestShouldUseInteractiveMode(t *testing.T) {
	// Save original stdout for cleanup
	originalStdout := os.Stdout

	tests := []struct {
		name      string
		config    *cliapi.Config
		explicit  bool
		isTTY     bool
		expected  bool
	}{
		{
			name:     "explicit interactive mode requested",
			config:   &cliapi.Config{Format: "table", Quiet: false},
			explicit: true,
			isTTY:    true,
			expected: true,
		},
		{
			name:     "explicit interactive mode requested even with json format",
			config:   &cliapi.Config{Format: "json", Quiet: false},
			explicit: true,
			isTTY:    true,
			expected: true,
		},
		{
			name:     "explicit interactive mode requested even without TTY",
			config:   &cliapi.Config{Format: "table", Quiet: false},
			explicit: true,
			isTTY:    false,
			expected: true,
		},
		{
			name:     "auto-detect: table format, not quiet, TTY",
			config:   &cliapi.Config{Format: "table", Quiet: false},
			explicit: false,
			isTTY:    true,
			expected: true,
		},
		{
			name:     "auto-detect: json format should disable interactive",
			config:   &cliapi.Config{Format: "json", Quiet: false},
			explicit: false,
			isTTY:    true,
			expected: false,
		},
		{
			name:     "auto-detect: quiet mode should disable interactive",
			config:   &cliapi.Config{Format: "table", Quiet: true},
			explicit: false,
			isTTY:    true,
			expected: false,
		},
		{
			name:     "auto-detect: not a TTY should disable interactive",
			config:   &cliapi.Config{Format: "table", Quiet: false},
			explicit: false,
			isTTY:    false,
			expected: false,
		},
		{
			name:     "auto-detect: multiple disqualifying factors",
			config:   &cliapi.Config{Format: "json", Quiet: true},
			explicit: false,
			isTTY:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a test utility function - we'll implement the actual function later
			result := shouldUseInteractiveMode(tt.config, tt.explicit, tt.isTTY)
			if result != tt.expected {
				t.Errorf("shouldUseInteractiveMode(%+v, %t, %t) = %t, expected %t",
					tt.config, tt.explicit, tt.isTTY, result, tt.expected)
			}
		})
	}

	// Restore stdout
	os.Stdout = originalStdout
}

func TestIsTerminalDetection(t *testing.T) {
	// This test verifies that isatty detection works as expected
	// In test environment, we expect this to return false
	isTTY := isatty.IsTerminal(os.Stdout.Fd())
	
	// In most testing environments, this should be false
	// But we'll just verify the function works without error
	if isTTY {
		t.Logf("Running in a terminal environment: %t", isTTY)
	} else {
		t.Logf("Running in a non-terminal environment: %t", isTTY)
	}
}

// Note: shouldUseInteractiveMode is now implemented in list.go