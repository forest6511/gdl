package errors

import (
	"fmt"
	"net/http"
)

// NetworkErrorCode represents network-related error codes.
type NetworkErrorCode string

const (
	// DNS-related errors.
	NetworkDNSNotFound NetworkErrorCode = "DNS_NOT_FOUND"
	NetworkDNSTimeout  NetworkErrorCode = "DNS_TIMEOUT"
	NetworkDNSFailure  NetworkErrorCode = "DNS_FAILURE"

	// Connection errors.
	NetworkConnectionRefused  NetworkErrorCode = "CONNECTION_REFUSED"
	NetworkConnectionTimeout  NetworkErrorCode = "CONNECTION_TIMEOUT"
	NetworkConnectionReset    NetworkErrorCode = "CONNECTION_RESET"
	NetworkConnectionAborted  NetworkErrorCode = "CONNECTION_ABORTED"
	NetworkHostUnreachable    NetworkErrorCode = "HOST_UNREACHABLE"
	NetworkNetworkUnreachable NetworkErrorCode = "NETWORK_UNREACHABLE"

	// Protocol errors.
	NetworkTLSHandshakeFailure NetworkErrorCode = "TLS_HANDSHAKE_FAILURE"
	NetworkProtocolError       NetworkErrorCode = "PROTOCOL_ERROR"
	NetworkProxyError          NetworkErrorCode = "PROXY_ERROR"

	// Timeout errors.
	NetworkReadTimeout    NetworkErrorCode = "READ_TIMEOUT"
	NetworkWriteTimeout   NetworkErrorCode = "WRITE_TIMEOUT"
	NetworkRequestTimeout NetworkErrorCode = "REQUEST_TIMEOUT"
)

// HTTPErrorCode represents HTTP-related error codes.
type HTTPErrorCode string

const (
	// 4xx Client Errors.
	HTTPBadRequest           HTTPErrorCode = "HTTP_400_BAD_REQUEST"
	HTTPUnauthorized         HTTPErrorCode = "HTTP_401_UNAUTHORIZED"
	HTTPPaymentRequired      HTTPErrorCode = "HTTP_402_PAYMENT_REQUIRED"
	HTTPForbidden            HTTPErrorCode = "HTTP_403_FORBIDDEN"
	HTTPNotFound             HTTPErrorCode = "HTTP_404_NOT_FOUND"
	HTTPMethodNotAllowed     HTTPErrorCode = "HTTP_405_METHOD_NOT_ALLOWED"
	HTTPNotAcceptable        HTTPErrorCode = "HTTP_406_NOT_ACCEPTABLE"
	HTTPProxyAuthRequired    HTTPErrorCode = "HTTP_407_PROXY_AUTH_REQUIRED"
	HTTPRequestTimeout       HTTPErrorCode = "HTTP_408_REQUEST_TIMEOUT"
	HTTPConflict             HTTPErrorCode = "HTTP_409_CONFLICT"
	HTTPGone                 HTTPErrorCode = "HTTP_410_GONE"
	HTTPLengthRequired       HTTPErrorCode = "HTTP_411_LENGTH_REQUIRED"
	HTTPPreconditionFailed   HTTPErrorCode = "HTTP_412_PRECONDITION_FAILED"
	HTTPPayloadTooLarge      HTTPErrorCode = "HTTP_413_PAYLOAD_TOO_LARGE"
	HTTPURITooLong           HTTPErrorCode = "HTTP_414_URI_TOO_LONG"
	HTTPUnsupportedMediaType HTTPErrorCode = "HTTP_415_UNSUPPORTED_MEDIA_TYPE"
	HTTPRangeNotSatisfiable  HTTPErrorCode = "HTTP_416_RANGE_NOT_SATISFIABLE"
	HTTPExpectationFailed    HTTPErrorCode = "HTTP_417_EXPECTATION_FAILED"
	HTTPTooManyRequests      HTTPErrorCode = "HTTP_429_TOO_MANY_REQUESTS"

	// 5xx Server Errors.
	HTTPInternalServerError HTTPErrorCode = "HTTP_500_INTERNAL_SERVER_ERROR"
	HTTPNotImplemented      HTTPErrorCode = "HTTP_501_NOT_IMPLEMENTED"
	HTTPBadGateway          HTTPErrorCode = "HTTP_502_BAD_GATEWAY"
	HTTPServiceUnavailable  HTTPErrorCode = "HTTP_503_SERVICE_UNAVAILABLE"
	HTTPGatewayTimeout      HTTPErrorCode = "HTTP_504_GATEWAY_TIMEOUT"
	HTTPVersionNotSupported HTTPErrorCode = "HTTP_505_VERSION_NOT_SUPPORTED"
	HTTPInsufficientStorage HTTPErrorCode = "HTTP_507_INSUFFICIENT_STORAGE"
	HTTPLoopDetected        HTTPErrorCode = "HTTP_508_LOOP_DETECTED"
	HTTPNotExtended         HTTPErrorCode = "HTTP_510_NOT_EXTENDED"
	HTTPNetworkAuthRequired HTTPErrorCode = "HTTP_511_NETWORK_AUTH_REQUIRED"
)

