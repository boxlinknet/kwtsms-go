package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	kwtsms "github.com/boxlinknet/kwtsms-go"
)

// --- Test helpers (same pattern as library mock_test.go) ---

type rewriteTransport struct {
	base    http.RoundTripper
	fromURL string
	toURL   string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	url := req.URL.String()
	url = strings.Replace(url, t.fromURL, t.toURL, 1)
	newReq, _ := http.NewRequest(req.Method, url, req.Body)
	newReq.Header = req.Header
	return t.base.RoundTrip(newReq)
}

const testBaseURL = "https://www.kwtsms.com/API/"

func mockHTTPClient(serverURL string) *http.Client {
	return &http.Client{
		Transport: &rewriteTransport{
			base:    http.DefaultTransport,
			fromURL: testBaseURL,
			toURL:   serverURL + "/API/",
		},
	}
}

func apiRouter(handlers map[string]http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/API/")
		path = strings.TrimSuffix(path, "/")
		if h, ok := handlers[path]; ok {
			h(w, r)
			return
		}
		http.NotFound(w, r)
	}
}

func jsonResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func newTestApp(t *testing.T, handlers map[string]http.HandlerFunc) (*app, *bytes.Buffer, *bytes.Buffer, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(apiRouter(handlers))

	hc := mockHTTPClient(server.URL)
	client, _ := kwtsms.New("testuser", "testpass",
		kwtsms.WithTestMode(true),
		kwtsms.WithLogFile(""),
		kwtsms.WithHTTPClient(hc),
	)

	var stdout, stderr bytes.Buffer
	a := &app{
		stdin:   strings.NewReader(""),
		stdout:  &stdout,
		stderr:  &stderr,
		envFile: filepath.Join(t.TempDir(), ".env"),
		newClient: func() (*kwtsms.KwtSMS, error) {
			return client, nil
		},
	}
	return a, &stdout, &stderr, server
}

// newSetupApp creates an app wired to a mock server for setup wizard tests.
func newSetupApp(t *testing.T, handlers map[string]http.HandlerFunc, input string) (*app, *bytes.Buffer, *bytes.Buffer, string, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(apiRouter(handlers))

	hc := mockHTTPClient(server.URL)
	envFile := filepath.Join(t.TempDir(), ".env")
	var stdout, stderr bytes.Buffer
	a := &app{
		stdin:     strings.NewReader(input),
		stdout:    &stdout,
		stderr:    &stderr,
		envFile:   envFile,
		extraOpts: []kwtsms.Option{kwtsms.WithHTTPClient(hc)},
	}
	return a, &stdout, &stderr, envFile, server
}

// --- Run / routing tests ---

func TestRunNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	a := &app{stdout: &stdout, stderr: &stderr, envFile: ".env"}
	code := a.run(nil)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stdout.String(), "kwtsms - kwtSMS SMS API client") {
		t.Error("expected usage output")
	}
}

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	a := &app{stdout: &stdout, stderr: &stderr, envFile: ".env"}
	code := a.run([]string{"help"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "kwtsms setup") {
		t.Error("help should mention setup command")
	}
}

func TestRunHelpFlags(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			var stdout bytes.Buffer
			a := &app{stdout: &stdout, stderr: &bytes.Buffer{}, envFile: ".env"}
			code := a.run([]string{flag})
			if code != 0 {
				t.Errorf("expected exit code 0 for %s, got %d", flag, code)
			}
		})
	}
}

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	a := &app{stdout: &stdout, stderr: &bytes.Buffer{}, envFile: ".env"}
	code := a.run([]string{"version"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), kwtsms.Version) {
		t.Errorf("version output should contain %q, got %q", kwtsms.Version, stdout.String())
	}
}

func TestRunVersionFlags(t *testing.T) {
	for _, flag := range []string{"--version", "-v"} {
		t.Run(flag, func(t *testing.T) {
			var stdout bytes.Buffer
			a := &app{stdout: &stdout, stderr: &bytes.Buffer{}, envFile: ".env"}
			code := a.run([]string{flag})
			if code != 0 {
				t.Errorf("expected exit code 0 for %s, got %d", flag, code)
			}
		})
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stderr bytes.Buffer
	a := &app{stdout: &bytes.Buffer{}, stderr: &stderr, envFile: ".env"}
	code := a.run([]string{"badcmd"})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Error("should print unknown command error")
	}
}

