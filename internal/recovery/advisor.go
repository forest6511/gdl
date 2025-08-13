// Package recovery provides intelligent failure analysis and recovery mechanisms.
package recovery

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/forest6511/godl/internal/network"
	"github.com/forest6511/godl/internal/storage"
	"github.com/forest6511/godl/pkg/errors"
)

// RecoveryAction represents a specific recovery action that can be taken.
const (
	unknownValue = "unknown"
)

type RecoveryAction struct {
	Type        ActionType        `json:"type"`
	Description string            `json:"description"`
	Priority    ActionPriority    `json:"priority"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Confidence  float64           `json:"confidence"` // 0.0 to 1.0
	Command     string            `json:"command,omitempty"`
	AutoApply   bool              `json:"auto_apply"`
}

// ActionType represents different types of recovery actions.
type ActionType int

const (
	// ActionRetryWithDelay retries the operation after a delay.
	ActionRetryWithDelay ActionType = iota

	// ActionChangeUserAgent changes the user agent string.
	ActionChangeUserAgent

	// ActionReduceConcurrency reduces concurrent connections.
	ActionReduceConcurrency

	// ActionChangeTimeout adjusts request timeout.
	ActionChangeTimeout

	// ActionTryHTTP falls back to HTTP from HTTPS.
	ActionTryHTTP

	// ActionCheckDiskSpace checks and frees disk space.
	ActionCheckDiskSpace

	// ActionCheckNetworkConnectivity tests network connectivity.
	ActionCheckNetworkConnectivity

	// ActionTryMirrorURL tries alternative mirror URLs.
	ActionTryMirrorURL

	// ActionResumeDownload attempts to resume partial download.
	ActionResumeDownload

	// ActionClearCache clears any cached data.
	ActionClearCache

	// ActionChangeHeaders modifies request headers.
	ActionChangeHeaders

	// ActionWaitAndRetry waits for server availability.
	ActionWaitAndRetry

	// ActionCheckPermissions verifies file system permissions.
	ActionCheckPermissions

	// ActionTryDirectConnection bypasses proxy settings.
	ActionTryDirectConnection

	// ActionRepairPartialFile attempts to repair corrupted partial downloads.
	ActionRepairPartialFile
)

// String returns the string representation of ActionType.
func (at ActionType) String() string {
	switch at {
	case ActionRetryWithDelay:
		return "retry_with_delay"
	case ActionChangeUserAgent:
		return "change_user_agent"
	case ActionReduceConcurrency:
		return "reduce_concurrency"
	case ActionChangeTimeout:
		return "change_timeout"
	case ActionTryHTTP:
		return "try_http"
	case ActionCheckDiskSpace:
		return "check_disk_space"
	case ActionCheckNetworkConnectivity:
		return "check_network"
	case ActionTryMirrorURL:
		return "try_mirror"
	case ActionResumeDownload:
		return "resume_download"
	case ActionClearCache:
		return "clear_cache"
	case ActionChangeHeaders:
		return "change_headers"
	case ActionWaitAndRetry:
		return "wait_and_retry"
	case ActionCheckPermissions:
		return "check_permissions"
	case ActionTryDirectConnection:
		return "try_direct"
	case ActionRepairPartialFile:
		return "repair_partial"
	default:
		return unknownValue
	}
}

// ActionPriority represents the priority level of a recovery action.
type ActionPriority int

const (
	// PriorityLow for actions that are worth trying but not urgent.
	PriorityLow ActionPriority = iota

	// PriorityMedium for actions that are likely to help.
	PriorityMedium

	// PriorityHigh for actions that are very likely to solve the problem.
	PriorityHigh

	// PriorityCritical for actions that must be taken immediately.
	PriorityCritical
)

// String returns the string representation of ActionPriority.
func (ap ActionPriority) String() string {
	switch ap {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return unknownValue
	}
}

// FailureAnalysis contains detailed analysis of a download failure.
type FailureAnalysis struct {
	Error              error                  `json:"-"`
	ErrorCode          errors.ErrorCode       `json:"error_code"`
	FailureType        FailureType            `json:"failure_type"`
	URL                string                 `json:"url"`
	HTTPStatusCode     int                    `json:"http_status_code,omitempty"`
	NetworkConditions  *network.NetworkHealth `json:"network_conditions,omitempty"`
	DiskSpaceInfo      *storage.SpaceInfo     `json:"disk_space_info,omitempty"`
	AttemptCount       int                    `json:"attempt_count"`
	BytesTransferred   int64                  `json:"bytes_transferred"`
	TotalSize          int64                  `json:"total_size"`
	Duration           time.Duration          `json:"duration"`
	PreviousActions    []ActionType           `json:"previous_actions,omitempty"`
	EnvironmentContext map[string]interface{} `json:"environment_context,omitempty"`
}

// FailureType categorizes the type of failure that occurred.
type FailureType int

const (
	// FailureNetwork indicates network-related failures.
	FailureNetwork FailureType = iota

	// FailureHTTP indicates HTTP protocol failures.
	FailureHTTP

	// FailureFileSystem indicates file system related failures.
	FailureFileSystem

	// FailureDiskSpace indicates insufficient disk space.
	FailureDiskSpace

	// FailureAuthentication indicates authentication failures.
	FailureAuthentication

	// FailureServer indicates server-side errors.
	FailureServer

	// FailureTimeout indicates timeout errors.
	FailureTimeout

	// FailureCorruption indicates data corruption.
	FailureCorruption

	// FailurePermission indicates permission errors.
	FailurePermission

	// FailureUnknown indicates unclassified failures.
	FailureUnknown
)

// String returns the string representation of FailureType.
func (ft FailureType) String() string {
	switch ft {
	case FailureNetwork:
		return "network"
	case FailureHTTP:
		return "http"
	case FailureFileSystem:
		return "filesystem"
	case FailureDiskSpace:
		return "disk_space"
	case FailureAuthentication:
		return "authentication"
	case FailureServer:
		return "server"
	case FailureTimeout:
		return "timeout"
	case FailureCorruption:
		return "corruption"
	case FailurePermission:
		return "permission"
	default:
		return unknownValue
	}
}

// RecoveryAdvisor provides intelligent failure analysis and recovery suggestions.
type RecoveryAdvisor struct {
	networkDiagnostics *network.Diagnostics
	spaceChecker       *storage.SpaceChecker
	analysisHistory    []FailureAnalysis
	userAgents         []string
	mirrorSources      map[string][]string // domain -> alternative mirrors
	maxHistorySize     int
}

// RecoveryRecommendation contains the complete recovery recommendation.
type RecoveryRecommendation struct {
	Analysis              FailureAnalysis  `json:"analysis"`
	RecommendedActions    []RecoveryAction `json:"recommended_actions"`
	AlternativeStrategies []string         `json:"alternative_strategies,omitempty"`
	EstimatedSuccessRate  float64          `json:"estimated_success_rate"`
	GeneratedAt           time.Time        `json:"generated_at"`
}

// NewRecoveryAdvisor creates a new recovery advisor instance.
func NewRecoveryAdvisor() *RecoveryAdvisor {
	return &RecoveryAdvisor{
		networkDiagnostics: network.NewDiagnostics(),
		spaceChecker:       storage.NewSpaceChecker(),
		analysisHistory:    []FailureAnalysis{},
		userAgents:         getCommonUserAgents(),
		mirrorSources:      getKnownMirrors(),
		maxHistorySize:     50,
	}
}

// AnalyzeFailure performs comprehensive failure analysis.
func (ra *RecoveryAdvisor) AnalyzeFailure(
	ctx context.Context,
	err error,
	url string,
	attemptCount int,
	bytesTransferred, totalSize int64,
	duration time.Duration,
	previousActions []ActionType,
) (*FailureAnalysis, error) {
	analysis := &FailureAnalysis{
		Error:              err,
		URL:                url,
		AttemptCount:       attemptCount,
		BytesTransferred:   bytesTransferred,
		TotalSize:          totalSize,
		Duration:           duration,
		PreviousActions:    previousActions,
		EnvironmentContext: make(map[string]interface{}),
	}

	// Analyze error type and code
	downloadErr := &errors.DownloadError{}
	if stderrors.As(err, &downloadErr) {
		analysis.ErrorCode = downloadErr.Code
		analysis.HTTPStatusCode = downloadErr.HTTPStatusCode
	} else {
		analysis.ErrorCode = ra.classifyGenericError(err)
	}

	// Determine failure type
	analysis.FailureType = ra.determineFailureType(analysis.ErrorCode, analysis.HTTPStatusCode, err)

	// Gather network diagnostics if it's a network-related failure
	if analysis.FailureType == FailureNetwork || analysis.FailureType == FailureHTTP ||
		analysis.FailureType == FailureTimeout {
		networkHealth, err := ra.networkDiagnostics.RunFullDiagnostics(
			ctx,
			&network.DiagnosticOptions{
				Timeout:          10 * time.Second,
				TestHosts:        []string{extractHostFromURL(url)},
				IncludeBandwidth: false,
				IncludeProxy:     true,
				Verbose:          false,
			},
		)
		if err == nil {
			analysis.NetworkConditions = networkHealth
		}
	}

	// Check disk space if it's a file system related failure
	if analysis.FailureType == FailureDiskSpace || analysis.FailureType == FailureFileSystem {
		spaceInfo, err := ra.spaceChecker.GetSpaceInfo(".")
		if err == nil {
			analysis.DiskSpaceInfo = spaceInfo
		}
	}

	// Add environment context
	ra.addEnvironmentContext(analysis)

	// Store in history
	ra.addToHistory(*analysis)

	return analysis, nil
}

// GenerateRecoveryRecommendation creates comprehensive recovery recommendations.
func (ra *RecoveryAdvisor) GenerateRecoveryRecommendation(
	ctx context.Context,
	analysis *FailureAnalysis,
) *RecoveryRecommendation {
	recommendation := &RecoveryRecommendation{
		Analysis:              *analysis,
		RecommendedActions:    []RecoveryAction{},
		AlternativeStrategies: []string{},
		GeneratedAt:           time.Now(),
	}

	// Generate specific recovery actions based on failure type
	actions := ra.generateActionsForFailureType(analysis)

	// Add general recovery actions
	actions = append(actions, ra.generateGeneralActions(analysis)...)

	// Filter out actions that have already been tried
	actions = ra.filterPreviousActions(actions, analysis.PreviousActions)

	// Sort actions by priority and confidence
	sort.Slice(actions, func(i, j int) bool {
		if actions[i].Priority != actions[j].Priority {
			return actions[i].Priority > actions[j].Priority
		}

		return actions[i].Confidence > actions[j].Confidence
	})

	recommendation.RecommendedActions = actions

	// Generate alternative strategies
	recommendation.AlternativeStrategies = ra.generateAlternativeStrategies(analysis)

	// Calculate estimated success rate
	recommendation.EstimatedSuccessRate = ra.calculateSuccessRate(analysis, actions)

	return recommendation
}

// generateActionsForFailureType creates failure-specific recovery actions.
func (ra *RecoveryAdvisor) generateActionsForFailureType(
	analysis *FailureAnalysis,
) []RecoveryAction {
	var actions []RecoveryAction

	switch analysis.FailureType {
	case FailureNetwork:
		actions = append(actions, ra.generateNetworkActions(analysis)...)
	case FailureHTTP:
		actions = append(actions, ra.generateHTTPActions(analysis)...)
	case FailureFileSystem, FailurePermission:
		actions = append(actions, ra.generateFileSystemActions(analysis)...)
	case FailureDiskSpace:
		actions = append(actions, ra.generateDiskSpaceActions(analysis)...)
	case FailureAuthentication:
		actions = append(actions, ra.generateAuthActions(analysis)...)
	case FailureServer:
		actions = append(actions, ra.generateServerActions(analysis)...)
	case FailureTimeout:
		actions = append(actions, ra.generateTimeoutActions(analysis)...)
	case FailureCorruption:
		actions = append(actions, ra.generateCorruptionActions(analysis)...)
	}

	return actions
}

// generateNetworkActions creates network-specific recovery actions.
func (ra *RecoveryAdvisor) generateNetworkActions(analysis *FailureAnalysis) []RecoveryAction {
	var actions []RecoveryAction

	// Check network connectivity
	actions = append(actions, RecoveryAction{
		Type:        ActionCheckNetworkConnectivity,
		Description: "Test network connectivity and diagnose connection issues",
		Priority:    PriorityHigh,
		Confidence:  0.8,
		AutoApply:   true,
	})

	// Try direct connection (bypass proxy)
	if analysis.NetworkConditions != nil && analysis.NetworkConditions.ProxyInfo != nil &&
		analysis.NetworkConditions.ProxyInfo.Detected {
		actions = append(actions, RecoveryAction{
			Type:        ActionTryDirectConnection,
			Description: "Bypass proxy settings and connect directly",
			Priority:    PriorityMedium,
			Confidence:  0.7,
			Parameters:  map[string]string{"no_proxy": "true"},
			Command:     "--no-proxy",
		})
	}

	// Retry with delay
	actions = append(actions, RecoveryAction{
		Type:        ActionRetryWithDelay,
		Description: "Wait for network conditions to improve and retry",
		Priority:    PriorityMedium,
		Confidence:  0.6,
		Parameters:  map[string]string{"delay": "30s"},
	})

	return actions
}

// generateHTTPActions creates HTTP-specific recovery actions.
func (ra *RecoveryAdvisor) generateHTTPActions(analysis *FailureAnalysis) []RecoveryAction {
	var actions []RecoveryAction

	switch analysis.HTTPStatusCode {
	case http.StatusTooManyRequests, http.StatusServiceUnavailable:
		actions = append(actions, RecoveryAction{
			Type: ActionWaitAndRetry,
			Description: fmt.Sprintf(
				"Server returned %d, wait for rate limit to reset",
				analysis.HTTPStatusCode,
			),
			Priority:   PriorityHigh,
			Confidence: 0.9,
			Parameters: map[string]string{"wait_time": "300s"},
		})

	case http.StatusForbidden:
		actions = append(actions, RecoveryAction{
			Type:        ActionChangeUserAgent,
			Description: "Change user agent string to bypass access restrictions",
			Priority:    PriorityHigh,
			Confidence:  0.8,
			Parameters:  map[string]string{"user_agent": ra.getAlternativeUserAgent()},
			Command:     "--user-agent",
		})

	case http.StatusBadGateway, http.StatusGatewayTimeout:
		actions = append(actions, RecoveryAction{
			Type:        ActionTryMirrorURL,
			Description: "Try alternative mirror or CDN endpoint",
			Priority:    PriorityHigh,
			Confidence:  0.7,
		})

	case http.StatusRequestedRangeNotSatisfiable:
		if analysis.BytesTransferred > 0 {
			actions = append(actions, RecoveryAction{
				Type:        ActionRepairPartialFile,
				Description: "Remove corrupted partial file and restart download",
				Priority:    PriorityHigh,
				Confidence:  0.8,
				AutoApply:   true,
			})
		}
	}

	// Try HTTPS to HTTP fallback for SSL issues
	if strings.HasPrefix(analysis.URL, "https://") && ra.isSSLRelatedError(analysis.Error) {
		actions = append(actions, RecoveryAction{
			Type:        ActionTryHTTP,
			Description: "Fallback to HTTP if HTTPS/SSL is causing issues",
			Priority:    PriorityMedium,
			Confidence:  0.6,
			Parameters: map[string]string{
				"fallback_url": strings.Replace(analysis.URL, "https://", "http://", 1),
			},
			Command: "--allow-http",
		})
	}

	return actions
}

// generateFileSystemActions creates file system specific recovery actions.
func (ra *RecoveryAdvisor) generateFileSystemActions(analysis *FailureAnalysis) []RecoveryAction {
	var actions []RecoveryAction

	actions = append(actions, RecoveryAction{
		Type:        ActionCheckPermissions,
		Description: "Verify and fix file system permissions",
		Priority:    PriorityHigh,
		Confidence:  0.9,
		AutoApply:   true,
	})

	// Suggest using --create-dirs if path doesn't exist
	if analysis.ErrorCode == errors.CodeFileNotFound {
		actions = append(actions, RecoveryAction{
			Type:        ActionChangeHeaders,
			Description: "Create parent directories automatically",
			Priority:    PriorityHigh,
			Confidence:  0.8,
			Command:     "--create-dirs",
		})
	}

	return actions
}

// generateDiskSpaceActions creates disk space specific recovery actions.
func (ra *RecoveryAdvisor) generateDiskSpaceActions(analysis *FailureAnalysis) []RecoveryAction {
	var actions []RecoveryAction

	actions = append(actions, RecoveryAction{
		Type:        ActionCheckDiskSpace,
		Description: "Free up disk space and clean temporary files",
		Priority:    PriorityCritical,
		Confidence:  0.9,
		AutoApply:   true,
	})

	// If we know the total size, suggest how much space is needed
	if analysis.TotalSize > 0 && analysis.DiskSpaceInfo != nil {
		needed := analysis.TotalSize - analysis.BytesTransferred
		// #nosec G115 -- AvailableBytes is system disk space which fits within int64 range
		available := int64(analysis.DiskSpaceInfo.AvailableBytes)

		if needed > available {
			shortfall := needed - available
			actions = append(actions, RecoveryAction{
				Type: ActionCheckDiskSpace,
				Description: fmt.Sprintf(
					"Free up at least %s of additional space",
					formatBytes(shortfall),
				),
				Priority:   PriorityCritical,
				Confidence: 0.95,
				Parameters: map[string]string{"required_space": fmt.Sprintf("%d", shortfall)},
			})
		}
	}

	return actions
}

// generateAuthActions creates authentication specific recovery actions.
func (ra *RecoveryAdvisor) generateAuthActions(analysis *FailureAnalysis) []RecoveryAction {
	var actions []RecoveryAction

	actions = append(actions, RecoveryAction{
		Type:        ActionChangeHeaders,
		Description: "Check authentication credentials and headers",
		Priority:    PriorityHigh,
		Confidence:  0.7,
	})

	// Suggest changing user agent for sites that block certain clients
	actions = append(actions, RecoveryAction{
		Type:        ActionChangeUserAgent,
		Description: "Use a different user agent to bypass client restrictions",
		Priority:    PriorityMedium,
		Confidence:  0.6,
		Parameters:  map[string]string{"user_agent": ra.getAlternativeUserAgent()},
		Command:     "--user-agent",
	})

	return actions
}

// generateServerActions creates server error specific recovery actions.
func (ra *RecoveryAdvisor) generateServerActions(analysis *FailureAnalysis) []RecoveryAction {
	var actions []RecoveryAction

	actions = append(actions, RecoveryAction{
		Type:        ActionWaitAndRetry,
		Description: "Wait for server to recover from temporary issues",
		Priority:    PriorityHigh,
		Confidence:  0.8,
		Parameters:  map[string]string{"wait_time": "120s"},
	})

	actions = append(actions, RecoveryAction{
		Type:        ActionTryMirrorURL,
		Description: "Try alternative server or mirror",
		Priority:    PriorityMedium,
		Confidence:  0.7,
	})

	return actions
}

// generateTimeoutActions creates timeout specific recovery actions.
func (ra *RecoveryAdvisor) generateTimeoutActions(analysis *FailureAnalysis) []RecoveryAction {
	var actions []RecoveryAction

	actions = append(actions, RecoveryAction{
		Type:        ActionChangeTimeout,
		Description: "Increase timeout values for slow connections",
		Priority:    PriorityHigh,
		Confidence:  0.8,
		Parameters:  map[string]string{"timeout": "300s"},
		Command:     "--timeout",
	})

	actions = append(actions, RecoveryAction{
		Type:        ActionReduceConcurrency,
		Description: "Reduce concurrent connections to avoid overwhelming the server",
		Priority:    PriorityMedium,
		Confidence:  0.7,
		Parameters:  map[string]string{"concurrent": "1"},
		Command:     "--concurrent",
	})

	return actions
}

// generateCorruptionActions creates data corruption specific recovery actions.
func (ra *RecoveryAdvisor) generateCorruptionActions(analysis *FailureAnalysis) []RecoveryAction {
	var actions []RecoveryAction

	if analysis.BytesTransferred > 0 {
		actions = append(actions, RecoveryAction{
			Type:        ActionRepairPartialFile,
			Description: "Remove corrupted partial download and restart",
			Priority:    PriorityHigh,
			Confidence:  0.9,
			AutoApply:   true,
		})
	}

	actions = append(actions, RecoveryAction{
		Type:        ActionClearCache,
		Description: "Clear any cached data that might be corrupted",
		Priority:    PriorityMedium,
		Confidence:  0.6,
		AutoApply:   true,
	})

	return actions
}

// generateGeneralActions creates general recovery actions applicable to all failures.
func (ra *RecoveryAdvisor) generateGeneralActions(analysis *FailureAnalysis) []RecoveryAction {
	var actions []RecoveryAction

	// If there's a partial download, suggest resuming
	if analysis.BytesTransferred > 0 && analysis.BytesTransferred < analysis.TotalSize {
		actions = append(actions, RecoveryAction{
			Type: ActionResumeDownload,
			Description: fmt.Sprintf(
				"Resume download from %s",
				formatBytes(analysis.BytesTransferred),
			),
			Priority:   PriorityHigh,
			Confidence: 0.8,
			Command:    "--resume",
		})
	}

	// Always offer basic retry
	if analysis.AttemptCount < 3 {
		actions = append(actions, RecoveryAction{
			Type:        ActionRetryWithDelay,
			Description: "Retry the download after a short delay",
			Priority:    PriorityMedium,
			Confidence:  0.5,
			Parameters:  map[string]string{"delay": "10s"},
		})
	}

	return actions
}

// generateAlternativeStrategies creates high-level alternative approaches.
func (ra *RecoveryAdvisor) generateAlternativeStrategies(analysis *FailureAnalysis) []string {
	var strategies []string

	// Check for mirror URLs
	host := extractHostFromURL(analysis.URL)
	if mirrors, exists := ra.mirrorSources[host]; exists && len(mirrors) > 0 {
		strategies = append(
			strategies,
			fmt.Sprintf("Try alternative mirrors: %s", strings.Join(mirrors, ", ")),
		)
	}

	// Suggest alternative protocols
	if strings.HasPrefix(analysis.URL, "https://") {
		strategies = append(strategies, "Try HTTP instead of HTTPS")
	}

	// Suggest different download methods
	if analysis.AttemptCount > 2 {
		strategies = append(strategies, "Try single-threaded download instead of concurrent")
		strategies = append(strategies, "Use a different download client or method")
		strategies = append(strategies, "Download during off-peak hours when server load is lower")
	}

	// Suggest network troubleshooting
	if analysis.FailureType == FailureNetwork {
		strategies = append(strategies, "Check firewall and antivirus settings")
		strategies = append(strategies, "Try a different network connection")
		strategies = append(strategies, "Use a VPN or proxy service")
	}

	return strategies
}

// Helper functions

// classifyGenericError attempts to classify non-DownloadError errors.
func (ra *RecoveryAdvisor) classifyGenericError(err error) errors.ErrorCode {
	errorStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errorStr, "timeout"):
		return errors.CodeTimeout
	case strings.Contains(errorStr, "network"), strings.Contains(errorStr, "connection"):
		return errors.CodeNetworkError
	case strings.Contains(errorStr, "permission"), strings.Contains(errorStr, "access"):
		return errors.CodePermissionDenied
	case strings.Contains(errorStr, "space"), strings.Contains(errorStr, "disk"):
		return errors.CodeInsufficientSpace
	case strings.Contains(errorStr, "not found"):
		return errors.CodeFileNotFound
	default:
		return errors.CodeUnknown
	}
}

// determineFailureType categorizes the failure type based on error code and HTTP status.
func (ra *RecoveryAdvisor) determineFailureType(
	code errors.ErrorCode,
	httpStatus int,
	err error,
) FailureType {
	switch code {
	case errors.CodeNetworkError:
		return FailureNetwork
	case errors.CodeTimeout:
		return FailureTimeout
	case errors.CodeInsufficientSpace:
		return FailureDiskSpace
	case errors.CodePermissionDenied:
		return FailurePermission
	case errors.CodeFileNotFound:
		return FailureFileSystem
	case errors.CodeAuthenticationFailed:
		return FailureAuthentication
	case errors.CodeCorruptedData:
		return FailureCorruption
	case errors.CodeServerError:
		return FailureServer
	case errors.CodeClientError:
		if httpStatus >= 400 && httpStatus < 500 {
			return FailureHTTP
		}

		return FailureUnknown
	default:
		if httpStatus >= 400 && httpStatus < 600 {
			return FailureHTTP
		}

		return FailureUnknown
	}
}

// isSSLRelatedError checks if the error is related to SSL/TLS issues.
func (ra *RecoveryAdvisor) isSSLRelatedError(err error) bool {
	errorStr := strings.ToLower(err.Error())
	sslKeywords := []string{"ssl", "tls", "certificate", "x509", "handshake"}

	for _, keyword := range sslKeywords {
		if strings.Contains(errorStr, keyword) {
			return true
		}
	}

	return false
}

// extractHostFromURL extracts the hostname from a URL.
func extractHostFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	return parsed.Host
}

// filterPreviousActions removes actions that have already been tried.
func (ra *RecoveryAdvisor) filterPreviousActions(
	actions []RecoveryAction,
	previousActions []ActionType,
) []RecoveryAction {
	if len(previousActions) == 0 {
		return actions
	}

	previousSet := make(map[ActionType]bool)
	for _, action := range previousActions {
		previousSet[action] = true
	}

	var filtered []RecoveryAction

	for _, action := range actions {
		if !previousSet[action.Type] {
			filtered = append(filtered, action)
		}
	}

	return filtered
}

// calculateSuccessRate estimates the likelihood of success based on the analysis and actions.
func (ra *RecoveryAdvisor) calculateSuccessRate(
	analysis *FailureAnalysis,
	actions []RecoveryAction,
) float64 {
	if len(actions) == 0 {
		return 0.1 // Very low if no actions available
	}

	// Base success rate depends on failure type and attempt count
	var baseRate float64

	switch analysis.FailureType {
	case FailureNetwork:
		baseRate = 0.6 // Network issues often resolve themselves
	case FailureHTTP:
		baseRate = 0.7 // HTTP errors often have clear solutions
	case FailureDiskSpace:
		baseRate = 0.9 // Disk space is usually fixable
	case FailureTimeout:
		baseRate = 0.8 // Timeout issues often fixable with settings
	case FailurePermission:
		baseRate = 0.8 // Permission issues are usually fixable
	case FailureCorruption:
		baseRate = 0.7 // Corruption can usually be resolved by restarting
	default:
		baseRate = 0.4
	}

	// Reduce success rate for repeated failures
	if analysis.AttemptCount > 3 {
		baseRate *= 0.8
	}

	if analysis.AttemptCount > 5 {
		baseRate *= 0.6
	}

	// Adjust based on action confidence
	if len(actions) > 0 {
		avgConfidence := 0.0
		for _, action := range actions {
			avgConfidence += action.Confidence
		}

		avgConfidence /= float64(len(actions))

		// Weight the base rate with action confidence
		baseRate = (baseRate + avgConfidence) / 2
	}

	// Cap between 0.05 and 0.95
	if baseRate < 0.05 {
		baseRate = 0.05
	}

	if baseRate > 0.95 {
		baseRate = 0.95
	}

	return baseRate
}

// addEnvironmentContext adds relevant environment information to the analysis.
func (ra *RecoveryAdvisor) addEnvironmentContext(analysis *FailureAnalysis) {
	analysis.EnvironmentContext["timestamp"] = time.Now().Format(time.RFC3339)
	analysis.EnvironmentContext["url_host"] = extractHostFromURL(analysis.URL)

	if analysis.BytesTransferred > 0 && analysis.TotalSize > 0 {
		progress := float64(analysis.BytesTransferred) / float64(analysis.TotalSize) * 100
		analysis.EnvironmentContext["download_progress"] = fmt.Sprintf("%.1f%%", progress)
	}

	analysis.EnvironmentContext["duration_seconds"] = analysis.Duration.Seconds()
}

// addToHistory adds the analysis to the historical record.
func (ra *RecoveryAdvisor) addToHistory(analysis FailureAnalysis) {
	ra.analysisHistory = append(ra.analysisHistory, analysis)

	// Limit history size
	if len(ra.analysisHistory) > ra.maxHistorySize {
		ra.analysisHistory = ra.analysisHistory[1:]
	}
}

// getAlternativeUserAgent returns a different user agent from the common list.
func (ra *RecoveryAdvisor) getAlternativeUserAgent() string {
	if len(ra.userAgents) > 0 {
		// Return a random user agent from the list
		index := time.Now().UnixNano() % int64(len(ra.userAgents))
		return ra.userAgents[index]
	}

	return "Mozilla/5.0 (compatible; GODL/1.0)"
}

// getCommonUserAgents returns a list of commonly used user agent strings.
func getCommonUserAgents() []string {
	return []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"curl/7.68.0",
		"wget/1.20.3",
	}
}

// getKnownMirrors returns a map of known mirror sites for common domains.
func getKnownMirrors() map[string][]string {
	return map[string][]string{
		"github.com": {
			"raw.githubusercontent.com",
			"github.githubassets.com",
		},
		"sourceforge.net": {
			"downloads.sourceforge.net",
			"netcologne.dl.sourceforge.net",
			"phoenixnap.dl.sourceforge.net",
		},
		"apache.org": {
			"archive.apache.org",
			"mirrors.apache.org",
			"www-us.apache.org",
		},
	}
}

// formatBytes formats byte counts in human-readable format.
func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}

	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(bytes)

	for _, unit := range units {
		value /= 1024
		if value < 1024 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}

	return fmt.Sprintf("%.1f PB", value/1024)
}

// GetHistoricalAnalysis returns analysis history for trend analysis.
func (ra *RecoveryAdvisor) GetHistoricalAnalysis() []FailureAnalysis {
	return ra.analysisHistory
}

// GetFailurePatterns analyzes historical data to identify recurring failure patterns.
func (ra *RecoveryAdvisor) GetFailurePatterns() map[string]int {
	patterns := make(map[string]int)

	for _, analysis := range ra.analysisHistory {
		key := fmt.Sprintf("%s_%s", analysis.FailureType.String(), analysis.ErrorCode.String())
		patterns[key]++
	}

	return patterns
}

// ClearHistory clears the analysis history.
func (ra *RecoveryAdvisor) ClearHistory() {
	ra.analysisHistory = []FailureAnalysis{}
}
