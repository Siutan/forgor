package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type StderrOnlyCapture struct {
	TempDir         string
	MaxOutputSizeKB int
}

func (s *StderrOnlyCapture) Name() string                     { return "stderr-only" }
func (s *StderrOnlyCapture) IsAvailable() bool                { return true }
func (s *StderrOnlyCapture) EstimatedOverhead() time.Duration { return 10 * time.Millisecond }

func (s *StderrOnlyCapture) CaptureOutput(ctx context.Context, entry *BufferEntry) (*CaptureResult, error) {
	dir := s.TempDir
	if dir == "" {
		dir = outputsDir()
	}
	path := filepath.Join(dir, fmt.Sprintf("forgor-stderr-%s", entry.ID))
	// Read if exists
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &CaptureResult{Combined: "[stderr capture not available]", ExitCode: entry.ExitCode}, nil
		}
		return nil, err
	}
	_ = os.Remove(path)
	combined := string(b)
	max := s.MaxOutputSizeKB * 1024
	truncatedAt := 0
	if max > 0 && len(combined) > max {
		truncatedAt = max
		combined = combined[:max]
	}
	return &CaptureResult{Combined: combined, ExitCode: entry.ExitCode, TruncatedAt: truncatedAt}, nil
}