// --- Verify tests ---

func TestCmdVerifySuccess(t *testing.T) {
	a, stdout, _, server := newTestApp(t, map[string]http.HandlerFunc{
		"balance": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":    "OK",
				"available": 150.0,
				"purchased": 1000.0,
			})
		},
	})
	defer server.Close()

	code := a.cmdVerify()
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "Credentials OK") {
		t.Errorf("expected 'Credentials OK', got %q", out)
	}
	if !strings.Contains(out, "150.00") {
		t.Errorf("expected balance 150.00 in output, got %q", out)
	}
	if !strings.Contains(out, "1000.00") {
		t.Errorf("expected purchased 1000.00 in output, got %q", out)
	}
}

func TestCmdVerifyAuthError(t *testing.T) {
	a, _, stderr, server := newTestApp(t, map[string]http.HandlerFunc{
		"balance": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			json.NewEncoder(w).Encode(map[string]any{
				"result":      "ERROR",
				"code":        "ERR003",
				"description": "Authentication error.",
			})
		},
	})
	defer server.Close()

	code := a.cmdVerify()
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Error") {
		t.Error("expected error output on stderr")
	}
}

// --- Balance tests ---

func TestCmdBalanceSuccess(t *testing.T) {
	a, stdout, _, server := newTestApp(t, map[string]http.HandlerFunc{
		"balance": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":    "OK",
				"available": 250.0,
				"purchased": 500.0,
			})
		},
	})
	defer server.Close()

	code := a.cmdBalance()
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "250.00") {
		t.Errorf("expected 250.00 in output, got %q", out)
	}
	if !strings.Contains(out, "500.00") {
		t.Errorf("expected 500.00 in output, got %q", out)
	}
}

// --- SenderID tests ---

func TestCmdSenderIDSuccess(t *testing.T) {
	a, stdout, _, server := newTestApp(t, map[string]http.HandlerFunc{
		"senderid": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":   "OK",
				"senderid": []any{"KWT-SMS", "MY-APP"},
			})
		},
	})
	defer server.Close()

	code := a.cmdSenderID()
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "KWT-SMS") || !strings.Contains(out, "MY-APP") {
		t.Errorf("expected sender IDs in output, got %q", out)
	}
}

func TestCmdSenderIDEmpty(t *testing.T) {
	a, stdout, _, server := newTestApp(t, map[string]http.HandlerFunc{
		"senderid": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":   "OK",
				"senderid": []any{},
			})
		},
	})
	defer server.Close()

	code := a.cmdSenderID()
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "No sender IDs") {
		t.Error("expected 'No sender IDs' message")
	}
}

// --- Coverage tests ---

func TestCmdCoverageSuccess(t *testing.T) {
	a, stdout, _, server := newTestApp(t, map[string]http.HandlerFunc{
		"coverage": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result": "OK",
				"coverage": []any{
					map[string]any{"prefix": "965", "country": "Kuwait"},
				},
			})
		},
	})
	defer server.Close()

	code := a.cmdCoverage()
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "965") {
		t.Error("expected coverage data with 965")
	}
}

// --- Send tests ---

func TestCmdSendSuccess(t *testing.T) {
	a, stdout, _, server := newTestApp(t, map[string]http.HandlerFunc{
		"send": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":         "OK",
				"msg-id":         "cli-msg-123",
				"numbers":        1,
				"points-charged": 1,
				"balance-after":  99.0,
			})
		},
	})
	defer server.Close()

	code := a.cmdSend([]string{"96598765432", "Hello from CLI"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "Message sent successfully") {
		t.Errorf("expected success message, got %q", out)
	}
	if !strings.Contains(out, "cli-msg-123") {
		t.Errorf("expected msg-id in output, got %q", out)
	}
}

func TestCmdSendWithSenderFlag(t *testing.T) {
	var sentSender string
	a, _, _, server := newTestApp(t, map[string]http.HandlerFunc{
		"send": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			sentSender, _ = body["sender"].(string)
			jsonResponse(w, map[string]any{
				"result":         "OK",
				"msg-id":         "sender-test",
				"numbers":        1,
				"points-charged": 1,
				"balance-after":  98.0,
			})
		},
	})
	defer server.Close()

	a.cmdSend([]string{"96598765432", "Hello", "--sender", "MY APP"})
	if sentSender != "MY APP" {
		t.Errorf("sender = %q, want \"MY APP\"", sentSender)
	}
}