// FileSystemErrorCode represents file system-related error codes.
type FileSystemErrorCode string

const (
	// Permission errors.
	FSPermissionDenied       FileSystemErrorCode = "FS_PERMISSION_DENIED"
	FSAccessDenied           FileSystemErrorCode = "FS_ACCESS_DENIED"
	FSReadOnlyFilesystem     FileSystemErrorCode = "FS_READ_ONLY"
	FSInsufficientPrivileges FileSystemErrorCode = "FS_INSUFFICIENT_PRIVILEGES"

	// Space errors.
	FSInsufficientSpace FileSystemErrorCode = "FS_INSUFFICIENT_SPACE"
	FSQuotaExceeded     FileSystemErrorCode = "FS_QUOTA_EXCEEDED"
	FSDeviceFull        FileSystemErrorCode = "FS_DEVICE_FULL"
	FSInodeExhausted    FileSystemErrorCode = "FS_INODE_EXHAUSTED"

	// File/Directory errors.
	FSFileNotFound      FileSystemErrorCode = "FS_FILE_NOT_FOUND"
	FSDirectoryNotFound FileSystemErrorCode = "FS_DIRECTORY_NOT_FOUND"
	FSFileExists        FileSystemErrorCode = "FS_FILE_EXISTS"
	FSDirectoryExists   FileSystemErrorCode = "FS_DIRECTORY_EXISTS"
	FSIsDirectory       FileSystemErrorCode = "FS_IS_DIRECTORY"
	FSNotDirectory      FileSystemErrorCode = "FS_NOT_DIRECTORY"
	FSDirectoryNotEmpty FileSystemErrorCode = "FS_DIRECTORY_NOT_EMPTY"

	// I/O errors.
	FSIOError    FileSystemErrorCode = "FS_IO_ERROR"
	FSReadError  FileSystemErrorCode = "FS_READ_ERROR"
	FSWriteError FileSystemErrorCode = "FS_WRITE_ERROR"
	FSSeekError  FileSystemErrorCode = "FS_SEEK_ERROR"
	FSFlushError FileSystemErrorCode = "FS_FLUSH_ERROR"

	// Corruption errors.
	FSCorruptedFile     FileSystemErrorCode = "FS_CORRUPTED_FILE"
	FSChecksumMismatch  FileSystemErrorCode = "FS_CHECKSUM_MISMATCH"
	FSUnexpectedEOF     FileSystemErrorCode = "FS_UNEXPECTED_EOF"
	FSBadFileDescriptor FileSystemErrorCode = "FS_BAD_FILE_DESCRIPTOR"

	// Device errors.
	FSDeviceNotReady FileSystemErrorCode = "FS_DEVICE_NOT_READY"
	FSDeviceError    FileSystemErrorCode = "FS_DEVICE_ERROR"
	FSNetworkDrive   FileSystemErrorCode = "FS_NETWORK_DRIVE_ERROR"
	FSRemoteIOError  FileSystemErrorCode = "FS_REMOTE_IO_ERROR"

	// Lock errors.
	FSFileLocked       FileSystemErrorCode = "FS_FILE_LOCKED"
	FSLockViolation    FileSystemErrorCode = "FS_LOCK_VIOLATION"
	FSSharingViolation FileSystemErrorCode = "FS_SHARING_VIOLATION"
)

