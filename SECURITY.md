# Security Policy

## Supported Versions

We release security fixes for the latest stable version of `abaper-mcp`.

| Version | Supported |
|---------|-----------|
| Latest  | Yes       |
| Older   | No        |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Please report security issues by emailing **security@bluefunda.com** with:

- A description of the vulnerability
- Steps to reproduce
- Potential impact
- Any suggested mitigations

We will acknowledge your report within 48 hours and aim to release a fix within 7 days for critical issues.

## Security Practices

- `abaper-mcp` never connects to SAP directly — all ADT operations are delegated to the `abaper` backend over HTTP; no SAP credentials are held or stored by this binary.
- Release archives are checksummed (SHA256), signed (Sigstore/cosign, keyless), and shipped with a Software Bill of Materials (SBOM) covering every Go dependency.
- The SSE/HTTP transport supports optional bearer-token authentication (`ABAPER_AUTH_TOKEN`); when unset, the server logs a warning and expects to be fronted by an authenticating gateway.
- Batch script execution (`s4-batch-analyze`) is restricted to an explicit allowlist (`S4_ALLOWED_SCRIPTS`) rather than accepting arbitrary script names.