func TestCmdSendMissingArgs(t *testing.T) {
	var stderr bytes.Buffer
	a := &app{stdout: &bytes.Buffer{}, stderr: &stderr, envFile: ".env"}
	code := a.cmdSend(nil)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Usage") {
		t.Error("expected usage message on stderr")
	}
}

func TestCmdSendEmptyMessage(t *testing.T) {
	var stderr bytes.Buffer
	a := &app{stdout: &bytes.Buffer{}, stderr: &stderr, envFile: ".env"}
	code := a.cmdSend([]string{"96598765432"})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestCmdSendAPIError(t *testing.T) {
	a, _, stderr, server := newTestApp(t, map[string]http.HandlerFunc{
		"send": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":      "ERROR",
				"code":        "ERR010",
				"description": "Account balance is zero.",
			})
		},
	})
	defer server.Close()

	code := a.cmdSend([]string{"96598765432", "Hello"})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "balance is zero") {
		t.Errorf("expected error in stderr, got %q", stderr.String())
	}
}

// --- Validate tests ---

func TestCmdValidateSuccess(t *testing.T) {
	a, stdout, _, server := newTestApp(t, map[string]http.HandlerFunc{
		"validate": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result": "OK",
				"mobile": map[string]any{
					"OK": []any{"96598765432"},
					"ER": []any{},
					"NR": []any{"966558724477"},
				},
			})
		},
	})
	defer server.Close()

	code := a.cmdValidate([]string{"96598765432", "966558724477"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "96598765432") {
		t.Error("expected valid number in output")
	}
	if !strings.Contains(out, "No route") {
		t.Error("expected NR section in output")
	}
}

func TestCmdValidateCommaSeparated(t *testing.T) {
	a, stdout, _, server := newTestApp(t, map[string]http.HandlerFunc{
		"validate": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result": "OK",
				"mobile": map[string]any{
					"OK": []any{"96598765432", "96512345678"},
					"ER": []any{},
					"NR": []any{},
				},
			})
		},
	})
	defer server.Close()

	code := a.cmdValidate([]string{"96598765432,96512345678"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "96598765432") {
		t.Error("expected valid numbers in output")
	}
}

func TestCmdValidateMissingArgs(t *testing.T) {
	var stderr bytes.Buffer
	a := &app{stdout: &bytes.Buffer{}, stderr: &stderr, envFile: ".env"}
	code := a.cmdValidate(nil)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

// --- Status tests ---

func TestCmdStatusSuccess(t *testing.T) {
	a, stdout, _, server := newTestApp(t, map[string]http.HandlerFunc{
		"status": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result": "OK",
				"status": "sent",
			})
		},
	})
	defer server.Close()

	code := a.cmdStatus([]string{"msg-id-123"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "sent") {
		t.Error("expected status in output")
	}
}

func TestCmdStatusMissingArgs(t *testing.T) {
	var stderr bytes.Buffer
	a := &app{stdout: &bytes.Buffer{}, stderr: &stderr, envFile: ".env"}
	code := a.cmdStatus(nil)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

// --- DLR tests ---

func TestCmdDLRSuccess(t *testing.T) {
	a, stdout, _, server := newTestApp(t, map[string]http.HandlerFunc{
		"dlr": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result": "OK",
				"report": []any{
					map[string]any{"Number": "96598765432", "Status": "Received"},
				},
			})
		},
	})
	defer server.Close()

	code := a.cmdDLR([]string{"msg-id-456"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "Received") {
		t.Error("expected DLR data in output")
	}
}

