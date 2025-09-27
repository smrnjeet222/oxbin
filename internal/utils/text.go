package utils

import (
	"fmt"
)

// WrapText wraps text to fit within specified width, preserving words when possible
func WrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	if len(text) <= width {
		return []string{text}
	}

	var lines []string
	remaining := text

	for len(remaining) > width {
		// Try to find a good break point (space, slash, dash)
		breakPoint := width
		for i := width - 1; i >= width/2; i-- {
			if remaining[i] == ' ' || remaining[i] == '/' || remaining[i] == '-' || remaining[i] == '_' {
				breakPoint = i
				break
			}
		}

		lines = append(lines, remaining[:breakPoint])
		remaining = remaining[breakPoint:]

		// Skip leading spaces in the next line
		for len(remaining) > 0 && remaining[0] == ' ' {
			remaining = remaining[1:]
		}
	}

	if len(remaining) > 0 {
		lines = append(lines, remaining)
	}

	return lines
}

// FormatFileSize formats file size in human-readable format
func FormatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// TruncateString truncates a string to maxLength with ellipsis
func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	if maxLength <= 3 {
		return s[:maxLength]
	}
	return s[:maxLength-3] + "..."
}
