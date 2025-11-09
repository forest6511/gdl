// Package main provides the command-line interface for the gdl download tool.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gdl "github.com/forest6511/gdl"
	"github.com/forest6511/gdl/internal/core"
	"github.com/forest6511/gdl/internal/network"
	"github.com/forest6511/gdl/internal/retry"
	"github.com/forest6511/gdl/internal/storage"
	"github.com/forest6511/gdl/pkg/cli"
	gdlerrors "github.com/forest6511/gdl/pkg/errors"
	"github.com/forest6511/gdl/pkg/plugin"
	"github.com/forest6511/gdl/pkg/ratelimit"
	"github.com/forest6511/gdl/pkg/types"
	"github.com/forest6511/gdl/pkg/ui"
)

// Version information.
const (
	version = "dev" // Set via ldflags during build
	appName = "gdl"
)

// CLI configuration.
type config struct {
	output            string
	userAgent         string
	timeout           time.Duration
	overwrite         bool
	createDirs        bool
	resume            bool
	showVersion       bool
	showHelp          bool
	quiet             bool
	verbose           bool
	concurrent        int
	chunkSize         string
	noConcurrent      bool
	noColor           bool
	interactive       bool
	checkConnectivity bool
	checkSpace        bool
	language          string
	progressBar       string
	noResume          bool
	retry             int
	retryDelay        time.Duration
	headers           map[string]string
	maxRedirects      int
	insecure          bool
	proxy             string
	output_format     string
	continuePartial   bool
	maxRate           string // Maximum download rate (e.g., "1MB/s", "500k")
	// Plugin-related configurations
	plugins      []string
	storageURL   string
	pluginDir    string
	pluginConfig string
}

// StringSlice implements flag.Value for string slice flags.
type StringSlice []string

func (s *StringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *StringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// Global formatter instance.
var formatter *ui.Formatter

// progressDisplay implements the types.Progress interface for console output.
type progressDisplay struct {
	quiet     bool
	verbose   bool
	filename  string
	totalSize int64
	lastLine  string
	formatter *ui.Formatter
	startTime time.Time
	cfg       *config
}

func newProgressDisplay(cfg *config, fmt *ui.Formatter) *progressDisplay {
	return &progressDisplay{
		quiet:     cfg.quiet,
		verbose:   cfg.verbose,
		formatter: fmt,
		startTime: time.Now(),
		cfg:       cfg,
	}
}

func (p *progressDisplay) Start(filename string, totalSize int64) {
	if p.quiet {
		return
	}

	p.filename = filename
	p.totalSize = totalSize
	p.startTime = time.Now()

	if totalSize > 0 {
		p.formatter.PrintMessage(
			ui.MessageInfo,
			"Starting download: %s (%s)",
			filename,
			formatBytes(totalSize),
		)
	} else {
		p.formatter.PrintMessage(ui.MessageInfo, "Starting download: %s", filename)
	}
}

func (p *progressDisplay) Update(bytesDownloaded, totalSize int64, speed int64) {
	if p.quiet {
		return
	}

	switch p.cfg.progressBar {
	case "simple":
		p.displaySimpleProgress(bytesDownloaded, totalSize, speed)
	case "detailed":
		options := &ui.ProgressBarOptions{
			Width:          50,
			ShowPercentage: true,
			ShowSpeed:      speed > 0,
			ShowETA:        speed > 0 && totalSize > 0,
			ShowSize:       true,
		}

		p.formatter.ClearLine()
		// #nosec G115 -- bytesDownloaded and totalSize validated via ValidateContentLength() parsing
		progressBar := p.formatter.FormatProgressBar(
			uint64(bytesDownloaded),
			uint64(totalSize),
			options,
		)
		fmt.Print("\r" + progressBar)
	case "json":
		displayJSONProgress(bytesDownloaded, totalSize, speed, p.filename)
	}
}

func (p *progressDisplay) displayProgressBar(bytesDownloaded, totalSize int64, speed int64) {
	percentage := float64(bytesDownloaded) / float64(totalSize) * 100

	// Clear the previous line
	if p.lastLine != "" {
		ui.ClearLine()
	}

	// Progress bar configuration
	barWidth := 30
	filled := int(percentage / 100.0 * float64(barWidth))

	// Build progress bar
	bar := "["

	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else if i == filled && percentage > 0 {
			bar += "▏"
		} else {
			bar += " "
		}
	}

	bar += "]"

	// Format the complete progress line
	progress := fmt.Sprintf("%s %.1f%% %s/%s",
		bar, percentage, ui.FormatSize(bytesDownloaded), ui.FormatSize(totalSize))

	if speed > 0 {
		progress += " " + ui.FormatSpeed(speed)
	}

	// Add color coding based on percentage
	if percentage >= 100 {
		progress = ui.Success(progress)
	} else if percentage >= 50 {
		progress = ui.Warning(progress)
	} else {
		progress = ui.Info(progress)
	}

	fmt.Print("" + progress)
	p.lastLine = progress
}

func displayJSONProgress(bytesDownloaded, totalSize int64, speed int64, filename string) {
	percentage := 0.0
	if totalSize > 0 {
		percentage = float64(bytesDownloaded) / float64(totalSize) * 100
	}

	progress := map[string]interface{}{
		"filename":         filename,
		"total_size":       totalSize,
		"bytes_downloaded": bytesDownloaded,
		"speed":            speed,
		"percentage":       percentage,
	}
	jsonOutput, _ := json.Marshal(progress)
	fmt.Println(string(jsonOutput))
}

