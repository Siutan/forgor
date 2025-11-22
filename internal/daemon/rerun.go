package daemon

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type RerunCapture struct {
	TimeoutSeconds  int
	MaxOutputSizeKB int
	AllowRerun      bool
	SafetyCheck     bool
}

func NewDefaultRerunCapture() *RerunCapture {
	return &RerunCapture{
		TimeoutSeconds:  30,
		MaxOutputSizeKB: 100,
		AllowRerun:      true,
		SafetyCheck:     true,
	}
}

func (r *RerunCapture) Name() string { return "rerun" }
func (r *RerunCapture) IsAvailable() bool {
	return r.AllowRerun
}

func (r *RerunCapture) EstimatedOverhead() time.Duration { return time.Second }

var unsafeCommands = []string{
	" rm ", " rm-", " mv ", " dd ", " mkfs ",
	" shutdown", " reboot", " halt", " kill ", " pkill ", " killall ",
	">", ">>",
}

func isSafeToRerun(cmd string) bool {
	// crude safety heuristic: blacklist substrings
	spaced := " " + strings.ToLower(cmd) + " "
	for _, u := range unsafeCommands {
		if strings.Contains(spaced, u) {
			return false
		}
	}
	base := filepath.Base(strings.Fields(spaced)[0])
	if base == "rm" || base == "dd" || base == "mkfs" {
		return false
	}
	return true
}

func (r *RerunCapture) CaptureOutput(ctx context.Context, entry *BufferEntry) (*CaptureResult, error) {
	if r.SafetyCheck && !isSafeToRerun(entry.Command) {
		return nil, fmt.Errorf("unsafe to rerun: %s", entry.Command)
	}
	timeout := time.Duration(r.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, "sh", "-c", entry.Command)
	if entry.CWD != "" {
		cmd.Dir = entry.CWD
	}
	cmd.Env = os.Environ()
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	start := time.Now()
	err := cmd.Run()
	dur := time.Since(start)
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to execute: %w", err)
		}
	}

	combined := stdoutBuf.String() + stderrBuf.String()
	max := r.MaxOutputSizeKB * 1024
	truncatedAt := 0
	if max > 0 && len(combined) > max {
		truncatedAt = max
		combined = combined[:max]
	}

	return &CaptureResult{
		Stdout:      stdoutBuf.String(),
		Stderr:      stderrBuf.String(),
		Combined:    combined,
		ExitCode:    exitCode,
		Duration:    dur,
		TruncatedAt: truncatedAt,
	}, nil
}
