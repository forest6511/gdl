package errors

import (
	"strings"
	"testing"
)

func TestSuggestionProvider_GetSuggestion(t *testing.T) {
	provider := NewSuggestionProvider()

	tests := []struct {
		name         string
		errorCode    ErrorCode
		wantContains []string
	}{
		{
			name:      "Network error",
			errorCode: CodeNetworkError, // This maps to connection refused in our implementation
			wantContains: []string{
				"Cannot establish connection",
				"server is online",
			},
		},
		{
			name:      "File exists error",
			errorCode: CodeFileExists,
			wantContains: []string{
				"Use --force",
				"different output",
			},
		},
		{
			name:      "Permission denied error",
			errorCode: CodePermissionDenied,
			wantContains: []string{
				"write permissions",
				"different output",
			},
		},
		{
			name:      "Invalid URL error",
			errorCode: CodeInvalidURL,
			wantContains: []string{
				"URL is not valid",
				"http://",
			},
		},
		{
			name:      "Unknown error code",
			errorCode: ErrorCode(999), // Non-existent code
			wantContains: []string{
				"error occurred",
				"Try the download again",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := provider.GetSuggestion(tt.errorCode)
			if suggestion == nil {
				t.Fatal("Expected suggestion, got nil")
			}

			formatted := provider.FormatSuggestion(suggestion)

			for _, want := range tt.wantContains {
				if !strings.Contains(formatted, want) {
					t.Errorf(
						"Expected formatted suggestion to contain %q, got:\n%s",
						want,
						formatted,
					)
				}
			}
		})
	}
}

func TestSuggestionProvider_GetNetworkSuggestion(t *testing.T) {
	provider := NewSuggestionProvider()

	tests := []struct {
		name        string
		code        NetworkErrorCode
		wantSummary string
		wantSteps   int
	}{
		{
			name:        "DNS not found",
			code:        NetworkDNSNotFound,
			wantSummary: "domain name could not be resolved",
			wantSteps:   4,
		},
		{
			name:        "Connection refused",
			code:        NetworkConnectionRefused,
			wantSummary: "Cannot establish connection",
			wantSteps:   4,
		},
		{
			name:        "TLS handshake failure",
			code:        NetworkTLSHandshakeFailure,
			wantSummary: "SSL/TLS handshake failed",
			wantSteps:   4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := provider.getNetworkSuggestion(tt.code)
			if suggestion == nil {
				t.Fatal("Expected suggestion, got nil")
			}

			if !strings.Contains(
				strings.ToLower(suggestion.Summary),
				strings.ToLower(tt.wantSummary),
			) {
				t.Errorf(
					"Expected summary to contain %q, got %q",
					tt.wantSummary,
					suggestion.Summary,
				)
			}

			if len(suggestion.Steps) != tt.wantSteps {
				t.Errorf("Expected %d steps, got %d", tt.wantSteps, len(suggestion.Steps))
			}
		})
	}
}

func TestSuggestionProvider_GetHTTPSuggestion(t *testing.T) {
	provider := NewSuggestionProvider()

	tests := []struct {
		name        string
		code        HTTPErrorCode
		wantSummary string
	}{
		{
			name:        "Not found",
			code:        HTTPNotFound,
			wantSummary: "not found on the server",
		},
		{
			name:        "Unauthorized",
			code:        HTTPUnauthorized,
			wantSummary: "Authentication is required",
		},
		{
			name:        "Too many requests",
			code:        HTTPTooManyRequests,
			wantSummary: "too many requests",
		},
		{
			name:        "Internal server error",
			code:        HTTPInternalServerError,
			wantSummary: "technical difficulties",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := provider.getHTTPSuggestion(tt.code)
			if suggestion == nil {
				t.Fatal("Expected suggestion, got nil")
			}

			if !strings.Contains(
				strings.ToLower(suggestion.Summary),
				strings.ToLower(tt.wantSummary),
			) {
				t.Errorf(
					"Expected summary to contain %q, got %q",
					tt.wantSummary,
					suggestion.Summary,
				)
			}
		})
	}
}

func TestSuggestionProvider_GetFileSystemSuggestion(t *testing.T) {
	provider := NewSuggestionProvider()

	tests := []struct {
		name         string
		code         FileSystemErrorCode
		wantExamples bool
	}{
		{
			name:         "Permission denied",
			code:         FSPermissionDenied,
			wantExamples: true,
		},
		{
			name:         "Insufficient space",
			code:         FSInsufficientSpace,
			wantExamples: true,
		},
		{
			name:         "File exists",
			code:         FSFileExists,
			wantExamples: true,
		},
		{
			name:         "Directory not found",
			code:         FSDirectoryNotFound,
			wantExamples: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := provider.getFileSystemSuggestion(tt.code)
			if suggestion == nil {
				t.Fatal("Expected suggestion, got nil")
			}

			if tt.wantExamples && len(suggestion.Examples) == 0 {
				t.Error("Expected examples in suggestion")
			}

			if len(suggestion.Steps) == 0 {
				t.Error("Expected recovery steps in suggestion")
			}
		})
	}
}

