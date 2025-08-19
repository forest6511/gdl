package ui

import (
	"errors"
	"strings"
	"testing"

	downloadErrors "github.com/forest6511/gdl/pkg/errors"
)

func TestNewFormatter(t *testing.T) {
	formatter := NewFormatter()

	if formatter == nil {
		t.Fatal("NewFormatter should not return nil")
	}
}

func TestFormatter_WithColor(t *testing.T) {
	formatter := NewFormatter()

	// Test enabling colors
	formatterWithColor := formatter.WithColor(true)
	if formatterWithColor == nil {
		t.Error("WithColor should return formatter")
	}

	// Test disabling colors
	formatterNoColor := formatter.WithColor(false)
	if formatterNoColor == nil {
		t.Error("WithColor should return formatter")
	}
}

func TestFormatter_FormatError(t *testing.T) {
	formatter := NewFormatter()

	// Test with generic error
	genericErr := errors.New("Test error message")
	errorMsg := formatter.FormatError(genericErr, nil)

	if errorMsg == "" {
		t.Error("FormatError should return non-empty string")
	}

	if !strings.Contains(errorMsg, "Test error message") {
		t.Error("Formatted error should contain original message")
	}
}

func TestFormatter_FormatDownloadError(t *testing.T) {
	formatter := NewFormatter()

	// Test with DownloadError
	downloadErr := downloadErrors.NewDownloadError(
		downloadErrors.CodeNetworkError,
		"Network failed",
	)
	errorMsg := formatter.FormatError(downloadErr, nil)

	if errorMsg == "" {
		t.Error("FormatError should return non-empty string")
	}

	if !strings.Contains(errorMsg, "Network failed") {
		t.Error("Formatted error should contain original message")
	}
}

