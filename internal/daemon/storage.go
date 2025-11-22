package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type Storage struct {
	baseDir    string
	bufferFile string
	lockFile   string
	mu         sync.Mutex
}

func NewStorage() *Storage {
	base := outputsDir()
	return &Storage{
		baseDir:    base,
		bufferFile: filepath.Join(base, "buffer.json"),
		lockFile:   filepath.Join(base, "buffer.json.lock"),
	}
}

func (s *Storage) SaveBuffer(rb *RollingBuffer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lock, err := os.OpenFile(s.lockFile, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

	snapshot := struct {
		Version string         `json:"version"`
		SavedAt time.Time      `json:"saved_at"`
		Entries []*BufferEntry `json:"entries"`
		Stats   BufferStats    `json:"stats"`
	}{
		Version: "1.0",
		SavedAt: time.Now(),
		Entries: rb.GetLast(0),
		Stats:   rb.Stats(),
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	if err := ensureFileParent(s.bufferFile); err != nil {
		return err
	}

	tmp := s.bufferFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.bufferFile); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (s *Storage) LoadBuffer(rb *RollingBuffer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := os.Stat(s.bufferFile); err != nil {
		return nil
	}
	data, err := os.ReadFile(s.bufferFile)
	if err != nil {
		return err
	}
	var snapshot struct {
		Version string         `json:"version"`
		Entries []*BufferEntry `json:"entries"`
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return err
	}
	// restore entries without validation, prune will clean up
	for _, e := range snapshot.Entries {
		_ = rb.Add(e)
	}
	rb.Prune()
	return nil
}

func (s *Storage) WriteOutput(id, content string) (string, error) {
	path := filepath.Join(s.baseDir, fmt.Sprintf("output-%s.txt", id))
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Storage) ReadOutput(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
