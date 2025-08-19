package recovery

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/forest6511/gdl/internal/network"
	"github.com/forest6511/gdl/internal/storage"
	"github.com/forest6511/gdl/pkg/errors"
)

func TestNewRecoveryAdvisor(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	if advisor == nil {
		t.Fatal("NewRecoveryAdvisor() returned nil")
	}

	if advisor.networkDiagnostics == nil {
		t.Error("networkDiagnostics should not be nil")
	}

	if advisor.spaceChecker == nil {
		t.Error("spaceChecker should not be nil")
	}

	if advisor.analysisHistory == nil {
		t.Error("analysisHistory should not be nil")
	}

	if len(advisor.userAgents) == 0 {
		t.Error("userAgents should not be empty")
	}

	if len(advisor.mirrorSources) == 0 {
		t.Error("mirrorSources should not be empty")
	}

	if advisor.maxHistorySize != 50 {
		t.Errorf("maxHistorySize = %d, want 50", advisor.maxHistorySize)
	}
}

func TestRecoveryAdvisor_AnalyzeFailure(t *testing.T) {
	advisor := NewRecoveryAdvisor()
	ctx := context.Background()

	testCases := []struct {
		name             string
		err              error
		url              string
		expectedType     FailureType
		expectedCode     errors.ErrorCode
		bytesTransferred int64
		totalSize        int64
	}{
		{
			name: "Network error",
			err: &errors.DownloadError{
				Code:    errors.CodeNetworkError,
				Message: "Connection failed",
			},
			url:          "https://example.com/file.zip",
			expectedType: FailureNetwork,
			expectedCode: errors.CodeNetworkError,
		},
		{
			name: "HTTP error",
			err: &errors.DownloadError{
				Code:           errors.CodeServerError,
				HTTPStatusCode: 500,
				Message:        "Server error",
			},
			url:          "https://example.com/file.zip",
			expectedType: FailureServer,
			expectedCode: errors.CodeServerError,
		},
		{
			name: "Disk space error with partial download",
			err: &errors.DownloadError{
				Code:    errors.CodeInsufficientSpace,
				Message: "Not enough space",
			},
			url:              "https://example.com/file.zip",
			expectedType:     FailureDiskSpace,
			expectedCode:     errors.CodeInsufficientSpace,
			bytesTransferred: 1024,
			totalSize:        2048,
		},
		{
			name: "Timeout error",
			err: &errors.DownloadError{
				Code:    errors.CodeTimeout,
				Message: "Request timeout",
			},
			url:          "https://example.com/file.zip",
			expectedType: FailureTimeout,
			expectedCode: errors.CodeTimeout,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			analysis, err := advisor.AnalyzeFailure(
				ctx,
				tc.err,
				tc.url,
				1,
				tc.bytesTransferred,
				tc.totalSize,
				time.Second,
				nil,
			)
			if err != nil {
				t.Fatalf("AnalyzeFailure failed: %v", err)
			}

			if analysis == nil {
				t.Fatal("AnalyzeFailure returned nil analysis")
			}

			if analysis.FailureType != tc.expectedType {
				t.Errorf("FailureType = %v, want %v", analysis.FailureType, tc.expectedType)
			}

			if analysis.ErrorCode != tc.expectedCode {
				t.Errorf("ErrorCode = %v, want %v", analysis.ErrorCode, tc.expectedCode)
			}

			if analysis.URL != tc.url {
				t.Errorf("URL = %s, want %s", analysis.URL, tc.url)
			}

			if analysis.BytesTransferred != tc.bytesTransferred {
				t.Errorf(
					"BytesTransferred = %d, want %d",
					analysis.BytesTransferred,
					tc.bytesTransferred,
				)
			}

			if analysis.TotalSize != tc.totalSize {
				t.Errorf("TotalSize = %d, want %d", analysis.TotalSize, tc.totalSize)
			}

			if analysis.AttemptCount != 1 {
				t.Errorf("AttemptCount = %d, want 1", analysis.AttemptCount)
			}

			if len(analysis.EnvironmentContext) == 0 {
				t.Error("EnvironmentContext should not be empty")
			}
		})
	}
}