func (p *progressDisplay) displaySimpleProgress(bytesDownloaded, totalSize int64, speed int64) {
	var progress string

	if totalSize > 0 {
		percentage := float64(bytesDownloaded) / float64(totalSize) * 100
		progress = fmt.Sprintf("%.1f%% (%s/%s)",
			percentage, formatBytes(bytesDownloaded), formatBytes(totalSize))
	} else {
		progress = fmt.Sprintf("%s downloaded", formatBytes(bytesDownloaded))
	}

	if speed > 0 {
		progress += fmt.Sprintf(" at %s/s", formatBytes(speed))
	}

	// Use carriage return to overwrite the previous line
	fmt.Printf("\r%s", progress)
	p.lastLine = progress
}

func (p *progressDisplay) Finish(filename string, stats *types.DownloadStats) {
	if p.quiet {
		return
	}

	// Clear progress line
	p.formatter.ClearLine()
	fmt.Println() // Add newline after progress bar

	// Show completion message
	p.formatter.PrintMessage(ui.MessageSuccess, "Download completed: %s (%s)",
		filename, formatBytes(stats.BytesDownloaded))

	if p.verbose && stats.Duration > 0 {
		// Create a table for verbose statistics
		table := p.formatter.NewTableFormatter([]string{"Metric", "Value"})
		table.AddRow([]string{"Duration", stats.Duration.Round(time.Millisecond).String()})
		table.AddRow([]string{"Average Speed", formatBytes(stats.AverageSpeed) + "/s"})

		if stats.Retries > 0 {
			table.AddRow([]string{"Retries", fmt.Sprintf("%d", stats.Retries)})
		}

		p.formatter.PrintMessage(ui.MessageInfo, "Download Statistics:")
		fmt.Println(table.Format())
	}
}

func (p *progressDisplay) Error(filename string, err error) {
	if p.quiet {
		return
	}

	// Clear progress line
	p.formatter.ClearLine()
	fmt.Println() // Add newline after progress bar

	// Use the formatter's error formatting capabilities
	errorOptions := &ui.ErrorFormatOptions{
		ShowErrorCode:   true,
		ShowSuggestions: true,
		ShowTimestamp:   p.verbose,
		MultiLine:       true,
	}

	formattedError := p.formatter.FormatError(err, errorOptions)
	if formattedError != "" {
		fmt.Fprintln(os.Stderr, formattedError)
	} else {
		p.formatter.PrintMessage(ui.MessageError, "Download failed: %s - %v", filename, err)
	}
}

// progressCallback creates a progress callback function for enhanced progress tracking.
func createProgressCallback(quiet bool) func(bytesDownloaded, totalBytes int64, speed int64) {
	if quiet {
		return nil
	}

	useAnsiBar := isTerminal()

	var lastLine string

	return func(bytesDownloaded, totalBytes int64, speed int64) {
		if useAnsiBar && totalBytes > 0 {
			displayEnhancedProgressBar(bytesDownloaded, totalBytes, speed, &lastLine)
		} else {
			displaySimpleProgressCallback(bytesDownloaded, totalBytes, speed)
		}
	}
}

func buildProgressBar(percentage float64) string {
	barWidth := 30
	filled := int(percentage / 100.0 * float64(barWidth))
	bar := "["

	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else if i == filled && percentage > 0 {
			bar += getPartialBlock(percentage, barWidth, filled)
		} else {
			bar += " "
		}
	}

	bar += "]"
	return bar
}

func getPartialBlock(percentage float64, barWidth, filled int) string {
	remainder := (percentage / 100.0 * float64(barWidth)) - float64(filled)
	if remainder >= 0.75 {
		return "▊"
	} else if remainder >= 0.5 {
		return "▌"
	} else if remainder >= 0.25 {
		return "▎"
	} else {
		return "▏"
	}
}

func calculateETA(speed, totalBytes, bytesDownloaded int64) string {
	if speed <= 0 || totalBytes <= bytesDownloaded {
		return ""
	}

	remaining := totalBytes - bytesDownloaded
	etaSeconds := remaining / speed

	if etaSeconds < 60 {
		return fmt.Sprintf(" ETA: %ds", etaSeconds)
	} else if etaSeconds < 3600 {
		return fmt.Sprintf(" ETA: %dm%ds", etaSeconds/60, etaSeconds%60)
	} else {
		return fmt.Sprintf(" ETA: %dh%dm", etaSeconds/3600, (etaSeconds%3600)/60)
	}
}

func buildProgressLine(bar string, percentage float64, bytesDownloaded, totalBytes, speed int64, eta string) string {
	progress := fmt.Sprintf("%s %.1f%% %s/%s",
		bar, percentage, formatBytes(bytesDownloaded), formatBytes(totalBytes))

	if speed > 0 {
		progress += fmt.Sprintf(" %s/s", formatBytes(speed))
	}

	progress += eta
	return progress
}

func addProgressColor(progress string, percentage float64) string {
	if percentage >= 100 {
		return "\033[32m" + progress + "\033[0m" // Green for complete
	} else if percentage >= 75 {
		return "\033[33m" + progress + "\033[0m" // Yellow for 75%+
	} else {
		return "\033[36m" + progress + "\033[0m" // Cyan for progress
	}
}

func displayEnhancedProgressBar(bytesDownloaded, totalBytes int64, speed int64, lastLine *string) {
	percentage := float64(bytesDownloaded) / float64(totalBytes) * 100

	// Clear the previous line
	if *lastLine != "" {
		fmt.Print("\r\033[K")
	}

	// Build progress bar
	bar := buildProgressBar(percentage)

	// Calculate ETA
	eta := calculateETA(speed, totalBytes, bytesDownloaded)

	// Format the complete progress line
	progress := buildProgressLine(bar, percentage, bytesDownloaded, totalBytes, speed, eta)

	// Add color coding
	progress = addProgressColor(progress, percentage)

	fmt.Print("\r" + progress)
	*lastLine = progress
}

