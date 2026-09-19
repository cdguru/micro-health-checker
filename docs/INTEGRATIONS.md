# Integration catalog

<!-- canonical-language: en -->
**English** | [Español](es/INTEGRATIONS.md)

This catalog answers two separate questions for every target product:

1. **Can it be configured in `micro-health-checker` today?**
2. **What does that configuration actually prove?**

Those answers are intentionally not conflated. A successful TCP connection proves reachability; it does not prove that a database can authenticate, a broker can exchange protocol frames or a cluster can serve work.

## Support levels

| Level | Meaning |
| --- | --- |
| **Semantic** | An available driver performs a meaningful, read-only application operation. |
| **Recipe** | An available generic engine calls a health endpoint exposed by the product. The endpoint and any prerequisite feature are documented below. |
| **Connectivity only** | The product can be checked with `tcp` today, but its semantic driver is still planned. |

> [!IMPORTANT]
> The catalog contains 30 target profiles, not 30 completed native drivers. The current release implements the `postgres`, `http` and `tcp` engines. Every example below is honest about which one it uses.

All snippets are entries for the top-level `checks:` list. Environment references such as `${MYSQL_HOST}` must be set before the process starts. Default ports are examples; change them to match your deployment.

## Compatibility matrix

| # | Product or family | Support today | Engine / prerequisite | What a successful check proves |
| ---: | --- | --- | --- | --- |
| 1 | [PostgreSQL](#postgresql) | **Semantic** | `postgres` | Login, read-only transaction and query work |
| 2 | [MySQL](#mysql) | Connectivity only | `tcp` | Port accepts TCP connections |
| 3 | [MariaDB](#mariadb) | Connectivity only | `tcp` | Port accepts TCP connections |
| 4 | [Microsoft SQL Server](#microsoft-sql-server) | Connectivity only | `tcp` | Port accepts TCP connections |
| 5 | [Oracle Database](#oracle-database) | Connectivity only | `tcp` | Listener accepts TCP connections |
| 6 | [MongoDB](#mongodb) | Connectivity only | `tcp` | Port accepts TCP connections |
| 7 | [Redis / Valkey](#redis-valkey) | Connectivity only | `tcp` | Port accepts TCP connections |
| 8 | [Elasticsearch](#elasticsearch) | **Recipe** | `http`; cluster API enabled | Cluster health API responds and does not time out waiting for yellow |
| 9 | [OpenSearch](#opensearch) | **Recipe** | `http`; cluster API enabled | Cluster health API responds and does not time out waiting for yellow |
| 10 | [Cassandra / ScyllaDB](#cassandra-scylladb) | Connectivity only | `tcp` | CQL port accepts TCP connections |
| 11 | [ClickHouse](#clickhouse) | **Recipe** | `http`; HTTP interface enabled | `/ping` responds with `Ok` |
| 12 | [Neo4j](#neo4j) | Connectivity only | `tcp` | Bolt port accepts TCP connections |
| 13 | [Couchbase](#couchbase) | Connectivity only | `tcp` | Management port accepts TCP connections |
| 14 | [InfluxDB](#influxdb) | **Recipe** | `http`; InfluxDB 2.x health endpoint | `/health` returns HTTP 200 |
| 15 | [CockroachDB](#cockroachdb) | **Semantic** | `postgres`; PostgreSQL wire protocol | Login, read-only transaction and query work |
| 16 | [Aerospike](#aerospike) | Connectivity only | `tcp` | Service port accepts TCP connections |
| 17 | [Apache Kafka](#apache-kafka) | Connectivity only | `tcp` | Broker port accepts TCP connections |
| 18 | [RabbitMQ](#rabbitmq) | **Recipe** | `http`; management plugin enabled | RabbitMQ reports it is ready to serve clients |
| 19 | [NATS](#nats) | **Recipe** | `http`; monitoring server enabled | NATS monitoring health endpoint returns HTTP 200 |
| 20 | [MQTT brokers](#mqtt-brokers) | Connectivity only | `tcp` | Broker port accepts TCP connections |
| 21 | [Apache Pulsar](#apache-pulsar) | Connectivity only | `tcp` | Binary protocol port accepts TCP connections |
| 22 | [Memcached](#memcached) | Connectivity only | `tcp` | Port accepts TCP connections |
| 23 | [LDAP / Active Directory](#ldap-active-directory) | Connectivity only | `tcp` | LDAP or LDAPS port accepts TCP connections |
| 24 | [SMTP](#smtp) | Connectivity only | `tcp` | SMTP submission port accepts TCP connections |
| 25 | [S3 / MinIO](#s3-minio) | **Recipe, partial** | `http`; MinIO health endpoint | A MinIO node reports ready; authenticated S3 `HeadBucket` is planned |
| 26 | [Kubernetes API Server](#kubernetes-api-server) | **Recipe** | `http`; credentials may be required | Authenticated `/readyz` returns HTTP 200 |
| 27 | [Docker Engine](#docker-engine) | **Recipe, conditional** | `http`; safely exposed Engine API | Docker responds to `/_ping`; Unix sockets and mTLS are not yet supported |
| 28 | [Vault / OpenBao](#vault-openbao) | **Recipe** | `http` | Node is initialized, unsealed and accepted by the selected health policy |
| 29 | [Consul](#consul) | **Recipe** | `http` | Status API returns a leader address |
| 30 | [etcd](#etcd) | **Recipe** | `http`; HTTP health endpoint reachable | Health endpoint reports `health=true` |

## Semantic checks available now

<a id="postgresql"></a>
### PostgreSQL

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

Use a dedicated least-privilege login. The driver authenticates, opens a read-only transaction and requires the query to return at least one row.

<a id="cockroachdb"></a>
### CockroachDB

CockroachDB speaks the PostgreSQL wire protocol, so the existing semantic driver can be reused:

```yaml
- id: cockroach-prod
  name: CockroachDB Production
  type: postgres
  interval: 15s
  timeout: 3s
  postgres:
    dsn: ${COCKROACH_PROD_DSN}
    query: SELECT 1
```

## HTTP recipes available now

These recipes use health interfaces supplied by each product. They are useful and runnable today, but they are not native protocol drivers inside `micro-health-checker`.

<a id="elasticsearch"></a>
### Elasticsearch

```yaml
- id: elasticsearch-prod
  name: Elasticsearch Production
  type: http
  http:
    url: ${ELASTICSEARCH_URL}/_cluster/health?wait_for_status=yellow&timeout=2s
    method: GET
    expected_status: [200]
    body_contains: '"timed_out":false'
    headers:
      Authorization: ${ELASTICSEARCH_AUTH_HEADER}
```

Set `ELASTICSEARCH_AUTH_HEADER` to the complete value expected by Elasticsearch, for example `ApiKey …` or `Basic …`.

<a id="opensearch"></a>
### OpenSearch

```yaml
- id: opensearch-prod
  name: OpenSearch Production
  type: http
  http:
    url: ${OPENSEARCH_URL}/_cluster/health?wait_for_status=yellow&timeout=2s
    method: GET
    expected_status: [200]
    body_contains: '"timed_out":false'
    headers:
      Authorization: ${OPENSEARCH_AUTH_HEADER}
```

<a id="clickhouse"></a>
### ClickHouse

```yaml
- id: clickhouse-prod
  name: ClickHouse Production
  type: http
  http:
    url: ${CLICKHOUSE_URL}/ping
    method: GET
    expected_status: [200]
    body_contains: Ok
```

<a id="influxdb"></a>
### InfluxDB 2.x

```yaml
- id: influxdb-prod
  name: InfluxDB Production
  type: http
  http:
    url: ${INFLUXDB_URL}/health
    method: GET
    expected_status: [200]
```

Confirm the endpoint for your InfluxDB major version; the example targets InfluxDB 2.x.

<a id="rabbitmq"></a>
### RabbitMQ

Requires the management plugin and a user allowed to access its HTTP API.

```yaml
- id: rabbitmq-prod
  name: RabbitMQ Production
  type: http
  http:
    url: ${RABBITMQ_MANAGEMENT_URL}/api/health/checks/ready-to-serve-clients
    method: GET
    expected_status: [200]
    headers:
      Authorization: ${RABBITMQ_AUTH_HEADER}
```

This uses RabbitMQ's health API. A native AMQP connection/channel check remains planned.

<a id="nats"></a>
### NATS

Requires the NATS monitoring HTTP server, commonly enabled on a separate port such as `8222`.

```yaml
- id: nats-prod
  name: NATS Production
  type: http
  http:
    url: ${NATS_MONITORING_URL}/healthz
    method: GET
    expected_status: [200]
```

This does not perform the planned native `CONNECT` plus `PING`/`PONG` exchange.

<a id="s3-minio"></a>
### S3 / MinIO

```yaml
- id: minio-prod
  name: MinIO Production
  type: http
  http:
    url: ${MINIO_URL}/minio/health/ready
    method: GET
    expected_status: [200]
```

This recipe is MinIO-specific. Generic S3 and cloud object storage need the planned authenticated `HeadBucket` driver.

<a id="kubernetes-api-server"></a>
### Kubernetes API Server

```yaml
- id: kubernetes-api
  name: Kubernetes API Server
  type: http
  http:
    url: ${KUBERNETES_API_URL}/readyz
    method: GET
    expected_status: [200]
    headers:
      Authorization: Bearer ${KUBERNETES_TOKEN}
```

Use a narrowly scoped service account and the cluster CA in production. The current HTTP engine uses the host trust store and does not yet accept a custom CA file.

<a id="docker-engine"></a>
### Docker Engine

```yaml
- id: docker-engine
  name: Docker Engine
  type: http
  http:
    url: ${DOCKER_API_URL}/_ping
    method: GET
    expected_status: [200]
    body_contains: OK
```

The current engine cannot dial a Unix socket or present a TLS client certificate. Point this recipe at a properly authenticated HTTPS gateway; **never expose an unauthenticated Docker TCP socket to an untrusted network**.

<a id="vault-openbao"></a>
### Vault / OpenBao

```yaml
- id: vault-prod
  name: Vault Production
  type: http
  http:
    url: ${VAULT_URL}/v1/sys/health?standbyok=true&perfstandbyok=true
    method: GET
    expected_status: [200]
```

This policy accepts active and eligible standby nodes but rejects sealed or uninitialized nodes. Adjust the URL and expected statuses if your load balancer intentionally targets other replication roles.

<a id="consul"></a>
### Consul

```yaml
- id: consul-prod
  name: Consul Production
  type: http
  http:
    url: ${CONSUL_URL}/v1/status/leader
    method: GET
    expected_status: [200]
    body_contains: ':'
    headers:
      X-Consul-Token: ${CONSUL_HTTP_TOKEN}
```

The body assertion rejects Consul's empty leader value. Remove the header if ACLs are disabled on a trusted development environment.

<a id="etcd"></a>
### etcd

```yaml
- id: etcd-prod
  name: etcd Production
  type: http
  http:
    url: ${ETCD_URL}/health
    method: GET
    expected_status: [200]
    body_contains: '"health":"true"'
```

This recipe depends on the HTTP health endpoint exposed by your etcd version and proxy configuration. Native gRPC maintenance status remains planned.

## Connectivity-only configurations

The following configurations are valid now, but a green result means only that a TCP connection could be established. They are useful as an interim signal and for network diagnosis; they must not be described as proof of application health.

<a id="mysql"></a>
### MySQL

```yaml
- id: mysql-prod
  name: MySQL Production
  type: tcp
  tcp:
    address: ${MYSQL_HOST}:3306
```

The planned semantic driver will authenticate and execute `SELECT 1`.

<a id="mariadb"></a>
### MariaDB

```yaml
- id: mariadb-prod
  name: MariaDB Production
  type: tcp
  tcp:
    address: ${MARIADB_HOST}:3306
```

The planned semantic driver will reuse the MySQL engine to authenticate and execute `SELECT 1`.

<a id="microsoft-sql-server"></a>
### Microsoft SQL Server

```yaml
- id: mssql-prod
  name: Microsoft SQL Server Production
  type: tcp
  tcp:
    address: ${MSSQL_HOST}:1433
```

The planned semantic driver will authenticate and execute `SELECT 1`.

<a id="oracle-database"></a>
### Oracle Database

```yaml
- id: oracle-prod
  name: Oracle Database Production
  type: tcp
  tcp:
    address: ${ORACLE_HOST}:1521
```

The planned semantic driver will authenticate and execute `SELECT 1 FROM DUAL`.

<a id="mongodb"></a>
### MongoDB

```yaml
- id: mongodb-prod
  name: MongoDB Production
  type: tcp
  tcp:
    address: ${MONGODB_HOST}:27017
```

The planned semantic driver will execute MongoDB's `ping` command.

<a id="redis-valkey"></a>
### Redis / Valkey

```yaml
- id: redis-prod
  name: Redis Production
  type: tcp
  tcp:
    address: ${REDIS_HOST}:6379
```

The planned semantic driver will exchange `PING`/`PONG`, with an optional role assertion.

<a id="cassandra-scylladb"></a>
### Cassandra / ScyllaDB

```yaml
- id: cassandra-prod
  name: Cassandra Production
  type: tcp
  tcp:
    address: ${CASSANDRA_HOST}:9042
```

The planned semantic driver will execute a read-only CQL query against `system.local`.

<a id="neo4j"></a>
### Neo4j

```yaml
- id: neo4j-prod
  name: Neo4j Production
  type: tcp
  tcp:
    address: ${NEO4J_HOST}:7687
```

The planned semantic driver will open a Bolt session and execute `RETURN 1`.

<a id="couchbase"></a>
### Couchbase

```yaml
- id: couchbase-prod
  name: Couchbase Production
  type: tcp
  tcp:
    address: ${COUCHBASE_HOST}:8091
```

The planned semantic driver will use the Couchbase SDK ping and diagnostics operations.

<a id="aerospike"></a>
### Aerospike

```yaml
- id: aerospike-prod
  name: Aerospike Production
  type: tcp
  tcp:
    address: ${AEROSPIKE_HOST}:3000
```

The planned semantic driver will use the info protocol with an optional namespace assertion.

<a id="memcached"></a>
### Memcached

```yaml
- id: memcached-prod
  name: Memcached Production
  type: tcp
  tcp:
    address: ${MEMCACHED_HOST}:11211
```

The planned semantic driver will issue the read-only `version` or `stats` command.

<a id="apache-kafka"></a>
### Apache Kafka

```yaml
- id: kafka-prod
  name: Apache Kafka Production
  type: tcp
  tcp:
    address: ${KAFKA_HOST}:9092
```

The planned semantic driver will request API versions and cluster metadata.

<a id="mqtt-brokers"></a>
### MQTT brokers

```yaml
- id: mqtt-prod
  name: MQTT Broker Production
  type: tcp
  tcp:
    address: ${MQTT_HOST}:1883
```

The planned semantic driver will exchange MQTT `CONNECT`/`CONNACK` frames.

<a id="apache-pulsar"></a>
### Apache Pulsar

```yaml
- id: pulsar-prod
  name: Apache Pulsar Production
  type: tcp
  tcp:
    address: ${PULSAR_HOST}:6650
```

The planned semantic driver will validate broker health and lookup operations.

<a id="ldap-active-directory"></a>
### LDAP / Active Directory

Choose the endpoint that matches the deployment:

```yaml
- id: ldap-prod
  name: LDAP Production
  type: tcp
  tcp:
    address: ${LDAP_HOST}:389

- id: ldaps-prod
  name: Active Directory LDAPS
  type: tcp
  tcp:
    address: ${ACTIVE_DIRECTORY_HOST}:636
```

These are alternatives for the same catalog profile. The planned semantic driver will perform a bind plus a base search.

<a id="smtp"></a>
### SMTP

```yaml
- id: smtp-prod
  name: SMTP Production
  type: tcp
  tcp:
    address: ${SMTP_HOST}:587
```

The planned semantic driver will validate the banner, `EHLO` and optionally STARTTLS without sending mail.

## Choosing a check

Use the highest available level that matches your installation:

1. Prefer a **semantic** driver when available.
2. Otherwise use an official product health endpoint through an **HTTP recipe**.
3. Use **TCP connectivity** as an explicitly labeled interim check, not as evidence that the application is healthy.

HTTP and TCP are reusable engines, not product plug-ins. New semantic protocols require code, tests and dependencies, but users never need to download runtime plug-ins: supported drivers are compiled into the single binary.