// ValidationErrorCode represents validation-related error codes.
type ValidationErrorCode string

const (
	// URL validation errors.
	ValidationInvalidURL        ValidationErrorCode = "VALIDATION_INVALID_URL"
	ValidationUnsupportedScheme ValidationErrorCode = "VALIDATION_UNSUPPORTED_SCHEME"
	ValidationMalformedURL      ValidationErrorCode = "VALIDATION_MALFORMED_URL"
	ValidationEmptyURL          ValidationErrorCode = "VALIDATION_EMPTY_URL"
	ValidationURLTooLong        ValidationErrorCode = "VALIDATION_URL_TOO_LONG"
	ValidationInvalidDomain     ValidationErrorCode = "VALIDATION_INVALID_DOMAIN"
	ValidationInvalidPort       ValidationErrorCode = "VALIDATION_INVALID_PORT"

	// File format validation errors.
	ValidationInvalidFileFormat ValidationErrorCode = "VALIDATION_INVALID_FILE_FORMAT"
	ValidationUnsupportedFormat ValidationErrorCode = "VALIDATION_UNSUPPORTED_FORMAT"
	ValidationCorruptedHeader   ValidationErrorCode = "VALIDATION_CORRUPTED_HEADER"
	ValidationInvalidMimeType   ValidationErrorCode = "VALIDATION_INVALID_MIME_TYPE"
	ValidationFileSizeMismatch  ValidationErrorCode = "VALIDATION_FILE_SIZE_MISMATCH"
	ValidationInvalidChecksum   ValidationErrorCode = "VALIDATION_INVALID_CHECKSUM"

	// Parameter validation errors.
	ValidationInvalidParameter  ValidationErrorCode = "VALIDATION_INVALID_PARAMETER"
	ValidationMissingParameter  ValidationErrorCode = "VALIDATION_MISSING_PARAMETER"
	ValidationParameterTooLarge ValidationErrorCode = "VALIDATION_PARAMETER_TOO_LARGE"
	ValidationParameterTooSmall ValidationErrorCode = "VALIDATION_PARAMETER_TOO_SMALL"
	ValidationInvalidRange      ValidationErrorCode = "VALIDATION_INVALID_RANGE"
	ValidationInvalidFormat     ValidationErrorCode = "VALIDATION_INVALID_FORMAT"

	// Configuration validation errors.
	ValidationInvalidConfig  ValidationErrorCode = "VALIDATION_INVALID_CONFIG"
	ValidationMissingConfig  ValidationErrorCode = "VALIDATION_MISSING_CONFIG"
	ValidationConfigConflict ValidationErrorCode = "VALIDATION_CONFIG_CONFLICT"
)

// ErrorCodeRegistry provides centralized error code management.
type ErrorCodeRegistry struct {
	networkCodes    map[NetworkErrorCode]string
	httpCodes       map[HTTPErrorCode]string
	fileSystemCodes map[FileSystemErrorCode]string
	validationCodes map[ValidationErrorCode]string
}

// NewErrorCodeRegistry creates a new error code registry with predefined mappings.
func NewErrorCodeRegistry() *ErrorCodeRegistry {
	registry := &ErrorCodeRegistry{
		networkCodes:    make(map[NetworkErrorCode]string),
		httpCodes:       make(map[HTTPErrorCode]string),
		fileSystemCodes: make(map[FileSystemErrorCode]string),
		validationCodes: make(map[ValidationErrorCode]string),
	}

	registry.initializeNetworkCodes()
	registry.initializeHTTPCodes()
	registry.initializeFileSystemCodes()
	registry.initializeValidationCodes()

	return registry
}

// GetNetworkCodeMessage returns the message for a network error code.
func (r *ErrorCodeRegistry) GetNetworkCodeMessage(code NetworkErrorCode) string {
	if message, exists := r.networkCodes[code]; exists {
		return message
	}

	return "Unknown network error"
}

// GetHTTPCodeMessage returns the message for an HTTP error code.
func (r *ErrorCodeRegistry) GetHTTPCodeMessage(code HTTPErrorCode) string {
	if message, exists := r.httpCodes[code]; exists {
		return message
	}

	return "Unknown HTTP error"
}