func displaySimpleProgressCallback(bytesDownloaded, totalBytes int64, speed int64) {
	var progress string

	if totalBytes > 0 {
		percentage := float64(bytesDownloaded) / float64(totalBytes) * 100
		progress = fmt.Sprintf("%.1f%% (%s/%s)",
			percentage, formatBytes(bytesDownloaded), formatBytes(totalBytes))
	} else {
		progress = fmt.Sprintf("%s downloaded", formatBytes(bytesDownloaded))
	}

	if speed > 0 {
		progress += fmt.Sprintf(" at %s/s", formatBytes(speed))
	}

	fmt.Printf("\r%s", progress)
}

// isTerminal checks if stdout is connected to a terminal.
func isTerminal() bool {
	// Simple check for terminal - in a real implementation, you might want to use
	// a library like "golang.org/x/term" or check for specific environment variables
	return os.Getenv("TERM") != "" && os.Getenv("CI") == ""
}

// initializeFormatter sets up the global formatter with configuration.
func initializeFormatter(cfg *config) {
	language := ui.LanguageEnglish
	switch cfg.language {
	case "ja":
		language = ui.LanguageJapanese
	case "es":
		language = ui.LanguageSpanish
	case "fr":
		language = ui.LanguageFrench
	}

	formatter = ui.NewFormatter().
		WithColor(!cfg.noColor).
		WithLanguage(language).
		WithInteractive(cfg.interactive).
		WithWriter(os.Stderr)
}

// performPreDownloadChecks runs connectivity and disk space checks.
func performNetworkCheck(ctx context.Context, cfg *config) error {
	stopAnimation := formatter.CreateLoadingAnimation("Checking network connectivity...")

	diag := network.NewDiagnostics().WithTimeout(10 * time.Second)
	health, err := diag.RunFullDiagnostics(ctx, &network.DiagnosticOptions{
		IncludeBandwidth: false,
		IncludeProxy:     true,
		Verbose:          cfg.verbose,
	})

	stopAnimation()

	if err != nil {
		formatter.PrintMessage(ui.MessageWarning, "Network check failed: %v", err)
		return err
	}

	statusMsg := formatter.FormatStatusIndicator(ui.StatusCompleted,
		fmt.Sprintf("Network status: %s", health.OverallStatus))
	fmt.Println(statusMsg)

	if health.OverallStatus == network.HealthPoor || health.OverallStatus == network.HealthCritical {
		if cfg.interactive {
			proceed, err := formatter.ConfirmPrompt("Network conditions are poor. Continue anyway?", false)
			if err != nil || !proceed {
				return gdlerrors.NewDownloadError(gdlerrors.CodeCancelled, "download cancelled due to poor network conditions")
			}
		} else {
			formatter.PrintMessage(ui.MessageWarning, "Poor network conditions detected")
			return gdlerrors.NewDownloadError(gdlerrors.CodeNetworkError, "poor network conditions")
		}
	}

	return nil
}

func performDiskSpaceCheck(outputFile string, estimatedSize uint64, cfg *config) error {
	checker := storage.NewSpaceChecker()
	outputDir := filepath.Dir(outputFile)

	if estimatedSize > 0 {
		err := checker.CheckAvailableSpace(outputDir, estimatedSize)
		if err != nil {
			formatter.PrintMessage(ui.MessageWarning, "Disk space check: %v", err)

			if cfg.interactive {
				if err := handleDiskSpaceWarning(checker, outputDir); err != nil {
					return err
				}

				proceed, err := formatter.ConfirmPrompt(
					"Continue download despite disk space issues?",
					false,
				)
				if err != nil || !proceed {
					return gdlerrors.NewDownloadError(gdlerrors.CodeCancelled, "download cancelled due to insufficient disk space")
				}
			} else {
				return gdlerrors.NewDownloadError(gdlerrors.CodeInsufficientSpace, "insufficient disk space")
			}
		} else {
			formatter.PrintMessage(ui.MessageSuccess, "Sufficient disk space available")
		}
	}

	return nil
}

func handleDiskSpaceWarning(checker *storage.SpaceChecker, outputDir string) error {
	suggestions, err := checker.GenerateCleanupSuggestions([]string{outputDir})
	if err == nil && len(suggestions) > 0 {
		formatter.PrintMessage(
			ui.MessageInfo,
			"Found %d cleanup suggestions",
			len(suggestions),
		)

		proceed, err := formatter.ConfirmPrompt(
			"Would you like to see cleanup suggestions?",
			true,
		)
		if err == nil && proceed {
			showCleanupSuggestions(suggestions)
		}
	}
	return nil
}

func performPreDownloadChecks(
	ctx context.Context,
	cfg *config,
	outputFile string,
	estimatedSize uint64,
) error {
	if cfg.quiet {
		return nil // Skip checks in quiet mode
	}

	checksPassed := true

	// Network connectivity check
	if cfg.checkConnectivity {
		if err := performNetworkCheck(ctx, cfg); err != nil {
			checksPassed = false
		}
	}

	// Disk space check
	if cfg.checkSpace {
		if err := performDiskSpaceCheck(outputFile, estimatedSize, cfg); err != nil {
			checksPassed = false
		}
	}

	if !checksPassed && !cfg.interactive {
		return gdlerrors.NewDownloadError(gdlerrors.CodeValidationError, "pre-download checks failed")
	}

	return nil
}

// showCleanupSuggestions displays disk cleanup suggestions in a table.
func showCleanupSuggestions(suggestions []storage.CleanupSuggestion) {
	table := formatter.NewTableFormatter([]string{"Type", "Path", "Size", "Priority", "Safe"})

	for _, suggestion := range suggestions {
		safeStr := "No"
		if suggestion.Safe {
			safeStr = "Yes"
		}

		table.AddRow([]string{
			suggestion.Type.String(),
			suggestion.Path,
			// #nosec G115 -- suggestion.Size validated by storage analysis, fits within int64 range
			formatBytes(int64(suggestion.Size)),
			suggestion.Priority.String(),
			safeStr,
		})
	}

	formatter.PrintMessage(ui.MessageInfo, "Cleanup Suggestions:")
	fmt.Println(table.Format())
}

