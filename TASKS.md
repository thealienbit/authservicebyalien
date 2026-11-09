# TASKS.md — Multi-tenant Auth Upgrade

Goal: evolve the auth service into a multi-tenant IdP that can be shared across
multiple apps (skfirst: `sks` backend), with organizations under each app and
org-scoped user roles.

## Agreed design decisions

| Decision | Choice |
|---|---|
| Tenancy model | `apps` → `organizations` → `memberships` (users) |
| Org creation | Invite-based only; org/user creation happens in the admin portal by platform admin / org admins |
| Ownership split (Option B) | Auth owns tenancy + membership (bare org: id/name). Business apps own their own org *profile* (gstin, address...) keyed by the same UUID |
| Users | Per-app. Same human in two apps = two accounts. Emails unique per (app, email) |
| Appkey | Static API key generated on app creation, given to backend apps as env var, sent as `X-App-Key` header on app→auth calls. Stored hashed (SHA-256), shown once |
| JWT claims | `sub` (user UUID), `aud` (app UUID — never the appkey), `org` (org UUID), `role` (org-scoped: SUPERADMIN / ADMIN / CLERK) |
| Appkey exposure | Server-side only. Never in `aud`, claims, logs, or anything handed to a client |
| Admin portal | Server-rendered Go templates behind platform-admin login. v1: create app → create org → create user (SUPERADMIN). Invites/roles management later |
| JWKS | Stays public (optionally rate-limited); appkey is NOT required for JWKS |
| Admin REST API | Only if/when portal needs it; portal HTML first |
| Platform admin | `users.is_platform_admin` flag; one seeded manually via SQL to bootstrap |

## Roles

- **Platform admin** — owns apps. Creates apps / orgs / first SUPERADMIN per org.
- **SUPERADMIN** (org) — full control of one organization.
- **ADMIN** (org) — can invite/manage CLERKs within their org.
- **CLERK** (org) — regular worker.

Claims include the org role, so business apps can gate endpoints without
calling auth.

## Slices

### Slice 1 —.schema + migrations strategy
- [x] Migration approach: tiny embedded runner (`migrations/migrate.go`) — `embed.FS`, `schema_migrations`, per-file transaction, `pg_advisory_lock`. No new dependency. Runs at boot from `main.go`.
- [x] Migrations are the only source of truth: `init.sql` moved to `migrations/0001_init.sql` and deleted; the `docker-entrypoint-initdb.d` mount is removed from `docker-compose.yml`. Existing dev volumes need `docker compose down -v` (local data only).
- [x] `migrations/0002_multitenant.sql`:
  - `apps` (id, name, key_hash UNIQUE, status, timestamps)
  - `users.app_id` (nullable FK → apps), `users.is_platform_admin` (bool default false), index on app_id
  - `organizations` (id, app_id NOT NULL FK, name, status, timestamps, index app_id)
  - `memberships` (user_id, organization_id, role IN SUPERADMIN/ADMIN/CLERK, PK(user_id, organization_id)), FK + index on organization_id
  - `user_emails.app_id` FK; drop global email unique; add unique (app_id, email); partial unique on email where app_id IS NULL
  - Note: `user_emails.app_id` is denormalized from `users.app_id` — write path must keep both in sync
- [x] `migrations/0003_signing_keys.sql` — persist RSA keys (today `main.go:22` regenerates on every boot, which invalidates all tokens and breaks JWKS caches)
  - `signing_keys` (id, kid UNIQUE, alg default 'RS256', private_key_encrypted TEXT, is_active bool, created_at, retired_at)
  - Partial unique index enforces at most one active signing key
  - `kid` = RFC 7638 JWK thumbprint, not a constant — a key change must be a JWKS cache miss, not a silent `kid` collision
  - Store the private key encrypted at rest (AES-GCM with a KEK from env) so a DB dump alone isn't enough to mint tokens
  - Bootstrap: if no active key exists, generate and insert under a Postgres advisory lock so replicas cannot race
  - Sign with the active key; serve every non-retired key in JWKS; retire an old key only after the max access-token TTL (15 min) plus the JWKS refresh interval
- [ ] Backfill: existing rows have `app_id NULL`. Left as-is for now; no production data exists. Revisit before any real deploy.

### Slice 2 — Go models + repo layer
- [x] `internal/store` package: `Store` wrapping `*pgxpool.Pool`; models + repositories for apps, organizations, memberships
- [x] Appkey generation: 32-byte random, base64url plaintext shown once, SHA-256 base64url hash persisted; lookup by hash
- [x] Signing-key repository: `LoadSigningKeys` / `EnsureSigningKey` (advisory-locked bootstrap), decrypts with `JWT_KEK`
- [x] Crypto moved to `internal/signing` (`Key`, `Generate`, `Thumbprint`, `Cipher`, PEM helpers); `internal/auth/keys.go` removed
- [x] `main.go` loads the active key + all non-retired keys from the DB instead of generating a throwaway key each boot
- [x] `JWT_KEK` added to config (required, base64 32 bytes) and to local `docker-compose.yml`

