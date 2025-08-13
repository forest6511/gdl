// Package ui provides user interface formatting and interaction utilities.
package ui

import (
	"bufio"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/forest6511/godl/pkg/errors"
)

// Color constants for terminal output.
const (
	// ColorReset resets all colors and styles.
	ColorReset = "\033[0m"

	// ColorBlack is black color.
	ColorBlack = "\033[30m"

	// ColorRed is red color.
	ColorRed = "\033[31m"

	// ColorGreen is green color.
	ColorGreen = "\033[32m"

	// ColorYellow is yellow color.
	ColorYellow = "\033[33m"

	// ColorBlue is blue color.
	ColorBlue = "\033[34m"

	// ColorMagenta is magenta color.
	ColorMagenta = "\033[35m"

	// ColorCyan is cyan color.
	ColorCyan = "\033[36m"

	// ColorWhite is white color.
	ColorWhite = "\033[37m"
)

// Formatter provides comprehensive formatting capabilities for various output types.
type Formatter struct {
	colorEnabled bool
	writer       io.Writer
	language     Language
	interactive  bool
}

// Language represents supported languages.
type Language string

const (
	// English language.
	LanguageEnglish Language = "en"

	// Japanese language.
	LanguageJapanese Language = "ja"

	// Spanish language.
	LanguageSpanish Language = "es"

	// French language.
	LanguageFrench Language = "fr"
)

// MessageType represents different types of messages.
type MessageType int

const (
	// MessageInfo for informational messages.
	MessageInfo MessageType = iota

	// MessageSuccess for success messages.
	MessageSuccess

	// MessageWarning for warning messages.
	MessageWarning

	// MessageError for error messages.
	MessageError

	// MessageDebug for debug messages.
	MessageDebug

	// MessagePrompt for interactive prompts.
	MessagePrompt
)

// ProgressBarOptions configures progress bar appearance and behavior.
type ProgressBarOptions struct {
	Width           int           // Width of the progress bar
	ShowPercentage  bool          // Show percentage
	ShowSpeed       bool          // Show download speed
	ShowETA         bool          // Show estimated time remaining
	ShowSize        bool          // Show downloaded/total size
	RefreshInterval time.Duration // How often to refresh
	Template        string        // Custom template for progress display
}

// ErrorFormatOptions configures error message formatting.
type ErrorFormatOptions struct {
	ShowErrorCode   bool // Show error codes
	ShowSuggestions bool // Show recovery suggestions
	ShowTimestamp   bool // Show timestamp
	Compact         bool // Use compact format
	MultiLine       bool // Use multi-line format for complex errors
}

// StatusIndicator represents different status states.
type StatusIndicator int

const (
	// StatusPending for pending operations.
	StatusPending StatusIndicator = iota

	// StatusInProgress for ongoing operations.
	StatusInProgress

	// StatusCompleted for completed operations.
	StatusCompleted

	// StatusFailed for failed operations.
	StatusFailed

	// StatusPaused for paused operations.
	StatusPaused

	// StatusCancelled for cancelled operations.
	StatusCancelled
)

// IsColorSupported checks if the terminal supports color output.
func IsColorSupported() bool {
	term := os.Getenv("TERM")
	if term == "" {
		return false
	}

	// Check for common terminals that support color
	colorTerms := []string{"xterm", "xterm-color", "xterm-256color", "screen", "tmux", "linux"}
	for _, colorTerm := range colorTerms {
		if strings.Contains(term, colorTerm) {
			return true
		}
	}

	// Check for NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	return term != "dumb"
}

// IsTerminalInteractive checks if the current session is interactive.
func IsTerminalInteractive() bool {
	// Check if stdin, stdout, and stderr are terminals
	return os.Getenv("TERM") != "" && os.Getenv("CI") == ""
}

// colorize applies color formatting to text if colors are enabled.
func (f *Formatter) colorize(color, text string) string {
	if !f.colorEnabled {
		return text
	}

	return color + text + ColorReset
}

// NewFormatter creates a new formatter with default settings.
func NewFormatter() *Formatter {
	return &Formatter{
		colorEnabled: IsColorSupported(),
		writer:       os.Stdout,
		language:     LanguageEnglish,
		interactive:  IsTerminalInteractive(),
	}
}

