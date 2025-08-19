package help

import (
	"strings"
	"testing"

	"github.com/forest6511/gdl/pkg/errors"
)

func TestNewHelpProvider(t *testing.T) {
	hp := NewHelpProvider()

	if hp == nil {
		t.Fatal("NewHelpProvider() should not return nil")
	}

	if hp.commandExamples == nil {
		t.Error("commandExamples should be initialized")
	}

	if hp.errorGuides == nil {
		t.Error("errorGuides should be initialized")
	}

	if hp.troubleshooting == nil {
		t.Error("troubleshooting should be initialized")
	}
}

func TestHelpProvider_GetCommandHelp(t *testing.T) {
	hp := NewHelpProvider()

	// Test known command
	help := hp.GetCommandHelp("download")
	if help == "" {
		t.Error("GetCommandHelp should return non-empty help for 'download'")
	}

	if !strings.Contains(help, "# Help for 'download'") {
		t.Error("Help should contain command title")
	}

	if !strings.Contains(help, "## Examples") {
		t.Error("Help should contain examples section")
	}

	// Test unknown command
	help = hp.GetCommandHelp("unknown-command")
	if !strings.Contains(help, "No specific help available") {
		t.Error("Should return fallback message for unknown command")
	}
}

func TestHelpProvider_GetErrorHelp(t *testing.T) {
	hp := NewHelpProvider()

	// Test known error code
	help := hp.GetErrorHelp(errors.CodeNetworkError)
	if help == "" {
		t.Error("GetErrorHelp should return non-empty help for network errors")
	}

	if !strings.Contains(help, "# Error Help: network_error") {
		t.Error("Error help should contain error code title")
	}

	if !strings.Contains(help, "## Description") {
		t.Error("Error help should contain description section")
	}

	if !strings.Contains(help, "## Common Causes") {
		t.Error("Error help should contain common causes section")
	}

	if !strings.Contains(help, "## Solutions") {
		t.Error("Error help should contain solutions section")
	}

	// Test unknown error code
	help = hp.GetErrorHelp(errors.ErrorCode(999))
	if !strings.Contains(help, "No specific guidance available") {
		t.Error("Should return fallback message for unknown error code")
	}
}

func TestHelpProvider_GetTroubleshootingHelp(t *testing.T) {
	hp := NewHelpProvider()

	// Test known scenario
	help := hp.GetTroubleshootingHelp("slow-downloads")
	if help == "" {
		t.Error("GetTroubleshootingHelp should return non-empty help for slow downloads")
	}

	if !strings.Contains(help, "# Troubleshooting: Slow Downloads") {
		t.Error("Troubleshooting help should contain scenario title")
	}

	if !strings.Contains(help, "## Steps") {
		t.Error("Troubleshooting help should contain steps section")
	}

	// Test unknown scenario
	help = hp.GetTroubleshootingHelp("unknown-scenario")
	if !strings.Contains(help, "No specific troubleshooting guide available") {
		t.Error("Should return fallback message for unknown scenario")
	}
}

func TestHelpProvider_GetQuickHelp(t *testing.T) {
	hp := NewHelpProvider()

	help := hp.GetQuickHelp()
	if help == "" {
		t.Error("GetQuickHelp should return non-empty help")
	}

	if !strings.Contains(help, "# GDL Quick Help") {
		t.Error("Quick help should contain title")
	}

	if !strings.Contains(help, "## Common Commands") {
		t.Error("Quick help should contain common commands section")
	}

	if !strings.Contains(help, "## Quick Fixes") {
		t.Error("Quick help should contain quick fixes section")
	}

	if !strings.Contains(help, "gdl download") {
		t.Error("Quick help should contain basic download command")
	}
}

func TestHelpProvider_GetContextualHelp(t *testing.T) {
	hp := NewHelpProvider()

	// Test error context
	help := hp.GetContextualHelp("error", errors.CodeNetworkError)
	if !strings.Contains(help, "network_error") {
		t.Error("Contextual error help should contain error code")
	}

	// Test command context
	help = hp.GetContextualHelp("command", "download")
	if !strings.Contains(help, "download") {
		t.Error("Contextual command help should contain command name")
	}

	// Test troubleshooting context
	help = hp.GetContextualHelp("troubleshooting", "slow-downloads")
	if !strings.Contains(help, "Slow Downloads") {
		t.Error("Contextual troubleshooting help should contain scenario name")
	}

	// Test quick context
	help = hp.GetContextualHelp("quick")
	if !strings.Contains(help, "GDL Quick Help") {
		t.Error("Quick context should return quick help")
	}

	// Test unknown context
	help = hp.GetContextualHelp("unknown")
	if !strings.Contains(help, "GDL Quick Help") {
		t.Error("Unknown context should return quick help as fallback")
	}
}