func TestRecoveryAdvisor_GenerateRecoveryRecommendation(t *testing.T) {
	advisor := NewRecoveryAdvisor()
	ctx := context.Background()

	testCases := []struct {
		name                    string
		analysis                *FailureAnalysis
		expectedActionCount     int
		expectedHasAlternatives bool
		expectedMinSuccessRate  float64
	}{
		{
			name: "Network failure",
			analysis: &FailureAnalysis{
				FailureType:  FailureNetwork,
				ErrorCode:    errors.CodeNetworkError,
				URL:          "https://example.com/file.zip",
				AttemptCount: 1,
				Error: &errors.DownloadError{
					Code:    errors.CodeNetworkError,
					Message: "Network error",
				},
			},
			expectedActionCount:     2, // At least network check and retry
			expectedHasAlternatives: false,
			expectedMinSuccessRate:  0.3,
		},
		{
			name: "HTTP 403 Forbidden",
			analysis: &FailureAnalysis{
				FailureType:    FailureHTTP,
				ErrorCode:      errors.CodeClientError,
				HTTPStatusCode: http.StatusForbidden,
				URL:            "https://example.com/file.zip",
				AttemptCount:   1,
				Error: &errors.DownloadError{
					Code:    errors.CodeClientError,
					Message: "Forbidden",
				},
			},
			expectedActionCount:     1, // User agent change
			expectedHasAlternatives: false,
			expectedMinSuccessRate:  0.3,
		},
		{
			name: "Disk space issue",
			analysis: &FailureAnalysis{
				FailureType:      FailureDiskSpace,
				ErrorCode:        errors.CodeInsufficientSpace,
				URL:              "https://example.com/file.zip",
				AttemptCount:     1,
				BytesTransferred: 1024,
				TotalSize:        2048,
				Error: &errors.DownloadError{
					Code:    errors.CodeInsufficientSpace,
					Message: "Insufficient space",
				},
				DiskSpaceInfo: &storage.SpaceInfo{
					TotalBytes:     10000,
					FreeBytes:      500,
					AvailableBytes: 500,
				},
			},
			expectedActionCount:     1, // Disk space check
			expectedHasAlternatives: false,
			expectedMinSuccessRate:  0.8, // Disk space issues are usually fixable
		},
		{
			name: "Partial download corruption",
			analysis: &FailureAnalysis{
				FailureType:      FailureCorruption,
				ErrorCode:        errors.CodeCorruptedData,
				URL:              "https://example.com/file.zip",
				AttemptCount:     2,
				BytesTransferred: 1024,
				TotalSize:        2048,
				Error: &errors.DownloadError{
					Code:    errors.CodeCorruptedData,
					Message: "Data corrupted",
				},
			},
			expectedActionCount:     2, // Repair partial file + clear cache
			expectedHasAlternatives: false,
			expectedMinSuccessRate:  0.5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recommendation := advisor.GenerateRecoveryRecommendation(ctx, tc.analysis)

			if recommendation == nil {
				t.Fatal("GenerateRecoveryRecommendation returned nil")
			}

			if len(recommendation.RecommendedActions) < tc.expectedActionCount {
				t.Errorf("RecommendedActions count = %d, want at least %d",
					len(recommendation.RecommendedActions), tc.expectedActionCount)
			}

			// Check actions are sorted by priority and confidence
			for i := 1; i < len(recommendation.RecommendedActions); i++ {
				prev := recommendation.RecommendedActions[i-1]
				curr := recommendation.RecommendedActions[i]

				if prev.Priority < curr.Priority {
					t.Errorf(
						"Actions should be sorted by priority: %v < %v",
						prev.Priority,
						curr.Priority,
					)
				} else if prev.Priority == curr.Priority && prev.Confidence < curr.Confidence {
					t.Errorf("Actions with same priority should be sorted by confidence: %f < %f",
						prev.Confidence, curr.Confidence)
				}
			}

			if tc.expectedHasAlternatives {
				if len(recommendation.AlternativeStrategies) == 0 {
					t.Error("Expected alternative strategies but got none")
				}
			}

			if recommendation.EstimatedSuccessRate < tc.expectedMinSuccessRate {
				t.Errorf("EstimatedSuccessRate = %f, want at least %f",
					recommendation.EstimatedSuccessRate, tc.expectedMinSuccessRate)
			}

			if recommendation.GeneratedAt.IsZero() {
				t.Error("GeneratedAt should not be zero")
			}
		})
	}
}

