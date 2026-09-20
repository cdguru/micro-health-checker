# Referencia de configuración

<!-- translation-source: ../CONFIGURATION.md -->
[English](../CONFIGURATION.md) | **Español**

> Esta es la traducción española. La documentación inglesa es la fuente canónica ante cualquier diferencia.

`micro-health-checker` carga un archivo YAML. La ruta predeterminada es `/etc/micro-health-checker/config.yml` y puede cambiarse con `-config` o `MHC_CONFIG`.

Los campos YAML desconocidos son rechazados. Las referencias con formato `${VARIABLE_NAME}` se expanden antes de interpretar el archivo; el inicio o la recarga fallan si una variable referenciada no está definida.

Esta página documenta los motores de configuración. Para ejemplos específicos por producto y una matriz honesta de soporte semántico frente a conectividad, consultá el [catálogo de integraciones](INTEGRATIONS.md).

## Servidor

```yaml
server:
  address: ":8080"
  read_timeout: 5s
  write_timeout: 10s
  enable_reload_endpoint: false
```

`enable_reload_endpoint` expone `POST /-/reload`. Dejalo deshabilitado salvo que el acceso de red al servicio esté controlado.

## Scheduler

```yaml
scheduler:
  default_interval: 30s
  default_timeout: 5s
```

Cada check puede sobrescribir estos valores. Las duraciones aceptan la sintaxis de Go, por ejemplo `500ms`, `15s`, `5m` o `24h`; la retención también admite valores como `30d`.

## Almacenamiento

### SQLite — predeterminado

```yaml
storage:
  type: sqlite
  retention: 30d
  sqlite:
    path: /data/micro-health-checker.db
```

SQLite se ejecuta en modo WAL, con busy timeout y creación automática del esquema. Persistí `/data` al ejecutar el contenedor.

### PostgreSQL

```yaml
storage:
  type: postgres
  retention: 90d
  postgres:
    dsn: ${MHC_DATABASE_URL}
```

La base de datos y el usuario deben existir previamente. Las tablas e índices se crean automáticamente. Cambiar el backend de almacenamiento o su DSN requiere reiniciar el proceso.

## Campos comunes de un check

```yaml
- id: unique-machine-id
  name: Human-readable name
  type: tcp | http | postgres
  enabled: true
  interval: 30s
  timeout: 5s
```

- `id`: obligatorio, único y con un máximo de 64 caracteres; admite letras, números, `_` y `-`.
- `name`: opcional; utiliza `id` de manera predeterminada.
- `enabled`: opcional; su valor predeterminado es `true`.
- `interval`: opcional; utiliza el valor del scheduler de manera predeterminada.
- `timeout`: opcional; utiliza el valor del scheduler de manera predeterminada.

## TCP

```yaml
- id: bacula-director
  name: Bacula Director
  type: tcp
  tcp:
    address: bacula.internal:9101
```

Un resultado exitoso solamente demuestra que se pudo establecer una conexión TCP. No demuestra la salud del protocolo ni de la aplicación.

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

Cuando se omite `expected_status`, cualquier respuesta `2xx` es exitosa. Los bodies de respuesta están limitados a 1 MiB. `body_contains` realiza una aserción literal de substring.

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

Cada ejecución abre una transacción de solo lectura y requiere al menos una fila como resultado. Utilizá un login dedicado con permiso `CONNECT` y solamente los privilegios mínimos requeridos por una consulta personalizada.

## Recarga automática

El directorio padre del archivo YAML es observado para detectar reemplazos atómicos utilizados por los editores. Los cambios de checks se agrupan, interpretan y validan completamente antes de aplicarse.

Se puede recargar:

- Agregado, eliminación o modificación de checks
- Intervalos y timeouts de checks
- Valores predeterminados del scheduler heredados por los checks

Requiere reiniciar:

- Dirección y timeouts del servidor
- Backend, ruta o DSN de almacenamiento
- Configuración del endpoint de recarga

Si un cambio es inválido, los workers existentes continúan usando la última configuración válida.

### Docker: montá el directorio, no el archivo

Al ejecutar con Docker, montá el **directorio** que contiene el archivo de configuración, no el archivo en sí:

```yaml
volumes:
  - ./configs:/etc/micro-health-checker:ro
```

Montar un único archivo (`./configs/config.yml:/etc/micro-health-checker/config.yml:ro`) rompe la recarga automática: los editores y herramientas que guardan de forma atómica (escriben un archivo temporal y luego lo renombran sobre el original — el comportamiento predeterminado de vim, VS Code, `sed -i`, etc.) reemplazan el inodo del archivo en el host, pero la capa de compartición de archivos de Docker (osxfs/gRPC-FUSE en Docker Desktop) no propaga una notificación de cambio para ese montaje de un solo archivo dentro del contenedor. Montar el directorio padre mantiene la notificación funcionando porque lo que cambia es la entrada del propio directorio.