// WithColor enables or disables color output.
func (f *Formatter) WithColor(enabled bool) *Formatter {
	f.colorEnabled = enabled
	return f
}

// WithWriter sets the output writer.
func (f *Formatter) WithWriter(w io.Writer) *Formatter {
	f.writer = w
	return f
}

// WithLanguage sets the language for localized messages.
func (f *Formatter) WithLanguage(lang Language) *Formatter {
	f.language = lang
	return f
}

// WithInteractive enables or disables interactive mode.
func (f *Formatter) WithInteractive(interactive bool) *Formatter {
	f.interactive = interactive
	return f
}

// FormatMessage formats a message with appropriate styling based on type.
func (f *Formatter) FormatMessage(msgType MessageType, format string, args ...interface{}) string {
	message := fmt.Sprintf(format, args...)

	if !f.colorEnabled {
		return f.addPrefix(msgType, message)
	}

	switch msgType {
	case MessageInfo:
		return f.colorize(ColorBlue, f.addPrefix(msgType, message))
	case MessageSuccess:
		return f.colorize(ColorGreen, f.addPrefix(msgType, message))
	case MessageWarning:
		return f.colorize(ColorYellow, f.addPrefix(msgType, message))
	case MessageError:
		return f.colorize(ColorRed, f.addPrefix(msgType, message))
	case MessageDebug:
		return f.colorize(ColorCyan, f.addPrefix(msgType, message))
	case MessagePrompt:
		return f.colorize(ColorMagenta, f.addPrefix(msgType, message))
	default:
		return message
	}
}

// PrintMessage prints a formatted message to the output writer.
func (f *Formatter) PrintMessage(msgType MessageType, format string, args ...interface{}) {
	formatted := f.FormatMessage(msgType, format, args...)
	_, _ = fmt.Fprintln(f.writer, formatted)
}

// FormatError formats an error message with optional error details.
func (f *Formatter) FormatError(err error, options *ErrorFormatOptions) string {
	if err == nil {
		return ""
	}

	if options == nil {
		options = &ErrorFormatOptions{
			ShowErrorCode:   true,
			ShowSuggestions: true,
			ShowTimestamp:   false,
			Compact:         false,
			MultiLine:       true,
		}
	}

	var parts []string

	// Add timestamp if requested
	if options.ShowTimestamp {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		parts = append(parts, f.localize("timestamp")+": "+timestamp)
	}

	// Handle DownloadError with additional details
	downloadErr := &errors.DownloadError{}
	if stderrors.As(err, &downloadErr) {
		return f.formatDownloadError(downloadErr, options)
	}

	// Basic error formatting
	if options.Compact {
		return f.colorize(ColorRed, "✗ "+err.Error())
	}

	parts = append(parts, f.colorize(ColorRed, "✗ "+f.localize("error")+": "+err.Error()))

	if options.MultiLine {
		return strings.Join(parts, "\n")
	}

	return strings.Join(parts, " | ")
}

// formatDownloadError formats DownloadError with full details.
func (f *Formatter) formatDownloadError(
	err *errors.DownloadError,
	options *ErrorFormatOptions,
) string {
	var parts []string

	// Add timestamp if requested
	if options.ShowTimestamp {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		parts = append(parts, f.localize("timestamp")+": "+timestamp)
	}

	// Main error message
	errorIcon := "✗"
	mainMsg := fmt.Sprintf("%s %s: %s", errorIcon, f.localize("error"), err.Message)
	parts = append(parts, f.colorize(ColorRed, mainMsg))

	// Add error code if requested
	if options.ShowErrorCode {
		codeMsg := fmt.Sprintf("%s: %s", f.localize("error_code"), err.Code.String())
		parts = append(parts, f.colorize(ColorYellow, codeMsg))
	}

	// Add additional details
	if err.Details != "" {
		detailsMsg := fmt.Sprintf("%s: %s", f.localize("details"), err.Details)
		parts = append(parts, f.colorize(ColorCyan, detailsMsg))
	}

	// Add URL context if available
	if err.URL != "" {
		urlMsg := fmt.Sprintf("%s: %s", f.localize("url"), err.URL)
		parts = append(parts, f.colorize(ColorBlue, urlMsg))
	}

	// Add filename context if available
	if err.Filename != "" {
		filenameMsg := fmt.Sprintf("%s: %s", f.localize("filename"), err.Filename)
		parts = append(parts, f.colorize(ColorBlue, filenameMsg))
	}

	// Add HTTP status code if available
	if err.HTTPStatusCode != 0 {
		statusMsg := fmt.Sprintf("%s: %d", f.localize("http_status"), err.HTTPStatusCode)
		parts = append(parts, f.colorize(ColorMagenta, statusMsg))
	}

	if options.MultiLine {
		return strings.Join(parts, "\n")
	}

	return strings.Join(parts, " | ")
}

