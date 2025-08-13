package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// PromptType represents the type of prompt.
type PromptType int

const (
	PromptTypeInfo PromptType = iota
	PromptTypeWarning
	PromptTypeError
	PromptTypeSuccess
	PromptTypeQuestion
)

// Prompt displays a prompt and waits for user input.
func Prompt(message string, promptType PromptType) string {
	prefix := getPromptPrefix(promptType)
	fmt.Printf("%s %s ", prefix, message)

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')

	return strings.TrimSpace(input)
}

// PromptYesNo asks a yes/no question.
func PromptYesNo(message string, defaultYes bool) bool {
	var suffix string
	if defaultYes {
		suffix = "[Y/n]"
	} else {
		suffix = "[y/N]"
	}

	response := Prompt(fmt.Sprintf("%s %s", message, suffix), PromptTypeQuestion)
	response = strings.ToLower(response)

	if response == "" {
		return defaultYes
	}

	return response == "y" || response == "yes"
}

// PromptChoice presents multiple choices to the user.
func PromptChoice(message string, choices []string, defaultChoice int) int {
	fmt.Println(Colorize(Bold+Cyan, message))

	for i, choice := range choices {
		marker := " "
		if i == defaultChoice {
			marker = "►"
		}

		fmt.Printf("%s %d. %s\n", marker, i+1, choice)
	}

	response := Prompt("Enter choice number", PromptTypeQuestion)
	if response == "" {
		return defaultChoice
	}

	var choice int
	if _, err := fmt.Sscanf(response, "%d", &choice); err == nil {
		if choice >= 1 && choice <= len(choices) {
			return choice - 1
		}
	}

	return defaultChoice
}

// PromptPassword prompts for a password (input is hidden).
func PromptPassword(message string) string {
	fmt.Printf("%s %s ", getPromptPrefix(PromptTypeQuestion), message)

	// Disable echo
	fmt.Print("\033[8m")
	defer fmt.Print("\033[28m")

	reader := bufio.NewReader(os.Stdin)
	password, _ := reader.ReadString('\n')

	fmt.Println() // New line after password input

	return strings.TrimSpace(password)
}

// getPromptPrefix returns the prefix for different prompt types.
func getPromptPrefix(promptType PromptType) string {
	switch promptType {
	case PromptTypeInfo:
		return Colorize(Blue, "ℹ")
	case PromptTypeWarning:
		return Colorize(Yellow, "⚠")
	case PromptTypeError:
		return Colorize(Red, "✗")
	case PromptTypeSuccess:
		return Colorize(Green, "✓")
	case PromptTypeQuestion:
		return Colorize(Cyan, "?")
	default:
		return ">"
	}
}

// Alert displays an alert message.
func Alert(message string, alertType PromptType) {
	prefix := getPromptPrefix(alertType)
	fmt.Printf("%s %s\n", prefix, message)
}

// AlertInfo displays an info alert.
func AlertInfo(message string) {
	Alert(message, PromptTypeInfo)
}

// AlertWarning displays a warning alert.
func AlertWarning(message string) {
	Alert(message, PromptTypeWarning)
}

// AlertError displays an error alert.
func AlertError(message string) {
	Alert(message, PromptTypeError)
}

// AlertSuccess displays a success alert.
func AlertSuccess(message string) {
	Alert(message, PromptTypeSuccess)
}

// ShowProgress displays a simple text progress indicator.
func ShowProgress(current, total int64, message string) {
	percentage := float64(current) / float64(total) * 100
	barWidth := 40
	filled := int(percentage / 100 * float64(barWidth))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	ClearLine()
	fmt.Printf("%s [%s] %.1f%% %s",
		Colorize(Cyan, "↓"),
		Colorize(Green, bar),
		percentage,
		DimText(message))
}
