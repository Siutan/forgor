package cache

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

var (
	// Global refresh state
	refreshInProgress atomic.Bool
	lastToolRefresh   atomic.Value // stores time.Time
)

// RefreshScheduler manages background cache refreshes
type RefreshScheduler struct {
	toolRefreshInterval time.Duration
	enabled             bool
	verbose             bool
}

// NewRefreshScheduler creates a new scheduler with the given interval
func NewRefreshScheduler(interval time.Duration) *RefreshScheduler {
	s := &RefreshScheduler{
		toolRefreshInterval: interval,
		enabled:             true,
		verbose:             os.Getenv("FORGOR_VERBOSE") == "true",
	}
	
	// Initialize last refresh time from cache if available
	if tools, err := LoadToolCache(); err == nil && !tools.LastScan.IsZero() {
		lastToolRefresh.Store(tools.LastScan)
	}
	
	return s
}

// MaybeRefreshTools checks if tools need refresh and triggers if needed
func (s *RefreshScheduler) MaybeRefreshTools() {
	if !s.enabled {
		return
	}

	// Check if we need to refresh
	lastRefresh := s.getLastToolRefresh()
	if !lastRefresh.IsZero() && time.Since(lastRefresh) < s.toolRefreshInterval {
		if s.verbose {
			remaining := s.toolRefreshInterval - time.Since(lastRefresh)
			fmt.Fprintf(os.Stderr, "Tool refresh skipped (next in %v)\n", remaining.Round(time.Minute))
		}
		return
	}

	// Check if already refreshing
	if !refreshInProgress.CompareAndSwap(false, true) {
		if s.verbose {
			fmt.Fprintf(os.Stderr, "Tool refresh already in progress\n")
		}
		return
	}

	// background refresh
	if s.verbose {
		fmt.Fprintf(os.Stderr, "Starting background tool refresh...\n")
	}

	go func() {
		defer refreshInProgress.Store(false)

		if err := s.performToolRefresh(); err != nil {
			if s.verbose {
				fmt.Fprintf(os.Stderr, "Background tool refresh failed: %v\n", err)
			}
		} else if s.verbose {
			fmt.Fprintf(os.Stderr, "Background tool refresh completed\n")
		}
	}()
}

// ForceRefreshTools triggers an immediate tool refresh
// This blocks until the refresh completes
// useful for manual refresh commands
func (s *RefreshScheduler) ForceRefreshTools() error {
	if s.verbose {
		fmt.Fprintf(os.Stderr, "Forcing tool refresh...\n")
	}

	start := time.Now()
	err := s.performToolRefresh()
	duration := time.Since(start)

	if err != nil {
		if s.verbose {
			fmt.Fprintf(os.Stderr, "Tool refresh failed after %v: %v\n", duration, err)
		}
		return err
	}

	if s.verbose {
		fmt.Fprintf(os.Stderr, "Tool refresh completed in %v\n", duration)
	}

	return nil
}

// performToolRefresh executes the actual tool scanning and cache update
func (s *RefreshScheduler) performToolRefresh() error {
	start := time.Now()

	tools, err := scanAllTools()
	if err != nil {
		return fmt.Errorf("tool scan failed: %w", err)
	}

	tools.LastScan = time.Now()
	tools.ScanDurationMs = time.Since(start).Milliseconds()
	tools.Version = CacheVersion

	if err := SaveToolCache(tools); err != nil {
		return fmt.Errorf("failed to save tool cache: %w", err)
	}

	s.setLastToolRefresh(time.Now())

	if s.verbose {
		fmt.Fprintf(os.Stderr, "Scanned %d tools in %dms\n", tools.GetToolCount(), tools.ScanDurationMs)
	}

	return nil
}

// getLastToolRefresh returns the timestamp of the last successful refresh
func (s *RefreshScheduler) getLastToolRefresh() time.Time {
	if t := lastToolRefresh.Load(); t != nil {
		return t.(time.Time)
	}

	// Check cache file timestamp as fallback
	if tools, err := LoadToolCache(); err == nil && !tools.LastScan.IsZero() {
		return tools.LastScan
	}

	return time.Time{} // Never refreshed
}

