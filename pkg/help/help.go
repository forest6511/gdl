// Package help provides context-sensitive help and guidance for the gdl CLI tool.
package help

import (
	"fmt"
	"strings"

	"github.com/forest6511/gdl/pkg/errors"
)

// HelpProvider provides context-sensitive help and error guidance.
type HelpProvider struct {
	commandExamples map[string][]Example
	errorGuides     map[errors.ErrorCode]ErrorGuide
	troubleshooting map[string]TroubleshootingGuide
}

// Example represents a command example with description.
type Example struct {
	Command     string
	Description string
	Tags        []string
}

// ErrorGuide provides guidance for specific error types.
type ErrorGuide struct {
	ErrorCode    errors.ErrorCode
	Description  string
	CommonCauses []string
	Solutions    []Solution
	Examples     []Example
	SeeAlso      []string
}

// Solution represents a solution to an error.
type Solution struct {
	Title       string
	Description string
	Commands    []string
	Priority    int // Lower number = higher priority
}

// TroubleshootingGuide provides comprehensive troubleshooting for scenarios.
type TroubleshootingGuide struct {
	Scenario    string
	Description string
	Steps       []TroubleshootingStep
	Examples    []Example
}

// TroubleshootingStep represents a step in troubleshooting.
type TroubleshootingStep struct {
	Step        int
	Title       string
	Description string
	Commands    []string
	Expected    string
}

// NewHelpProvider creates a new help provider with comprehensive guidance.
func NewHelpProvider() *HelpProvider {
	hp := &HelpProvider{
		commandExamples: make(map[string][]Example),
		errorGuides:     make(map[errors.ErrorCode]ErrorGuide),
		troubleshooting: make(map[string]TroubleshootingGuide),
	}

	hp.initializeCommandExamples()
	hp.initializeErrorGuides()
	hp.initializeTroubleshootingGuides()

	return hp
}

// GetCommandHelp returns help for a specific command.
func (hp *HelpProvider) GetCommandHelp(command string) string {
	var help strings.Builder

	help.WriteString(fmt.Sprintf("# Help for '%s'\n\n", command))

	examples, exists := hp.commandExamples[command]
	if !exists {
		help.WriteString("No specific help available for this command.\n")
		help.WriteString("Run 'gdl --help' for general usage information.\n")

		return help.String()
	}

	help.WriteString("## Examples\n\n")

	for _, example := range examples {
		help.WriteString(fmt.Sprintf("### %s\n", example.Description))
		help.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", example.Command))

		if len(example.Tags) > 0 {
			help.WriteString(fmt.Sprintf("*Tags: %s*\n\n", strings.Join(example.Tags, ", ")))
		}
	}

	return help.String()
}

// GetErrorHelp returns comprehensive help for a specific error.
func (hp *HelpProvider) GetErrorHelp(errorCode errors.ErrorCode) string {
	var help strings.Builder

	guide, exists := hp.errorGuides[errorCode]
	if !exists {
		help.WriteString(fmt.Sprintf("# Error Help: %s\n\n", errorCode.String()))
		help.WriteString("No specific guidance available for this error.\n")
		help.WriteString("See the general troubleshooting guide for help.\n")

		return help.String()
	}

	help.WriteString(fmt.Sprintf("# Error Help: %s\n\n", guide.ErrorCode.String()))
	help.WriteString(fmt.Sprintf("## Description\n%s\n\n", guide.Description))

	if len(guide.CommonCauses) > 0 {
		help.WriteString("## Common Causes\n")

		for _, cause := range guide.CommonCauses {
			help.WriteString(fmt.Sprintf("- %s\n", cause))
		}

		help.WriteString("\n")
	}

	if len(guide.Solutions) > 0 {
		help.WriteString("## Solutions\n\n")

		for _, solution := range guide.Solutions {
			help.WriteString(fmt.Sprintf("### %s\n", solution.Title))
			help.WriteString(fmt.Sprintf("%s\n\n", solution.Description))

			if len(solution.Commands) > 0 {
				for _, cmd := range solution.Commands {
					help.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", cmd))
				}
			}
		}
	}

	if len(guide.Examples) > 0 {
		help.WriteString("## Examples\n\n")

		for _, example := range guide.Examples {
			help.WriteString(fmt.Sprintf("### %s\n", example.Description))
			help.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", example.Command))
		}
	}

	if len(guide.SeeAlso) > 0 {
		help.WriteString("## See Also\n")

		for _, ref := range guide.SeeAlso {
			help.WriteString(fmt.Sprintf("- %s\n", ref))
		}

		help.WriteString("\n")
	}

	return help.String()
}

