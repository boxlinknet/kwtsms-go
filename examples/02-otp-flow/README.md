# 02 - OTP Flow

Demonstrates a complete OTP (One-Time Password) workflow: validate the recipient, generate a secure code, send it, and capture the message ID for tracking.

## What it does

1. Validates the phone number using `ValidatePhoneInput()` before any API call
2. Generates a cryptographically random 6-digit OTP using `crypto/rand`
3. Formats the message with an app name ("Your OTP for MYAPP is: 123456")
4. Sends the SMS and saves the message ID for later delivery checks

## Running

```bash
cd examples/02-otp-flow
cp ../../.env.example .env   # edit with your credentials
go run main.go
```

Replace the phone number in `main.go` with your own.
Set `KWTSMS_TEST_MODE=1` to avoid consuming credits during testing.
