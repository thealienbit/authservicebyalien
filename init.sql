-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS citext;

-- =========================
-- USERS
-- =========================
CREATE TABLE users (
    id              UUID PRIMARY KEY,
    status          TEXT NOT NULL CHECK (status IN ('ACTIVE', 'LOCKED', 'DISABLED')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- =========================
-- USER EMAILS
-- =========================
CREATE TABLE user_emails (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email           CITEXT NOT NULL UNIQUE,
    is_verified     BOOLEAN NOT NULL DEFAULT FALSE,
    is_primary      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    verified_at     TIMESTAMPTZ
);

CREATE INDEX idx_user_emails_user_id ON user_emails(user_id);

-- =========================
-- PASSWORD CREDENTIALS
-- =========================
CREATE TABLE password_credentials (
    user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    password_hash   TEXT NOT NULL,
    algo            TEXT NOT NULL DEFAULT 'argon2id',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- =========================
-- EMAIL VERIFICATION TOKENS
-- =========================
CREATE TABLE email_verification_tokens (
    id              UUID PRIMARY KEY,
    email_id        UUID NOT NULL REFERENCES user_emails(id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL UNIQUE,
    expires_at      TIMESTAMPTZ NOT NULL,
    used_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- =========================
-- PASSWORD RESET TOKENS
-- =========================
CREATE TABLE password_reset_tokens (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL UNIQUE,
    expires_at      TIMESTAMPTZ NOT NULL,
    used_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- =========================
-- REFRESH TOKENS
-- =========================
CREATE TABLE refresh_tokens (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL UNIQUE,
    user_agent      TEXT,
    ip_address      INET,
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- =========================
-- AUDIT LOGS
-- =========================
CREATE TABLE audit_logs (
    id              UUID PRIMARY KEY,
    user_id         UUID,
    action          TEXT NOT NULL,
    ip_address      INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);