// GetFileSystemCodeMessage returns the message for a file system error code.
func (r *ErrorCodeRegistry) GetFileSystemCodeMessage(code FileSystemErrorCode) string {
	if message, exists := r.fileSystemCodes[code]; exists {
		return message
	}

	return "Unknown file system error"
}

// GetValidationCodeMessage returns the message for a validation error code.
func (r *ErrorCodeRegistry) GetValidationCodeMessage(code ValidationErrorCode) string {
	if message, exists := r.validationCodes[code]; exists {
		return message
	}

	return "Unknown validation error"
}

// GetHTTPCodeFromStatus converts HTTP status code to HTTPErrorCode.
func (r *ErrorCodeRegistry) GetHTTPCodeFromStatus(statusCode int) HTTPErrorCode {
	statusCodeMap := map[int]HTTPErrorCode{
		http.StatusBadRequest:                    HTTPBadRequest,
		http.StatusUnauthorized:                  HTTPUnauthorized,
		http.StatusPaymentRequired:               HTTPPaymentRequired,
		http.StatusForbidden:                     HTTPForbidden,
		http.StatusNotFound:                      HTTPNotFound,
		http.StatusMethodNotAllowed:              HTTPMethodNotAllowed,
		http.StatusNotAcceptable:                 HTTPNotAcceptable,
		http.StatusProxyAuthRequired:             HTTPProxyAuthRequired,
		http.StatusRequestTimeout:                HTTPRequestTimeout,
		http.StatusConflict:                      HTTPConflict,
		http.StatusGone:                          HTTPGone,
		http.StatusLengthRequired:                HTTPLengthRequired,
		http.StatusPreconditionFailed:            HTTPPreconditionFailed,
		http.StatusRequestEntityTooLarge:         HTTPPayloadTooLarge,
		http.StatusRequestURITooLong:             HTTPURITooLong,
		http.StatusUnsupportedMediaType:          HTTPUnsupportedMediaType,
		http.StatusRequestedRangeNotSatisfiable:  HTTPRangeNotSatisfiable,
		http.StatusExpectationFailed:             HTTPExpectationFailed,
		http.StatusTooManyRequests:               HTTPTooManyRequests,
		http.StatusInternalServerError:           HTTPInternalServerError,
		http.StatusNotImplemented:                HTTPNotImplemented,
		http.StatusBadGateway:                    HTTPBadGateway,
		http.StatusServiceUnavailable:            HTTPServiceUnavailable,
		http.StatusGatewayTimeout:                HTTPGatewayTimeout,
		http.StatusHTTPVersionNotSupported:       HTTPVersionNotSupported,
		http.StatusInsufficientStorage:           HTTPInsufficientStorage,
		http.StatusLoopDetected:                  HTTPLoopDetected,
		http.StatusNotExtended:                   HTTPNotExtended,
		http.StatusNetworkAuthenticationRequired: HTTPNetworkAuthRequired,
	}

	if code, exists := statusCodeMap[statusCode]; exists {
		return code
	}
	return HTTPErrorCode(fmt.Sprintf("HTTP_%d_UNKNOWN", statusCode))
}

// IsRetryableNetworkCode determines if a network error code is retryable.
func (r *ErrorCodeRegistry) IsRetryableNetworkCode(code NetworkErrorCode) bool {
	retryableCodes := map[NetworkErrorCode]bool{
		NetworkDNSTimeout:         true,
		NetworkConnectionTimeout:  true,
		NetworkConnectionRefused:  true,
		NetworkConnectionReset:    true,
		NetworkReadTimeout:        true,
		NetworkWriteTimeout:       true,
		NetworkRequestTimeout:     true,
		NetworkHostUnreachable:    true,
		NetworkNetworkUnreachable: true,
	}

	return retryableCodes[code]
}

// IsRetryableHTTPCode determines if an HTTP error code is retryable.
func (r *ErrorCodeRegistry) IsRetryableHTTPCode(code HTTPErrorCode) bool {
	retryableCodes := map[HTTPErrorCode]bool{
		HTTPRequestTimeout:      true,
		HTTPTooManyRequests:     true,
		HTTPInternalServerError: true,
		HTTPBadGateway:          true,
		HTTPServiceUnavailable:  true,
		HTTPGatewayTimeout:      true,
		HTTPInsufficientStorage: true,
	}

	return retryableCodes[code]
}

