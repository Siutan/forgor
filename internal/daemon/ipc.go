package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

type IPCServer struct {
	socketPath string
	listener   net.Listener
	daemon     *Daemon
	active     sync.WaitGroup
	stopCh     chan struct{}
	mu         sync.Mutex
}

type IPCMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func NewIPCServer(socketPath string, d *Daemon) *IPCServer {
	return &IPCServer{
		socketPath: socketPath,
		daemon:     d,
		stopCh:     make(chan struct{}),
	}
}

func (s *IPCServer) Start() error {
	_ = os.Remove(s.socketPath)
	l, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.socketPath, err)
	}
	if err := os.Chmod(s.socketPath, 0o600); err != nil {
		_ = l.Close()
		return err
	}
	s.listener = l
	go s.acceptLoop()
	return nil
}

func (s *IPCServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			default:
			}
			continue
		}
		s.active.Add(1)
		go func() {
			defer s.active.Done()
			_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
			var msg IPCMessage
			dec := json.NewDecoder(conn)
			if err := dec.Decode(&msg); err != nil {
				_ = conn.Close()
				return
			}
			s.daemon.touchActivity()
			resp, err := s.route(msg)
			enc := json.NewEncoder(conn)
			_ = enc.Encode(map[string]any{
				"success": err == nil,
				"data":    resp,
				"error":   errorString(err),
			})
			_ = conn.Close()
		}()
	}
}

func (s *IPCServer) route(msg IPCMessage) (any, error) {
	switch msg.Type {
	case "command_start":
		return s.handleCommandStart(msg.Payload)
	case "command_end":
		return s.handleCommandEnd(msg.Payload)
	case "query":
		return s.handleQuery(msg.Payload)
	case "capture_request":
		return s.handleCaptureRequest(msg.Payload)
	default:
		return nil, fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

func (s *IPCServer) handleCommandStart(payload json.RawMessage) (any, error) {
	var p struct {
		ID        string `json:"id"`
		Command   string `json:"command"`
		Timestamp int64  `json:"timestamp"`
		CWD       string `json:"cwd"`
		Shell     string `json:"shell"`
		ShellPID  int    `json:"shell_pid"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, err
	}
	e := &BufferEntry{
		ID:        p.ID,
		SessionID: p.SessionID,
		Command:   p.Command,
		Timestamp: time.Unix(0, p.Timestamp),
		CWD:       p.CWD,
		Shell:     p.Shell,
		ShellPID:  p.ShellPID,
		ExitCode:  -1,
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	if err := s.daemon.buffer.Add(e); err != nil {
		return nil, err
	}
	return map[string]string{"status": "recorded"}, nil
}

func (s *IPCServer) handleCommandEnd(payload json.RawMessage) (any, error) {
	var p struct {
		ID         string `json:"id"`
		ExitCode   int    `json:"exit_code"`
		DurationNS int64  `json:"duration_ns"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, err
	}
	err := s.daemon.buffer.Update(p.ID, func(e *BufferEntry) {
		e.ExitCode = p.ExitCode
		e.Duration = p.DurationNS
	})
	if err != nil {
		return nil, err
	}
	return map[string]string{"status": "updated"}, nil
}

func (s *IPCServer) handleQuery(payload json.RawMessage) (any, error) {
	var q struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &q); err != nil {
		return nil, err
	}
	switch q.Type {
	case "last":
		var n int
		_ = json.Unmarshal(q.Data, &n)
		return s.daemon.buffer.GetLast(n), nil
	case "last_failed":
		return s.daemon.buffer.GetLastFailed(), nil
	case "by_id":
		var id string
		_ = json.Unmarshal(q.Data, &id)
		return s.daemon.buffer.GetByID(id)
	case "stats":
		return s.daemon.buffer.Stats(), nil
	default:
		return nil, fmt.Errorf("unknown query type: %s", q.Type)
	}
}

func (s *IPCServer) handleCaptureRequest(payload json.RawMessage) (any, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, err
	}
	entry, err := s.daemon.buffer.GetByID(p.ID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Select strategy on-demand (lazy capture)
	strategy := s.daemon.selectStrategy(entry)
	res, err := strategy.CaptureOutput(ctx, entry)
	if err != nil {
		return nil, err
	}
	path, err := s.daemon.storage.WriteOutput(entry.ID, res.Combined)
	if err != nil {
		return nil, err
	}
	_ = s.daemon.buffer.Update(entry.ID, func(e *BufferEntry) {
		e.OutputPath = path
		e.OutputSize = int64(len(res.Combined))
		e.CapturedAt = time.Now()
	})
	return map[string]any{"status": "captured", "path": path}, nil
}

func (s *IPCServer) Stop() {
	close(s.stopCh)
	if s.listener != nil {
		_ = s.listener.Close()
	}
}

func (s *IPCServer) Wait(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		s.active.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
