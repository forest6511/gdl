package ui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestGetPromptPrefix(t *testing.T) {
	tests := []struct {
		name       string
		promptType PromptType
		wantPrefix string
	}{
		{"Info", PromptTypeInfo, Colorize(Blue, "ℹ")},
		{"Warning", PromptTypeWarning, Colorize(Yellow, "⚠")},
		{"Error", PromptTypeError, Colorize(Red, "✗")},
		{"Success", PromptTypeSuccess, Colorize(Green, "✓")},
		{"Question", PromptTypeQuestion, Colorize(Cyan, "?")},
		{"Unknown", PromptType(99), ">"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPromptPrefix(tt.promptType)
			if got != tt.wantPrefix {
				t.Errorf("getPromptPrefix(%v) = %v, want %v", tt.promptType, got, tt.wantPrefix)
			}
		})
	}
}

func TestAlert(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		alertType PromptType
	}{
		{"Info Alert", "This is info", PromptTypeInfo},
		{"Warning Alert", "This is warning", PromptTypeWarning},
		{"Error Alert", "This is error", PromptTypeError},
		{"Success Alert", "This is success", PromptTypeSuccess},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			Alert(tt.message, tt.alertType)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)

			output := buf.String()
			if !strings.Contains(output, tt.message) {
				t.Errorf("Alert output should contain message %q", tt.message)
			}
		})
	}
}

func TestAlertHelpers(t *testing.T) {
	testCases := []struct {
		name    string
		fn      func(string)
		message string
	}{
		{"AlertInfo", AlertInfo, "Info message"},
		{"AlertWarning", AlertWarning, "Warning message"},
		{"AlertError", AlertError, "Error message"},
		{"AlertSuccess", AlertSuccess, "Success message"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			tc.fn(tc.message)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)

			output := buf.String()
			if !strings.Contains(output, tc.message) {
				t.Errorf("%s output should contain message %q", tc.name, tc.message)
			}
		})
	}
}

func TestShowProgressDisplay(t *testing.T) {
	testCases := []struct {
		name    string
		current int64
		total   int64
		message string
	}{
		{"0%", 0, 100, "Starting"},
		{"50%", 50, 100, "Halfway"},
		{"100%", 100, 100, "Complete"},
		{"25%", 25, 100, "Quarter done"},
		{"75%", 75, 100, "Three quarters"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			ShowProgress(tc.current, tc.total, tc.message)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)

			output := buf.String()
			if !strings.Contains(output, tc.message) {
				t.Errorf("ShowProgress output should contain message %q", tc.message)
			}

			expectedPercentage := float64(tc.current) / float64(tc.total) * 100
			if !strings.Contains(output, strings.TrimSpace(strings.Split(strings.TrimSuffix(formatFloat(expectedPercentage), "0"), ".")[0])) {
				t.Errorf("ShowProgress output should show correct percentage")
			}
		})
	}
}

func TestPromptYesNo(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		defaultYes bool
		want       bool
	}{
		{"Yes response", "yes\n", false, true},
		{"Y response", "y\n", false, true},
		{"No response", "no\n", true, false},
		{"N response", "n\n", true, false},
		{"Empty default yes", "\n", true, true},
		{"Empty default no", "\n", false, false},
		{"Invalid defaults to no", "maybe\n", false, false},
		{"Invalid defaults to yes", "maybe\n", true, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock stdin
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			// Mock stdout to suppress output
			oldStdout := os.Stdout
			_, wOut, _ := os.Pipe()
			os.Stdout = wOut

			// Write test input
			go func() {
				w.Write([]byte(tc.input))
				w.Close()
			}()

			got := PromptYesNo("Test prompt", tc.defaultYes)

			// Restore
			os.Stdin = oldStdin
			os.Stdout = oldStdout
			wOut.Close()

			if got != tc.want {
				t.Errorf("PromptYesNo(%q, %v) = %v, want %v", tc.input, tc.defaultYes, got, tc.want)
			}
		})
	}
}

