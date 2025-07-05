package history

// HistoryEntry represents a single command from the shell's history.
type HistoryEntry struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
}
