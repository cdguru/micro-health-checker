# Changelog

All notable changes are documented here. The project follows [Semantic Versioning](https://semver.org/) and this file follows [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

### Added

- YAML configuration with environment-variable expansion and hot reload.
- TCP, HTTP and semantic PostgreSQL checks.
- Homepage-compatible live `GET`/`HEAD` endpoints with `200`/`503` responses.
- Embedded status UI, REST API and Prometheus metrics.
- SQLite storage by default and optional PostgreSQL storage.
- Automatic history retention and schema migrations.
- Distroless container image and Docker Compose example.
- Thirty-profile integration catalog with current support levels and copy-ready recipes.
- In-page navigation from every compatibility-matrix product to its configuration recipe.
- Bilingual public documentation with prominent language navigation and CI parity checks.

### Fixed

- Anchor the binary-only `.gitignore` rule so `cmd/micro-health-checker/main.go` is tracked and available to Docker builds.
- Docker Compose example: bind-mount the config directory instead of a single file so hot reload works when editors save atomically (previously the file-sharing layer silently dropped the change notification, requiring a container restart).

[Unreleased]: https://github.com/christiandente/micro-health-checker/commits/main
