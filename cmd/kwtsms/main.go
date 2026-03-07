// Command kwtsms provides a CLI for the kwtSMS SMS API.
//
// Install: go install github.com/boxlinknet/kwtsms-go/cmd/kwtsms@latest
//
// Usage:
//
//	kwtsms setup                                     Interactive setup wizard
//	kwtsms verify                                    Test credentials, show balance
//	kwtsms balance                                   Show available + purchased credits
//	kwtsms senderid                                  List sender IDs
//	kwtsms coverage                                  List active country prefixes
//	kwtsms send <mobile> <message> [--sender ID]     Send SMS
//	kwtsms validate <number> [number2 ...]           Validate phone numbers
//	kwtsms status <msg-id>                           Check message status
//	kwtsms dlr <msg-id>                              Delivery report (intl only)
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	kwtsms "github.com/boxlinknet/kwtsms-go"
)

type app struct {
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
	envFile   string
	newClient func() (*kwtsms.KwtSMS, error)
	extraOpts []kwtsms.Option // extra options for clients created during setup (testing)
}

func main() {
	a := &app{
		stdin:   os.Stdin,
		stdout:  os.Stdout,
		stderr:  os.Stderr,
		envFile: ".env",
		newClient: func() (*kwtsms.KwtSMS, error) {
			return kwtsms.FromEnv("")
		},
	}
	os.Exit(a.run(os.Args[1:]))
}

func (a *app) run(args []string) int {
	if len(args) == 0 {
		a.printUsage()
		return 1
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "setup":
		return a.cmdSetup()
	case "verify":
		return a.cmdVerify()
	case "balance":
		return a.cmdBalance()
	case "senderid":
		return a.cmdSenderID()
	case "coverage":
		return a.cmdCoverage()
	case "send":
		return a.cmdSend(rest)
	case "validate":
		return a.cmdValidate(rest)
	case "status":
		return a.cmdStatus(rest)
	case "dlr":
		return a.cmdDLR(rest)
	case "help", "--help", "-h":
		a.printUsage()
		return 0
	case "version", "--version", "-v":
		fmt.Fprintf(a.stdout, "kwtsms %s\n", kwtsms.Version)
		return 0
	default:
		fmt.Fprintf(a.stderr, "Error: unknown command %q. Run 'kwtsms help' for usage.\n", cmd)
		return 1
	}
}

func (a *app) printUsage() {
	fmt.Fprintln(a.stdout, `kwtsms - kwtSMS SMS API client

Usage:
  kwtsms setup                                     Interactive setup wizard
  kwtsms verify                                    Test credentials, show balance
  kwtsms balance                                   Show available + purchased credits
  kwtsms senderid                                  List sender IDs
  kwtsms coverage                                  List active country prefixes
  kwtsms send <mobile> <message> [--sender ID]     Send SMS (comma-separated for multiple)
  kwtsms validate <number> [number2 ...]           Validate phone numbers
  kwtsms status <msg-id>                           Check message status
  kwtsms dlr <msg-id>                              Delivery report (international only)
  kwtsms version                                   Show version
  kwtsms help                                      Show this help

Environment variables (or .env file):
  KWTSMS_USERNAME    API username (required)
  KWTSMS_PASSWORD    API password (required)
  KWTSMS_SENDER_ID   Default sender ID (default: KWT-SMS)
  KWTSMS_TEST_MODE   Set to 1 for test mode (default: 0)
  KWTSMS_LOG_FILE    Log file path (default: kwtsms.log)`)
}

func (a *app) getClient() (*kwtsms.KwtSMS, int) {
	c, err := a.newClient()
	if err != nil {
		// Auto-setup: if no .env file exists, run setup wizard
		if _, statErr := os.Stat(a.envFile); os.IsNotExist(statErr) {
			fmt.Fprintln(a.stdout, "No .env file found. Starting first-time setup...")
			fmt.Fprintln(a.stdout)
			if code := a.cmdSetup(); code != 0 {
				return nil, code
			}
			c, err = a.newClient()
		}
		if err != nil {
			fmt.Fprintf(a.stderr, "Error: %v\n\nSet KWTSMS_USERNAME and KWTSMS_PASSWORD environment variables or create a .env file.\nRun 'kwtsms setup' for interactive configuration.\n", err)
			return nil, 1
		}
	}
	return c, 0
}