// FormatProgressBar creates a formatted progress bar.
func (f *Formatter) FormatProgressBar(current, total uint64, options *ProgressBarOptions) string {
	if options == nil {
		options = &ProgressBarOptions{
			Width:          50,
			ShowPercentage: true,
			ShowSpeed:      true,
			ShowETA:        true,
			ShowSize:       true,
		}
	}

	if options.Template != "" {
		return f.formatProgressWithTemplate(current, total, options)
	}

	return f.formatDefaultProgressBar(current, total, options)
}

// formatDefaultProgressBar creates the default progress bar format.
func (f *Formatter) formatDefaultProgressBar(
	current, total uint64,
	options *ProgressBarOptions,
) string {
	var parts []string

	// Calculate percentage
	percentage := float64(0)
	if total > 0 {
		percentage = float64(current) / float64(total) * 100
	}

	// Create the progress bar
	filled := int(float64(options.Width) * percentage / 100)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", options.Width-filled)

	if f.colorEnabled {
		filledBar := f.colorize(ColorGreen, strings.Repeat("█", filled))
		emptyBar := f.colorize(ColorWhite, strings.Repeat("░", options.Width-filled))
		bar = filledBar + emptyBar
	}

	parts = append(parts, "["+bar+"]")

	// Add percentage
	if options.ShowPercentage {
		percentageStr := fmt.Sprintf("%.1f%%", percentage)
		if f.colorEnabled {
			percentageStr = f.colorize(ColorCyan, percentageStr)
		}

		parts = append(parts, percentageStr)
	}

	// Add size information
	if options.ShowSize {
		sizeStr := fmt.Sprintf("%s/%s", formatBytes(current), formatBytes(total))
		if f.colorEnabled {
			sizeStr = f.colorize(ColorBlue, sizeStr)
		}

		parts = append(parts, sizeStr)
	}

	return strings.Join(parts, " ")
}

// formatProgressWithTemplate formats progress using a custom template.
func (f *Formatter) formatProgressWithTemplate(
	current, total uint64,
	options *ProgressBarOptions,
) string {
	template := options.Template

	// Replace template variables
	template = strings.ReplaceAll(template, "{current}", formatBytes(current))
	template = strings.ReplaceAll(template, "{total}", formatBytes(total))
	template = strings.ReplaceAll(
		template,
		"{percentage}",
		fmt.Sprintf("%.1f", float64(current)/float64(total)*100),
	)

	// Create progress bar
	percentage := float64(current) / float64(total) * 100
	filled := int(float64(options.Width) * percentage / 100)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", options.Width-filled)
	template = strings.ReplaceAll(template, "{bar}", bar)

	return template
}

// FormatStatusIndicator formats a status indicator with appropriate styling.
func (f *Formatter) FormatStatusIndicator(status StatusIndicator, message string) string {
	var icon, color string

	switch status {
	case StatusPending:
		icon, color = "○", ColorYellow
	case StatusInProgress:
		icon, color = "●", ColorBlue
	case StatusCompleted:
		icon, color = "✓", ColorGreen
	case StatusFailed:
		icon, color = "✗", ColorRed
	case StatusPaused:
		icon, color = "⏸", ColorYellow
	case StatusCancelled:
		icon, color = "⏹", ColorMagenta
	default:
		icon, color = "?", ColorWhite
	}

	formatted := fmt.Sprintf("%s %s", icon, message)
	if f.colorEnabled {
		return f.colorize(color, formatted)
	}

	return formatted
}