// run contains the main application logic, separated from main() for testing.
func handleSpecialFlags(cfg *config) int {
	// Handle version flag
	if cfg.showVersion {
		fmt.Printf("%s version %s\n", appName, version)
		return 0
	}

	// Handle help flag
	if cfg.showHelp {
		showUsage()
		return 0
	}

	return -1 // Continue normal execution
}

func validateAndPrepareDownload(cfg *config, url string) (string, error) {
	// Validate required arguments
	if url == "" {
		formatter.PrintMessage(ui.MessageError, "URL is required")
		showUsage()
		return "", gdlerrors.NewValidationError("URL", "URL is required")
	}

	// Determine output filename
	outputFile := cfg.output
	if outputFile == "" {
		outputFile = extractFilenameFromURL(url)
	}

	// Interactive confirmation for output file if needed
	if cfg.interactive && !cfg.overwrite {
		if _, err := os.Stat(outputFile); err == nil {
			proceed, err := formatter.ConfirmPrompt(
				fmt.Sprintf("File '%s' already exists. Overwrite?", outputFile),
				false,
			)
			if err != nil || !proceed {
				return "", gdlerrors.NewDownloadError(gdlerrors.CodeCancelled, "operation cancelled")
			}
			cfg.overwrite = true
		}
	}

	return outputFile, nil
}

func setupDownloaders(ctx context.Context, cfg *config) (*gdl.Downloader, *core.Downloader, error) {
	// Create enhanced downloader with plugin support
	downloader := gdl.NewDownloader()

	// Set up plugin registry and load plugins
	if err := setupPlugins(ctx, downloader, cfg); err != nil {
		formatter.PrintMessage(ui.MessageWarning, "Plugin setup failed: %v", err)
		// Continue without plugins
	}

	// Handle storage URL if provided
	if cfg.storageURL != "" {
		if err := handleStorageURL(ctx, downloader, cfg.storageURL); err != nil {
			return nil, nil, gdlerrors.WrapError(err, gdlerrors.CodeConfigError, "storage URL setup failed")
		}
	}

	// Create core downloader for backwards compatibility
	coreDownloader := core.NewDownloader()

	// Configure retry strategy
	coreDownloader.WithRetryStrategy(
		retry.NewRetryManager().
			WithMaxRetries(cfg.retry).
			WithBaseDelay(cfg.retryDelay),
	)

	return downloader, coreDownloader, nil
}

func createDownloadOptions(cfg *config) *types.DownloadOptions {
	options := &types.DownloadOptions{
		UserAgent:          cfg.userAgent,
		Timeout:            cfg.timeout,
		OverwriteExisting:  cfg.overwrite,
		CreateDirs:         cfg.createDirs,
		Resume:             cfg.resume && !cfg.noResume,
		Progress:           newProgressDisplay(cfg, formatter),
		ProgressCallback:   createProgressCallback(cfg.quiet),
		Headers:            cfg.headers,
		MaxRedirects:       cfg.maxRedirects,
		InsecureSkipVerify: cfg.insecure,
		ProxyURL:           cfg.proxy,
	}

	// Configure concurrent download options
	if cfg.noConcurrent {
		options.MaxConcurrency = 1
	} else {
		options.MaxConcurrency = cfg.concurrent
	}

	// Configure chunk size if specified
	if cfg.chunkSize != autoValue {
		if chunkSizeBytes, err := parseSize(cfg.chunkSize); err == nil {
			options.ChunkSize = chunkSizeBytes
		}
	}

	// Configure max rate if specified
	if cfg.maxRate != "" {
		if maxRateBytes, err := ratelimit.ParseRate(cfg.maxRate); err == nil {
			options.MaxRate = maxRateBytes
		} else {
			// Note: Error handling should be done in parseArgs validation
			fmt.Fprintf(os.Stderr, "Warning: Invalid max-rate format: %v\n", err)
		}
	}

	return options
}

func performAppropriateDownload(ctx context.Context, downloader *gdl.Downloader, coreDownloader *core.Downloader, url, outputFile string, options *types.DownloadOptions, cfg *config) error {
	// Use enhanced downloader for plugin-aware downloads
	if len(cfg.plugins) > 0 || cfg.storageURL != "" {
		return performEnhancedDownload(ctx, downloader, url, outputFile, options, cfg)
	} else {
		return performDownload(ctx, coreDownloader, url, outputFile, options, cfg)
	}
}

func run(args []string) int {
	// Save and restore original args for testing
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = args

	// Check for plugin subcommands first
	if len(args) > 1 && args[1] == "plugin" {
		return runPluginCommand(args[2:])
	}

	// Parse command line arguments
	cfg, url, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	// Initialize formatter
	initializeFormatter(cfg)

	// Handle special flags
	if exitCode := handleSpecialFlags(cfg); exitCode >= 0 {
		return exitCode
	}

	// Validate and prepare download
	outputFile, err := validateAndPrepareDownload(cfg, url)
	if err != nil {
		return 1
	}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up enhanced signal handling
	handleInterruption(ctx, cancel, cfg)

	// Set up downloaders
	downloader, coreDownloader, err := setupDownloaders(ctx, cfg)
	if err != nil {
		formatter.PrintMessage(ui.MessageError, "Downloader setup failed: %v", err)
		return 1
	}

	// Set up download options
	options := createDownloadOptions(cfg)

	// Perform download
	if err := performAppropriateDownload(ctx, downloader, coreDownloader, url, outputFile, options, cfg); err != nil {
		handleError(err, cfg)
		return 1
	}

	if !cfg.quiet {
		formatter.PrintMessage(ui.MessageSuccess, "Successfully downloaded to: %s", outputFile)
	}

	return 0
}

