package daemon

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// BufferEntry represents a single command execution record.
type BufferEntry struct {
	ID                string    `json:"id"`
	SessionID         string    `json:"session_id"`
	Command           string    `json:"command"`
	ExitCode          int       `json:"exit_code"`
	Timestamp         time.Time `json:"timestamp"`
	Duration          int64     `json:"duration_ns"`
	CWD               string    `json:"cwd"`
	Shell             string    `json:"shell"`
	ShellPID          int       `json:"shell_pid"`
	OutputPath        string    `json:"output_path,omitempty"`
	OutputSize        int64     `json:"output_size"`
	CapturedAt        time.Time `json:"captured_at,omitempty"`
	Sanitized         bool      `json:"sanitized"`
	ContainsSensitive bool      `json:"contains_sensitive"`
	CreatedAt         time.Time `json:"created_at"`
	AccessedAt        time.Time `json:"accessed_at"`
	ExpiresAt         time.Time `json:"expires_at"`
}

type RollingBuffer struct {
	entries            []*BufferEntry
	index              map[string]*BufferEntry
	maxEntries         int
	maxSizeBytes       int64
	currentSize        int64
	ttl                time.Duration
	prioritizeFailures bool
	mu                 sync.RWMutex
}

type BufferConfig struct {
	MaxEntries         int
	MaxSizeMB          int
	TTL                time.Duration
	PrioritizeFailures bool
}

func NewRollingBuffer(cfg BufferConfig) *RollingBuffer {
	return &RollingBuffer{
		entries:            make([]*BufferEntry, 0, cfg.MaxEntries),
		index:              make(map[string]*BufferEntry),
		maxEntries:         cfg.MaxEntries,
		maxSizeBytes:       int64(cfg.MaxSizeMB) * 1024 * 1024,
		ttl:                cfg.TTL,
		prioritizeFailures: cfg.PrioritizeFailures,
	}
}

func (rb *RollingBuffer) Add(entry *BufferEntry) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if entry.ID == "" {
		return fmt.Errorf("entry ID is required")
	}
	if _, ok := rb.index[entry.ID]; ok {
		return fmt.Errorf("entry with ID %s already exists", entry.ID)
	}

	now := time.Now()
	entry.CreatedAt = now
	if entry.Timestamp.IsZero() {
		entry.Timestamp = now
	}
	if rb.ttl > 0 {
		entry.ExpiresAt = entry.Timestamp.Add(rb.ttl)
	}

	rb.entries = append(rb.entries, entry)
	rb.index[entry.ID] = entry
	rb.currentSize += entry.OutputSize

	rb.pruneLocked()
	return nil
}

func (rb *RollingBuffer) Update(id string, fn func(*BufferEntry)) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	e, ok := rb.index[id]
	if !ok {
		return fmt.Errorf("entry %s not found", id)
	}
	oldSize := e.OutputSize
	fn(e)
	rb.currentSize = rb.currentSize - oldSize + e.OutputSize
	rb.pruneLocked()
	return nil
}

func (rb *RollingBuffer) GetLast(n int) []*BufferEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	if n <= 0 || n > len(rb.entries) {
		n = len(rb.entries)
	}
	start := len(rb.entries) - n
	out := make([]*BufferEntry, 0, n)
	for i := start; i < len(rb.entries); i++ {
		e := rb.entries[i]
		e.AccessedAt = time.Now()
		out = append(out, e)
	}
	return out
}

func (rb *RollingBuffer) GetLastFailed() *BufferEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	for i := len(rb.entries) - 1; i >= 0; i-- {
		if rb.entries[i].ExitCode != 0 {
			rb.entries[i].AccessedAt = time.Now()
			return rb.entries[i]
		}
	}
	return nil
}

func (rb *RollingBuffer) GetByID(id string) (*BufferEntry, error) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	e, ok := rb.index[id]
	if !ok {
		return nil, fmt.Errorf("entry %s not found", id)
	}
	e.AccessedAt = time.Now()
	return e, nil
}

func (rb *RollingBuffer) Search(filter func(*BufferEntry) bool) []*BufferEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	var out []*BufferEntry
	for _, e := range rb.entries {
		if filter(e) {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out
}

func (rb *RollingBuffer) Prune() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.pruneLocked()
}

func (rb *RollingBuffer) pruneLocked() {
	now := time.Now()
	if rb.ttl > 0 {
		filtered := rb.entries[:0]
		for _, e := range rb.entries {
			if !e.ExpiresAt.IsZero() && now.After(e.ExpiresAt) {
				delete(rb.index, e.ID)
				rb.currentSize -= e.OutputSize
				continue
			}
			filtered = append(filtered, e)
		}
		rb.entries = filtered
	}

	if len(rb.entries) > rb.maxEntries {
		if rb.prioritizeFailures {
			toRemove := len(rb.entries) - rb.maxEntries
			// remove oldest successes first
			filtered := make([]*BufferEntry, 0, len(rb.entries))
			for _, e := range rb.entries {
				if toRemove > 0 && e.ExitCode == 0 {
					delete(rb.index, e.ID)
					rb.currentSize -= e.OutputSize
					toRemove--
					continue
				}
				filtered = append(filtered, e)
			}
			// if still over, trim from the front
			for len(filtered) > rb.maxEntries {
				old := filtered[0]
				delete(rb.index, old.ID)
				rb.currentSize -= old.OutputSize
				filtered = filtered[1:]
			}
			rb.entries = filtered
		} else {
			excess := len(rb.entries) - rb.maxEntries
			for i := 0; i < excess; i++ {
				old := rb.entries[i]
				delete(rb.index, old.ID)
				rb.currentSize -= old.OutputSize
			}
			rb.entries = rb.entries[excess:]
		}
	}

	for rb.currentSize > rb.maxSizeBytes && len(rb.entries) > 0 {
		// remove largest output to reduce size quickly
		maxIdx := 0
		maxSize := rb.entries[0].OutputSize
		for i := 1; i < len(rb.entries); i++ {
			if rb.entries[i].OutputSize > maxSize {
				maxIdx = i
				maxSize = rb.entries[i].OutputSize
			}
		}
		victim := rb.entries[maxIdx]
		rb.entries = append(rb.entries[:maxIdx], rb.entries[maxIdx+1:]...)
		delete(rb.index, victim.ID)
		rb.currentSize -= victim.OutputSize
	}
}

type BufferStats struct {
	TotalEntries       int   `json:"total_entries"`
	SuccessfulCommands int   `json:"successful_commands"`
	FailedCommands     int   `json:"failed_commands"`
	EntriesWithOutput  int   `json:"entries_with_output"`
	TotalSize          int64 `json:"total_size"`
}

func (rb *RollingBuffer) Stats() BufferStats {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	var s BufferStats
	s.TotalEntries = len(rb.entries)
	s.TotalSize = rb.currentSize
	for _, e := range rb.entries {
		if e.ExitCode != 0 {
			s.FailedCommands++
		} else {
			s.SuccessfulCommands++
		}
		if e.OutputPath != "" {
			s.EntriesWithOutput++
		}
	}
	return s
}
