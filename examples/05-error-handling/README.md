# 05 - Error Handling

Demonstrates every error handling pattern available in kwtsms-go, from input validation to API error mapping.

## Patterns covered

1. **Phone validation**: `ValidatePhoneInput()` catches empty input, emails, non-numeric text, too-short, and too-long numbers
2. **Message cleaning**: `CleanMessage()` strips emojis, HTML, and hidden characters. Check for empty results before sending.
3. **API send errors**: `Send()` returns structured `SendResult` with error code, description, and recommended action
4. **Error code mapping**: Convert raw API codes (ERR003, ERR006, etc.) to safe, user-facing messages using `kwtsms.APIErrors`

## Running

```bash
cd examples/05-error-handling
cp ../../.env.example .env   # edit with your credentials
go run main.go
```

Patterns 1 and 2 run without credentials.
Patterns 3 and 4 require valid API credentials but will skip gracefully if none are found.
