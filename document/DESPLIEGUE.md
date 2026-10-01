# Despliegue (AWS Lambda + API Gateway)

Stages: `dev` (automático al hacer push a `master`) y `prod` (manual, con aprobación opcional vía environment de GitHub).

## 1. Prerrequisitos de AWS (una sola vez por cuenta)

### 1.1 Parámetros en SSM Parameter Store
`serverless.yml` lee estos valores al desplegar. Reemplaza `<stage>` por `dev` o `prod`:

```bash
aws ssm put-parameter --region us-east-2 --type SecureString --name /orioncaps-api/<stage>/mailgun-api-key --value "key-xxxx"
aws ssm put-parameter --region us-east-2 --type String --name /orioncaps-api/<stage>/mailgun-domain  --value "mg.orioncaps.com"
aws ssm put-parameter --region us-east-2 --type String --name /orioncaps-api/<stage>/from-email      --value "Orion Caps <no-reply@mg.orioncaps.com>"
aws ssm put-parameter --region us-east-2 --type String --name /orioncaps-api/<stage>/admin-email     --value "ventas@orioncaps.com"
aws ssm put-parameter --region us-east-2 --type String --name /orioncaps-api/<stage>/allowed-origins --value "https://www.orioncaps.com"
```

> Los valores de ejemplo son provisionales; ver la sección "Pendientes de infraestructura" del plan.
> `admin-email` y `allowed-origins` aceptan varios valores separados por coma.

### 1.2 Rol IAM para GitHub Actions (OIDC, sin llaves estáticas)
1. En IAM crear el *Identity provider* OIDC `token.actions.githubusercontent.com` (audience `sts.amazonaws.com`).
2. Crear un rol con esta *trust policy* (ajusta cuenta, org y repo):

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": { "Federated": "arn:aws:iam::<ACCOUNT_ID>:oidc-provider/token.actions.githubusercontent.com" },
    "Action": "sts:AssumeRoleWithWebIdentity",
    "Condition": {
      "StringEquals": { "token.actions.githubusercontent.com:aud": "sts.amazonaws.com" },
      "StringLike":   { "token.actions.githubusercontent.com:sub": "repo:<ORG>/orioncaps-api:*" }
    }
  }]
}
```
3. Permisos del rol: CloudFormation, Lambda, API Gateway, IAM (roles del servicio), S3 (bucket de despliegue de Serverless), CloudWatch Logs y Alarms, SNS, y `ssm:GetParameter` sobre `/orioncaps-api/*`.
   Para empezar se puede usar una política amplia en `dev` y acotarla antes de `prod`.

### 1.3 GitHub
- *Settings → Environments*: crear `dev` y `prod` (en `prod` añadir *Required reviewers*).
- Secret `AWS_ROLE_ARN` (por environment, o a nivel repo) con el ARN del rol.
- Variable opcional `AWS_REGION` (por defecto `us-east-2`).

## 2. Flujo de CI/CD

| Workflow | Cuándo | Qué hace |
|---|---|---|
| `ci.yml` | PR y push a ramas ≠ `master` | `gofmt`, `go vet`, tests con `-race`, build del Lambda |
| `deploy.yml` | push a `master` → `dev`; manual → `dev`/`prod` | tests → build → `serverless deploy` → smoke test `/health` |

## 3. Despliegue manual (opcional)

Requiere Go, Node 20, `zip` y credenciales AWS en el entorno.

```bash
make build-lambda
npm ci
npx serverless deploy --stage dev
```

## 4. Después del primer despliegue

1. **API Key**: `npx serverless info --stage dev --verbose` muestra la key; el sitio la envía en el header `x-api-key`.
2. **Alertas**: suscribir un correo al topic `orioncaps-api-<stage>-alerts` (output `AlertsTopicArn`):
   ```bash
   aws sns subscribe --topic-arn <AlertsTopicArn> --protocol email --notification-endpoint tu@correo.com
   ```
3. **Prueba**:
   ```bash
   curl -X POST "$ENDPOINT/api/v1/contact" \
     -H "Content-Type: application/json" -H "x-api-key: $API_KEY" \
     -d '{"full_name":"Prueba","email":"tu@correo.com","phone":"+52 123 456 7890","message":"Mensaje de prueba del formulario."}'
   ```

## 5. Rollback

- Revertir el commit en `master` (redeploy automático a `dev`) o ejecutar manualmente el workflow sobre el commit anterior.
- Alternativa rápida: `npx serverless rollback --stage <stage>` (lista timestamps) y `npx serverless rollback --timestamp <ts> --stage <stage>`.

## 6. Notas de seguridad

- **La API Key es visible en el navegador**: sirve para cuotas/throttling, no como secreto (plan: 5 req/s, ráfaga 10, 5000/día; ajustable en `serverless.yml`).
- El preflight `OPTIONS` no exige key (los navegadores no la envían); la restricción por origen la aplica la app con `allowed-origins`.
- Los secretos se resuelven al desplegar y quedan como variables de entorno cifradas en reposo de la función Lambda; no se guardan en el repositorio.
