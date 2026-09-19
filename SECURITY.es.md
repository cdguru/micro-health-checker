# Política de seguridad

<!-- translation-source: SECURITY.md -->
[English](SECURITY.md) | **Español**

> Esta es la traducción española. La documentación inglesa es la fuente canónica ante cualquier diferencia.

## Versiones soportadas

Hasta `v1.0.0`, las correcciones de seguridad se aplican solamente a la última versión menor publicada.

## Informar una vulnerabilidad

No abras un issue público de GitHub para vulnerabilidades sospechadas.

Utilizá el flujo privado **Report a vulnerability** de los security advisories de GitHub en este repositorio. Incluí:

- Versión o commit afectado
- Pasos de reproducción o prueba de concepto
- Impacto esperado
- Mitigación sugerida, si se conoce

Deberías recibir un acuse de recibo dentro de los 5 días hábiles. Luego del triage se acordará una fecha de divulgación coordinada.

## Recomendaciones de despliegue

- Ubicá la UI y la API en una red confiable o detrás de un reverse proxy autenticado.
- Deshabilitá `POST /-/reload` salvo que sea necesario operativamente.
- Inyectá secretos mediante variables de entorno o un administrador de secretos.
- Utilizá credenciales dedicadas y de solo lectura para los servicios objetivo.
- Montá la configuración como solo lectura y `/data` como lectura-escritura.
- No publiques detalles de errores de servicios objetivo en Internet.

Los secretos incluidos en el historial de Git deben considerarse comprometidos incluso después de eliminarlos.
