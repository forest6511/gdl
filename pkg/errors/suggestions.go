package errors

import (
	"errors"
	"fmt"
	"strings"
)

// SuggestionProvider provides user-friendly recovery suggestions for different error types.
type SuggestionProvider struct {
	registry *ErrorCodeRegistry
}

// NewSuggestionProvider creates a new suggestion provider.
func NewSuggestionProvider() *SuggestionProvider {
	return &SuggestionProvider{
		registry: NewErrorCodeRegistry(),
	}
}

// Suggestion represents a user-friendly suggestion with recovery steps.
type Suggestion struct {
	Summary    string   `json:"summary"`
	Steps      []string `json:"steps"`
	Tips       []string `json:"tips,omitempty"`
	DocLinks   []string `json:"doc_links,omitempty"`
	Examples   []string `json:"examples,omitempty"`
	References []string `json:"references,omitempty"`
}

// GetSuggestion returns a comprehensive suggestion for the given error code.
func (sp *SuggestionProvider) GetSuggestion(code ErrorCode) *Suggestion {
	// Map ErrorCode to appropriate suggestion based on the error type
	switch code {
	case CodeInvalidURL:
		return sp.getValidationSuggestion(ValidationInvalidURL)
	case CodeFileExists:
		return sp.getFileSystemSuggestion(FSFileExists)
	case CodeInsufficientSpace:
		return sp.getFileSystemSuggestion(FSInsufficientSpace)
	case CodeNetworkError:
		return sp.getNetworkSuggestion(NetworkConnectionRefused) // Generic network error
	case CodeTimeout:
		return sp.getNetworkSuggestion(NetworkConnectionTimeout)
	case CodePermissionDenied:
		return sp.getFileSystemSuggestion(FSPermissionDenied)
	case CodeFileNotFound:
		return sp.getFileSystemSuggestion(FSFileNotFound)
	case CodeAuthenticationFailed:
		return sp.getHTTPSuggestion(HTTPUnauthorized)
	case CodeServerError:
		return sp.getHTTPSuggestion(HTTPInternalServerError)
	case CodeClientError:
		return sp.getHTTPSuggestion(HTTPBadRequest)
	case CodeCorruptedData:
		return sp.getFileSystemSuggestion(FSCorruptedFile)
	default:
		return sp.getGenericSuggestion()
	}
}

// GetSuggestionForError returns a suggestion based on the error type and context.
func (sp *SuggestionProvider) GetSuggestionForError(err error) *Suggestion {
	downloadErr := &DownloadError{}
	if errors.As(err, &downloadErr) {
		return sp.GetSuggestion(downloadErr.Code)
	}

	return sp.getGenericSuggestion()
}

// Network error suggestions.
func (sp *SuggestionProvider) getNetworkSuggestion(code NetworkErrorCode) *Suggestion {
	switch code {
	case NetworkDNSNotFound:
		return &Suggestion{
			Summary: "The domain name could not be resolved",
			Steps: []string{
				"Verify the URL is spelled correctly",
				"Check if the domain exists by visiting it in a web browser",
				"Try using a different DNS server (e.g., 8.8.8.8, 1.1.1.1)",
				"Flush your DNS cache",
			},
			Tips: []string{
				"DNS issues are often temporary - try again in a few minutes",
				"Corporate networks may block certain domains",
			},
			Examples: []string{
				"sudo systemctl flush-dns  # Linux",
				"ipconfig /flushdns        # Windows",
				"sudo dscacheutil -flushcache  # macOS",
			},
		}

	case NetworkDNSTimeout:
		return &Suggestion{
			Summary: "DNS resolution is taking too long",
			Steps: []string{
				"Check your internet connection",
				"Try using a faster DNS server",
				"Increase the timeout setting with --timeout flag",
				"Try the download again - DNS issues are often temporary",
			},
			Tips: []string{
				"Public DNS servers are often faster than ISP defaults",
				"VPN connections may cause DNS timeouts",
			},
		}

	case NetworkConnectionRefused, NetworkConnectionTimeout:
		return &Suggestion{
			Summary: "Cannot establish connection to the server",
			Steps: []string{
				"Check if the server is online by visiting the URL in a browser",
				"Verify your internet connection is working",
				"Try again later - the server may be temporarily down",
				"Check if you're behind a firewall blocking the connection",
			},
			Tips: []string{
				"Server maintenance windows often cause temporary outages",
				"Corporate firewalls may block certain ports or protocols",
			},
		}

	case NetworkConnectionReset:
		return &Suggestion{
			Summary: "Connection was reset by the remote server",
			Steps: []string{
				"Retry the download - connection resets are often temporary",
				"Try downloading smaller chunks with --chunk-size",
				"Use single-threaded download with --no-concurrent",
				"Check if the server has rate limiting in place",
			},
			Tips: []string{
				"Some servers are sensitive to multiple concurrent connections",
				"Large file downloads may timeout on unstable connections",
			},
		}

	case NetworkTLSHandshakeFailure:
		return &Suggestion{
			Summary: "SSL/TLS handshake failed",
			Steps: []string{
				"Check if the server certificate is valid",
				"Verify the server supports modern TLS versions",
				"Try updating your system's certificate store",
				"Check if you're behind a corporate proxy intercepting HTTPS",
			},
			DocLinks: []string{
				"https://curl.se/docs/sslcerts.html",
			},
		}

	default:
		return &Suggestion{
			Summary: "Network connectivity issue",
			Steps: []string{
				"Check your internet connection",
				"Try the download again",
				"Use --verbose flag for more detailed error information",
			},
		}
	}
}

