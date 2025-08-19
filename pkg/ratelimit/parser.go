package ratelimit

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ParseRate parses a human-readable rate string into bytes per second.
// Supports formats like:
//   - "1048576" (bytes per second)
//   - "1MB/s", "1mb/s" (megabytes per second)
//   - "500KB/s", "500k/s" (kilobytes per second)
//   - "2GB/s", "2g/s" (gigabytes per second)
//   - "1MB", "500k" (without /s suffix)
//
// Returns 0 for invalid input or explicit "0".
func ParseRate(rateStr string) (int64, error) {
	if rateStr == "" || rateStr == "0" {
		return 0, nil
	}

	// Remove whitespace and convert to lowercase for easier parsing
	rateStr = strings.TrimSpace(strings.ToLower(rateStr))

	// Remove optional "/s" suffix
	rateStr = strings.TrimSuffix(rateStr, "/s")

	// Regular expression to parse number and unit
	// Matches: number + optional unit (k, kb, m, mb, g, gb)
	re := regexp.MustCompile(`^(\d*\.?\d+)(k|kb|m|mb|g|gb)?$`)
	matches := re.FindStringSubmatch(rateStr)

	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid rate format: %q (examples: 1MB/s, 500k, 2048)", rateStr)
	}

	// Parse the numeric part
	numStr := matches[1]
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in rate: %q", numStr)
	}

	if num < 0 {
		return 0, fmt.Errorf("rate cannot be negative: %f", num)
	}

	// Parse the unit (if present)
	var multiplier int64 = 1
	if len(matches) > 2 && matches[2] != "" {
		unit := matches[2]
		switch unit {
		case "k", "kb":
			multiplier = 1024
		case "m", "mb":
			multiplier = 1024 * 1024
		case "g", "gb":
			multiplier = 1024 * 1024 * 1024
		default:
			return 0, fmt.Errorf("unsupported unit: %q (supported: k, kb, m, mb, g, gb)", unit)
		}
	}

	result := int64(num * float64(multiplier))

	// Sanity check for very small rates
	if result > 0 && result < 1 {
		return 0, fmt.Errorf("rate too small: %d bytes/s (minimum: 1 byte/s)", result)
	}

	return result, nil
}

// FormatRate formats a rate in bytes per second to a human-readable string.
func FormatRate(bytesPerSec int64) string {
	if bytesPerSec == 0 {
		return "unlimited"
	}

	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytesPerSec >= GB:
		if bytesPerSec%GB == 0 {
			return fmt.Sprintf("%dGB/s", bytesPerSec/GB)
		}
		return fmt.Sprintf("%.1fGB/s", float64(bytesPerSec)/float64(GB))
	case bytesPerSec >= MB:
		if bytesPerSec%MB == 0 {
			return fmt.Sprintf("%dMB/s", bytesPerSec/MB)
		}
		return fmt.Sprintf("%.1fMB/s", float64(bytesPerSec)/float64(MB))
	case bytesPerSec >= KB:
		if bytesPerSec%KB == 0 {
			return fmt.Sprintf("%dKB/s", bytesPerSec/KB)
		}
		return fmt.Sprintf("%.1fKB/s", float64(bytesPerSec)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes/s", bytesPerSec)
	}
}

// ValidateRate checks if a rate string is valid without parsing it.
func ValidateRate(rateStr string) error {
	_, err := ParseRate(rateStr)
	return err
}