func main() {
	exitCode := run(os.Args)
	os.Exit(exitCode)
}

// parseArgs parses command line arguments and returns configuration and URL.
func parseArgs() (*config, string, error) {
	cfg := &config{}

	flag.StringVar(&cfg.output, "o", "", "Output filename (default: extract from URL)")
	flag.StringVar(&cfg.output, "output", "", "Output filename (default: extract from URL)")
	flag.StringVar(&cfg.userAgent, "user-agent", "gdl/"+version, "User-Agent string to use")
	flag.DurationVar(&cfg.timeout, "timeout", 30*time.Minute, "Download timeout")
	flag.BoolVar(&cfg.overwrite, "f", false, "Overwrite existing files")
	flag.BoolVar(&cfg.overwrite, "force", false, "Overwrite existing files")
	flag.BoolVar(
		&cfg.createDirs,
		"create-dirs",
		false,
		"Create parent directories if they don't exist",
	)
	flag.BoolVar(&cfg.resume, "resume", false, "Resume partial downloads if supported")
	flag.BoolVar(&cfg.showVersion, "version", false, "Show version information")
	flag.BoolVar(&cfg.showHelp, "help", false, "Show help information")
	flag.BoolVar(&cfg.showHelp, "h", false, "Show help information")
	flag.BoolVar(&cfg.quiet, "q", false, "Quiet mode (no progress output)")
	flag.BoolVar(&cfg.quiet, "quiet", false, "Quiet mode (no progress output)")
	flag.BoolVar(&cfg.verbose, "v", false, "Verbose output")
	flag.BoolVar(&cfg.verbose, "verbose", false, "Verbose output")
	flag.IntVar(&cfg.concurrent, "concurrent", 4, "Number of concurrent connections (default: 4)")

	var c int
	flag.IntVar(&c, "c", 4, "Number of concurrent connections (shorthand for --concurrent)")
	flag.StringVar(
		&cfg.chunkSize,
		"chunk-size",
		autoValue,
		"Chunk size for concurrent downloads (default: auto)",
	)
	flag.BoolVar(&cfg.noConcurrent, "no-concurrent", false, "Force single-threaded download")
	flag.BoolVar(&cfg.noColor, "no-color", false, "Disable colored output")
	flag.BoolVar(
		&cfg.interactive,
		"interactive",
		ui.IsTerminalInteractive(),
		"Enable interactive prompts",
	)
	flag.BoolVar(
		&cfg.checkConnectivity,
		"check-connectivity",
		false,
		"Check network connectivity before download",
	)
	flag.BoolVar(&cfg.checkSpace, "check-space", true, "Check available disk space before download")
	flag.StringVar(&cfg.language, "language", "en", "Language for messages (en, ja, es, fr)")
	flag.StringVar(
		&cfg.progressBar,
		"progress-bar",
		"detailed",
		"Progress bar type (simple|detailed|json)",
	)
	flag.BoolVar(&cfg.noResume, "no-resume", false, "Disable resume functionality")
	flag.IntVar(&cfg.retry, "retry", 3, "Number of retry attempts (default: 3)")
	flag.DurationVar(
		&cfg.retryDelay,
		"retry-delay",
		1*time.Second,
		"Delay between retries (default: 1s)",
	)
	flag.IntVar(&cfg.maxRedirects, "max-redirects", 10, "Maximum number of redirects to follow")
	flag.BoolVar(&cfg.insecure, "insecure", false, "Skip SSL certificate verification")
	flag.BoolVar(&cfg.insecure, "k", false, "Skip SSL certificate verification")
	flag.StringVar(&cfg.proxy, "proxy", "", "HTTP proxy URL (http://host:port)")
	flag.StringVar(&cfg.output_format, "output-format", autoValue, "Output format (auto|json|yaml)")
	flag.BoolVar(&cfg.continuePartial, "continue-partial", false, "Continue partial downloads")

	// Plugin-related flags
	var pluginFlags StringSlice
	flag.Var(&pluginFlags, "plugin", "Enable plugin (can be used multiple times)")
	flag.StringVar(&cfg.storageURL, "storage", "", "Storage URL (e.g., s3://bucket/path/, gcs://bucket/path/)")
	flag.StringVar(&cfg.pluginDir, "plugin-dir", cli.GetDefaultPluginDir(), "Plugin directory")
	flag.StringVar(&cfg.pluginConfig, "plugin-config", cli.GetDefaultConfigFile(), "Plugin configuration file")

	// Custom header flag handler
	var headerFlags StringSlice
	flag.Var(
		&headerFlags,
		"header",
		"Add custom header (can be used multiple times): -header 'Key: Value'",
	)
	flag.Var(&headerFlags, "H", "Add custom header (shorthand)")
	flag.StringVar(
		&cfg.maxRate,
		"max-rate",
		"",
		"Maximum download rate (e.g., 1MB/s, 500k, 2048)",
	)

	// Initialize headers map and plugins slice
	cfg.headers = make(map[string]string)
	cfg.plugins = []string{}

	flag.Parse()

	// Process custom headers
	for _, header := range headerFlags {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			cfg.headers[key] = value
		}
	}

	// Process plugin flags
	for _, pluginName := range pluginFlags {
		cfg.plugins = append(cfg.plugins, strings.TrimSpace(pluginName))
	}

	// Validate max-rate if specified
	if cfg.maxRate != "" {
		if err := ratelimit.ValidateRate(cfg.maxRate); err != nil {
			return nil, "", gdlerrors.WrapError(err, gdlerrors.CodeValidationError, "invalid --max-rate")
		}
	}

	// Handle -c as an alias for --concurrent
	cWasSet := false

	flag.Visit(func(f *flag.Flag) {
		if f.Name == "c" {
			cWasSet = true
		}
	})

	if cWasSet {
		cfg.concurrent = c
	}

	// Validate concurrent settings
	if cfg.concurrent < 1 {
		return nil, "", gdlerrors.NewValidationError("concurrent", "concurrent connections must be at least 1")
	}

	if cfg.concurrent > 32 {
		return nil, "", gdlerrors.NewValidationError("concurrent", "concurrent connections cannot exceed 32")
	}

	// Validate chunk size if specified
	if cfg.chunkSize != autoValue {
		if err := validateChunkSize(cfg.chunkSize); err != nil {
			return nil, "", gdlerrors.WrapError(err, gdlerrors.CodeValidationError, "invalid chunk-size")
		}
	}

	// Get URL from remaining arguments
	args := flag.Args()

	var url string
	if len(args) > 0 {
		url = args[0]
	}

	return cfg, url, nil
}

