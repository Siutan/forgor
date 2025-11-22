package daemon

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type Daemon struct {
	config   DaemonConfig
	buffer   *RollingBuffer
	ipc      *IPCServer
	capturer CaptureStrategy
	storage  *Storage
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.RWMutex
	lastUse  time.Time
	pidFile  string
}

func NewDaemon() *Daemon {
	d := &Daemon{
		config:  DefaultDaemonConfig(),
		buffer:  NewRollingBuffer(DefaultBufferConfig()),
		storage: NewStorage(),
		pidFile: filepath.Join(userConfigDir(), "daemon.pid"),
	}
	d.capturer = &NoopCapture{}
	return d
}

func (d *Daemon) touchActivity() {
	d.mu.Lock()
	d.lastUse = time.Now()
	d.mu.Unlock()
}

func (d *Daemon) Start() error {
	if d.isRunning() {
		return fmt.Errorf("daemon already running")
	}
	// Spawn a detached child via CLI subcommand: `forgor daemon child`
	cmd := exec.Command(os.Args[0], "daemon", "child")
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

// RunChild runs the daemon in the foreground (invoked by `forgor daemon child`).
func (d *Daemon) RunChild() error {
	// Load persisted buffer
	_ = d.storage.LoadBuffer(d.buffer)

	// Detach from controlling terminal session
	_, _ = syscall.Setsid()
	_ = os.Chdir("/")

	// Redirect stdout/stderr to log file
	if err := ensureFileParent(d.config.LogPath); err == nil {
		if f, err := os.OpenFile(d.config.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600); err == nil {
			_ = syscall.Dup2(int(f.Fd()), int(os.Stdout.Fd()))
			_ = syscall.Dup2(int(f.Fd()), int(os.Stderr.Fd()))
		}
	}

	// Write PID file
	if err := os.WriteFile(d.pidFile, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0o600); err != nil {
		return err
	}

	d.ctx, d.cancel = context.WithCancel(context.Background())
	d.ipc = NewIPCServer(d.config.SocketPath, d)
	if err := d.ipc.Start(); err != nil {
		return err
	}

	idleTicker := time.NewTicker(1 * time.Minute)
	defer idleTicker.Stop()

	d.touchActivity()
	for {
		select {
		case <-d.ctx.Done():
			d.shutdown()
			return nil
		case <-idleTicker.C:
			if d.config.IdleTimeout > 0 {
				d.mu.RLock()
				idle := time.Since(d.lastUse)
				d.mu.RUnlock()
				if idle > d.config.IdleTimeout {
					d.shutdown()
					return nil
				}
			}
		}
	}
}

func (d *Daemon) shutdown() {
	if d.ipc != nil {
		d.ipc.Stop()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		d.ipc.Wait(ctx)
		cancel()
	}
	_ = d.storage.SaveBuffer(d.buffer)
	_ = os.Remove(d.pidFile)
}

func (d *Daemon) Stop() error {
	pid, err := d.readPID()
	if err != nil {
		return fmt.Errorf("daemon not running")
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := p.Signal(syscall.SIGTERM); err != nil {
		return err
	}
	for i := 0; i < 50; i++ {
		if !d.isRunning() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return p.Kill()
}

func (d *Daemon) isRunning() bool {
	pid, err := d.readPID()
	if err != nil {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if err := p.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	return true
}

func (d *Daemon) readPID() (int, error) {
	b, err := os.ReadFile(d.pidFile)
	if err != nil {
		return 0, err
	}
	var pid int
	_, err = fmt.Sscanf(string(b), "%d", &pid)
	if err != nil {
		return 0, err
	}
	return pid, nil
}

// selectStrategy picks a capture strategy based on availability and safety.
func (d *Daemon) selectStrategy(entry *BufferEntry) CaptureStrategy {
	// PTY disabled for now; placeholder for session-aware logic
	pty := &PTYCapture{}
	if pty.IsAvailable() {
		return pty
	}
	rerun := NewDefaultRerunCapture()
	if rerun.IsAvailable() && isSafeToRerun(entry.Command) {
		return rerun
	}
	return &StderrOnlyCapture{MaxOutputSizeKB: 100}
}

// NoopCapture implements CaptureStrategy for early wiring; replaced later.
type NoopCapture struct{}

func (n *NoopCapture) Name() string                     { return "noop" }
func (n *NoopCapture) IsAvailable() bool                { return true }
func (n *NoopCapture) EstimatedOverhead() time.Duration { return 0 }
func (n *NoopCapture) CaptureOutput(ctx context.Context, e *BufferEntry) (*CaptureResult, error) {
	return &CaptureResult{Combined: "", ExitCode: e.ExitCode}, nil
}