### Slice 3 — `X-App-Key` middleware
- [x] `AppKeyMiddleware` mounted on the `/auth/v1` group only. `/health` + JWKS stay public; portal routes will use cookie auth
- [x] All `/auth/v1/*` is backend-only, including `/register` — no public self-signup
- [x] Resolves app from `HashAppKey(header)`; attaches `*store.App` to the request context (`auth.WithApp`)
- [x] 401 missing/invalid key, 403 `DISABLED` app
- [x] Rate limiting keyed on `(app_id, client IP)` in the `/auth/v1` group; public routes stay per-IP
- [x] `TRUSTED_PROXIES` (comma-separated IP/CIDR) gates `X-Forwarded-For`; `middleware.RealIP` removed
- [x] Keyed limiter rewritten with `lastSeen` eviction (the old cleanup deleted live limiters and reset buckets)
- [ ] Audit rows still unwritten; `ClientIP` is now safe to use when they land
- [ ] Query scoping (register/login/forgot/resend/refresh) lands in Slice 4. Until then those handlers receive the app in context but do not filter by it

### Slice 4 — Login claims
- [x] Login resolves user → membership (single org for now) and mints: `sub`, `email`, `email_verified`, `org`, `role`, `aud` = app UUID, `iat`, `exp`
- [x] No membership → **403 `No organization membership`** (login is blocked; no half-authenticated tokens)
- [x] `aud` = app UUID; the appkey never enters a claim
- [x] `JWTIssuer.IssueAccessToken` now takes `AccessTokenClaims`; `JWTIssuer` no longer holds an audience
- [x] Auth's own verifier skips audience (`""`) because per-app audiences are not known up front; only `/logout/all` uses it and only needs `sub`
- [x] JWT middleware puts `org` / `role` in the request context
- [x] App scoping added to: register (writes `users.app_id` + `user_emails.app_id`), login, forgot password, resend verification, refresh
- [x] Refresh re-derives app + membership and rejects a refresh token from a different app
- [x] Refresh now commits its transaction (previously it always rolled back, so rotation never persisted) and the reuse path's broken SQL (`SET revoked_at now()`, `WHERE userID`) is fixed
- [x] Register sends the verification email after commit, and the token is no longer printed to stdout
- [ ] `verify-email` / `reset-password` remain unscoped: their tokens are unguessable, so app scoping is defence-in-depth only

### Slice 5 — Admin portal (Go HTML templates)
- [x] `migrations/0005_portal_sessions.sql`: `portal_sessions` (id, user_id FK, token_hash UNIQUE, csrf_token, ip_address, user_agent, expires_at, revoked_at, created_at)
- [x] Server-side sessions (12h absolute), revocable on logout — not bearer JWTs
- [x] `internal/portal`: handlers + embedded `html/template` (stdlib only), mounted at `/portal`
- [x] Platform-admin login via `is_platform_admin AND app_id IS NULL`; existing partial unique index covers the email
- [x] Cookie: `HttpOnly`, `SameSite=Lax`, `Secure` when `APP_ENV != development`, `Path=/portal`, opaque token stored hashed
- [x] CSRF: per-session token in a hidden field, constant-time compared on every non-GET (including logout)
- [x] Pages: apps (create + key shown once), orgs per app, users per org (creates SUPERADMIN, auto-verified)
- [x] `cmd/seed` bootstraps the first platform admin; built into the image as `/app/seed`
- [x] Portal login is rate-limited via the injected `LoginLimiter`; whole portal behind the global per-IP limiter
- [ ] Platform-admin password reset is not implemented (forgot-password is app-keyed and platform admins have no app)

### Slice 6 — Django (sks) side integration
- [ ] Add `AUTH_APP_KEY` env var; send `X-App-Key` on any calls to auth
- [ ] JWT verification middleware: validate **locally** — no introspection endpoint. Fetch + cache JWKS (with refresh interval), validate RS256 signature, `exp`, `iss`, and `aud == sks app UUID`
- [ ] Resolve tenant from `org` claim (reject 403 when missing)
- [ ] Thread the org into DRF queries (every queryset filtered by org)
- [ ] Guard: `created_by = sub` claim; role-based permission classes use `role` claim

### Slice 7 — Invites & role management (after v1)
- [ ] ADMIN/SUPERADMIN can invite CLERKs (email verification + temp password)
- [ ] Role CRUD within org (claim refresh on next login)

## Cleanup (done before Slice 5)

- [x] `migrations/0004_cleanup.sql`: `apps.base_url` + `set_updated_at` triggers on users/apps/organizations/memberships
- [x] Email links built from `app.BaseURL` (verify + reset) instead of hardcoded `yourapp.com`
- [x] Verify-email and reset-password queries scoped by `app_id`
- [x] Debug `println` removed from `password.go`
- [x] Unit tests: `signing` (thumbprint, KEK round-trip), `store` (appkey), `http` (trusted-proxy `ClientIP`)
- [x] README rewritten to match the actual API, claims, env vars and migration flow

## Open questions

None currently. Migration tooling, portal `aud`, and JWKS rate limiting are resolved above.

## Dev/test seeds (for local only)

- [x] Platform admin: `docker compose run --rm --entrypoint /app/seed auth-service -email <email> -password <password>`
- [ ] App + appkey, org, SUPERADMIN: create through the portal (or reuse the same seed command later for a full demo seed) — used by sks local dev.
