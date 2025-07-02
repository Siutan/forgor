package utils

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{time.Nanosecond * 500, "500ns"},
		{time.Microsecond * 1500, "1.5ms"},
		{time.Millisecond * 250, "250.0ms"},
		{time.Second * 2, "2.00s"},
	}

	for _, test := range tests {
		result := formatDuration(test.input)
		if result != test.expected {
			t.Errorf("formatDuration(%v) = %s; want %s", test.input, result, test.expected)
		}
	}
}

func TestGetStepIcon(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"config loading", "⚙️"},
		{"system context", "🔍"},
		{"llm provider", "🤖"},
		{"api request", "🌐"},
		{"tool detection", "🔧"},
		{"history processing", "📚"},
		{"validation check", "🛡️"},
		{"unknown step", "📋"},
	}

	for _, test := range tests {
		result := getStepIcon(test.input)
		if result != test.expected {
			t.Errorf("getStepIcon(%s) = %s; want %s", test.input, result, test.expected)
		}
	}
}

func TestNewTimer(t *testing.T) {
	timer := NewTimer("test", false)

	if timer == nil {
		t.Error("NewTimer should not return nil")
	}

	if timer.name != "test" {
		t.Errorf("Expected timer name 'test', got '%s'", timer.name)
	}

	if timer.verbose != false {
		t.Error("Expected timer verbose to be false")
	}

	if len(timer.steps) != 0 {
		t.Error("Expected empty steps slice")
	}
}

func TestTimerMeasureOperation(t *testing.T) {
	executed := false

	duration := MeasureOperation("test operation", false, func() {
		executed = true
		time.Sleep(time.Millisecond * 10) // Small delay for measurable duration
	})

	if !executed {
		t.Error("Operation should have been executed")
	}

	if duration < time.Millisecond*5 {
		t.Error("Duration should be at least 5ms")
	}
}

func TestMeasureOperationWithResult(t *testing.T) {
	expectedResult := "test result"

	result, duration := MeasureOperationWithResult("test operation", false, func() string {
		time.Sleep(time.Millisecond * 10)
		return expectedResult
	})

	if result != expectedResult {
		t.Errorf("Expected result '%s', got '%s'", expectedResult, result)
	}

	if duration < time.Millisecond*5 {
		t.Error("Duration should be at least 5ms")
	}
}

func TestMeasureOperationWithError(t *testing.T) {
	// Test successful operation
	duration, err := MeasureOperationWithError("test operation", false, func() error {
		time.Sleep(time.Millisecond * 10)
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if duration < time.Millisecond*5 {
		t.Error("Duration should be at least 5ms")
	}

	// Test operation with error
	expectedError := "test error"
	duration, err = MeasureOperationWithError("test operation", false, func() error {
		return &testError{msg: expectedError}
	})

	if err == nil {
		t.Error("Expected an error")
	}

	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// Helper type for testing errors
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