// Prompt displays an interactive prompt and returns the user's response.
func (f *Formatter) Prompt(message string) (string, error) {
	if !f.interactive {
		return "", fmt.Errorf("interactive mode not available")
	}

	promptMsg := f.FormatMessage(MessagePrompt, "%s", message)
	_, _ = fmt.Fprint(f.writer, promptMsg+" ")

	reader := bufio.NewReader(os.Stdin)

	response, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(response), nil
}

// ConfirmPrompt displays a yes/no confirmation prompt.
func (f *Formatter) ConfirmPrompt(message string, defaultValue bool) (bool, error) {
	if !f.interactive {
		return defaultValue, nil
	}

	defaultStr := "y/N"
	if defaultValue {
		defaultStr = "Y/n"
	}

	promptMsg := fmt.Sprintf("%s [%s]", message, defaultStr)

	response, err := f.Prompt(promptMsg)
	if err != nil {
		return defaultValue, err
	}

	if response == "" {
		return defaultValue, nil
	}

	response = strings.ToLower(response)

	return response == "y" || response == "yes", nil
}

// SelectPrompt displays a selection prompt with multiple options.
func (f *Formatter) SelectPrompt(message string, options []string, defaultIndex int) (int, error) {
	if !f.interactive {
		return defaultIndex, nil
	}

	f.PrintMessage(MessagePrompt, "%s", message)

	for i, option := range options {
		marker := " "
		if i == defaultIndex {
			marker = "*"
		}

		f.PrintMessage(MessageInfo, "%s %d. %s", marker, i+1, option)
	}

	response, err := f.Prompt(
		fmt.Sprintf("Select option (1-%d, default: %d)", len(options), defaultIndex+1),
	)
	if err != nil {
		return defaultIndex, err
	}

	if response == "" {
		return defaultIndex, nil
	}

	index, err := strconv.Atoi(response)
	if err != nil || index < 1 || index > len(options) {
		return defaultIndex, fmt.Errorf("invalid selection")
	}

	return index - 1, nil
}

// addPrefix adds a prefix to messages based on type.
func (f *Formatter) addPrefix(msgType MessageType, message string) string {
	switch msgType {
	case MessageInfo:
		return f.localize("info") + ": " + message
	case MessageSuccess:
		return f.localize("success") + ": " + message
	case MessageWarning:
		return f.localize("warning") + ": " + message
	case MessageError:
		return f.localize("error") + ": " + message
	case MessageDebug:
		return f.localize("debug") + ": " + message
	case MessagePrompt:
		return f.localize("prompt") + ": " + message
	default:
		return message
	}
}

// localize returns localized text based on the current language.
func (f *Formatter) localize(key string) string {
	translations := map[Language]map[string]string{
		LanguageEnglish: {
			"info":        "INFO",
			"success":     "SUCCESS",
			"warning":     "WARNING",
			"error":       "ERROR",
			"debug":       "DEBUG",
			"prompt":      "PROMPT",
			"timestamp":   "Timestamp",
			"error_code":  "Error Code",
			"details":     "Details",
			"context":     "Context",
			"suggestions": "Suggested Actions",
			"url":         "URL",
			"filename":    "Filename",
			"http_status": "HTTP Status",
		},
		LanguageJapanese: {
			"info":        "情報",
			"success":     "成功",
			"warning":     "警告",
			"error":       "エラー",
			"debug":       "デバッグ",
			"prompt":      "プロンプト",
			"timestamp":   "タイムスタンプ",
			"error_code":  "エラーコード",
			"details":     "詳細",
			"context":     "コンテキスト",
			"suggestions": "推奨アクション",
			"url":         "URL",
			"filename":    "ファイル名",
			"http_status": "HTTPステータス",
		},
		LanguageSpanish: {
			"info":        "INFO",
			"success":     "ÉXITO",
			"warning":     "ADVERTENCIA",
			"error":       "ERROR",
			"debug":       "DEBUG",
			"prompt":      "PROMPT",
			"timestamp":   "Marca de tiempo",
			"error_code":  "Código de error",
			"details":     "Detalles",
			"context":     "Contexto",
			"suggestions": "Acciones sugeridas",
		},
		LanguageFrench: {
			"info":        "INFO",
			"success":     "SUCCÈS",
			"warning":     "AVERTISSEMENT",
			"error":       "ERREUR",
			"debug":       "DEBUG",
			"prompt":      "PROMPT",
			"timestamp":   "Horodatage",
			"error_code":  "Code d'erreur",
			"details":     "Détails",
			"context":     "Contexte",
			"suggestions": "Actions suggérées",
		},
	}

	if lang, exists := translations[f.language]; exists {
		if text, exists := lang[key]; exists {
			return text
		}
	}

	// Fallback to English
	if lang, exists := translations[LanguageEnglish]; exists {
		if text, exists := lang[key]; exists {
			return text
		}
	}

	// Final fallback
	return strings.ToUpper(key)
}

