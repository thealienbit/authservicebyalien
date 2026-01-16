# Auth Service (Go + PostgreSQL)

A standalone **Authentication & Identity Provider (IdP)** built in Go, designed to be reused across multiple applications.

This service provides:
- User registration & email verification
- Secure login with JWT (RS256)
- Refresh token rotation
- Logout (single & all sessions)
- Forgot / reset password
- Rate limiting
- SMTP / AWS SES email delivery
- JWKS for public key discovery

It is **stateless for access tokens**, **stateful for refresh tokens**, and **secure by default**.

---

## Architecture Overview
┌──────────────┐
│ Client Apps │
│ (Web / API) │
└──────┬───────┘
│ JWT (Bearer)
▼
┌────────────────────┐
│ Auth Service │
│ (Go + Chi) │
│ │
│ - JWT Issuer │
│ - JWKS endpoint │
│ - Rate limiting │
│ - Email sender │
└──────┬─────────────┘
│
▼
┌────────────────────┐
│ PostgreSQL (Auth) │
│ - users │
│ - emails │
│ - refresh tokens │
│ - reset tokens │
└────────────────────┘


---

## Core Design Principles

- **Asymmetric JWT signing (RS256)**
- **No shared secrets** with application services
- **Hashed tokens only** stored in DB
- **No user / email enumeration**
- **Single-use refresh tokens**
- **Key rotation supported**
- **Provider-agnostic email layer**

---

## Base URL
/auth/v1



---

## Authentication Model

### Access Tokens
- JWT (RS256)
- Short-lived (15 min)
- Stateless
- Verified via JWKS

### Refresh Tokens
- Opaque
- Stored hashed in DB
- Rotated on every use
- Revoked on logout / password reset

---

## Public Endpoints (No Auth Required)

### Register
POST /auth/v1/register
```json
{
  "email": "user@example.com",
  "password": "StrongPassword123!"
}
```
Response
```json
{
  "message": "Registration successful. Please verify your email."
}
```

### Verify Email
POST /auth/v1/email/verify
```json
{
  "token": "verification-token"
}
```

### Resend Verification Email
POST /auth/v1/email/resend
```json
{
  "email": "user@example.com"
}
```

### Login
POST /auth/v1/login
```json
{
  "email": "user@example.com",
  "password": "StrongPassword123!"
}
```
Response
```json
{
  "access_token": "jwt",
  "refresh_token": "opaque",
  "expires_in": 900
}
```

### Refresh Token 
POST /auth/v1/token/refresh
```json
{
  "refresh_token": "opaque"
}
```

### Forgot Password
```json
POST /auth/v1/password/forgot
{
  "email": "user@example.com"
}
```

### Reset Password
POST /auth/v1/password/reset
```json
{
  "token": "reset-token",
  "new_password": "NewStrongPass123!"
}
```
### JWKS (Public Key Discovery)
GET /auth/v1/.well-known/jwks.json
Used by other services to verify JWT signatures.


## Protected Endpoints(JWT Required)

### Logout (All Sessions)
POST /auth/v1/logout/all
Authorization: Bearer <access_token>

### Logout (Single Session)
Uses refresh tojen (JWT not required)
POST /auth/v1/logout
```json
{
  "refresh_token": "opaque"
}
```

## JWT Claims
```json
{
  "sub": "user_id",
  "iss": "auth-service",
  "aud": "your-app",
  "email": "user@example.com",
  "email_verified": true,
  "iat": 1234567890,
  "exp": 1234569999
}
```

## How Other Applications Integrate
1. Login via Auth Service
Call /login
Store access + refresh token
2. Call Protected APIs
Authorization: Bearer <access_token>
3. Validate JWT
Fetch JWKS from:
/auth/v1/.well-known/jwks.json
Cache keys
Verify locally (no DB calls)
4. Refresh Access Token
Call /token/refresh when access token expires
5. Logout
Call /logout or /logout/all



## Rate Limiting
| Endpoint        | Limit         |
| --------------- | ------------- |
| Login           | 5 / min / IP  |
| Register        | 3 / min / IP  |
| Email resend    | 3 / min / IP  |
| Forgot password | 3 / min / IP  |
| Global          | 60 / min / IP |


## Email Delivery
Local / Dev
- Gmail SMTP (App Password)
- From must match Gmail account

Production
- AWS SES (recommended)
- Domain-verified sender
- IAM-based auth
- Email failures never block auth flows.


## Environment Variables
\# Server
HTTP_PORT=8080
APP_ENV=development

\# Database
DATABASE_URL=postgres://auth_user:password@localhost:5432/auth_db?sslmode=disable

\# SMTP (local)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your@gmail.com
SMTP_PASS=app-password
SMTP_FROM=your@gmail.com

# AWS SES (prod)
AWS_REGION=us-east-1


## Running Locally 
### Start DB 
docker compose up -d
### Run Auth Service
go run ./cmd/server
### Health Check
curl http://localhost:8080/health


## Security Notes
Passwords: Argon2id
Tokens: SHA-256 hashed
JWT: RS256
No plaintext secrets in DB
No user enumeration
Email delivery is non-blocking


## Recommended Production Hardening (Future goals)
Persist signing keys (KMS / Vault)
Redis-backed rate limiting
Audit logs
Account lockout on repeated failures
HTML email templates
Background email queue
