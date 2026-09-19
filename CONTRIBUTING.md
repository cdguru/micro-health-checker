# Contributing

<!-- canonical-language: en -->
**English** | [Español](CONTRIBUTING.es.md)

Thank you for helping make semantic service health easier to operate.

## Before writing code

- Search existing issues and the integration catalog.
- Open a feature proposal before adding a new driver or changing the public configuration/API.
- Keep default checks read-only and minimally invasive.
- Never include credentials, production addresses or captured private responses in tests.

## Development

Requirements: Go 1.26+, Git and optionally Docker.

```bash
git clone https://github.com/christiandente/micro-health-checker.git
cd micro-health-checker
go mod download
make test
make build
```

Before submitting:

```bash
make fmt
make vet
make test-race
make docs-check
make build
```

## Driver expectations

A new product driver should include:

1. Strict YAML validation and documented defaults.
2. Context cancellation and bounded timeouts.
3. TLS and authentication support appropriate to the protocol.
4. No mutation by default.
5. Unit tests and, where practical, a containerized integration test.
6. Metrics and UI behavior through the common scheduler.
7. Configuration and integration-catalog documentation in English and Spanish.

Prefer a reusable engine or declarative preset when a product exposes an ordinary HTTP, SQL or gRPC health contract.

## Pull requests

- Keep changes focused.
- Explain user impact and operational tradeoffs.
- Add a changelog entry under `Unreleased` for user-visible changes.
- Use conventional commit-style subjects when practical, such as `feat(redis): add PING check`.
- Update the canonical English documentation and its Spanish translation in the same pull request.
- Keep private architecture and roadmap planning outside the public repository tree.
- Confirm that you have the right to submit the contribution under Apache-2.0.

By submitting a contribution, you agree that it is licensed under the repository's Apache License 2.0.
