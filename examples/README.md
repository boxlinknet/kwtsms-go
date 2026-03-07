# kwtsms-go Examples

Seven runnable examples demonstrating the kwtSMS API and the kwtsms-go library.

| # | Directory | Description |
|---|-----------|-------------|
| 0 | [00-raw-api](./00-raw-api/) | Call every kwtSMS endpoint using only the Go standard library (no dependencies) |
| 1 | [01-basic-usage](./01-basic-usage/) | Load credentials, verify account, check balance, send a single SMS |
| 2 | [02-otp-flow](./02-otp-flow/) | Generate a 6-digit OTP, validate the phone number, send the code |
| 3 | [03-bulk-sms](./03-bulk-sms/) | Send to multiple numbers using SendMulti(), handle mixed formats |
| 4 | [04-http-handler](./04-http-handler/) | Minimal HTTP server that accepts POST JSON and sends SMS |
| 5 | [05-error-handling](./05-error-handling/) | Every error handling pattern: validation, cleaning, API errors |
| 6 | [06-otp-production](./06-otp-production/) | Production OTP server: rate limiting, expiry, resend cooldown, user-facing errors |

## Prerequisites

All examples require a `.env` file (or exported environment variables) with your kwtSMS API credentials:

```
KWTSMS_USERNAME=go_api_user
KWTSMS_PASSWORD=go_api_pass
KWTSMS_SENDER_ID=KWT-SMS
KWTSMS_TEST_MODE=1
```

Set `KWTSMS_TEST_MODE=1` to queue messages without actually delivering them.