// GetTroubleshootingHelp returns troubleshooting guidance for a scenario.
func (hp *HelpProvider) GetTroubleshootingHelp(scenario string) string {
	var help strings.Builder

	guide, exists := hp.troubleshooting[scenario]
	if !exists {
		help.WriteString(fmt.Sprintf("# Troubleshooting: %s\n\n", scenario))
		help.WriteString("No specific troubleshooting guide available.\n")

		return help.String()
	}

	help.WriteString(fmt.Sprintf("# Troubleshooting: %s\n\n", guide.Scenario))
	help.WriteString(fmt.Sprintf("## Description\n%s\n\n", guide.Description))

	if len(guide.Steps) > 0 {
		help.WriteString("## Steps\n\n")

		for _, step := range guide.Steps {
			help.WriteString(fmt.Sprintf("### Step %d: %s\n", step.Step, step.Title))
			help.WriteString(fmt.Sprintf("%s\n\n", step.Description))

			if len(step.Commands) > 0 {
				for _, cmd := range step.Commands {
					help.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", cmd))
				}
			}

			if step.Expected != "" {
				help.WriteString(fmt.Sprintf("**Expected result:** %s\n\n", step.Expected))
			}
		}
	}

	return help.String()
}

// GetQuickHelp returns quick help for common scenarios.
func (hp *HelpProvider) GetQuickHelp() string {
	return `# GDL Quick Help

## Common Commands

### Basic Download
` + "```bash" + `
gdl download "https://example.com/file.zip"
` + "```" + `

### Download with Options
` + "```bash" + `
gdl download "https://example.com/file.zip" \
  --output "/downloads/myfile.zip" \
  --retry 5 \
  --timeout 300s
` + "```" + `

### Resume Download
` + "```bash" + `
gdl download "https://example.com/file.zip" --resume
` + "```" + `

### Batch Download
` + "```bash" + `
gdl download -f urls.txt --concurrent 4
` + "```" + `

## Quick Fixes

### Connection Issues
` + "```bash" + `
# Test connectivity
gdl --check-network

# Use different DNS
gdl download "https://example.com/file.zip" --dns "8.8.8.8"

# Increase timeout
gdl download "https://example.com/file.zip" --timeout 600s
` + "```" + `

### File Issues
` + "```bash" + `
# Overwrite existing file
gdl download "https://example.com/file.zip" --overwrite

# Download to different location
gdl download "https://example.com/file.zip" -o "/tmp/file.zip"

# Check available space
df -h /downloads
` + "```" + `

### Debug Issues
` + "```bash" + `
# Enable verbose output
gdl download "https://example.com/file.zip" --verbose

# Save logs
gdl download "https://example.com/file.zip" --log-file debug.log

# Test URL only
gdl head "https://example.com/file.zip"
` + "```" + `

## Get More Help
- ` + "`gdl help <command>`" + ` - Command-specific help
- ` + "`gdl troubleshoot <issue>`" + ` - Troubleshooting guide
- ` + "`gdl config --help`" + ` - Configuration help
- ` + "`gdl examples`" + ` - More examples

For detailed documentation, visit: https://github.com/forest6511/gdl/docs`
}