func TestFormatter_FormatMessage(t *testing.T) {
	formatter := NewFormatter()

	// Test different message types
	testCases := []struct {
		msgType MessageType
		message string
	}{
		{MessageError, "Test error"},
		{MessageSuccess, "Test success"},
		{MessageWarning, "Test warning"},
		{MessageInfo, "Test info"},
	}

	for _, tc := range testCases {
		msg := formatter.FormatMessage(tc.msgType, "%s", tc.message)

		if msg == "" {
			t.Errorf("FormatMessage should return non-empty string for %v", tc.msgType)
		}

		if !strings.Contains(msg, tc.message) {
			t.Errorf("Formatted message should contain original message for %v", tc.msgType)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	testCases := []struct {
		bytes    uint64
		expected string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"}, // 1.5 * 1024
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tc := range testCases {
		result := formatBytes(tc.bytes)
		if result != tc.expected {
			t.Errorf("formatBytes(%d) = %s, want %s", tc.bytes, result, tc.expected)
		}
	}
}

func TestNewSpinner(t *testing.T) {
	spinner := NewSpinner(SpinnerStyleDots, "Loading...")

	if spinner == nil {
		t.Fatal("NewSpinner should not return nil")
	}
}

func TestSpinner_Start_Stop(t *testing.T) {
	spinner := NewSpinner(SpinnerStyleDots, "Loading...")

	spinner.Start()

	// Check that spinner was started (we can't check internal state easily)
	// So we just ensure Start and Stop don't panic

	spinner.Stop()
}

func TestSpinner_SetMessage(t *testing.T) {
	spinner := NewSpinner(SpinnerStyleDots, "Loading...")

	message := "Test message"
	spinner.SetMessage(message)

	// Since SetMessage doesn't return anything, we just ensure it doesn't panic
}

func TestColors(t *testing.T) {
	// Test that color constants are defined and non-empty
	colors := []string{
		Reset, Red, Green, Yellow, Blue, Magenta, Cyan, White,
		Bold, Dim,
	}

	for i, color := range colors {
		if color == "" && i > 0 { // Reset can be empty
			t.Errorf("Color constant %d should not be empty", i)
		}
	}
}

func TestColorFunctions(t *testing.T) {
	// Test color utility functions
	testText := "test"

	functions := []func(string) string{
		Success, Error, Warning, Info, Highlight, BoldText, DimText,
	}

	for _, fn := range functions {
		result := fn(testText)
		if result == "" {
			t.Error("Color function should return non-empty string")
		}
	}
}

func TestFormatSize(t *testing.T) {
	testCases := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{1024, "1.00 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
	}

	for _, tc := range testCases {
		result := FormatSize(tc.bytes)
		// Just check if the result contains the expected text (ignore colors)
		if !strings.Contains(result, tc.expected) {
			t.Errorf("FormatSize(%d) should contain %s, got %s", tc.bytes, tc.expected, result)
		}
	}
}

func TestFormatSpeed(t *testing.T) {
	result := FormatSpeed(1024)
	if result == "" {
		t.Error("FormatSpeed should return non-empty string")
	}

	if !strings.Contains(result, "KB/s") {
		t.Error("FormatSpeed should contain speed unit")
	}
}

func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		seconds  int
		expected string
	}{
		{0, "0s"},
		{30, "30s"},
		{60, "1m"},
		{90, "1m 30s"},
		{3600, "1h"},
		{3661, "1h 1m 1s"},
	}

	for _, tc := range testCases {
		result := FormatDuration(tc.seconds)
		// Just check if the result contains the expected text (ignore colors)
		if !strings.Contains(result, tc.expected) {
			t.Errorf(
				"FormatDuration(%d) should contain %s, got %s",
				tc.seconds,
				tc.expected,
				result,
			)
		}
	}
}

func TestFormatter_FormatProgressBar(t *testing.T) {
	formatter := NewFormatter()

	// Test basic progress bar
	progressBar := formatter.FormatProgressBar(50, 100, nil)
	if progressBar == "" {
		t.Error("FormatProgressBar should return non-empty string")
	}

	// Test with custom options
	options := &ProgressBarOptions{
		Width:          20,
		ShowPercentage: true,
		ShowSize:       true,
	}

	progressBarWithOptions := formatter.FormatProgressBar(25, 100, options)
	if progressBarWithOptions == "" {
		t.Error("FormatProgressBar with options should return non-empty string")
	}

	// Test with template
	templateOptions := &ProgressBarOptions{
		Width:    30,
		Template: "Progress: {bar} {percentage}%",
	}

	progressBarWithTemplate := formatter.FormatProgressBar(75, 100, templateOptions)
	if progressBarWithTemplate == "" {
		t.Error("FormatProgressBar with template should return non-empty string")
	}
}

func TestFormatter_FormatStatusIndicator(t *testing.T) {
	formatter := NewFormatter()

	statusTypes := []StatusIndicator{
		StatusPending,
		StatusInProgress,
		StatusCompleted,
		StatusFailed,
		StatusPaused,
		StatusCancelled,
	}

	for _, status := range statusTypes {
		indicator := formatter.FormatStatusIndicator(status, "Test message")
		if indicator == "" {
			t.Errorf("FormatStatusIndicator should return non-empty string for status %v", status)
		}

		if !strings.Contains(indicator, "Test message") {
			t.Errorf("Status indicator should contain message for status %v", status)
		}
	}
}

func TestFormatter_WithLanguage(t *testing.T) {
	formatter := NewFormatter()

	// Test different languages
	languages := []Language{
		LanguageEnglish,
		LanguageJapanese,
		LanguageSpanish,
		LanguageFrench,
	}

	for _, lang := range languages {
		langFormatter := formatter.WithLanguage(lang)
		if langFormatter == nil {
			t.Errorf("WithLanguage should return formatter for language %v", lang)
		}
	}
}

func TestFormatter_WithWriter(t *testing.T) {
	formatter := NewFormatter()

	// Test with a custom writer
	var buf strings.Builder

	writerFormatter := formatter.WithWriter(&buf)
	if writerFormatter == nil {
		t.Error("WithWriter should return formatter")
	}
}

func TestFormatter_WithInteractive(t *testing.T) {
	formatter := NewFormatter()

	// Test enabling/disabling interactive mode
	interactiveFormatter := formatter.WithInteractive(true)
	if interactiveFormatter == nil {
		t.Error("WithInteractive(true) should return formatter")
	}

	nonInteractiveFormatter := formatter.WithInteractive(false)
	if nonInteractiveFormatter == nil {
		t.Error("WithInteractive(false) should return formatter")
	}
}

func TestTableFormatter(t *testing.T) {
	formatter := NewFormatter()

	// Test creating table formatter
	headers := []string{"Name", "Age", "City"}

	table := formatter.NewTableFormatter(headers)
	if table == nil {
		t.Error("NewTableFormatter should not return nil")
	}

	// Test adding rows
	table.AddRow([]string{"John", "25", "NYC"})
	table.AddRow([]string{"Alice", "30", "LA"})

	// Test formatting
	result := table.Format()
	if result == "" {
		t.Error("Table Format should return non-empty string")
	}

	// Check that headers are included
	for _, header := range headers {
		if !strings.Contains(result, header) {
			t.Errorf("Formatted table should contain header %s", header)
		}
	}
}

func TestFormatter_ErrorFormatOptions(t *testing.T) {
	formatter := NewFormatter()

	downloadErr := downloadErrors.NewDownloadErrorWithDetails(
		downloadErrors.CodeNetworkError,
		"Network failed",
		"Connection timeout details",
	)
	downloadErr.URL = "https://example.com/file.zip"
	downloadErr.Filename = "test.zip"

	// Test with different error format options
	options := &ErrorFormatOptions{
		ShowErrorCode:   true,
		ShowSuggestions: true,
		ShowTimestamp:   true,
		Compact:         false,
		MultiLine:       true,
	}

	errorMsg := formatter.FormatError(downloadErr, options)
	if errorMsg == "" {
		t.Error("FormatError with options should return non-empty string")
	}

	// Test compact format
	compactOptions := &ErrorFormatOptions{
		Compact: true,
	}

	compactMsg := formatter.FormatError(downloadErr, compactOptions)
	if compactMsg == "" {
		t.Error("FormatError with compact option should return non-empty string")
	}

	// Test with nil error
	nilMsg := formatter.FormatError(nil, nil)
	if nilMsg != "" {
		t.Error("FormatError with nil error should return empty string")
	}
}

func TestFormatter_MessageTypes(t *testing.T) {
	formatter := NewFormatter()

	// Test all message types
	messageTypes := []MessageType{
		MessageInfo,
		MessageSuccess,
		MessageWarning,
		MessageError,
		MessageDebug,
		MessagePrompt,
	}

	for _, msgType := range messageTypes {
		msg := formatter.FormatMessage(msgType, "Test message for %v", msgType)
		if msg == "" {
			t.Errorf("FormatMessage should return non-empty string for type %v", msgType)
		}

		if !strings.Contains(msg, "Test message") {
			t.Errorf("Formatted message should contain original text for type %v", msgType)
		}
	}
}

func TestFormatter_ClearLine(t *testing.T) {
	formatter := NewFormatter()

	// Just test that ClearLine doesn't panic
	formatter.ClearLine()
}

func TestIsColorSupported(t *testing.T) {
	// Test the IsColorSupported function
	result := IsColorSupported()
	// We can't test exact behavior since it depends on environment
	// Just ensure it returns a boolean without panic
	_ = result
}

func TestIsTerminalInteractive(t *testing.T) {
	// Test the IsTerminalInteractive function
	result := IsTerminalInteractive()
	// We can't test exact behavior since it depends on environment
	// Just ensure it returns a boolean without panic
	_ = result
}

func TestSupportsColor(t *testing.T) {
	// Test the supportsColor function
	result := supportsColor()
	// We can't test exact behavior since it depends on environment
	// Just ensure it returns a boolean without panic
	_ = result
}

func TestSetColorEnabled(t *testing.T) {
	// Test SetColorEnabled function
	SetColorEnabled(true)
	SetColorEnabled(false)
}

func TestForceColor(t *testing.T) {
	// Test ForceColor function
	ForceColor(true)
	ForceColor(false)
}

func TestColorize(t *testing.T) {
	// Test Colorize function with proper parameter order (color, text)
	result := Colorize("red", "test")
	if result == "" {
		t.Error("Colorize should return non-empty string")
	}

	// Test with empty color
	result2 := Colorize("", "test")
	if result2 != "test" {
		t.Error("Colorize with empty color should return original text")
	}
}

func TestPrintFunctions(t *testing.T) {
	// Test PrintHeader, PrintSection, PrintKeyValue
	PrintHeader("Test Header")
	PrintSection("Test Section")
	PrintKeyValue("Key", "Value")
}

func TestCursorFunctions(t *testing.T) {
	// Test cursor manipulation functions
	ClearLine()
	MoveCursorUp(1)
	MoveCursorDown(1)
	HideCursor()
	ShowCursor()
}

func TestFormatter_PrintMessage(t *testing.T) {
	formatter := NewFormatter()

	// Test PrintMessage (which outputs to the writer)
	formatter.PrintMessage(MessageInfo, "Test message")
	formatter.PrintMessage(MessageError, "Error: %s", "test error")
}

func TestFormatter_PromptFunctions(t *testing.T) {
	formatter := NewFormatter()

	// These functions require interactive input, so we just test they don't panic
	// In a real test environment, we would mock the input

	// We can't actually test these without mocking stdin, but we can test they don't panic during setup
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Prompt functions should not panic during basic calls: %v", r)
		}
	}()

	// Test that the functions exist and can be called (they'll return immediately due to non-interactive environment)
	_, _ = formatter.Prompt("Test prompt")
	_, _ = formatter.ConfirmPrompt("Confirm?", true)
	_, _ = formatter.SelectPrompt("Select", []string{"a", "b"}, 0)
}