func TestCmdDLRMissingArgs(t *testing.T) {
	var stderr bytes.Buffer
	a := &app{stdout: &bytes.Buffer{}, stderr: &stderr, envFile: ".env"}
	code := a.cmdDLR(nil)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

// --- printErrorResult tests ---

func TestPrintErrorResult(t *testing.T) {
	var stderr bytes.Buffer
	a := &app{stdout: &bytes.Buffer{}, stderr: &stderr, envFile: ".env"}

	a.printErrorResult(map[string]any{
		"description": "Something went wrong",
		"code":        "ERR999",
		"action":      "Try again later",
	})
	out := stderr.String()
	if !strings.Contains(out, "Something went wrong") {
		t.Error("expected description in output")
	}
	if !strings.Contains(out, "ERR999") {
		t.Error("expected code in output")
	}
	if !strings.Contains(out, "Try again later") {
		t.Error("expected action in output")
	}
}

func TestPrintErrorResultCodeOnly(t *testing.T) {
	var stderr bytes.Buffer
	a := &app{stdout: &bytes.Buffer{}, stderr: &stderr, envFile: ".env"}

	a.printErrorResult(map[string]any{
		"code": "ERR003",
	})
	if !strings.Contains(stderr.String(), "ERR003") {
		t.Error("expected code in output")
	}
}

// --- Setup wizard tests ---

func defaultSetupHandlers() map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"balance": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":    "OK",
				"available": 100.0,
				"purchased": 500.0,
			})
		},
		"senderid": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":   "OK",
				"senderid": []any{"KWT-SMS", "MY-APP"},
			})
		},
	}
}

func TestCmdSetupSuccess(t *testing.T) {
	// Input: username, password, pick sender ID 2, test mode default, log default
	input := "testuser\ntestpass\n2\n\n\n"
	a, stdout, _, envFile, server := newSetupApp(t, defaultSetupHandlers(), input)
	defer server.Close()

	code := a.cmdSetup()
	if code != 0 {
		t.Errorf("expected exit code 0, got %d\nstdout: %s", code, stdout.String())
	}

	data, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("failed to read .env: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "KWTSMS_USERNAME=testuser") {
		t.Errorf("expected username in .env, got:\n%s", content)
	}
	if !strings.Contains(content, "KWTSMS_PASSWORD=testpass") {
		t.Errorf("expected password in .env, got:\n%s", content)
	}
	if !strings.Contains(content, "KWTSMS_SENDER_ID=MY-APP") {
		t.Errorf("expected sender ID MY-APP (pick 2), got:\n%s", content)
	}
	if !strings.Contains(content, "KWTSMS_TEST_MODE=1") {
		t.Errorf("expected test mode 1, got:\n%s", content)
	}
	if !strings.Contains(content, "KWTSMS_LOG_FILE=kwtsms.log") {
		t.Errorf("expected default log file, got:\n%s", content)
	}
}

func TestCmdSetupSenderByName(t *testing.T) {
	// Input: username, password, type custom sender name, test mode, log default
	input := "user\npass\nCUSTOM-SENDER\n\n\n"
	a, _, _, envFile, server := newSetupApp(t, defaultSetupHandlers(), input)
	defer server.Close()

	a.cmdSetup()

	data, _ := os.ReadFile(envFile)
	if !strings.Contains(string(data), "KWTSMS_SENDER_ID=CUSTOM-SENDER") {
		t.Errorf("expected custom sender ID, got:\n%s", string(data))
	}
}

func TestCmdSetupNoSenderIDs(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"balance": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{"result": "OK", "available": 50.0})
		},
		"senderid": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{"result": "OK", "senderid": []any{}})
		},
	}
	// Input: username, password, sender default (KWT-SMS), test mode, log
	input := "user\npass\n\n\n\n"
	a, _, _, envFile, server := newSetupApp(t, handlers, input)
	defer server.Close()

	a.cmdSetup()

	data, _ := os.ReadFile(envFile)
	if !strings.Contains(string(data), "KWTSMS_SENDER_ID=KWT-SMS") {
		t.Errorf("expected default KWT-SMS, got:\n%s", string(data))
	}
}

