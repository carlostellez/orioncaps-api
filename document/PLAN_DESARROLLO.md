# Plan de desarrollo — orioncaps-api (proveedor de correos)

Referencia analizada: https://github.com/carlostellez/zititex-api (FastAPI + Mailgun + Serverless/AWS Lambda).

## 1. Objetivo

`orioncaps-api` recibe los formularios de contacto/cotización del sitio de Orion Caps y envía correos transaccionales:

1. Notificación al administrador (con `Reply-To` al cliente).
2. Confirmación automática al cliente.

Alcance inicial: **un solo endpoint** (`POST /api/v1/contact`) + `/health`. Sin base de datos en la v1.

## 2. Validación del proyecto de muestra

### Qué conviene replicar
| Elemento | Detalle |
|---|---|
| Stack | FastAPI + Pydantic v2 + Mangum, Python 3.12 |
| Despliegue | Serverless Framework → AWS Lambda + API Gateway (us-east-2) |
| Seguridad básica | API Key de API Gateway (`private: true`) + CORS |
| Estructura | `app/{api,core,schemas,services}`, `tests/`, `document/` |
| Config | `pydantic-settings` leyendo variables de entorno |
| CI/CD | GitHub Actions → `serverless deploy` en push a `master` |

### Problemas detectados (NO copiar tal cual)
| # | Problema | Dónde | Corrección en Orion |
|---|---|---|---|
| 1 | **Inyección HTML en correos**: `full_name`, `company`, `message`, etc. se interpolan sin escapar | `services/mailgun.py` | Plantillas Jinja2 con autoescape, o `html.escape` |
| 2 | Responde `success: true` aunque el correo al admin falle (comentario dice "data was saved", pero la BD está comentada) | `api/v1/contact.py` | Devolver 502/503 si falla el envío al admin |
| 3 | `requests.post` sin `timeout` y síncrono dentro de endpoint `async` | `mailgun.py` | `httpx.AsyncClient` con timeout (~10 s) y 1–2 reintentos |
| 4 | Marca "Zititex" hardcodeada en remitente, asuntos y cuerpos | `mailgun.py` | Parametrizar `BRAND_NAME`, `FROM_EMAIL` en settings |
| 5 | `quantity: int` en la firma pero el schema usa `str` | `mailgun.py` | Tipar correctamente |
| 6 | Se imprimen datos personales (`print(contact_data)`) y se devuelve `str(e)` al cliente | `contact.py` | Logging estructurado sin PII; errores genéricos |
| 7 | CORS `*` con `allow_credentials=True`; la API Key viaja en el frontend | `main.py`, `serverless.yml` | Lista blanca de orígenes de Orion; ver sección 5 |
| 8 | Código muerto de BD (SQLAlchemy, aiomysql, alembic, models, repositories) y `Depends(get_async_db)` aún inyectado | varios | **Omitir por completo** en la v1 |
| 9 | Dependencias de más (boto3, passlib, jose, faker…) y versiones viejas (fastapi 0.104) | `requirements.txt` | Separar `requirements.txt` / `requirements-dev.txt`, versiones actuales |
| 10 | Pipeline: `continue-on-error: true` + output nunca seteado ⇒ el paso de éxito jamás corre; tests desactivados; credenciales AWS estáticas | `deploy.yml` | Tests obligatorios, OIDC con rol IAM, sin `continue-on-error` |
| 11 | Secretos de Mailgun como variables de entorno planas del Lambda | `serverless.yml` | SSM Parameter Store / Secrets Manager |
| 12 | Sin anti-spam (rate limit, honeypot, captcha) | — | Throttling en API Gateway + honeypot; captcha opcional |
| 13 | 17 documentos `.md` redundantes en `document/` | `document/` | Mantener solo README, DESPLIEGUE y API_CORREO |

## 3. Estructura objetivo

> **Cambio de enfoque: la implementación es en Go** (el ejemplo Zititex es Python; las conclusiones de la sección 2 aplican igual). Stack: Go 1.26, `net/http` (stdlib), `html/template` (autoescape), Lambda con `aws-lambda-go` + `httpadapter` (runtime `provided.al2023`, arm64). Los tests usan solo `httptest`.

```
orioncaps-api/
├── cmd/
│   ├── server/main.go          # servidor local (+ carga de .env)
│   └── lambda/main.go          # entrada AWS Lambda
├── internal/
│   ├── config/                 # variables de entorno
│   ├── email/                  # Provider (interfaz) + Mailgun (reintentos, timeout)
│   ├── contact/                # Form+validación, Mailer, Handler, templates/*.html
│   └── server/                 # rutas, CORS, recover
├── .github/workflows/          # (Fase 4)
├── serverless.yml              # (Fase 4)
├── go.mod / Makefile / .env.example
└── document/
```

## 4. Fases