// Initialize methods for each error code category.
func (r *ErrorCodeRegistry) initializeNetworkCodes() {
	r.networkCodes[NetworkDNSNotFound] = "Domain name could not be resolved"
	r.networkCodes[NetworkDNSTimeout] = "DNS resolution timed out"
	r.networkCodes[NetworkDNSFailure] = "DNS resolution failed"
	r.networkCodes[NetworkConnectionRefused] = "Connection was refused by the server"
	r.networkCodes[NetworkConnectionTimeout] = "Connection attempt timed out"
	r.networkCodes[NetworkConnectionReset] = "Connection was reset by the server"
	r.networkCodes[NetworkConnectionAborted] = "Connection was aborted"
	r.networkCodes[NetworkHostUnreachable] = "Host is unreachable"
	r.networkCodes[NetworkNetworkUnreachable] = "Network is unreachable"
	r.networkCodes[NetworkTLSHandshakeFailure] = "TLS/SSL handshake failed"
	r.networkCodes[NetworkProtocolError] = "Protocol error occurred"
	r.networkCodes[NetworkProxyError] = "Proxy server error"
	r.networkCodes[NetworkReadTimeout] = "Read operation timed out"
	r.networkCodes[NetworkWriteTimeout] = "Write operation timed out"
	r.networkCodes[NetworkRequestTimeout] = "Request timed out"
}

func (r *ErrorCodeRegistry) initializeHTTPCodes() {
	r.httpCodes[HTTPBadRequest] = "Bad request - invalid syntax"
	r.httpCodes[HTTPUnauthorized] = "Authentication required"
	r.httpCodes[HTTPPaymentRequired] = "Payment required"
	r.httpCodes[HTTPForbidden] = "Access forbidden"
	r.httpCodes[HTTPNotFound] = "Resource not found"
	r.httpCodes[HTTPMethodNotAllowed] = "HTTP method not allowed"
	r.httpCodes[HTTPNotAcceptable] = "Response format not acceptable"
	r.httpCodes[HTTPProxyAuthRequired] = "Proxy authentication required"
	r.httpCodes[HTTPRequestTimeout] = "Request timeout"
	r.httpCodes[HTTPConflict] = "Request conflicts with current state"
	r.httpCodes[HTTPGone] = "Resource no longer available"
	r.httpCodes[HTTPLengthRequired] = "Content-Length header required"
	r.httpCodes[HTTPPreconditionFailed] = "Precondition failed"
	r.httpCodes[HTTPPayloadTooLarge] = "Request payload too large"
	r.httpCodes[HTTPURITooLong] = "URI too long"
	r.httpCodes[HTTPUnsupportedMediaType] = "Unsupported media type"
	r.httpCodes[HTTPRangeNotSatisfiable] = "Range not satisfiable"
	r.httpCodes[HTTPExpectationFailed] = "Expectation failed"
	r.httpCodes[HTTPTooManyRequests] = "Too many requests"
	r.httpCodes[HTTPInternalServerError] = "Internal server error"
	r.httpCodes[HTTPNotImplemented] = "Feature not implemented"
	r.httpCodes[HTTPBadGateway] = "Bad gateway"
	r.httpCodes[HTTPServiceUnavailable] = "Service temporarily unavailable"
	r.httpCodes[HTTPGatewayTimeout] = "Gateway timeout"
	r.httpCodes[HTTPVersionNotSupported] = "HTTP version not supported"
	r.httpCodes[HTTPInsufficientStorage] = "Insufficient storage space"
	r.httpCodes[HTTPLoopDetected] = "Loop detected"
	r.httpCodes[HTTPNotExtended] = "Further extensions required"
	r.httpCodes[HTTPNetworkAuthRequired] = "Network authentication required"
}