func TestRecoveryAdvisor_GenerateNetworkActions(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	analysis := &FailureAnalysis{
		FailureType: FailureNetwork,
		ErrorCode:   errors.CodeNetworkError,
		URL:         "https://example.com/file.zip",
		NetworkConditions: &network.NetworkHealth{
			ProxyInfo: &network.ProxyInfo{
				Detected: true,
				Type:     network.ProxyHTTP,
			},
		},
	}

	actions := advisor.generateNetworkActions(analysis)

	if len(actions) == 0 {
		t.Fatal("generateNetworkActions returned no actions")
	}

	// Should include network connectivity check
	hasConnectivityCheck := false
	hasDirectConnection := false
	hasRetryWithDelay := false

	for _, action := range actions {
		switch action.Type {
		case ActionCheckNetworkConnectivity:
			hasConnectivityCheck = true
		case ActionTryDirectConnection:
			hasDirectConnection = true
		case ActionRetryWithDelay:
			hasRetryWithDelay = true
		}
	}

	if !hasConnectivityCheck {
		t.Error("Should include network connectivity check")
	}

	if !hasDirectConnection {
		t.Error("Should include direct connection attempt when proxy detected")
	}

	if !hasRetryWithDelay {
		t.Error("Should include retry with delay")
	}
}

func TestRecoveryAdvisor_GenerateHTTPActions(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	testCases := []struct {
		name           string
		httpStatus     int
		url            string
		expectedAction ActionType
	}{
		{
			name:           "Rate limited",
			httpStatus:     http.StatusTooManyRequests,
			expectedAction: ActionWaitAndRetry,
		},
		{
			name:           "Forbidden",
			httpStatus:     http.StatusForbidden,
			expectedAction: ActionChangeUserAgent,
		},
		{
			name:           "Bad Gateway",
			httpStatus:     http.StatusBadGateway,
			expectedAction: ActionTryMirrorURL,
		},
		{
			name:           "Range Not Satisfiable with partial download",
			httpStatus:     http.StatusRequestedRangeNotSatisfiable,
			expectedAction: ActionRepairPartialFile,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			analysis := &FailureAnalysis{
				FailureType:      FailureHTTP,
				ErrorCode:        errors.CodeClientError,
				HTTPStatusCode:   tc.httpStatus,
				URL:              "https://example.com/file.zip",
				BytesTransferred: 1024, // For range not satisfiable test
				Error: &errors.DownloadError{
					Code:    errors.CodeClientError,
					Message: "HTTP error",
				},
			}

			actions := advisor.generateHTTPActions(analysis)

			if len(actions) == 0 {
				t.Fatal("generateHTTPActions returned no actions")
			}

			hasExpectedAction := false

			for _, action := range actions {
				if action.Type == tc.expectedAction {
					hasExpectedAction = true
					break
				}
			}

			if !hasExpectedAction {
				t.Errorf("Expected action %v not found in actions", tc.expectedAction)
			}
		})
	}
}

func TestRecoveryAdvisor_FilterPreviousActions(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	originalActions := []RecoveryAction{
		{Type: ActionRetryWithDelay, Description: "Retry"},
		{Type: ActionChangeUserAgent, Description: "Change UA"},
		{Type: ActionCheckDiskSpace, Description: "Check space"},
	}

	previousActions := []ActionType{ActionRetryWithDelay, ActionCheckDiskSpace}

	filtered := advisor.filterPreviousActions(originalActions, previousActions)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 filtered action, got %d", len(filtered))
	}

	if filtered[0].Type != ActionChangeUserAgent {
		t.Errorf("Expected ActionChangeUserAgent, got %v", filtered[0].Type)
	}
}

func TestRecoveryAdvisor_CalculateSuccessRate(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	testCases := []struct {
		name            string
		analysis        *FailureAnalysis
		actions         []RecoveryAction
		expectedMinRate float64
		expectedMaxRate float64
	}{
		{
			name: "No actions",
			analysis: &FailureAnalysis{
				FailureType: FailureNetwork,
			},
			actions:         []RecoveryAction{},
			expectedMinRate: 0.05,
			expectedMaxRate: 0.15,
		},
		{
			name: "Disk space issue with high confidence action",
			analysis: &FailureAnalysis{
				FailureType:  FailureDiskSpace,
				AttemptCount: 1,
			},
			actions: []RecoveryAction{
				{Type: ActionCheckDiskSpace, Confidence: 0.9},
			},
			expectedMinRate: 0.85,
			expectedMaxRate: 0.95,
		},
		{
			name: "Multiple attempts reduce success rate",
			analysis: &FailureAnalysis{
				FailureType:  FailureNetwork,
				AttemptCount: 6,
			},
			actions: []RecoveryAction{
				{Type: ActionRetryWithDelay, Confidence: 0.5},
			},
			expectedMinRate: 0.05,
			expectedMaxRate: 0.4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rate := advisor.calculateSuccessRate(tc.analysis, tc.actions)

			if rate < tc.expectedMinRate {
				t.Errorf("Success rate %f is below minimum %f", rate, tc.expectedMinRate)
			}

			if rate > tc.expectedMaxRate {
				t.Errorf("Success rate %f is above maximum %f", rate, tc.expectedMaxRate)
			}

			if rate < 0.05 || rate > 0.95 {
				t.Errorf("Success rate %f should be capped between 0.05 and 0.95", rate)
			}
		})
	}
}