func TestErrorGuide_NetworkError(t *testing.T) {
	hp := NewHelpProvider()
	guide, exists := hp.errorGuides[errors.CodeNetworkError]

	if !exists {
		t.Fatal("Network error guide should exist")
	}

	if guide.ErrorCode != errors.CodeNetworkError {
		t.Error("Error guide should have correct error code")
	}

	if guide.Description == "" {
		t.Error("Error guide should have description")
	}

	if len(guide.CommonCauses) == 0 {
		t.Error("Error guide should have common causes")
	}

	if len(guide.Solutions) == 0 {
		t.Error("Error guide should have solutions")
	}

	// Check solution priorities
	for _, solution := range guide.Solutions {
		if solution.Priority < 1 {
			t.Error("Solution should have valid priority")
		}

		if solution.Title == "" {
			t.Error("Solution should have title")
		}

		if solution.Description == "" {
			t.Error("Solution should have description")
		}
	}
}

func TestErrorGuide_TimeoutError(t *testing.T) {
	hp := NewHelpProvider()
	guide, exists := hp.errorGuides[errors.CodeTimeout]

	if !exists {
		t.Fatal("Timeout error guide should exist")
	}

	if len(guide.Solutions) == 0 {
		t.Error("Timeout error guide should have solutions")
	}

	// Should include increase timeout solution
	foundTimeoutSolution := false

	for _, solution := range guide.Solutions {
		if strings.Contains(strings.ToLower(solution.Title), "timeout") {
			foundTimeoutSolution = true
			break
		}
	}

	if !foundTimeoutSolution {
		t.Error("Timeout error guide should include timeout-related solution")
	}
}

func TestTroubleshootingGuide_SlowDownloads(t *testing.T) {
	hp := NewHelpProvider()
	guide, exists := hp.troubleshooting["slow-downloads"]

	if !exists {
		t.Fatal("Slow downloads troubleshooting guide should exist")
	}

	if guide.Scenario != "Slow Downloads" {
		t.Error("Guide should have correct scenario name")
	}

	if guide.Description == "" {
		t.Error("Guide should have description")
	}

	if len(guide.Steps) == 0 {
		t.Error("Guide should have steps")
	}

	// Check steps structure
	for i, step := range guide.Steps {
		if step.Step != i+1 {
			t.Errorf("Step %d should have correct step number", i)
		}

		if step.Title == "" {
			t.Errorf("Step %d should have title", i)
		}

		if step.Description == "" {
			t.Errorf("Step %d should have description", i)
		}
	}
}

func TestCommandExamples_Download(t *testing.T) {
	hp := NewHelpProvider()
	examples, exists := hp.commandExamples["download"]

	if !exists {
		t.Fatal("Download command examples should exist")
	}

	if len(examples) == 0 {
		t.Error("Download command should have examples")
	}

	// Check for basic example
	foundBasicExample := false

	for _, example := range examples {
		if strings.Contains(example.Command, "gdl download") &&
			strings.Contains(example.Description, "Basic") {
			foundBasicExample = true

			if example.Command == "" {
				t.Error("Example should have command")
			}

			if example.Description == "" {
				t.Error("Example should have description")
			}

			break
		}
	}

	if !foundBasicExample {
		t.Error("Should have basic download example")
	}
}

func TestCommandExamples_Config(t *testing.T) {
	hp := NewHelpProvider()
	examples, exists := hp.commandExamples["config"]

	if !exists {
		t.Fatal("Config command examples should exist")
	}

	if len(examples) == 0 {
		t.Error("Config command should have examples")
	}

	// Check for config show example
	foundShowExample := false

	for _, example := range examples {
		if strings.Contains(example.Command, "gdl config show") {
			foundShowExample = true
			break
		}
	}

	if !foundShowExample {
		t.Error("Should have config show example")
	}
}

func TestFormatHelp(t *testing.T) {
	content := "# Test Help\n\nThis is test content."
	formatted := FormatHelp(content)

	// Basic formatting test - in a real implementation this might do more
	if formatted != content {
		t.Error("FormatHelp should return the content (basic implementation)")
	}
}

func TestExample_Structure(t *testing.T) {
	example := Example{
		Command:     "gdl download https://example.com/file.zip",
		Description: "Basic download example",
		Tags:        []string{"basic", "simple"},
	}

	if example.Command == "" {
		t.Error("Example should have command")
	}

	if example.Description == "" {
		t.Error("Example should have description")
	}

	if len(example.Tags) == 0 {
		t.Error("Example should have tags")
	}
}

func TestSolution_Priority(t *testing.T) {
	solution := Solution{
		Title:       "Test Solution",
		Description: "Test description",
		Commands:    []string{"test command"},
		Priority:    1,
	}

	if solution.Priority != 1 {
		t.Error("Solution should have correct priority")
	}

	if len(solution.Commands) == 0 {
		t.Error("Solution should have commands")
	}
}

func TestTroubleshootingStep_Structure(t *testing.T) {
	step := TroubleshootingStep{
		Step:        1,
		Title:       "Test Step",
		Description: "Test description",
		Commands:    []string{"test command"},
		Expected:    "Expected result",
	}

	if step.Step != 1 {
		t.Error("Step should have correct step number")
	}

	if step.Title == "" {
		t.Error("Step should have title")
	}

	if step.Description == "" {
		t.Error("Step should have description")
	}

	if step.Expected == "" {
		t.Error("Step should have expected result")
	}
}
