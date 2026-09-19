# Configuration reference

<!-- canonical-language: en -->
**English** | [Español](es/CONFIGURATION.md)

`micro-health-checker` loads one YAML file. The path defaults to `/etc/micro-health-checker/config.yml` and can be changed with `-config` or `MHC_CONFIG`.

Unknown YAML fields are rejected. References in the form `${VARIABLE_NAME}` are expanded before parsing; startup or reload fails if a referenced variable is unset.

This page documents the configuration engines. For product-specific examples and an honest semantic-versus-connectivity support matrix, see the [integration catalog](INTEGRATIONS.md).

## Server

```yaml
server:
  address: ":8080"
  read_timeout: 5s
  write_timeout: 10s
  enable_reload_endpoint: false
```

`enable_reload_endpoint` exposes `POST /-/reload`. Leave it disabled unless network access to the service is controlled.

## Scheduler

```yaml
scheduler:
  default_interval: 30s
  default_timeout: 5s
```

Each check may override these values. Durations accept Go duration syntax such as `500ms`, `15s`, `5m`, `24h`; retention additionally supports values such as `30d`.

## Storage

### SQLite — default

```yaml
storage:
  type: sqlite
  retention: 30d
  sqlite:
    path: /data/micro-health-checker.db
```

SQLite runs in WAL mode with a busy timeout and automatic schema creation. Persist `/data` when running the container.

### PostgreSQL

```yaml
storage:
  type: postgres
  retention: 90d
  postgres:
    dsn: ${MHC_DATABASE_URL}
```

The database and user must already exist. Tables and indexes are created automatically. Changing the storage backend or DSN requires a process restart.

## Common check fields

```yaml
- id: unique-machine-id
  name: Human-readable name
  type: tcp | http | postgres
  enabled: true
  interval: 30s
  timeout: 5s
```

- `id`: required, unique, maximum 64 characters; letters, numbers, `_` and `-`.
- `name`: optional; defaults to `id`.
- `enabled`: optional; defaults to `true`.
- `interval`: optional; defaults to the scheduler value.
- `timeout`: optional; defaults to the scheduler value.

## TCP

```yaml
- id: bacula-director
  name: Bacula Director
  type: tcp
  tcp:
    address: bacula.internal:9101
```

Success only proves that a TCP connection can be established. It does not prove protocol or application health.

## HTTP

```yaml
- id: private-api
  name: Private API
  type: http
  http:
    url: https://api.internal/ready
    method: GET
    expected_status: [200, 204]
    body_contains: ready
    follow_redirects: false
    insecure_skip_verify: false
    headers:
      Authorization: Bearer ${PRIVATE_API_TOKEN}
```

When `expected_status` is omitted, every `2xx` response succeeds. Response bodies are capped at 1 MiB. `body_contains` performs a literal substring assertion.

## PostgreSQL

```yaml
- id: postgres-prod
  name: PostgreSQL Production
  type: postgres
  interval: 15s
  timeout: 3s
  postgres:
    dsn: ${POSTGRES_PROD_DSN}
    query: SELECT 1
```

Every execution opens a read-only transaction and requires at least one result row. Use a dedicated login with `CONNECT` permission and only the minimum permissions required by a custom query.

## Hot reload

The parent directory of the YAML file is watched so editor-style atomic replacements are detected. Check changes are debounced, fully parsed and validated before being applied.

Reloadable:

- Addition, deletion or modification of checks
- Check intervals and timeouts
- Scheduler defaults as inherited by checks

Restart required:

- Server address and timeouts
- Storage backend, path or DSN
- Reload endpoint setting

If a change is invalid, the existing workers continue using the last-known-good configuration.
