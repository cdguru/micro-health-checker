# Security policy

<!-- canonical-language: en -->
**English** | [Español](SECURITY.es.md)

## Supported versions

Until `v1.0.0`, security fixes are applied to the latest released minor version only.

## Reporting a vulnerability

Do not open a public GitHub issue for suspected vulnerabilities.

Use GitHub's **Report a vulnerability** private security advisory flow in this repository. Include:

- Affected version or commit
- Reproduction steps or proof of concept
- Expected impact
- Suggested mitigation, if known

You should receive an acknowledgement within 5 business days. A coordinated disclosure date will be agreed after triage.

## Deployment guidance

- Place the UI/API on a trusted network or behind an authenticated reverse proxy.
- Disable `POST /-/reload` unless it is operationally required.
- Inject secrets through environment variables or a secret manager.
- Use dedicated, read-only target credentials.
- Mount configuration read-only and `/data` read-write.
- Do not publish target error details to the public Internet.

Secrets committed to Git history must be considered compromised even after removal.