func TestRecoveryAdvisor_GenerateAlternativeStrategies(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	testCases := []struct {
		name             string
		analysis         *FailureAnalysis
		expectedContains []string
	}{
		{
			name: "HTTPS with known mirror",
			analysis: &FailureAnalysis{
				URL:          "https://github.com/user/repo/archive/main.zip",
				FailureType:  FailureHTTP,
				AttemptCount: 1,
			},
			expectedContains: []string{"Try alternative mirrors", "Try HTTP instead"},
		},
		{
			name: "Multiple attempts",
			analysis: &FailureAnalysis{
				URL:          "https://example.com/file.zip",
				FailureType:  FailureNetwork,
				AttemptCount: 3,
			},
			expectedContains: []string{
				"single-threaded",
				"different download client",
				"off-peak hours",
			},
		},
		{
			name: "Network failure",
			analysis: &FailureAnalysis{
				URL:         "https://example.com/file.zip",
				FailureType: FailureNetwork,
			},
			expectedContains: []string{"firewall", "different network", "VPN"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			strategies := advisor.generateAlternativeStrategies(tc.analysis)

			for _, expected := range tc.expectedContains {
				found := false

				for _, strategy := range strategies {
					if strings.Contains(strings.ToLower(strategy), strings.ToLower(expected)) {
						found = true
						break
					}
				}

				if !found {
					t.Errorf(
						"Expected strategy containing '%s' not found in: %v",
						expected,
						strategies,
					)
				}
			}
		})
	}
}

func TestRecoveryAdvisor_HistoryManagement(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	// Test adding to history
	analysis := FailureAnalysis{
		FailureType: FailureNetwork,
		URL:         "https://example.com/file.zip",
	}

	advisor.addToHistory(analysis)

	history := advisor.GetHistoricalAnalysis()
	if len(history) != 1 {
		t.Errorf("Expected 1 item in history, got %d", len(history))
	}

	// Test history size limit
	advisor.maxHistorySize = 3

	for i := 0; i < 5; i++ {
		analysis.URL = fmt.Sprintf("https://example.com/file%d.zip", i)
		advisor.addToHistory(analysis)
	}

	history = advisor.GetHistoricalAnalysis()
	if len(history) != 3 {
		t.Errorf("Expected 3 items in history (size limit), got %d", len(history))
	}

	// Test pattern analysis
	patterns := advisor.GetFailurePatterns()
	if len(patterns) == 0 {
		t.Error("Expected failure patterns but got none")
	}

	networkPattern := fmt.Sprintf("%s_%s", FailureNetwork.String(), errors.CodeUnknown.String())
	if count, exists := patterns[networkPattern]; !exists || count == 0 {
		t.Errorf("Expected network failure pattern to exist with count > 0")
	}

	// Test clearing history
	advisor.ClearHistory()

	history = advisor.GetHistoricalAnalysis()
	if len(history) != 0 {
		t.Errorf("Expected empty history after clear, got %d items", len(history))
	}
}

func TestActionType_String(t *testing.T) {
	testCases := []struct {
		actionType ActionType
		expected   string
	}{
		{ActionRetryWithDelay, "retry_with_delay"},
		{ActionChangeUserAgent, "change_user_agent"},
		{ActionReduceConcurrency, "reduce_concurrency"},
		{ActionTryHTTP, "try_http"},
		{ActionCheckDiskSpace, "check_disk_space"},
		{ActionTryMirrorURL, "try_mirror"},
		{ActionResumeDownload, "resume_download"},
		{ActionType(999), "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := tc.actionType.String()
			if result != tc.expected {
				t.Errorf("ActionType.String() = %s, want %s", result, tc.expected)
			}
		})
	}
}

