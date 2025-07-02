package utils

import (
	"fmt"
	"strings"
)

// ANSI color codes
const (
	Reset = "\033[0m"
	Bold  = "\033[1m"
	Dim   = "\033[2m"

	// Colors
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Gray    = "\033[90m"

	// Bright colors
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	BrightWhite   = "\033[97m"

	// Background colors
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
)

// Style types
type StyleType int

const (
	StyleSuccess StyleType = iota
	StyleError
	StyleWarning
	StyleInfo
	StyleCommand
	StyleHighlight
	StyleSubtle
	StyleDanger
	StyleCritical
)

// getStyle returns the ANSI codes for a given style type
func getStyle(style StyleType) string {
	switch style {
	case StyleSuccess:
		return Green + Bold
	case StyleError:
		return Red + Bold
	case StyleWarning:
		return Yellow + Bold
	case StyleInfo:
		return Blue + Bold
	case StyleCommand:
		return Cyan + Bold
	case StyleHighlight:
		return Magenta + Bold
	case StyleSubtle:
		return Gray
	case StyleDanger:
		return BrightRed + Bold
	case StyleCritical:
		return BgRed + BrightWhite + Bold
	default:
		return Reset
	}
}

// Styled applies a style to text
func Styled(text string, style StyleType) string {
	return getStyle(style) + text + Reset
}

// Box creates a bordered box around text
func Box(title string, content string, style StyleType) string {
	lines := strings.Split(content, "\n")
	maxWidth := len(title)

	for _, line := range lines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}

	// Ensure minimum width
	if maxWidth < 40 {
		maxWidth = 40
	}

	boxWidth := maxWidth + 4 // 2 spaces padding on each side

	// Top border
	topBorder := "┌" + strings.Repeat("─", boxWidth-2) + "┐"

	// Title line
	titlePadding := (boxWidth - len(title) - 2) / 2
	titleLine := "│ " + strings.Repeat(" ", titlePadding) + title +
		strings.Repeat(" ", boxWidth-len(title)-titlePadding-3) + "│"

	// Separator
	separator := "├" + strings.Repeat("─", boxWidth-2) + "┤"

	// Content lines
	var contentLines []string
	for _, line := range lines {
		padding := boxWidth - len(line) - 3
		contentLine := "│ " + line + strings.Repeat(" ", padding) + "│"
		contentLines = append(contentLines, contentLine)
	}

	// Bottom border
	bottomBorder := "└" + strings.Repeat("─", boxWidth-2) + "┘"

	// Combine all parts
	result := getStyle(style) + topBorder + Reset + "\n"
	result += getStyle(style) + titleLine + Reset + "\n"
	result += getStyle(style) + separator + Reset + "\n"

	for _, line := range contentLines {
		result += getStyle(style) + line + Reset + "\n"
	}

	result += getStyle(style) + bottomBorder + Reset

	return result
}

// SimpleBox creates a simple bordered box
func SimpleBox(content string, style StyleType) string {
	lines := strings.Split(content, "\n")
	maxWidth := 0

	for _, line := range lines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}

	if maxWidth < 20 {
		maxWidth = 20
	}

	boxWidth := maxWidth + 4

	// Borders
	topBorder := "┌" + strings.Repeat("─", boxWidth-2) + "┐"
	bottomBorder := "└" + strings.Repeat("─", boxWidth-2) + "┘"

	// Content
	var result strings.Builder
	result.WriteString(getStyle(style) + topBorder + Reset + "\n")

	for _, line := range lines {
		padding := boxWidth - len(line) - 3
		contentLine := "│ " + line + strings.Repeat(" ", padding) + "│"
		result.WriteString(getStyle(style) + contentLine + Reset + "\n")
	}

	result.WriteString(getStyle(style) + bottomBorder + Reset)

	return result.String()
}

// Divider creates a horizontal divider
func Divider(title string, style StyleType) string {
	width := 60
	titleLen := len(title)

	if titleLen == 0 {
		return getStyle(style) + strings.Repeat("─", width) + Reset
	}

	sideLen := (width - titleLen - 2) / 2
	leftSide := strings.Repeat("─", sideLen)
	rightSide := strings.Repeat("─", width-titleLen-sideLen-2)

	return getStyle(style) + leftSide + " " + title + " " + rightSide + Reset
}

// Header creates a styled header
func Header(text string, style StyleType) string {
	return "\n" + Divider(text, style) + "\n"
}

// StatusIcon returns an appropriate icon for status
func StatusIcon(success bool) string {
	if success {
		return Styled("[OK]", StyleSuccess)
	}
	return Styled("[ERROR]", StyleError)
}

// DangerIcon returns an icon for danger levels
func DangerIcon(level string) string {
	switch level {
	case "safe":
		return Styled("[SAFE]", StyleSuccess)
	case "low":
		return Styled("[LOW]", StyleWarning)
	case "medium":
		return Styled("[MEDIUM]", StyleWarning)
	case "high":
		return Styled("[HIGH]", StyleDanger)
	case "critical":
		return Styled("[CRITICAL]", StyleCritical)
	default:
		return Styled("[UNKNOWN]", StyleSubtle)
	}
}

// ProgressBar creates a simple progress indicator
func ProgressBar(current, total int, width int) string {
	if total == 0 {
		return ""
	}

	filled := (current * width) / total
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	percentage := (current * 100) / total

	return fmt.Sprintf("%s %d%% (%d/%d)",
		Styled(bar, StyleInfo), percentage, current, total)
}

// List creates a formatted list
func List(items []string, style StyleType) string {
	var result strings.Builder
	for i, item := range items {
		prefix := "├─ "
		if i == len(items)-1 {
			prefix = "└─ "
		}
		result.WriteString(getStyle(style) + prefix + Reset + item + "\n")
	}
	return strings.TrimSuffix(result.String(), "\n")
}

// Table creates a simple table
func Table(headers []string, rows [][]string, style StyleType) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}

	// Calculate column widths
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	var result strings.Builder

	// Header
	result.WriteString(getStyle(style))
	for i, header := range headers {
		result.WriteString(fmt.Sprintf("%-*s", colWidths[i]+2, header))
	}
	result.WriteString(Reset + "\n")

	// Separator
	result.WriteString(getStyle(style))
	for _, width := range colWidths {
		result.WriteString(strings.Repeat("─", width+2))
	}
	result.WriteString(Reset + "\n")

	// Rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				result.WriteString(fmt.Sprintf("%-*s", colWidths[i]+2, cell))
			}
		}
		result.WriteString("\n")
	}

	return strings.TrimSuffix(result.String(), "\n")
}

// Indent adds indentation to text
func Indent(text string, level int) string {
	indent := strings.Repeat("  ", level)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

// Truncate truncates text to fit within specified width
func Truncate(text string, width int) string {
	if len(text) <= width {
		return text
	}
	if width <= 3 {
		return text[:width]
	}
	return text[:width-3] + "..."
}
