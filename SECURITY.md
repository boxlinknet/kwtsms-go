# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.2.x   | Yes       |
| 0.1.x   | Yes       |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public GitHub issue.
2. Email **security@kwtsms.com** with a description of the vulnerability.
3. Include steps to reproduce if possible.
4. Allow up to 72 hours for an initial response.

## Security Design

The library follows these security principles:

- **Credentials are never logged.** Passwords are masked as `***` in all JSONL log entries.
- **Always POST, never GET.** GET requests log credentials in server logs even over HTTPS.
- **Content-Type and Accept headers** are always set to `application/json`.
- **HTTP timeout** is set to 15 seconds to prevent hanging connections.
- **No credential storage.** The library reads credentials from environment variables or `.env` files at runtime. It never writes, caches, or transmits credentials anywhere other than the kwtSMS API.
- **Input sanitization.** Phone numbers and messages are cleaned before every API call.
- **No external dependencies.** Zero third-party code reduces supply chain risk.

## Credential Safety

- Never hardcode credentials in source code.
- Store credentials in environment variables, `.env` files (gitignored), or a secrets manager.
- The `.env` file must be listed in `.gitignore`.
- Use `FromEnv()` as the primary way to create a client.