func TestActionPriority_String(t *testing.T) {
	testCases := []struct {
		priority ActionPriority
		expected string
	}{
		{PriorityLow, "low"},
		{PriorityMedium, "medium"},
		{PriorityHigh, "high"},
		{PriorityCritical, "critical"},
		{ActionPriority(999), "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := tc.priority.String()
			if result != tc.expected {
				t.Errorf("ActionPriority.String() = %s, want %s", result, tc.expected)
			}
		})
	}
}

func TestFailureType_String(t *testing.T) {
	testCases := []struct {
		failureType FailureType
		expected    string
	}{
		{FailureNetwork, "network"},
		{FailureHTTP, "http"},
		{FailureFileSystem, "filesystem"},
		{FailureDiskSpace, "disk_space"},
		{FailureAuthentication, "authentication"},
		{FailureServer, "server"},
		{FailureTimeout, "timeout"},
		{FailureCorruption, "corruption"},
		{FailurePermission, "permission"},
		{FailureUnknown, "unknown"},
		{FailureType(999), "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := tc.failureType.String()
			if result != tc.expected {
				t.Errorf("FailureType.String() = %s, want %s", result, tc.expected)
			}
		})
	}
}

func TestRecoveryAdvisor_ClassifyGenericError(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	testCases := []struct {
		name         string
		err          error
		expectedCode errors.ErrorCode
	}{
		{
			name:         "Timeout error",
			err:          fmt.Errorf("connection timeout occurred"),
			expectedCode: errors.CodeTimeout,
		},
		{
			name:         "Network error",
			err:          fmt.Errorf("network connection failed"),
			expectedCode: errors.CodeNetworkError,
		},
		{
			name:         "Permission error",
			err:          fmt.Errorf("permission denied to access file"),
			expectedCode: errors.CodePermissionDenied,
		},
		{
			name:         "Disk space error",
			err:          fmt.Errorf("no space left on disk"),
			expectedCode: errors.CodeInsufficientSpace,
		},
		{
			name:         "File not found error",
			err:          fmt.Errorf("file not found on server"),
			expectedCode: errors.CodeFileNotFound,
		},
		{
			name:         "Unknown error",
			err:          fmt.Errorf("something unexpected happened"),
			expectedCode: errors.CodeUnknown,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code := advisor.classifyGenericError(tc.err)
			if code != tc.expectedCode {
				t.Errorf("classifyGenericError() = %v, want %v", code, tc.expectedCode)
			}
		})
	}
}

func TestRecoveryAdvisor_DetermineFailureType(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	testCases := []struct {
		name         string
		code         errors.ErrorCode
		httpStatus   int
		expectedType FailureType
	}{
		{
			name:         "Network error code",
			code:         errors.CodeNetworkError,
			expectedType: FailureNetwork,
		},
		{
			name:         "Timeout error code",
			code:         errors.CodeTimeout,
			expectedType: FailureTimeout,
		},
		{
			name:         "HTTP 404",
			code:         errors.CodeUnknown,
			httpStatus:   404,
			expectedType: FailureHTTP,
		},
		{
			name:         "HTTP 500",
			code:         errors.CodeUnknown,
			httpStatus:   500,
			expectedType: FailureHTTP,
		},
		{
			name:         "Unknown with no HTTP status",
			code:         errors.CodeUnknown,
			httpStatus:   0,
			expectedType: FailureUnknown,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			failureType := advisor.determineFailureType(tc.code, tc.httpStatus, nil)
			if failureType != tc.expectedType {
				t.Errorf("determineFailureType() = %v, want %v", failureType, tc.expectedType)
			}
		})
	}
}

func TestRecoveryAdvisor_IsSSLRelatedError(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "SSL handshake error",
			err:      fmt.Errorf("SSL handshake failed"),
			expected: true,
		},
		{
			name:     "Certificate error",
			err:      fmt.Errorf("x509: certificate verify failed"),
			expected: true,
		},
		{
			name:     "TLS error",
			err:      fmt.Errorf("TLS connection error"),
			expected: true,
		},
		{
			name:     "Regular network error",
			err:      fmt.Errorf("connection refused"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := advisor.isSSLRelatedError(tc.err)
			if result != tc.expected {
				t.Errorf("isSSLRelatedError() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestExtractHostFromURL(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "Valid HTTPS URL",
			url:      "https://example.com/path/file.zip",
			expected: "example.com",
		},
		{
			name:     "URL with port",
			url:      "http://example.com:8080/file.zip",
			expected: "example.com:8080",
		},
		{
			name:     "Invalid URL",
			url:      "not-a-valid-url",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractHostFromURL(tc.url)
			if result != tc.expected {
				t.Errorf("extractHostFromURL() = %s, want %s", result, tc.expected)
			}
		})
	}
}

