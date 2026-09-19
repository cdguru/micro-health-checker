# Catálogo de integraciones

<!-- translation-source: ../INTEGRATIONS.md -->
[English](../INTEGRATIONS.md) | **Español**

> Esta es la traducción española. La documentación inglesa es la fuente canónica ante cualquier diferencia.

Este catálogo responde dos preguntas diferentes para cada producto objetivo:

1. **¿Puede configurarse actualmente en `micro-health-checker`?**
2. **¿Qué demuestra realmente esa configuración?**

Estas respuestas no se mezclan intencionalmente. Una conexión TCP exitosa demuestra alcance de red; no demuestra que una base de datos pueda autenticar, que un broker pueda intercambiar frames del protocolo ni que un clúster pueda procesar trabajo.

## Niveles de soporte

| Nivel | Significado |
| --- | --- |
| **Semántico** | Un driver disponible ejecuta una operación significativa y de solo lectura sobre la aplicación. |
| **Receta** | Un motor genérico disponible llama a un endpoint de salud expuesto por el producto. El endpoint y sus requisitos se documentan más abajo. |
| **Sólo conectividad** | El producto puede comprobarse actualmente con `tcp`, pero su driver semántico continúa planificado. |

> [!IMPORTANT]
> El catálogo contiene 30 perfiles objetivo, no 30 drivers nativos terminados. La versión actual implementa los motores `postgres`, `http` y `tcp`. Cada ejemplo siguiente indica honestamente cuál utiliza.

Todos los fragmentos son elementos de la lista superior `checks:`. Las referencias de entorno como `${MYSQL_HOST}` deben estar definidas antes de iniciar el proceso. Los puertos predeterminados son ejemplos; cambialos según tu despliegue.

## Matriz de compatibilidad