func TestCmdSetupExistingDefaults(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"balance": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{"result": "OK", "available": 100.0})
		},
		"senderid": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{"result": "OK", "senderid": []any{}})
		},
	}
	// User presses Enter for all prompts (keep defaults)
	input := "\n\n\n\n\n"
	a, _, _, envFile, server := newSetupApp(t, handlers, input)
	defer server.Close()

	// Write existing .env
	os.WriteFile(envFile, []byte("KWTSMS_USERNAME=olduser\nKWTSMS_PASSWORD=oldpass\nKWTSMS_SENDER_ID=OLD-SENDER\nKWTSMS_TEST_MODE=0\nKWTSMS_LOG_FILE=old.log\n"), 0644)

	code := a.cmdSetup()
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	data, _ := os.ReadFile(envFile)
	content := string(data)
	if !strings.Contains(content, "KWTSMS_USERNAME=olduser") {
		t.Errorf("should keep existing username, got:\n%s", content)
	}
	if !strings.Contains(content, "KWTSMS_PASSWORD=oldpass") {
		t.Errorf("should keep existing password, got:\n%s", content)
	}
	if !strings.Contains(content, "KWTSMS_SENDER_ID=OLD-SENDER") {
		t.Errorf("should keep existing sender ID, got:\n%s", content)
	}
	// Existing mode was 0, so default should be "2" (live), pressing Enter keeps "2" → test_mode=0
	if !strings.Contains(content, "KWTSMS_TEST_MODE=0") {
		t.Errorf("should keep existing live mode, got:\n%s", content)
	}
	if !strings.Contains(content, "KWTSMS_LOG_FILE=old.log") {
		t.Errorf("should keep existing log file, got:\n%s", content)
	}
}

func TestCmdSetupLiveMode(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"balance": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{"result": "OK", "available": 50.0})
		},
		"senderid": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{"result": "OK", "senderid": []any{}})
		},
	}
	// Input: username, password, sender default, live mode (2), log default
	input := "user\npass\n\n2\n\n"
	a, stdout, _, envFile, server := newSetupApp(t, handlers, input)
	defer server.Close()

	a.cmdSetup()

	data, _ := os.ReadFile(envFile)
	if !strings.Contains(string(data), "KWTSMS_TEST_MODE=0") {
		t.Errorf("expected test mode 0 for live mode, got:\n%s", string(data))
	}
	if !strings.Contains(stdout.String(), "Live mode selected") {
		t.Error("expected live mode confirmation message")
	}
}

func TestCmdSetupLogOff(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"balance": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{"result": "OK", "available": 50.0})
		},
		"senderid": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{"result": "OK", "senderid": []any{}})
		},
	}
	// Input: username, password, sender default, test mode, log = off
	input := "user\npass\n\n1\noff\n"
	a, stdout, _, envFile, server := newSetupApp(t, handlers, input)
	defer server.Close()

	a.cmdSetup()

	data, _ := os.ReadFile(envFile)
	if !strings.Contains(string(data), "KWTSMS_LOG_FILE=\n") {
		t.Errorf("expected empty log file for 'off', got:\n%s", string(data))
	}
	if !strings.Contains(stdout.String(), "Logging disabled") {
		t.Error("expected logging disabled message")
	}
}

func TestCmdSetupCustomLogPath(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"balance": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{"result": "OK", "available": 50.0})
		},
		"senderid": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{"result": "OK", "senderid": []any{}})
		},
	}
	input := "user\npass\n\n1\ncustom.log\n"
	a, _, _, envFile, server := newSetupApp(t, handlers, input)
	defer server.Close()

	a.cmdSetup()

	data, _ := os.ReadFile(envFile)
	if !strings.Contains(string(data), "KWTSMS_LOG_FILE=custom.log") {
		t.Errorf("expected custom.log, got:\n%s", string(data))
	}
}

func TestCmdSetupNewlinesSanitized(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"balance": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{"result": "OK", "available": 50.0})
		},
		"senderid": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{"result": "OK", "senderid": []any{}})
		},
	}
	envFile := filepath.Join(t.TempDir(), ".env")
	// Pre-populate with values containing \r (simulating pasted Windows input)
	os.WriteFile(envFile, []byte("KWTSMS_USERNAME=user\rname\nKWTSMS_PASSWORD=pass\rword\n"), 0644)

	server := httptest.NewServer(apiRouter(handlers))
	defer server.Close()

	hc := mockHTTPClient(server.URL)
	// Press Enter for all to keep defaults
	input := "\n\n\n\n\n"
	var stdout bytes.Buffer
	a := &app{
		stdin:     strings.NewReader(input),
		stdout:    &stdout,
		stderr:    &bytes.Buffer{},
		envFile:   envFile,
		extraOpts: []kwtsms.Option{kwtsms.WithHTTPClient(hc)},
	}

	code := a.cmdSetup()
	if code != 0 {
		t.Errorf("expected exit code 0, got %d\nstdout: %s", code, stdout.String())
	}

	data, _ := os.ReadFile(envFile)
	content := string(data)
	if strings.Contains(content, "\r") {
		t.Errorf("expected no \\r in .env, got:\n%q", content)
	}
}