func TestPromptChoice(t *testing.T) {
	choices := []string{"Option 1", "Option 2", "Option 3"}

	testCases := []struct {
		name          string
		input         string
		defaultChoice int
		want          int
	}{
		{"Select 1", "1\n", 0, 0},
		{"Select 2", "2\n", 0, 1},
		{"Select 3", "3\n", 0, 2},
		{"Empty uses default", "\n", 1, 1},
		{"Invalid number uses default", "4\n", 1, 1},
		{"Invalid input uses default", "abc\n", 2, 2},
		{"Zero uses default", "0\n", 1, 1},
		{"Negative uses default", "-1\n", 0, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock stdin
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			// Mock stdout to suppress output
			oldStdout := os.Stdout
			_, wOut, _ := os.Pipe()
			os.Stdout = wOut

			// Write test input
			go func() {
				w.Write([]byte(tc.input))
				w.Close()
			}()

			got := PromptChoice("Choose an option", choices, tc.defaultChoice)

			// Restore
			os.Stdin = oldStdin
			os.Stdout = oldStdout
			wOut.Close()

			if got != tc.want {
				t.Errorf("PromptChoice(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestPrompt(t *testing.T) {
	testCases := []struct {
		name       string
		message    string
		promptType PromptType
		input      string
		want       string
	}{
		{"Info prompt", "Enter info", PromptTypeInfo, "test info\n", "test info"},
		{"Warning prompt", "Enter warning", PromptTypeWarning, "test warning\n", "test warning"},
		{"Error prompt", "Enter error", PromptTypeError, "test error\n", "test error"},
		{"Success prompt", "Enter success", PromptTypeSuccess, "test success\n", "test success"},
		{"Question prompt", "Enter answer", PromptTypeQuestion, "test answer\n", "test answer"},
		{"Whitespace trimmed", "Enter text", PromptTypeInfo, "  test  \n", "test"},
		{"Empty input", "Enter text", PromptTypeInfo, "\n", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock stdin
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			// Mock stdout to suppress output
			oldStdout := os.Stdout
			_, wOut, _ := os.Pipe()
			os.Stdout = wOut

			// Write test input
			go func() {
				w.Write([]byte(tc.input))
				w.Close()
			}()

			got := Prompt(tc.message, tc.promptType)

			// Restore
			os.Stdin = oldStdin
			os.Stdout = oldStdout
			wOut.Close()

			if got != tc.want {
				t.Errorf("Prompt(%q, %v) = %q, want %q", tc.message, tc.promptType, got, tc.want)
			}
		})
	}
}

func TestPromptPassword(t *testing.T) {
	testCases := []struct {
		name     string
		message  string
		input    string
		expected string
	}{
		{"Simple password", "Enter password", "secret123\n", "secret123"},
		{"Password with spaces", "Enter password", "my secret pass\n", "my secret pass"},
		{"Empty password", "Enter password", "\n", ""},
		{"Password with special chars", "Enter password", "p@$$w0rd!\n", "p@$$w0rd!"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock stdin
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			// Mock stdout to suppress output
			oldStdout := os.Stdout
			_, wOut, _ := os.Pipe()
			os.Stdout = wOut

			// Write test input
			go func() {
				w.Write([]byte(tc.input))
				w.Close()
			}()

			got := PromptPassword(tc.message)

			// Restore
			os.Stdin = oldStdin
			os.Stdout = oldStdout
			wOut.Close()

			if got != tc.expected {
				t.Errorf("PromptPassword(%q) = %q, want %q", tc.message, got, tc.expected)
			}
		})
	}
}

// Helper function to format float
func formatFloat(f float64) string {
	// Simple float to string conversion for percentage display
	intVal := int(f)
	return strings.TrimRight(strings.TrimRight(
		strings.TrimSuffix(fmt.Sprintf("%d", intVal), ".0"), "0"), ".")
}