| # | Producto o familia | Soporte actual | Motor / requisito | Qué demuestra un check exitoso |
| ---: | --- | --- | --- | --- |
| 1 | [PostgreSQL](#postgresql) | **Semántico** | `postgres` | Funcionan login, transacción de solo lectura y consulta |
| 2 | [MySQL](#mysql) | Sólo conectividad | `tcp` | El puerto acepta conexiones TCP |
| 3 | [MariaDB](#mariadb) | Sólo conectividad | `tcp` | El puerto acepta conexiones TCP |
| 4 | [Microsoft SQL Server](#microsoft-sql-server) | Sólo conectividad | `tcp` | El puerto acepta conexiones TCP |
| 5 | [Oracle Database](#oracle-database) | Sólo conectividad | `tcp` | El listener acepta conexiones TCP |
| 6 | [MongoDB](#mongodb) | Sólo conectividad | `tcp` | El puerto acepta conexiones TCP |
| 7 | [Redis / Valkey](#redis-valkey) | Sólo conectividad | `tcp` | El puerto acepta conexiones TCP |
| 8 | [Elasticsearch](#elasticsearch) | **Receta** | `http`; API de clúster habilitada | La API de salud responde sin timeout al esperar estado yellow |
| 9 | [OpenSearch](#opensearch) | **Receta** | `http`; API de clúster habilitada | La API de salud responde sin timeout al esperar estado yellow |
| 10 | [Cassandra / ScyllaDB](#cassandra-scylladb) | Sólo conectividad | `tcp` | El puerto CQL acepta conexiones TCP |
| 11 | [ClickHouse](#clickhouse) | **Receta** | `http`; interfaz HTTP habilitada | `/ping` responde con `Ok` |
| 12 | [Neo4j](#neo4j) | Sólo conectividad | `tcp` | El puerto Bolt acepta conexiones TCP |
| 13 | [Couchbase](#couchbase) | Sólo conectividad | `tcp` | El puerto de administración acepta conexiones TCP |
| 14 | [InfluxDB](#influxdb) | **Receta** | `http`; endpoint de salud de InfluxDB 2.x | `/health` devuelve HTTP 200 |
| 15 | [CockroachDB](#cockroachdb) | **Semántico** | `postgres`; protocolo wire de PostgreSQL | Funcionan login, transacción de solo lectura y consulta |
| 16 | [Aerospike](#aerospike) | Sólo conectividad | `tcp` | El puerto de servicio acepta conexiones TCP |
| 17 | [Apache Kafka](#apache-kafka) | Sólo conectividad | `tcp` | El puerto del broker acepta conexiones TCP |
| 18 | [RabbitMQ](#rabbitmq) | **Receta** | `http`; plugin de administración habilitado | RabbitMQ informa que está listo para servir clientes |
| 19 | [NATS](#nats) | **Receta** | `http`; servidor de monitoreo habilitado | El endpoint de salud de NATS devuelve HTTP 200 |
| 20 | [Brokers MQTT](#mqtt-brokers) | Sólo conectividad | `tcp` | El puerto del broker acepta conexiones TCP |
| 21 | [Apache Pulsar](#apache-pulsar) | Sólo conectividad | `tcp` | El puerto del protocolo binario acepta conexiones TCP |
| 22 | [Memcached](#memcached) | Sólo conectividad | `tcp` | El puerto acepta conexiones TCP |
| 23 | [LDAP / Active Directory](#ldap-active-directory) | Sólo conectividad | `tcp` | El puerto LDAP o LDAPS acepta conexiones TCP |
| 24 | [SMTP](#smtp) | Sólo conectividad | `tcp` | El puerto de submission SMTP acepta conexiones TCP |
| 25 | [S3 / MinIO](#s3-minio) | **Receta, parcial** | `http`; endpoint de salud MinIO | Un nodo MinIO informa readiness; S3 `HeadBucket` autenticado está planificado |
| 26 | [Kubernetes API Server](#kubernetes-api-server) | **Receta** | `http`; puede requerir credenciales | `/readyz` autenticado devuelve HTTP 200 |
| 27 | [Docker Engine](#docker-engine) | **Receta, condicional** | `http`; API Engine expuesta de forma segura | Docker responde a `/_ping`; todavía no se soportan sockets Unix ni mTLS |
| 28 | [Vault / OpenBao](#vault-openbao) | **Receta** | `http` | El nodo está inicializado, unsealed y aceptado por la política elegida |
| 29 | [Consul](#consul) | **Receta** | `http` | La API de estado devuelve una dirección de líder |
| 30 | [etcd](#etcd) | **Receta** | `http`; endpoint HTTP de salud accesible | El endpoint informa `health=true` |

## Checks semánticos disponibles actualmente

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

Utilizá un login dedicado con privilegios mínimos. El driver autentica, abre una transacción de solo lectura y exige que la consulta devuelva al menos una fila.

<a id="cockroachdb"></a>
### CockroachDB

CockroachDB utiliza el protocolo wire de PostgreSQL, por lo que puede reutilizarse el driver semántico existente:

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

## Recetas HTTP disponibles actualmente

Estas recetas utilizan interfaces de salud provistas por cada producto. Son útiles y pueden ejecutarse actualmente, pero no son drivers nativos de protocolo dentro de `micro-health-checker`.

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

Definí `ELASTICSEARCH_AUTH_HEADER` con el valor completo esperado por Elasticsearch, por ejemplo `ApiKey …` o `Basic …`.

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

Confirmá el endpoint correspondiente a la versión principal de InfluxDB; este ejemplo apunta a InfluxDB 2.x.

<a id="rabbitmq"></a>
### RabbitMQ

Requiere el plugin de administración y un usuario autorizado para acceder a su API HTTP.

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

Esta receta utiliza la API de salud de RabbitMQ. Un check nativo de conexión y canal AMQP continúa planificado.

<a id="nats"></a>
### NATS

Requiere el servidor HTTP de monitoreo de NATS, normalmente habilitado en un puerto separado como `8222`.

```yaml
- id: nats-prod
  name: NATS Production
  type: http
  http:
    url: ${NATS_MONITORING_URL}/healthz
    method: GET
    expected_status: [200]
```

Esta receta no realiza todavía el intercambio nativo planificado `CONNECT` más `PING`/`PONG`.

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

Esta receta es específica para MinIO. S3 genérico y los almacenamientos de objetos cloud requieren el driver autenticado `HeadBucket` planificado.

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

Utilizá una service account con alcance mínimo y la CA del clúster en producción. El motor HTTP actual utiliza el trust store del host y todavía no acepta un archivo CA personalizado.

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

El motor actual no puede conectarse a un socket Unix ni presentar un certificado TLS de cliente. Apuntá esta receta a un gateway HTTPS correctamente autenticado; **nunca expongas un socket TCP de Docker sin autenticación a una red no confiable**.

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

Esta política acepta nodos activos y standby elegibles, pero rechaza nodos sealed o sin inicializar. Ajustá la URL y los estados esperados si tu balanceador apunta intencionalmente a otros roles de replicación.

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

La aserción sobre el body rechaza el valor vacío de líder de Consul. Quitá el header si las ACL están deshabilitadas en un entorno de desarrollo confiable.

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

Esta receta depende del endpoint HTTP de salud expuesto por tu versión y configuración de proxy de etcd. El estado nativo de mantenimiento gRPC continúa planificado.

## Configuraciones de sólo conectividad

Las siguientes configuraciones son válidas actualmente, pero un resultado verde solamente significa que pudo establecerse una conexión TCP. Son útiles como señal transitoria y para diagnosticar la red; no deben describirse como prueba de la salud de la aplicación.

<a id="mysql"></a>
### MySQL

```yaml
- id: mysql-prod
  name: MySQL Production
  type: tcp
  tcp:
    address: ${MYSQL_HOST}:3306
```

El driver semántico planificado autenticará y ejecutará `SELECT 1`.

<a id="mariadb"></a>
### MariaDB

```yaml
- id: mariadb-prod
  name: MariaDB Production
  type: tcp
  tcp:
    address: ${MARIADB_HOST}:3306
```

El driver semántico planificado reutilizará el motor MySQL para autenticar y ejecutar `SELECT 1`.

<a id="microsoft-sql-server"></a>
### Microsoft SQL Server

```yaml
- id: mssql-prod
  name: Microsoft SQL Server Production
  type: tcp
  tcp:
    address: ${MSSQL_HOST}:1433
```

El driver semántico planificado autenticará y ejecutará `SELECT 1`.

<a id="oracle-database"></a>
### Oracle Database

```yaml
- id: oracle-prod
  name: Oracle Database Production
  type: tcp
  tcp:
    address: ${ORACLE_HOST}:1521
```

El driver semántico planificado autenticará y ejecutará `SELECT 1 FROM DUAL`.

<a id="mongodb"></a>
### MongoDB

```yaml
- id: mongodb-prod
  name: MongoDB Production
  type: tcp
  tcp:
    address: ${MONGODB_HOST}:27017
```

El driver semántico planificado ejecutará el comando `ping` de MongoDB.

<a id="redis-valkey"></a>
### Redis / Valkey

```yaml
- id: redis-prod
  name: Redis Production
  type: tcp
  tcp:
    address: ${REDIS_HOST}:6379
```

El driver semántico planificado intercambiará `PING`/`PONG`, con una aserción opcional de rol.

<a id="cassandra-scylladb"></a>
### Cassandra / ScyllaDB

```yaml
- id: cassandra-prod
  name: Cassandra Production
  type: tcp
  tcp:
    address: ${CASSANDRA_HOST}:9042
```

El driver semántico planificado ejecutará una consulta CQL de solo lectura contra `system.local`.

<a id="neo4j"></a>
### Neo4j

```yaml
- id: neo4j-prod
  name: Neo4j Production
  type: tcp
  tcp:
    address: ${NEO4J_HOST}:7687
```

El driver semántico planificado abrirá una sesión Bolt y ejecutará `RETURN 1`.

<a id="couchbase"></a>
### Couchbase

```yaml
- id: couchbase-prod
  name: Couchbase Production
  type: tcp
  tcp:
    address: ${COUCHBASE_HOST}:8091
```

El driver semántico planificado utilizará las operaciones ping y diagnostics del SDK de Couchbase.

<a id="aerospike"></a>
### Aerospike

```yaml
- id: aerospike-prod
  name: Aerospike Production
  type: tcp
  tcp:
    address: ${AEROSPIKE_HOST}:3000
```

El driver semántico planificado utilizará el protocolo info con una aserción opcional de namespace.

<a id="memcached"></a>
### Memcached

```yaml
- id: memcached-prod
  name: Memcached Production
  type: tcp
  tcp:
    address: ${MEMCACHED_HOST}:11211
```

El driver semántico planificado enviará el comando de solo lectura `version` o `stats`.

<a id="apache-kafka"></a>
### Apache Kafka

```yaml
- id: kafka-prod
  name: Apache Kafka Production
  type: tcp
  tcp:
    address: ${KAFKA_HOST}:9092
```

El driver semántico planificado solicitará las versiones de API y la metadata del clúster.

<a id="mqtt-brokers"></a>
### Brokers MQTT

```yaml
- id: mqtt-prod
  name: MQTT Broker Production
  type: tcp
  tcp:
    address: ${MQTT_HOST}:1883
```

El driver semántico planificado intercambiará frames MQTT `CONNECT`/`CONNACK`.

<a id="apache-pulsar"></a>
### Apache Pulsar

```yaml
- id: pulsar-prod
  name: Apache Pulsar Production
  type: tcp
  tcp:
    address: ${PULSAR_HOST}:6650
```

El driver semántico planificado validará la salud del broker y las operaciones de lookup.

<a id="ldap-active-directory"></a>
### LDAP / Active Directory

Elegí el endpoint correspondiente al despliegue:

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

Estas son alternativas para el mismo perfil del catálogo. El driver semántico planificado realizará un bind seguido por una búsqueda base.

<a id="smtp"></a>
### SMTP

```yaml
- id: smtp-prod
  name: SMTP Production
  type: tcp
  tcp:
    address: ${SMTP_HOST}:587
```

El driver semántico planificado validará el banner, `EHLO` y opcionalmente STARTTLS sin enviar correo.

## Elegir un check

Utilizá el nivel más alto disponible que corresponda a tu instalación:

1. Preferí un driver **semántico** cuando esté disponible.
2. En caso contrario, utilizá un endpoint oficial de salud del producto mediante una **receta HTTP**.
3. Utilizá la **conectividad TCP** como un check transitorio etiquetado explícitamente, no como evidencia de que la aplicación está saludable.

HTTP y TCP son motores reutilizables, no plugins de producto. Los nuevos protocolos semánticos requieren código, pruebas y dependencias, pero los usuarios nunca necesitan descargar plugins de runtime: los drivers soportados se compilan dentro del binario único.
