package llm

import (
	"testing"
)

func TestDangerLevel(t *testing.T) {
	tests := []struct {
		level DangerLevel
		value int
	}{
		{DangerLevelSafe, 0},
		{DangerLevelLow, 1},
		{DangerLevelMedium, 2},
		{DangerLevelHigh, 3},
		{DangerLevelCritical, 4},
	}

	for _, test := range tests {
		result := GetDangerLevelValue(test.level)
		if result != test.value {
			t.Errorf("GetDangerLevelValue(%v) = %d; want %d", test.level, result, test.value)
		}
	}
}

func TestDangerLevelComparison(t *testing.T) {
	// Test that danger levels have the expected relative values
	if GetDangerLevelValue(DangerLevelSafe) >= GetDangerLevelValue(DangerLevelLow) {
		t.Error("Safe should be less dangerous than Low")
	}

	if GetDangerLevelValue(DangerLevelLow) >= GetDangerLevelValue(DangerLevelMedium) {
		t.Error("Low should be less dangerous than Medium")
	}

	if GetDangerLevelValue(DangerLevelMedium) >= GetDangerLevelValue(DangerLevelHigh) {
		t.Error("Medium should be less dangerous than High")
	}

	if GetDangerLevelValue(DangerLevelHigh) >= GetDangerLevelValue(DangerLevelCritical) {
		t.Error("High should be less dangerous than Critical")
	}
}

func TestError(t *testing.T) {
	err := &Error{
		Type:    ErrorTypeNetwork,
		Message: "test error",
	}

	errorString := err.Error()
	if errorString != "test error" {
		t.Errorf("Error() = %s; want 'test error'", errorString)
	}
}

func TestErrorWithCause(t *testing.T) {
	causeErr := &testError{msg: "underlying error"}
	err := &Error{
		Type:    ErrorTypeNetwork,
		Message: "test error",
		Cause:   causeErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != causeErr {
		t.Errorf("Unwrap() did not return the correct cause error")
	}
}

func TestUsage(t *testing.T) {
	usage := &Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	if usage.TotalTokens != usage.PromptTokens+usage.CompletionTokens {
		t.Errorf("TotalTokens should equal PromptTokens + CompletionTokens")
	}
}

func TestResponse(t *testing.T) {
	response := &Response{
		Command:     "ls -la",
		Explanation: "List files with details",
		Confidence:  0.95,
		DangerLevel: DangerLevelSafe,
		Warnings:    []string{},
		Usage: &Usage{
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

	if response.DangerLevel != DangerLevelSafe {
		t.Errorf("Expected DangerLevelSafe, got %v", response.DangerLevel)
	}
}

// Helper type for testing errors
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