func TestFormatBytes_Recovery(t *testing.T) {
	testCases := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "Zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "Bytes",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "Kilobytes",
			bytes:    1536,
			expected: "1.5 KB",
		},
		{
			name:     "Megabytes",
			bytes:    1048576,
			expected: "1.0 MB",
		},
		{
			name:     "Gigabytes",
			bytes:    1610612736,
			expected: "1.5 GB",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatBytes(tc.bytes)
			if result != tc.expected {
				t.Errorf("formatBytes() = %s, want %s", result, tc.expected)
			}
		})
	}
}

func TestGetCommonUserAgents(t *testing.T) {
	userAgents := getCommonUserAgents()

	if len(userAgents) == 0 {
		t.Fatal("getCommonUserAgents() returned empty slice")
	}

	// Check that we have various types of user agents
	hasChrome := false
	hasFirefox := false
	hasCurl := false

	for _, ua := range userAgents {
		if strings.Contains(ua, "Chrome") {
			hasChrome = true
		}

		if strings.Contains(ua, "Firefox") {
			hasFirefox = true
		}

		if strings.Contains(ua, "curl") {
			hasCurl = true
		}
	}

	if !hasChrome {
		t.Error("Should include Chrome user agents")
	}

	if !hasFirefox {
		t.Error("Should include Firefox user agents")
	}

	if !hasCurl {
		t.Error("Should include curl user agents")
	}
}

func TestGetKnownMirrors(t *testing.T) {
	mirrors := getKnownMirrors()

	if len(mirrors) == 0 {
		t.Fatal("getKnownMirrors() returned empty map")
	}

	expectedDomains := []string{"github.com", "sourceforge.net", "apache.org"}

	for _, domain := range expectedDomains {
		if mirrorList, exists := mirrors[domain]; !exists {
			t.Errorf("Expected mirrors for %s", domain)
		} else if len(mirrorList) == 0 {
			t.Errorf("Expected non-empty mirror list for %s", domain)
		}
	}
}

func TestRecoveryAdvisor_GetAlternativeUserAgent(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	// Test that it returns different user agents
	ua1 := advisor.getAlternativeUserAgent()

	if ua1 == "" {
		t.Error("getAlternativeUserAgent() should not return empty string")
	}

	// Test with empty user agents list
	advisor.userAgents = []string{}

	ua3 := advisor.getAlternativeUserAgent()
	if ua3 != "Mozilla/5.0 (compatible; GODL/1.0)" {
		t.Errorf("Expected fallback user agent, got: %s", ua3)
	}
}

func TestRecoveryAdvisor_GenerateFileSystemActions(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	// Test file system actions for file permission error
	analysis := &FailureAnalysis{
		URL:             "file:///test/file.txt",
		Error:           errors.NewDownloadError(errors.CodePermissionDenied, "permission denied"),
		FailureType:     FailureFileSystem,
		PreviousActions: []ActionType{},
	}

	actions := advisor.generateFileSystemActions(analysis)

	if len(actions) == 0 {
		t.Error("Expected file system actions to be generated")
	}

	// Check for expected action types
	foundCheckPermissions := false

	for _, action := range actions {
		switch action.Type {
		case ActionCheckPermissions:
			foundCheckPermissions = true
		}
	}

	if !foundCheckPermissions {
		t.Error("Expected CheckPermissions action")
	}

	// Verify all actions have proper fields
	for _, action := range actions {
		if action.Description == "" {
			t.Error("Action should have description")
		}

		if action.Priority == PriorityLow || action.Priority == PriorityMedium ||
			action.Priority == PriorityHigh || action.Priority == PriorityCritical {
			// Valid priority
		} else {
			t.Error("Action should have valid priority")
		}
	}
}

