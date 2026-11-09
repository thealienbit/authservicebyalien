# Auth Service (Go + PostgreSQL)

A multi-tenant authentication and identity provider built in Go. It is shared
across multiple business apps, with organizations and org-scoped roles under
each app.

## Features

- Per-app users, organizations, and memberships
- App authentication via `X-App-Key` (backend-to-auth only)
- JWT access tokens (RS256, per-app `aud`) with `org` and `role` claims
- Refresh token rotation with reuse detection
- Email verification, password reset, logout (single and all sessions)
- Public JWKS for local token verification
- Signing keys persisted in PostgreSQL and encrypted at rest
- Rate limiting keyed by app + client IP
- Server-rendered admin portal with cookie sessions and CSRF protection
- SMTP email delivery

## Architecture

```
Browser ──JWT──▶ Business backend ──X-App-Key──▶ Auth service ──▶ PostgreSQL
                        │
                        └──verifies JWT locally via JWKS
```

The browser never calls the auth service directly. The business backend holds
the appkey and proxies auth calls. Access tokens are minted per app and handed
to the browser; the backend validates them against the public JWKS.

### Tenancy model

```
apps → organizations → memberships (users)
```

- Users are per-app. The same human in two apps is two accounts.
- Emails are unique per `(app_id, email)`.
- The appkey is a static secret generated on app creation. It is stored as a
  SHA-256 hash, shown once, and sent by backends as `X-App-Key`.
- Platform admins are `users` with `app_id NULL` and `is_platform_admin = true`.

### Roles

| Role | Scope | Notes |
|---|---|---|
| Platform admin | All apps | Creates apps, orgs, first SUPERADMIN |
| SUPERADMIN | One organization | Full control of the org |
| ADMIN | One organization | Manages CLERKs |
| CLERK | One organization | Regular worker |

## Calling the API

Every `/auth/v1/*` endpoint requires the app key:

```
X-App-Key: <appkey>
```

Public exceptions: `GET /health` and
`GET /auth/v1/.well-known/jwks.json`.

### Register (backend only)

```http
POST /auth/v1/register
X-App-Key: <appkey>
Content-Type: application/json

{"email": "user@example.com", "password": "StrongPassword123!"}
```

Returns `202`. The user receives a verification link built from the app's
`base_url`. Registering does not grant access: a membership is required before
login succeeds.

### Verify email

```http
POST /auth/v1/email/verify
{"token": "<verification-token>"}
```

### Resend verification

```http
POST /auth/v1/email/resend
{"email": "user@example.com"}
```

### Login

```http
POST /auth/v1/login
{"email": "user@example.com", "password": "StrongPassword123!"}
```

```json
{
  "access_token": "<jwt>",
  "refresh_token": "<opaque>",
  "expires_in": 900
}
```

Login fails with `403` when the user has no organization membership.

### Refresh

```http
POST /auth/v1/token/refresh
{"refresh_token": "<opaque>"}
```

Refresh tokens are single-use and rotated on every call. Reusing a revoked
token revokes the user's sessions.

### Forgot / reset password

```http
POST /auth/v1/password/forgot
{"email": "user@example.com"}
```

```http
POST /auth/v1/password/reset
{"token": "<reset-token>", "new_password": "NewStrongPass123!"}
```

The reset link is built from the app's `base_url`.

### Logout

```http
POST /auth/v1/logout
{"refresh_token": "<opaque>"}
```

### Logout all sessions

```http
POST /auth/v1/logout/all
Authorization: Bearer <access_token>
```

### JWKS

```http
GET /auth/v1/.well-known/jwks.json
```

Public. Every non-retired signing key is published; `kid` is the RFC 7638
thumbprint, so rotation is visible to caches immediately.

## Admin portal

The portal is served at `/portal` and uses server-side sessions (12h), not
bearer tokens. It does not use `X-App-Key`. Only platform admins can sign in.