// initializeCommandExamples sets up command examples.
func (hp *HelpProvider) initializeCommandExamples() {
	// Download command examples
	hp.commandExamples["download"] = []Example{
		{
			Command:     "gdl download \"https://example.com/file.zip\"",
			Description: "Basic download to current directory",
			Tags:        []string{"basic", "simple"},
		},
		{
			Command:     "gdl download \"https://example.com/file.zip\" -o \"/downloads/myfile.zip\"",
			Description: "Download to specific location",
			Tags:        []string{"output", "path"},
		},
		{
			Command:     "gdl download \"https://example.com/file.zip\" --resume",
			Description: "Resume interrupted download",
			Tags:        []string{"resume", "recovery"},
		},
		{
			Command:     "gdl download \"https://example.com/file.zip\" --retry 10 --timeout 600s",
			Description: "Download with custom retry and timeout settings",
			Tags:        []string{"retry", "timeout", "reliability"},
		},
		{
			Command:     "gdl download \"https://example.com/file.zip\" --concurrent 8",
			Description: "Download with multiple concurrent connections",
			Tags:        []string{"concurrent", "performance"},
		},
		{
			Command:     "gdl download \"https://example.com/file.zip\" --header \"Authorization: Bearer token123\"",
			Description: "Download with authentication header",
			Tags:        []string{"auth", "headers", "security"},
		},
		{
			Command:     "gdl download -f urls.txt --concurrent 4 --output-dir \"/downloads\"",
			Description: "Batch download from URL list",
			Tags:        []string{"batch", "multiple", "file"},
		},
		{
			Command:     "gdl download \"https://example.com/file.zip\" --verbose --log-file download.log",
			Description: "Download with detailed logging",
			Tags:        []string{"debug", "logging", "verbose"},
		},
		{
			Command:     "gdl download \"https://example.com/file.zip\" --chunk-size 16384 --buffer-size 8192",
			Description: "Download with optimized buffer settings for slow connections",
			Tags:        []string{"performance", "optimization", "slow-network"},
		},
		{
			Command:     "gdl download \"https://example.com/file.zip\" --user-agent \"Mozilla/5.0\" --max-redirects 5",
			Description: "Download with custom user agent and redirect handling",
			Tags:        []string{"user-agent", "redirects", "compatibility"},
		},
	}

	// Config command examples
	hp.commandExamples["config"] = []Example{
		{
			Command:     "gdl config show",
			Description: "Display current configuration",
			Tags:        []string{"view", "current"},
		},
		{
			Command:     "gdl config set retry_policy.max_retries 10",
			Description: "Set maximum retry attempts",
			Tags:        []string{"retry", "set"},
		},
		{
			Command:     "gdl config set timeouts.download_timeout \"1h\"",
			Description: "Set download timeout to 1 hour",
			Tags:        []string{"timeout", "duration"},
		},
		{
			Command:     "gdl config reset",
			Description: "Reset configuration to defaults",
			Tags:        []string{"reset", "defaults"},
		},
		{
			Command:     "gdl config validate",
			Description: "Validate current configuration",
			Tags:        []string{"validate", "check"},
		},
	}

	// Head command examples
	hp.commandExamples["head"] = []Example{
		{
			Command:     "gdl head \"https://example.com/file.zip\"",
			Description: "Get file information without downloading",
			Tags:        []string{"info", "test"},
		},
		{
			Command:     "gdl head \"https://example.com/file.zip\" --format json",
			Description: "Get file information in JSON format",
			Tags:        []string{"json", "format"},
		},
	}
}

