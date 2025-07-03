package tests

import (
	"forgor/internal/llm"
	"testing"
)

func TestDangerLevel(t *testing.T) {
	tests := []struct {
		level llm.DangerLevel
		value int
	}{
		{llm.DangerLevelSafe, 0},
		{llm.DangerLevelLow, 1},
		{llm.DangerLevelMedium, 2},
		{llm.DangerLevelHigh, 3},
		{llm.DangerLevelCritical, 4},
	}

	for _, test := range tests {
		result := llm.GetDangerLevelValue(test.level)
		if result != test.value {
			t.Errorf("GetDangerLevelValue(%v) = %d; want %d", test.level, result, test.value)
		}
	}
}

func TestDangerLevelComparison(t *testing.T) {
	// Test that danger levels have the expected relative values
	if llm.GetDangerLevelValue(llm.DangerLevelSafe) >= llm.GetDangerLevelValue(llm.DangerLevelLow) {
		t.Error("Safe should be less dangerous than Low")
	}

	if llm.GetDangerLevelValue(llm.DangerLevelLow) >= llm.GetDangerLevelValue(llm.DangerLevelMedium) {
		t.Error("Low should be less dangerous than Medium")
	}

	if llm.GetDangerLevelValue(llm.DangerLevelMedium) >= llm.GetDangerLevelValue(llm.DangerLevelHigh) {
		t.Error("Medium should be less dangerous than High")
	}

	if llm.GetDangerLevelValue(llm.DangerLevelHigh) >= llm.GetDangerLevelValue(llm.DangerLevelCritical) {
		t.Error("High should be less dangerous than Critical")
	}
}

func TestError(t *testing.T) {
	err := &llm.Error{
		Type:    llm.ErrorTypeNetwork,
		Message: "test error",
	}

	errorString := err.Error()
	if errorString != "test error" {
		t.Errorf("Error() = %s; want 'test error'", errorString)
	}
}

func TestErrorWithCause(t *testing.T) {
	causeErr := &providerTestError{msg: "underlying error"}
	err := &llm.Error{
		Type:    llm.ErrorTypeNetwork,
		Message: "test error",
		Cause:   causeErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != causeErr {
		t.Errorf("Unwrap() did not return the correct cause error")
	}
}

func TestUsage(t *testing.T) {
	usage := &llm.Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	if usage.TotalTokens != usage.PromptTokens+usage.CompletionTokens {
		t.Errorf("TotalTokens should equal PromptTokens + CompletionTokens")
	}
}

func TestResponse(t *testing.T) {
	response := &llm.Response{
		Command:     "ls -la",
		Explanation: "List files with details",
		Confidence:  0.95,
		DangerLevel: llm.DangerLevelSafe,
		Warnings:    []string{},
		Usage: &llm.Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}

	if response.Command == "" {
		t.Error("Response Command should not be empty")
	}

	if response.Confidence < 0 || response.Confidence > 1 {
		t.Errorf("Response Confidence should be between 0 and 1, got %f", response.Confidence)
	}

	if response.DangerLevel != llm.DangerLevelSafe {
		t.Errorf("Expected DangerLevelSafe, got %v", response.DangerLevel)
	}
}

// Helper type for testing errors
type providerTestError struct {
	msg string
}

func (e *providerTestError) Error() string {
	return e.msg
}