| Route | Purpose |
|---|---|
| `GET/POST /portal/login` | platform-admin login |
| `POST /portal/logout` | revoke the session |
| `GET /portal` | list apps, create app (appkey shown once) |
| `GET /portal/apps/{id}/orgs` | list/create organizations |
| `GET /portal/orgs/{id}/users` | list/create SUPERADMIN |

Bootstrap the first platform admin:

```bash
docker compose run --rm --entrypoint /app/seed auth-service \
  -email admin@example.com -password 'change-me-please'
```

The command is idempotent for the same email. Platform admins are not scoped to
an app (`app_id IS NULL`) and are created verified.

## JWT claims

```json
{
  "sub": "user-uuid",
  "iss": "auth-service",
  "aud": "app-uuid",
  "email": "user@example.com",
  "email_verified": true,
  "org": "organization-uuid",
  "role": "SUPERADMIN",
  "iat": 1234567890,
  "exp": 1234568790
}
```

`aud` is the app UUID. The appkey is never placed in a token or claim.

Business apps should verify `iss`, `exp`, the RS256 signature, and
`aud == <their app UUID>`.

## Rate limiting

| Scope | Limit |
|---|---|
| Public routes (JWKS, health) | 60 burst / 1 req/s per IP |
| `/auth/v1/*` global | 60 burst / 1 req/s per (app, client IP) |
| Login | 5 / min per (app, client IP) |
| Register | 3 / min per (app, client IP) |
| Email verify/resend | 3 / min per (app, client IP) |

`X-Forwarded-For` is honoured only when the immediate peer is listed in
`TRUSTED_PROXIES`; the header is walked right to left, skipping trusted hops.

## Environment variables

```bash
# Server
APP_ENV=development
HTTP_PORT=8080

# Database
DATABASE_URL=postgres://auth_user:auth_password@localhost:5430/auth_db?sslmode=disable

# Required: base64-encoded 32-byte key used to encrypt signing keys at rest
JWT_KEK=<base64 32 bytes>

# Optional: comma-separated IPs/CIDRs allowed to set X-Forwarded-For
TRUSTED_PROXIES=10.0.0.0/8

# SMTP
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your@gmail.com
SMTP_PASS=app-password
SMTP_FROM=your@gmail.com
```

Generate a KEK with: `openssl rand -base64 32`.

## Running locally

```bash
docker compose up --build -d
```

Migrations run automatically at service startup and are recorded in
`schema_migrations`. There is no separate `init.sql`; the migration files are
the single source of truth. An existing database created before migrations were
introduced must be recreated (`docker compose down -v`) or baselined manually.

Health check:

```bash
curl http://localhost:8082/health
```

## Migrations

SQL files live in `migrations/` and are embedded into the binary. The runner
applies unapplied files in filename order, one transaction each, under a
Postgres advisory lock.

```bash
docker compose exec authbyaliendb psql -U auth_user -d auth_db -c '\dt'
```

## Deployment

Production uses a separate compose file and host-level nginx:

```bash
docker compose -f docker-compose.prod.yml up -d
```

See [deploy/README.md](deploy/README.md) for image builds, nginx/TLS, the
platform-admin seed, backups and firewall rules.

## Tests

The runtime image has no Go toolchain. Run tests with a throwaway container:

```bash
docker run --rm -v "$PWD":/src -w /src \
  -v authservice-go-mod:/go/pkg/mod \
  golang:1.25-alpine sh -c "go build ./... && go test ./..."
```

## Security notes

- Passwords: Argon2id
- Refresh and reset tokens: opaque, SHA-256 hashed at rest
- App keys: 32 random bytes, SHA-256 hashed at rest, shown once
- Signing keys: persisted in the database, AES-GCM encrypted with `JWT_KEK`
- JWT: RS256, short-lived (15 min), per-app audience
- No user or email enumeration on registration/reset endpoints

## Roadmap

- Invites and role management
- Audit logging
- Redis-backed rate limiting
- KMS/Vault for signing keys