func TestCmdSetupAuthFailure(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"balance": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			json.NewEncoder(w).Encode(map[string]any{
				"result":      "ERROR",
				"code":        "ERR003",
				"description": "Authentication error.",
			})
		},
	}
	input := "baduser\nbadpass\n"
	a, _, stderr, _, server := newSetupApp(t, handlers, input)
	defer server.Close()

	code := a.cmdSetup()
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Error") {
		t.Error("expected error output for auth failure")
	}
}

func TestCmdSetupMissingUsername(t *testing.T) {
	input := "\n"
	a, _, stderr, _, server := newSetupApp(t, defaultSetupHandlers(), input)
	defer server.Close()

	code := a.cmdSetup()
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "username is required") {
		t.Error("expected username required error")
	}
}

func TestCmdSetupMissingPassword(t *testing.T) {
	input := "user\n\n"
	a, _, stderr, _, server := newSetupApp(t, defaultSetupHandlers(), input)
	defer server.Close()

	code := a.cmdSetup()
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "password is required") {
		t.Error("expected password required error")
	}
}

func TestCmdSetupFilePermissions(t *testing.T) {
	input := "user\npass\n\n1\n\n"
	a, _, _, envFile, server := newSetupApp(t, map[string]http.HandlerFunc{
		"balance": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{"result": "OK", "available": 50.0})
		},
		"senderid": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{"result": "OK", "senderid": []any{}})
		},
	}, input)
	defer server.Close()

	a.cmdSetup()

	info, err := os.Stat(envFile)
	if err != nil {
		t.Fatalf("failed to stat .env: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected permissions 0600, got %o", perm)
	}
}

// --- Auto-setup tests ---

func TestGetClientAutoSetup(t *testing.T) {
	server := httptest.NewServer(apiRouter(map[string]http.HandlerFunc{
		"balance": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{"result": "OK", "available": 50.0})
		},
		"senderid": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{"result": "OK", "senderid": []any{}})
		},
	}))
	defer server.Close()

	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	hc := mockHTTPClient(server.URL)

	// Simulate setup input: username, password, sender default, test mode, log default
	input := "autouser\nautopass\n\n1\n\n"
	var stdout bytes.Buffer

	callCount := 0
	a := &app{
		stdin:     strings.NewReader(input),
		stdout:    &stdout,
		stderr:    &bytes.Buffer{},
		envFile:   envFile,
		extraOpts: []kwtsms.Option{kwtsms.WithHTTPClient(hc)},
		newClient: func() (*kwtsms.KwtSMS, error) {
			callCount++
			if callCount == 1 {
				return nil, fmt.Errorf("missing credentials: KWTSMS_USERNAME")
			}
			return kwtsms.New("autouser", "autopass",
				kwtsms.WithLogFile(""),
				kwtsms.WithHTTPClient(hc),
			)
		},
	}

	c, code := a.getClient()
	if c == nil {
		t.Fatalf("expected client after auto-setup, got nil (code=%d)\nstdout: %s", code, stdout.String())
	}
	if code != 0 {
		t.Errorf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "No .env file found") {
		t.Error("expected auto-setup message")
	}
}

func TestGetClientNoAutoSetupWhenEnvExists(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	// Create an existing .env (but with bad creds)
	os.WriteFile(envFile, []byte("KWTSMS_USERNAME=\nKWTSMS_PASSWORD=\n"), 0644)

	var stderr bytes.Buffer
	a := &app{
		stdin:   strings.NewReader(""),
		stdout:  &bytes.Buffer{},
		stderr:  &stderr,
		envFile: envFile,
		newClient: func() (*kwtsms.KwtSMS, error) {
			return nil, fmt.Errorf("missing credentials: KWTSMS_USERNAME")
		},
	}

	c, code := a.getClient()
	if c != nil {
		t.Error("expected nil client when .env exists but creds are bad")
	}
	if code != 1 {
		t.Errorf("expected code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "missing credentials") {
		t.Error("expected error message about missing credentials")
	}
}