// HTTP error suggestions.
func (sp *SuggestionProvider) getHTTPSuggestion(code HTTPErrorCode) *Suggestion {
	switch code {
	case HTTPNotFound:
		return &Suggestion{
			Summary: "The requested file was not found on the server",
			Steps: []string{
				"Double-check the URL for typos",
				"Visit the website in a browser to verify the file exists",
				"Check if the file has moved to a different location",
				"Contact the website owner if you believe this is an error",
			},
			Tips: []string{
				"URLs are case-sensitive on many servers",
				"Files may have been moved or deleted",
			},
		}

	case HTTPUnauthorized:
		return &Suggestion{
			Summary: "Authentication is required to access this resource",
			Steps: []string{
				"Check if you need to log in to the website first",
				"Verify you have the correct credentials",
				"Look for authentication headers or API keys needed",
				"Contact the resource owner for access permissions",
			},
			Examples: []string{
				"godl --user-agent 'MyApp/1.0' https://example.com/file",
				"Use browser session cookies if authentication is session-based",
			},
		}

	case HTTPForbidden:
		return &Suggestion{
			Summary: "Access to this resource is forbidden",
			Steps: []string{
				"Check if you have permission to access this file",
				"Verify the file is publicly accessible",
				"Try accessing the file through a web browser",
				"Contact the site administrator if you believe you should have access",
			},
			Tips: []string{
				"Some sites block automated downloads",
				"User-Agent headers may affect access permissions",
			},
		}

	case HTTPTooManyRequests:
		return &Suggestion{
			Summary: "You're making too many requests to the server",
			Steps: []string{
				"Wait a few minutes before retrying",
				"Use --no-concurrent to reduce connection load",
				"Increase delays between requests",
				"Check the Retry-After header for guidance",
			},
			Tips: []string{
				"Rate limiting is common for large file downloads",
				"Some APIs have daily or hourly limits",
			},
		}

	case HTTPInternalServerError, HTTPBadGateway, HTTPServiceUnavailable:
		return &Suggestion{
			Summary: "The server is experiencing technical difficulties",
			Steps: []string{
				"Try again in a few minutes - server issues are often temporary",
				"Check the website's status page or social media for outage reports",
				"Try downloading at a different time when server load may be lower",
				"Contact the website administrator if the issue persists",
			},
			Tips: []string{
				"Server maintenance often happens during off-peak hours",
				"High-traffic sites may have temporary outages",
			},
		}

	case HTTPGatewayTimeout:
		return &Suggestion{
			Summary: "The server gateway timed out",
			Steps: []string{
				"Increase the timeout setting with --timeout flag",
				"Try again later when server load may be lower",
				"Use smaller chunk sizes with --chunk-size",
				"Consider downloading during off-peak hours",
			},
		}

	default:
		return &Suggestion{
			Summary: fmt.Sprintf("HTTP error occurred (%s)", code),
			Steps: []string{
				"Check the server response for more details",
				"Try the request again",
				"Use --verbose flag for more information",
			},
		}
	}
}