func TestSuggestionProvider_GetValidationSuggestion(t *testing.T) {
	provider := NewSuggestionProvider()

	tests := []struct {
		name         string
		code         ValidationErrorCode
		wantExamples bool
	}{
		{
			name:         "Invalid URL",
			code:         ValidationInvalidURL,
			wantExamples: true,
		},
		{
			name:         "Unsupported scheme",
			code:         ValidationUnsupportedScheme,
			wantExamples: false,
		},
		{
			name:         "Empty URL",
			code:         ValidationEmptyURL,
			wantExamples: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := provider.getValidationSuggestion(tt.code)
			if suggestion == nil {
				t.Fatal("Expected suggestion, got nil")
			}

			if tt.wantExamples && len(suggestion.Examples) == 0 {
				t.Error("Expected examples in suggestion")
			}

			if len(suggestion.Steps) == 0 {
				t.Error("Expected recovery steps in suggestion")
			}
		})
	}
}

func TestSuggestionProvider_GetSuggestionForError(t *testing.T) {
	provider := NewSuggestionProvider()

	tests := []struct {
		name      string
		err       error
		wantIsNil bool
	}{
		{
			name: "DownloadError",
			err: &DownloadError{
				Code:    CodeInvalidURL,
				Message: "Invalid URL format",
			},
			wantIsNil: false,
		},
		{
			name:      "Generic error",
			err:       NewDownloadError(CodeNetworkError, "Network failure"),
			wantIsNil: false,
		},
		{
			name:      "Non-DownloadError",
			err:       NewDownloadError(CodeUnknown, "Unknown error"),
			wantIsNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := provider.GetSuggestionForError(tt.err)

			if tt.wantIsNil {
				if suggestion != nil {
					t.Error("Expected nil suggestion")
				}
			} else {
				if suggestion == nil {
					t.Error("Expected non-nil suggestion")
				}
			}
		})
	}
}

func TestSuggestionProvider_FormatSuggestion(t *testing.T) {
	provider := NewSuggestionProvider()

	suggestion := &Suggestion{
		Summary:  "Test error occurred",
		Steps:    []string{"Step 1", "Step 2"},
		Tips:     []string{"Tip 1", "Tip 2"},
		Examples: []string{"example command"},
		DocLinks: []string{"https://example.com/docs"},
	}

	formatted := provider.FormatSuggestion(suggestion)

	expectedElements := []string{
		"💡 Test error occurred",
		"🔧 What to try:",
		"1. Step 1",
		"2. Step 2",
		"💭 Helpful tips:",
		"• Tip 1",
		"• Tip 2",
		"📋 Examples:",
		"example command",
		"📚 More information:",
		"https://example.com/docs",
	}

	for _, element := range expectedElements {
		if !strings.Contains(formatted, element) {
			t.Errorf("Expected formatted suggestion to contain %q, got:\n%s", element, formatted)
		}
	}
}

func TestSuggestionProvider_FormatSuggestion_Nil(t *testing.T) {
	provider := NewSuggestionProvider()

	formatted := provider.FormatSuggestion(nil)
	if formatted != "" {
		t.Errorf("Expected empty string for nil suggestion, got %q", formatted)
	}
}

func TestSuggestionProvider_GetTroubleshootingSteps(t *testing.T) {
	provider := NewSuggestionProvider()

	steps := provider.GetTroubleshootingSteps()

	if len(steps) == 0 {
		t.Error("Expected troubleshooting steps, got none")
	}

	// Check that some expected steps are present
	expectedSteps := []string{
		"internet connection",
		"verbose flag",
		"timeout flag",
		"disk space",
		"permissions",
	}

	stepsText := strings.Join(steps, " ")
	for _, expected := range expectedSteps {
		if !strings.Contains(strings.ToLower(stepsText), expected) {
			t.Errorf("Expected troubleshooting steps to contain %q", expected)
		}
	}
}

func TestSuggestionProvider_GetQuickFixes(t *testing.T) {
	provider := NewSuggestionProvider()

	fixes := provider.GetQuickFixes()

	if len(fixes) == 0 {
		t.Error("Expected quick fixes, got none")
	}

	// Check that some expected fixes are present
	expectedFixes := []string{
		"File exists",
		"Permission denied",
		"Network timeout",
		"Invalid URL",
		"Server error",
	}

	for _, expected := range expectedFixes {
		if _, exists := fixes[expected]; !exists {
			t.Errorf("Expected quick fix for %q", expected)
		}
	}
}

func TestGenericSuggestion(t *testing.T) {
	provider := NewSuggestionProvider()

	suggestion := provider.getGenericSuggestion()

	if suggestion == nil {
		t.Fatal("Expected generic suggestion, got nil")
	}

	if suggestion.Summary == "" {
		t.Error("Expected non-empty summary")
	}

	if len(suggestion.Steps) == 0 {
		t.Error("Expected recovery steps")
	}

	if len(suggestion.Tips) == 0 {
		t.Error("Expected tips")
	}
}