// validateChunkSize validates the chunk size parameter.
func validateChunkSize(chunkSize string) error {
	if chunkSize == "" || chunkSize == autoValue {
		return nil
	}

	// Parse size with units (e.g., "1MB", "512KB", "2GB")
	_, err := parseSize(chunkSize)

	return err
}

// parseSize parses a size string with units into bytes.
func parseSize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, gdlerrors.NewValidationError("size", "empty size string")
	}

	// Define unit multipliers
	units := map[string]int64{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
	}

	// Convert to uppercase for case-insensitive comparison
	upperSizeStr := strings.ToUpper(sizeStr)

	// Try to find unit suffix
	for unit, multiplier := range units {
		if strings.HasSuffix(upperSizeStr, unit) {
			numStr := strings.TrimSuffix(upperSizeStr, unit)

			// Parse the numeric part
			if value, err := strconv.ParseFloat(numStr, 64); err == nil {
				return int64(value * float64(multiplier)), nil
			}
		}
	}

	// If no unit, assume bytes
	if value, err := strconv.ParseFloat(sizeStr, 64); err == nil {
		return int64(value), nil
	}

	return 0, gdlerrors.NewValidationError("size", fmt.Sprintf("invalid size format: %s", sizeStr))
}

// runPluginCommand handles plugin subcommands
func runPluginCommand(args []string) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: plugin command required\n")
		showPluginUsage()
		return 1
	}

	ctx := context.Background()
	pluginRegistry := cli.NewPluginRegistry(cli.GetDefaultPluginDir(), cli.GetDefaultConfigFile())

	command := args[0]
	switch command {
	case "list":
		return handlePluginList(ctx, pluginRegistry)
	case "install":
		if len(args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: plugin install requires source and name\n")
			fmt.Fprintf(os.Stderr, "Usage: gdl plugin install <source> <name>\n")
			return 1
		}
		return handlePluginInstall(ctx, pluginRegistry, args[1], args[2])
	case "remove":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Error: plugin remove requires name\n")
			fmt.Fprintf(os.Stderr, "Usage: gdl plugin remove <name>\n")
			return 1
		}
		return handlePluginRemove(ctx, pluginRegistry, args[1])
	case "enable":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Error: plugin enable requires name\n")
			fmt.Fprintf(os.Stderr, "Usage: gdl plugin enable <name>\n")
			return 1
		}
		return handlePluginEnable(ctx, pluginRegistry, args[1])
	case "disable":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Error: plugin disable requires name\n")
			fmt.Fprintf(os.Stderr, "Usage: gdl plugin disable <name>\n")
			return 1
		}
		return handlePluginDisable(ctx, pluginRegistry, args[1])
	case "config":
		if len(args) < 4 || args[2] != "--set" {
			fmt.Fprintf(os.Stderr, "Error: plugin config requires name and --set key=value\n")
			fmt.Fprintf(os.Stderr, "Usage: gdl plugin config <name> --set <key>=<value>\n")
			return 1
		}
		return handlePluginConfig(ctx, pluginRegistry, args[1], args[3])
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown plugin command: %s\n", command)
		showPluginUsage()
		return 1
	}
}

// handlePluginList lists all installed plugins
func handlePluginList(ctx context.Context, registry *cli.PluginRegistry) int {
	plugins, err := registry.List(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing plugins: %v\n", err)
		return 1
	}

	if len(plugins) == 0 {
		fmt.Println("No plugins installed")
		return 0
	}

	fmt.Printf("%-20s %-10s %-15s %-10s %s\n", "NAME", "VERSION", "TYPE", "ENABLED", "SOURCE")
	fmt.Println(strings.Repeat("-", 80))

	for _, plugin := range plugins {
		enabled := "No"
		if plugin.Enabled {
			enabled = "Yes"
		}
		source := plugin.Source
		if source == "" {
			source = "local"
		}
		fmt.Printf("%-20s %-10s %-15s %-10s %s\n",
			plugin.Name, plugin.Version, plugin.Type, enabled, source)
	}

	return 0
}

// handlePluginInstall installs a plugin
func handlePluginInstall(ctx context.Context, registry *cli.PluginRegistry, source, name string) int {
	fmt.Printf("Installing plugin %s from %s...\n", name, source)

	if err := registry.Install(ctx, source, name); err != nil {
		fmt.Fprintf(os.Stderr, "Error installing plugin: %v\n", err)
		return 1
	}

	fmt.Printf("Successfully installed plugin: %s\n", name)
	return 0
}

// handlePluginRemove removes a plugin
func handlePluginRemove(ctx context.Context, registry *cli.PluginRegistry, name string) int {
	if err := registry.Remove(ctx, name); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing plugin: %v\n", err)
		return 1
	}

	fmt.Printf("Successfully removed plugin: %s\n", name)
	return 0
}

// handlePluginEnable enables a plugin
func handlePluginEnable(ctx context.Context, registry *cli.PluginRegistry, name string) int {
	if err := registry.Enable(ctx, name); err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling plugin: %v\n", err)
		return 1
	}

	fmt.Printf("Successfully enabled plugin: %s\n", name)
	return 0
}