// File system error suggestions.
func (sp *SuggestionProvider) getFileSystemSuggestion(code FileSystemErrorCode) *Suggestion {
	switch code {
	case FSPermissionDenied, FSAccessDenied:
		return &Suggestion{
			Summary: "You don't have permission to write to this location",
			Steps: []string{
				"Choose a different output directory where you have write permissions",
				"Run the command with appropriate privileges (sudo on Unix systems)",
				"Check and modify file/directory permissions",
				"Use --create-dirs to create parent directories automatically",
			},
			Examples: []string{
				"godl -o ~/Downloads/file.zip https://example.com/file.zip",
				"sudo godl -o /opt/files/file.zip https://example.com/file.zip",
				"chmod 755 /path/to/directory",
			},
		}

	case FSInsufficientSpace, FSDeviceFull:
		return &Suggestion{
			Summary: "Not enough disk space to complete the download",
			Steps: []string{
				"Free up disk space by deleting unnecessary files",
				"Choose a different output location with more space",
				"Check available space with df -h (Unix) or dir (Windows)",
				"Consider downloading to an external drive",
			},
			Examples: []string{
				"df -h                    # Check disk usage (Unix)",
				"du -sh * | sort -hr     # Find large directories (Unix)",
				"godl -o /mnt/external/file.zip https://example.com/file.zip",
			},
		}

	case FSFileExists:
		return &Suggestion{
			Summary: "A file with this name already exists",
			Steps: []string{
				"Use --force flag to overwrite the existing file",
				"Choose a different output filename with -o flag",
				"Move or rename the existing file first",
				"Use --resume flag if you want to continue a partial download",
			},
			Examples: []string{
				"godl --force https://example.com/file.zip",
				"godl -o file_v2.zip https://example.com/file.zip",
				"mv file.zip file_backup.zip",
			},
		}

	case FSDirectoryNotFound:
		return &Suggestion{
			Summary: "The target directory doesn't exist",
			Steps: []string{
				"Create the directory manually before downloading",
				"Use --create-dirs flag to create directories automatically",
				"Check the path spelling and permissions",
			},
			Examples: []string{
				"mkdir -p /path/to/directory",
				"godl --create-dirs -o /new/path/file.zip https://example.com/file.zip",
			},
		}

	case FSReadOnlyFilesystem:
		return &Suggestion{
			Summary: "The target filesystem is read-only",
			Steps: []string{
				"Choose a different output location",
				"Remount the filesystem with write permissions if appropriate",
				"Check if the device is write-protected",
			},
		}

	case FSCorruptedFile, FSChecksumMismatch:
		return &Suggestion{
			Summary: "The downloaded file appears to be corrupted",
			Steps: []string{
				"Delete the corrupted file and download again",
				"Try downloading from a different server or mirror",
				"Check your internet connection stability",
				"Use --no-concurrent for more reliable single-threaded download",
			},
			Tips: []string{
				"Network interruptions can cause file corruption",
				"Some servers may serve corrupted files under load",
			},
		}

	default:
		return &Suggestion{
			Summary: "File system error occurred",
			Steps: []string{
				"Check file and directory permissions",
				"Verify available disk space",
				"Try a different output location",
			},
		}
	}
}

// Validation error suggestions.
func (sp *SuggestionProvider) getValidationSuggestion(code ValidationErrorCode) *Suggestion {
	switch code {
	case ValidationInvalidURL, ValidationMalformedURL:
		return &Suggestion{
			Summary: "The provided URL is not valid",
			Steps: []string{
				"Check the URL for typos and correct formatting",
				"Ensure the URL starts with http:// or https://",
				"Verify special characters are properly encoded",
				"Test the URL in a web browser first",
			},
			Examples: []string{
				"✓ https://example.com/file.zip",
				"✓ http://downloads.example.com/path/file.tar.gz",
				"✗ example.com/file.zip (missing protocol)",
				"✗ https://example.com/file with spaces.zip (needs encoding)",
			},
		}

	case ValidationUnsupportedScheme:
		return &Suggestion{
			Summary: "This URL scheme is not supported",
			Steps: []string{
				"Use HTTP or HTTPS URLs only",
				"Convert FTP URLs to HTTP if the server supports it",
				"Use appropriate tools for other protocols (ftp, sftp, etc.)",
			},
			Tips: []string{
				"Only HTTP and HTTPS protocols are supported",
				"Many FTP servers also provide HTTP access",
			},
		}

	case ValidationEmptyURL:
		return &Suggestion{
			Summary: "No URL was provided",
			Steps: []string{
				"Provide a URL as the last argument",
				"Check your command syntax",
			},
			Examples: []string{
				"godl https://example.com/file.zip",
				"godl -o myfile.zip https://example.com/download",
			},
		}

	case ValidationURLTooLong:
		return &Suggestion{
			Summary: "The URL is too long",
			Steps: []string{
				"Use URL shorteners if appropriate",
				"Check if the URL can be simplified",
				"Contact the server administrator about the long URL",
			},
		}

	default:
		return &Suggestion{
			Summary: "Input validation failed",
			Steps: []string{
				"Check your input parameters",
				"Refer to the help documentation",
				"Use --help flag for usage information",
			},
		}
	}
}

