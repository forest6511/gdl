// Package config provides configuration management for the gdl download tool.
// It handles retry policies, error handling preferences, output formats, and timeout values.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RetryPolicyConfig defines the retry policy configuration.
type RetryPolicyConfig struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// BaseDelay is the initial delay between retries
	BaseDelay time.Duration `json:"base_delay" yaml:"base_delay"`

	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration `json:"max_delay" yaml:"max_delay"`

	// BackoffFactor is the exponential backoff multiplier
	BackoffFactor float64 `json:"backoff_factor" yaml:"backoff_factor"`

	// Jitter enables random jitter in retry delays
	Jitter bool `json:"jitter" yaml:"jitter"`

	// Strategy defines the retry strategy type (exponential, linear, constant, custom)
	Strategy string `json:"strategy" yaml:"strategy"`

	// RetryableErrors is a list of error types that should be retried
	RetryableErrors []string `json:"retryable_errors,omitempty" yaml:"retryable_errors,omitempty"`

	// NonRetryableErrors is a list of error types that should never be retried
	NonRetryableErrors []string `json:"non_retryable_errors,omitempty" yaml:"non_retryable_errors,omitempty"`
}

// ErrorHandlingConfig defines error handling preferences.
type ErrorHandlingConfig struct {
	// VerboseErrors enables detailed error messages
	VerboseErrors bool `json:"verbose_errors" yaml:"verbose_errors"`

	// ShowStackTrace includes stack traces in error output
	ShowStackTrace bool `json:"show_stack_trace" yaml:"show_stack_trace"`

	// LogErrors enables error logging to file
	LogErrors bool `json:"log_errors" yaml:"log_errors"`

	// LogFile is the path to the error log file
	LogFile string `json:"log_file,omitempty" yaml:"log_file,omitempty"`

	// FailFast stops on first error instead of retrying
	FailFast bool `json:"fail_fast" yaml:"fail_fast"`

	// ErrorFormat defines how errors should be formatted (json, text, structured)
	ErrorFormat string `json:"error_format" yaml:"error_format"`

	// RecoveryEnabled enables intelligent error recovery suggestions
	RecoveryEnabled bool `json:"recovery_enabled" yaml:"recovery_enabled"`

	// NetworkDiagnostics enables network connectivity diagnostics
	NetworkDiagnostics bool `json:"network_diagnostics" yaml:"network_diagnostics"`
}

// OutputFormatConfig defines output format settings.
type OutputFormatConfig struct {
	// Format defines the output format (json, yaml, text, table)
	Format string `json:"format" yaml:"format"`

	// Pretty enables pretty-printing for structured formats
	Pretty bool `json:"pretty" yaml:"pretty"`

	// Color enables colored output
	Color bool `json:"color" yaml:"color"`

	// ShowProgress enables progress bars and indicators
	ShowProgress bool `json:"show_progress" yaml:"show_progress"`

	// Quiet reduces output verbosity
	Quiet bool `json:"quiet" yaml:"quiet"`

	// Verbose increases output verbosity
	Verbose bool `json:"verbose" yaml:"verbose"`

	// TimestampFormat defines timestamp format for logs
	TimestampFormat string `json:"timestamp_format,omitempty" yaml:"timestamp_format,omitempty"`

	// LogLevel defines the minimum log level to output
	LogLevel string `json:"log_level" yaml:"log_level"`
}