// initializeErrorGuides sets up error-specific guidance.
func (hp *HelpProvider) initializeErrorGuides() {
	// Network Error Guide
	hp.errorGuides[errors.CodeNetworkError] = ErrorGuide{
		ErrorCode:   errors.CodeNetworkError,
		Description: "Network connection issues prevent the download from completing. This includes DNS resolution failures, connection refused errors, and general connectivity problems.",
		CommonCauses: []string{
			"Internet connection is down or unstable",
			"DNS resolution failures",
			"Firewall blocking the connection",
			"Proxy configuration issues",
			"Server is temporarily unreachable",
			"Network routing problems",
		},
		Solutions: []Solution{
			{
				Title:       "Check Internet Connection",
				Description: "Verify that your internet connection is working properly.",
				Commands: []string{
					"ping google.com",
					"curl -I https://httpbin.org/get",
					"gdl --check-network",
				},
				Priority: 1,
			},
			{
				Title:       "Test Alternative DNS Servers",
				Description: "Try using different DNS servers if DNS resolution is failing.",
				Commands: []string{
					"gdl config set network.dns_servers \"8.8.8.8,8.8.4.4\"",
					"nslookup example.com 8.8.8.8",
				},
				Priority: 2,
			},
			{
				Title:       "Configure Proxy Settings",
				Description: "If behind a corporate firewall, configure proxy settings.",
				Commands: []string{
					"export HTTP_PROXY=http://proxy.example.com:8080",
					"export HTTPS_PROXY=http://proxy.example.com:8080",
					"gdl config set network.proxy \"http://proxy.example.com:8080\"",
				},
				Priority: 3,
			},
			{
				Title:       "Increase Timeout Values",
				Description: "For slow networks, increase timeout values to allow more time for connection.",
				Commands: []string{
					"gdl download \"https://example.com/file.zip\" --timeout 300s",
					"gdl config set timeouts.connect_timeout \"30s\"",
					"gdl config set timeouts.read_timeout \"120s\"",
				},
				Priority: 4,
			},
		},
		Examples: []Example{
			{
				Command:     "gdl download \"https://example.com/file.zip\" --retry 5 --timeout 300s",
				Description: "Download with increased retry and timeout for unreliable networks",
			},
			{
				Command:     "gdl download \"https://example.com/file.zip\" --dns \"8.8.8.8\" --verbose",
				Description: "Download using alternative DNS with verbose output for debugging",
			},
		},
		SeeAlso: []string{
			"Timeout errors",
			"Configuration guide",
			"Network troubleshooting",
		},
	}

	// Timeout Error Guide
	hp.errorGuides[errors.CodeTimeout] = ErrorGuide{
		ErrorCode:   errors.CodeTimeout,
		Description: "The download operation exceeded the configured timeout limits. This can occur during connection establishment, data transfer, or overall operation time.",
		CommonCauses: []string{
			"Slow network connection",
			"Large file size",
			"Server overload or slow response",
			"Restrictive timeout configuration",
			"Network congestion",
		},
		Solutions: []Solution{
			{
				Title:       "Increase Timeout Values",
				Description: "Configure longer timeouts for slow connections or large files.",
				Commands: []string{
					"gdl download \"https://example.com/file.zip\" --timeout 1800s",
					"gdl config set timeouts.download_timeout \"30m\"",
					"gdl config set timeouts.read_timeout \"300s\"",
				},
				Priority: 1,
			},
			{
				Title:       "Use Resume Support",
				Description: "Enable resume to continue interrupted downloads.",
				Commands: []string{
					"gdl download \"https://example.com/file.zip\" --resume",
					"gdl config set storage.resume_support true",
				},
				Priority: 2,
			},
			{
				Title:       "Optimize Download Settings",
				Description: "Use smaller chunk sizes for slow connections.",
				Commands: []string{
					"gdl download \"https://example.com/file.zip\" --chunk-size 8192",
					"gdl config set network.chunk_size 8192",
				},
				Priority: 3,
			},
		},
		Examples: []Example{
			{
				Command:     "gdl download \"https://example.com/largefile.zip\" --timeout 3600s --resume",
				Description: "Download large file with 1-hour timeout and resume support",
			},
		},
		SeeAlso: []string{
			"Network errors",
			"Resume support",
			"Performance optimization",
		},
	}

	// File Exists Error Guide
	hp.errorGuides[errors.CodeFileExists] = ErrorGuide{
		ErrorCode:   errors.CodeFileExists,
		Description: "The destination file already exists and overwrite is not enabled.",
		CommonCauses: []string{
			"Previous download to same location",
			"File already exists in destination directory",
			"Overwrite protection is enabled",
		},
		Solutions: []Solution{
			{
				Title:       "Enable Overwrite",
				Description: "Allow overwriting existing files.",
				Commands: []string{
					"gdl download \"https://example.com/file.zip\" --overwrite",
					"gdl config set storage.overwrite_existing true",
				},
				Priority: 1,
			},
			{
				Title:       "Use Different Filename",
				Description: "Download to a different location or filename.",
				Commands: []string{
					"gdl download \"https://example.com/file.zip\" -o \"file_new.zip\"",
					"gdl download \"https://example.com/file.zip\" -o \"/tmp/file.zip\"",
				},
				Priority: 2,
			},
			{
				Title:       "Resume Existing Download",
				Description: "If the file is incomplete, resume the download.",
				Commands: []string{
					"gdl download \"https://example.com/file.zip\" --resume",
				},
				Priority: 3,
			},
		},
		Examples: []Example{
			{
				Command:     "gdl download \"https://example.com/file.zip\" --overwrite --backup",
				Description: "Overwrite existing file while creating a backup",
			},
		},
		SeeAlso: []string{
			"Resume support",
			"File management",
		},
	}

	// Insufficient Space Error Guide
	hp.errorGuides[errors.CodeInsufficientSpace] = ErrorGuide{
		ErrorCode:   errors.CodeInsufficientSpace,
		Description: "There is not enough disk space available for the download.",
		CommonCauses: []string{
			"Disk is full or nearly full",
			"Large file exceeds available space",
			"Temporary files consuming space",
			"Insufficient space buffer configured",
		},
		Solutions: []Solution{
			{
				Title:       "Free Up Disk Space",
				Description: "Remove unnecessary files to make space.",
				Commands: []string{
					"df -h",
					"du -sh /downloads/*",
					"rm -rf /tmp/gdl_*",
					"gdl --cleanup-temp",
				},
				Priority: 1,
			},
			{
				Title:       "Change Download Location",
				Description: "Download to a different disk or directory with more space.",
				Commands: []string{
					"gdl download \"https://example.com/file.zip\" -o \"/mnt/storage/file.zip\"",
					"gdl config set storage.default_download_dir \"/mnt/storage\"",
				},
				Priority: 2,
			},
			{
				Title:       "Adjust Space Requirements",
				Description: "Configure minimum free space requirements.",
				Commands: []string{
					"gdl config set storage.min_free_space 52428800", // 50MB
				},
				Priority: 3,
			},
		},
		Examples: []Example{
			{
				Command:     "df -h && gdl download \"https://example.com/file.zip\" -o \"/tmp/file.zip\"",
				Description: "Check disk space and download to temporary directory",
			},
		},
		SeeAlso: []string{
			"Storage management",
			"Configuration guide",
		},
	}

	// Authentication Failed Error Guide
	hp.errorGuides[errors.CodeAuthenticationFailed] = ErrorGuide{
		ErrorCode:   errors.CodeAuthenticationFailed,
		Description: "Authentication or authorization failed when accessing the resource.",
		CommonCauses: []string{
			"Invalid credentials",
			"Expired authentication tokens",
			"Missing authorization headers",
			"Insufficient permissions",
			"Server-side authentication issues",
		},
		Solutions: []Solution{
			{
				Title:       "Add Authentication Headers",
				Description: "Include proper authentication in the request.",
				Commands: []string{
					"gdl download \"https://example.com/file.zip\" --header \"Authorization: Bearer your_token\"",
					"gdl download \"https://example.com/file.zip\" --header \"X-API-Key: your_key\"",
				},
				Priority: 1,
			},
			{
				Title:       "Use Basic Authentication",
				Description: "Include username and password in the URL.",
				Commands: []string{
					"gdl download \"https://username:password@example.com/file.zip\"",
				},
				Priority: 2,
			},
			{
				Title:       "Verify Credentials",
				Description: "Test authentication with curl or browser.",
				Commands: []string{
					"curl -I -H \"Authorization: Bearer your_token\" \"https://example.com/file.zip\"",
				},
				Priority: 3,
			},
		},
		Examples: []Example{
			{
				Command:     "gdl download \"https://api.example.com/file.zip\" --header \"Authorization: Bearer $(cat token.txt)\"",
				Description: "Download using bearer token from file",
			},
		},
		SeeAlso: []string{
			"Security configuration",
			"Headers and authentication",
		},
	}
}

