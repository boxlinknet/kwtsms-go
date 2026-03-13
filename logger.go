package kwtsms

import (
	"encoding/json"
	"os"
	"time"
)

// logEntry represents a single JSONL log line for an API call.
type logEntry struct {
	Timestamp string         `json:"ts"`
	Endpoint  string         `json:"endpoint"`
	Request   map[string]any `json:"request"`
	Response  any            `json:"response"`
	OK        bool           `json:"ok"`
	Error     string         `json:"error,omitempty"`
}

// writeLog appends a JSONL log entry. Never returns an error or panics.
// Logging must never break the main flow.
func writeLog(logFile string, entry logEntry) {
	if logFile == "" {
		return
	}

	entry.Timestamp = time.Now().UTC().Format(time.RFC3339)

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	_, _ = f.Write(data)
	_, _ = f.Write([]byte("\n"))
}

// maskPassword creates a copy of the payload with the password field masked.
func maskPassword(payload map[string]any) map[string]any {
	safe := make(map[string]any, len(payload))
	for k, v := range payload {
		if k == "password" {
			safe[k] = "***"
		} else {
			safe[k] = v
		}
	}
	return safe
}