func TestRecoveryAdvisor_GenerateAuthActions(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	// Test authentication actions
	analysis := &FailureAnalysis{
		URL: "https://secure.example.com/file.zip",
		Error: errors.NewDownloadError(
			errors.CodeAuthenticationFailed,
			"401 Unauthorized",
		),
		FailureType:     FailureAuthentication,
		PreviousActions: []ActionType{},
	}

	actions := advisor.generateAuthActions(analysis)

	if len(actions) == 0 {
		t.Error("Expected auth actions to be generated")
	}

	// Check for expected action types
	foundChangeHeaders := false

	for _, action := range actions {
		switch action.Type {
		case ActionChangeHeaders:
			foundChangeHeaders = true
		}
	}

	if !foundChangeHeaders {
		t.Error("Expected ChangeHeaders action")
	}

	// Verify all actions have proper fields
	for _, action := range actions {
		if action.Description == "" {
			t.Error("Action should have description")
		}

		if action.Priority == PriorityLow || action.Priority == PriorityMedium ||
			action.Priority == PriorityHigh || action.Priority == PriorityCritical {
			// Valid priority
		} else {
			t.Error("Action should have valid priority")
		}
	}
}

func TestRecoveryAdvisor_GenerateServerActions(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	// Test server error actions
	analysis := &FailureAnalysis{
		URL: "https://example.com/file.zip",
		Error: errors.NewDownloadError(
			errors.CodeServerError,
			"500 Internal Server Error",
		),
		FailureType:     FailureServer,
		PreviousActions: []ActionType{},
	}

	actions := advisor.generateServerActions(analysis)

	if len(actions) == 0 {
		t.Error("Expected server actions to be generated")
	}

	// Check for expected action types
	foundWaitAndRetry := false
	foundTryMirrorURL := false

	for _, action := range actions {
		switch action.Type {
		case ActionWaitAndRetry:
			foundWaitAndRetry = true
		case ActionTryMirrorURL:
			foundTryMirrorURL = true
		}
	}

	if !foundWaitAndRetry {
		t.Error("Expected WaitAndRetry action")
	}

	if !foundTryMirrorURL {
		t.Error("Expected TryMirrorURL action")
	}

	// Verify all actions have proper fields
	for _, action := range actions {
		if action.Description == "" {
			t.Error("Action should have description")
		}

		if action.Priority == PriorityLow || action.Priority == PriorityMedium ||
			action.Priority == PriorityHigh || action.Priority == PriorityCritical {
			// Valid priority
		} else {
			t.Error("Action should have valid priority")
		}
	}
}

func TestRecoveryAdvisor_GenerateTimeoutActions(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	// Test timeout actions
	analysis := &FailureAnalysis{
		URL:             "https://slow.example.com/large-file.zip",
		Error:           errors.NewDownloadError(errors.CodeTimeout, "timeout"),
		FailureType:     FailureTimeout,
		PreviousActions: []ActionType{},
	}

	actions := advisor.generateTimeoutActions(analysis)

	if len(actions) == 0 {
		t.Error("Expected timeout actions to be generated")
	}

	// Check for expected action types
	foundChangeTimeout := false

	for _, action := range actions {
		switch action.Type {
		case ActionChangeTimeout:
			foundChangeTimeout = true
		}
	}

	if !foundChangeTimeout {
		t.Error("Expected ChangeTimeout action")
	}

	// Verify all actions have proper fields
	for _, action := range actions {
		if action.Description == "" {
			t.Error("Action should have description")
		}

		if action.Priority == PriorityLow || action.Priority == PriorityMedium ||
			action.Priority == PriorityHigh || action.Priority == PriorityCritical {
			// Valid priority
		} else {
			t.Error("Action should have valid priority")
		}
	}
}

func TestFailureType_String_Complete(t *testing.T) {
	// Test all enum values including ones that might be missing coverage
	testCases := []struct {
		failureType FailureType
		expected    string
	}{
		{FailureNetwork, "network"},
		{FailureHTTP, "http"},
		{FailureFileSystem, "filesystem"},
		{FailureDiskSpace, "disk_space"},
		{FailureAuthentication, "authentication"},
		{FailureServer, "server"},
		{FailureTimeout, "timeout"},
		{FailureCorruption, "corruption"},
		{FailurePermission, "permission"},
		{FailureUnknown, "unknown"},
		{FailureType(999), "unknown"}, // Unknown failure type
	}

	for _, tc := range testCases {
		result := tc.failureType.String()
		if result != tc.expected {
			t.Errorf("FailureType(%d).String() = %s, want %s", tc.failureType, result, tc.expected)
		}
	}
}