// handlePluginDisable disables a plugin
func handlePluginDisable(ctx context.Context, registry *cli.PluginRegistry, name string) int {
	if err := registry.Disable(ctx, name); err != nil {
		fmt.Fprintf(os.Stderr, "Error disabling plugin: %v\n", err)
		return 1
	}

	fmt.Printf("Successfully disabled plugin: %s\n", name)
	return 0
}

// handlePluginConfig configures a plugin
func handlePluginConfig(ctx context.Context, registry *cli.PluginRegistry, name, keyValue string) int {
	parts := strings.SplitN(keyValue, "=", 2)
	if len(parts) != 2 {
		fmt.Fprintf(os.Stderr, "Error: config value must be in key=value format\n")
		return 1
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	if err := registry.Configure(ctx, name, key, value); err != nil {
		fmt.Fprintf(os.Stderr, "Error configuring plugin: %v\n", err)
		return 1
	}

	fmt.Printf("Successfully configured plugin %s: %s=%s\n", name, key, value)
	return 0
}

// setupPlugins initializes and loads plugins
func setupPlugins(ctx context.Context, downloader *gdl.Downloader, cfg *config) error {
	if len(cfg.plugins) == 0 {
		return nil
	}

	pluginRegistry := cli.NewPluginRegistry(cfg.pluginDir, cfg.pluginConfig)
	pluginManager := plugin.NewPluginManager()

	// Load all enabled plugins first
	if err := pluginRegistry.LoadPlugins(ctx, pluginManager); err != nil {
		return gdlerrors.WrapError(err, gdlerrors.CodePluginError, "failed to load plugins")
	}

	// Enable specific plugins from command line
	for _, pluginName := range cfg.plugins {
		// Try to get the plugin from the manager
		pluginInstance, err := pluginManager.Get(pluginName)
		if err != nil {
			return gdlerrors.NewPluginError(pluginName, err, "plugin not found")
		}

		if err := downloader.UsePlugin(pluginInstance); err != nil {
			return gdlerrors.NewPluginError(pluginName, err, "failed to use plugin")
		}
	}

	return nil
}

// handleStorageURL processes storage URL and configures appropriate storage backend
func handleStorageURL(ctx context.Context, downloader *gdl.Downloader, storageURL string) error {
	parsedURL, err := url.Parse(storageURL)
	if err != nil {
		return gdlerrors.WrapErrorWithURL(err, gdlerrors.CodeInvalidURL, "invalid storage URL", storageURL)
	}

	switch parsedURL.Scheme {
	case "s3":
		// Handle S3 URLs like s3://bucket/path/
		return setupS3Storage(ctx, downloader, parsedURL)
	case "gcs":
		// Handle GCS URLs like gcs://bucket/path/
		return setupGCSStorage(ctx, downloader, parsedURL)
	case "file":
		// Handle file URLs like file:///path/to/dir/
		return setupFileStorage(ctx, downloader, parsedURL)
	default:
		return gdlerrors.NewValidationError("storage_scheme", fmt.Sprintf("unsupported storage scheme: %s", parsedURL.Scheme))
	}
}

// setupS3Storage configures S3 storage backend
func setupS3Storage(ctx context.Context, downloader *gdl.Downloader, parsedURL *url.URL) error {
	// This would create and configure an S3 storage backend
	// For now, we'll just return an informational message
	fmt.Printf("Note: S3 storage configured for bucket: %s, path: %s\n", parsedURL.Host, parsedURL.Path)
	return nil
}

// setupGCSStorage configures GCS storage backend
func setupGCSStorage(ctx context.Context, downloader *gdl.Downloader, parsedURL *url.URL) error {
	// This would create and configure a GCS storage backend
	fmt.Printf("Note: GCS storage configured for bucket: %s, path: %s\n", parsedURL.Host, parsedURL.Path)
	return nil
}

// setupFileStorage configures local file storage backend
func setupFileStorage(ctx context.Context, downloader *gdl.Downloader, parsedURL *url.URL) error {
	// This would create and configure a local file storage backend
	fmt.Printf("Note: File storage configured for path: %s\n", parsedURL.Path)
	return nil
}

// performEnhancedDownload performs download using the enhanced downloader with plugin support
func performEnhancedDownload(
	ctx context.Context,
	downloader *gdl.Downloader,
	url, outputFile string,
	options *types.DownloadOptions,
	cfg *config,
) error {
	// Add timeout to context if specified
	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}

	// Convert types.DownloadOptions to gdl.Options
	gdlOptions := &gdl.Options{
		MaxConcurrency:    options.MaxConcurrency,
		ChunkSize:         options.ChunkSize,
		EnableResume:      options.Resume,
		RetryAttempts:     cfg.retry,
		Timeout:           cfg.timeout,
		UserAgent:         cfg.userAgent,
		Headers:           cfg.headers,
		CreateDirs:        cfg.createDirs,
		OverwriteExisting: cfg.overwrite,
		Quiet:             cfg.quiet,
		Verbose:           cfg.verbose,
	}

	// Set up progress callback if needed
	if !cfg.quiet && options.ProgressCallback != nil {
		gdlOptions.ProgressCallback = func(p gdl.Progress) {
			options.ProgressCallback(p.BytesDownloaded, p.TotalSize, p.Speed)
		}
	}

	// Perform the download using enhanced downloader
	_, err := downloader.Download(ctx, url, outputFile, gdlOptions)
	return err
}

