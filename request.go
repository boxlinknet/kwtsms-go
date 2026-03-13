package kwtsms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://www.kwtsms.com/API/"

var httpClient = &http.Client{
	Timeout: 15 * time.Second,
}

// request POSTs JSON to a kwtSMS API endpoint and returns the parsed response.
// It reads 4xx/5xx response bodies (kwtSMS returns JSON error details in 403s).
// Logs every call to logFile if provided. Password is always masked in logs.
func (c *KwtSMS) request(endpoint string, payload map[string]any) (map[string]any, error) {
	url := baseURL + endpoint + "/"

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	entry := logEntry{
		Endpoint: endpoint,
		Request:  maskPassword(payload),
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		entry.Error = err.Error()
		writeLog(c.logFile, entry)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	hc := httpClient
	if c.httpClient != nil {
		hc = c.httpClient
	}

	resp, err := hc.Do(req)
	if err != nil {
		entry.Error = fmt.Sprintf("Network error: %v", err)
		writeLog(c.logFile, entry)
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		entry.Error = fmt.Sprintf("Failed to read response: %v", err)
		writeLog(c.logFile, entry)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		entry.Error = fmt.Sprintf("Invalid JSON response: %v", err)
		writeLog(c.logFile, entry)
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	entry.Response = data
	result, _ := data["result"].(string)
	entry.OK = result == "OK"
	writeLog(c.logFile, entry)

	return data, nil
}
