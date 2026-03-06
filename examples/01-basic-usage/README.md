# 01 - Basic Usage

Demonstrates the simplest kwtsms-go workflow: load credentials, verify the account, check the balance, and send a single SMS.

## What it does

1. Loads API credentials from environment variables or a `.env` file
2. Calls `Verify()` to confirm credentials and fetch the balance
3. Sends a test SMS to a single Kuwait number
4. Prints the send result, including message ID and remaining balance

## Running

```bash
cd examples/01-basic-usage
cp ../../.env.example .env   # edit with your credentials
go run main.go
```

Replace the phone number in `main.go` with a real number you own.
Set `KWTSMS_TEST_MODE=1` in your `.env` to avoid consuming credits.
