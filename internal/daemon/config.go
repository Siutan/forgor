package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config types for the daemon subsystem. These are local to the daemon package
// and can be mapped from the top-level config when wiring up the daemon.

type DaemonConfig struct {
	AutoStart   bool
	IdleTimeout time.Duration
	LogPath     string
	SocketPath  string
}

type BufferLayerConfig struct {
	Buffer BufferConfig
}

func DefaultDaemonConfig() DaemonConfig {
	base := userConfigDir()
	return DaemonConfig{
		AutoStart:   true,
		IdleTimeout: 30 * time.Minute,
		LogPath:     filepath.Join(base, "daemon.log"),
		SocketPath:  filepath.Join(base, "daemon.sock"),
	}
}

func DefaultBufferConfig() BufferConfig {
	return BufferConfig{
		MaxEntries:         50,
		MaxSizeMB:          1,
		TTL:                15 * time.Minute,
		PrioritizeFailures: true,
	}
}

func userConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "/tmp/forgor"
	}
	dir := filepath.Join(home, ".config", "forgor")
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

func outputsDir() string {
	dir := filepath.Join(userConfigDir(), "outputs")
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

func ensureFileParent(path string) error {
	d := filepath.Dir(path)
	if err := os.MkdirAll(d, 0o700); err != nil {
		return fmt.Errorf("failed to create dir %s: %w", d, err)
	}
	return nil
}