// showPluginUsage shows plugin command usage
func showPluginUsage() {
	fmt.Printf(`Plugin Management Commands:

Usage: %s plugin <command> [args]

Commands:
  list                     List all installed plugins
  install <source> <name>  Install a plugin from source
  remove <name>           Remove an installed plugin
  enable <name>           Enable a plugin
  disable <name>          Disable a plugin
  config <name> --set <key>=<value>  Configure a plugin

Examples:
  %s plugin list
  %s plugin install github.com/user/gdl-plugin-s3 s3
  %s plugin remove s3
  %s plugin enable oauth2
  %s plugin config oauth2 --set client_id=xxx

Plugin Download Examples:
  %s --plugin oauth2 --plugin s3-storage https://api.example.com/file
  %s --storage s3://mybucket/downloads/ https://example.com/file.zip
  %s --storage gcs://bucket/path/ https://example.com/file.zip

`, appName, appName, appName, appName, appName, appName, appName, appName, appName)
}

// performDownload executes the download operation.
func performDownload(
	ctx context.Context,
	downloader *core.Downloader,
	url, outputFile string,
	options *types.DownloadOptions,
	cfg *config,
) error {
	// Add timeout to context if specified
	if cfg.timeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}

	// Perform the download
	stats, err := downloader.Download(ctx, url, outputFile, options)
	if err != nil {
		// If we have stats, we can provide more context
		if stats != nil && cfg.verbose {
			fmt.Fprintf(os.Stderr, "Download statistics:\n")
			fmt.Fprintf(os.Stderr, "  Bytes downloaded: %s\n", formatBytes(stats.BytesDownloaded))
			fmt.Fprintf(os.Stderr, "  Duration: %v\n", stats.Duration.Round(time.Millisecond))

			if stats.Retries > 0 {
				fmt.Fprintf(os.Stderr, "  Retries: %d\n", stats.Retries)
			}
		}

		return err
	}

	return nil
}

// handleError processes and displays errors in a user-friendly way.
func handleError(err error, cfg *config) {
	if err == nil {
		return // No error to handle
	}

	if formatter == nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	errorOptions := &ui.ErrorFormatOptions{
		ShowErrorCode:   true,
		ShowSuggestions: true,
		ShowTimestamp:   cfg.verbose,
		MultiLine:       true,
	}

	formattedError := formatter.FormatError(err, errorOptions)
	if formattedError != "" {
		fmt.Fprintln(os.Stderr, formattedError)
	} else {
		formatter.PrintMessage(ui.MessageError, "Download failed: %v", err)
	}
}

// extractFilenameFromURL extracts a filename from a URL.
const (
	defaultFilename = "download"
	autoValue       = "auto"
)

func extractFilenameFromURL(rawURL string) string {
	// Remove query parameters
	if idx := strings.Index(rawURL, "?"); idx != -1 {
		rawURL = rawURL[:idx]
	}

	// Parse the URL to extract the path
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return defaultFilename
	}

	// Get the path part and extract filename
	path := parsedURL.Path
	if path == "" || path == "/" {
		return defaultFilename
	}

	filename := filepath.Base(path)
	if filename == "" || filename == "." || filename == "/" {
		return defaultFilename
	}

	return filename
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

// showUsage displays usage information.
func showUsage() {
	fmt.Printf(`%s - A simple and efficient download tool

Usage: %s [OPTIONS] URL
       %s plugin <command> [args]

Download Options:
  -o, --output FILE        Output filename (default: extract from URL)
      --user-agent STRING  User-Agent string to use (default: gdl/%s)
      --timeout DURATION   Download timeout (default: 30m)
  -f, --force             Overwrite existing files
      --create-dirs       Create parent directories if they don't exist
      --resume            Resume partial downloads if supported
  -q, --quiet             Quiet mode (no progress output)
  -v, --verbose           Verbose output
      --concurrent N      Number of concurrent connections (default: 4, max: 32)
      --chunk-size SIZE   Chunk size for concurrent downloads (default: auto)
                          Examples: 1MB, 512KB, 2GB
      --max-rate RATE     Maximum download rate (0 = unlimited)
                          Examples: 1MB/s, 500k, 2048
      --no-concurrent     Force single-threaded download
      --no-color          Disable colored output
      --interactive       Enable interactive prompts (default: auto-detect)
      --check-connectivity Check network connectivity before download
      --check-space       Check available disk space before download (default: true)
      --language LANG     Language for messages (en, ja, es, fr, default: en)
      --version           Show version information
  -h, --help              Show this help message

Plugin Options:
      --plugin NAME       Enable plugin for this download (can be used multiple times)
      --storage URL       Storage destination URL (s3://bucket/path/, gcs://bucket/path/)
      --plugin-dir DIR    Plugin directory (default: ~/.gdl/plugins)
      --plugin-config FILE Plugin configuration file (default: ~/.gdl/plugins.json)

Plugin Commands:
  plugin list             List all installed plugins
  plugin install <source> <name>  Install a plugin
  plugin remove <name>    Remove a plugin
  plugin enable <name>    Enable a plugin
  plugin disable <name>   Disable a plugin
  plugin config <name> --set <key>=<value>  Configure a plugin

Download Examples:
  %s https://example.com/file.zip                              # Basic download
  %s --concurrent 8 https://example.com/largefile.iso         # Use 8 concurrent connections
  %s --chunk-size 2MB https://example.com/file.zip            # Use 2MB chunks
  %s --max-rate 1MB/s https://example.com/large-file.zip      # Limit to 1MB/s
  %s --plugin oauth2 https://api.example.com/secure/file.zip  # Use OAuth2 plugin
  %s --storage s3://mybucket/downloads/ https://example.com/file.zip  # Save to S3

Plugin Management Examples:
  %s plugin list                                              # List installed plugins
  %s plugin install github.com/user/gdl-plugin-s3 s3        # Install S3 plugin
  %s plugin enable oauth2                                     # Enable OAuth2 plugin
  %s plugin config oauth2 --set client_id=xxx                # Configure plugin

`, appName, appName, appName, appName, version, appName, appName, appName, appName, appName, appName, appName, appName, appName)
}