// TimeoutConfig defines timeout values for various operations.
type TimeoutConfig struct {
	// ConnectTimeout is the timeout for establishing connections
	ConnectTimeout time.Duration `json:"connect_timeout" yaml:"connect_timeout"`

	// ReadTimeout is the timeout for reading data
	ReadTimeout time.Duration `json:"read_timeout" yaml:"read_timeout"`

	// WriteTimeout is the timeout for writing data
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`

	// RequestTimeout is the overall timeout for HTTP requests
	RequestTimeout time.Duration `json:"request_timeout" yaml:"request_timeout"`

	// DownloadTimeout is the maximum time allowed for a complete download
	DownloadTimeout time.Duration `json:"download_timeout" yaml:"download_timeout"`

	// IdleTimeout is the timeout for idle connections
	IdleTimeout time.Duration `json:"idle_timeout" yaml:"idle_timeout"`
}

// NetworkConfig defines network-related configuration.
type NetworkConfig struct {
	// UserAgent is the HTTP User-Agent string
	UserAgent string `json:"user_agent,omitempty" yaml:"user_agent,omitempty"`

	// MaxConcurrentDownloads limits the number of simultaneous downloads
	MaxConcurrentDownloads int `json:"max_concurrent_downloads" yaml:"max_concurrent_downloads"`

	// ChunkSize is the size of download chunks in bytes
	ChunkSize int `json:"chunk_size" yaml:"chunk_size"`

	// BufferSize is the I/O buffer size in bytes
	BufferSize int `json:"buffer_size" yaml:"buffer_size"`

	// FollowRedirects enables automatic redirect following
	FollowRedirects bool `json:"follow_redirects" yaml:"follow_redirects"`

	// MaxRedirects is the maximum number of redirects to follow
	MaxRedirects int `json:"max_redirects" yaml:"max_redirects"`

	// InsecureTLS disables TLS certificate verification
	InsecureTLS bool `json:"insecure_tls" yaml:"insecure_tls"`
}

// StorageConfig defines storage-related configuration.
type StorageConfig struct {
	// DefaultDownloadDir is the default directory for downloads
	DefaultDownloadDir string `json:"default_download_dir,omitempty" yaml:"default_download_dir,omitempty"`

	// CreateDirs enables automatic directory creation
	CreateDirs bool `json:"create_dirs" yaml:"create_dirs"`

	// OverwriteExisting allows overwriting existing files
	OverwriteExisting bool `json:"overwrite_existing" yaml:"overwrite_existing"`

	// ResumeSupport enables resume support for interrupted downloads
	ResumeSupport bool `json:"resume_support" yaml:"resume_support"`

	// MinFreeSpace is the minimum free space required before downloading
	MinFreeSpace int64 `json:"min_free_space" yaml:"min_free_space"`

	// TempDir is the directory for temporary files
	TempDir string `json:"temp_dir,omitempty" yaml:"temp_dir,omitempty"`
}

// PluginConfig represents the configuration for a single plugin
type PluginConfig struct {
	Enabled  bool                   `json:"enabled" yaml:"enabled"`
	Name     string                 `json:"name" yaml:"name"`
	Type     string                 `json:"type" yaml:"type"`
	Path     string                 `json:"path,omitempty" yaml:"path,omitempty"`
	Settings map[string]interface{} `json:"settings,omitempty" yaml:"settings,omitempty"`
	Priority int                    `json:"priority,omitempty" yaml:"priority,omitempty"`
}

// MiddlewareConfig represents the configuration for middleware
type MiddlewareConfig struct {
	Name     string                 `json:"name" yaml:"name"`
	Enabled  bool                   `json:"enabled" yaml:"enabled"`
	Priority int                    `json:"priority,omitempty" yaml:"priority,omitempty"`
	Settings map[string]interface{} `json:"settings,omitempty" yaml:"settings,omitempty"`
}

// Config represents the complete configuration for the gdl application.
type Config struct {
	// Version is the configuration schema version
	Version string `json:"version" yaml:"version"`

	// RetryPolicy defines retry behavior
	RetryPolicy RetryPolicyConfig `json:"retry_policy" yaml:"retry_policy"`

	// ErrorHandling defines error handling preferences
	ErrorHandling ErrorHandlingConfig `json:"error_handling" yaml:"error_handling"`

	// OutputFormat defines output formatting
	OutputFormat OutputFormatConfig `json:"output_format" yaml:"output_format"`

	// Timeouts defines various timeout values
	Timeouts TimeoutConfig `json:"timeouts" yaml:"timeouts"`

	// Network defines network-related settings
	Network NetworkConfig `json:"network" yaml:"network"`

	// Storage defines storage-related settings
	Storage StorageConfig `json:"storage" yaml:"storage"`

	// Plugins defines plugin configurations
	Plugins []PluginConfig `json:"plugins,omitempty" yaml:"plugins,omitempty"`

	// Middleware defines middleware chain configuration
	Middleware []MiddlewareConfig `json:"middleware,omitempty" yaml:"middleware,omitempty"`

	// Hooks defines event hook configurations
	Hooks map[string][]string `json:"hooks,omitempty" yaml:"hooks,omitempty"`
}

// DefaultConfig returns a configuration with sensible default values.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	defaultDownloadDir := filepath.Join(homeDir, "Downloads")

	return &Config{
		Version: "1.0",
		RetryPolicy: RetryPolicyConfig{
			MaxRetries:    3,
			BaseDelay:     1 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
			Jitter:        true,
			Strategy:      "exponential",
			RetryableErrors: []string{
				"network_error",
				"timeout",
				"server_error",
			},
			NonRetryableErrors: []string{
				"invalid_url",
				"file_exists",
				"permission_denied",
				"authentication_failed",
			},
		},
		ErrorHandling: ErrorHandlingConfig{
			VerboseErrors:      false,
			ShowStackTrace:     false,
			LogErrors:          false,
			FailFast:           false,
			ErrorFormat:        "text",
			RecoveryEnabled:    true,
			NetworkDiagnostics: true,
		},
		OutputFormat: OutputFormatConfig{
			Format:          "text",
			Pretty:          true,
			Color:           true,
			ShowProgress:    true,
			Quiet:           false,
			Verbose:         false,
			TimestampFormat: "2006-01-02 15:04:05",
			LogLevel:        "info",
		},
		Timeouts: TimeoutConfig{
			ConnectTimeout:  10 * time.Second,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    10 * time.Second,
			RequestTimeout:  60 * time.Second,
			DownloadTimeout: 30 * time.Minute,
			IdleTimeout:     90 * time.Second,
		},
		Network: NetworkConfig{
			UserAgent:              "gdl/1.0",
			MaxConcurrentDownloads: 4,
			ChunkSize:              32 * 1024, // 32KB
			BufferSize:             8 * 1024,  // 8KB
			FollowRedirects:        true,
			MaxRedirects:           10,
			InsecureTLS:            false,
		},
		Storage: StorageConfig{
			DefaultDownloadDir: defaultDownloadDir,
			CreateDirs:         true,
			OverwriteExisting:  false,
			ResumeSupport:      true,
			MinFreeSpace:       100 * 1024 * 1024, // 100MB
			TempDir:            os.TempDir(),
		},
	}
}

// ConfigLoader handles loading and saving configuration.
type ConfigLoader struct {
	configPath string
}

// NewConfigLoader creates a new configuration loader.
func NewConfigLoader(configPath string) *ConfigLoader {
	return &ConfigLoader{
		configPath: configPath,
	}
}

// DefaultConfigPath returns the default configuration file path.
func DefaultConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "gdl")
	configPath := filepath.Join(configDir, "config.json")

	return configPath, nil
}

// Load loads configuration from file, falling back to defaults if file doesn't exist.
func (cl *ConfigLoader) Load() (*Config, error) {
	// If config file doesn't exist, return default configuration
	if _, err := os.Stat(cl.configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	// Read configuration file
	data, err := os.ReadFile(cl.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", cl.configPath, err)
	}

	// Parse JSON configuration
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", cl.configPath, err)
	}

	// Validate and apply defaults for missing fields
	cl.applyDefaults(&config)

	return &config, nil
}

// Save saves configuration to file.
func (cl *ConfigLoader) Save(config *Config) error {
	// Ensure config directory exists
	configDir := filepath.Dir(cl.configPath)
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	// Marshal configuration to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write configuration file
	if err := os.WriteFile(cl.configPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", cl.configPath, err)
	}

	return nil
}

// applyDefaults applies default values for any missing configuration fields.
func (cl *ConfigLoader) applyRetryPolicyDefaults(config, defaults *Config) {
	if config.RetryPolicy.MaxRetries == 0 {
		config.RetryPolicy.MaxRetries = defaults.RetryPolicy.MaxRetries
	}
	if config.RetryPolicy.BaseDelay == 0 {
		config.RetryPolicy.BaseDelay = defaults.RetryPolicy.BaseDelay
	}
	if config.RetryPolicy.MaxDelay == 0 {
		config.RetryPolicy.MaxDelay = defaults.RetryPolicy.MaxDelay
	}
	if config.RetryPolicy.BackoffFactor == 0 {
		config.RetryPolicy.BackoffFactor = defaults.RetryPolicy.BackoffFactor
	}
	if config.RetryPolicy.Strategy == "" {
		config.RetryPolicy.Strategy = defaults.RetryPolicy.Strategy
	}
}

func (cl *ConfigLoader) applyErrorHandlingDefaults(config, defaults *Config) {
	if config.ErrorHandling.ErrorFormat == "" {
		config.ErrorHandling.ErrorFormat = defaults.ErrorHandling.ErrorFormat
	}
}

func (cl *ConfigLoader) applyOutputFormatDefaults(config, defaults *Config) {
	if config.OutputFormat.Format == "" {
		config.OutputFormat.Format = defaults.OutputFormat.Format
	}
	if config.OutputFormat.TimestampFormat == "" {
		config.OutputFormat.TimestampFormat = defaults.OutputFormat.TimestampFormat
	}
	if config.OutputFormat.LogLevel == "" {
		config.OutputFormat.LogLevel = defaults.OutputFormat.LogLevel
	}
}

func (cl *ConfigLoader) applyTimeoutDefaults(config, defaults *Config) {
	if config.Timeouts.ConnectTimeout == 0 {
		config.Timeouts.ConnectTimeout = defaults.Timeouts.ConnectTimeout
	}
	if config.Timeouts.ReadTimeout == 0 {
		config.Timeouts.ReadTimeout = defaults.Timeouts.ReadTimeout
	}
	if config.Timeouts.WriteTimeout == 0 {
		config.Timeouts.WriteTimeout = defaults.Timeouts.WriteTimeout
	}
	if config.Timeouts.RequestTimeout == 0 {
		config.Timeouts.RequestTimeout = defaults.Timeouts.RequestTimeout
	}
	if config.Timeouts.DownloadTimeout == 0 {
		config.Timeouts.DownloadTimeout = defaults.Timeouts.DownloadTimeout
	}
	if config.Timeouts.IdleTimeout == 0 {
		config.Timeouts.IdleTimeout = defaults.Timeouts.IdleTimeout
	}
}

func (cl *ConfigLoader) applyNetworkDefaults(config, defaults *Config) {
	if config.Network.UserAgent == "" {
		config.Network.UserAgent = defaults.Network.UserAgent
	}
	if config.Network.MaxConcurrentDownloads == 0 {
		config.Network.MaxConcurrentDownloads = defaults.Network.MaxConcurrentDownloads
	}
	if config.Network.ChunkSize == 0 {
		config.Network.ChunkSize = defaults.Network.ChunkSize
	}
	if config.Network.BufferSize == 0 {
		config.Network.BufferSize = defaults.Network.BufferSize
	}
	if config.Network.MaxRedirects == 0 {
		config.Network.MaxRedirects = defaults.Network.MaxRedirects
	}
}

func (cl *ConfigLoader) applyStorageDefaults(config, defaults *Config) {
	if config.Storage.DefaultDownloadDir == "" {
		config.Storage.DefaultDownloadDir = defaults.Storage.DefaultDownloadDir
	}
	if config.Storage.MinFreeSpace == 0 {
		config.Storage.MinFreeSpace = defaults.Storage.MinFreeSpace
	}
	if config.Storage.TempDir == "" {
		config.Storage.TempDir = defaults.Storage.TempDir
	}
}

func (cl *ConfigLoader) applyDefaults(config *Config) {
	defaults := DefaultConfig()

	cl.applyRetryPolicyDefaults(config, defaults)
	cl.applyErrorHandlingDefaults(config, defaults)
	cl.applyOutputFormatDefaults(config, defaults)
	cl.applyTimeoutDefaults(config, defaults)
	cl.applyNetworkDefaults(config, defaults)
	cl.applyStorageDefaults(config, defaults)

	// Set version if not specified
	if config.Version == "" {
		config.Version = defaults.Version
	}
}

// Validate validates the configuration for consistency and correctness.
func (c *Config) validateRetryPolicy() error {
	if c.RetryPolicy.MaxRetries < 0 {
		return fmt.Errorf(
			"retry policy max_retries must be non-negative, got %d",
			c.RetryPolicy.MaxRetries,
		)
	}
	if c.RetryPolicy.BaseDelay <= 0 {
		return fmt.Errorf(
			"retry policy base_delay must be positive, got %v",
			c.RetryPolicy.BaseDelay,
		)
	}
	if c.RetryPolicy.MaxDelay <= 0 {
		return fmt.Errorf("retry policy max_delay must be positive, got %v", c.RetryPolicy.MaxDelay)
	}
	if c.RetryPolicy.BackoffFactor <= 0 {
		return fmt.Errorf(
			"retry policy backoff_factor must be positive, got %f",
			c.RetryPolicy.BackoffFactor,
		)
	}

	validStrategies := map[string]bool{
		"exponential": true,
		"linear":      true,
		"constant":    true,
		"custom":      true,
	}
	if !validStrategies[c.RetryPolicy.Strategy] {
		return fmt.Errorf("invalid retry strategy: %s", c.RetryPolicy.Strategy)
	}

	return nil
}

func (c *Config) validateErrorHandling() error {
	validErrorFormats := map[string]bool{
		"json":       true,
		"text":       true,
		"structured": true,
	}
	if !validErrorFormats[c.ErrorHandling.ErrorFormat] {
		return fmt.Errorf("invalid error format: %s", c.ErrorHandling.ErrorFormat)
	}
	return nil
}

func (c *Config) validateOutputFormat() error {
	validOutputFormats := map[string]bool{
		"json":  true,
		"yaml":  true,
		"text":  true,
		"table": true,
	}
	if !validOutputFormats[c.OutputFormat.Format] {
		return fmt.Errorf("invalid output format: %s", c.OutputFormat.Format)
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.OutputFormat.LogLevel] {
		return fmt.Errorf("invalid log level: %s", c.OutputFormat.LogLevel)
	}

	return nil
}

func (c *Config) validateTimeouts() error {
	if c.Timeouts.ConnectTimeout <= 0 {
		return fmt.Errorf("connect timeout must be positive, got %v", c.Timeouts.ConnectTimeout)
	}
	if c.Timeouts.ReadTimeout <= 0 {
		return fmt.Errorf("read timeout must be positive, got %v", c.Timeouts.ReadTimeout)
	}
	if c.Timeouts.WriteTimeout <= 0 {
		return fmt.Errorf("write timeout must be positive, got %v", c.Timeouts.WriteTimeout)
	}
	if c.Timeouts.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive, got %v", c.Timeouts.RequestTimeout)
	}
	if c.Timeouts.DownloadTimeout <= 0 {
		return fmt.Errorf("download timeout must be positive, got %v", c.Timeouts.DownloadTimeout)
	}
	return nil
}

func (c *Config) validateNetwork() error {
	if c.Network.MaxConcurrentDownloads <= 0 {
		return fmt.Errorf(
			"max concurrent downloads must be positive, got %d",
			c.Network.MaxConcurrentDownloads,
		)
	}
	if c.Network.ChunkSize <= 0 {
		return fmt.Errorf("chunk size must be positive, got %d", c.Network.ChunkSize)
	}
	if c.Network.BufferSize <= 0 {
		return fmt.Errorf("buffer size must be positive, got %d", c.Network.BufferSize)
	}
	if c.Network.MaxRedirects < 0 {
		return fmt.Errorf("max redirects must be non-negative, got %d", c.Network.MaxRedirects)
	}
	return nil
}

func (c *Config) validateStorage() error {
	if c.Storage.MinFreeSpace < 0 {
		return fmt.Errorf("min free space must be non-negative, got %d", c.Storage.MinFreeSpace)
	}
	return nil
}

func (c *Config) Validate() error {
	if err := c.validateRetryPolicy(); err != nil {
		return err
	}
	if err := c.validateErrorHandling(); err != nil {
		return err
	}
	if err := c.validateOutputFormat(); err != nil {
		return err
	}
	if err := c.validateTimeouts(); err != nil {
		return err
	}
	if err := c.validateNetwork(); err != nil {
		return err
	}
	if err := c.validateStorage(); err != nil {
		return err
	}

	return nil
}

// Clone creates a deep copy of the configuration.
func (c *Config) Clone() *Config {
	// Marshal to JSON and back to create a deep copy
	data, _ := json.Marshal(c)

	var clone Config

	_ = json.Unmarshal(data, &clone)

	return &clone
}

// Merge merges another configuration into this one, with the other config taking precedence.
func (c *Config) mergeRetryPolicy(other *RetryPolicyConfig) {
	if other.MaxRetries != 0 {
		c.RetryPolicy.MaxRetries = other.MaxRetries
	}
	if other.BaseDelay != 0 {
		c.RetryPolicy.BaseDelay = other.BaseDelay
	}
	if other.MaxDelay != 0 {
		c.RetryPolicy.MaxDelay = other.MaxDelay
	}
	if other.BackoffFactor != 0 {
		c.RetryPolicy.BackoffFactor = other.BackoffFactor
	}
	if other.Strategy != "" {
		c.RetryPolicy.Strategy = other.Strategy
	}
	c.RetryPolicy.Jitter = other.Jitter

	// Merge retryable/non-retryable errors
	if len(other.RetryableErrors) > 0 {
		c.RetryPolicy.RetryableErrors = other.RetryableErrors
	}
	if len(other.NonRetryableErrors) > 0 {
		c.RetryPolicy.NonRetryableErrors = other.NonRetryableErrors
	}
}

func (c *Config) mergeErrorHandling(other *ErrorHandlingConfig) {
	c.ErrorHandling.VerboseErrors = other.VerboseErrors
	c.ErrorHandling.ShowStackTrace = other.ShowStackTrace
	c.ErrorHandling.LogErrors = other.LogErrors
	c.ErrorHandling.FailFast = other.FailFast
	c.ErrorHandling.RecoveryEnabled = other.RecoveryEnabled
	c.ErrorHandling.NetworkDiagnostics = other.NetworkDiagnostics

	if other.LogFile != "" {
		c.ErrorHandling.LogFile = other.LogFile
	}
	if other.ErrorFormat != "" {
		c.ErrorHandling.ErrorFormat = other.ErrorFormat
	}
}

func (c *Config) mergeOutputFormat(other *OutputFormatConfig) {
	if other.Format != "" {
		c.OutputFormat.Format = other.Format
	}
	c.OutputFormat.Pretty = other.Pretty
	c.OutputFormat.Color = other.Color
	c.OutputFormat.ShowProgress = other.ShowProgress
	c.OutputFormat.Quiet = other.Quiet
	c.OutputFormat.Verbose = other.Verbose

	if other.TimestampFormat != "" {
		c.OutputFormat.TimestampFormat = other.TimestampFormat
	}
	if other.LogLevel != "" {
		c.OutputFormat.LogLevel = other.LogLevel
	}
}

func (c *Config) mergeTimeouts(other *TimeoutConfig) {
	if other.ConnectTimeout != 0 {
		c.Timeouts.ConnectTimeout = other.ConnectTimeout
	}
	if other.ReadTimeout != 0 {
		c.Timeouts.ReadTimeout = other.ReadTimeout
	}
	if other.WriteTimeout != 0 {
		c.Timeouts.WriteTimeout = other.WriteTimeout
	}
	if other.RequestTimeout != 0 {
		c.Timeouts.RequestTimeout = other.RequestTimeout
	}
	if other.DownloadTimeout != 0 {
		c.Timeouts.DownloadTimeout = other.DownloadTimeout
	}
	if other.IdleTimeout != 0 {
		c.Timeouts.IdleTimeout = other.IdleTimeout
	}
}

func (c *Config) mergeNetwork(other *NetworkConfig) {
	if other.UserAgent != "" {
		c.Network.UserAgent = other.UserAgent
	}
	if other.MaxConcurrentDownloads != 0 {
		c.Network.MaxConcurrentDownloads = other.MaxConcurrentDownloads
	}
	if other.ChunkSize != 0 {
		c.Network.ChunkSize = other.ChunkSize
	}
	if other.BufferSize != 0 {
		c.Network.BufferSize = other.BufferSize
	}
	if other.MaxRedirects != 0 {
		c.Network.MaxRedirects = other.MaxRedirects
	}
	c.Network.FollowRedirects = other.FollowRedirects
	c.Network.InsecureTLS = other.InsecureTLS
}

func (c *Config) mergeStorage(other *StorageConfig) {
	if other.DefaultDownloadDir != "" {
		c.Storage.DefaultDownloadDir = other.DefaultDownloadDir
	}
	if other.TempDir != "" {
		c.Storage.TempDir = other.TempDir
	}
	if other.MinFreeSpace != 0 {
		c.Storage.MinFreeSpace = other.MinFreeSpace
	}
	c.Storage.CreateDirs = other.CreateDirs
	c.Storage.OverwriteExisting = other.OverwriteExisting
	c.Storage.ResumeSupport = other.ResumeSupport
}

func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}

	c.mergeRetryPolicy(&other.RetryPolicy)
	c.mergeErrorHandling(&other.ErrorHandling)
	c.mergeOutputFormat(&other.OutputFormat)
	c.mergeTimeouts(&other.Timeouts)
	c.mergeNetwork(&other.Network)
	c.mergeStorage(&other.Storage)

	// Update version
	if other.Version != "" {
		c.Version = other.Version
	}
}

// ConfigManager provides high-level configuration management operations.
type ConfigManager struct {
	loader *ConfigLoader
	config *Config
}

// NewConfigManager creates a new configuration manager.
func NewConfigManager(configPath string) (*ConfigManager, error) {
	loader := NewConfigLoader(configPath)

	config, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &ConfigManager{
		loader: loader,
		config: config,
	}, nil
}

// NewDefaultConfigManager creates a configuration manager with default path.
func NewDefaultConfigManager() (*ConfigManager, error) {
	configPath, err := DefaultConfigPath()
	if err != nil {
		return nil, err
	}

	return NewConfigManager(configPath)
}

// GetConfig returns the current configuration.
func (cm *ConfigManager) GetConfig() *Config {
	return cm.config.Clone()
}

// UpdateConfig updates the configuration and saves it to file.
func (cm *ConfigManager) UpdateConfig(config *Config) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	cm.config = config.Clone()

	return cm.loader.Save(cm.config)
}

// SaveConfig saves the current configuration to file.
func (cm *ConfigManager) SaveConfig() error {
	return cm.loader.Save(cm.config)
}

// ReloadConfig reloads the configuration from file.
func (cm *ConfigManager) ReloadConfig() error {
	config, err := cm.loader.Load()
	if err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	cm.config = config

	return nil
}