// setLastToolRefresh updates the last refresh timestamp
func (s *RefreshScheduler) setLastToolRefresh(t time.Time) {
	lastToolRefresh.Store(t)
}

// IsRefreshInProgress returns true if a refresh is currently running
func (s *RefreshScheduler) IsRefreshInProgress() bool {
	return refreshInProgress.Load()
}

// GetTimeSinceLastRefresh returns how long ago the last refresh occurred
func (s *RefreshScheduler) GetTimeSinceLastRefresh() time.Duration {
	lastRefresh := s.getLastToolRefresh()
	if lastRefresh.IsZero() {
		return 0 // Never refreshed
	}
	return time.Since(lastRefresh)
}

// SetEnabled enables or disables automatic background refreshes
func (s *RefreshScheduler) SetEnabled(enabled bool) {
	s.enabled = enabled
	if s.verbose {
		if enabled {
			fmt.Fprintf(os.Stderr, "Background refresh enabled\n")
		} else {
			fmt.Fprintf(os.Stderr, "Background refresh disabled\n")
		}
	}
}

// IsEnabled returns whether automatic background refreshes are enabled
func (s *RefreshScheduler) IsEnabled() bool {
	return s.enabled
}

// SetInterval updates the refresh interval
func (s *RefreshScheduler) SetInterval(interval time.Duration) {
	s.toolRefreshInterval = interval
	if s.verbose {
		fmt.Fprintf(os.Stderr, "Refresh interval set to %v\n", interval)
	}
}

// GetInterval returns the current refresh interval
func (s *RefreshScheduler) GetInterval() time.Duration {
	return s.toolRefreshInterval
}

// SetVerbose enables or disables verbose logging
func (s *RefreshScheduler) SetVerbose(verbose bool) {
	s.verbose = verbose
}

// NeedsRefresh checks if a refresh is needed based on the current interval
func (s *RefreshScheduler) NeedsRefresh() bool {
	lastRefresh := s.getLastToolRefresh()
	if lastRefresh.IsZero() {
		return true // Never refreshed
	}
	return time.Since(lastRefresh) >= s.toolRefreshInterval
}

// GetRefreshStatus returns a status of the refresh state
func (s *RefreshScheduler) GetRefreshStatus() string {
	if s.IsRefreshInProgress() {
		return "Refresh in progress..."
	}

	lastRefresh := s.getLastToolRefresh()
	if lastRefresh.IsZero() {
		return "Never refreshed"
	}

	age := time.Since(lastRefresh)
	if age < time.Minute {
		return "Just refreshed"
	} else if age < time.Hour {
		return fmt.Sprintf("Refreshed %d minutes ago", int(age.Minutes()))
	} else if age < 24*time.Hour {
		return fmt.Sprintf("Refreshed %d hours ago", int(age.Hours()))
	} else {
		days := int(age.Hours() / 24)
		if days == 1 {
			return "Refreshed 1 day ago"
		}
		return fmt.Sprintf("Refreshed %d days ago", days)
	}
}

// WaitForRefresh blocks until any in-progress refresh completes
// Times out after the specified duration
func (s *RefreshScheduler) WaitForRefresh(timeout time.Duration) bool {
	if !s.IsRefreshInProgress() {
		return true // Nothing to wait for
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !s.IsRefreshInProgress() {
			return true // Refresh completed
		}
		time.Sleep(100 * time.Millisecond)
	}

	return false // Timed out
}

// Global scheduler instance
// Should be the one that gets initialized by main package
var globalScheduler *RefreshScheduler

// InitGlobalScheduler initializes the global scheduler with the given interval
func InitGlobalScheduler(interval time.Duration) {
	globalScheduler = NewRefreshScheduler(interval)
}

// GetGlobalScheduler returns the global scheduler instance
func GetGlobalScheduler() *RefreshScheduler {
	if globalScheduler == nil {
		// Initialize with default 24h interval if not already initialized
		globalScheduler = NewRefreshScheduler(24 * time.Hour)
	}
	return globalScheduler
}

func MaybeRefresh() {
	GetGlobalScheduler().MaybeRefreshTools()
}

func ForceRefresh() error {
	return GetGlobalScheduler().ForceRefreshTools()
}

func IsRefreshInProgress() bool {
	return GetGlobalScheduler().IsRefreshInProgress()
}