func TestFormatter_CreateLoadingAnimation(t *testing.T) {
	formatter := NewFormatter()

	// Test CreateLoadingAnimation
	stopFunc := formatter.CreateLoadingAnimation("Loading...")
	if stopFunc == nil {
		t.Error("CreateLoadingAnimation should return a stop function")
	}

	// Stop the animation
	stopFunc()
}

func TestPromptFunctions(t *testing.T) {
	// Test standalone prompt functions
	// These require interactive input, so we just test they don't panic during creation
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Prompt functions should not panic: %v", r)
		}
	}()

	// Test basic function calls (they'll return empty/default values in non-interactive environment)
	Prompt("Test", PromptTypeQuestion)
	PromptYesNo("Test question", false)
	PromptChoice("Pick one", []string{"a", "b"}, 0)
	PromptPassword("Password")
}

func TestAlertFunctions(t *testing.T) {
	// Test alert functions
	Alert("Test alert", PromptTypeInfo)
	AlertInfo("Info alert")
	AlertWarning("Warning alert")
	AlertError("Error alert")
	AlertSuccess("Success alert")
}

func TestShowProgress(t *testing.T) {
	// Test ShowProgress function
	ShowProgress(50, 100, "Working...")
}

func TestSpinner_AdditionalFunctions(t *testing.T) {
	spinner := NewSpinner(SpinnerStyleDots, "Loading...")

	// Test SetInterval
	spinner.SetInterval(100)

	// Test various stop functions
	spinner.StopWithMessage("Completed")

	spinner2 := NewSpinner(SpinnerStyleDots, "Loading...")
	spinner2.StopWithSuccess("Success")

	spinner3 := NewSpinner(SpinnerStyleDots, "Loading...")
	spinner3.StopWithError("Error occurred")

	spinner4 := NewSpinner(SpinnerStyleDots, "Loading...")
	spinner4.StopWithWarning("Warning")
}

func TestGetSpinnerFrames(t *testing.T) {
	// Test different spinner styles
	styles := []SpinnerStyle{
		SpinnerStyleDots,
		SpinnerStyleLine,
		SpinnerStyleCircle,
		SpinnerStyleSquare,
	}

	for _, style := range styles {
		frames := getSpinnerFrames(style)
		if len(frames) == 0 {
			t.Errorf("getSpinnerFrames should return frames for style %v", style)
		}
	}
}