// Helper functions for formatting and utility

// formatBytes formats byte counts in human-readable format.
func formatBytes(bytes uint64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	value := float64(bytes)

	for _, unit := range units {
		value /= 1024
		if value < 1024 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}

	return fmt.Sprintf("%.1f EB", value/1024)
}

// CreateLoadingAnimation creates a simple loading animation.
func (f *Formatter) CreateLoadingAnimation(message string) func() {
	if !f.interactive {
		f.PrintMessage(MessageInfo, "%s", message)
		return func() {}
	}

	chars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	stop := make(chan bool)

	go func() {
		i := 0

		for {
			select {
			case <-stop:
				return
			default:
				char := chars[i%len(chars)]
				if f.colorEnabled {
					char = f.colorize(ColorCyan, char)
				}

				_, _ = fmt.Fprintf(f.writer, "\r%s %s", char, message)
				time.Sleep(100 * time.Millisecond)

				i++
			}
		}
	}()

	return func() {
		stop <- true

		_, _ = fmt.Fprintf(f.writer, "\r%s\r", strings.Repeat(" ", len(message)+5))
	}
}

// ClearLine clears the current line in the terminal.
func (f *Formatter) ClearLine() {
	if f.interactive {
		_, _ = fmt.Fprintf(f.writer, "\r%s\r", strings.Repeat(" ", 100))
	}
}

// TableFormatter provides table formatting capabilities.
type TableFormatter struct {
	Headers   []string
	Rows      [][]string
	formatter *Formatter
}

// NewTableFormatter creates a new table formatter.
func (f *Formatter) NewTableFormatter(headers []string) *TableFormatter {
	return &TableFormatter{
		Headers:   headers,
		Rows:      [][]string{},
		formatter: f,
	}
}

// AddRow adds a row to the table.
func (tf *TableFormatter) AddRow(row []string) {
	tf.Rows = append(tf.Rows, row)
}

// Format formats the table for display.
func (tf *TableFormatter) Format() string {
	if len(tf.Headers) == 0 && len(tf.Rows) == 0 {
		return ""
	}

	// Calculate column widths
	widths := make([]int, len(tf.Headers))
	for i, header := range tf.Headers {
		widths[i] = len(header)
	}

	for _, row := range tf.Rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var lines []string

	// Header
	if len(tf.Headers) > 0 {
		headerLine := tf.formatRow(tf.Headers, widths)
		if tf.formatter.colorEnabled {
			headerLine = tf.formatter.colorize(ColorCyan, headerLine)
		}

		lines = append(lines, headerLine)

		// Separator
		separatorParts := make([]string, len(widths))
		for i, width := range widths {
			separatorParts[i] = strings.Repeat("-", width)
		}

		lines = append(lines, tf.formatRow(separatorParts, widths))
	}

	// Rows
	for _, row := range tf.Rows {
		lines = append(lines, tf.formatRow(row, widths))
	}

	return strings.Join(lines, "\n")
}

// formatRow formats a single row with proper padding.
func (tf *TableFormatter) formatRow(row []string, widths []int) string {
	paddedCells := make([]string, len(row))
	for i, cell := range row {
		if i < len(widths) {
			paddedCells[i] = fmt.Sprintf("%-*s", widths[i], cell)
		} else {
			paddedCells[i] = cell
		}
	}

	return strings.Join(paddedCells, " | ")
}