// Generic fallback suggestion.
func (sp *SuggestionProvider) getGenericSuggestion() *Suggestion {
	return &Suggestion{
		Summary: "An error occurred during the download process",
		Steps: []string{
			"Try the download again - many issues are temporary",
			"Use --verbose flag for more detailed error information",
			"Check your internet connection",
			"Verify the URL is correct and accessible",
		},
		Tips: []string{
			"Network issues are often resolved by retrying",
			"Use --help to see all available options",
		},
		DocLinks: []string{
			"https://github.com/forest6511/godl/wiki/Troubleshooting",
		},
	}
}

// FormatSuggestion formats a suggestion for display to the user.
func (sp *SuggestionProvider) FormatSuggestion(suggestion *Suggestion) string {
	if suggestion == nil {
		return ""
	}

	var sb strings.Builder

	// Summary
	sb.WriteString(fmt.Sprintf("💡 %s\n\n", suggestion.Summary))

	// Recovery steps
	if len(suggestion.Steps) > 0 {
		sb.WriteString("🔧 What to try:\n")

		for i, step := range suggestion.Steps {
			sb.WriteString(fmt.Sprintf("   %d. %s\n", i+1, step))
		}

		sb.WriteString("\n")
	}

	// Tips
	if len(suggestion.Tips) > 0 {
		sb.WriteString("💭 Helpful tips:\n")

		for _, tip := range suggestion.Tips {
			sb.WriteString(fmt.Sprintf("   • %s\n", tip))
		}

		sb.WriteString("\n")
	}

	// Examples
	if len(suggestion.Examples) > 0 {
		sb.WriteString("📋 Examples:\n")

		for _, example := range suggestion.Examples {
			sb.WriteString(fmt.Sprintf("   %s\n", example))
		}

		sb.WriteString("\n")
	}

	// Documentation links
	if len(suggestion.DocLinks) > 0 {
		sb.WriteString("📚 More information:\n")

		for _, link := range suggestion.DocLinks {
			sb.WriteString(fmt.Sprintf("   %s\n", link))
		}

		sb.WriteString("\n")
	}

	// References
	if len(suggestion.References) > 0 {
		sb.WriteString("🔗 Related:\n")

		for _, ref := range suggestion.References {
			sb.WriteString(fmt.Sprintf("   %s\n", ref))
		}
	}

	return sb.String()
}

// GetTroubleshootingSteps returns general troubleshooting steps for download issues.
func (sp *SuggestionProvider) GetTroubleshootingSteps() []string {
	return []string{
		"Verify the URL is correct and accessible in a web browser",
		"Check your internet connection stability",
		"Try using --verbose flag for detailed error information",
		"Use --timeout flag to increase timeout for slow connections",
		"Try --no-concurrent for single-threaded download",
		"Check available disk space in the target directory",
		"Verify you have write permissions to the output location",
		"Try downloading to a different directory",
		"Consider the server may be experiencing high load",
		"Check if your firewall or proxy is blocking the connection",
	}
}

// GetQuickFixes returns common quick fixes for download problems.
func (sp *SuggestionProvider) GetQuickFixes() map[string]string {
	return map[string]string{
		"File exists":        "Use --force to overwrite or -o to specify different name",
		"Permission denied":  "Try different output directory or use appropriate privileges",
		"Network timeout":    "Use --timeout flag to increase timeout duration",
		"Connection refused": "Check URL and internet connection, try again later",
		"Invalid URL":        "Verify URL format starts with http:// or https://",
		"Insufficient space": "Free up disk space or choose different output location",
		"Server error":       "Try again later, server may be temporarily unavailable",
		"DNS resolution":     "Check domain name spelling, try different DNS server",
		"SSL/TLS error":      "Update system certificates or check corporate proxy settings",
		"Rate limited":       "Use --no-concurrent and wait before retrying",
	}
}
