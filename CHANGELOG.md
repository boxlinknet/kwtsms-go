# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-03-06

### Added

- `KwtSMS` client with `New()` and `FromEnv()` constructors
- `Verify()` to test credentials and check balance
- `Balance()` to get current balance
- `Send()` and `SendMulti()` for single and multi-number sends
- `SendWithRetry()` for automatic ERR028 retry
- Bulk send support (>200 numbers auto-batched with ERR013 retry)
- `Validate()` to validate phone numbers via the API
- `SenderIDs()` to list registered sender IDs
- `Coverage()` to list active country prefixes
- `Status()` to check message delivery status
- `DLR()` to retrieve delivery reports (international only)
- `NormalizePhone()` utility: Arabic/Persian digits, strip +/00/spaces/dashes
- `ValidatePhoneInput()` utility: catches email, empty, too short, too long, no digits
- `CleanMessage()` utility: strips emojis, HTML, BOM, zero-width chars, C0/C1 controls
- `APIErrors` map with all 33 error codes and developer-friendly action messages
- `EnrichError()` to add action guidance to API error responses
- Phone number deduplication before send
- Thread-safe cached balance with `sync.Mutex`
- JSONL logging with password masking
- `.env` file parser with environment variable priority
- CLI with verify, balance, send, validate, senderid, coverage, status, dlr commands
- Unit tests for phone normalization, message cleaning, error enrichment
- Mocked HTTP API tests for all error codes and endpoints
- Integration tests (build tag `integration`, uses `GO_USERNAME`/`GO_PASSWORD`)
- GitHub Actions CI (Go 1.18-1.22, race detector, integration on tag)

[0.1.0]: https://github.com/boxlinknet/kwtsms-go/releases/tag/v0.1.0
