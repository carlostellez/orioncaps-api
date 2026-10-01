# orioncaps-api

API de correos de Orion Caps (Go). Recibe el formulario de contacto del sitio, avisa al administrador por Mailgun y envía una confirmación al cliente.

## Endpoints

| Método | Ruta | Descripción |
|---|---|---|
| POST | `/api/v1/contact` | Envía el formulario (requiere `x-api-key` en AWS) |
| GET | `/health` | Estado del servicio |

Campos: `full_name`, `email`, `phone`, `message` (obligatorios); `company`, `product_type`, `quantity` (opcionales); `website` (honeypot, debe ir vacío).

## Desarrollo local

```bash
cp .env.example .env     # completa MAILGUN_API_KEY, etc.
make run                 # http://localhost:8080
make test
```

Requiere Go 1.26+.

## Documentación

- [Plan de desarrollo](document/PLAN_DESARROLLO.md)
- [Despliegue en AWS](document/DESPLIEGUE.md)
