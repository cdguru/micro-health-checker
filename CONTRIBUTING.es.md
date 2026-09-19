# Contribuir

<!-- translation-source: CONTRIBUTING.md -->
[English](CONTRIBUTING.md) | **Español**

> Esta es la traducción española. La documentación inglesa es la fuente canónica ante cualquier diferencia.

Gracias por ayudar a que la salud semántica de servicios sea más fácil de operar.

## Antes de escribir código

- Buscá en los issues existentes y en el catálogo de integraciones.
- Abrí una propuesta antes de agregar un driver o modificar la configuración o API pública.
- Mantené los checks predeterminados en modo de solo lectura y con el mínimo impacto posible.
- Nunca incluyas credenciales, direcciones de producción ni respuestas privadas capturadas dentro de las pruebas.

## Desarrollo

Requisitos: Go 1.26+, Git y Docker de manera opcional.

```bash
git clone https://github.com/christiandente/micro-health-checker.git
cd micro-health-checker
go mod download
make test
make build
```

Antes de enviar cambios:

```bash
make fmt
make vet
make test-race
make docs-check
make build
```

## Requisitos para drivers

Un nuevo driver de producto debería incluir:

1. Validación YAML estricta y valores predeterminados documentados.
2. Cancelación mediante contexto y timeouts limitados.
3. Soporte TLS y autenticación apropiados para el protocolo.
4. Ninguna mutación por defecto.
5. Pruebas unitarias y, cuando sea práctico, una prueba de integración contenerizada.
6. Métricas y comportamiento de UI mediante el scheduler común.
7. Documentación de configuración y del catálogo de integraciones en inglés y español.

Preferí un motor reutilizable o preset declarativo cuando un producto exponga un contrato de salud HTTP, SQL o gRPC convencional.

## Pull requests

- Mantené los cambios enfocados.
- Explicá el impacto para el usuario y los compromisos operativos.
- Agregá una entrada al changelog bajo `Unreleased` para cambios visibles por el usuario.
- Utilizá asuntos con estilo conventional commits cuando sea práctico, por ejemplo `feat(redis): add PING check`.
- Actualizá la documentación inglesa canónica y su traducción española en el mismo pull request.
- Mantené la planificación privada de arquitectura y roadmap fuera del árbol público del repositorio.
- Confirmá que tenés derecho a enviar la contribución bajo Apache-2.0.

Al enviar una contribución, aceptás que se distribuya bajo la licencia Apache License 2.0 del repositorio.
