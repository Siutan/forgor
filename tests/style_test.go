package tests

import (
	"strings"
	"testing"

	"forgor/internal/utils"
)

func TestStyled(t *testing.T) {
	text := "test"
	styled := utils.Styled(text, utils.StyleSuccess)

	// Should contain the original text
	if !strings.Contains(styled, text) {
		t.Errorf("Styled text should contain original text '%s', got '%s'", text, styled)
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		success      bool
		expectedText string
	}{
		{true, "[OK]"},
		{false, "[ERROR]"},
	}

	for _, test := range tests {
		result := utils.StatusIcon(test.success)
		// The result includes styling, so we just check it contains the expected text
		if !strings.Contains(result, test.expectedText) {
			t.Errorf("StatusIcon(%v) should contain %s, got %s", test.success, test.expectedText, result)
		}
	}
}

func TestDangerIcon(t *testing.T) {
	tests := []struct {
		level        string
		expectedText string
	}{
		{"low", "[LOW]"},
		{"medium", "[MEDIUM]"},
		{"high", "[HIGH]"},
		{"critical", "[CRITICAL]"},
		{"unknown", "[UNKNOWN]"},
	}

	for _, test := range tests {
		result := utils.DangerIcon(test.level)
		// The result includes styling, so we just check it contains the expected text
		if !strings.Contains(result, test.expectedText) {
			t.Errorf("DangerIcon(%s) should contain %s, got %s", test.level, test.expectedText, result)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		text     string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a very long string", 10, "this is..."},
		{"exact", 5, "exact"},
		{"", 5, ""},
		{"test", 0, ""}, // For width <= 3, returns first width characters
	}

	for _, test := range tests {
		result := utils.Truncate(test.text, test.maxLen)
		if result != test.expected {
			t.Errorf("Truncate(%s, %d) = %s; want %s", test.text, test.maxLen, result, test.expected)
		}
	}
}

func TestIndent(t *testing.T) {
	text := "line1\nline2\nline3"
	indented := utils.Indent(text, 2)

	lines := strings.Split(indented, "\n")
	for i, line := range lines {
		if i < len(lines)-1 && !strings.HasPrefix(line, "  ") { // Skip last line if empty
			t.Errorf("Line %d should be indented with 2 spaces: '%s'", i, line)
		}
	}
}

func TestHeader(t *testing.T) {
	title := "Test Header"
	header := utils.Header(title, utils.StyleInfo)

	if !strings.Contains(header, title) {
		t.Errorf("Header should contain title '%s'", title)
	}
}

func TestSimpleBox(t *testing.T) {
	content := "test content"
	box := utils.SimpleBox(content, utils.StyleInfo)

	if !strings.Contains(box, content) {
		t.Errorf("Box should contain content '%s'", content)
	}

	// Should have box characters
	if !strings.Contains(box, "┌") || !strings.Contains(box, "└") {
		t.Error("Box should contain box drawing characters")
	}
}

func TestDivider(t *testing.T) {
	title := "Test Section"
	divider := utils.Divider(title, utils.StyleInfo)

	if !strings.Contains(divider, title) {
		t.Errorf("Divider should contain title '%s'", title)
	}
}
