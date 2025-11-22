package daemon

import (
	"context"
	"fmt"
	"time"
)

// PTYCapture is a stub for now; full PTY integration will be added later.
type PTYCapture struct{}

func (p *PTYCapture) Name() string                     { return "pty" }
func (p *PTYCapture) IsAvailable() bool                { return false }
func (p *PTYCapture) EstimatedOverhead() time.Duration { return 0 }
func (p *PTYCapture) CaptureOutput(ctx context.Context, entry *BufferEntry) (*CaptureResult, error) {
	return nil, fmt.Errorf("pty capture not available")
}
