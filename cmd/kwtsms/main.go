// Command kwtsms provides a CLI for the kwtSMS SMS API.
//
// Install: go install github.com/boxlinknet/kwtsms-go/cmd/kwtsms@latest
//
// Usage:
//
//	kwtsms verify                                  # test credentials, show balance
//	kwtsms balance                                 # show available + purchased credits
//	kwtsms senderid                                # list sender IDs
//	kwtsms coverage                                # list active country prefixes
//	kwtsms send <mobile> <message> [--sender ID]   # send SMS
//	kwtsms validate <number> [number2 ...]         # validate numbers
//	kwtsms status <msg-id>                         # check message status
//	kwtsms dlr <msg-id>                            # delivery report (intl only)
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	kwtsms "github.com/boxlinknet/kwtsms-go"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "verify":
		cmdVerify()
	case "balance":
		cmdBalance()
	case "senderid":
		cmdSenderID()
	case "coverage":
		cmdCoverage()
	case "send":
		cmdSend(args)
	case "validate":
		cmdValidate(args)
	case "status":
		cmdStatus(args)
	case "dlr":
		cmdDLR(args)
	case "help", "--help", "-h":
		printUsage()
	case "version", "--version", "-v":
		fmt.Printf("kwtsms %s\n", kwtsms.Version)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q. Run 'kwtsms help' for usage.\n", cmd)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`kwtsms - kwtSMS SMS API client

Usage:
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

func getClient() *kwtsms.KwtSMS {
	c, err := kwtsms.FromEnv("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\nSet KWTSMS_USERNAME and KWTSMS_PASSWORD environment variables or create a .env file.\n", err)
		os.Exit(1)
	}
	return c
}

func cmdVerify() {
	c := getClient()
	if c.CachedBalance() == nil {
		// Show test mode warning
		printTestModeWarning(c)
	}

	ok, balance, err := c.Verify()
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Credentials OK\n")
	fmt.Printf("Available balance: %.2f credits\n", balance)
	if p := c.CachedPurchased(); p != nil {
		fmt.Printf("Total purchased:   %.2f credits\n", *p)
	}
}

func cmdBalance() {
	c := getClient()
	bal, err := c.Balance()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Available: %.2f credits\n", bal)
	if p := c.CachedPurchased(); p != nil {
		fmt.Printf("Purchased: %.2f credits\n", *p)
	}
}

func cmdSenderID() {
	c := getClient()
	result := c.SenderIDs()
	if result["result"] != "OK" {
		printErrorResult(result)
		os.Exit(1)
	}
	sids, _ := result["senderids"].([]string)
	if len(sids) == 0 {
		fmt.Println("No sender IDs registered on this account.")
		return
	}
	fmt.Println("Sender IDs:")
	for _, sid := range sids {
		fmt.Printf("  %s\n", sid)
	}
}

func cmdCoverage() {
	c := getClient()
	result := c.Coverage()
	if result["result"] != "OK" {
		printErrorResult(result)
		os.Exit(1)
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}

func cmdSend(args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: kwtsms send <mobile> <message> [--sender ID]\n")
		os.Exit(1)
	}

	mobile := args[0]
	var message string
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
	message = strings.Join(messageParts, " ")

	if message == "" {
		fmt.Fprintf(os.Stderr, "Error: message is required\n")
		os.Exit(1)
	}

	c := getClient()
	printTestModeWarning(c)

	result, err := c.Send(mobile, message, sender)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if result.Result == "OK" {
		fmt.Printf("Message sent successfully\n")
		fmt.Printf("  msg-id:         %s\n", result.MsgID)
		fmt.Printf("  numbers:        %d\n", result.Numbers)
		fmt.Printf("  points-charged: %d\n", result.PointsCharged)
		fmt.Printf("  balance-after:  %.2f\n", result.BalanceAfter)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", result.Description)
		if result.Action != "" {
			fmt.Fprintf(os.Stderr, "Action: %s\n", result.Action)
		}
		os.Exit(1)
	}

	if len(result.Invalid) > 0 {
		fmt.Printf("\nInvalid numbers skipped:\n")
		for _, inv := range result.Invalid {
			fmt.Printf("  %s: %s\n", inv.Input, inv.Error)
		}
	}
}

func cmdValidate(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: kwtsms validate <number> [number2 ...]\n")
		os.Exit(1)
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

	c := getClient()
	result := c.Validate(phones)

	if len(result.OK) > 0 {
		fmt.Println("Valid (OK):")
		for _, n := range result.OK {
			fmt.Printf("  %s\n", n)
		}
	}
	if len(result.ER) > 0 {
		fmt.Println("Format error (ER):")
		for _, n := range result.ER {
			fmt.Printf("  %s\n", n)
		}
	}
	if len(result.NR) > 0 {
		fmt.Println("No route (NR):")
		for _, n := range result.NR {
			fmt.Printf("  %s\n", n)
		}
	}
	if len(result.Rejected) > 0 {
		fmt.Println("Locally rejected:")
		for _, r := range result.Rejected {
			fmt.Printf("  %s: %s\n", r.Input, r.Error)
		}
	}
	if result.Error != "" {
		fmt.Fprintf(os.Stderr, "\nError: %s\n", result.Error)
		os.Exit(1)
	}
}

func cmdStatus(args []string) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: kwtsms status <msg-id>\n")
		os.Exit(1)
	}

	c := getClient()
	result := c.Status(args[0])
	if result["result"] != "OK" {
		printErrorResult(result)
		os.Exit(1)
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}

func cmdDLR(args []string) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: kwtsms dlr <msg-id>\n")
		os.Exit(1)
	}

	c := getClient()
	result := c.DLR(args[0])
	if result["result"] != "OK" {
		printErrorResult(result)
		os.Exit(1)
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}

func printErrorResult(result map[string]any) {
	desc, _ := result["description"].(string)
	code, _ := result["code"].(string)
	action, _ := result["action"].(string)

	if desc != "" {
		fmt.Fprintf(os.Stderr, "Error: %s", desc)
		if code != "" {
			fmt.Fprintf(os.Stderr, " (%s)", code)
		}
		fmt.Fprintln(os.Stderr)
	} else if code != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", code)
	}
	if action != "" {
		fmt.Fprintf(os.Stderr, "Action: %s\n", action)
	}
}

func printTestModeWarning(c *kwtsms.KwtSMS) {
	// Check via environment since testMode is unexported
	if os.Getenv("KWTSMS_TEST_MODE") == "1" {
		fmt.Println("WARNING: Test mode is ON. Messages will be queued but NOT delivered.")
	}
}