func TestDetermineFailureType_Complete(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	testCases := []struct {
		name       string
		errorCode  errors.ErrorCode
		httpStatus int
		err        error
		expected   FailureType
	}{
		{
			"Network error",
			errors.CodeNetworkError,
			0,
			errors.NewDownloadError(errors.CodeNetworkError, "network unreachable"),
			FailureNetwork,
		},
		{
			"HTTP client error",
			errors.CodeClientError,
			404,
			errors.NewDownloadError(errors.CodeClientError, "404 not found"),
			FailureHTTP,
		},
		{
			"HTTP server error",
			errors.CodeServerError,
			500,
			errors.NewDownloadError(errors.CodeServerError, "500 internal server error"),
			FailureServer,
		},
		{
			"Permission denied",
			errors.CodePermissionDenied,
			0,
			errors.NewDownloadError(errors.CodePermissionDenied, "permission denied"),
			FailurePermission,
		},
		{
			"File not found",
			errors.CodeFileNotFound,
			0,
			errors.NewDownloadError(errors.CodeFileNotFound, "file not found"),
			FailureFileSystem,
		},
		{
			"Authentication failed",
			errors.CodeAuthenticationFailed,
			401,
			errors.NewDownloadError(errors.CodeAuthenticationFailed, "unauthorized"),
			FailureAuthentication,
		},
		{
			"Timeout error",
			errors.CodeTimeout,
			0,
			errors.NewDownloadError(errors.CodeTimeout, "timeout"),
			FailureTimeout,
		},
		{
			"Corrupted data",
			errors.CodeCorruptedData,
			0,
			errors.NewDownloadError(errors.CodeCorruptedData, "checksum mismatch"),
			FailureCorruption,
		},
		{
			"Insufficient space",
			errors.CodeInsufficientSpace,
			0,
			errors.NewDownloadError(errors.CodeInsufficientSpace, "not enough disk space"),
			FailureDiskSpace,
		},
		{
			"Unknown error code",
			errors.ErrorCode(999),
			0,
			errors.NewDownloadError(errors.ErrorCode(999), "unknown error"),
			FailureUnknown,
		},
		{
			"Generic error",
			errors.CodeUnknown,
			0,
			fmt.Errorf("generic error"),
			FailureUnknown,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := advisor.determineFailureType(tc.errorCode, tc.httpStatus, tc.err)
			if result != tc.expected {
				t.Errorf(
					"determineFailureType(%v, %d, %v) = %v, want %v",
					tc.errorCode,
					tc.httpStatus,
					tc.err,
					result,
					tc.expected,
				)
			}
		})
	}
}

func TestExtractHostFromURL_EdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected string
	}{
		{
			"Valid HTTP URL",
			"http://example.com/path",
			"example.com",
		},
		{
			"Valid HTTPS URL with port",
			"https://example.com:8080/path",
			"example.com:8080",
		},
		{
			"Invalid URL",
			"not-a-url",
			"",
		},
		{
			"Empty URL",
			"",
			"",
		},
		{
			"URL without scheme",
			"example.com/path",
			"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractHostFromURL(tc.url)
			if result != tc.expected {
				t.Errorf("extractHostFromURL(%s) = %s, want %s", tc.url, result, tc.expected)
			}
		})
	}
}

func TestCalculateSuccessRate_EdgeCases(t *testing.T) {
	advisor := NewRecoveryAdvisor()

	// Create a basic analysis for testing
	analysis := &FailureAnalysis{
		URL:         "https://example.com",
		Error:       errors.NewDownloadError(errors.CodeNetworkError, "test error"),
		FailureType: FailureNetwork,
	}

	// Test with empty actions
	emptyActions := []RecoveryAction{}

	result := advisor.calculateSuccessRate(analysis, emptyActions)
	if result < 0 || result > 100 {
		t.Errorf("calculateSuccessRate with empty actions = %f, want between 0 and 100", result)
	}

	// Test with some actions
	actions := []RecoveryAction{
		{Type: ActionRetryWithDelay, Priority: PriorityHigh},
		{Type: ActionChangeUserAgent, Priority: PriorityMedium},
	}

	result2 := advisor.calculateSuccessRate(analysis, actions)
	if result2 < 0 || result2 > 100 {
		t.Errorf("calculateSuccessRate with actions = %f, want between 0 and 100", result2)
	}
}

func TestFormatBytes_Recovery_Complete(t *testing.T) {
	testCases := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
		{1099511627776, "1.0 TB"},
		{1649267441664, "1.5 TB"},
		{1125899906842624, "1.0 PB"},
		{2251799813685248, "2.0 PB"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%d bytes", tc.bytes), func(t *testing.T) {
			result := formatBytes(tc.bytes)
			if result != tc.expected {
				t.Errorf("formatBytes(%d) = %s, want %s", tc.bytes, result, tc.expected)
			}
		})
	}
}