func (a *app) cmdVerify() int {
	c, code := a.getClient()
	if c == nil {
		return code
	}
	a.printTestModeWarning()

	ok, balance, err := c.Verify()
	if !ok {
		fmt.Fprintf(a.stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Fprintln(a.stdout, "Credentials OK")
	fmt.Fprintf(a.stdout, "Available balance: %.2f credits\n", balance)
	if p := c.CachedPurchased(); p != nil {
		fmt.Fprintf(a.stdout, "Total purchased:   %.2f credits\n", *p)
	}
	return 0
}

func (a *app) cmdBalance() int {
	c, code := a.getClient()
	if c == nil {
		return code
	}
	bal, err := c.Balance()
	if err != nil {
		fmt.Fprintf(a.stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Fprintf(a.stdout, "Available: %.2f credits\n", bal)
	if p := c.CachedPurchased(); p != nil {
		fmt.Fprintf(a.stdout, "Purchased: %.2f credits\n", *p)
	}
	return 0
}

func (a *app) cmdSenderID() int {
	c, code := a.getClient()
	if c == nil {
		return code
	}
	result := c.SenderIDs()
	if result["result"] != "OK" {
		a.printErrorResult(result)
		return 1
	}
	sids, _ := result["senderids"].([]string)
	if len(sids) == 0 {
		fmt.Fprintln(a.stdout, "No sender IDs registered on this account.")
		return 0
	}
	fmt.Fprintln(a.stdout, "Sender IDs:")
	for _, sid := range sids {
		fmt.Fprintf(a.stdout, "  %s\n", sid)
	}
	return 0
}

func (a *app) cmdCoverage() int {
	c, code := a.getClient()
	if c == nil {
		return code
	}
	result := c.Coverage()
	if result["result"] != "OK" {
		a.printErrorResult(result)
		return 1
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Fprintln(a.stdout, string(data))
	return 0
}

func (a *app) cmdSend(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(a.stderr, "Usage: kwtsms send <mobile> <message> [--sender ID]")
		return 1
	}

	mobile := args[0]
	var sender string

	// Parse args: message and optional --sender flag
	i := 1
	var messageParts []string
	for i < len(args) {
		if args[i] == "--sender" && i+1 < len(args) {
			sender = args[i+1]
			i += 2
			continue
		}
		messageParts = append(messageParts, args[i])
		i++
	}
	message := strings.Join(messageParts, " ")

	if message == "" {
		fmt.Fprintln(a.stderr, "Error: message is required")
		return 1
	}

	c, code := a.getClient()
	if c == nil {
		return code
	}
	a.printTestModeWarning()

	result, err := c.Send(mobile, message, sender)
	if err != nil {
		fmt.Fprintf(a.stderr, "Error: %v\n", err)
		return 1
	}

	if result.Result == "OK" {
		fmt.Fprintln(a.stdout, "Message sent successfully")
		fmt.Fprintf(a.stdout, "  msg-id:         %s\n", result.MsgID)
		fmt.Fprintf(a.stdout, "  numbers:        %d\n", result.Numbers)
		fmt.Fprintf(a.stdout, "  points-charged: %d\n", result.PointsCharged)
		fmt.Fprintf(a.stdout, "  balance-after:  %.2f\n", result.BalanceAfter)
	} else {
		fmt.Fprintf(a.stderr, "Error: %s\n", result.Description)
		if result.Action != "" {
			fmt.Fprintf(a.stderr, "Action: %s\n", result.Action)
		}
		return 1
	}

	if len(result.Invalid) > 0 {
		fmt.Fprintln(a.stdout, "\nInvalid numbers skipped:")
		for _, inv := range result.Invalid {
			fmt.Fprintf(a.stdout, "  %s: %s\n", inv.Input, inv.Error)
		}
	}
	return 0
}

func (a *app) cmdValidate(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.stderr, "Usage: kwtsms validate <number> [number2 ...]")
		return 1
	}

	// Support comma-separated and space-separated
	var phones []string
	for _, arg := range args {
		for _, p := range strings.Split(arg, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				phones = append(phones, p)
			}
		}
	}

	c, code := a.getClient()
	if c == nil {
		return code
	}
	result := c.Validate(phones)

	if len(result.OK) > 0 {
		fmt.Fprintln(a.stdout, "Valid (OK):")
		for _, n := range result.OK {
			fmt.Fprintf(a.stdout, "  %s\n", n)
		}
	}
	if len(result.ER) > 0 {
		fmt.Fprintln(a.stdout, "Format error (ER):")
		for _, n := range result.ER {
			fmt.Fprintf(a.stdout, "  %s\n", n)
		}
	}
	if len(result.NR) > 0 {
		fmt.Fprintln(a.stdout, "No route (NR):")
		for _, n := range result.NR {
			fmt.Fprintf(a.stdout, "  %s\n", n)
		}
	}
	if len(result.Rejected) > 0 {
		fmt.Fprintln(a.stdout, "Locally rejected:")
		for _, r := range result.Rejected {
			fmt.Fprintf(a.stdout, "  %s: %s\n", r.Input, r.Error)
		}
	}
	if result.Error != "" {
		fmt.Fprintf(a.stderr, "\nError: %s\n", result.Error)
		return 1
	}
	return 0
}

func (a *app) cmdStatus(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(a.stderr, "Usage: kwtsms status <msg-id>")
		return 1
	}

	c, code := a.getClient()
	if c == nil {
		return code
	}
	result := c.Status(args[0])
	if result["result"] != "OK" {
		a.printErrorResult(result)
		return 1
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Fprintln(a.stdout, string(data))
	return 0
}

func (a *app) cmdDLR(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(a.stderr, "Usage: kwtsms dlr <msg-id>")
		return 1
	}

	c, code := a.getClient()
	if c == nil {
		return code
	}
	result := c.DLR(args[0])
	if result["result"] != "OK" {
		a.printErrorResult(result)
		return 1
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Fprintln(a.stdout, string(data))
	return 0
}

func (a *app) printErrorResult(result map[string]any) {
	desc, _ := result["description"].(string)
	code, _ := result["code"].(string)
	action, _ := result["action"].(string)

	if desc != "" {
		fmt.Fprintf(a.stderr, "Error: %s", desc)
		if code != "" {
			fmt.Fprintf(a.stderr, " (%s)", code)
		}
		fmt.Fprintln(a.stderr)
	} else if code != "" {
		fmt.Fprintf(a.stderr, "Error: %s\n", code)
	}
	if action != "" {
		fmt.Fprintf(a.stderr, "Action: %s\n", action)
	}
}

func (a *app) printTestModeWarning() {
	if os.Getenv("KWTSMS_TEST_MODE") == "1" {
		fmt.Fprintln(a.stdout, "WARNING: Test mode is ON. Messages will be queued but NOT delivered.")
	}
}

func (a *app) cmdSetup() int {
	reader := bufio.NewReader(a.stdin)

	fmt.Fprintln(a.stdout, "\n-- kwtSMS Setup -------------------------------------------------------")
	fmt.Fprintln(a.stdout, "Verifies your API credentials and creates a .env file.")
	fmt.Fprintln(a.stdout, "Press Enter to keep the value shown in brackets.")
	fmt.Fprintln(a.stdout)

	// Load existing .env for defaults
	existing := kwtsms.LoadEnvFile(a.envFile)

	// Username
	defaultUser := existing["KWTSMS_USERNAME"]
	if defaultUser != "" {
		fmt.Fprintf(a.stdout, "API username [%s]: ", defaultUser)
	} else {
		fmt.Fprint(a.stdout, "API username: ")
	}
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)
	if username == "" {
		username = defaultUser
	}
	if username == "" {
		fmt.Fprintln(a.stderr, "Error: username is required")
		return 1
	}

	// Password
	defaultPass := existing["KWTSMS_PASSWORD"]
	if defaultPass != "" {
		fmt.Fprint(a.stdout, "API password [keep existing]: ")
	} else {
		fmt.Fprint(a.stdout, "API password: ")
	}
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)
	if password == "" {
		password = defaultPass
	}
	if password == "" {
		fmt.Fprintln(a.stderr, "Error: password is required")
		return 1
	}

	// Sanitize early: strip \r and \n from credentials before using them
	replacer := strings.NewReplacer("\r", "", "\n", "")
	username = replacer.Replace(username)
	password = replacer.Replace(password)

	// Verify credentials
	fmt.Fprint(a.stdout, "\nVerifying credentials... ")

	opts := []kwtsms.Option{kwtsms.WithLogFile("")}
	opts = append(opts, a.extraOpts...)
	c, err := kwtsms.New(username, password, opts...)
	if err != nil {
		fmt.Fprintln(a.stdout, "FAILED")
		fmt.Fprintf(a.stderr, "Error: %v\n", err)
		return 1
	}

	ok, balance, verifyErr := c.Verify()
	if !ok {
		fmt.Fprintln(a.stdout, "FAILED")
		fmt.Fprintf(a.stderr, "Error: %v\n", verifyErr)
		fmt.Fprintln(a.stderr, "Fix your username/password and run 'kwtsms setup' again.")
		return 1
	}
	fmt.Fprintf(a.stdout, "OK (Balance: %.2f)\n", balance)

	// Fetch Sender IDs
	fmt.Fprint(a.stdout, "Fetching Sender IDs... ")
	sidResult := c.SenderIDs()
	sids, _ := sidResult["senderids"].([]string)

	defaultSID := existing["KWTSMS_SENDER_ID"]
	var senderID string

	if len(sids) > 0 {
		fmt.Fprintln(a.stdout, "OK")
		fmt.Fprintln(a.stdout, "\nAvailable Sender IDs:")
		for i, sid := range sids {
			fmt.Fprintf(a.stdout, "  %d. %s\n", i+1, sid)
		}
		if defaultSID == "" {
			defaultSID = sids[0]
		}
		fmt.Fprintf(a.stdout, "\nSelect Sender ID (number or name) [%s]: ", defaultSID)
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		if choice == "" {
			senderID = defaultSID
		} else {
			// Check if choice is a number index
			idx := 0
			isNum := true
			for _, ch := range choice {
				if ch >= '0' && ch <= '9' {
					idx = idx*10 + int(ch-'0')
				} else {
					isNum = false
					break
				}
			}
			if isNum && idx >= 1 && idx <= len(sids) {
				senderID = sids[idx-1]
			} else {
				senderID = choice
			}
		}
	} else {
		fmt.Fprintln(a.stdout, "(none returned)")
		if defaultSID == "" {
			defaultSID = "KWT-SMS"
		}
		fmt.Fprintf(a.stdout, "Sender ID [%s]: ", defaultSID)
		sid, _ := reader.ReadString('\n')
		senderID = strings.TrimSpace(sid)
		if senderID == "" {
			senderID = defaultSID
		}
	}

	// Send mode
	currentMode := existing["KWTSMS_TEST_MODE"]
	modeDefault := "1"
	if currentMode == "0" {
		modeDefault = "2"
	}
	fmt.Fprintln(a.stdout, "\nSend mode:")
	fmt.Fprintln(a.stdout, "  1. Test mode: messages queued but NOT delivered, no credits consumed  [default]")
	fmt.Fprintln(a.stdout, "  2. Live mode: messages delivered to handsets, credits consumed")
	fmt.Fprintf(a.stdout, "\nChoose [%s]: ", modeDefault)
	modeLine, _ := reader.ReadString('\n')
	modeLine = strings.TrimSpace(modeLine)
	if modeLine == "" {
		modeLine = modeDefault
	}
	testMode := "1"
	if modeLine == "2" {
		testMode = "0"
		fmt.Fprintln(a.stdout, "  -> Live mode selected. Real messages will be sent and credits consumed.")
	} else {
		fmt.Fprintln(a.stdout, "  -> Test mode selected.")
	}

	// Log file
	defaultLog := existing["KWTSMS_LOG_FILE"]
	if defaultLog == "" {
		defaultLog = "kwtsms.log"
	}
	fmt.Fprintln(a.stdout, "\nAPI logging (every API call is logged to a file, passwords are always masked):")
	fmt.Fprintf(a.stdout, "  Current: %s\n", defaultLog)
	fmt.Fprintln(a.stdout, `  Type "off" to disable logging.`)
	fmt.Fprintf(a.stdout, "  Log file path [%s]: ", defaultLog)
	logInput, _ := reader.ReadString('\n')
	logInput = strings.TrimSpace(logInput)
	logFilePath := defaultLog
	if strings.ToLower(logInput) == "off" {
		logFilePath = ""
		fmt.Fprintln(a.stdout, "  -> Logging disabled.")
	} else if logInput != "" {
		logFilePath = logInput
	}

	// Sanitize remaining fields (username/password already sanitized before verification)
	senderID = replacer.Replace(senderID)
	logFilePath = replacer.Replace(logFilePath)

	// Write .env
	envContent := fmt.Sprintf("# kwtSMS credentials, generated by kwtsms setup\nKWTSMS_USERNAME=%s\nKWTSMS_PASSWORD=%s\nKWTSMS_SENDER_ID=%s\nKWTSMS_TEST_MODE=%s\nKWTSMS_LOG_FILE=%s\n",
		username, password, senderID, testMode, logFilePath)

	if err := os.WriteFile(a.envFile, []byte(envContent), 0600); err != nil {
		fmt.Fprintf(a.stderr, "Error writing %s: %v\n", a.envFile, err)
		return 1
	}

	fmt.Fprintf(a.stdout, "\n  Saved to %s\n", a.envFile)
	if testMode == "1" {
		fmt.Fprintln(a.stdout, "  Mode: TEST: messages queued but not delivered (no credits consumed)")
	} else {
		fmt.Fprintln(a.stdout, "  Mode: LIVE: messages will be delivered and credits consumed")
	}
	fmt.Fprintln(a.stdout, "  Run 'kwtsms setup' at any time to change settings.")
	fmt.Fprintln(a.stdout, "----------------------------------------------------------------------")
	return 0
}
