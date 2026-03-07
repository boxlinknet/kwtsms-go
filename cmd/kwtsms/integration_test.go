//go:build integration

package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	kwtsms "github.com/boxlinknet/kwtsms-go"
)

func getIntegrationApp(t *testing.T) (*app, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	username := os.Getenv("GO_USERNAME")
	password := os.Getenv("GO_PASSWORD")
	if username == "" || password == "" {
		t.Skip("GO_USERNAME / GO_PASSWORD not set, skipping integration test")
	}

	var stdout, stderr bytes.Buffer
	a := &app{
		stdin:   strings.NewReader(""),
		stdout:  &stdout,
		stderr:  &stderr,
		envFile: "/nonexistent/.env",
		newClient: func() (*kwtsms.KwtSMS, error) {
			return kwtsms.New(username, password,
				kwtsms.WithTestMode(true),
				kwtsms.WithLogFile(""),
			)
		},
	}
	return a, &stdout, &stderr
}

// parseField extracts a numeric value from CLI output lines like "  balance-after:  1220.00"
// or "Available balance: 1670.00 credits". It takes the first whitespace-delimited token after
// the field prefix and parses it as a float.
func parseField(output, field string) (float64, bool) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, field) {
			valStr := strings.TrimSpace(strings.TrimPrefix(line, field))
			// Take only the first token (e.g. "1670.00" from "1670.00 credits")
			if idx := strings.IndexByte(valStr, ' '); idx > 0 {
				valStr = valStr[:idx]
			}
			v, err := strconv.ParseFloat(valStr, 64)
			if err == nil {
				return v, true
			}
		}
	}
	return 0, false
}

func TestCLIIntegrationBulkSend250(t *testing.T) {
	a, stdout, stderr := getIntegrationApp(t)

	// 1. Get initial balance via the verify command
	code := a.cmdVerify()
	if code != 0 {
		t.Fatalf("verify failed (code=%d): %s", code, stderr.String())
	}
	verifyOut := stdout.String()
	t.Logf("Verify output:\n%s", verifyOut)

	initialBalance, ok := parseField(verifyOut, "Available balance:")
	if !ok {
		t.Fatal("could not parse initial balance from verify output")
	}
	t.Logf("Parsed initial balance: %.2f", initialBalance)

	// 2. Probe credit rate with a single-number send
	stdout.Reset()
	stderr.Reset()

	code = a.cmdSend([]string{"96599229999", "Rate probe"})
	if code != 0 {
		t.Fatalf("rate probe send failed (code=%d): %s", code, stderr.String())
	}
	probeOut := stdout.String()
	t.Logf("Probe output:\n%s", probeOut)

	probePoints, ok := parseField(probeOut, "points-charged:")
	if !ok {
		t.Fatal("could not parse points-charged from probe output")
	}
	probeBalance, ok := parseField(probeOut, "balance-after:")
	if !ok {
		t.Fatal("could not parse balance-after from probe output")
	}

	// Verify probe balance math
	expectedProbeBalance := initialBalance - probePoints
	if probeBalance != expectedProbeBalance {
		t.Errorf("probe balance mismatch: initial(%.2f) - points(%.0f) = %.2f, but balance-after = %.2f",
			initialBalance, probePoints, expectedProbeBalance, probeBalance)
	}

	creditsPerNumber := probePoints // 1 number = probePoints credits
	t.Logf("Credit rate: %.0f credits/number", creditsPerNumber)

	// Update initial balance to after the probe
	initialBalance = probeBalance

	// 3. Calculate expected cost for 250 numbers
	const numRecipients = 250
	expectedCredits := creditsPerNumber * float64(numRecipients)
	t.Logf("Expected credits for %d numbers: %.0f (%.0f x %d)",
		numRecipients, expectedCredits, creditsPerNumber, numRecipients)

	if initialBalance < expectedCredits {
		t.Fatalf("Insufficient balance: have %.2f, need %.0f credits for %d numbers",
			initialBalance, expectedCredits, numRecipients)
	}

	// 4. Generate 250 unique numbers as comma-separated string
	stdout.Reset()
	stderr.Reset()

	numbers := make([]string, numRecipients)
	for i := range numbers {
		numbers[i] = fmt.Sprintf("9659922%04d", i)
	}
	mobile := strings.Join(numbers, ",")

	// 5. Call send via CLI — one call, triggers internal bulk (200 + 50)
	code = a.cmdSend([]string{mobile, "Go CLI bulk test 250 numbers"})
	sendOut := stdout.String()
	sendErr := stderr.String()
	t.Logf("Send stdout:\n%s", sendOut)
	if sendErr != "" {
		t.Logf("Send stderr:\n%s", sendErr)
	}

	if code != 0 {
		t.Fatalf("send failed (code=%d): %s", code, sendErr)
	}

	if !strings.Contains(sendOut, "Message sent successfully") {
		t.Error("expected 'Message sent successfully' in output")
	}
	if !strings.Contains(sendOut, "msg-id:") {
		t.Error("expected msg-id in output")
	}

	// 6. Parse and verify credit consumption
	pointsCharged, ok := parseField(sendOut, "points-charged:")
	if !ok {
		t.Fatal("could not parse points-charged from send output")
	}
	balanceAfter, ok := parseField(sendOut, "balance-after:")
	if !ok {
		t.Fatal("could not parse balance-after from send output")
	}

	// Verify points-charged matches expected
	if pointsCharged != expectedCredits {
		t.Errorf("expected %.0f points-charged (%.0f x %d), got %.0f",
			expectedCredits, creditsPerNumber, numRecipients, pointsCharged)
	}

	// Verify balance math: balance-after == initial - points-charged
	expectedBalance := initialBalance - pointsCharged
	if balanceAfter != expectedBalance {
		t.Errorf("balance mismatch: initial(%.2f) - points-charged(%.0f) = %.2f, but balance-after = %.2f",
			initialBalance, pointsCharged, expectedBalance, balanceAfter)
	}

	// 7. Extract the msg-id from output for status check
	var msgID string
	for _, line := range strings.Split(sendOut, "\n") {
		if strings.Contains(line, "msg-id:") {
			msgID = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "msg-id:"))
			break
		}
	}
	t.Logf("Extracted msg-id: %s", msgID)

	// 8. Check status — expect ERR030 for test mode
	stdout.Reset()
	stderr.Reset()

	if msgID != "" {
		code = a.cmdStatus([]string{msgID})
		statusOut := stdout.String()
		statusErr := stderr.String()
		t.Logf("Status stdout:\n%s", statusOut)
		if statusErr != "" {
			t.Logf("Status stderr:\n%s", statusErr)
		}

		// ERR030 is expected for test=1 messages stuck in queue
		if code == 1 && strings.Contains(statusErr, "ERR030") {
			t.Log("Status returned ERR030 as expected for test-mode messages")
		} else if code == 0 {
			t.Log("Status returned OK (message may have been processed)")
		} else {
			t.Logf("Status returned code=%d (may vary by timing)", code)
		}
	}
}