// initializeTroubleshootingGuides sets up scenario-based troubleshooting.
func (hp *HelpProvider) initializeTroubleshootingGuides() {
	// Slow Downloads Troubleshooting
	hp.troubleshooting["slow-downloads"] = TroubleshootingGuide{
		Scenario:    "Slow Downloads",
		Description: "Downloads are taking longer than expected or showing poor performance.",
		Steps: []TroubleshootingStep{
			{
				Step:        1,
				Title:       "Check Network Speed",
				Description: "Test your internet connection speed to establish a baseline.",
				Commands: []string{
					"curl -o /dev/null -s -w \"%{speed_download}\\n\" https://httpbin.org/bytes/10485760",
					"gdl --speed-test",
				},
				Expected: "Speed should be reasonable for your connection type",
			},
			{
				Step:        2,
				Title:       "Optimize Buffer Settings",
				Description: "Adjust chunk and buffer sizes based on your connection speed.",
				Commands: []string{
					"# For fast connections (>50 Mbps)",
					"gdl config set network.chunk_size 131072",
					"gdl config set network.buffer_size 32768",
					"# For slow connections (<10 Mbps)",
					"gdl config set network.chunk_size 16384",
					"gdl config set network.buffer_size 8192",
				},
				Expected: "Improved download speeds after configuration change",
			},
			{
				Step:        3,
				Title:       "Enable Concurrent Connections",
				Description: "Use multiple connections for faster downloads (if server supports it).",
				Commands: []string{
					"gdl download \"https://example.com/file.zip\" --concurrent 4",
					"gdl config set network.max_concurrent_downloads 6",
				},
				Expected: "Faster downloads with multiple connections",
			},
			{
				Step:        4,
				Title:       "Check Server Response Time",
				Description: "Verify if the server is responding slowly.",
				Commands: []string{
					"curl -w \"@curl-format.txt\" -o /dev/null -s \"https://example.com/file.zip\"",
					"gdl head \"https://example.com/file.zip\" --verbose",
				},
				Expected: "Identify if delay is server-side or client-side",
			},
		},
		Examples: []Example{
			{
				Command:     "gdl download \"https://example.com/file.zip\" --concurrent 8 --chunk-size 65536",
				Description: "High-performance download configuration for fast connections",
			},
		},
	}

	// Connection Issues Troubleshooting
	hp.troubleshooting["connection-issues"] = TroubleshootingGuide{
		Scenario:    "Connection Issues",
		Description: "Unable to establish connections or experiencing frequent connection failures.",
		Steps: []TroubleshootingStep{
			{
				Step:        1,
				Title:       "Test Basic Connectivity",
				Description: "Verify internet connection and DNS resolution.",
				Commands: []string{
					"ping -c 4 google.com",
					"nslookup example.com",
					"gdl --check-network",
				},
				Expected: "Successful pings and DNS resolution",
			},
			{
				Step:        2,
				Title:       "Test Target Server",
				Description: "Check if the target server is accessible.",
				Commands: []string{
					"curl -I \"https://example.com/file.zip\"",
					"telnet example.com 443",
					"gdl head \"https://example.com/file.zip\"",
				},
				Expected: "Server responds with valid HTTP headers",
			},
			{
				Step:        3,
				Title:       "Check Proxy/Firewall",
				Description: "Configure proxy settings if behind corporate firewall.",
				Commands: []string{
					"env | grep -i proxy",
					"gdl config set network.proxy \"http://proxy.company.com:8080\"",
					"gdl download \"https://example.com/file.zip\" --proxy \"http://proxy:8080\"",
				},
				Expected: "Connection successful through proxy",
			},
			{
				Step:        4,
				Title:       "Try Alternative Methods",
				Description: "Use different approaches if standard methods fail.",
				Commands: []string{
					"# Try IPv4 only",
					"gdl download \"https://example.com/file.zip\" --ipv4-only",
					"# Disable TLS verification for testing",
					"gdl download \"https://example.com/file.zip\" --insecure",
					"# Use different User-Agent",
					"gdl download \"https://example.com/file.zip\" --user-agent \"Mozilla/5.0\"",
				},
				Expected: "Identify working configuration",
			},
		},
		Examples: []Example{
			{
				Command:     "gdl download \"https://example.com/file.zip\" --proxy \"http://proxy:8080\" --retry 5",
				Description: "Download through proxy with retries",
			},
		},
	}

	// Large Files Troubleshooting
	hp.troubleshooting["large-files"] = TroubleshootingGuide{
		Scenario:    "Large File Downloads",
		Description: "Issues downloading very large files (>1GB) including timeouts and interruptions.",
		Steps: []TroubleshootingStep{
			{
				Step:        1,
				Title:       "Configure Appropriate Timeouts",
				Description: "Set longer timeouts for large file downloads.",
				Commands: []string{
					"gdl config set timeouts.download_timeout \"2h\"",
					"gdl config set timeouts.read_timeout \"300s\"",
					"gdl download \"https://example.com/largefile.zip\" --timeout 7200s",
				},
				Expected: "No timeout errors during download",
			},
			{
				Step:        2,
				Title:       "Enable Resume Support",
				Description: "Configure resume support to handle interruptions.",
				Commands: []string{
					"gdl config set storage.resume_support true",
					"gdl download \"https://example.com/largefile.zip\" --resume",
				},
				Expected: "Ability to resume interrupted downloads",
			},
			{
				Step:        3,
				Title:       "Optimize Memory Usage",
				Description: "Configure buffer sizes to avoid memory issues.",
				Commands: []string{
					"gdl config set network.chunk_size 32768",
					"gdl config set network.buffer_size 16384",
				},
				Expected: "Stable memory usage during download",
			},
			{
				Step:        4,
				Title:       "Monitor Progress",
				Description: "Enable progress tracking and logging.",
				Commands: []string{
					"gdl download \"https://example.com/largefile.zip\" --progress --verbose --log-file large-download.log",
				},
				Expected: "Detailed progress information and logging",
			},
		},
		Examples: []Example{
			{
				Command:     "gdl download \"https://example.com/10gb-file.zip\" --resume --timeout 14400s --chunk-size 65536",
				Description: "Download 10GB file with optimal settings",
			},
		},
	}
}

// FormatHelp formats help text for display.
func FormatHelp(content string) string {
	// Basic formatting - in a real implementation, you might want to use
	// a more sophisticated formatting library
	return content
}

// GetContextualHelp provides context-sensitive help based on the current situation.
func (hp *HelpProvider) GetContextualHelp(context string, args ...interface{}) string {
	switch context {
	case "error":
		if len(args) > 0 {
			if errorCode, ok := args[0].(errors.ErrorCode); ok {
				return hp.GetErrorHelp(errorCode)
			}
		}
	case "command":
		if len(args) > 0 {
			if command, ok := args[0].(string); ok {
				return hp.GetCommandHelp(command)
			}
		}
	case "troubleshooting":
		if len(args) > 0 {
			if scenario, ok := args[0].(string); ok {
				return hp.GetTroubleshootingHelp(scenario)
			}
		}
	case "quick":
		return hp.GetQuickHelp()
	}

	return hp.GetQuickHelp()
}
