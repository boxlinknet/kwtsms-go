# kwtsms-go

[![Go Reference](https://pkg.go.dev/badge/github.com/boxlinknet/kwtsms-go.svg)](https://pkg.go.dev/github.com/boxlinknet/kwtsms-go)
[![CI](https://github.com/boxlinknet/kwtsms-go/actions/workflows/ci.yml/badge.svg)](https://github.com/boxlinknet/kwtsms-go/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/boxlinknet/kwtsms-go)](https://goreportcard.com/report/github.com/boxlinknet/kwtsms-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

Go client library for the [kwtSMS SMS API](https://www.kwtsms.com). Zero external dependencies. Go 1.18+.

Send SMS, check balance, validate phone numbers, list sender IDs, check coverage, and retrieve delivery reports. All with automatic phone normalization, message cleaning, and developer-friendly error messages.

## Install

```bash
go get github.com/boxlinknet/kwtsms-go
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    kwtsms "github.com/boxlinknet/kwtsms-go"
)

func main() {
    // Load credentials from environment variables or .env file
    sms, err := kwtsms.FromEnv("")
    if err != nil {
        log.Fatal(err)
    }

    // Verify credentials and check balance
    ok, balance, err := sms.Verify()
    if !ok {
        log.Fatalf("verification failed: %v", err)
    }
    fmt.Printf("Balance: %.2f credits\n", balance)

    // Send an SMS
    result, err := sms.Send("96598765432", "Your OTP for MYAPP is: 123456", "")
    if err != nil {
        log.Fatal(err)
    }
    if result.Result == "OK" {
        fmt.Printf("Sent! msg-id: %s, balance: %.2f\n", result.MsgID, result.BalanceAfter)
    } else {
        fmt.Printf("Error: %s\nAction: %s\n", result.Description, result.Action)
    }
}
```

## Configuration

### Environment variables

Set these environment variables or create a `.env` file in your project root:

```ini
KWTSMS_USERNAME=your_api_user
KWTSMS_PASSWORD=your_api_pass
KWTSMS_SENDER_ID=YOUR-SENDERID
KWTSMS_TEST_MODE=1
KWTSMS_LOG_FILE=kwtsms.log
```

### Constructor

```go
// From environment variables / .env file (recommended)
sms, err := kwtsms.FromEnv("")

// From .env at a custom path
sms, err := kwtsms.FromEnv("/path/to/.env")

// Direct constructor with options
sms, err := kwtsms.New("username", "password",
    kwtsms.WithSenderID("MY-APP"),
    kwtsms.WithTestMode(true),
    kwtsms.WithLogFile("sms.log"),
)
```

Environment variables take priority over `.env` file values.

## API Reference

### Verify Credentials

```go
ok, balance, err := sms.Verify()
// ok=true:  credentials valid, balance is the available credit count
// ok=false: err describes the problem with an action to take
```

### Check Balance

```go
balance, err := sms.Balance()
// Returns live balance, or cached value if the API call fails
```

### Send SMS

```go
// Single number
result, err := sms.Send("96598765432", "Hello from Go!", "")

// Multiple numbers (comma-separated)
result, err := sms.Send("96598765432,96512345678", "Bulk message", "")

// Multiple numbers (slice)
result, err := sms.SendMulti(
    []string{"96598765432", "+96512345678", "0096587654321"},
    "Hello everyone!",
    "",
)

// Override sender ID for one call
result, err := sms.Send("96598765432", "Hello", "OTHER-SENDER")
```

**Always save `msg-id` immediately after a successful send.** You need it for status checks and delivery reports. If you do not store it at send time, you cannot retrieve it later.

**Never call `Balance()` after `Send()`.** The send response already includes your updated balance in `result.BalanceAfter`. Save it to your database. The client also caches it internally.

Phone numbers are normalized automatically: `+`, `00` prefixes, spaces, dashes, Arabic/Persian digits are all handled. Duplicate numbers (after normalization) are sent only once.

Messages are cleaned automatically: emojis, HTML tags, hidden control characters (BOM, zero-width spaces), and C0/C1 controls are stripped. Arabic digits are converted to Latin. Arabic text is preserved.

### Send with Retry (ERR028)

```go
// Auto-retries on ERR028 (15-second rate limit), waits 16s between retries
result, err := sms.SendWithRetry("96598765432", "Hello", "", 3)
```

### Bulk Send (>200 numbers)

Sending to more than 200 numbers is handled automatically. The client splits numbers into batches of 200, adds a 0.5s delay between batches, and retries on ERR013 (queue full) with 30s/60s/120s backoff.

```go
numbers := make([]string, 500)
// ... populate numbers ...
result, err := sms.SendMulti(numbers, "Campaign message", "MY-SENDER")
```

### Validate Phone Numbers

```go
result := sms.Validate([]string{"96598765432", "+96512345678", "bad@email.com", "123"})
fmt.Println("Valid:", result.OK)       // valid and routable
fmt.Println("Errors:", result.ER)      // format errors
fmt.Println("No route:", result.NR)    // country not activated
fmt.Println("Rejected:", result.Rejected) // locally rejected (email, too short, etc.)
```

Numbers that fail local validation (empty, email, too short, too long, no digits) are rejected before the API call is made.

### Sender IDs

```go
result := sms.SenderIDs()
if result["result"] == "OK" {
    sids := result["senderids"].([]string)
    fmt.Println("Sender IDs:", sids)
}
```

### Coverage

```go
result := sms.Coverage()
if result["result"] == "OK" {
    fmt.Println("Active coverage:", result)
}
```

### Message Status

```go
result := sms.Status("msg-id-from-send-response")
fmt.Println(result)
```

### Delivery Report (international only)

```go
result := sms.DLR("msg-id-from-send-response")
fmt.Println(result)
```

Kuwait numbers do not support DLR. Only international (non-Kuwait) numbers have delivery reports. Wait at least 5 minutes after sending before checking.

### Cached Balance

```go
// Available after Verify() or successful Send()
if bal := sms.CachedBalance(); bal != nil {
    fmt.Printf("Cached balance: %.2f\n", *bal)
}

// Total purchased credits (available after Verify())
if p := sms.CachedPurchased(); p != nil {
    fmt.Printf("Purchased: %.2f\n", *p)
}
```

## Utility Functions

These are exported for direct use:

```go
// Normalize a phone number: Arabic digits to Latin, strip non-digits, strip leading zeros
normalized := kwtsms.NormalizePhone("+965 9876 5432") // "96598765432"

// Validate phone input before sending
v := kwtsms.ValidatePhoneInput("user@gmail.com")
// v.Valid=false, v.Error="'user@gmail.com' is an email address, not a phone number"

v = kwtsms.ValidatePhoneInput("+96598765432")
// v.Valid=true, v.Normalized="96598765432"

// Clean a message: strip emojis, HTML, control chars, convert Arabic digits
cleaned := kwtsms.CleanMessage("Hello 😀 <b>World</b> ١٢٣")
// "Hello  World 123"

// API error map (for building custom error UIs)
action := kwtsms.APIErrors["ERR003"]
// "Wrong API username or password. Check KWTSMS_USERNAME and KWTSMS_PASSWORD..."

// Enrich an error response with action guidance
enriched := kwtsms.EnrichError(apiResponse)
```

## Exported Types

```go
kwtsms.KwtSMS           // Client struct
kwtsms.SendResult       // Send response
kwtsms.BulkSendResult   // Bulk send response (>200 numbers)
kwtsms.ValidateResult   // Validate response
kwtsms.InvalidEntry     // Rejected phone number with error message
kwtsms.BatchError       // Error from a single batch in bulk send
kwtsms.PhoneValidation  // Result of ValidatePhoneInput
kwtsms.Option           // Functional option for New()
```

## CLI

Install the CLI:

```bash
go install github.com/boxlinknet/kwtsms-go/cmd/kwtsms@latest
```

Usage:

```bash
kwtsms verify                                    # test credentials, show balance
kwtsms balance                                   # show available + purchased credits
kwtsms senderid                                  # list sender IDs
kwtsms coverage                                  # list active country prefixes
kwtsms send 96598765432 "Your OTP is: 1234"      # send SMS
kwtsms send 96598765432,96512345678 "Hello"       # multi-number
kwtsms send 96598765432 "Hello" --sender MY-APP   # custom sender
kwtsms validate 96598765432 96512345678           # validate numbers
kwtsms status f4c841adee210f31307633ceaebff2ec    # check message status
kwtsms dlr f4c841adee210f31307633ceaebff2ec       # delivery report
kwtsms version                                    # show version
```

The CLI reads credentials from `KWTSMS_USERNAME` and `KWTSMS_PASSWORD` environment variables or a `.env` file.

## Credential Management

**Never hardcode credentials in source code.** Credentials must be changeable without recompiling.

### Environment variables / .env file (recommended for servers)

```go
sms, err := kwtsms.FromEnv("")  // reads KWTSMS_USERNAME, KWTSMS_PASSWORD
```

The `.env` file must be in `.gitignore`. Never commit credentials.

### Constructor injection (for custom config systems)

```go
sms, err := kwtsms.New(
    config.Get("sms_username"),
    config.Get("sms_password"),
)
```

Works with any config source: Vault, AWS Secrets Manager, database, DI containers, etc.

### Admin settings UI (recommended for web apps)

Provide a settings page where an admin can enter API credentials and toggle test mode. Include a "Test Connection" button that calls `Verify()`.

## Error Handling

Every API error includes a developer-friendly `action` field explaining what to do. All 33 kwtSMS error codes are mapped.

```go
result, _ := sms.Send("96598765432", "Hello", "")
if result.Result == "ERROR" {
    fmt.Println("Code:", result.Code)             // "ERR003"
    fmt.Println("Description:", result.Description) // "Authentication error..."
    fmt.Println("Action:", result.Action)           // "Wrong API username or password. Check..."
}
```

### User-Facing Error Messages

Raw API errors are for developers, not end users. Map them for your UI:

| Situation | API error | Show to user |
|-----------|-----------|--------------|
| Invalid phone | ERR006, ERR025 | "Please enter a valid phone number in international format (e.g., +965 9876 5432)." |
| Wrong credentials | ERR003 | "SMS service is temporarily unavailable. Please try again later." |
| No balance | ERR010, ERR011 | "SMS service is temporarily unavailable. Please try again later." |
| Country not supported | ERR026 | "SMS delivery to this country is not available. Please contact support." |
| Rate limited | ERR028 | "Please wait a moment before requesting another code." |
| Message rejected | ERR031, ERR032 | "Your message could not be sent. Please try again with different content." |
| Network error | timeout | "Could not connect to SMS service. Check your internet connection." |
| Queue full | ERR013 | "SMS service is busy. Please try again in a few minutes." |

**Key principle:** user-recoverable errors (bad phone, rate limited) get a helpful message. System-level errors (auth, balance, network) get a generic message + log the real error + alert the admin.

## Best Practices

### 1. Validate before calling the API

The #1 cause of wasted API calls: sending invalid input and letting the API reject it. Validate locally first:

```go
// BAD: wastes an API call on every invalid input
result, _ := sms.Send(userInput, message, "")

// GOOD: validate locally, only hit API with clean input
v := kwtsms.ValidatePhoneInput(userInput)
if !v.Valid {
    return fmt.Errorf("invalid phone: %s", v.Error)
}
cleaned := kwtsms.CleanMessage(message)
if strings.TrimSpace(cleaned) == "" {
    return fmt.Errorf("message is empty after cleaning")
}
result, _ := sms.Send(v.Normalized, message, "")
```

The `Send()` method does validate and clean internally, but checking first lets you return errors to the user immediately without a network round-trip.

### 2. Cache coverage at startup

```go
coverage := sms.Coverage()
// Cache the active country prefixes, check before every send
```

### 3. Save balance-after and msg-id

```go
if result.Result == "OK" {
    db.SaveBalance(result.BalanceAfter)   // track balance without extra API calls
    db.SaveMsgID(result.MsgID)            // needed for Status() and DLR() later
}
```

### 4. Sender ID

| | Promotional | Transactional |
|--|-------------|---------------|
| **Use for** | Bulk SMS, marketing, offers | OTP, alerts, notifications |
| **DND numbers** | Blocked/filtered, credits lost | Bypasses DND |
| **Speed** | May have delays | Priority delivery |

`KWT-SMS` is the shared test sender. It causes delays and is blocked on Virgin Kuwait. **Never use in production.** Register a private sender ID at kwtsms.com.

For OTP/authentication, you **must** use a Transactional sender ID. Promotional sender IDs are filtered by DND (Do Not Disturb) on Zain and Ooredoo, meaning OTP messages silently fail and credits are still deducted.

Sender ID is **case sensitive**: `Kuwait` is not the same as `KUWAIT`.

### 5. OTP implementation

- Always include the app/company name: `"Your OTP for APPNAME is: 123456"`
- Resend timer: minimum 3-4 minutes (KNET standard is 4 minutes)
- OTP expiry: 3-5 minutes
- Generate a new code on resend, invalidate all previous codes
- Send to one number per request (avoid ERR028 batch rejection)
- Use a Transactional sender ID (not Promotional)

### 6. Timezone

`unix-timestamp` in API responses is **GMT+3 (Asia/Kuwait server time), not UTC**. Always convert when storing or displaying. Log timestamps written by the client are always UTC ISO-8601.

### 7. Rate limiting

Wait at least **15 seconds** before sending to the same number again (ERR028). The entire request is rejected if any number in a batch triggers this, even if other numbers are fine.

## Security Checklist

Before going live:

```
[ ] Bot protection enabled (CAPTCHA for web apps)
[ ] Rate limit per phone number (max 3-5/hour)
[ ] Rate limit per IP address (max 10-20/hour)
[ ] Rate limit per user/session if authenticated
[ ] Monitoring/alerting on abuse patterns
[ ] Admin notification on low balance
[ ] Test mode OFF (KWTSMS_TEST_MODE=0)
[ ] Private Sender ID registered (not KWT-SMS)
[ ] Transactional Sender ID for OTP (not promotional)
```

Without rate limiting, a bot can drain your entire SMS balance in minutes.

## Testing

```bash
# Unit + mocked API tests (no credentials needed)
go test -v ./...

# With race detector
go test -race ./...

# Integration tests (hits live API with test_mode=true, no credits consumed)
GO_USERNAME=your_user GO_PASSWORD=your_pass go test -v -tags integration ./...
```

## JSONL Logging

Every API call is logged to `kwtsms.log` (configurable) as one JSON line. Passwords are always masked as `***`. Timestamps are UTC ISO-8601.

```json
{"ts":"2026-03-06T12:00:00Z","endpoint":"send","request":{"username":"myuser","password":"***","sender":"MY-APP","mobile":"96598765432","message":"Hello","test":"1"},"response":{"result":"OK","msg-id":"abc123"},"ok":true}
```

Logging never crashes the main flow. Disk errors are silently ignored.

## Publishing

Go modules are published via git tags. No registry submission needed.

```bash
git tag v0.2.0
git push origin v0.2.0
```

pkg.go.dev indexes new versions automatically within minutes. Users install with:

```bash
go get github.com/boxlinknet/kwtsms-go@latest
```

## License

MIT. See [LICENSE](LICENSE).

## Links

- [kwtSMS website](https://www.kwtsms.com)
- [API documentation (PDF)](https://www.kwtsms.com/doc/KwtSMS.com_API_Documentation_v41.pdf)
- [Implementation best practices](https://www.kwtsms.com/articles/sms-api-implementation-best-practices.html)
- [Integration test checklist](https://www.kwtsms.com/articles/sms-api-integration-test-checklist.html)
- [pkg.go.dev reference](https://pkg.go.dev/github.com/boxlinknet/kwtsms-go)
