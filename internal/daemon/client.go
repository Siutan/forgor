package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

type Client struct {
	socketPath string
	timeout    time.Duration
}

func NewClient() *Client {
	return &Client{
		socketPath: DefaultDaemonConfig().SocketPath,
		timeout:    5 * time.Second,
	}
}

func (c *Client) send(msg IPCMessage) (map[string]any, error) {
	conn, err := net.DialTimeout("unix", c.socketPath, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("connect to daemon: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(c.timeout))
	enc := json.NewEncoder(conn)
	if err := enc.Encode(msg); err != nil {
		return nil, err
	}
	dec := json.NewDecoder(conn)
	var resp map[string]any
	if err := dec.Decode(&resp); err != nil {
		return nil, err
	}
	if ok, _ := resp["success"].(bool); !ok {
		if s, _ := resp["error"].(string); s != "" {
			return nil, fmt.Errorf("daemon error: %s", s)
		}
		return nil, fmt.Errorf("daemon error")
	}
	return resp, nil
}

func (c *Client) QueryLast(n int) ([]*BufferEntry, error) {
	msg := IPCMessage{Type: "query", Payload: json.RawMessage([]byte(fmt.Sprintf(`{"type":"last","data":%d}`, n)))}
	resp, err := c.send(msg)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(resp["data"])
	var entries []*BufferEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (c *Client) QueryByID(id string) (*BufferEntry, error) {
	msg := IPCMessage{Type: "query", Payload: json.RawMessage([]byte(fmt.Sprintf(`{"type":"by_id","data":"%s"}`, id)))}
	resp, err := c.send(msg)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(resp["data"])
	var entry BufferEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (c *Client) Capture(id string) (string, error) {
	msg := IPCMessage{Type: "capture_request", Payload: json.RawMessage([]byte(fmt.Sprintf(`{"id":"%s"}`, id)))}
	resp, err := c.send(msg)
	if err != nil {
		return "", err
	}
	if p, ok := resp["data"].(map[string]any); ok {
		if s, _ := p["path"].(string); s != "" {
			return s, nil
		}
	}
	return "", fmt.Errorf("no path returned")
}

func EnsureDaemonRunning() error {
	c := NewClient()
	// quick check
	if conn, err := net.DialTimeout("unix", c.socketPath, 200*time.Millisecond); err == nil {
		_ = conn.Close()
		return nil
	}

	// start daemon
	bin := os.Args[0]
	if _, err := os.StartProcess(bin, []string{bin, "daemon", "start"}, &os.ProcAttr{Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}}); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// poll for socket readiness up to ~5s
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if conn, err := net.DialTimeout("unix", c.socketPath, 200*time.Millisecond); err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("daemon did not start (socket not available at %s)", c.socketPath)
}
