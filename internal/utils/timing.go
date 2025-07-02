package utils

import (
	"fmt"
	"strings"
	"time"
)

// Timer represents a timing measurement tool
type Timer struct {
	name      string
	startTime time.Time
	steps     []TimingStep
	verbose   bool
}

// TimingStep represents a single measured operation
type TimingStep struct {
	Name     string        `json:"name"`
	Duration time.Duration `json:"duration"`
	Started  time.Time     `json:"started"`
	Ended    time.Time     `json:"ended"`
}

// TimingSummary contains overall timing information
type TimingSummary struct {
	TotalDuration time.Duration `json:"total_duration"`
	Steps         []TimingStep  `json:"steps"`
	Started       time.Time     `json:"started"`
	Ended         time.Time     `json:"ended"`
}

// NewTimer creates a new timer with the given name
func NewTimer(name string, verbose bool) *Timer {
	return &Timer{
		name:      name,
		startTime: time.Now(),
		steps:     make([]TimingStep, 0),
		verbose:   verbose,
	}
}

// StartStep begins timing a specific operation
func (t *Timer) StartStep(stepName string) *StepTimer {
	return &StepTimer{
		timer:     t,
		stepName:  stepName,
		startTime: time.Now(),
	}
}

// AddStep records a completed timing step
func (t *Timer) AddStep(name string, duration time.Duration, started time.Time) {
	step := TimingStep{
		Name:     name,
		Duration: duration,
		Started:  started,
		Ended:    started.Add(duration),
	}

	t.steps = append(t.steps, step)

	if t.verbose {
		icon := getStepIcon(name)
		fmt.Printf("⏱️  %s %s: %v\n", icon, name, formatDuration(duration))
	}
}

// GetSummary returns a summary of all timing measurements
func (t *Timer) GetSummary() TimingSummary {
	endTime := time.Now()
	totalDuration := endTime.Sub(t.startTime)

	return TimingSummary{
		TotalDuration: totalDuration,
		Steps:         t.steps,
		Started:       t.startTime,
		Ended:         endTime,
	}
}

// PrintSummary prints the timing summary
func (t *Timer) PrintSummary() {
	if !t.verbose {
		return
	}

	totalDuration := time.Since(t.startTime)

	fmt.Printf("\n%s\n", Divider("TIMING SUMMARY", StyleInfo))

	headers := []string{"Step", "Duration", "Percentage"}
	var rows [][]string

	for _, step := range t.steps {
		percentage := float64(step.Duration.Nanoseconds()) / float64(totalDuration.Nanoseconds()) * 100

		// Format duration
		durationStr := formatDuration(step.Duration)

		// Format percentage
		percentageStr := fmt.Sprintf("%.1f%%", percentage)

		// Format step name with styling based on content
		statusStr := step.Name
		if strings.Contains(strings.ToLower(step.Name), "success") {
			statusStr = Styled(step.Name, StyleSuccess)
		} else if strings.Contains(strings.ToLower(step.Name), "error") {
			statusStr = Styled(step.Name, StyleError)
		} else {
			statusStr = step.Name
		}

		rows = append(rows, []string{statusStr, durationStr, percentageStr})
	}

	// Add total row
	rows = append(rows, []string{
		Styled("Total", StyleHighlight),
		Styled(formatDuration(totalDuration), StyleHighlight),
		Styled("100.0%", StyleHighlight),
	})

	// Print table
	fmt.Printf("%s\n", Table(headers, rows, StyleInfo))

	// Add performance tips
	if totalDuration > 10*time.Second {
		fmt.Printf("\n%s Command took longer than usual. Check your network connection.\n",
			Styled("[TIP]", StyleWarning))
	} else if totalDuration > 5*time.Second {
		fmt.Printf("\n%s Consider using cache or optimizing your query.\n",
			Styled("[TIP]", StyleInfo))
	}
}

// StepTimer represents an active timing measurement
type StepTimer struct {
	timer     *Timer
	stepName  string
	startTime time.Time
}

// End completes the timing measurement for this step
func (st *StepTimer) End() {
	duration := time.Since(st.startTime)
	st.timer.AddStep(st.stepName, duration, st.startTime)
}

// EndWithResult completes the timing measurement and logs a result
func (st *StepTimer) EndWithResult(result string) {
	duration := time.Since(st.startTime)
	stepName := fmt.Sprintf("%s (%s)", st.stepName, result)
	st.timer.AddStep(stepName, duration, st.startTime)
}

// Helper functions

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.1fμs", float64(d.Nanoseconds())/1000)
	} else if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1000000)
	} else {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

// getStepIcon returns an appropriate icon for different step types
func getStepIcon(stepName string) string {
	stepName = strings.ToLower(stepName)

	switch {
	case strings.Contains(stepName, "config"):
		return "⚙️"
	case strings.Contains(stepName, "context") || strings.Contains(stepName, "system"):
		return "🔍"
	case strings.Contains(stepName, "provider") || strings.Contains(stepName, "llm"):
		return "🤖"
	case strings.Contains(stepName, "api") || strings.Contains(stepName, "request"):
		return "🌐"
	case strings.Contains(stepName, "tool") || strings.Contains(stepName, "detect"):
		return "🔧"
	case strings.Contains(stepName, "history"):
		return "📚"
	case strings.Contains(stepName, "validation") || strings.Contains(stepName, "safety"):
		return "🛡️"
	default:
		return "📋"
	}
}

// MeasureOperation is a convenience function to measure a single operation
func MeasureOperation(name string, verbose bool, operation func()) time.Duration {
	start := time.Now()
	operation()
	duration := time.Since(start)

	if verbose {
		icon := getStepIcon(name)
		fmt.Printf("⏱️  %s %s: %v\n", icon, name, formatDuration(duration))
	}

	return duration
}

// MeasureOperationWithResult measures an operation that returns a result
func MeasureOperationWithResult[T any](name string, verbose bool, operation func() T) (T, time.Duration) {
	start := time.Now()
	result := operation()
	duration := time.Since(start)

	if verbose {
		icon := getStepIcon(name)
		fmt.Printf("⏱️  %s %s: %v\n", icon, name, formatDuration(duration))
	}

	return result, duration
}

// MeasureOperationWithError measures an operation that can return an error
func MeasureOperationWithError(name string, verbose bool, operation func() error) (time.Duration, error) {
	start := time.Now()
	err := operation()
	duration := time.Since(start)

	if verbose {
		icon := getStepIcon(name)
		status := "✅"
		if err != nil {
			status = "❌"
		}
		fmt.Printf("⏱️  %s %s %s: %v\n", icon, status, name, formatDuration(duration))
	}

	return duration, err
}
