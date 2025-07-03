package tests

import (
	"testing"
	"time"

	"forgor/internal/utils"
)

func TestNewTimer(t *testing.T) {
	timer := utils.NewTimer("test", false)

	if timer == nil {
		t.Error("NewTimer should not return nil")
	}

	// Test the timer by getting its summary
	summary := timer.GetSummary()
	if len(summary.Steps) != 0 {
		t.Error("Expected empty steps slice for new timer")
	}

	if summary.TotalDuration < 0 {
		t.Error("Expected non-negative total duration")
	}
}

func TestTimerAddStep(t *testing.T) {
	timer := utils.NewTimer("test", false)

	// Add a step manually
	startTime := time.Now()
	duration := time.Millisecond * 100
	timer.AddStep("test step", duration, startTime)

	summary := timer.GetSummary()
	if len(summary.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(summary.Steps))
	}

	if summary.Steps[0].Name != "test step" {
		t.Errorf("Expected step name 'test step', got '%s'", summary.Steps[0].Name)
	}

	if summary.Steps[0].Duration != duration {
		t.Errorf("Expected duration %v, got %v", duration, summary.Steps[0].Duration)
	}
}

func TestTimerMeasureOperation(t *testing.T) {
	executed := false

	duration := utils.MeasureOperation("test operation", false, func() {
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

	result, duration := utils.MeasureOperationWithResult("test operation", false, func() string {
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
	duration, err := utils.MeasureOperationWithError("test operation", false, func() error {
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
	duration, err = utils.MeasureOperationWithError("test operation", false, func() error {
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