### Fase 0 — Decisiones y prerrequisitos (bloqueante)
- [ ] Definir dominio de envío (p. ej. `mg.orioncaps.com`) y crear el dominio en Mailgun (región US/EU).
- [ ] Configurar DNS: **SPF, DKIM, MX/CNAME de tracking** y validar el dominio (sin esto los correos caen en spam).
- [ ] Definir `ADMIN_EMAIL` (uno o varios destinatarios) y `FROM_EMAIL` (ej. `Orion Caps <no-reply@mg.orioncaps.com>`).
- [ ] Confirmar origen(es) del frontend para CORS (sitio de producción y staging).
- [ ] Cuenta AWS, región y rol IAM para despliegue (OIDC con GitHub).
- [ ] Campos exactos del formulario de Orion Caps (¿se mantienen `company`, `product_type`, `quantity`?).

### Fase 1 — Esqueleto del proyecto
- [ ] `.gitignore`, `.env.example`, `requirements*.txt`, `pytest.ini`, `Makefile`, `pre-commit` (ruff/black).
- [ ] `core/config.py` con `BRAND_NAME`, `MAILGUN_*`, `ADMIN_EMAIL`, `FROM_EMAIL`, `ALLOWED_ORIGINS`.
- [ ] `main.py`: CORS restringido, `/health`, handler global de errores (sin filtrar detalles), `Mangum`.

### Fase 2 — Envío de correo
- [ ] Schema `ContactForm` (nombre, email, teléfono, mensaje; campos opcionales según Fase 0) + campo honeypot.
- [ ] `EmailProvider` (Protocol) y `MailgunProvider` con `httpx`, timeout y reintento.
- [ ] Plantillas Jinja2 con autoescape (admin y cliente), texto plano alterno.
- [ ] `ContactMailer`: admin primero (falla ⇒ error al cliente); confirmación al cliente es *best-effort*.
- [ ] Endpoint `POST /api/v1/contact` con códigos: 200 OK, 422 validación, 502 fallo del proveedor, 500 config faltante.

### Fase 3 — Pruebas
- [ ] Unitarias: schema (email/teléfono/longitudes), escapado HTML, armado de payload Mailgun.
- [ ] Endpoint: éxito, fallo admin, fallo confirmación, honeypot, config ausente (Mailgun mockeado con `respx`).
- [ ] Cobertura mínima 90 % exigida en CI.
- [ ] Prueba manual real contra el dominio sandbox/verificado de Mailgun.

### Fase 4 — Infraestructura y CI/CD
- [ ] `serverless.yml`: Lambda python3.12, API Gateway con API Key + Usage Plan (throttle/burst), `logRetentionInDays: 14`, secretos desde SSM.
- [ ] Workflow `ci.yml` (PR): lint + tests. `deploy.yml` (push a `master`): tests → deploy con OIDC; stages `dev` y `prod`.
- [ ] Secrets de GitHub: `MAILGUN_API_KEY`, `MAILGUN_DOMAIN`, `ADMIN_EMAIL`, `AWS_ROLE_ARN`.

### Fase 5 — Integración y salida a producción
- [ ] Conectar el formulario del sitio de Orion Caps al endpoint.
- [ ] Probar en `dev`: entrega a bandeja de entrada (no spam), `Reply-To`, caracteres especiales/acentos.
- [ ] Alarma CloudWatch en errores del Lambda; revisar logs de Mailgun.
- [ ] Documentar contrato en `document/API_CORREO.md` y runbook de despliegue/rollback.

## 5. Decisión de seguridad pendiente: la API Key

En Zititex la API Key de API Gateway la usa el frontend, por lo que **es visible en el navegador**; solo sirve como limitador, no como secreto. Opciones:

- **A (recomendada para v1):** mantener API Key como control de cuota + CORS por origen + throttling + honeypot.
- **B:** añadir captcha (Turnstile/reCAPTCHA) validado en el backend.
- **C:** que el backend del sitio llame a esta API server-to-server (la key deja de ser pública).

## 6. Criterios de aceptación

- Un envío válido genera 2 correos (admin con `Reply-To` correcto, y confirmación al cliente) en bandeja de entrada.
- Entradas con HTML/scripts se muestran escapadas.
- Si Mailgun falla, la API responde error (no éxito falso) y no filtra detalles internos.
- CI en verde (lint + tests ≥ 90 %) antes de cada despliegue.
- Ningún secreto en el repositorio ni en logs; sin PII en `print`/logs.

## 7. Estimación orientativa

| Fase | Esfuerzo |
|---|---|
| 0 (DNS/Mailgun/decisiones) | 0.5–1 día (+ propagación DNS) |
| 1–2 | 1–1.5 días |
| 3 | 0.5–1 día |
| 4 | 1 día |
| 5 | 0.5 día |
| **Total** | **~4–5 días** |

## 8. Estado y pendientes de infraestructura

- Fases 1–4 implementadas en la rama `feat/email-provider-api` (código, tests, `serverless.yml`, CI/CD).
- **Pendientes (configuración de infraestructura, se definen más adelante):**
  - Dominio de Mailgun + DNS (SPF/DKIM) y remitente definitivo.
  - Destinatarios (`admin-email`) y orígenes del sitio (`allowed-origins`).
  - Cuenta/región AWS, rol OIDC y environments de GitHub (`dev`, `prod`).
  - Decisión sobre la API Key pública (ver sección 5) y campos finales del formulario.
- Fase 5 (integración con el sitio y salida a producción) depende de lo anterior.
