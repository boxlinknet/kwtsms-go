//go:build integration

package kwtsms

import (
	"os"
	"strings"
	"testing"
)

func getIntegrationClient(t *testing.T) *KwtSMS {
	t.Helper()
	username := os.Getenv("GO_USERNAME")
	password := os.Getenv("GO_PASSWORD")
	if username == "" || password == "" {
		t.Skip("GO_USERNAME / GO_PASSWORD not set, skipping integration test")
	}
	c, err := New(username, password, WithTestMode(true), WithLogFile(""))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return c
}

func TestIntegrationVerifySuccess(t *testing.T) {
	c := getIntegrationClient(t)

	ok, balance, err := c.Verify()
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if !ok {
		t.Error("Verify should succeed with valid credentials")
	}
	if balance < 0 {
		t.Errorf("balance should be >= 0, got %f", balance)
	}
	t.Logf("Balance: %.2f", balance)
}

func TestIntegrationVerifyWrongCredentials(t *testing.T) {
	c, _ := New("go_invalid_user", "go_invalid_pass", WithTestMode(true), WithLogFile(""))

	ok, _, err := c.Verify()
	if ok {
		t.Error("Verify should fail with wrong credentials")
	}
	if err == nil {
		t.Error("expected error for wrong credentials")
	}
	if !strings.Contains(err.Error(), "KWTSMS_USERNAME") && !strings.Contains(err.Error(), "Authentication") {
		t.Logf("error message: %v", err)
	}
}

func TestIntegrationBalance(t *testing.T) {
	c := getIntegrationClient(t)

	bal, err := c.Balance()
	if err != nil {
		t.Fatalf("Balance error: %v", err)
	}
	if bal < 0 {
		t.Errorf("balance should be >= 0, got %f", bal)
	}
}

func TestIntegrationSendValidKuwaitNumber(t *testing.T) {
	c := getIntegrationClient(t)

	result, err := c.Send("96598765432", "Go client integration test", "")
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	// In test mode, we expect OK or an expected error
	t.Logf("Send result: %s (code: %s)", result.Result, result.Code)
	if result.Result == "OK" {
		if result.MsgID == "" {
			t.Error("msg-id should be set on success")
		}
		t.Logf("msg-id: %s, balance-after: %.2f", result.MsgID, result.BalanceAfter)
	}
}

func TestIntegrationSendInvalidInput(t *testing.T) {
	c := getIntegrationClient(t)

	tests := []struct {
		name   string
		mobile string
	}{
		{"email", "user@gmail.com"},
		{"too short", "123"},
		{"letters", "abcdefgh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.Send(tt.mobile, "Test message", "")
			if err != nil {
				t.Fatal(err)
			}
			if result.Result != "ERROR" {
				t.Errorf("expected ERROR for %q, got %s", tt.mobile, result.Result)
			}
			t.Logf("%s: code=%s desc=%s", tt.name, result.Code, result.Description)
		})
	}
}

func TestIntegrationSendMixedValidInvalid(t *testing.T) {
	c := getIntegrationClient(t)

	result, err := c.SendMulti(
		[]string{"96598765432", "bad@email.com", "123"},
		"Mixed input test",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Invalid) != 2 {
		t.Errorf("expected 2 invalid entries, got %d", len(result.Invalid))
	}
	t.Logf("Result: %s, Invalid: %d", result.Result, len(result.Invalid))
}

func TestIntegrationSendNormalization(t *testing.T) {
	c := getIntegrationClient(t)

	tests := []struct {
		name   string
		mobile string
	}{
		{"plus prefix", "+96598765432"},
		{"double zero prefix", "0096598765432"},
		{"Arabic digits", "٩٦٥٩٨٧٦٥٤٣٢"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.Send(tt.mobile, "Normalization test", "")
			if err != nil {
				t.Fatal(err)
			}
			// Should not fail with format error
			if result.Code == "ERR025" || result.Code == "ERR006" {
				t.Errorf("normalization should handle %q, got %s", tt.mobile, result.Code)
			}
			t.Logf("%s: result=%s code=%s", tt.name, result.Result, result.Code)
		})
	}
}

func TestIntegrationSendDeduplication(t *testing.T) {
	c := getIntegrationClient(t)

	// All normalize to the same number
	result, err := c.SendMulti(
		[]string{"+96598765432", "0096598765432", "96598765432"},
		"Dedup test",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	// Should send to only 1 number
	if result.Result == "OK" && result.Numbers > 1 {
		t.Errorf("dedup should send to 1 number, got %d", result.Numbers)
	}
	t.Logf("Dedup result: %s, numbers=%d", result.Result, result.Numbers)
}

func TestIntegrationSenderIDs(t *testing.T) {
	c := getIntegrationClient(t)

	result := c.SenderIDs()
	if result["result"] != "OK" {
		t.Logf("SenderIDs result: %v", result)
		return
	}
	sids, _ := result["senderids"].([]string)
	if len(sids) == 0 {
		t.Log("No sender IDs found (account may not have any registered)")
	}
	t.Logf("Sender IDs: %v", sids)
}

func TestIntegrationCoverage(t *testing.T) {
	c := getIntegrationClient(t)

	result := c.Coverage()
	if result["result"] != "OK" {
		t.Logf("Coverage result: %v", result)
		return
	}
	t.Logf("Coverage: result=%s", result["result"])
}

func TestIntegrationValidate(t *testing.T) {
	c := getIntegrationClient(t)

	result := c.Validate([]string{"96598765432", "+96512345678", "bad@email.com"})
	t.Logf("Validate OK=%v ER=%v NR=%v Rejected=%d Error=%s",
		result.OK, result.ER, result.NR, len(result.Rejected), result.Error)

	if len(result.Rejected) != 1 {
		t.Errorf("expected 1 rejected (email), got %d", len(result.Rejected))
	}
}

func TestIntegrationSendEmptySenderID(t *testing.T) {
	c := getIntegrationClient(t)
	// Use a blank sender to test API behavior
	result, err := c.Send("96598765432", "Empty sender test", " ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Empty sender: result=%s code=%s", result.Result, result.Code)
}

func TestIntegrationSendWrongSenderID(t *testing.T) {
	c := getIntegrationClient(t)
	result, err := c.Send("96598765432", "Wrong sender test", "NONEXISTENT-SENDER-XYZ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Wrong sender: result=%s code=%s action=%s", result.Result, result.Code, result.Action)
}
