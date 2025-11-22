package daemon

import (
	"context"
	"time"
)

type CaptureStrategy interface {
	Name() string
	IsAvailable() bool
	CaptureOutput(ctx context.Context, entry *BufferEntry) (*CaptureResult, error)
	EstimatedOverhead() time.Duration
}

type CaptureResult struct {
	Stdout      string
	Stderr      string
	Combined    string
	ExitCode    int
	Duration    time.Duration
	TruncatedAt int
}