func (r *ErrorCodeRegistry) initializeFileSystemCodes() {
	r.fileSystemCodes[FSPermissionDenied] = "Permission denied"
	r.fileSystemCodes[FSAccessDenied] = "Access denied"
	r.fileSystemCodes[FSReadOnlyFilesystem] = "Read-only filesystem"
	r.fileSystemCodes[FSInsufficientPrivileges] = "Insufficient privileges"
	r.fileSystemCodes[FSInsufficientSpace] = "Insufficient disk space"
	r.fileSystemCodes[FSQuotaExceeded] = "Disk quota exceeded"
	r.fileSystemCodes[FSDeviceFull] = "Device is full"
	r.fileSystemCodes[FSInodeExhausted] = "No free inodes available"
	r.fileSystemCodes[FSFileNotFound] = "File not found"
	r.fileSystemCodes[FSDirectoryNotFound] = "Directory not found"
	r.fileSystemCodes[FSFileExists] = "File already exists"
	r.fileSystemCodes[FSDirectoryExists] = "Directory already exists"
	r.fileSystemCodes[FSIsDirectory] = "Target is a directory"
	r.fileSystemCodes[FSNotDirectory] = "Not a directory"
	r.fileSystemCodes[FSDirectoryNotEmpty] = "Directory not empty"
	r.fileSystemCodes[FSIOError] = "Input/output error"
	r.fileSystemCodes[FSReadError] = "Read error"
	r.fileSystemCodes[FSWriteError] = "Write error"
	r.fileSystemCodes[FSSeekError] = "Seek error"
	r.fileSystemCodes[FSFlushError] = "Flush error"
	r.fileSystemCodes[FSCorruptedFile] = "File is corrupted"
	r.fileSystemCodes[FSChecksumMismatch] = "Checksum verification failed"
	r.fileSystemCodes[FSUnexpectedEOF] = "Unexpected end of file"
	r.fileSystemCodes[FSBadFileDescriptor] = "Bad file descriptor"
	r.fileSystemCodes[FSDeviceNotReady] = "Device not ready"
	r.fileSystemCodes[FSDeviceError] = "Device error"
	r.fileSystemCodes[FSNetworkDrive] = "Network drive error"
	r.fileSystemCodes[FSRemoteIOError] = "Remote I/O error"
	r.fileSystemCodes[FSFileLocked] = "File is locked"
	r.fileSystemCodes[FSLockViolation] = "Lock violation"
	r.fileSystemCodes[FSSharingViolation] = "Sharing violation"
}

func (r *ErrorCodeRegistry) initializeValidationCodes() {
	r.validationCodes[ValidationInvalidURL] = "Invalid URL format"
	r.validationCodes[ValidationUnsupportedScheme] = "Unsupported URL scheme"
	r.validationCodes[ValidationMalformedURL] = "Malformed URL"
	r.validationCodes[ValidationEmptyURL] = "Empty URL provided"
	r.validationCodes[ValidationURLTooLong] = "URL too long"
	r.validationCodes[ValidationInvalidDomain] = "Invalid domain name"
	r.validationCodes[ValidationInvalidPort] = "Invalid port number"
	r.validationCodes[ValidationInvalidFileFormat] = "Invalid file format"
	r.validationCodes[ValidationUnsupportedFormat] = "Unsupported file format"
	r.validationCodes[ValidationCorruptedHeader] = "Corrupted file header"
	r.validationCodes[ValidationInvalidMimeType] = "Invalid MIME type"
	r.validationCodes[ValidationFileSizeMismatch] = "File size mismatch"
	r.validationCodes[ValidationInvalidChecksum] = "Invalid checksum"
	r.validationCodes[ValidationInvalidParameter] = "Invalid parameter"
	r.validationCodes[ValidationMissingParameter] = "Missing required parameter"
	r.validationCodes[ValidationParameterTooLarge] = "Parameter value too large"
	r.validationCodes[ValidationParameterTooSmall] = "Parameter value too small"
	r.validationCodes[ValidationInvalidRange] = "Invalid range specification"
	r.validationCodes[ValidationInvalidFormat] = "Invalid format"
	r.validationCodes[ValidationInvalidConfig] = "Invalid configuration"
	r.validationCodes[ValidationMissingConfig] = "Missing configuration"
	r.validationCodes[ValidationConfigConflict] = "Configuration conflict"
}
