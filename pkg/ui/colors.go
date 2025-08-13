package ui

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Color codes.
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
	Italic = "\033[3m"
	Under  = "\033[4m"

	// Foreground colors.
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"

	// Bright foreground colors.
	BrightBlack   = "\033[90m"
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	BrightWhite   = "\033[97m"

	// Background colors.
	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
)

// ColorConfig manages color output settings.
type ColorConfig struct {
	Enabled bool
	Force   bool
}

var defaultColorConfig = &ColorConfig{
	Enabled: isTerminal() && supportsColor(),
	Force:   false,
}

// isTerminal checks if output is a terminal.
func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// supportsColor checks if the terminal supports colors.
func supportsColor() bool {
	// Check for NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check for color term environment variables
	term := os.Getenv("TERM")
	colorTerm := os.Getenv("COLORTERM")

	if colorTerm == "truecolor" || colorTerm == "24bit" {
		return true
	}

	if strings.Contains(term, "color") || strings.Contains(term, "256") {
		return true
	}

	// Windows specific check
	if runtime.GOOS == "windows" {
		// Windows 10+ supports ANSI colors
		return true
	}

	return term != "dumb" && term != ""
}

// SetColorEnabled enables or disables color output.
func SetColorEnabled(enabled bool) {
	defaultColorConfig.Enabled = enabled
}

// ForceColor forces color output even if not in a terminal.
func ForceColor(force bool) {
	defaultColorConfig.Force = force
	if force {
		defaultColorConfig.Enabled = true
	}
}

// Colorize applies color to text if colors are enabled.
func Colorize(color, text string) string {
	if !defaultColorConfig.Enabled && !defaultColorConfig.Force || color == "" {
		return text
	}

	return color + text + Reset
}

// Success returns text in green.
func Success(text string) string {
	return Colorize(Green, text)
}

// Error returns text in red.
func Error(text string) string {
	return Colorize(Red, text)
}

// Warning returns text in yellow.
func Warning(text string) string {
	return Colorize(Yellow, text)
}

// Info returns text in blue.
func Info(text string) string {
	return Colorize(Blue, text)
}

// Highlight returns text in cyan.
func Highlight(text string) string {
	return Colorize(Cyan, text)
}

// BoldText returns bold text.
func BoldText(text string) string {
	return Colorize(Bold, text)
}

// DimText returns dimmed text.
func DimText(text string) string {
	return Colorize(Dim, text)
}

// FormatSize formats a file size with appropriate units and colors.
func FormatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	var (
		result string
		color  string
	)

	switch {
	case size >= GB:
		result = fmt.Sprintf("%.2f GB", float64(size)/GB)
		color = Red
	case size >= MB:
		result = fmt.Sprintf("%.2f MB", float64(size)/MB)
		color = Yellow
	case size >= KB:
		result = fmt.Sprintf("%.2f KB", float64(size)/KB)
		color = Green
	default:
		result = fmt.Sprintf("%d B", size)
		color = Blue
	}

	return Colorize(color, result)
}

// FormatSpeed formats download speed with colors.
func FormatSpeed(bytesPerSecond int64) string {
	const (
		KB = 1024
		MB = KB * 1024
	)

	var (
		result string
		color  string
	)

	switch {
	case bytesPerSecond >= MB:
		result = fmt.Sprintf("%.2f MB/s", float64(bytesPerSecond)/MB)
		color = Green
	case bytesPerSecond >= KB:
		result = fmt.Sprintf("%.2f KB/s", float64(bytesPerSecond)/KB)
		color = Yellow
	default:
		result = fmt.Sprintf("%d B/s", bytesPerSecond)
		color = Red
	}

	return Colorize(color, result)
}

// FormatDuration formats duration with colors.
func FormatDuration(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	var parts []string
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}

	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}

	if secs > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", secs))
	}

	result := strings.Join(parts, " ")

	// Color based on duration
	var color string

	switch {
	case seconds < 10:
		color = Green
	case seconds < 60:
		color = Yellow
	default:
		color = Red
	}

	return Colorize(color, result)
}

// PrintHeader prints a formatted header.
func PrintHeader(text string) {
	fmt.Println()
	fmt.Println(Colorize(Bold+Blue, "═══════════════════════════════════════════"))
	fmt.Println(Colorize(Bold+Cyan, "  "+text))
	fmt.Println(Colorize(Bold+Blue, "═══════════════════════════════════════════"))
	fmt.Println()
}

// PrintSection prints a section header.
func PrintSection(text string) {
	fmt.Println()
	fmt.Println(Colorize(Bold+Cyan, "▶ "+text))
	fmt.Println(Colorize(Dim, strings.Repeat("─", 40)))
}

// PrintKeyValue prints a key-value pair with formatting.
func PrintKeyValue(key, value string) {
	fmt.Printf("%s %s\n",
		Colorize(Bold+Blue, key+":"),
		value)
}

// ClearLine clears the current line.
func ClearLine() {
	fmt.Print("\r\033[K")
}

// MoveCursorUp moves cursor up n lines.
func MoveCursorUp(n int) {
	fmt.Printf("\033[%dA", n)
}

// MoveCursorDown moves cursor down n lines.
func MoveCursorDown(n int) {
	fmt.Printf("\033[%dB", n)
}

// HideCursor hides the cursor.
func HideCursor() {
	fmt.Print("\033[?25l")
}

// ShowCursor shows the cursor.
func ShowCursor() {
	fmt.Print("\033[?25h")
}
